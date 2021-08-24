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
		desc        string
		url         Webhook
		expectedErr error
	}{
		{
			desc: "Blank String",
			url: Webhook{Config: DeliveryConfig{URL: ""},
				FailureURL: ""},
			expectedErr: errInvalidURL,
		},
		{
			desc: "No https url",
			url: Webhook{Config: DeliveryConfig{URL: "http://www.google.com/"},
				FailureURL: "https://www.google.com/"},
			expectedErr: errInvalidURL,
		},
		{
			desc: "Good case",
			url: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://www.google.com:1030/software/index.html"},
		},
		{
			desc: "Bad Failure URL scheme",
			url: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "127.0.0.1:1030"},
			expectedErr: errInvalidFailureURL,
		},
		{
			desc: "Bad Failure URL",
			url: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://127.0.0.1:1030"},
			expectedErr: errInvalidFailureURL,
		},
		{
			desc: "Good case",
			url: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
		},
		{
			desc: "Bad Alternative URLs",
			url: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "http://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			expectedErr: errInvalidAlternativeURL,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodURL([]ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()})(tc.url)
			assert.True(errors.Is(err, tc.expectedErr))
		})
	}
}

func TestHTTPSOnlyEndpoints(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
	}{
		{
			desc:        "No https URL",
			url:         "http://www.google.com/",
			expectedErr: errURLIsNotHTTPS,
		},
		{
			desc:        "Invalid host with Port",
			url:         "http://www.example.com:1030/software/index.html",
			expectedErr: errURLIsNotHTTPS,
		},
		{
			desc: "Loopback URL with Port",
			url:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.url)
			err := HTTPSOnlyEndpoints()(u)
			assert.True(errors.Is(err, tc.expectedErr))
		})
	}
}

func TestRejectHosts(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
	}{
		{
			desc: "No https URL",
			url:  "http://www.google.com/",
		},
		{
			desc:        "Invalid host with Port",
			url:         "http://www.example.com:1030/software/index.html",
			expectedErr: errInvalidHost,
		},
		{
			desc: "Loopback IP with Port",
			url:  "https://127.0.0.1:1030",
		},
		{
			desc: "Loopback URL with Port",
			url:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.url)
			err := RejectHosts([]string{"example"})(u)
			assert.True(errors.Is(err, tc.expectedErr))
		})
	}
}

func TestRejectAllIPs(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
	}{
		{
			desc: "No https URL",
			url:  "http://www.google.com/",
		},
		{
			desc: "Invalid host with Port",
			url:  "http://www.example.com:1030/software/index.html",
		},
		{
			desc:        "Loopback IP with Port",
			url:         "https://127.0.0.1:1030",
			expectedErr: errIPGivenAsHost,
		},
		{
			desc:        "Loopback IP",
			url:         "https://127.0.0.1",
			expectedErr: errIPGivenAsHost,
		},

		{
			desc: "Loopback URL with Port",
			url:  "https://localhost:9000",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.url)
			err := RejectAllIPs()(u)
			assert.True(errors.Is(err, tc.expectedErr))
		})
	}
}

func TestRejectLoopback(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
	}{
		{
			desc: "No https URL",
			url:  "http://www.google.com/",
		},
		{
			desc: "Invalid host with Port",
			url:  "http://www.example.com:1030/software/index.html",
		},
		{
			desc:        "Loopback IP with Port",
			url:         "https://127.0.0.1:1030",
			expectedErr: errLoopbackGivenAsHost,
		},
		{
			desc:        "Loopback URL with Port",
			url:         "https://localhost:9000",
			expectedErr: errLocalhostGivenAsHost,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, _ := url.ParseRequestURI(tc.url)
			err := RejectLoopback()(u)
			assert.True(errors.Is(err, tc.expectedErr))
		})
	}
}
