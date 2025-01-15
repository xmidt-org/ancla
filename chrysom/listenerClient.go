// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Errors that can be returned by this package. Since some of these errors are returned wrapped, it
// is safest to use errors.Is() to check for them.
// Some internal errors might be unwrapped from output errors but unless these errors become exported,
// they are not part of the library API and may change in future versions.
var (
	ErrFailedAuthentication = errors.New("failed to authentication with argus")

	ErrListenerNotStopped = errors.New("listener is either running or starting")
	ErrListenerNotRunning = errors.New("listener is either stopped or stopping")
	ErrNoListenerProvided = errors.New("no listener provided")
	ErrNoReaderProvided   = errors.New("no reader provided")
)

// listening states
const (
	stopped int32 = iota
	running
	transitioning
)

const (
	defaultPullInterval = time.Second * 5
)

// ListenerClient is the client used to poll Argus for updates.
type ListenerClient struct {
	observer  *observerConfig
	getLogger func(context.Context) *zap.Logger
	setLogger func(context.Context, *zap.Logger) context.Context
	reader    Reader
}

type observerConfig struct {
	listener          Listener
	ticker            *time.Ticker
	pullInterval      time.Duration
	pollsTotalCounter *prometheus.CounterVec

	shutdown chan struct{}
	state    int32
}

// NewListenerClient creates a new ListenerClient to be used to poll Argus
// for updates.
func NewListenerClient(listener Listener,
	getLogger func(context.Context) *zap.Logger,
	setLogger func(context.Context, *zap.Logger) context.Context,
	pullInterval time.Duration, pollsTotalCounter *prometheus.CounterVec, reader Reader) (*ListenerClient, error) {
	if listener == nil {
		return nil, ErrNoListenerProvided
	}
	if pullInterval == 0 {
		pullInterval = defaultPullInterval
	}
	if setLogger == nil {
		setLogger = func(ctx context.Context, _ *zap.Logger) context.Context {
			return ctx
		}
	}
	if reader == nil {
		return nil, ErrNoReaderProvided
	}
	return &ListenerClient{
		observer: &observerConfig{
			listener:          listener,
			ticker:            time.NewTicker(pullInterval),
			pullInterval:      pullInterval,
			pollsTotalCounter: pollsTotalCounter,
			shutdown:          make(chan struct{}),
		},
		getLogger: getLogger,
		setLogger: setLogger,
		reader:    reader,
	}, nil
}

// Start begins listening for updates on an interval given that client configuration
// is setup correctly. If a listener process is already in progress, calling Start()
// is a NoOp. If you want to restart the current listener process, call Stop() first.
func (c *ListenerClient) Start(ctx context.Context) error {
	logger := c.getLogger(ctx)
	if c.observer == nil || c.observer.listener == nil {
		logger.Warn("No listener was setup to receive updates.")
		return nil
	}
	if c.observer.ticker == nil {
		logger.Error("Observer ticker is nil", zap.Error(ErrUndefinedIntervalTicker))
		return ErrUndefinedIntervalTicker
	}

	if !atomic.CompareAndSwapInt32(&c.observer.state, stopped, transitioning) {
		logger.Error("Start called when a listener was not in stopped state", zap.Error(ErrListenerNotStopped))
		return ErrListenerNotStopped
	}

	c.observer.ticker.Reset(c.observer.pullInterval)
	go func() {
		for {
			select {
			case <-c.observer.shutdown:
				return
			case <-c.observer.ticker.C:
				outcome := SuccessOutcome
				ctx := c.setLogger(context.Background(), logger)
				items, err := c.reader.GetItems(ctx, "")
				if err == nil {
					c.observer.listener.Update(items)
				} else {
					outcome = FailureOutcome
					logger.Error("Failed to get items for listeners", zap.Error(err))
				}
				c.observer.pollsTotalCounter.With(prometheus.Labels{
					OutcomeLabel: outcome}).Add(1)
			}
		}
	}()

	atomic.SwapInt32(&c.observer.state, running)
	return nil
}

// Stop requests the current listener process to stop and waits for its goroutine to complete.
// Calling Stop() when a listener is not running (or while one is getting stopped) returns an
// error.
func (c *ListenerClient) Stop(ctx context.Context) error {
	if c.observer == nil || c.observer.ticker == nil {
		return nil
	}

	logger := c.getLogger(ctx)
	if !atomic.CompareAndSwapInt32(&c.observer.state, running, transitioning) {
		logger.Error("Stop called when a listener was not in running state", zap.Error(ErrListenerNotStopped))
		return ErrListenerNotRunning
	}

	c.observer.ticker.Stop()
	c.observer.shutdown <- struct{}{}
	atomic.SwapInt32(&c.observer.state, stopped)
	return nil
}
