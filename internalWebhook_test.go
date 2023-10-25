// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/model"
)

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

func getRefTime() time.Time {
	refTime, err := time.Parse(time.RFC3339, "2021-01-02T15:04:00Z")
	if err != nil {
		panic(err)
	}
	return refTime
}
