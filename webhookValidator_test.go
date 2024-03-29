// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	positiveFiveDuration  = 5 * time.Minute
	negativeFiveDuration  = -5 * time.Minute
	positiveThreeDuration = 3 * time.Minute
	negativeThreeDuration = -3 * time.Minute
)

func TestAlwaysValid(t *testing.T) {
	err := AlwaysValid()(Webhook{})
	assert.NoError(t, err)
}

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
		desc               string
		ttl                time.Duration
		webhook            Webhook
		expectedInitialErr error
		expectedLatterErr  error
	}{
		{
			desc:               "Invalid ttl Failure",
			ttl:                negativeFiveDuration,
			webhook:            Webhook{},
			expectedInitialErr: errInvalidTTL,
		},
		{
			desc:              "Duration out of lower bounds Failure",
			ttl:               positiveFiveDuration,
			webhook:           Webhook{Duration: negativeFiveDuration},
			expectedLatterErr: errInvalidDuration,
		},
		{
			desc:              "Duration out of upper bounds Failure",
			ttl:               positiveThreeDuration,
			webhook:           Webhook{Duration: negativeFiveDuration},
			expectedLatterErr: errInvalidDuration,
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
			f, err := CheckDuration(tc.ttl)
			if tc.expectedInitialErr != nil {
				assert.True(errors.Is(err, tc.expectedInitialErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedInitialErr))
				assert.Nil(f)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f)
			err = f(tc.webhook)
			assert.True(errors.Is(err, tc.expectedLatterErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedLatterErr))
		})
	}
}

func TestCheckUntil(t *testing.T) {
	var mockNow func() time.Time = func() time.Time {
		return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}

	tcs := []struct {
		desc               string
		jitter             time.Duration
		ttl                time.Duration
		now                func() time.Time
		webhook            Webhook
		expectedInitialErr error
		expectedLatterErr  error
	}{
		{
			desc:    "No Until given Success",
			jitter:  time.Second,
			ttl:     positiveThreeDuration,
			now:     mockNow,
			webhook: Webhook{},
		},
		{
			desc:    "Until Success",
			jitter:  time.Second,
			ttl:     positiveFiveDuration,
			now:     mockNow,
			webhook: Webhook{Until: time.Date(2009, time.November, 10, 23, 2, 0, 0, time.UTC)},
		},
		{
			desc:               "Invalid jitter Failure",
			jitter:             negativeFiveDuration,
			ttl:                positiveFiveDuration,
			now:                mockNow,
			webhook:            Webhook{},
			expectedInitialErr: errInvalidJitter,
		},
		{
			desc:               "Invalid ttl Failure",
			jitter:             positiveThreeDuration,
			ttl:                negativeThreeDuration,
			now:                mockNow,
			webhook:            Webhook{},
			expectedInitialErr: errInvalidTTL,
		},
		{
			desc:              "Out of bounds Until Failure",
			jitter:            time.Second,
			ttl:               positiveFiveDuration,
			now:               mockNow,
			webhook:           Webhook{Until: time.Date(2009, time.November, 10, 23, 6, 0, 0, time.UTC)},
			expectedLatterErr: errInvalidUntil,
		},
		{
			desc:              "Nil Now function Failure",
			jitter:            time.Second,
			ttl:               positiveFiveDuration,
			webhook:           Webhook{Until: time.Now().Add(10000 * time.Hour)},
			expectedLatterErr: errInvalidUntil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			f, err := CheckUntil(tc.jitter, tc.ttl, tc.now)
			if tc.expectedInitialErr != nil {
				assert.True(errors.Is(err, tc.expectedInitialErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedInitialErr))
				assert.Nil(f)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f)
			err = f(tc.webhook)
			assert.True(errors.Is(err, tc.expectedLatterErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedLatterErr))
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
			webhook: Webhook{Until: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC), Duration: time.Second},
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
