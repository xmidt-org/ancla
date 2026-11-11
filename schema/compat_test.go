// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/model"
	webhook "github.com/xmidt-org/webhook-schema"
)

// v0.3.12 is the last release that used the legacy layout.  The decoder below
// is a verbatim copy of its read path (InternalWebhook / ItemToInternalWebhook
// with its plain time.Duration), so tests prove what a v0.3.12 reader does.
type v0312Webhook struct {
	Address string `json:"registered_from_address"`
	Config  struct {
		URL             string   `json:"url"`
		ContentType     string   `json:"content_type"`
		Secret          string   `json:"secret,omitempty"`
		AlternativeURLs []string `json:"alt_urls,omitempty"`
	} `json:"config"`
	FailureURL string `json:"failure_url"`
	Events     []string
	Matcher    struct {
		DeviceID []string `json:"device_id"`
	} `json:"matcher,omitempty"`
	Duration time.Duration `json:"duration"`
	Until    time.Time     `json:"until"`
}

type v0312InternalWebhook struct {
	PartnerIDs []string
	Webhook    v0312Webhook
}

func v0312ItemToInternalWebhook(i model.Item) (v0312InternalWebhook, error) {
	var iw v0312InternalWebhook
	b, err := json.Marshal(i.Data)
	if err != nil {
		return iw, err
	}
	err = json.Unmarshal(b, &iw)
	return iw, err
}

var compatNow = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// compatManifest is the registration the golden file was produced from.
// nolint:staticcheck
func compatManifest() *ManifestV1 {
	return &ManifestV1{
		PartnerIDs: []string{"company"},
		Registration: webhook.RegistrationV1{
			Address: "127.0.0.1:1234",
			Config: webhook.DeliveryConfig{
				ReceiverURL:     "http://example.com/hook",
				ContentType:     "application/json",
				Secret:          "s3cret",
				AlternativeURLs: []string{"http://alt.example.com"},
			},
			FailureURL: "http://example.com/fail",
			Events:     []string{"online", "offline"},
			Matcher:    webhook.MetadataMatcherConfig{DeviceID: []string{"mac:.*"}},
			Duration:   webhook.CustomDuration(5 * time.Minute),
			Until:      compatNow.Add(5 * time.Minute),
		},
	}
}

func fixedNow() time.Time { return compatNow }

// goldenV0312Item is an item produced by the real ancla v0.3.12 code.
func goldenV0312Item(t *testing.T) model.Item {
	t.Helper()
	b, err := os.ReadFile("testdata/v0.3.12_item.json")
	require.NoError(t, err)
	var item model.Item
	require.NoError(t, json.Unmarshal(b, &item))
	return item
}

// Latest reads what v0.3.12 wrote.
func TestReadV0312Item(t *testing.T) {
	m, err := ItemToSchema(goldenV0312Item(t))
	require.NoError(t, err)

	v1, ok := m.(*ManifestV1)
	require.True(t, ok, "got %T", m)
	assert.Equal(t, compatManifest(), v1)
	// v0.3.12 stored nanoseconds; make sure they weren't read as seconds.
	assert.Equal(t, 5*time.Minute, time.Duration(v1.Registration.Duration))
	assert.Equal(t, "http://example.com/hook", v1.GetId())
	assert.Equal(t, compatNow.Add(5*time.Minute), v1.GetUntil())
}

// Latest writes a v1 registration exactly as v0.3.12 would have.
func TestWriteV1MatchesV0312(t *testing.T) {
	golden := goldenV0312Item(t)

	got, err := SchemaToItem(fixedNow, compatManifest())
	require.NoError(t, err)

	assert.Equal(t, golden.ID, got.ID, "same ID so updates overwrite v0.3.12 entries")
	assert.Equal(t, golden.TTL, got.TTL)
	assert.Equal(t, golden.Data, got.Data)
}

