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
	"github.com/xmidt-org/ancla/schema"
	webhook "github.com/xmidt-org/webhook-schema"
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
			ExpectedErr: errFailedWRPEventStreamPush,
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

	inputWRPEventStream := getTestSchemas()[0]

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
			err := svc.Add(context.TODO(), tc.Owner, inputWRPEventStream)
			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
			}
			// nolint:typecheck
			m.AssertExpectations(t)
		})
	}
}

func TestAllSchemas(t *testing.T) {
	type testCase struct {
		Description     string
		GetItemsResp    chrysom.Items
		GetItemsErr     error
		ExpectedSchemas []schema.RegistryManifest
		ExpectedErr     error
	}

	tcs := []testCase{
		{
			Description: "Fetching argus wrpEventStreams fails",
			GetItemsErr: errors.New("db failed"),
			ExpectedErr: errFailedWRPEventStreamsFetch,
		},
		{
			Description:     "WRPEventStreams fetch success",
			GetItemsResp:    getTestItems(),
			ExpectedSchemas: getTestSchemas(),
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
				assert.EqualValues(tc.ExpectedSchemas, iws)
			}

			// nolint:typecheck
			m.AssertExpectations(t)
		})
	}
}

func getTestSchemas() []schema.RegistryManifest {
	var reg []schema.RegistryManifest
	refTime := getRefTime()
	reg = append(reg, &schema.RegistryV1{
		// nolint:staticcheck
		Registration: webhook.RegistrationV1{
			Address: "example.com",
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: "example.com",
				ContentType: "application/json",
				Secret:      "superSecretXYZ",
			},
			Events: []string{"online"},
			Matcher: webhook.MetadataMatcherConfig{
				DeviceID: []string{"mac:aabbccddee.*"},
			},
			FailureURL: "example.com",
			Duration:   webhook.CustomDuration(10 * time.Second),
			Until:      refTime.Add(10 * time.Second),
		},
		PartnerIDs: []string{"comcast"},
	}, &schema.RegistryV1{
		// nolint:staticcheck
		Registration: webhook.RegistrationV1{
			Address: "example.com",
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: "example.com",
				ContentType: "application/json",
				Secret:      "doNotShare:e=mc^2",
			},
			Events: []string{"online"},
			Matcher: webhook.MetadataMatcherConfig{
				DeviceID: []string{"mac:aabbccddee.*"},
			},
			FailureURL: "example.com",
			Duration:   webhook.CustomDuration(20 * time.Second),
			Until:      refTime.Add(20 * time.Second),
		},
		PartnerIDs: []string{},
	})

	return reg
}

func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
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
				"registration_v1": map[string]interface{}{
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
					"duration":    "10s",
					"until":       "2021-01-02T15:04:10Z",
				},
				"PartnerIDs": []interface{}{"comcast"},
			},

			TTL: &firstItemExpiresInSecs,
		},
		model.Item{
			ID: "c97b4d17f7eb406720a778f73eecf419438659091039a312bebba4570e80a778",
			Data: map[string]interface{}{
				"registration_v1": map[string]interface{}{
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
					"duration":    "20s",
					"until":       "2021-01-02T15:04:20Z",
				},
				"partnerids": []string{},
			},
			TTL: &secondItemExpiresInSecs,
		},
	}
}
