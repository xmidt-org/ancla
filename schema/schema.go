// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	// nolint:typecheck
	"github.com/xmidt-org/ancla/model"
	webhook "github.com/xmidt-org/webhook-schema"
)

const (
	keyV1     = "wrp_event_stream_schema_v1"
	keyV2     = "wrp_event_stream_schema_v2"
	keyLegacy = "Webhook"
)

var (
	errNilItemData      = errors.New("item data is nil")
	errUnrecognizedItem = errors.New("item data is not a recognized wrpEventStream layout")
)

type Manifest interface {
	GetId() string
	GetUntil() time.Time
}

type ManifestV1 struct {
	PartnerIDs []string
	// nolint:staticcheck
	Registration webhook.RegistrationV1 `json:"wrp_event_stream_schema_v1"`
}

type ManifestV2 struct {
	PartnerIds   []string
	Registration webhook.RegistrationV2 `json:"wrp_event_stream_schema_v2"`
}

func (v1 *ManifestV1) GetId() string {
	return v1.Registration.Config.ReceiverURL
}

func (v1 *ManifestV1) GetUntil() time.Time {
	return v1.Registration.Until
}

func (v2 *ManifestV2) GetId() string {
	return v2.Registration.CanonicalName
}

func (v2 *ManifestV2) GetUntil() time.Time {
	return v2.Registration.Expires
}

// SchemaToItem converts a manifest into a store item using OutputLegacy.
func SchemaToItem(now func() time.Time, manifest Manifest) (model.Item, error) {
	return SchemaToItemFormat(now, manifest, OutputLegacy)
}

// SchemaToItemFormat converts a manifest into a store item.  The format only
// affects v1 manifests; a v2 manifest is stored under
// "wrp_event_stream_schema_v2" whatever the format.
func SchemaToItemFormat(now func() time.Time, manifest Manifest, format OutputFormat) (model.Item, error) {
	if format != OutputLegacy && format != OutputV1 {
		return model.Item{}, fmt.Errorf("%w: %d", errUnknownOutputFormat, int(format))
	}

	TTLSeconds := int64(math.Max(0, manifest.GetUntil().Sub(now()).Seconds()))

	// OutputV1 is the manifest's own encoding.  Its duration is a string,
	// which is what releases reading that label expect.
	var toEncode any = manifest
	if v1, ok := manifest.(*ManifestV1); ok && format == OutputLegacy {
		toEncode = toLegacy(v1)
	}

	encodedSchema, err := json.Marshal(toEncode)
	if err != nil {
		return model.Item{}, err
	}

	var data map[string]any
	if err = json.Unmarshal(encodedSchema, &data); err != nil {
		return model.Item{}, err
	}

	return model.Item{
		Data: data,
		ID:   fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.GetId()))),
		TTL:  &TTLSeconds,
	}, nil
}

// ItemToSchema converts a store item written by any version of ancla into a
// Manifest.  Items in the v0.3.12 layout, and those stored under
// "wrp_event_stream_schema_v1", are returned as *ManifestV1.
func ItemToSchema(i model.Item) (Manifest, error) {
	if i.Data == nil {
		return nil, errNilItemData
	}

	encodedSchema, err := json.Marshal(i.Data)
	if err != nil {
		return nil, err
	}

	// Layouts are tried newest first, and only when their key is present, so
	// data in one layout is never mistaken for another.  A layout that fails
	// to decode falls through to the next.
	var errs error
	if _, ok := i.Data[keyV2]; ok {
		var v2 ManifestV2
		if err := json.Unmarshal(encodedSchema, &v2); err == nil {
			return &v2, nil
		} else {
			errs = errors.Join(errs, fmt.Errorf("%T Unmarshal error: %w", &v2, err))
		}
	}

	if _, ok := i.Data[keyV1]; ok {
		var v1 interimItem
		if err := json.Unmarshal(encodedSchema, &v1); err == nil {
			return v1.manifest(), nil
		} else {
			errs = errors.Join(errs, fmt.Errorf("v1 Unmarshal error: %w", err))
		}
	}

	if _, ok := i.Data[keyLegacy]; ok {
		var l legacyItem
		if err := json.Unmarshal(encodedSchema, &l); err == nil {
			return l.manifest(), nil
		} else {
			errs = errors.Join(errs, fmt.Errorf("legacy Unmarshal error: %w", err))
		}
	}

	if errs == nil {
		return nil, errUnrecognizedItem
	}

	return nil, fmt.Errorf("could not Unmarshal data into any known layout: %w", errs)
}

func ItemsToSchemas(items []model.Item) ([]Manifest, error) {
	ms := []Manifest{}
	for _, item := range items {
		m, err := ItemToSchema(item)
		if err != nil {
			return nil, err
		}

		ms = append(ms, m)
	}

	return ms, nil
}

func SchemasToWRPEventStreams(manifests []Manifest) []any {
	w := make([]any, 0, len(manifests))
	for _, m := range manifests {
		switch r := m.(type) {
		case *ManifestV1:
			w = append(w, r.Registration)
		case *ManifestV2:
			w = append(w, r.Registration)
		}
	}

	return w
}
