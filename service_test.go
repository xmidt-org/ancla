// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package ancla

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/model"
)

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
				argus: m,
				now:   time.Now,
			}
			// nolint:typecheck
			m.On("PushItem", context.TODO(), tc.Owner, mock.Anything).Return(tc.PushItemResults.result, tc.PushItemResults.err)
			err := svc.Add(context.TODO(), tc.Owner, inputWebhook)
			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
			// nolint:typecheck
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
				argus: m,
			}
			// nolint:typecheck
			m.On("GetItems", context.TODO(), "").Return(tc.GetItemsResp, tc.GetItemsErr)
			iws, err := svc.GetAll(context.TODO())

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
				assert.Empty(iws)
			} else {
				assert.EqualValues(tc.ExpectedInternalWebhooks, iws)
			}

			// nolint:typecheck
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
		model.Item{
			ID: "a379a6f6eeafb9a55e378c118034e2751e682fab9f2d30ab13d2125586ce1947",
			Data: map[string]interface{}{
				"Webhook": map[string]interface{}{
					"registered_from_address": "example.com",
					"config": map[string]interface{}{
						"url":          "example.com",
						"content_type": "application/json",
						"secret":       "superSecretXYZ",
					},
					"events": []interface{}{"online"},
					"matcher": map[string]interface{}{
						"device_id": []interface{}{"mac:aabbccddee.*"},
					},
					"failure_url": "example.com",
					"duration":    float64((10 * time.Second).Nanoseconds()),
					"until":       "2021-01-02T15:04:10Z",
				},
				"PartnerIDs": []interface{}{"comcast"},
			},

			TTL: &firstItemExpiresInSecs,
		},
		model.Item{
			ID: "c97b4d17f7eb406720a778f73eecf419438659091039a312bebba4570e80a778",
			Data: map[string]interface{}{
				"webhook": map[string]interface{}{
					"registered_from_address": "example.com",
					"config": map[string]interface{}{
						"url":          "example.com",
						"content_type": "application/json",
						"secret":       "doNotShare:e=mc^2",
					},
					"events": []interface{}{"online"},
					"matcher": map[string]interface{}{
						"device_id": []interface{}{"mac:aabbccddee.*"},
					},
					"failure_url": "example.com",
					"duration":    float64((20 * time.Second).Nanoseconds()),
					"until":       "2021-01-02T15:04:20Z",
				},
				"partnerids": []string{},
			},
			TTL: &secondItemExpiresInSecs,
		},
	}
}
