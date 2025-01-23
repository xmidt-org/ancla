// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

var (
	mockListener = ListenerFunc((func(_ Items) {
		time.Sleep(time.Millisecond * 100)
	}))
	pollsTotalCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "testPollsCounter",
			Help: "testPollsCounter",
		},
		[]string{OutcomeLabel},
	)
)

func TestValidateListenerOptions(t *testing.T) {
	type testCase struct {
		Description    string
		ValidateOption ListenerOption
		Client         ListenerClient
	}

	tcs := []testCase{
		{
			Description:    "Nil listener",
			ValidateOption: validateListener(),
			Client: ListenerClient{
				pullInterval: defaultPullInterval,
				ticker:       time.NewTicker(defaultPullInterval),
				getLogger: func(context.Context) *zap.Logger {
					return zap.NewNop()
				},
				setLogger: func(context.Context, *zap.Logger) context.Context {
					return context.Background()
				},
				reader: &BasicClient{},
			},
		},
		{
			Description:    "Non-postive pull interval",
			ValidateOption: validatePullInterval(),
			Client: ListenerClient{
				listener:     ListenerFunc(func(Items) {}),
				pullInterval: -1,
				ticker:       time.NewTicker(defaultPullInterval),
				getLogger: func(context.Context) *zap.Logger {
					return zap.NewNop()
				},
				setLogger: func(context.Context, *zap.Logger) context.Context {
					return context.Background()
				},
				reader: &BasicClient{},
			},
		},
		{
			Description:    "Nil ticker",
			ValidateOption: validatePullInterval(),
			Client: ListenerClient{
				listener:     ListenerFunc(func(Items) {}),
				pullInterval: defaultPullInterval,
				getLogger: func(context.Context) *zap.Logger {
					return zap.NewNop()
				},
				setLogger: func(context.Context, *zap.Logger) context.Context {
					return context.Background()
				},
				reader: &BasicClient{},
			},
		},
		{
			Description:    "Nil SetListenerLogger",
			ValidateOption: validateSetListenerLogger(),
			Client: ListenerClient{
				listener:     ListenerFunc(func(Items) {}),
				pullInterval: defaultPullInterval,
				ticker:       time.NewTicker(defaultPullInterval),
				getLogger: func(context.Context) *zap.Logger {
					return zap.NewNop()
				},
				reader: &BasicClient{},
			},
		},
		{
			Description:    "Nil GetListenerLogger",
			ValidateOption: validateGetListenerLogger(),
			Client: ListenerClient{
				listener:     ListenerFunc(func(Items) {}),
				pullInterval: defaultPullInterval,
				ticker:       time.NewTicker(defaultPullInterval),
				setLogger: func(context.Context, *zap.Logger) context.Context {
					return context.Background()
				},
				reader: &BasicClient{},
			},
		},
		{
			Description:    "Nil Reader",
			ValidateOption: validateReader(),
			Client: ListenerClient{
				listener:     ListenerFunc(func(Items) {}),
				pullInterval: defaultPullInterval,
				ticker:       time.NewTicker(defaultPullInterval),
				getLogger: func(context.Context) *zap.Logger {
					return zap.NewNop()
				},
				setLogger: func(context.Context, *zap.Logger) context.Context {
					return context.Background()
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			err := tc.ValidateOption.apply(&tc.Client)
			assert.ErrorIs(err, ErrMisconfiguredListener)
		})
	}
}

func TestListenerStartStopPairsParallel(t *testing.T) {
	require := require.New(t)
	client, close, err := newStartStopClient(true)
	require.NoError(err)
	require.NotNil(client)
	defer close()

	t.Run("ParallelGroup", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			testNumber := i
			t.Run(strconv.Itoa(testNumber), func(t *testing.T) {
				t.Parallel()
				assert := assert.New(t)
				errStart := client.Start(context.Background())
				if errStart != nil {
					assert.Equal(ErrListenerNotStopped, errStart)
				}
				client.listener.Update(Items{})
				time.Sleep(time.Millisecond * 400)
				errStop := client.Stop(context.Background())
				if errStop != nil {
					assert.Equal(ErrListenerNotRunning, errStop)
				}
			})
		}
	})

	require.Equal(stopped, client.state)
}

func TestListenerStartStopPairsSerial(t *testing.T) {
	require := require.New(t)
	client, close, err := newStartStopClient(true)
	assert.Nil(t, err)
	defer close()

	for i := 0; i < 5; i++ {
		testNumber := i
		t.Run(strconv.Itoa(testNumber), func(t *testing.T) {
			assert := assert.New(t)
			fmt.Printf("%d: Start\n", testNumber)
			assert.Nil(client.Start(context.Background()))
			assert.Nil(client.Stop(context.Background()))
			fmt.Printf("%d: Done\n", testNumber)
		})
	}
	require.Equal(stopped, client.state)
}

func TestListenerEdgeCases(t *testing.T) {
	t.Run("NoListener", func(t *testing.T) {
		_, _, err := newStartStopClient(false)
		assert.ErrorIs(t, err, ErrMisconfiguredListener)
	})

	t.Run("NilTicker", func(t *testing.T) {
		assert := assert.New(t)
		client, stopServer, err := newStartStopClient(true)
		assert.Nil(err)
		defer stopServer()
		client.ticker = nil
		assert.Equal(ErrUndefinedIntervalTicker, client.Start(context.Background()))
	})
}

func newStartStopClient(includeListener bool) (*ListenerClient, func(), error) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(getItemsValidPayload())
	}))

	var listener ListenerInterface
	if includeListener {
		listener = mockListener
	}
	anclaClient, err := NewBasicClient(ClientOptions{
		StoreBaseURL("https://example.com"),
		Bucket("bucket-name"),
		GetClientLogger(func(context.Context) *zap.Logger { return zap.NewNop() }),
		HTTPClient(http.DefaultClient),
	})
	if err != nil {
		return nil, func() {}, err
	}

	listenerClient, err := NewListenerClient(pollsTotalCounter,
		ListenerOptions{
			PullInterval(time.Millisecond * 200),
			reader(anclaClient),
			Listener(listener),
			GetListenerLogger(func(context.Context) *zap.Logger { return zap.NewNop() }),
			SetListenerLogger(func(context.Context, *zap.Logger) context.Context { return context.Background() }),
		})
	if err != nil {
		return nil, nil, err
	}

	return listenerClient, server.Close, nil
}

func TestValidateListenerConfig(t *testing.T) {
	tcs := []struct {
		desc        string
		options     ListenerOptions
		expectedErr error
	}{
		{
			desc:        "New listener client failure",
			expectedErr: ErrMisconfiguredListener,
		},
		{
			desc: "New listener client success",
			options: ListenerOptions{
				PullInterval(time.Second),
				reader(&BasicClient{}),
				Listener(mockListener),
				GetListenerLogger(func(context.Context) *zap.Logger { return zap.NewNop() }),
				SetListenerLogger(func(context.Context, *zap.Logger) context.Context { return context.Background() }),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			_, err := NewListenerClient(pollsTotalCounter, tc.options)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}
