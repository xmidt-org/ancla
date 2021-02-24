package ancla

import (
	"io/ioutil"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
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
	//TODO:
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
