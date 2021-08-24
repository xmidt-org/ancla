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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ Validator = GoodURL([]ValidURLFunc{HTTPSOnlyEndpoints(), RejectHosts(nil), RejectAllIPs()})
)

func TestGoodURL(t *testing.T) {
	tcs := []struct {
		Desc        string
		URL         Webhook
		ExpectedErr error
	}{
		{
			Desc: "Blank String",
			URL: Webhook{Config: DeliveryConfig{URL: ""},
				FailureURL: ""},
			ExpectedErr: errInvalidURL,
		},
		{
			Desc: "No https URL",
			URL: Webhook{Config: DeliveryConfig{URL: "http://www.google.com/"},
				FailureURL: "https://www.google.com/"},
			ExpectedErr: errInvalidURL,
		},
		{
			Desc: "Good case",
			URL: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://www.google.com:1030/software/index.html"},
		},
		{
			Desc: "Bad Failure URL scheme",
			URL: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "127.0.0.1:1030"},
			ExpectedErr: errInvalidFailureURL,
		},
		{
			Desc: "Bad Failure URL",
			URL: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://127.0.0.1:1030"},
			ExpectedErr: errInvalidFailureURL,
		},
		{
			Desc: "Good case",
			URL: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
		},
		{
			Desc: "Bad Alternative URLs",
			URL: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "http://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			ExpectedErr: errInvalidAlternativeURL,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodURL([]ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()})(tc.URL)
			assert.True(errors.Is(err, tc.ExpectedErr))
		})
	}
}

func TestHTTPSOnlyEndpoints(t *testing.T) {
	tcs := []struct {
		Desc        string
		URL         string
		ExpectedErr error
	}{
		{
			Desc:        "No https URL",
			URL:         "http://www.google.com/",
			ExpectedErr: errURLIsNotHTTPS,
		},
		{
			Desc:        "Invalid host with Port",
			URL:         "http://www.example.com:1030/software/index.html",
			ExpectedErr: errURLIsNotHTTPS,
		},
		{
			Desc: "Loopback URL with Port",
			URL:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.URL)
			err := HTTPSOnlyEndpoints()(u)
			assert.True(errors.Is(err, tc.ExpectedErr))
		})
	}
}

func TestRejectHosts(t *testing.T) {
	tcs := []struct {
		Desc        string
		URL         string
		ExpectedErr error
	}{
		{
			Desc: "No https URL",
			URL:  "http://www.google.com/",
		},
		{
			Desc:        "Invalid host with Port",
			URL:         "http://www.example.com:1030/software/index.html",
			ExpectedErr: errInvalidHost,
		},
		{
			Desc: "Loopback IP with Port",
			URL:  "https://127.0.0.1:1030",
		},
		{
			Desc: "Loopback URL with Port",
			URL:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.URL)
			err := RejectHosts([]string{"example"})(u)
			assert.True(errors.Is(err, tc.ExpectedErr))
		})
	}
}

func TestRejectAllIPs(t *testing.T) {
	tcs := []struct {
		Desc        string
		URL         string
		ExpectedErr error
	}{
		{
			Desc: "No https URL",
			URL:  "http://www.google.com/",
		},
		{
			Desc: "Invalid host with Port",
			URL:  "http://www.example.com:1030/software/index.html",
		},
		{
			Desc:        "Loopback IP with Port",
			URL:         "https://127.0.0.1:1030",
			ExpectedErr: errIPGivenAsHost,
		},
		{
			Desc:        "Loopback IP",
			URL:         "https://127.0.0.1",
			ExpectedErr: errIPGivenAsHost,
		},

		{
			Desc: "Loopback URL with Port",
			URL:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.URL)
			err := RejectAllIPs()(u)
			assert.True(errors.Is(err, tc.ExpectedErr))
		})
	}
}

func TestRejectLoopback(t *testing.T) {
	tcs := []struct {
		Desc        string
		URL         string
		ExpectedErr error
	}{
		{
			Desc: "No https URL",
			URL:  "http://www.google.com/",
		},
		{
			Desc: "Invalid host with Port",
			URL:  "http://www.example.com:1030/software/index.html",
		},
		{
			Desc:        "Loopback IP with Port",
			URL:         "https://127.0.0.1:1030",
			ExpectedErr: errLoopbackGivenAsHost,
		},
		{
			Desc:        "Loopback URL with Port",
			URL:         "https://localhost:9000",
			ExpectedErr: errLocalHostGivenAsHost,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.URL)
			err := RejectLoopback()(u)
			assert.True(errors.Is(err, tc.ExpectedErr))
		})
	}
}
