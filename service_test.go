package ancla

import (
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/store"
)

var (
	refTime              = getRefTime()
	migrationWebhookZero = Webhook{
		Address: "http://webhook-requester-to-sns.example",
		Config: WebhookConfig{
			URL: "http://events-webhook0-here.example",
		},
		Until: refTime,
	}
	migrationWebhookTwo = Webhook{
		Address: "http://webhook-requester-to-sns.example",
		Config: WebhookConfig{
			URL: "http://events-webhook2-here.example",
		},
		Until: refTime.Add(8 * time.Second),
	}
	webhookZero = Webhook{
		Address: "http://webhook-requester-to-argus.example",
		Config: WebhookConfig{
			URL: "http://events-webhook0-here.example",
		},
		Until: refTime.Add(5 * time.Second),
	}
	webhookOne = Webhook{
		Address: "http://webhook-requester-to-argus.example",
		Config: WebhookConfig{
			URL: "http://events-webhook1-here.example",
		},
		Until: refTime.Add(10 * time.Second),
	}
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

func TestAllWebhooks(t *testing.T) {
	type getItemsMockResp struct {
		Items chrysom.Items
		Err   error
	}

	type testCase struct {
		Description          string
		Owner                string
		CaptureMigratedItems bool
		CaptureItems         bool
		MigrationItemsResp   getItemsMockResp
		ItemsResp            getItemsMockResp
		ExpectedWebhooks     []Webhook
		ExpectedErr          error
	}

	migrationItemsResp := getItemsMockResp{
		Items: getTestMigrationItems(),
	}

	itemsResp := getItemsMockResp{
		Items: getTestItems(),
	}

	tcs := []testCase{
		{
			Description:          "Fetching migrated webhooks fails",
			Owner:                "Owner",
			CaptureMigratedItems: true,
			CaptureItems:         false,
			MigrationItemsResp: getItemsMockResp{
				Err: errors.New("db failed"),
			},
			ItemsResp:   itemsResp,
			ExpectedErr: errFailedMigratedWebhooksFetch,
		},

		{
			Description:          "Fetching argus webhooks fails",
			Owner:                "Owner",
			CaptureMigratedItems: true,
			CaptureItems:         true,
			MigrationItemsResp:   migrationItemsResp,
			ItemsResp: getItemsMockResp{
				Err: errors.New("db failed"),
			},
			ExpectedErr: errFailedWebhooksFetch,
		},
		{
			Description:          "Migration capture disabled. Webhooks fetch success",
			Owner:                "Owner",
			CaptureMigratedItems: false,
			CaptureItems:         true,
			MigrationItemsResp:   migrationItemsResp,
			ItemsResp:            itemsResp,
			ExpectedWebhooks:     getWebhooksFromArgusOnly(),
		},
		{
			Description:          "Only Migrated webhooks",
			Owner:                "Owner",
			CaptureMigratedItems: true,
			CaptureItems:         true,
			MigrationItemsResp:   migrationItemsResp,
			ItemsResp: getItemsMockResp{
				Items: chrysom.Items{},
			},
			ExpectedWebhooks: getWebhooksFromMigrationOnly(),
		},
		{
			Description:          "Success fetch from both sources",
			Owner:                "Owner",
			CaptureMigratedItems: true,
			CaptureItems:         true,
			MigrationItemsResp:   migrationItemsResp,
			ItemsResp:            itemsResp,
			ExpectedWebhooks:     getWebhooksFromMixedSources(),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockPushReader)
			migrationCfg := &MigrationConfig{
				Bucket: "webhooks",
				Owner:  "migration-owner",
			}
			cfg := Config{}
			if tc.CaptureMigratedItems {
				cfg.Migration = migrationCfg
			}

			svc := service{
				argus:  m,
				logger: log.NewNopLogger(),
				config: cfg,
			}
			if tc.CaptureMigratedItems {
				m.On("GetItems", migrationCfg.Bucket, migrationCfg.Owner).
					Return(tc.MigrationItemsResp.Items, tc.MigrationItemsResp.Err)
			}

			if tc.CaptureItems {
				m.On("GetItems", svc.config.Bucket, tc.Owner).Return(tc.ItemsResp.Items, tc.ItemsResp.Err)
			}

			webhooks, err := svc.AllWebhooks(tc.Owner)

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
				assert.Empty(webhooks)
			} else {
				assert.EqualValues(tc.ExpectedWebhooks, webhooks)
			}

			if !tc.CaptureMigratedItems {
				m.AssertNotCalled(t, "GetItems", migrationCfg.Bucket, migrationCfg.Owner)
			}

			if !tc.CaptureItems {
				m.AssertNotCalled(t, "GetItems", svc.config.Bucket, tc.Owner)
			}

			m.AssertExpectations(t)
		})
	}
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

func getTestItems() chrysom.Items {
	return chrysom.Items{
		{
			ID: "d73ec0f8e6137f50284453bf1da67f94659fae70cefef8745a94a638bab41b90",
			Data: map[string]interface{}{
				"registered_from_address": "http://webhook-requester-to-argus.example",
				"config": map[string]interface{}{
					"url": "http://events-webhook0-here.example",
				},
				"until": "2021-01-02T15:04:05Z",
			},
		},
		{
			ID: "900018a2bd769407338e49693d5e8dc91301a10620b79b3e5bffdc8791e43bc6",
			Data: map[string]interface{}{
				"registered_from_address": "http://webhook-requester-to-argus.example",
				"config": map[string]interface{}{
					"url": "http://events-webhook1-here.example",
				},
				"until": "2021-01-02T15:04:10Z",
			},
		},
	}
}

func getTestMigrationItems() chrysom.Items {
	return chrysom.Items{
		{
			ID: "d73ec0f8e6137f50284453bf1da67f94659fae70cefef8745a94a638bab41b90",
			Data: map[string]interface{}{
				"registered_from_address": "http://webhook-requester-to-sns.example",
				"config": map[string]interface{}{
					"url": "http://events-webhook0-here.example",
				},
				"until": "2021-01-02T15:04:00Z",
			},
		},
		{
			ID: "4f4908c2d545216b8996a701866ce1c0312bec2bf316e4cfecf2465634bf8398",
			Data: map[string]interface{}{
				"registered_from_address": "http://webhook-requester-to-sns.example",
				"config": map[string]interface{}{
					"url": "http://events-webhook2-here.example",
				},
				"until": "2021-01-02T15:04:08Z",
			},
		},
	}
}

func getWebhooksFromArgusOnly() []Webhook {
	return []Webhook{webhookZero, webhookOne}
}

func getWebhooksFromMigrationOnly() []Webhook {
	return []Webhook{migrationWebhookZero, migrationWebhookTwo}
}

func getWebhooksFromMixedSources() []Webhook {
	return []Webhook{webhookZero, webhookOne, migrationWebhookTwo}
}

func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
}
