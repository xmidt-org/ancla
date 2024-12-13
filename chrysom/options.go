// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"errors"
	"net/http"
)

var (
	ErrAddressEmpty      = errors.New("store address is required")
	ErrStoreBaseURLEmpty = errors.New("store base url is required")
	ErrBucketEmpty       = errors.New("bucket name is required")
	ErrHttpClientNil     = errors.New("nil http client")
)

var (
	defaultValidateOptions = Options{
		validateHTTPClient(),
		validateStoreBaseURL(),
		validateBucket(),
	}
	defaultOptions = Options{
		HTTPClient(http.DefaultClient),
	}
)

// Option is a functional option type for BasicClient.
type Option interface {
	apply(*BasicClient) error
}

type Options []Option

func (opts Options) apply(c *BasicClient) (errs error) {
	for _, o := range opts {
		errs = errors.Join(errs, o.apply(c))
	}

	return errs
}

type optionFunc func(*BasicClient) error

func (f optionFunc) apply(c *BasicClient) error {
	return f(c)
}

// Address sets the store address for the client.
func Address(address string) Option {
	return optionFunc(
		func(c *BasicClient) error {
			if address == "" {
				return ErrAddressEmpty
			}

			c.storeBaseURL = address + storeAPIPath

			return nil
		})
}

// Bucket sets the partition to be used by this client.
func Bucket(bucket string) Option {
	return optionFunc(
		func(c *BasicClient) error {
			if bucket == "" {
				return ErrBucketEmpty
			}

			c.bucket = bucket

			return nil
		})
}

// HTTPClient sets the HTTP client.
func HTTPClient(client *http.Client) Option {
	return optionFunc(
		func(c *BasicClient) (err error) {
			if client != nil {
				return errors.Join(ErrHttpClientNil, err)
			}

			c.client = client

			return err
		})
}

func validateHTTPClient() Option {
	return optionFunc(
		func(c *BasicClient) error {
			if c.client == nil {
				return ErrHttpClientNil
			}

			return nil
		})
}

func validateStoreBaseURL() Option {
	return optionFunc(
		func(c *BasicClient) error {
			if c.storeBaseURL == "" {
				return ErrStoreBaseURLEmpty
			}

			return nil
		})
}

func validateBucket() Option {
	return optionFunc(
		func(c *BasicClient) error {
			if c.bucket == "" {
				return ErrBucketEmpty
			}

			return nil
		})
}
