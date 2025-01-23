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

var (
	ErrListenerNotStopped      = errors.New("listener is either running or starting")
	ErrListenerNotRunning      = errors.New("listener is either stopped or stopping")
	ErrUndefinedIntervalTicker = errors.New("interval ticker is nil. Can't listen for updates")
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
	listener          ListenerInterface
	ticker            *time.Ticker
	pullInterval      time.Duration
	pollsTotalCounter *prometheus.CounterVec
	getLogger         func(context.Context) *zap.Logger
	setLogger         func(context.Context, *zap.Logger) context.Context
	reader            Reader

	shutdown chan struct{}
	state    int32
}

// NewListenerClient creates a new ListenerClient to be used to poll Argus
// for updates.
func NewListenerClient(pollsTotalCounter *prometheus.CounterVec, opts ListenerOptions) (*ListenerClient, error) {
	client := ListenerClient{
		pollsTotalCounter: pollsTotalCounter,
		shutdown:          make(chan struct{}),
	}

	opts = append(defaultListenerOptions, opts)
	opts = append(opts, defaultValidateListenerOptions)

	return &client, opts.apply(&client)
}

// Start begins listening for updates on an interval given that client configuration
// is setup correctly. If a listener process is already in progress, calling Start()
// is a NoOp. If you want to restart the current listener process, call Stop() first.
func (c *ListenerClient) Start(ctx context.Context) error {
	logger := c.getLogger(ctx)
	if c.listener == nil {
		logger.Warn("No listener was setup to receive updates.")
		return nil
	}
	if c.ticker == nil {
		logger.Error("Observer ticker is nil", zap.Error(ErrUndefinedIntervalTicker))
		return ErrUndefinedIntervalTicker
	}

	if !atomic.CompareAndSwapInt32(&c.state, stopped, transitioning) {
		logger.Error("Start called when a listener was not in stopped state", zap.Error(ErrListenerNotStopped))
		return ErrListenerNotStopped
	}

	c.ticker.Reset(c.pullInterval)
	go func() {
		for {
			select {
			case <-c.shutdown:
				return
			case <-c.ticker.C:
				outcome := SuccessOutcome
				ctx := c.setLogger(context.Background(), logger)
				items, err := c.reader.GetItems(ctx, "")
				if err == nil {
					c.listener.Update(items)
				} else {
					outcome = FailureOutcome
					logger.Error("Failed to get items for listeners", zap.Error(err))
				}
				c.pollsTotalCounter.With(prometheus.Labels{
					OutcomeLabel: outcome}).Add(1)
			}
		}
	}()

	atomic.SwapInt32(&c.state, running)
	return nil
}

// Stop requests the current listener process to stop and waits for its goroutine to complete.
// Calling Stop() when a listener is not running (or while one is getting stopped) returns an
// error.
func (c *ListenerClient) Stop(ctx context.Context) error {
	if c.ticker == nil {
		return nil
	}

	logger := c.getLogger(ctx)
	if !atomic.CompareAndSwapInt32(&c.state, running, transitioning) {
		logger.Error("Stop called when a listener was not in running state", zap.Error(ErrListenerNotStopped))
		return ErrListenerNotRunning
	}

	c.ticker.Stop()
	c.shutdown <- struct{}{}
	atomic.SwapInt32(&c.state, stopped)
	return nil
}
