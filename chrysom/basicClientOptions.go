// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/xmidt-org/ancla/auth"
	"go.uber.org/zap"
)

var (
	ErrMisconfiguredClient = errors.New("ancla client configuration error")
)

// ClientOption is a functional option type for BasicClient.
type ClientOption interface {
	apply(*BasicClient) error
}

type ClientOptions []ClientOption

func (opts ClientOptions) apply(c *BasicClient) (errs error) {
	for _, o := range opts {
		errs = errors.Join(errs, o.apply(c))
	}

	return errs
}

type clientOptionFunc func(*BasicClient) error

func (f clientOptionFunc) apply(c *BasicClient) error {
	return f(c)
}

// StoreBaseURL sets the store address for the client.
func StoreBaseURL(url string) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.storeBaseURL = "http://localhost:6600"
			if url != "" {
				c.storeBaseURL = url
			}

			return nil
		})
}

// StoreAPIPath sets the store url api path.
// (Optional) Default is "/api/v1/store".
func StoreAPIPath(path string) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.storeAPIPath = storeV1APIPath
			if path != "" {
				c.storeAPIPath = path
			}

			return nil
		})
}

// Bucket sets the partition to be used by this client.
func Bucket(bucket string) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.bucket = bucket

			return nil
		})
}

// HTTPClient sets the HTTP client.
func HTTPClient(client *http.Client) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.client = http.DefaultClient
			if client != nil {
				c.client = client
			}

			return nil
		})
}

// GetLogger sets the getlogger, a func that returns a logger from the given context.
func GetClientLogger(get func(context.Context) *zap.Logger) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.getLogger = func(context.Context) *zap.Logger { return zap.NewNop() }
			if get != nil {
				c.getLogger = get
			}

			return nil
		})
}

// Auth sets auth, auth provides the mechanism to add auth headers to outgoing requests.
// (Optional) If not provided, no auth headers are added.
func Auth(authD auth.Decorator) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			c.auth = auth.Nop{}
			if authD != nil {
				c.auth = authD
			}

			return nil
		})
}

func clientValidator() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) (errs error) {
			c.storeBaseURL, errs = url.JoinPath(c.storeBaseURL, c.storeAPIPath)
			if errs != nil {
				errs = errors.Join(errors.New("failed to combine StoreBaseURL & StoreAPIPath"), errs)
			}
			if c.bucket == "" {
				errs = errors.Join(errs, errors.New("empty string Bucket"))
			}
			if errs != nil {
				errs = errors.Join(ErrMisconfiguredClient, errs)
			}

			return
		})
}
