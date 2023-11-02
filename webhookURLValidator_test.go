// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	goodURLWebhook = Webhook{
		Config: DeliveryConfig{
			URL:             "https://www.google.com/",
			AlternativeURLs: []string{"https://www.google.com/", "https://www.bing.com/"}},
		FailureURL: "https://www.google.com:1030/software/index.html"}
	simpleFuncs = []ValidURLFunc{GoodURLScheme(true), RejectAllIPs()}
)

func TestValidate(t *testing.T) {
	var mockError error = errors.New("mock")
	var mockFunc ValidatorFunc = func(w Webhook) error { return nil }
	var mockFuncTwo ValidatorFunc = func(w Webhook) error { return mockError }

	goodFuncs := []Validator{mockFunc, mockFunc}
	badFuncs := []Validator{mockFuncTwo}
	tcs := []struct {
		desc        string
		validators  Validators
		expectedErr error
	}{
		{
			desc:       "Empty Validators Success",
			validators: []Validator{},
		},
		{
			desc: "Nil Validators Success",
		},
		{
			desc:       "Valid Validators Success",
			validators: goodFuncs,
		},
		{
			desc:        "Validator error Failure",
			validators:  badFuncs,
			expectedErr: mockError,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := (tc.validators).Validate(Webhook{})
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestGoodConfigURL(t *testing.T) {
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
			validURLFuncs: simpleFuncs,
		},
		{
			desc: "Parse Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "\\\\"},
				FailureURL: ""},
			expectedErr:   errInvalidURL,
			validURLFuncs: simpleFuncs,
		},
		{
			desc: "No https url Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "http://www.google.com/"},
				FailureURL: "https://www.google.com/"},
			expectedErr:   errInvalidURL,
			validURLFuncs: simpleFuncs,
		},
		{
			desc:          "All URL Success",
			webhook:       goodURLWebhook,
			validURLFuncs: simpleFuncs,
		},
		{
			desc:    "Nil validURLFunc input Success",
			webhook: goodURLWebhook,
		},
		{
			desc:          "Empty validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{},
		},
		{
			desc:          "Nil validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{nil},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodConfigURL(tc.validURLFuncs)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestGoodFailureURL(t *testing.T) {
	tcs := []struct {
		desc          string
		webhook       Webhook
		expectedErr   error
		validURLFuncs []ValidURLFunc
	}{
		{
			desc: "Bad FailureURL scheme Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "127.0.0.1:1030"},
			expectedErr:   errInvalidFailureURL,
			validURLFuncs: simpleFuncs,
		},
		{
			desc: "Bad FailureURL Failure",
			webhook: Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"},
				FailureURL: "https://127.0.0.1:1030"},
			expectedErr:   errInvalidFailureURL,
			validURLFuncs: simpleFuncs,
		},
		{
			desc:          "Nil FailureURL Success",
			webhook:       Webhook{Config: DeliveryConfig{URL: "https://www.google.com/"}},
			validURLFuncs: simpleFuncs,
		},
		{
			desc:          "All URL Success",
			webhook:       goodURLWebhook,
			validURLFuncs: simpleFuncs,
		},
		{
			desc:    "Nil validURLFunc input Success",
			webhook: goodURLWebhook,
		},
		{
			desc:          "Empty validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{},
		},
		{
			desc:          "Nil validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{nil},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodFailureURL(tc.validURLFuncs)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestGoodAlternativeURLs(t *testing.T) {
	tcs := []struct {
		desc          string
		webhook       Webhook
		expectedErr   error
		validURLFuncs []ValidURLFunc
	}{
		{
			desc:          "All URL Success",
			webhook:       goodURLWebhook,
			validURLFuncs: simpleFuncs,
		},
		{
			desc: "Bad AlternativeURLs Failure",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"https://www.google.com/", "http://www.bing.com/"}},
				FailureURL: "https://www.google.com:1030/software/index.html"},
			validURLFuncs: simpleFuncs,
			expectedErr:   errInvalidAlternativeURL,
		},
		{
			desc: "Nil String AlternativeURLs Failure",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{""}}},
			expectedErr: errInvalidAlternativeURL,
		},
		{
			desc: "Unparseable AlternativeURLs Failure",
			webhook: Webhook{
				Config: DeliveryConfig{
					URL:             "https://www.google.com/",
					AlternativeURLs: []string{"www.google.com/", "http://www.bing.com/"}}},
			expectedErr: errInvalidAlternativeURL,
		},
		{
			desc:    "Nil validURLFunc input Success",
			webhook: goodURLWebhook,
		},
		{
			desc:          "Empty validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{},
		},
		{
			desc:          "Nil validURLFunc slice Success",
			webhook:       goodURLWebhook,
			validURLFuncs: []ValidURLFunc{nil},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := GoodAlternativeURLs(tc.validURLFuncs)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestGoodURLScheme(t *testing.T) {
	tcs := []struct {
		desc        string
		url         string
		expectedErr error
		httpsOnly   bool
	}{
		{
			desc:        "No https URL Failure",
			url:         "http://www.google.com/",
			expectedErr: errURLIsNotHTTPS,
			httpsOnly:   true,
		},
		{
			desc:      "No https URL Success",
			url:       "http://www.google.com/",
			httpsOnly: false,
		},
		{
			desc:        "Spongebob protocol Failure",
			url:         "spongebob://96.0.0.1:80/responder",
			expectedErr: errBadURLProtocol,
			httpsOnly:   true,
		},
		{
			desc:        "URL with no scheme Failure",
			url:         "www.example.com:1030/software/index.html",
			expectedErr: errBadURLProtocol,
			httpsOnly:   true,
		},
		{
			desc:      "Good https Success",
			url:       "https://localhost:9000",
			httpsOnly: true,
		},
		{
			desc:        "Path with no scheme Failure",
			url:         "/example/test",
			expectedErr: errBadURLProtocol,
			httpsOnly:   true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			u, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			res := GoodURLScheme(tc.httpsOnly)(u)
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
			expectedErr: errLoopbackGivenAsHost,
		},
		{
			desc:        "Unparseable Host Failure",
			url:         "https://localhost:9000:::2",
			expectedErr: errNoSuchHost,
		},
		{
			desc: "IP Host Success",
			url:  "http://96.118.133.128",
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

func TestInvalidSubnets(t *testing.T) {
	tcs := []struct {
		desc               string
		url                string
		subnetsList        []string
		expectedInitialErr error
		expectedLatterErr  error
	}{
		{
			desc:               "Invalid subnet provided Failure",
			url:                "https://2001:db8:a0b:12f0::1/32",
			subnetsList:        []string{"2001:db8:a0b:12f0::1//32"},
			expectedInitialErr: errInvalidSubnet,
		},
		{
			desc:              "IP in subnet Failure",
			url:               "https://192.0.2.56",
			subnetsList:       []string{"192.0.2.1/24"},
			expectedLatterErr: errIPinInvalidSubnets,
		},
		{
			desc: "Nil subnet Success",
			url:  "https://192.0.2.56:1030",
		},
		{
			desc:        "Valid IP given with valid subnets Success",
			url:         "https://[2001:db8:85a3:8d3:1319:8a2e:370:7348]:443/",
			subnetsList: []string{"192.0.2.1/24"},
		},
		{
			desc:              "Invalid URL Failure",
			url:               "/g/a",
			expectedLatterErr: errInvalidURL,
		},
		{
			desc:              "non-IP hostname in Subnet Failure",
			url:               "https://localhost:9000",
			subnetsList:       []string{"127.0.0.1/32"},
			expectedLatterErr: errIPinInvalidSubnets,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			url, err := url.ParseRequestURI(tc.url)
			assert.NoError(err)
			f, err := InvalidSubnets(tc.subnetsList)
			if tc.expectedInitialErr != nil {
				assert.True(errors.Is(err, tc.expectedInitialErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedInitialErr))
				assert.Nil(f)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f)
			err = f(url)
			assert.True(errors.Is(err, tc.expectedLatterErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedLatterErr))
		})
	}
}
