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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	positiveFiveDuration, _  = time.ParseDuration("5m")
	negativeFiveDuration, _  = time.ParseDuration("-5m")
	positiveThreeDuration, _ = time.ParseDuration("3m")
	negativeThreeDuration, _ = time.ParseDuration("-3m")
	OneSecDuration, _        = time.ParseDuration("1s")
)

func TestCheckEvents(t *testing.T) {
	tcs := []struct {
		desc        string
		webhook     Webhook
		expectedErr error
	}{
		{
			desc:        "Empty Webhook Failure",
			webhook:     Webhook{},
			expectedErr: errZeroEvents,
		},
		{
			desc:        "Empty slice Failure",
			webhook:     Webhook{Events: []string{}},
			expectedErr: errZeroEvents,
		},
		{
			desc:        "Unparseable event Failure",
			webhook:     Webhook{Events: []string{"google", `\M`}},
			expectedErr: errEventsUnparseable,
		},
		{
			desc:    "2 parseable events Success",
			webhook: Webhook{Events: []string{"google", "bing"}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := CheckEvents()(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestCheckDeviceID(t *testing.T) {
	tcs := []struct {
		desc        string
		webhook     Webhook
		expectedErr error
	}{
		{
			desc:    "Nil DeviceID Success",
			webhook: Webhook{},
		},
		{
			desc:    "Empty slice Success",
			webhook: Webhook{Events: []string{}},
		},
		{
			desc: "Unparseable deviceID Failure",
			webhook: Webhook{Matcher: MetadataMatcherConfig{
				DeviceID: []string{"", `\M`}}},
			expectedErr: errDeviceIDUnparseable,
		},
		{
			desc: "Parseable deviceID Success",
			webhook: Webhook{Matcher: MetadataMatcherConfig{
				DeviceID: []string{"google"}}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := CheckDeviceID()(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestCheckDuration(t *testing.T) {
	tcs := []struct {
		desc        string
		ttl         time.Duration
		webhook     Webhook
		expectedErr error
	}{
		{
			desc:        "Invalid ttl Failure",
			ttl:         negativeFiveDuration,
			webhook:     Webhook{},
			expectedErr: errInvalidTTL,
		},
		{
			desc:        "Duration out of lower bounds Failure",
			ttl:         positiveFiveDuration,
			webhook:     Webhook{Duration: negativeFiveDuration},
			expectedErr: errInvalidDuration,
		},
		{
			desc:        "Duration out of upper bounds Failure",
			ttl:         positiveThreeDuration,
			webhook:     Webhook{Duration: negativeFiveDuration},
			expectedErr: errInvalidDuration,
		},
		{
			desc:    "Duration Success",
			ttl:     positiveFiveDuration,
			webhook: Webhook{Duration: positiveThreeDuration},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := CheckDuration(tc.ttl)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestCheckUntil(t *testing.T) {
	var mockNow func() time.Time = func() time.Time {
		return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}

	tcs := []struct {
		desc        string
		jitter      time.Duration
		ttl         time.Duration
		now         func() time.Time
		webhook     Webhook
		expectedErr error
	}{
		{
			desc:    "No Until given Success",
			jitter:  OneSecDuration,
			ttl:     positiveThreeDuration,
			now:     mockNow,
			webhook: Webhook{},
		},
		{
			desc:    "Until Success",
			jitter:  OneSecDuration,
			ttl:     positiveFiveDuration,
			now:     mockNow,
			webhook: Webhook{Until: time.Date(2009, time.November, 10, 23, 2, 0, 0, time.UTC)},
		},
		{
			desc:        "Invalid jitter Failure",
			jitter:      negativeFiveDuration,
			ttl:         positiveFiveDuration,
			now:         mockNow,
			webhook:     Webhook{},
			expectedErr: errInvalidJitter,
		},
		{
			desc:        "Invalid ttl Failure",
			jitter:      positiveThreeDuration,
			ttl:         negativeThreeDuration,
			now:         mockNow,
			webhook:     Webhook{},
			expectedErr: errInvalidTTL,
		},
		{
			desc:        "Out of bounds Until Failure",
			jitter:      OneSecDuration,
			ttl:         positiveFiveDuration,
			now:         mockNow,
			webhook:     Webhook{Until: time.Date(2009, time.November, 10, 23, 6, 0, 0, time.UTC)},
			expectedErr: errInvalidUntil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := CheckUntil(tc.jitter, tc.ttl, tc.now)(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func TestCheckUntilOrDurationExist(t *testing.T) {
	tcs := []struct {
		desc        string
		webhook     Webhook
		expectedErr error
	}{
		{
			desc:        "Until and Duration not given Failure",
			webhook:     Webhook{},
			expectedErr: errUntilDurationAbsent,
		},
		{
			desc:    "Only Until given Success",
			webhook: Webhook{Until: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
		},
		{
			desc:    "Only Duration given Success",
			webhook: Webhook{Duration: positiveFiveDuration},
		},
		{
			desc:    "Until and Duration given Success",
			webhook: Webhook{Until: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC), Duration: OneSecDuration},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			err := CheckUntilOrDurationExist()(tc.webhook)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}
