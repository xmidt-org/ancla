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
	errURLIsNotHTTPS         = errors.New("URL scheme is not HTTPS")
	errInvalidHost           = errors.New("host is blocked")
	errIPGivenAsHost         = errors.New("cannot use IP as host")
	errLoopbackGivenAsHost   = errors.New("cannot use loopback host")
	errIPinInvalidSubnets    = errors.New("IP is within a blocked subnet")
	errInvalidSubnet         = errors.New("invalid subnet")
	errNoSuchHost            = errors.New("host does not exist")
	errBadURLProtocol        = errors.New("bad URL protocol")
	errEmptyURL              = errors.New("error cannot be an empty string")
)

// filterNil takes out all entries of Nil value from the slice.
func filterNil(vs []ValidURLFunc) (filtered []ValidURLFunc) {
	for _, v := range vs {
		if v != nil {
			filtered = append(filtered, v)
		}
	}
	return
}

// GoodConfigURL parses the given webhook's Config.URL
// and returns as soon as the URL is considered invalid. It returns nil if the URL is
// valid.
func GoodConfigURL(vs []ValidURLFunc) ValidatorFunc {
	vs = filterNil(vs)
	return func(w Webhook) error {
		if w.Config.URL == "" {
			return fmt.Errorf("%w: %v",
				errInvalidURL, errEmptyURL)
		}
		parsedURL, err := url.ParseRequestURI(w.Config.URL)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidURL, err)
		}
		for _, f := range vs {
			err = f(parsedURL)
			if err != nil {
				return fmt.Errorf("%w: %v", errInvalidURL, err)
			}
		}
		return nil
	}
}

// GoodFailureURL parses the given webhook's FailureURL
// and returns as soon as the URL is considered invalid. It returns nil if the URL is
// valid.
func GoodFailureURL(vs []ValidURLFunc) ValidatorFunc {
	vs = filterNil(vs)
	return func(w Webhook) error {
		if w.FailureURL == "" {
			return nil
		}
		parsedFailureURL, err := url.ParseRequestURI(w.FailureURL)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidFailureURL, err)
		}
		for _, f := range vs {
			if err = f(parsedFailureURL); err != nil {
				return fmt.Errorf("%w: %v", errInvalidFailureURL, err)
			}
		}
		return nil
	}
}

// GoodAlternativeURLs parses the given webhook's Config.AlternativeURLs
// and returns as soon as the URL is considered invalid. It returns nil if the URL is
// valid.
func GoodAlternativeURLs(vs []ValidURLFunc) ValidatorFunc {
	vs = filterNil(vs)
	return func(w Webhook) error {
		for _, u := range w.Config.AlternativeURLs {
			if u == "" {
				return fmt.Errorf("%w: %v",
					errInvalidAlternativeURL, errEmptyURL)
			}
			parsedAlternativeURL, err := url.ParseRequestURI(u)
			if err != nil {
				return fmt.Errorf("%w '%s': %v",
					errInvalidAlternativeURL, u, err)
			}
			for _, f := range vs {
				err = f(parsedAlternativeURL)
				if err != nil {
					return fmt.Errorf("%w '%s': %v",
						errInvalidAlternativeURL, u, err)
				}
			}
		}
		return nil
	}
}

// GoodURLScheme creates a ValidURLFunc that checks the scheme of the URL.
// If httpsOnly is true, then it will only allow URLs with "https" schemes.
// If httpsOnly is false, it will only allow URLs with "https" and "http" schemes.
func GoodURLScheme(httpsOnly bool) ValidURLFunc {
	return func(u *url.URL) error {
		if u.Scheme != "https" && u.Scheme != "http" {
			return fmt.Errorf("%w: %s", errBadURLProtocol, u.Scheme)
		}
		if httpsOnly && u.Scheme != "https" {
			return fmt.Errorf("%w: %s", errURLIsNotHTTPS, u.Scheme)
		}
		return nil
	}
}

// RejectHosts creates a ValidURLFunc that checks the URL and ensures the
// host does not contain any strings in the list of invalid hosts. It returns an error
// if the host does include an invalid host name.
func RejectHosts(invalidHosts []string) ValidURLFunc {
	ih := []string{}
	for _, v := range invalidHosts {
		if v != "" {
			ih = append(ih, v)
		}
	}
	return func(u *url.URL) error {
		for _, v := range ih {
			if strings.Contains(u.Host, v) {
				return fmt.Errorf("%w: %s", errInvalidHost, u.Host)
			}
		}
		return nil
	}
}

// RejectALLIPs creates a ValidURLFunc that checks if the URL is an IP and returns an error
// if it is.
func RejectAllIPs() ValidURLFunc {
	return func(u *url.URL) error {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if ip != nil {
			return fmt.Errorf("%w: %v", errIPGivenAsHost, host)
		}
		return nil
	}
}

// RejectLoopback creates a ValidURLFunc that returns an error if the given URL is
// a loopback address.
func RejectLoopback() ValidURLFunc {
	return func(u *url.URL) error {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			return fmt.Errorf("%w: %v", errLoopbackGivenAsHost, ip)
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("%w: %v", errNoSuchHost, err)
		}
		for _, i := range ips {
			if i.IsLoopback() {
				return fmt.Errorf("%w: %v lookup includes %v",
					errLoopbackGivenAsHost, host, i)
			}
		}
		return nil
	}
}

// InvalidSubnets checks if the given URL is in any subnets we are blocking and returns
// an error if it is. SpecialIPs will return nil if the URL is not in the subnet.
func InvalidSubnets(i []string) (ValidURLFunc, error) {
	invalidSubnets := []*net.IPNet{}
	for _, sp := range i {
		_, n, err := net.ParseCIDR(sp)
		if err != nil {
			return nil, fmt.Errorf("%w %s: %v", errInvalidSubnet, sp, err)
		}
		invalidSubnets = append(invalidSubnets, n)
	}
	return func(u *url.URL) error {
		ips, err := net.LookupIP(u.Hostname())
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidURL, err)
		}
		for _, d := range ips {
			for _, s := range invalidSubnets {
				if s.Contains(d) {
					return fmt.Errorf("%w: ip %s in %s",
						errIPinInvalidSubnets, d, s)
				}
			}
		}
		return nil
	}, nil
}
