// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package ancla

// import (
// 	"context"
// 	"errors"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/require"
// 	"github.com/xmidt-org/argus/chrysom"
// 	"github.com/xmidt-org/argus/model"
// 	"github.com/xmidt-org/sallust"
// 	"github.com/xmidt-org/webhook-schema"
// 	"go.uber.org/zap"
// )

// func TestNewService(t *testing.T) {
// 	tcs := []struct {
// 		desc        string
// 		config      Config
// 		getLogger   func(context.Context) *zap.Logger
// 		expectedErr bool
// 	}{
// 		{
// 			desc: "Success Case",
// 			config: Config{
// 				BasicClientConfig: chrysom.BasicClientConfig{
// 					Address: "test",
// 					Bucket:  "test",
// 				},
// 			},
// 		},
// 		{
// 			desc:        "Chrysom Basic Client Creation Failure",
// 			expectedErr: true,
// 		},
// 	}
// 	for _, tc := range tcs {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			assert := assert.New(t)
// 			_, err := NewService(tc.config, tc.getLogger)
// 			if tc.expectedErr {
// 				assert.NotNil(err)
// 				return
// 			}
// 			require.NoError(t, err)
// 		})
// 	}
// }

// func TestStartListener(t *testing.T) {
// 	mockServiceConfig := Config{
// 		BasicClientConfig: chrysom.BasicClientConfig{
// 			Address: "test",
// 			Bucket:  "test",
// 		},
// 	}
// 	mockService, _ := NewService(mockServiceConfig, nil)
// 	tcs := []struct {
// 		desc           string
// 		serviceConfig  Config
// 		listenerConfig ListenerConfig
// 		svc            service
// 		expectedErr    bool
// 	}{
// 		{
// 			desc: "Success Case",
// 			svc:  *mockService,
// 			listenerConfig: ListenerConfig{
// 				Config: chrysom.ListenerClientConfig{},
// 			},
// 		},
// 		{
// 			desc:        "Chrysom Listener Client Creation Failure",
// 			expectedErr: true,
// 		},
// 	}
// 	for _, tc := range tcs {
// 		t.Run(tc.desc, func(t *testing.T) {
// 			assert := assert.New(t)
// 			_, err := tc.svc.StartListener(tc.listenerConfig, nil)
// 			if tc.expectedErr {
// 				assert.NotNil(err)
// 				return
// 			}
// 			require.NoError(t, err)
// 		})
// 	}
// }

// func TestAdd(t *testing.T) {
// 	type pushItemResults struct {
// 		result chrysom.PushResult
// 		err    error
// 	}
// 	type testCase struct {
// 		Description     string
// 		Owner           string
// 		PushItemResults pushItemResults
// 		ExpectedErr     error
// 	}

// 	tcs := []testCase{
// 		{
// 			Description: "PushItem fails",
// 			PushItemResults: pushItemResults{
// 				err: errors.New("push item failed"),
// 			},
// 			ExpectedErr: errFailedWebhookPush,
// 		},
// 		{
// 			Description: "Unknown push result",
// 			PushItemResults: pushItemResults{
// 				result: chrysom.UnknownPushResult,
// 			},
// 			ExpectedErr: errNonSuccessPushResult,
// 		},
// 		{
// 			Description: "Item created",
// 			PushItemResults: pushItemResults{
// 				result: chrysom.CreatedPushResult,
// 			},
// 		},
// 		{
// 			Description: "Item update",
// 			PushItemResults: pushItemResults{
// 				result: chrysom.UpdatedPushResult,
// 			},
// 		},
// 	}

// 	inputWebhook := getTestInternalWebhooks()[0]

// 	for _, tc := range tcs {
// 		t.Run(tc.Description, func(t *testing.T) {
// 			assert := assert.New(t)
// 			m := new(mockPushReader)
// 			svc := service{
// 				logger: sallust.Default(),
// 				config: Config{},
// 				argus:  m,
// 				now:    time.Now,
// 			}
// 			// nolint:typecheck
// 			m.On("PushItem", context.TODO(), tc.Owner, mock.Anything).Return(tc.PushItemResults.result, tc.PushItemResults.err)
// 			err := svc.Add(context.TODO(), tc.Owner, inputWebhook)
// 			if tc.ExpectedErr != nil {
// 				assert.True(errors.Is(err, tc.ExpectedErr))
// 			}
// 			// nolint:typecheck
// 			m.AssertExpectations(t)
// 		})
// 	}
// }

