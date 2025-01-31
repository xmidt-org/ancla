// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// ListenerOption is a functional option type for ListenerClient.
type ListenerOption interface {
	apply(*ListenerClient) error
}

type ListenerOptions []ListenerOption

func (opts ListenerOptions) apply(c *ListenerClient) (errs error) {
	for _, o := range opts {
		errs = errors.Join(errs, o.apply(c))
	}

	return errs
}

type listenerOptionFunc func(*ListenerClient) error

func (f listenerOptionFunc) apply(c *ListenerClient) error {
	return f(c)
}

// reader sets the reader.
// Used internally by `ProvideReaderOption` for fx dependency injection.
func reader(reader Reader) ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) error {
			c.reader = reader

			return nil
		})
}

// GetListenerLogger sets the getlogger, a func that returns a logger from the given context.
func GetListenerLogger(get func(context.Context) *zap.Logger) ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) error {
			c.getLogger = func(context.Context) *zap.Logger { return zap.NewNop() }
			if get != nil {
				c.getLogger = get
			}

			return nil
		})
}

// SetListenerLogger sets the getlogger, a func that embeds the a given logger in outgoing request contexts.
func SetListenerLogger(set func(context.Context, *zap.Logger) context.Context) ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) error {
			c.setLogger = func(context.Context, *zap.Logger) context.Context { return context.TODO() }
			if set != nil {
				c.setLogger = set
			}

			return nil
		})
}

// PullInterval sets the pull interval, determines how often listeners should get updates.
// (Optional). Defaults to 5 seconds.
func PullInterval(duration time.Duration) ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) error {
			c.pullInterval = defaultPullInterval
			if duration > 0 {
				c.pullInterval = duration
			}

			c.ticker = time.NewTicker(c.pullInterval)

			return nil
		})
}

// Listener sets the Listener client's listener, listener is called during every PullInterval.
func Listener(listener ListenerInterface) ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) error {
			c.listener = listener

			return nil
		})
}

func listenerValidator() ListenerOption {
	return listenerOptionFunc(
		func(c *ListenerClient) (errs error) {
			if c.reader == nil {
				errs = errors.Join(errs, errors.New("nil Reader"))
			}
			if c.listener == nil {
				errs = errors.Join(errs, errors.New("nil Listener"))
			}
			if errs != nil {
				errs = errors.Join(ErrMisconfiguredListener, errs)
			}

			return
		})
}
