package ancla

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/store"
)

type validateTestconfig struct {
	input    *Config
	expected *Config
}

func TestValidateConfig(t *testing.T) {
	type testCase struct {
		Description string
		Data        validateTestconfig
		ExpectedErr error
	}

	tcs := []testCase{
		{
			Description: "Migration config provided without an item owner",
			Data:        getInvalidConfig(),
			ExpectedErr: errMigrationOwnerEmpty,
		},
		{
			Description: "Incomplete but valid config",
			Data:        getIncompleteButValidConfig(),
		},
		{
			Description: "No migration section but still valid",
			Data:        getNoMigrationValidConfig(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			err := validateConfig(tc.Data.input)
			if tc.ExpectedErr != nil {
				assert.Equal(tc.ExpectedErr, err)
			} else {
				assert.Nil(err)
				assert.EqualValues(tc.Data.expected, tc.Data.input)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	type pushItemResults struct {
		result chrysom.PushResult
		err    error
	}
	type testCase struct {
		Description     string
		Owner           string
		InputWebhook    Webhook
		PushItemResults pushItemResults
		ExpectedErr     error
	}

	tcs := []testCase{
		{
			Description: "PushItem fails",
			PushItemResults: pushItemResults{
				err: errors.New("push item failed"),
			},
			ExpectedErr: errFailedWebhookPush,
		},
		{
			Description: "Unknown push result",
			PushItemResults: pushItemResults{
				result: "unknownResult",
			},
			ExpectedErr: errNonSuccessPushResult,
		},
		{
			Description: "Item created",
			PushItemResults: pushItemResults{
				result: chrysom.CreatedPushResult,
			},
		},
		{
			Description: "Item update",
			PushItemResults: pushItemResults{
				result: chrysom.UpdatedPushResult,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockPushReader)
			svc := service{
				logger: log.NewNopLogger(),
				config: Config{},
				argus:  m,
			}
			m.On("PushItem", store.Sha256HexDigest(tc.InputWebhook.Address), svc.config.Bucket, tc.Owner,
				mock.Anything).Return(tc.PushItemResults.result, tc.PushItemResults.err)
			err := svc.Add(tc.Owner, tc.InputWebhook)
			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
			m.AssertExpectations(t)
		})
	}
}

// Test AllWebhooks
func TestAllWebhooks(t *testing.T) {
	//TODO:
}

func getIncompleteButValidConfig() validateTestconfig {
	return validateTestconfig{
		input: &Config{
			Migration: &MigrationConfig{
				Owner: "owner-provided",
			},
		},
		expected: &Config{
			Bucket: "webhooks",
			Migration: &MigrationConfig{
				Owner:  "owner-provided",
				Bucket: "webhooks",
			},
			Logger:          log.NewNopLogger(),
			MetricsProvider: provider.NewDiscardProvider(),
		},
	}
}

func getNoMigrationValidConfig() validateTestconfig {
	logger := log.NewJSONLogger(ioutil.Discard)
	metricsProvider := provider.NewExpvarProvider()

	return validateTestconfig{
		input: &Config{
			Bucket:          "myBucket",
			Logger:          logger,
			MetricsProvider: metricsProvider,
		},
		expected: &Config{
			Bucket:          "myBucket",
			Logger:          logger,
			MetricsProvider: metricsProvider,
		},
	}
}

func getInvalidConfig() validateTestconfig {
	return validateTestconfig{
		input: &Config{
			Migration: &MigrationConfig{
				Bucket: "myBucket",
			},
		},
	}
}