// func TestAllInternalWebhooks(t *testing.T) {
// 	type testCase struct {
// 		Description              string
// 		GetItemsResp             chrysom.Items
// 		GetItemsErr              error
// 		ExpectedInternalWebhooks []webhook.Register
// 		ExpectedErr              error
// 	}

// 	tcs := []testCase{
// 		{
// 			Description: "Fetching argus webhooks fails",
// 			GetItemsErr: errors.New("db failed"),
// 			ExpectedErr: errFailedWebhooksFetch,
// 		},
// 		{
// 			Description:              "Webhooks fetch success",
// 			GetItemsResp:             getTestItems(),
// 			ExpectedInternalWebhooks: getTestInternalWebhooks(),
// 		},
// 	}

// 	for _, tc := range tcs {
// 		t.Run(tc.Description, func(t *testing.T) {
// 			assert := assert.New(t)
// 			m := new(mockPushReader)

// 			svc := service{
// 				argus:  m,
// 				logger: sallust.Default(),
// 				config: Config{},
// 			}
// 			// nolint:typecheck
// 			m.On("GetItems", context.TODO(), "").Return(tc.GetItemsResp, tc.GetItemsErr)
// 			iws, err := svc.GetAll(context.TODO())

// 			if tc.ExpectedErr != nil {
// 				assert.True(errors.Is(err, tc.ExpectedErr))
// 				assert.Empty(iws)
// 			} else {
// 				assert.EqualValues(tc.ExpectedInternalWebhooks, iws)
// 			}

// 			// nolint:typecheck
// 			m.AssertExpectations(t)
// 		})
// 	}
// }

// func getTestItems() chrysom.Items {
// 	var (
// 		firstItemExpiresInSecs  int64 = 10
// 		secondItemExpiresInSecs int64 = 20
// 	)
// 	return chrysom.Items{
// 		model.Item{
// 			ID: "b3bbc3467366959e0aba3c33588a08c599f68a740fabf4aa348463d3dc7dcfe8",
// 			Data: map[string]interface{}{
// 				"Webhook": map[string]interface{}{
// 					"registered_from_address": "http://original-requester.example.net",
// 					"config": map[string]interface{}{
// 						"url":          "http://deliver-here-0.example.net",
// 						"content_type": "application/json",
// 						"secret":       "superSecretXYZ",
// 					},
// 					"events": []interface{}{"online"},
// 					"matcher": map[string]interface{}{
// 						"device_id": []interface{}{"mac:aabbccddee.*"},
// 					},
// 					"failure_url": "http://contact-here-when-fails.example.net",
// 					"duration":    float64((10 * time.Second).Nanoseconds()),
// 					"until":       "2021-01-02T15:04:10Z",
// 				},
// 				"PartnerIDs": []interface{}{"comcast"},
// 			},

// 			TTL: &firstItemExpiresInSecs,
// 		},
// 		model.Item{
// 			ID: "c97b4d17f7eb406720a778f73eecf419438659091039a312bebba4570e80a778",
// 			Data: map[string]interface{}{
// 				"webhook": map[string]interface{}{
// 					"registered_from_address": "http://original-requester.example.net",
// 					"config": map[string]interface{}{
// 						"url":          "http://deliver-here-1.example.net",
// 						"content_type": "application/json",
// 						"secret":       "doNotShare:e=mc^2",
// 					},
// 					"events": []interface{}{"online"},
// 					"matcher": map[string]interface{}{
// 						"device_id": []interface{}{"mac:aabbccddee.*"},
// 					},
// 					"failure_url": "http://contact-here-when-fails.example.net",
// 					"duration":    float64((20 * time.Second).Nanoseconds()),
// 					"until":       "2021-01-02T15:04:20Z",
// 				},
// 				"partnerids": []string{},
// 			},
// 			TTL: &secondItemExpiresInSecs,
// 		},
// 	}
// }
