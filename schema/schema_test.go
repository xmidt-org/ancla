// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/webhook-schema"
)

const (
	testOffline          = "offline"
	testOnline           = "online"
	testMac              = "mac:aabbccddee.*"
	testURL              = "example.com"
	testPartnerID        = "comcast"
	testDurationField    = "duration"
	testFailureURLField  = "failure_url"
	testMatcherField     = "matcher"
	testDeviceIDField    = "device_id"
	testContentType      = "application/json"
	testUntilField       = "until"
	testEventsField      = "events"
	testURLField         = "url"
	testContentTypeField = "content_type"
	testSecretField      = "secret"
	testConfigField      = "config"
	TestRegField         = "registered_from_address"
	TestWRPEventField    = "wrp_event_stream_schema_v1"
	testNotAnObject      = "not an object"
	testHTTPSScheme      = "https"
	testHTTPScheme       = "http"

	// nolint:gosec
	NOT_A_SECRET = "superSecretXYZ"
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
				Data: map[string]any{
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
		Data: map[string]any{
			TestWRPEventField: map[string]any{
				TestRegField: testURL,
				testConfigField: map[string]any{
					testURLField:         testURL,
					testContentTypeField: testContentType,
					testSecretField:      NOT_A_SECRET,
				},
				testEventsField: []any{testOnline},
				testMatcherField: map[string]any{
					testDeviceIDField: []any{testMac},
				},
				testFailureURLField: testURL,
				testDurationField:   "1ns",
				testUntilField:      "1970-01-01T00:00:01Z",
			},
			"PartnerIDs": []any{},
		},
		TTL: &expiresInSecs,
	}
}

func getExpiredSchema() Manifest {
	return &ManifestV1{
		// nolint:staticcheck
		Registration: webhook.RegistrationV1{
			Address: testURL,
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: testURL,
				ContentType: testContentType,
				Secret:      NOT_A_SECRET,
			},
			Events: []string{testOnline},
			Matcher: struct {
				DeviceID []string `json:"device_id"`
			}{
				DeviceID: []string{testMac},
			},
			FailureURL: testURL,
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
			Address: testURL,
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: testURL,
				ContentType: testContentType,
				Secret:      NOT_A_SECRET,
			},
			Events: []string{testOnline},
			Matcher: webhook.MetadataMatcherConfig{
				DeviceID: []string{testMac},
			},
			FailureURL: testURL,
			Duration:   webhook.CustomDuration(10 * time.Second),
			Until:      refTime.Add(10 * time.Second),
		},
		PartnerIDs: []string{testPartnerID},
	}, &ManifestV1{
		// nolint:staticcheck
		Registration: webhook.RegistrationV1{
			Address: testURL,
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: testURL,
				ContentType: testContentType,
				Secret:      NOT_A_SECRET,
			},
			Events: []string{testOnline},
			Matcher: webhook.MetadataMatcherConfig{
				DeviceID: []string{testMac},
			},
			FailureURL: testURL,
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
			Data: map[string]any{
				TestWRPEventField: map[string]any{
					TestRegField: testURL,
					testConfigField: map[string]any{
						testURLField:         testURL,
						testContentTypeField: testContentType,
						testSecretField:      NOT_A_SECRET,
					},
					testEventsField: []any{testOnline},
					testMatcherField: map[string]any{
						testDeviceIDField: []any{testMac},
					},
					testFailureURLField: testURL,
					testDurationField:   "10s",
					testUntilField:      "2021-01-02T15:04:10Z",
				},
				"PartnerIDs": []any{testPartnerID},
			},

			TTL: &firstItemExpiresInSecs,
		},
		model.Item{
			ID: "c97b4d17f7eb406720a778f73eecf419438659091039a312bebba4570e80a778",
			Data: map[string]any{
				TestWRPEventField: map[string]any{
					TestRegField: testURL,
					testConfigField: map[string]any{
						testURLField:         testURL,
						testContentTypeField: testContentType,
						testSecretField:      NOT_A_SECRET,
					},
					testEventsField: []any{testOnline},
					testMatcherField: map[string]any{
						testDeviceIDField: []any{testMac},
					},
					testDurationField:   testURL,
					testFailureURLField: "20s",
					testUntilField:      "2021-01-02T15:04:20Z",
				},
				"partnerids": []string{},
			},
			TTL: &secondItemExpiresInSecs,
		},
	}
}

