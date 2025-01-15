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

func TestListenerStartStopPairsParallel(t *testing.T) {
	require := require.New(t)
	client, close, err := newStartStopClient(true)
	assert.Nil(t, err)
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
				client.observer.listener.Update(Items{})
				time.Sleep(time.Millisecond * 400)
				errStop := client.Stop(context.Background())
				if errStop != nil {
					assert.Equal(ErrListenerNotRunning, errStop)
				}
			})
		}
	})

	require.Equal(stopped, client.observer.state)
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
	require.Equal(stopped, client.observer.state)
}

func TestListenerEdgeCases(t *testing.T) {
	t.Run("NoListener", func(t *testing.T) {
		_, _, err := newStartStopClient(false)
		assert.Equal(t, ErrNoListenerProvided, err)
	})

	t.Run("NilTicker", func(t *testing.T) {
		assert := assert.New(t)
		client, stopServer, err := newStartStopClient(true)
		assert.Nil(err)
		defer stopServer()
		client.observer.ticker = nil
		assert.Equal(ErrUndefinedIntervalTicker, client.Start(context.Background()))
	})
}

func newStartStopClient(includeListener bool) (*ListenerClient, func(), error) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(getItemsValidPayload())
	}))

	var listener Listener
	if includeListener {
		listener = mockListener
	}
	client, err := NewListenerClient(listener,
		func(context.Context) *zap.Logger { return zap.NewNop() },
		func(context.Context, *zap.Logger) context.Context { return context.Background() },
		time.Millisecond*200, pollsTotalCounter, &BasicClient{client: http.DefaultClient})
	if err != nil {
		return nil, nil, err
	}

	return client, server.Close, nil
}

func TestValidateListenerConfig(t *testing.T) {
	tcs := []struct {
		desc              string
		listener          Listener
		pullInterval      time.Duration
		expectedErr       error
		pollsTotalCounter *prometheus.CounterVec
		reader            Reader
	}{
		{
			desc:        "Listener Config Failure",
			expectedErr: ErrNoListenerProvided,
		},
		{
			desc:              "No reader Failure",
			listener:          mockListener,
			pullInterval:      time.Second,
			pollsTotalCounter: pollsTotalCounter,
			expectedErr:       ErrNoReaderProvided,
		},
		{
			desc:              "Happy case Success",
			listener:          mockListener,
			pullInterval:      time.Second,
			pollsTotalCounter: pollsTotalCounter,
			reader:            &BasicClient{},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			_, err := NewListenerClient(tc.listener,
				func(context.Context) *zap.Logger { return zap.NewNop() },
				func(context.Context, *zap.Logger) context.Context { return context.Background() },
				tc.pullInterval, tc.pollsTotalCounter, tc.reader)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}
