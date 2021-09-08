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
			HTTPSOnly:            true,
			AllowLoopback:        false,
			AllowIP:              false,
			AllowSpecialUseHosts: false,
			AllowSpecialUseIPs:   false,
			InvalidHosts:         []string{},
			InvalidSubnets:       []string{},
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
	}
	buildNoneConfig = ValidatorConfig{
		URL: URLVConfig{
			HTTPSOnly:            false,
			AllowLoopback:        true,
			AllowIP:              true,
			AllowSpecialUseHosts: true,
			AllowSpecialUseIPs:   true,
			InvalidHosts:         []string{},
			InvalidSubnets:       []string{},
		},
		TTL: TTLVConfig{
			Max:    mockMax,
			Jitter: mockJitter,
			Now:    mockNow,
		},
	}
)

//just check if validators is nil or not and check errors
//or check webhooks and see if the validators did what they were supposed to on them

func TestBuildValidURLFuncs(t *testing.T) {
	tcs := []struct {
		desc            string
		config          ValidatorConfig
		expectedErr     error
		expectedOutcome int
	}{
		{
			desc: "HTTPSOnly only",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            true,
					AllowLoopback:        true,
					AllowIP:              true,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   true,
				},
			},
			expectedOutcome: 1,
		},
		{
			desc: "AllowLoopback only",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        false,
					AllowIP:              true,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   true,
				},
			},
			expectedOutcome: 1,
		},
		{
			desc: "AllowIp Only",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        true,
					AllowIP:              false,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   true,
				},
			},
			expectedOutcome: 1,
		},
		{
			desc: "AllowSpecialUseHosts Only",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        true,
					AllowIP:              true,
					AllowSpecialUseHosts: false,
					AllowSpecialUseIPs:   true,
				},
			},
			expectedOutcome: 1,
		},
		{
			desc: "AllowSpecialuseIPS Only",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        true,
					AllowIP:              true,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   false,
				},
			},
			expectedOutcome: 1,
		},
		{
			desc: "InvalidSubnet Failure",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        true,
					AllowIP:              true,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   false,
					InvalidSubnets:       []string{"https://localhost:9000"},
				},
			},
			expectedErr: errFailedToBuildValidURLFuncs,
		},
		{
			desc: "Build None",
			config: ValidatorConfig{
				URL: buildNoneConfig.URL,
			},
			expectedOutcome: 0,
		},
		{
			desc: "Build All",
			config: ValidatorConfig{
				URL: buildAllConfig.URL,
			},
			expectedOutcome: 5,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			vals, err := BuildValidURLFuncs(tc.config)
			if tc.expectedErr != nil {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr))
				assert.Nil(vals)
				return
			}
			require.NoError(t, err)
			assert.Equal(tc.expectedOutcome, len(vals))
		})
	}
}

func TestBuildValidators(t *testing.T) {
	tcs := []struct {
		desc            string
		config          ValidatorConfig
		expectedErr     error
		expectedOutcome int
	}{
		{
			desc: "BuildValidURLFuncs Failure",
			config: ValidatorConfig{
				URL: URLVConfig{
					HTTPSOnly:            false,
					AllowLoopback:        true,
					AllowIP:              true,
					AllowSpecialUseHosts: true,
					AllowSpecialUseIPs:   false,
					InvalidSubnets:       []string{"https://localhost:9000"},
				},
			},
			expectedErr: errFailedToBuildValidators,
		},
		{
			desc: "CheckDuration Failure",
			config: ValidatorConfig{
				TTL: TTLVConfig{
					Max: -1 * time.Second,
				},
			},
			expectedErr: errFailedToBuildValidators,
		},
		{
			desc: "CheckUntil Failure",
			config: ValidatorConfig{
				TTL: TTLVConfig{
					Jitter: -1 * time.Second,
				},
			},
			expectedErr: errFailedToBuildValidators,
		},
		{
			desc:            "All Validators Added",
			expectedOutcome: 8,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			vals, err := BuildValidators(tc.config)
			if tc.expectedErr != nil {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr))
				assert.Nil(vals)
				return
			}
			require.NoError(t, err)
			assert.NotNil(vals)
			assert.Equal(tc.expectedOutcome, len(vals))
		})
	}
}
