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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_ Validator = GoodURL([]ValidURLFunc{HTTPSOnlyEndpoints(), RejectHosts(nil), RejectAllIPs()})
)

func TestGoodURL(t *testing.T) {
	tcs := []struct {
		desc          string
		webhook       Webhook
		expectedErr   error
		validURLFuncs []ValidURLFunc
	}{
		{
			desc: "Blank String Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: ""},
				FailureURL: ""},
			expectedErr:   errInvalidURL,
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
		},
		{
			desc: "No https url Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "http://www.google.com/"},
				FailureURL: "https://www.google.com/"},
			expectedErr:   errInvalidURL,
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
		},
		{
			desc: "Bad FailureURL scheme Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "127.0.0.1:1030"},
			expectedErr:   errInvalidFailureURL,
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
		},
		{
			desc: "Bad FailureURL Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://127.0.0.1:1030"},
			expectedErr:   errInvalidFailureURL,
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
		},
		{
			desc: "All URL Success",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
		},
		{
			desc: "Bad AlternativeURLs Failure",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "http://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			validURLFuncs: []ValidURLFunc{HTTPSOnlyEndpoints(), RejectAllIPs()},
			expectedErr:   errInvalidAlternativeURL,
		},
		{
			desc: "Nil validURLFunc input Success",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
		},
		{
			desc: "Empty validURLFunc slice Success",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			validURLFuncs: []ValidURLFunc{},
		},
		{
			desc: "Nil validURLFunc slice Success",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			validURLFuncs: []ValidURLFunc{nil},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodURL(tc.validURLFuncs)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
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
			desc:        "No https URL Failure",
			url:         "http://www.google.com/",
			expectedErr: errURLIsNotHTTPS,
		},
		{
			desc:        "URL with no scheme Failure",
			url:         "www.example.com:1030/software/index.html",
			expectedErr: errURLIsNotHTTPS,
		},
		{
			desc: "Good https Success",
			url:  "https://localhost:9000",
		},
		{
			desc:        "Path with no scheme Failure",
			url:         "/example/test",
			expectedErr: errURLIsNotHTTPS,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			res := HTTPSOnlyEndpoints()(u)
			assert.True(errors.Is(res, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					res, tc.expectedErr),
			)
		})
	}
}

func TestRejectHosts(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
		rejectHosts []string
	}{
		{
			desc:        "Good host Success",
			url:         "http://www.google.com/",
			rejectHosts: []string{"example", "localhost"},
		},
		{
			desc:        "host:example.com Failure",
			url:         "http://www.example.com:1030/software/index.html",
			rejectHosts: []string{"example", "localhost"},
			expectedErr: errInvalidHost,
		},
		{
			desc:        "host:Localhost Failure",
			url:         "https://localhost:9000",
			rejectHosts: []string{"example", "localhost"},
			expectedErr: errInvalidHost,
		},
		{
			desc:        "No Host Success",
			url:         "/example/test",
			rejectHosts: []string{"example", "localhost"},
		},
		{
			desc:        "Nil rejectedHosts input Success",
			url:         "http://www.google.com/",
			rejectHosts: nil,
		},
		{
			desc:        "Nil string in rejectedHosts Success",
			url:         "http://www.google.com/",
			rejectHosts: []string{""},
		},
		{
			desc:        "Last string in rejectedHosts Failure",
			url:         "http://www.google.com/",
			rejectHosts: []string{"bing", "google"},
			expectedErr: errInvalidHost,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			res := RejectHosts(tc.rejectHosts)(u)
			assert.True(errors.Is(res, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					res, tc.expectedErr),
			)
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
			desc: "Non-IP Success",
			url:  "http://www.example.com:1030/software/index.html",
		},
		{
			desc:        "Loopback IP with Port Failure",
			url:         "https://127.0.0.1:1030",
			expectedErr: errIPGivenAsHost,
		},
		{
			desc:        "Loopback IP with no port Failure",
			url:         "https://127.0.0.1",
			expectedErr: errIPGivenAsHost,
		},
		{
			desc: "No Host Success",
			url:  "/example/test",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			res := RejectAllIPs()(u)
			assert.True(errors.Is(res, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					res, tc.expectedErr),
			)
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
			desc: "Non loopback URL Success",
			url:  "http://www.example.com:1030/software/index.html",
		},
		{
			desc:        "Loopback IP with Port Failure",
			url:         "https://127.0.0.1:1030",
			expectedErr: errLoopbackGivenAsHost,
		},
		{
			desc:        "Loopback URL with Port Failure",
			url:         "https://localhost:9000",
			expectedErr: errLocalhostGivenAsHost,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			res := RejectLoopback()(u)
			assert.True(errors.Is(res, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					res, tc.expectedErr),
			)
		})
	}
}
