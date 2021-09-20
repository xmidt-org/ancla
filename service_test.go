package ancla

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func TestValidateConfig(t *testing.T) {
	type testCase struct {
		Description    string
		InputConfig    *Config
		ExpectedConfig *Config
	}

	logger := log.NewJSONLogger(ioutil.Discard)
	metricsProvider, err := xmetrics.NewRegistry(nil, Metrics)
	require.Nil(t, err)
	tcs := []testCase{
		{
			Description: "DefaultedValues",
			InputConfig: &Config{},
			ExpectedConfig: &Config{
				Logger: log.NewNopLogger(),
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

	inputWebhook := getTestInternalWebhooks()[0]

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m := new(mockPushReader)
			svc := service{
				logger: log.NewNopLogger(),
				config: Config{},
				argus:  m,
				now:    time.Now,
			}
			m.On("PushItem", context.TODO(), tc.Owner, mock.Anything).Return(tc.PushItemResults.result, tc.PushItemResults.err)
			err := svc.Add(context.TODO(), tc.Owner, inputWebhook)
			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
			m.AssertExpectations(t)
		})
	}
}

func TestAllInternalWebhooks(t *testing.T) {
	type testCase struct {
		Description              string
		GetItemsResp             chrysom.Items
		GetItemsErr              error
		ExpectedInternalWebhooks []InternalWebhook
		ExpectedErr              error
	}

	tcs := []testCase{
		{
			Description: "Fetching argus webhooks fails",
			GetItemsErr: errors.New("db failed"),
			ExpectedErr: errFailedWebhooksFetch,
		},
		{
			Description:              "Webhooks fetch success",
			GetItemsResp:             getTestItems(),
			ExpectedInternalWebhooks: getTestInternalWebhooks(),
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
			m.On("GetItems", context.TODO(), "").Return(tc.GetItemsResp, tc.GetItemsErr)
			iws, err := svc.AllInternalWebhooks(context.TODO())

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
				assert.Empty(iws)
			} else {
				assert.EqualValues(tc.ExpectedInternalWebhooks, iws)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestItemToInternalWebhook(t *testing.T) {
	items := getTestItems()
	iws := getTestInternalWebhooks()
	tcs := []struct {
		Description             string
		InputItem               model.Item
		ExpectedInternalWebhook InternalWebhook
		ShouldErr               bool
	}{
		{
			Description: "Err Marshaling",
			InputItem: model.Item{
				Data: map[string]interface{}{
					"cannotUnmarshal": make(chan int),
				},
			},
			ShouldErr: true,
		},
		{
			Description:             "Success",
			InputItem:               items[0],
			ExpectedInternalWebhook: iws[0],
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			w, err := itemToInternalWebhook(tc.InputItem)
			if tc.ShouldErr {
				assert.Error(err)
			}
			assert.Equal(tc.ExpectedInternalWebhook, w)
		})
	}
}
func TestInternalWebhookToItem(t *testing.T) {
	refTime := getRefTime()
	fixedNow := func() time.Time {
		return refTime
	}
	items := getTestItems()
	iws := getTestInternalWebhooks()
	tcs := []struct {
		Description          string
		InputInternalWebhook InternalWebhook
		ExpectedItem         model.Item
		ShouldErr            bool
	}{
		{
			Description:          "Expired item",
			InputInternalWebhook: getExpiredInternalWebhook(),
			ExpectedItem:         getExpiredItem(),
		},
		{
			Description:          "Happy path",
			InputInternalWebhook: iws[0],
			ExpectedItem:         items[0],
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			item, err := internalWebhookToItem(fixedNow, tc.InputInternalWebhook)
			if tc.ShouldErr {
				assert.Error(err)
			}
			assert.Equal(tc.ExpectedItem, item)
		})
	}
}

func getExpiredItem() model.Item {
	var expiresInSecs int64 = 0
	return model.Item{
		ID: "b3bbc3467366959e0aba3c33588a08c599f68a740fabf4aa348463d3dc7dcfe8",
		Data: map[string]interface{}{
			"Webhook": map[string]interface{}{
				"registered_from_address": "http://original-requester.example.net",
				"config": map[string]interface{}{
					"url":          "http://deliver-here-0.example.net",
					"content_type": "application/json",
					"secret":       "superSecretXYZ",
				},
				"events": []interface{}{"online"},
				"matcher": map[string]interface{}{
					"device_id": []interface{}{"mac:aabbccddee.*"},
				},
				"failure_url": "http://contact-here-when-fails.example.net",
				"duration":    float64(time.Second.Nanoseconds()),
				"until":       "1970-01-01T00:00:01Z",
			},
			"PartnerIDs": []interface{}{},
		},
		TTL: &expiresInSecs,
	}
}

func getExpiredInternalWebhook() InternalWebhook {
	return InternalWebhook{
		Webhook: Webhook{
			Address: "http://original-requester.example.net",
			Config: DeliveryConfig{
				URL:         "http://deliver-here-0.example.net",
				ContentType: "application/json",
				Secret:      "superSecretXYZ",
			},
			Events: []string{"online"},
			Matcher: struct {
				DeviceID []string `json:"device_id"`
			}{
				DeviceID: []string{"mac:aabbccddee.*"},
			},
			FailureURL: "http://contact-here-when-fails.example.net",
			Duration:   time.Second,
			Until:      time.Unix(1, 0).UTC(),
		},
		PartnerIDs: []string{},
	}
}

func getTestItems() chrysom.Items {
	var (
		firstItemExpiresInSecs  int64 = 10
		secondItemExpiresInSecs int64 = 20
	)
	return chrysom.Items{
		{
			ID: "b3bbc3467366959e0aba3c33588a08c599f68a740fabf4aa348463d3dc7dcfe8",
			Data: map[string]interface{}{
				"Webhook": map[string]interface{}{
					"registered_from_address": "http://original-requester.example.net",
					"config": map[string]interface{}{
						"url":          "http://deliver-here-0.example.net",
						"content_type": "application/json",
						"secret":       "superSecretXYZ",
					},
					"events": []interface{}{"online"},
					"matcher": map[string]interface{}{
						"device_id": []interface{}{"mac:aabbccddee.*"},
					},
					"failure_url": "http://contact-here-when-fails.example.net",
					"duration":    float64((10 * time.Second).Nanoseconds()),
					"until":       "2021-01-02T15:04:10Z",
				},
				"PartnerIDs": []interface{}{"comcast"},
			},

			TTL: &firstItemExpiresInSecs,
		},
		{
			ID: "c97b4d17f7eb406720a778f73eecf419438659091039a312bebba4570e80a778",
			Data: map[string]interface{}{
				"webhook": map[string]interface{}{
					"registered_from_address": "http://original-requester.example.net",
					"config": map[string]interface{}{
						"url":          "http://deliver-here-1.example.net",
						"content_type": "application/json",
						"secret":       "doNotShare:e=mc^2",
					},
					"events": []interface{}{"online"},
					"matcher": map[string]interface{}{
						"device_id": []interface{}{"mac:aabbccddee.*"},
					},
					"failure_url": "http://contact-here-when-fails.example.net",
					"duration":    float64((20 * time.Second).Nanoseconds()),
					"until":       "2021-01-02T15:04:20Z",
				},
				"partnerids": []string{},
			},
			TTL: &secondItemExpiresInSecs,
		},
	}
}

func getTestInternalWebhooks() []InternalWebhook {
	refTime := getRefTime()
	return []InternalWebhook{
		{
			Webhook: Webhook{
				Address: "http://original-requester.example.net",
				Config: DeliveryConfig{
					URL:         "http://deliver-here-0.example.net",
					ContentType: "application/json",
					Secret:      "superSecretXYZ",
				},
				Events: []string{"online"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},
				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   10 * time.Second,
				Until:      refTime.Add(10 * time.Second),
			},
			PartnerIDs: []string{"comcast"},
		},
		{
			Webhook: Webhook{
				Address: "http://original-requester.example.net",
				Config: DeliveryConfig{
					ContentType: "application/json",
					URL:         "http://deliver-here-1.example.net",
					Secret:      "doNotShare:e=mc^2",
				},
				Events: []string{"online"},
				Matcher: MetadataMatcherConfig{
					DeviceID: []string{"mac:aabbccddee.*"},
				},

				FailureURL: "http://contact-here-when-fails.example.net",
				Duration:   20 * time.Second,
				Until:      refTime.Add(20 * time.Second),
			},
			PartnerIDs: []string{},
		},
	}
}

func getTestWebhooks() []Webhook {
	refTime := getRefTime()
	return []Webhook{
		{
			Address: "http://original-requester.example.net",
			Config: DeliveryConfig{
				URL:         "http://deliver-here-0.example.net",
				ContentType: "application/json",
				Secret:      "superSecretXYZ",
			},
			Events: []string{"online"},
			Matcher: MetadataMatcherConfig{
				DeviceID: []string{"mac:aabbccddee.*"},
			},
			FailureURL: "http://contact-here-when-fails.example.net",
			Duration:   10 * time.Second,
			Until:      refTime.Add(10 * time.Second),
		},
		{
			Address: "http://original-requester.example.net",
			Config: DeliveryConfig{
				ContentType: "application/json",
				URL:         "http://deliver-here-1.example.net",
				Secret:      "doNotShare:e=mc^2",
			},
			Events: []string{"online"},
			Matcher: MetadataMatcherConfig{
				DeviceID: []string{"mac:aabbccddee.*"},
			},

			FailureURL: "http://contact-here-when-fails.example.net",
			Duration:   20 * time.Second,
			Until:      refTime.Add(20 * time.Second),
		},
	}
}

func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
}
