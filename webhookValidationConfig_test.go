package ancla

// import (
// 	"errors"
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// var (
// 	mockMax        = 5 * time.Minute
// 	mockJitter     = 5 * time.Second
// 	buildAllConfig = ValidatorConfig{
// 		URL: URLVConfig{
// 			Schemes:       []string{"https"},
// 			AllowLoopback: false,
// 		},
// 		IP: IPConfig{
// 			Allow: false,
// 		},
// 		Domain: DomainConfig{
// 			AllowSpecialUseDomains: false,
// 		},
// 		TTL: TTLVConfig{
// 			Max:    mockMax,
// 			Jitter: mockJitter,
// 			Now:    mockNow,
// 		},
// 	}
// 	buildNoneConfig = ValidatorConfig{
// 		URL: URLVConfig{
// 			Schemes:       []string{"https", "http"},
// 			AllowLoopback: true,
// 		},
// 		IP: IPConfig{
// 			Allow: true,
// 		},
// 		Domain: DomainConfig{
// 			AllowSpecialUseDomains: true,
// 		},
// 		TTL: TTLVConfig{
// 			Max:    mockMax,
// 			Jitter: mockJitter,
// 			Now:    mockNow,
// 		},
// 	}
// )

// func TestBuildValidURLFuncs(t *testing.T) {
// 	tcs := []struct {
// 		desc              string
// 		config            ValidatorConfig
// 		expectedErr       error
// 		expectedFuncCount int
// 	}{
// 		{
// 			desc: "HTTPSOnly only",
// 			config: ValidatorConfig{
// 				URL: URLVConfig{
// 					AllowLoopback: true,
// 					Schemes:       []string{"https"},
// 				},
// 			},
// 			expectedFuncCount: 1,
// 		},
// 		{
// 			desc: "AllowLoopback only",
// 			config: ValidatorConfig{
// 				URL: URLVConfig{
// 					AllowLoopback: false,
// 					Schemes:       []string{"https", "http"},
// 				},
// 			},
// 			expectedFuncCount: 2,
// 		},
// 		{
// 			desc: "AllowIp Only",
// 			config: ValidatorConfig{
// 				IP: IPConfig{
// 					Allow: false,
// 				},
// 			},
// 			expectedFuncCount: 2,
// 		},
// 		{
// 			desc: "AllowSpecialUseHosts Only",
// 			config: ValidatorConfig{
// 				Domain: DomainConfig{
// 					AllowSpecialUseDomains: false,
// 				},
// 			},
// 			expectedFuncCount: 2,
// 		},
// 		{
// 			desc: "AllowSpecialuseIPS Only",
// 			config: ValidatorConfig{
// 				IP: IPConfig{
// 					Allow: true,
// 				},
// 			},
// 			expectedFuncCount: 2,
// 		},
// 		{
// 			desc: "InvalidSubnet Failure",
// 			config: ValidatorConfig{
// 				IP: IPConfig{
// 					Allow:            false,
// 					ForbiddenSubnets: []string{"https://localhost:9000"},
// 				},
// 				Domain: DomainConfig{
// 					AllowSpecialUseDomains: true,
// 				},
// 				URL: URLVConfig{
// 					HTTPSOnly:            false,
// 					AllowLoopback:        true,
// 					AllowIP:              true,
// 					AllowSpecialUseHosts: true,
// 					AllowSpecialUseIPs:   false,
// 					InvalidSubnets:       []string{"https://localhost:9000"},
// 				},
// 			},
// 			expectedErr: fmt.Errorf("error"),
// 		},
// 		{
// 			desc: "Build None",
// 			config: ValidatorConfig{
// 				URL: buildNoneConfig.URL,
// 			},
// 			expectedFuncCount: 1,
// 		},
// 		{
// 			desc: "Build All",
// 			config: ValidatorConfig{
// 				URL: buildAllConfig.URL,
// 			},
// 			expectedFuncCount: 5,
// 		},
// 	}
// 	for _, tc := range tcs {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			assert := assert.New(t)
// 			vals, err := tc.config.GetValidator()
// 			if tc.expectedErr != nil {
// 				assert.True(errors.Is(err, tc.expectedErr),
// 					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
// 						err, tc.expectedErr))
// 				assert.Nil(vals)
// 				return
// 			}
// 			require.NoError(t, err)
// 		})
// 	}
// }

// func TestBuildValidators(t *testing.T) {
// 	tcs := []struct {
// 		desc              string
// 		config            ValidatorConfig
// 		expectedErr       error
// 		expectedFuncCount int
// 	}{
// 		{
// 			desc: "BuildValidURLFuncs Failure",
// 			config: ValidatorConfig{
// 				URL: URLVConfig{
// 					HTTPSOnly:            false,
// 					AllowLoopback:        true,
// 					AllowIP:              true,
// 					AllowSpecialUseHosts: true,
// 					AllowSpecialUseIPs:   false,
// 					InvalidSubnets:       []string{"https://localhost:9000"},
// 				},
// 			},
// 			expectedErr: fmt.Errorf("error"),
// 		},
// 		{
// 			desc: "CheckDuration Failure",
// 			config: ValidatorConfig{
// 				TTL: TTLVConfig{
// 					Max: -1 * time.Second,
// 				},
// 			},
// 			expectedErr: fmt.Errorf("error"),
// 		},
// 		{
// 			desc: "CheckUntil Failure",
// 			config: ValidatorConfig{
// 				TTL: TTLVConfig{
// 					Jitter: -1 * time.Second,
// 				},
// 			},
// 			expectedErr: fmt.Errorf("error"),
// 		},
// 		{
// 			desc:              "All Validators Added",
// 			expectedFuncCount: 8,
// 		},
// 	}
// 	for _, tc := range tcs {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			assert := assert.New(t)
// 			opts := BuildOptions(tc.config, nil)
// 			assert.NotNil(opts)
// 		})
// 	}
// }
