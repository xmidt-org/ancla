// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"time"

	"github.com/xmidt-org/urlegit"
	webhook "github.com/xmidt-org/webhook-schema"
)

type ValidatorConfig struct {
	URL    URLVConfig
	TTL    TTLVConfig
	IP     IPConfig
	Domain DomainConfig
	Opts   OptionsConfig
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

type OptionsConfig struct {
	AtLeastOneEvent                bool
	EventRegexMustCompile          bool
	DeviceIDRegexMustCompile       bool
	ValidateRegistrationDuration   bool
	ProvideReceiverURLValidator    bool
	ProvideFailureURLValidator     bool
	ProvideAlternativeURLValidator bool
	CheckUntil                     bool
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
	if len(config.IP.ForbiddenSubnets) > 0 {
		o = append(o, urlegit.ForbidSubnets(config.IP.ForbiddenSubnets))
	}
	if !config.Domain.AllowSpecialUseDomains {
		o = append(o, urlegit.ForbidSpecialUseDomains())
	}
	if len(config.Domain.ForbiddenDomains) > 0 {
		o = append(o, urlegit.ForbidDomainNames(config.Domain.ForbiddenDomains...))
	}
	return urlegit.New(o...)
}

// BuildOptions translates the configuration into a list of options to be used to validate the registration
func (config *ValidatorConfig) BuildOptions(checker *urlegit.Checker) []webhook.Option {
	var opts []webhook.Option
	if config.Opts.AtLeastOneEvent {
		opts = append(opts, webhook.AtLeastOneEvent())
	}
	if config.Opts.EventRegexMustCompile {
		opts = append(opts, webhook.EventRegexMustCompile())
	}
	if config.Opts.DeviceIDRegexMustCompile {
		opts = append(opts, webhook.DeviceIDRegexMustCompile())
	}
	if config.Opts.ValidateRegistrationDuration {
		opts = append(opts, webhook.ValidateRegistrationDuration(config.TTL.Max))
	}
	if config.Opts.ProvideReceiverURLValidator {
		opts = append(opts, webhook.ProvideReceiverURLValidator(checker))
	}
	if config.Opts.ProvideFailureURLValidator {
		opts = append(opts, webhook.ProvideFailureURLValidator(checker))
	}
	if config.Opts.ProvideAlternativeURLValidator {
		opts = append(opts, webhook.ProvideAlternativeURLValidator(checker))
	}
	if config.Opts.CheckUntil {
		opts = append(opts, webhook.Until(config.TTL.Now, config.TTL.Jitter, config.TTL.Max))
	}
	return opts
}
