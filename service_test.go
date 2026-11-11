// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package ancla

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	inputManifest := getTestSchemas()[0]
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
			err := svc.Add(context.TODO(), tc.Owner, inputManifest)
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
		ExpectedSchemas []schema.Manifest
		ExpectedErr     error
	}

	tcs := []testCase{
		{
			Description: "Fetching argus wrpEventStreams fails",
			GetItemsErr: errors.New("db failed"),
			ExpectedErr: errFailedWRPEventStreamsFetch,
		},
		{
			Description:     "wrpEventStreams fetch success",
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
			manifests, err := svc.GetAll(context.TODO())

			if tc.ExpectedErr != nil {
				assert.True(errors.Is(err, tc.ExpectedErr))
				assert.Empty(manifests)
			} else {
				assert.EqualValues(tc.ExpectedSchemas, manifests)
			}

			// nolint:typecheck
			m.AssertExpectations(t)
		})
	}
}

func getTestSchemas() []schema.Manifest {
	var reg []schema.Manifest
	refTime := getRefTime()
	reg = append(reg, &schema.ManifestV1{
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
	}, &schema.ManifestV1{
		// nolint:staticcheck
		Registration: webhook.RegistrationV1{
			Address: testURL,
			// nolint:staticcheck
			Config: webhook.DeliveryConfig{
				ReceiverURL: testURL,
				ContentType: testContentType,
				Secret:      NOT_A_SECRET, // nolint:gosec
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
					testFailureURLField: testURL,
					testDurationField:   "20s",
					testUntilField:      "2021-01-02T15:04:20Z",
				},
				"partnerids": []string{},
			},
			TTL: &secondItemExpiresInSecs,
		},
	}
}

func TestAddConversionFailure(t *testing.T) {
	assert := assert.New(t)
	m := new(mockPushReader)
	svc := service{argus: m, now: time.Now}
	err := svc.Add(context.TODO(), "owner", badManifest{})
	assert.ErrorIs(err, errFailedWRPEventStreamConversion)
	// nolint:typecheck
	m.AssertExpectations(t)
}

func TestGetAllConversionFailure(t *testing.T) {
	assert := assert.New(t)
	m := new(mockPushReader)
	svc := service{argus: m}
	// nolint:typecheck
	m.On("GetItems", context.TODO(), "").Return(chrysom.Items{
		model.Item{Data: map[string]any{TestWRPEventField: "not an object"}},
	}, nil)
	manifests, err := svc.GetAll(context.TODO())
	assert.ErrorIs(err, errFailedItemConversion)
	assert.Nil(manifests)
	// nolint:typecheck
	m.AssertExpectations(t)
}

func TestNewService(t *testing.T) {
	require := require.New(t)
	m := new(mockPushReader)
	svc := NewService(m)
	require.NotNil(svc)
	require.Equal(m, svc.argus)
	require.NotNil(svc.now)
}

// memStore is an in-memory store keyed by item ID, like Argus.
type memStore struct {
	items map[string]model.Item
}

func (m *memStore) PushItem(_ context.Context, _ string, item model.Item) (chrysom.PushResult, error) {
	if m.items == nil {
		m.items = map[string]model.Item{}
	}
	_, existed := m.items[item.ID]
	m.items[item.ID] = item
	if existed {
		return chrysom.UpdatedPushResult, nil
	}
	return chrysom.CreatedPushResult, nil
}

func (m *memStore) RemoveItem(_ context.Context, id, _ string) (model.Item, error) {
	return m.items[id], nil
}

func (m *memStore) GetItems(_ context.Context, _ string) (chrysom.Items, error) {
	out := chrysom.Items{}
	for _, i := range m.items {
		out = append(out, i)
	}
	return out, nil
}

