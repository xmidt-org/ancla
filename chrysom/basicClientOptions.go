// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/xmidt-org/ancla/auth"
	"go.uber.org/zap"
)

var (
	ErrMisconfiguredClient = errors.New("ancla client configuration error")
)

var (
	defaultValidateClientOptions = ClientOptions{
		validateHTTPClient(),
		validateStoreBaseURL(),
		validateStoreAPIPath(),
		validateBucket(),
		validateGetClientLogger(),
	}
	defaultClientOptions = ClientOptions{
		HTTPClient(http.DefaultClient),
		StoreAPIPath(storeV1APIPath),
	}
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
			if url == "" {
				return fmt.Errorf("%w: empty string StoreBaseURL", ErrMisconfiguredClient)
			}

			c.storeBaseURL = url

			return nil
		})
}

// StoreAPIPath sets the store url api path.
// (Optional) Default is "/api/v1/store".
func StoreAPIPath(path string) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if path == "" {
				return fmt.Errorf("%w: empty string StoreAPIPath", ErrMisconfiguredClient)
			}

			c.storeAPIPath = path

			return nil
		})
}

// Bucket sets the partition to be used by this client.
func Bucket(bucket string) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if bucket == "" {
				return fmt.Errorf("%w: empty string Bucket", ErrMisconfiguredClient)
			}

			c.bucket = bucket

			return nil
		})
}

// HTTPClient sets the HTTP client.
func HTTPClient(client *http.Client) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if client == nil {
				return fmt.Errorf("%w: nil http client", ErrMisconfiguredClient)
			}

			c.client = client

			return nil
		})
}

// GetLogger sets the getlogger, a func that returns a logger from the given context.
func GetClientLogger(get func(context.Context) *zap.Logger) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if get == nil {
				return fmt.Errorf("%w: nil GetLogger", ErrMisconfiguredClient)
			}

			c.getLogger = get

			return nil
		})
}

// Auth sets auth, auth provides the mechanism to add auth headers to outgoing requests.
// (Optional) If not provided, no auth headers are added.
func Auth(auth auth.Decorator) ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if auth == nil {
				return nil
			}

			c.auth = auth

			return nil
		})
}

func validateHTTPClient() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if c.client == nil {
				return fmt.Errorf("%w: nil http client", ErrMisconfiguredClient)
			}

			return nil
		})
}

func validateStoreBaseURL() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if c.storeBaseURL == "" {
				return fmt.Errorf("%w: empty string StoreBaseURL", ErrMisconfiguredClient)
			}

			url, err := url.JoinPath(c.storeBaseURL, c.storeAPIPath)
			if err != nil {
				return errors.Join(err, ErrMisconfiguredClient)
			}

			c.storeBaseURL = url

			return nil
		})
}

func validateStoreAPIPath() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if c.storeAPIPath == "" {
				return fmt.Errorf("%w: empty string StoreAPIPath", ErrMisconfiguredClient)
			}

			return nil
		})

}

func validateBucket() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if c.bucket == "" {
				return fmt.Errorf("%w: empty string Bucket", ErrMisconfiguredClient)
			}

			return nil
		})
}

func validateGetClientLogger() ClientOption {
	return clientOptionFunc(
		func(c *BasicClient) error {
			if c.getLogger == nil {
				return fmt.Errorf("%w: nil GetLogger", ErrMisconfiguredClient)
			}

			return nil
		})
}
