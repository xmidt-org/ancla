// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"time"

	"github.com/xmidt-org/urlegit"
)

type ValidatorConfig struct {
	URL    URLVConfig
	TTL    TTLVConfig
	IP     IPConfig
	Domain DomainConfig
}

type IPConfig struct {
	Allow            bool
	ForbiddenSubnets []string
}

type DomainConfig struct {
	AllowSpecialUseDomains bool
	ForbiddenDomains       []string
}

type URLVConfig struct {
	Schemes       []string
	AllowLoopback bool
}

type TTLVConfig struct {
	Max    time.Duration
	Jitter time.Duration
	Now    func() time.Time
}

// BuildURLChecker translates the configuration into url Checker to be run on the webhook.
func (config *ValidatorConfig) BuildURLChecker() (*urlegit.Checker, error) {
	var o []urlegit.Option
	if len(config.URL.Schemes) > 0 {
		o = append(o, urlegit.OnlyAllowSchemes(config.URL.Schemes...))
	}
	if !config.URL.AllowLoopback {
		o = append(o, urlegit.ForbidLoopback())
	}
	if !config.IP.Allow {
		o = append(o, urlegit.ForbidAnyIPs())
	}
	if !config.Domain.AllowSpecialUseDomains {
		o = append(o, urlegit.ForbidSpecialUseDomains())
	}
	checker, err := urlegit.New(o...)
	if err != nil {
		return nil, err
	}
	return checker, nil

}
