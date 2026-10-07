// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestProvideBasicClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		require := require.New(t)
		out, err := ProvideBasicClient(ProvideBasicClientIn{Options: requiredClientOptions})
		require.NoError(err)
		require.NotNil(out.PushReader)
		require.NotNil(out.Reader)
	})

	t.Run("misconfigured", func(t *testing.T) {
		_, err := ProvideBasicClient(ProvideBasicClientIn{})
		assert.ErrorIs(t, err, ErrMisconfiguredClient)
	})
}

func TestProvideListenerClient(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		require := require.New(t)
		basic, err := NewBasicClient(requiredClientOptions)
		require.NoError(err)

		client, err := ProvideListenerClient(ListenerClientIn{
			PollsTotalCounter: pollsTotalCounter,
			Options:           ListenerOptions{reader(basic), Listener(mockListener)},
		})
		require.NoError(err)
		require.NotNil(client)
	})

	t.Run("misconfigured", func(t *testing.T) {
		client, err := ProvideListenerClient(ListenerClientIn{PollsTotalCounter: pollsTotalCounter})
		assert.ErrorIs(t, err, ErrMisconfiguredListener)
		assert.Nil(t, client)
	})
}

func TestProvideStartListenerClient(t *testing.T) {
	require := require.New(t)
	client, stopServer, err := newStartStopClient(true)
	require.NoError(err)
	defer stopServer()

	lc := fxtest.NewLifecycle(t)
	require.NoError(ProvideStartListenerClient(StartListenerIn{Listener: client, LC: lc}))
	require.Equal(running, client.state)

	lc.RequireStart()
	lc.RequireStop()
	require.Equal(stopped, client.state)
}

func TestProvideMetrics(t *testing.T) {
	type metrics struct {
		fx.In

		Gauge   prometheus.Gauge       `name:"wrp_event_stream_list_size"`
		Counter *prometheus.CounterVec `name:"chrysom_polls_total"`
	}

	var m metrics
	app := fxtest.New(t,
		fx.Provide(func() (*touchstone.Factory, error) {
			cfg := touchstone.Config{DefaultNamespace: "n", DefaultSubsystem: "s"}
			_, pr, err := touchstone.New(cfg)
			if err != nil {
				return nil, err
			}
			return touchstone.NewFactory(cfg, zap.NewNop(), pr), nil
		}),
		ProvideMetrics(),
		fx.Populate(&m),
	)

	require := require.New(t)
	require.NoError(app.Err())
	require.NotNil(m.Gauge)
	require.NotNil(m.Counter)
}

type readerFunc func(context.Context, string) (Items, error)

func (f readerFunc) GetItems(ctx context.Context, owner string) (Items, error) { return f(ctx, owner) }

func TestListenerUpdateFailureIsLogged(t *testing.T) {
	require := require.New(t)
	called := make(chan struct{}, 1)

	client, err := NewListenerClient(pollsTotalCounter,
		PullInterval(10*time.Millisecond),
		reader(readerFunc(func(context.Context, string) (Items, error) { return Items{}, nil })),
		Listener(ListenerFunc(func(context.Context, Items) error {
			select {
			case called <- struct{}{}:
			default:
			}
			return errFails
		})),
	)
	require.NoError(err)

	require.NoError(client.Start(context.Background()))
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		require.Fail("listener was never called")
	}
	require.NoError(client.Stop(context.Background()))
}

func TestListenerStopWithNilTicker(t *testing.T) {
	client := &ListenerClient{}
	assert.NoError(t, client.Stop(context.Background()))
}
