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
	"time"
)

var (
	errFailedToBuildValidators = errors.New("failed to build validators")
)

type ValidatorConfig struct {
	URL URLVConfig
	TTL TTLVConfig
}

type URLVConfig struct {
	HTTPSOnly            bool
	allowLoopback        bool
	allowIP              bool
	allowSpecialUseHosts bool
	allowSpecialUseIPs   bool
	invalidHosts         []string
	invalidSubnets       []string
}

type TTLVConfig struct {
	max    time.Duration
	jitter time.Duration
	now    func() time.Time
}

// BuildValidators translates the configuration into a list of validators to be run on the
// webhook.
func BuildValidators(config ValidatorConfig) (Validator, error) {
	var v []ValidURLFunc
	if config.URL.HTTPSOnly {
		v = append(v, HTTPSOnlyEndpoints())
	}
	if !config.URL.allowLoopback {
		v = append(v, RejectLoopback())
	}
	if !config.URL.allowIP {
		v = append(v, RejectAllIPs())
	}
	if !config.URL.allowSpecialUseHosts {
		v = append(v, RejectHosts(config.URL.invalidHosts))
	}
	if !config.URL.allowSpecialUseIPs {
		fInvalidSubnets, err := InvalidSubnets(config.URL.invalidSubnets)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
		} else {
			v = append(v, fInvalidSubnets)
		}
	}
	vs := Validators{
		GoodConfigURL(v),
		GoodFailureURL(v),
		GoodAlternativeURLs(v),
		CheckEvents(),
		CheckDeviceID(),
		CheckUntilOrDurationExist(),
	}
	if fCheckDuration, err := CheckDuration(config.TTL.max); err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
	} else {
		vs = append(vs, fCheckDuration)
	}
	if fCheckUntil, err := CheckUntil(config.TTL.jitter, config.TTL.max, config.TTL.now); err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
	} else {
		vs = append(vs, fCheckUntil)
	}

	return vs, nil
}
