// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
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

func TestListenerOptions(t *testing.T) {

	listener := ListenerInterface(mockListener)
	anclaClient, err := NewBasicClient(requiredClientOptions)
	require.NoError(t, err)
	require.NotNil(t, anclaClient)

	requiredListenerOptions := ListenerOptions{
		reader(anclaClient),
		Listener(listener),
	}

	type testCase struct {
		Description     string
		ListenerOptions ListenerOptions
		ExpectedErr     error
	}

	tcs := []testCase{
		{
			Description: "Nil reader failure",
			ListenerOptions: ListenerOptions{
				reader(nil),
				Listener(listener),
			},
			ExpectedErr: ErrMisconfiguredListener,
		},
		{
			Description: "Nil listener failure",
			ListenerOptions: ListenerOptions{
				reader(anclaClient),
				Listener(nil),
			},
			ExpectedErr: ErrMisconfiguredListener,
		},
		{
			Description: "Correct required values and bad optional values (ignored)",
			ListenerOptions: append(requiredListenerOptions,
				ListenerOptions{
					GetListenerLogger(nil),
					SetListenerLogger(nil),
					PullInterval(-1),
				},
			),
		},
		{
			Description: "Correct required and optional values",
			ListenerOptions: append(requiredListenerOptions,
				ListenerOptions{
					GetListenerLogger(func(context.Context) *zap.Logger { return zap.NewNop() }),
					SetListenerLogger(func(context.Context, *zap.Logger) context.Context { return context.TODO() }),
					PullInterval(1),
				},
			),
		},
		{
			Description:     "Correct listener values",
			ListenerOptions: requiredListenerOptions,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			listener, errs := NewListenerClient(pollsTotalCounter, tc.ListenerOptions)
			if tc.ExpectedErr != nil {
				assert.ErrorIs(errs, tc.ExpectedErr)

				return
			}

			assert.NoError(errs)
			assert.NotNil(listener)
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
	anclaClient, err := NewBasicClient(requiredClientOptions)
	if err != nil {
		return nil, func() {}, err
	}

	listenerClient, err := NewListenerClient(pollsTotalCounter,
		ListenerOptions{
			PullInterval(time.Millisecond * 200),
			reader(anclaClient),
			Listener(listener),
			GetListenerLogger(nil),
			SetListenerLogger(nil),
		})
	if err != nil {
		return nil, nil, err
	}

	return listenerClient, server.Close, nil
}
