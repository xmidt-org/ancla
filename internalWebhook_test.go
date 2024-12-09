// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/webhook-schema"
)

func TestItemToInternalWebhook(t *testing.T) {
	items := getTestItems()
	iws := getTestInternalWebhooks()
	tcs := []struct {
		Description             string
		InputItem               model.Item
		ExpectedInternalWebhook Register
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
			w, err := ItemToInternalWebhook(tc.InputItem)
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
		InputInternalWebhook Register
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
			item, err := InternalWebhookToItem(fixedNow, tc.InputInternalWebhook)
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
				"duration":    "1ns",
				"until":       "1970-01-01T00:00:01Z",
			},
			"PartnerIDs": []interface{}{},
		},
		TTL: &expiresInSecs,
	}
}

func getExpiredInternalWebhook() Register {
	return &RegistryV1{
		Registration: webhook.RegistrationV1{
			Address: "example.com",
			Config: webhook.DeliveryConfig{
				ReceiverURL: "example.com",
				ContentType: "application/json",
				Secret:      "superSecretXYZ",
			},
			Events: []string{"online"},
			Matcher: struct {
				DeviceID []string `json:"device_id"`
			}{
				DeviceID: []string{"mac:aabbccddee.*"},
			},
			FailureURL: "example.com",
			Duration:   webhook.CustomDuration(1),
			Until:      time.Unix(1, 0).UTC(),
		},
		PartnerIDs: []string{},
	}
}

func getTestInternalWebhooks() []Register {
	var reg []Register
	refTime := getRefTime()
	reg = append(reg, &RegistryV1{
		Registration: webhook.RegistrationV1{
			Address: "example.com",
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
