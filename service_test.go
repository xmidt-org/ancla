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
)

func TestValidateConfig(t *testing.T) {
	type testCase struct {
		Description    string
		InputConfig    *Config
		ExpectedConfig *Config
	}

	logger := log.NewJSONLogger(ioutil.Discard)
	metricsProvider := provider.NewExpvarProvider()
	tcs := []testCase{
		{
			Description: "DefaultedValues",
			InputConfig: &Config{},
			ExpectedConfig: &Config{
				Logger:          log.NewNopLogger(),
				MetricsProvider: provider.NewDiscardProvider(),
			},
		},
		{
			Description: "Given values",
			InputConfig: &Config{
				Logger:          logger,
				MetricsProvider: metricsProvider,
			},
			ExpectedConfig: &Config{
				Logger:          logger,
				MetricsProvider: metricsProvider,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			validateConfig(tc.InputConfig)
			assert.EqualValues(tc.ExpectedConfig, tc.InputConfig)
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
			m.On("PushItem", tc.Owner, mock.Anything).Return(tc.PushItemResults.result, tc.PushItemResults.err)
			err := svc.Add(tc.Owner, tc.InputWebhook)
			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
			m.AssertExpectations(t)
		})
	}
}

func TestAllWebhooks(t *testing.T) {
	type testCase struct {
		Description      string
		GetItemsResp     chrysom.Items
		GetItemsErr      error
		ExpectedWebhooks []Webhook
		ExpectedErr      error
	}

	tcs := []testCase{
		{
			Description: "Fetching argus webhooks fails",
			GetItemsErr: errors.New("db failed"),
			ExpectedErr: errFailedWebhooksFetch,
		},
		{
			Description:      "Webhooks fetch success",
			GetItemsResp:     getTestItems(),
			ExpectedWebhooks: getWebhooks(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockPushReader)

			svc := service{
				argus:  m,
				logger: log.NewNopLogger(),
				config: Config{},
			}
			m.On("GetItems", "").Return(tc.GetItemsResp, tc.GetItemsErr)
			webhooks, err := svc.AllWebhooks()

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
				assert.Empty(webhooks)
			} else {
				assert.EqualValues(tc.ExpectedWebhooks, webhooks)
			}

			m.AssertExpectations(t)
		})
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

func getWebhooks() []Webhook {
	var (
		refTime     = getRefTime()
		webhookZero = Webhook{
			Address: "http://webhook-requester-to-argus.example",
			Config: DeliveryConfig{
				URL: "http://events-webhook0-here.example",
			},
			Until: refTime.Add(5 * time.Second),
		}
		webhookOne = Webhook{
			Address: "http://webhook-requester-to-argus.example",
			Config: DeliveryConfig{
				URL: "http://events-webhook1-here.example",
			},
			Until: refTime.Add(10 * time.Second),
		}
	)
	return []Webhook{webhookZero, webhookOne}
}

func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
}