func v1Manifest(url string) *schema.ManifestV1 {
	// nolint:staticcheck
	return &schema.ManifestV1{
		PartnerIDs: []string{"company"},
		Registration: webhook.RegistrationV1{
			Config:   webhook.DeliveryConfig{ReceiverURL: url, ContentType: "application/json"},
			Events:   []string{"online"},
			Matcher:  webhook.MetadataMatcherConfig{DeviceID: []string{".*"}},
			Duration: webhook.CustomDuration(time.Minute),
			Until:    time.Now().Add(time.Minute),
		},
	}
}

// A v1 registration is stored in the v0.3.12 layout, and a v2 one is not.
func TestServiceStoredLayout(t *testing.T) {
	store := &memStore{}
	svc := NewService(store)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "o", v1Manifest("http://a.example.com")))
	require.NoError(t, svc.Add(ctx, "o", &schema.ManifestV2{
		Registration: webhook.RegistrationV2{CanonicalName: "canon", Expires: time.Now().Add(time.Hour)},
	}))
	require.Len(t, store.items, 2)

	for _, item := range store.items {
		if _, v1 := item.Data["Webhook"]; v1 {
			assert.NotContains(t, item.Data, "wrp_event_stream_schema_v1")
			continue
		}
		assert.Contains(t, item.Data, "wrp_event_stream_schema_v2")
	}
}

// The store can hold items written by v0.3.12, by the interim releases, and
// by this one.  GetAll must return all of them intact.
func TestServiceGetAllMixedLayouts(t *testing.T) {
	store := &memStore{}
	svc := NewService(store)
	ctx := context.Background()

	require.NoError(t, svc.Add(ctx, "o", v1Manifest("http://new.example.com")))

	// As the interim releases wrote it.
	interim, err := json.Marshal(v1Manifest("http://interim.example.com"))
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(interim, &data))
	store.items["interim"] = model.Item{ID: "interim", Data: data}

	// As v0.3.12 wrote it.
	store.items["old"] = model.Item{ID: "old", Data: map[string]any{
		"PartnerIDs": []any{"company"},
		"Webhook": map[string]any{
			"config":   map[string]any{"url": "http://old.example.com"},
			"duration": float64(time.Minute),
		},
	}}

	got, err := svc.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, got, 3)

	ids := map[string]bool{}
	for _, m := range got {
		ids[m.GetId()] = true
		// nolint:staticcheck
		assert.Equal(t, time.Minute, time.Duration(m.(*schema.ManifestV1).Registration.Duration), m.GetId())
	}
	assert.Equal(t, map[string]bool{
		"http://new.example.com":     true,
		"http://interim.example.com": true,
		"http://old.example.com":     true,
	}, ids)
}

func TestServiceOutputFormat(t *testing.T) {
	tcs := []struct {
		name      string
		opts      []ServiceOption
		wantKey   string
		absentKey string
	}{
		{"default is legacy", nil, "Webhook", "wrp_event_stream_schema_v1"},
		{"explicit legacy", []ServiceOption{WithOutputFormat(schema.OutputLegacy)}, "Webhook", "wrp_event_stream_schema_v1"},
		{"v1", []ServiceOption{WithOutputFormat(schema.OutputV1)}, "wrp_event_stream_schema_v1", "Webhook"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			store := &memStore{}
			svc := NewService(store, tc.opts...)
			require.NoError(t, svc.Add(context.Background(), "o", v1Manifest("http://a.example.com")))

			require.Len(t, store.items, 1)
			for _, item := range store.items {
				assert.Contains(t, item.Data, tc.wantKey)
				assert.NotContains(t, item.Data, tc.absentKey)
			}

			got, err := svc.GetAll(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "http://a.example.com", got[0].GetId())
		})
	}
}

func TestServiceOutputFormatInvalid(t *testing.T) {
	store := &memStore{}
	svc := NewService(store, WithOutputFormat(schema.OutputFormat(99)))
	assert.ErrorIs(t, svc.Add(context.Background(), "o", v1Manifest("http://a.example.com")), errFailedWRPEventStreamConversion)
	assert.Empty(t, store.items)
}