// A v0.3.12 reader can read what latest writes for a v1 registration.
func TestV0312ReadsV1Write(t *testing.T) {
	item, err := SchemaToItem(fixedNow, compatManifest())
	require.NoError(t, err)

	iw, err := v0312ItemToInternalWebhook(item)
	require.NoError(t, err)

	assert.Equal(t, []string{"company"}, iw.PartnerIDs)
	w := iw.Webhook
	assert.Equal(t, "127.0.0.1:1234", w.Address)
	assert.Equal(t, "http://example.com/hook", w.Config.URL)
	assert.Equal(t, "application/json", w.Config.ContentType)
	assert.Equal(t, "s3cret", w.Config.Secret)
	assert.Equal(t, []string{"http://alt.example.com"}, w.Config.AlternativeURLs)
	assert.Equal(t, "http://example.com/fail", w.FailureURL)
	assert.Equal(t, []string{"online", "offline"}, w.Events)
	assert.Equal(t, []string{"mac:.*"}, w.Matcher.DeviceID)
	assert.Equal(t, 5*time.Minute, w.Duration)
	assert.True(t, compatNow.Add(5*time.Minute).Equal(w.Until))
}

func TestRoundTrip(t *testing.T) {
	tcs := []struct {
		name     string
		manifest Manifest
		wantKey  string
	}{
		{"v1", compatManifest(), keyLegacy},
		{"v2", &ManifestV2{
			PartnerIds: []string{"company"},
			Registration: webhook.RegistrationV2{
				CanonicalName: "canon",
				Expires:       compatNow.Add(time.Hour),
				Webhooks:      []webhook.Webhook{{ReceiverURLs: []string{"http://a.example.com"}}},
			},
		}, keyV2},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			item, err := SchemaToItem(fixedNow, tc.manifest)
			require.NoError(t, err)
			assert.Contains(t, item.Data, tc.wantKey)

			// Survive a trip through JSON, as it would through the store.
			b, err := json.Marshal(item)
			require.NoError(t, err)
			var stored model.Item
			require.NoError(t, json.Unmarshal(b, &stored))

			got, err := ItemToSchema(stored)
			require.NoError(t, err)
			assert.Equal(t, tc.manifest, got)
		})
	}
}

// Items written by the releases between v0.3.12 and now hold a v1
// registration under "wrp_event_stream_schema_v1".  They must stay readable.
func TestReadInterimV1Key(t *testing.T) {
	b, err := json.Marshal(compatManifest())
	require.NoError(t, err)
	var data map[string]any
	require.NoError(t, json.Unmarshal(b, &data))
	require.Contains(t, data, keyV1)

	got, err := ItemToSchema(model.Item{Data: data})
	require.NoError(t, err)
	assert.Equal(t, compatManifest(), got)
}

func TestItemToSchemaUnrecognized(t *testing.T) {
	m, err := ItemToSchema(model.Item{Data: map[string]any{"something": "else"}})
	assert.Error(t, err)
	assert.Nil(t, m)
}

func TestItemToSchemaBadLegacy(t *testing.T) {
	m, err := ItemToSchema(model.Item{Data: map[string]any{"Webhook": "nope"}})
	assert.Error(t, err)
	assert.Nil(t, m)
}

// Whatever label holds a v1 registration, its duration is read the same way:
// a duration string or integer nanoseconds.  ancla never stores integer
// seconds, so none of these may come back scaled.
func TestV1DurationUnderEitherLabel(t *testing.T) {
	for _, label := range []string{keyV1, keyLegacy} {
		for name, duration := range map[string]any{
			"nanoseconds": float64(5 * time.Minute),
			"string":      "5m",
			"go string":   (5 * time.Minute).String(),
		} {
			t.Run(label+"/"+name, func(t *testing.T) {
				m, err := ItemToSchema(model.Item{Data: map[string]any{
					"PartnerIDs": []any{"company"},
					label: map[string]any{
						"config":   map[string]any{"url": "http://example.com/hook"},
						"duration": duration,
					},
				}})
				require.NoError(t, err)

				v1 := m.(*ManifestV1)
				assert.Equal(t, 5*time.Minute, time.Duration(v1.Registration.Duration))
				assert.Equal(t, []string{"company"}, v1.PartnerIDs)
				assert.Equal(t, "http://example.com/hook", v1.GetId())
			})
		}
	}
}

