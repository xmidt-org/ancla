/**
 * Copyright 2022 Comcast Cable Communications Management, LLC
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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/sallust"
)

func TestNewService(t *testing.T) {
	tcs := []struct {
		desc        string
		config      Config
		getLogger   sallust.GetLoggerFunc
		expectedErr bool
	}{
		{
			desc: "Success Case",
			config: Config{
				BasicClientConfig: chrysom.BasicClientConfig{
					Address: "test",
					Bucket:  "test",
				},
			},
		},
		{
			desc:        "Chrysom Basic Client Creation Failure",
			expectedErr: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			_, err := NewService(tc.config, tc.getLogger)
			if tc.expectedErr {
				assert.NotNil(err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStartListener(t *testing.T) {
	mockServiceConfig := Config{
		BasicClientConfig: chrysom.BasicClientConfig{
			Address: "test",
			Bucket:  "test",
		},
	}
	mockService, _ := NewService(mockServiceConfig, nil)
	tcs := []struct {
		desc           string
		serviceConfig  Config
		listenerConfig ListenerConfig
		svc            service
		expectedErr    bool
	}{
		{
			desc: "Success Case",
			svc:  *mockService,
			listenerConfig: ListenerConfig{
				Config: chrysom.ListenerClientConfig{},
			},
		},
		{
			desc:        "Chrysom Listener Client Creation Failure",
			expectedErr: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			_, err := tc.svc.StartListener(tc.listenerConfig, nil)
			if tc.expectedErr {
				assert.NotNil(err)
				return
			}
			require.NoError(t, err)
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
				result: chrysom.UnknownPushResult,
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
				logger: sallust.Default(),
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
				logger: sallust.Default(),
				config: Config{},
			}
			m.On("GetItems", context.TODO(), "").Return(tc.GetItemsResp, tc.GetItemsErr)
			iws, err := svc.GetAll(context.TODO())

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
