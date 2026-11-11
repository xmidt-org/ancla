// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xmidt-org/ancla/schema"
	webhook "github.com/xmidt-org/webhook-schema"
)

// Example_outputForms shows the three layouts ancla can store a registration
// in: a v1 registration as OutputLegacy (the default, readable by every
// version including v0.3.12), a v1 registration as OutputV1, and a v2
// registration (which has one layout, whatever the format).
func Example_outputForms() {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	// nolint:staticcheck
	v1 := &schema.ManifestV1{
		PartnerIDs: []string{"company"},
		Registration: webhook.RegistrationV1{
			Config:   webhook.DeliveryConfig{ReceiverURL: "http://example.com/hook", ContentType: "application/json"},
			Events:   []string{"online"},
			Duration: webhook.CustomDuration(5 * time.Minute),
			Until:    now.Add(5 * time.Minute),
		},
	}

	v2 := &schema.ManifestV2{
		PartnerIds: []string{"company"},
		Registration: webhook.RegistrationV2{
			CanonicalName: "my-registration",
			Address:       "127.0.0.1:1234",
			Expires:       now.Add(5 * time.Minute),
			Webhooks: []webhook.Webhook{{
				ReceiverURLs: []string{"http://example.com/hook"},
			}},
		},
	}

	forms := []struct {
		title    string
		manifest schema.Manifest
		format   schema.OutputFormat
		field    string // a field inside the registration that shows the form
	}{
		{"v1, OutputLegacy (default)", v1, schema.OutputLegacy, "duration"},
		{"v1, OutputV1", v1, schema.OutputV1, "duration"},
		{"v2, either format", v2, schema.OutputLegacy, "canonical_name"},
	}

	for _, f := range forms {
		item, err := schema.SchemaToItemFormat(clock, f.manifest, f.format)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Each form keeps the registration under a key of its own.
		for key, registration := range item.Data {
			if key == "PartnerIDs" || key == "PartnerIds" {
				continue
			}
			field, _ := json.Marshal(registration.(map[string]any)[f.field])
			fmt.Printf("%-28s key=%-28s %s=%s\n", f.title, key, f.field, field)
		}
	}

	// Output:
	// v1, OutputLegacy (default)   key=Webhook                      duration=300000000000
	// v1, OutputV1                 key=wrp_event_stream_schema_v1   duration="5m0s"
	// v2, either format            key=wrp_event_stream_schema_v2   canonical_name="my-registration"
}
