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
	mockNow = func() time.Time {
		return time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	}

	mockMax        = 5 * time.Minute
	mockJitter     = 5 * time.Second
	buildAllConfig = ValidatorConfig{
		URL: URLVConfig{
			Schemes:       []string{"https"},
			AllowLoopback: false,
		},
		IP: IPConfig{
			Allow: false,
		},
		Domain: DomainConfig{
			AllowSpecialUseDomains: false,
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
		Opts: OptionsConfig{
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
	buildNoneConfig = ValidatorConfig{
		URL: URLVConfig{
			Schemes:       []string{"https", "http"},
			AllowLoopback: true,
		},
		IP: IPConfig{
			Allow: true,
		},
		Domain: DomainConfig{
			AllowSpecialUseDomains: true,
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
	}
)

func TestBuildValidURLFuncs(t *testing.T) {
	tcs := []struct {
		desc              string
		config            ValidatorConfig
		expectedErr       error
		expectedFuncCount int
	}{
		{
			desc: "HTTPSOnly only",
			config: ValidatorConfig{
				URL: URLVConfig{
					AllowLoopback: true,
					Schemes:       []string{"https"},
				},
			},
			expectedFuncCount: 1,
		},
		{
			desc: "AllowLoopback only",
			config: ValidatorConfig{
				URL: URLVConfig{
					AllowLoopback: false,
					Schemes:       []string{"https", "http"},
				},
			},
			expectedFuncCount: 2,
		},
		{
			desc: "AllowIp Only",
			config: ValidatorConfig{
				IP: IPConfig{
					Allow: false,
				},
			},
			expectedFuncCount: 2,
		},
		{
			desc: "AllowSpecialUseHosts Only",
			config: ValidatorConfig{
				Domain: DomainConfig{
					AllowSpecialUseDomains: false,
				},
			},
			expectedFuncCount: 2,
		},
		{
			desc: "AllowSpecialuseIPS Only",
			config: ValidatorConfig{
				IP: IPConfig{
					Allow: true,
				},
			},
			expectedFuncCount: 2,
		},
		{
			desc: "Forbidden Subnets",
			config: ValidatorConfig{
				IP: IPConfig{
					Allow:            false,
					ForbiddenSubnets: []string{"10.0.0.0/8"},
				},
			},
			expectedFuncCount: 1,
		},
		{
			desc: "Forbidden Domains",
			config: ValidatorConfig{
				Domain: DomainConfig{
					AllowSpecialUseDomains: true,
					ForbiddenDomains:       []string{"example.com."},
				},
			},
		},
		{
			desc: "Build None",
			config: ValidatorConfig{
				URL: buildNoneConfig.URL,
			},
			expectedFuncCount: 1,
		},
		{
			desc: "Build All",
			config: ValidatorConfig{
				URL: buildAllConfig.URL,
			},
			expectedFuncCount: 5,
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
				assert.Nil(vals)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildOptions(t *testing.T) {
	checker, err := buildAllConfig.BuildURLChecker()
	assert.NoError(t, err)
	opts := buildAllConfig.BuildOptions(checker)
	assert.NotNil(t, opts)
	assert.Len(t, opts, 8)
}
