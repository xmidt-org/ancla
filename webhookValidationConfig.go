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
	SpecialUseIPs = []string{
		"0.0.0.0/8",          //local ipv4
		"fe80::/10",          //local ipv6
		"255.255.255.255/32", //broadcast to neighbors
		"2001::/32",          //ipv6 TEREDO prefix
		"2001:5::/32",        //EID space for lisp
		"2002::/16",          //ipv6 6to4
		"fc00::/7",           //ipv6 unique local
		"192.0.0.0/24",       //ipv4 IANA
		"2001:0000::/23",     //ipv6 IANA
		"224.0.0.1/32",       //ipv4 multicast
	}
	SpecialUseHosts = []string{
		".example.",
		".invalid.",
		".test.",
		"localhost",
	}
	errFailedToBuildValidators    = errors.New("failed to build validators")
	errFailedToBuildValidURLFuncs = errors.New("failed to build ValidURLFuncs")
)

type ValidatorConfig struct {
	URL URLVConfig
	TTL TTLVConfig
}

type URLVConfig struct {
	HTTPSOnly            bool
	AllowLoopback        bool
	AllowIP              bool
	AllowSpecialUseHosts bool
	AllowSpecialUseIPs   bool
	InvalidHosts         []string
	InvalidSubnets       []string
}

type TTLVConfig struct {
	Max    time.Duration
	Jitter time.Duration
	Now    func() time.Time
}

// BuildValidURLFuncs translates the configuration into a list of ValidURLFuncs
// to be run on the webhook.
func BuildValidURLFuncs(config ValidatorConfig) ([]ValidURLFunc, error) {
	var v []ValidURLFunc
	if config.URL.HTTPSOnly {
		v = append(v, HTTPSOnlyEndpoints())
	}
	if !config.URL.AllowLoopback {
		v = append(v, RejectLoopback())
	}
	if !config.URL.AllowIP {
		v = append(v, RejectAllIPs())
	}
	if !config.URL.AllowSpecialUseHosts {
		config.URL.InvalidHosts = append(config.URL.InvalidHosts, SpecialUseHosts...)
	}
	if len(config.URL.InvalidHosts) > 0 {
		v = append(v, RejectHosts(config.URL.InvalidHosts))
	}
	if !config.URL.AllowSpecialUseIPs {
		config.URL.InvalidSubnets = append(config.URL.InvalidSubnets, SpecialUseIPs...)
	}
	if len(config.URL.InvalidSubnets) > 0 {
		fInvalidSubnets, err := InvalidSubnets(config.URL.InvalidSubnets)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errFailedToBuildValidURLFuncs, err)
		}
		v = append(v, fInvalidSubnets)
	}
	return v, nil
}

// BuildValidators translates the configuration into a list of validators to be run on the
// webhook.
func BuildValidators(config ValidatorConfig) (Validators, error) {
	v, err := BuildValidURLFuncs(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
	}

	vs := Validators{
		GoodConfigURL(v),
		GoodFailureURL(v),
		GoodAlternativeURLs(v),
		CheckEvents(),
		CheckDeviceID(),
		CheckUntilOrDurationExist(),
	}
	fCheckDuration, err := CheckDuration(config.TTL.Max)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
	}
	vs = append(vs, fCheckDuration)

	fCheckUntil, err := CheckUntil(config.TTL.Jitter, config.TTL.Max, config.TTL.Now)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedToBuildValidators, err)
	}
	vs = append(vs, fCheckUntil)

	return vs, nil
}
