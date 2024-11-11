// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/sallust"
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

// ListenerConfig contains config data for polling the Argus client.
type ListenerConfig struct {
	// Listener provides a mechanism to fetch a copy of all items within a bucket on
	// an interval.
	// (Optional). If not provided, listening won't be enabled for this client.
	Listener Listener

	// PullInterval is how often listeners should get updates.
	// (Optional). Defaults to 5 seconds.
	PullInterval time.Duration
}

// ListenerClient is the client used to poll Argus for updates.
type ListenerClient struct {
	observer *observerConfig
	reader   Reader
}

type observerConfig struct {
	listener     Listener
	ticker       *time.Ticker
	pullInterval time.Duration
	measures     Measures
	shutdown     chan struct{}
	state        int32
}

// NewListenerClient creates a new ListenerClient to be used to poll Argus
// for updates.
func NewListenerClient(config ListenerConfig, measures Measures, r Reader) (*ListenerClient, error) {
	if config.Listener == nil {
		return nil, ErrNoListenerProvided
	}
	if config.PullInterval == 0 {
		config.PullInterval = defaultPullInterval
	}
	if r == nil {
		return nil, ErrNoReaderProvided
	}
	return &ListenerClient{
		observer: &observerConfig{
			listener:     config.Listener,
			ticker:       time.NewTicker(config.PullInterval),
			pullInterval: config.PullInterval,
			measures:     measures,
			shutdown:     make(chan struct{}),
		},
		reader: r,
	}, nil
}

// Start begins listening for updates on an interval given that client configuration
// is setup correctly. If a listener process is already in progress, calling Start()
// is a NoOp. If you want to restart the current listener process, call Stop() first.
func (c *ListenerClient) Start(ctx context.Context) error {
	logger := sallust.Get(ctx)
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
				items, err := c.reader.GetItems(ctx, "")
				if err == nil {
					c.observer.listener.Update(ctx, items)
				} else {
					outcome = FailureOutcome
					logger.Error("Failed to get items for listeners", zap.Error(err))
				}
				c.observer.measures.PollsTotalCounter.With(prometheus.Labels{
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

	logger := sallust.Get(ctx)
	if !atomic.CompareAndSwapInt32(&c.observer.state, running, transitioning) {
		logger.Error("Stop called when a listener was not in running state", zap.Error(ErrListenerNotStopped))
		return errors.Join(ErrListenerNotStopped, ErrListenerNotRunning)
	}

	c.observer.ticker.Stop()
	c.observer.shutdown <- struct{}{}
	atomic.SwapInt32(&c.observer.state, stopped)
	return nil
}
