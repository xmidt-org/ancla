/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ancla

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	errInvalidURL            = errors.New("invalid Config URL")
	errInvalidFailureURL     = errors.New("invalid Failure URL")
	errInvalidAlternativeURL = errors.New("invalid Alternative URL(s)")
	errURLIsNotHTTPS         = errors.New("URL is not HTTPS")
	errInvalidHost           = errors.New("invalid host")
	errIPGivenAsHost         = errors.New("cannot use IP as host")
	errLocalhostGivenAsHost  = errors.New("cannot use Localhost as host")
	errLoopbackGivenAsHost   = errors.New("cannot use loopback host")
)

// Validator is
type Validator interface {
	Validate(w Webhook) error
}

// Validators is a WebhookValidator that ensures the webhook is valid with
// each validator in the list
type Validators []Validator

// ValidFunc is
type ValidFunc func(Webhook) error

// ValidURLFunc is
type ValidURLFunc func(*url.URL) error

// Validate runs the given webhook through each validator in the validators list.
// It returns as soon as the webhook is considered invalid and returns nil if the
// webhook is valid.
func (vs Validators) Validate(w Webhook) error {
	for _, v := range vs {
		err := v.Validate(w)
		if err != nil {
			return err
		}
	}
	return nil
}

// GoodURL parses the given webhook's Config.URL, FailureURL, and Config.AlternativeURLs
// and returns as soon as the URL is considered invalid. It returns nil if the URL is
// valid.
func GoodURL(vs []ValidURLFunc) ValidFunc {
	fs := []ValidURLFunc{}
	for _, v := range vs {
		if v != nil {
			fs = append(fs, v)
		}
	}
	return func(w Webhook) error {
		//check Config.URL
		parsedURL, err := url.ParseRequestURI(w.Config.URL)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidURL, err)
		}
		for _, f := range fs {
			err = f(parsedURL)
			if err != nil {
				return fmt.Errorf("%w: %v", errInvalidURL, err)
			}
		}
		//check FailureURL if it exists
		if w.FailureURL != "" {
			parsedFailureURL, err := url.ParseRequestURI(w.FailureURL)
			if err != nil {
				return fmt.Errorf("%w: %v", errInvalidFailureURL, err)
			}
			for _, f := range fs {
				err = f(parsedFailureURL)
				if err != nil {
					return fmt.Errorf("%w: %v", errInvalidFailureURL, err)
				}
			}
		}
		//check AlternativeURLs if they exist
		for _, u := range w.Config.AlternativeURLs {
			if u == "" {
				return errInvalidAlternativeURL
			}
			parsedAlternativeURL, err := url.ParseRequestURI(u)
			if err != nil {
				return fmt.Errorf("%w: %v", errInvalidAlternativeURL, err)
			}
			for _, f := range fs {
				err = f(parsedAlternativeURL)
				if err != nil {
					return fmt.Errorf("%w: %v", errInvalidAlternativeURL, err)
				}
			}
		}
		return nil
	}
}

// Validate runs the function and returns the result. This allows any ValidFunc to implement
// the Validator interface
func (vf ValidFunc) Validate(w Webhook) error {
	return vf(w)
}

// HTTPSOnlyEndpoints creates a ValidURLFunc that considers a URL valid if the scheme
// of the address is https.
func HTTPSOnlyEndpoints() ValidURLFunc {
	return func(u *url.URL) error {
		if u.Scheme != "https" {
			return errURLIsNotHTTPS
		}
		return nil
	}
}

// RejectHosts creates a ValidURLFunc that checks the URL and ensures the
// host does not contain any strings in the list of invalid hosts. It returns an error
// if the host does include an invalid host name.
func RejectHosts(invalidHosts []string) ValidURLFunc {
	return func(u *url.URL) error {
		host := u.Host
		for _, v := range invalidHosts {
			if strings.Contains(host, v) {
				return errInvalidHost
			}
		}
		return nil
	}
}

// RejectALLIPs creates a ValidURLFunc that checks if the URL is an IP and returns an error
// if it is.
func RejectAllIPs() ValidURLFunc {
	return func(u *url.URL) error {
		host, err := removePort(u.Host)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip != nil {
			return errIPGivenAsHost
		}
		return nil
	}
}

// RejectLoopback creates a ValidURLFunc that returns an error if the given URL is
// a loopback address.
func RejectLoopback() ValidURLFunc {
	return func(u *url.URL) error {
		host, err := removePort(u.Host)
		if err != nil {
			return err
		} else if host == "localhost" {
			return errLocalhostGivenAsHost
		}
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			return errLoopbackGivenAsHost
		}
		return nil
	}
}

func removePort(s string) (string, error) {
	if !strings.Contains(s, ":") {
		return s, nil
	}
	h, _, err := net.SplitHostPort(s)
	return h, err
}
