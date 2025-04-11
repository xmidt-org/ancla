// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/webhook-schema"
)

func TestItemToSchema(t *testing.T) {
	items := getTestItems()
	manifests := getTestSchemas()
	tcs := []struct {
		Description    string
		InputItem      model.Item
		ExpectedSchema Manifest
		ShouldErr      bool
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
			Description:    "Success",
			InputItem:      items[0],
			ExpectedSchema: manifests[0],
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			m, err := ItemToSchema(tc.InputItem)
			if tc.ShouldErr {
				assert.Error(err)
			}
			assert.Equal(tc.ExpectedSchema, m)
		})
	}
}

func TestSchemaToItem(t *testing.T) {
	refTime := getRefTime()
	fixedNow := func() time.Time {
		return refTime
	}
	items := getTestItems()
	manifests := getTestSchemas()
	tcs := []struct {
		Description  string
		InputSchema  Manifest
		ExpectedItem model.Item
		ShouldErr    bool
	}{
		{
			Description:  "Expired item",
			InputSchema:  getExpiredSchema(),
			ExpectedItem: getExpiredItem(),
		},
		{
			Description:  "Happy path",
			InputSchema:  manifests[0],
			ExpectedItem: items[0],
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			item, err := SchemaToItem(fixedNow, tc.InputSchema)
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
			"wrp_event_stream_schema_v1": map[string]interface{}{
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

func getExpiredSchema() Manifest {
	return &ManifestV1{
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

func getTestSchemas() []Manifest {
	var reg []Manifest
	refTime := getRefTime()
	reg = append(reg, &ManifestV1{
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
	}, &ManifestV1{
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
				"wrp_event_stream_schema_v1": map[string]interface{}{
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
				"wrp_event_stream_schema_v1": map[string]interface{}{
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
