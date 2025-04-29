// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"time"

	"github.com/xmidt-org/urlegit"
	webhook "github.com/xmidt-org/webhook-schema"
)

// SchemaURLValidatorConfig provides options for validating the wrpEventStream's URL and TTL
// related fields.
type SchemaURLValidatorConfig struct {
	URL       URLVConfig
	TTL       TTLVConfig
	IP        IPVConfig
	Domain    DomainVConfig
	BuildOpts BuildOptions
}

type IPVConfig struct {
	Allow            bool
	ForbiddenSubnets []string
}

type DomainVConfig struct {
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

// BuildOptions translates the configuration into a list of options to be used to validate the registration
type BuildOptions struct {
	AtLeastOneEvent                bool
	EventRegexMustCompile          bool
	DeviceIDRegexMustCompile       bool
	ValidateRegistrationDuration   bool
	ProvideReceiverURLValidator    bool
	ProvideFailureURLValidator     bool
	ProvideAlternativeURLValidator bool
	CheckUntil                     bool
}

// BuildURLChecker translates the configuration into url Checker to be run on the wrpEventStream.
func (config *SchemaURLValidatorConfig) BuildURLChecker() (*urlegit.Checker, error) {
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
func (config *SchemaURLValidatorConfig) BuildOptions(checker *urlegit.Checker) []webhook.Option {
	var opts []webhook.Option
	if config.BuildOpts.AtLeastOneEvent {
		opts = append(opts, webhook.AtLeastOneEvent())
	}
	if config.BuildOpts.EventRegexMustCompile {
		opts = append(opts, webhook.EventRegexMustCompile())
	}
	if config.BuildOpts.DeviceIDRegexMustCompile {
		opts = append(opts, webhook.DeviceIDRegexMustCompile())
	}
	if config.BuildOpts.ValidateRegistrationDuration {
		opts = append(opts, webhook.ValidateRegistrationDuration(config.TTL.Max))
	}
	if config.BuildOpts.ProvideReceiverURLValidator {
		opts = append(opts, webhook.ProvideReceiverURLValidator(checker))
	}
	if config.BuildOpts.ProvideFailureURLValidator {
		opts = append(opts, webhook.ProvideFailureURLValidator(checker))
	}
	if config.BuildOpts.ProvideAlternativeURLValidator {
		opts = append(opts, webhook.ProvideAlternativeURLValidator(checker))
	}
	if config.BuildOpts.CheckUntil {
		opts = append(opts, webhook.Until(config.TTL.Now, config.TTL.Jitter, config.TTL.Max))
	}
	return opts
}