func TestV1LabelWithLegacyPayloadIsWholeRegistration(t *testing.T) {
	golden := goldenV0312Item(t)
	data := map[string]any{
		"PartnerIDs": golden.Data["PartnerIDs"],
		keyV1:        golden.Data["Webhook"],
	}

	m, err := ItemToSchema(model.Item{Data: data})
	require.NoError(t, err)
	assert.Equal(t, compatManifest(), m)
}

func TestStoredDurationDecode(t *testing.T) {
	bad := []string{`"nope"`, `1.5`, `true`, `{}`, `"\x"`}
	for _, in := range bad {
		var d storedDuration
		assert.Error(t, d.UnmarshalJSON([]byte(in)), in)
	}

	d := storedDuration(time.Second)
	require.NoError(t, d.UnmarshalJSON([]byte(`null`)))
	assert.Equal(t, storedDuration(time.Second), d, "null leaves the value alone")

	out, err := storedDuration(5 * time.Minute).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `300000000000`, string(out))
}

func TestItemToSchemaBadV1Duration(t *testing.T) {
	m, err := ItemToSchema(model.Item{Data: map[string]any{
		keyV1: map[string]any{"duration": "nope"},
	}})
	assert.Error(t, err)
	assert.Nil(t, m)
}

func TestWriteOutputV1(t *testing.T) {
	item, err := SchemaToItemFormat(fixedNow, compatManifest(), OutputV1)
	require.NoError(t, err)

	assert.Contains(t, item.Data, keyV1)
	assert.NotContains(t, item.Data, keyLegacy)
	assert.Equal(t, goldenV0312Item(t).ID, item.ID, "same ID in either format")

	// The releases after v0.3.12 read this label with ManifestV1 and
	// webhook.CustomDuration; they must get the registration back intact.
	b, err := json.Marshal(item.Data)
	require.NoError(t, err)
	var interim ManifestV1
	require.NoError(t, json.Unmarshal(b, &interim))
	assert.Equal(t, compatManifest(), &interim)

	// And so does this release.
	got, err := ItemToSchema(item)
	require.NoError(t, err)
	assert.Equal(t, compatManifest(), got)
}

func TestWriteDefaultIsLegacy(t *testing.T) {
	def, err := SchemaToItem(fixedNow, compatManifest())
	require.NoError(t, err)
	legacy, err := SchemaToItemFormat(fixedNow, compatManifest(), OutputLegacy)
	require.NoError(t, err)
	assert.Equal(t, legacy, def)
}

func TestOutputFormatDoesNotAffectV2(t *testing.T) {
	v2 := &ManifestV2{Registration: webhook.RegistrationV2{CanonicalName: "canon", Expires: compatNow.Add(time.Hour)}}

	var items []model.Item
	for _, f := range []OutputFormat{OutputLegacy, OutputV1} {
		item, err := SchemaToItemFormat(fixedNow, v2, f)
		require.NoError(t, err)
		assert.Contains(t, item.Data, keyV2)
		items = append(items, item)
	}
	assert.Equal(t, items[0], items[1])
}

func TestUnknownOutputFormat(t *testing.T) {
	_, err := SchemaToItemFormat(fixedNow, compatManifest(), OutputFormat(99))
	assert.Error(t, err)
}

func TestOutputFormatText(t *testing.T) {
	for in, want := range map[string]OutputFormat{"": OutputLegacy, "legacy": OutputLegacy, "v1": OutputV1} {
		var f OutputFormat
		require.NoError(t, f.UnmarshalText([]byte(in)), in)
		assert.Equal(t, want, f, in)
	}

	var f OutputFormat
	assert.Error(t, f.UnmarshalText([]byte("v9")))

	assert.Equal(t, "legacy", OutputLegacy.String())
	assert.Equal(t, "v1", OutputV1.String())
	assert.Equal(t, "OutputFormat(7)", OutputFormat(7).String())
}
