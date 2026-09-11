// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/urlegit"
)

var (
	mockNow = func() time.Time {
		return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}
	mockMax        = 5 * time.Minute
	mockJitter     = 5 * time.Second
	buildAllConfig = SchemaURLValidatorConfig{
		URL: URLVConfig{
			Schemes:       []string{"https"},
			AllowLoopback: false,
		},
		IP: IPVConfig{
			Allow: false,
		},
		Domain: DomainVConfig{
			AllowSpecialUseDomains: false,
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
		BuildOpts: BuildOptions{
			AtLeastOneEvent:                true,
			EventRegexMustCompile:          true,
			DeviceIDRegexMustCompile:       true,
			ValidateRegistrationDuration:   true,
			ProvideReceiverURLValidator:    true,
			ProvideFailureURLValidator:     true,
			ProvideAlternativeURLValidator: true,
			CheckUntil:                     true,
		},
	}
	buildNoneConfig = SchemaURLValidatorConfig{
		URL: URLVConfig{
			Schemes:       []string{"https", "http"},
			AllowLoopback: true,
		},
		IP: IPVConfig{
			Allow: true,
		},
		Domain: DomainVConfig{
			AllowSpecialUseDomains: true,
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
	}
	badBuildAllConfig = SchemaURLValidatorConfig{
		URL: URLVConfig{
			Schemes:       []string{"https"},
			AllowLoopback: false,
		},
		IP: IPVConfig{
			Allow: false,
		},
		Domain: DomainVConfig{
			AllowSpecialUseDomains: true,
			ForbiddenDomains:       []string{"example...com."},
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
		BuildOpts: BuildOptions{
			AtLeastOneEvent:                true,
			EventRegexMustCompile:          true,
			DeviceIDRegexMustCompile:       true,
			ValidateRegistrationDuration:   true,
			ProvideReceiverURLValidator:    true,
			ProvideFailureURLValidator:     true,
			ProvideAlternativeURLValidator: true,
			CheckUntil:                     true,
		},
	}
)

func TestBuildValidURLFuncs(t *testing.T) {
	tcs := []struct {
		desc        string
		config      SchemaURLValidatorConfig
		expectedErr error
	}{
		{
			desc: "HTTPSOnly only",
			config: SchemaURLValidatorConfig{
				URL: URLVConfig{
					AllowLoopback: true,
					Schemes:       []string{"https"},
				},
			},
		},
		{
			desc: "AllowLoopback only",
			config: SchemaURLValidatorConfig{
				URL: URLVConfig{
					AllowLoopback: false,
					Schemes:       []string{"https", "http"},
				},
			},
		},
		{
			desc: "AllowIp Only",
			config: SchemaURLValidatorConfig{
				IP: IPVConfig{
					Allow: false,
				},
			},
		},
		{
			desc: "AllowSpecialUseHosts Only",
			config: SchemaURLValidatorConfig{
				Domain: DomainVConfig{
					AllowSpecialUseDomains: false,
				},
			},
		},
		{
			desc: "AllowSpecialuseIPS Only",
			config: SchemaURLValidatorConfig{
				IP: IPVConfig{
					Allow: true,
				},
			},
		},
		{
			desc: "Forbidden Subnets",
			config: SchemaURLValidatorConfig{
				IP: IPVConfig{
					Allow:            false,
					ForbiddenSubnets: []string{"10.0.0.0/8"},
				},
			},
		},
		{
			desc: "Forbidden Domains",
			config: SchemaURLValidatorConfig{
				Domain: DomainVConfig{
					AllowSpecialUseDomains: true,
					ForbiddenDomains:       []string{"example.com."},
				},
			},
		},
		{
			desc: "Invalid Forbidden Domains",
			config: SchemaURLValidatorConfig{
				Domain: DomainVConfig{
					AllowSpecialUseDomains: true,
					ForbiddenDomains:       []string{"example...com."},
				},
			},
			expectedErr: urlegit.ErrInvalidInput,
		},
		{
			desc:   "Build None",
			config: buildNoneConfig,
		},
		{
			desc:   "Build All",
			config: buildAllConfig,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			vals, err := tc.config.BuildURLChecker()
			if tc.expectedErr != nil {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr))

				return
			}

			assert.NoError(err)
			assert.NotZero(vals)
		})
	}
}

func TestBuildOptions(t *testing.T) {
	tcs := []struct {
		desc         string
		config       SchemaURLValidatorConfig
		optionsCount int
		expectedErr  error
	}{
		{
			desc:        "Invalid Forbidden Domains",
			config:      badBuildAllConfig,
			expectedErr: urlegit.ErrInvalidInput,
		},
		{
			desc:   "Build None",
			config: buildNoneConfig,
		},
		{
			desc:         "Build All",
			config:       buildAllConfig,
			optionsCount: 8,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			opts, err := tc.config.BuildOptions()
			if tc.expectedErr != nil {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr))

				return
			}

			assert.NoError(err)
			assert.Len(opts, tc.optionsCount)
		})
	}
}
