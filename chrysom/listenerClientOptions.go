// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

var (
	defaultValidateListenerOptions = ListenerOptions{
		validateReader(),
		validateGetListenerLogger(),
		validateSetListenerLogger(),
		validatePullInterval(),
		validateListener(),
	}
	defaultListenerOptions = ListenerOptions{
		PullInterval(defaultPullInterval),
		GetListenerLogger(func(context.Context) *zap.Logger { return zap.NewNop() }),
		SetListenerLogger(func(context.Context, *zap.Logger) context.Context { return context.Background() }),
	}
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

type optionFunc func(*ListenerClient) error

func (f optionFunc) apply(c *ListenerClient) error {
	return f(c)
}

// reader sets the reader.
// Used internally by `ProvideReaderOption` for fx dependency injection.
func reader(reader Reader) ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if reader == nil {
				return fmt.Errorf("%w: nil Reader", ErrMisconfiguredListener)
			}

			c.reader = reader

			return nil
		})
}

// GetListenerLogger sets the getlogger, a func that returns a logger from the given context.
func GetListenerLogger(get func(context.Context) *zap.Logger) ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if get == nil {
				return fmt.Errorf("%w: nil GetListenerLogger", ErrMisconfiguredListener)
			}

			c.getLogger = get

			return nil
		})
}

// SetListenerLogger sets the getlogger, a func that embeds the a given logger in outgoing request contexts.
func SetListenerLogger(set func(context.Context, *zap.Logger) context.Context) ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if set == nil {
				return fmt.Errorf("%w: nil SetListenerLogger", ErrMisconfiguredListener)
			}

			c.setLogger = set

			return nil
		})
}

// PullInterval sets the pull interval, determines how often listeners should get updates.
// (Optional). Defaults to 5 seconds.
func PullInterval(duration time.Duration) ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if duration < 0 {
				return fmt.Errorf("%w: negative PullInterval", ErrMisconfiguredListener)
			}

			c.pullInterval = duration
			c.ticker = time.NewTicker(duration)

			return nil
		})
}

// Listener sets the Listener client's listener, listener is called during every PullInterval.
func Listener(listener ListenerInterface) ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if listener == nil {
				return fmt.Errorf("%w: nil Listener", ErrMisconfiguredListener)
			}

			c.listener = listener

			return nil
		})
}

func validateReader() ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if c.reader == nil {
				return fmt.Errorf("%w: nil Reader", ErrMisconfiguredListener)
			}

			return nil
		})
}

func validateGetListenerLogger() ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if c.getLogger == nil {
				return fmt.Errorf("%w: nil GetListenerLogger", ErrMisconfiguredListener)
			}

			return nil
		})
}

func validateSetListenerLogger() ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if c.setLogger == nil {
				return fmt.Errorf("%w: nil SetListenerLogger", ErrMisconfiguredListener)
			}

			return nil
		})
}

func validatePullInterval() ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if c.pullInterval < 0 {
				return fmt.Errorf("%w: negative PullInterval", ErrMisconfiguredListener)
			} else if c.ticker == nil {
				return fmt.Errorf("%w: nil ticker", ErrMisconfiguredListener)
			}

			return nil
		})
}

func validateListener() ListenerOption {
	return optionFunc(
		func(c *ListenerClient) error {
			if c.listener == nil {
				return fmt.Errorf("%w: nil Listener", ErrMisconfiguredListener)
			}

			return nil
		})
}
