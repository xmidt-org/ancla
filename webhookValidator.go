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
	"regexp"
	"time"
)

var (
	errZeroEvents          = errors.New("cannot have zero events")
	errEventsUnparseable   = errors.New("event cannot be parsed")
	errDeviceIDUnparseable = errors.New("deviceID cannot be parsed")
	errInvalidDuration     = errors.New("duration value of webhook is out of bounds")
	errInvalidUntil        = errors.New("until value of webhook is out of bounds")
	errUntilDurationAbsent = errors.New("until and duration are both absent")
	errInvalidTTL          = errors.New("TTL must be non-negative")
	errInvalidJitter       = errors.New("jitter must be non-negative")
)

// Validator is a WebhookValidator that allows access to the Validate function.
type Validator interface {
	Validate(w Webhook) error
}

// Validators is a WebhookValidator that ensures the webhook is valid with
// each validator in the list.
type Validators []Validator

// ValidatorFunc is a WebhookValidator that takes Webhooks and validates them
// against functions.
type ValidatorFunc func(Webhook) error

// ValidURLFunc takes URLs and ensures they are valid.
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

// Validate runs the function and returns the result. This allows any ValidatorFunc to implement
// the Validator interface.
func (vf ValidatorFunc) Validate(w Webhook) error {
	return vf(w)
}

// CheckEvents makes sure there is at least one value in Events and ensures that
// all values should parse into regex.
func CheckEvents() ValidatorFunc {
	return func(w Webhook) error {
		if len(w.Events) == 0 {
			return errZeroEvents
		}
		for _, e := range w.Events {
			_, err := regexp.Compile(e)
			if err != nil {
				return errEventsUnparseable
			}
		}
		return nil
	}
}

// CheckDeviceID ensures that the DeviceIDs are able to parse into regex.
func CheckDeviceID() ValidatorFunc {
	return func(w Webhook) error {
		for _, i := range w.Matcher.DeviceID {
			_, err := regexp.Compile(i)
			if err != nil {
				return errDeviceIDUnparseable
			}
		}
		return nil
	}
}

// CheckDuration ensures that 0 <= Duration <= ttl. Duration returns an error
// if a negative value is given.
func CheckDuration(maxTTL time.Duration) (ValidatorFunc, error) {
	if maxTTL < 0 {
		return nil, errInvalidTTL
	}
	return func(w Webhook) error {
		if maxTTL < w.Duration || w.Duration < 0 {
			return fmt.Errorf("%w: %v not between 0 and %v",
				errInvalidDuration, w.Duration, maxTTL)
		}
		return nil
	}, nil
}

// CheckUntil ensures that Until, with jitter, is not more than ttl in the future.
func CheckUntil(jitter time.Duration, maxTTL time.Duration, now func() time.Time) (ValidatorFunc, error) {
	if now == nil {
		now = time.Now
	}
	if maxTTL < 0 {
		return nil, errInvalidTTL
	} else if jitter < 0 {
		return nil, errInvalidJitter
	}
	return func(w Webhook) error {
		if w.Until.IsZero() {
			return nil
		}
		limit := (now().Add(maxTTL)).Add(jitter)
		proposed := (w.Until)
		if proposed.After(limit) {
			return fmt.Errorf("%w: %v after %v",
				errInvalidUntil, proposed.String(), limit.String())
		}
		return nil
	}, nil
}

// CheckUntilAndDuration checks if either Until or Duration exists and returns an error
// if neither exist.
func CheckUntilOrDurationExist() ValidatorFunc {
	return func(w Webhook) error {
		if w.Duration == 0 && (w.Until).IsZero() {
			return errUntilDurationAbsent
		}
		return nil
	}
}
