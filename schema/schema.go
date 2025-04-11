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

func SchemaToItem(now func() time.Time, manifest Manifest) (model.Item, error) {
	TTLSeconds := int64(math.Max(0, manifest.GetUntil().Sub(now()).Seconds()))
	encodedSchema, err := json.Marshal(manifest)
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

func ItemToSchema(i model.Item) (Manifest, error) {
	var (
		v1   *ManifestV1
		v2   *ManifestV2
		errs error
	)

	encodedSchema, err := json.Marshal(i.Data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(encodedSchema, &v2)
	if err == nil && v2.Registration.CanonicalName != "" {
		return v2, nil
	} else if err != nil {
		errs = errors.Join(errs, fmt.Errorf("%T Unmarshal error: %s", v2, err))
	}

	err = json.Unmarshal(encodedSchema, &v1)
	if err == nil {
		return v1, nil
	}

	errs = errors.Join(errs, fmt.Errorf("%T Unmarshal error: %s", v1, err))

	return nil, fmt.Errorf("could not Unmarshal data into either %T or %T: %s", v1, v2, errs)
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