// badManifest cannot be marshaled as JSON.
type badManifest struct {
	Ch chan int
}

func (badManifest) GetId() string       { return "bad" }
func (badManifest) GetUntil() time.Time { return time.Time{} }

// arrayManifest marshals to a JSON array instead of an object.
type arrayManifest struct{}

func (arrayManifest) GetId() string                { return "array" }
func (arrayManifest) GetUntil() time.Time          { return time.Time{} }
func (arrayManifest) MarshalJSON() ([]byte, error) { return []byte(`[1, 2]`), nil }

func getTestV2Schema() *ManifestV2 {
	return &ManifestV2{
		PartnerIds: []string{testPartnerID},
		Registration: webhook.RegistrationV2{
			CanonicalName: "test-v2",
			Address:       testURL,
			Webhooks: []webhook.Webhook{{
				ReceiverURLs: []string{"https://" + testURL},
				Secret:       NOT_A_SECRET,
			}},
			Expires: getRefTime().Add(10 * time.Second),
		},
	}
}

func TestManifestAccessors(t *testing.T) {
	assert := assert.New(t)
	refTime := getRefTime()

	v1 := getTestSchemas()[0]
	assert.Equal(testURL, v1.GetId())
	assert.Equal(refTime.Add(10*time.Second), v1.GetUntil())

	v2 := getTestV2Schema()
	assert.Equal("test-v2", v2.GetId())
	assert.Equal(refTime.Add(10*time.Second), v2.GetUntil())
}

func TestSchemaToItemFailures(t *testing.T) {
	tcs := []struct {
		desc     string
		manifest Manifest
	}{
		{desc: "marshal failure", manifest: badManifest{}},
		{desc: "unmarshal failure", manifest: arrayManifest{}},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			item, err := SchemaToItem(time.Now, tc.manifest)
			assert.Error(err)
			assert.Equal(model.Item{}, item)
		})
	}
}

func TestSchemaToItemV2(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	v2 := getTestV2Schema()

	item, err := SchemaToItem(getRefTime, v2)
	require.NoError(err)
	require.NotNil(item.TTL)
	assert.Equal(int64(10), *item.TTL)
	assert.Equal(fmt.Sprintf("%x", sha256.Sum256([]byte("test-v2"))), item.ID)

	back, err := ItemToSchema(item)
	require.NoError(err)
	assert.Equal(v2, back)
}

func TestItemToSchemaFailures(t *testing.T) {
	tcs := []struct {
		desc string
		item model.Item
	}{
		{
			desc: "nil data",
			item: model.Item{},
		},
		{
			desc: "v2 unmarshal failure then v1 unmarshal failure",
			item: model.Item{Data: map[string]any{
				"wrp_event_stream_schema_v2": testNotAnObject,
				TestWRPEventField:            testNotAnObject,
			}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			m, err := ItemToSchema(tc.item)
			assert.Error(err)
			assert.Nil(m)
		})
	}
}

func TestItemToSchemaV2FailureFallsBackToV1(t *testing.T) {
	assert := assert.New(t)
	items := getTestItems()
	item := items[0]
	item.Data["wrp_event_stream_schema_v2"] = testNotAnObject

	m, err := ItemToSchema(item)
	assert.NoError(err)
	assert.Equal(getTestSchemas()[0], m)
}

func TestItemsToSchemas(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ms, err := ItemsToSchemas(getTestItems()[:1])
	require.NoError(err)
	assert.Equal(getTestSchemas()[:1], ms)

	ms, err = ItemsToSchemas(nil)
	require.NoError(err)
	assert.Equal([]Manifest{}, ms)

	ms, err = ItemsToSchemas(chrysom.Items{model.Item{}})
	assert.Error(err)
	assert.Nil(ms)
}

func TestSchemasToWRPEventStreams(t *testing.T) {
	assert := assert.New(t)
	v1 := getTestSchemas()[0].(*ManifestV1)
	v2 := getTestV2Schema()

	out := SchemasToWRPEventStreams([]Manifest{v1, v2, badManifest{}})
	assert.Equal([]any{v1.Registration, v2.Registration}, out)

	assert.Equal([]any{}, SchemasToWRPEventStreams(nil))
}
