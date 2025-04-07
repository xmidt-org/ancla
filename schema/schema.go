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

type RegistryManifest interface {
	GetId() string
	GetUntil() time.Time
}

type RegistryV1 struct {
	PartnerIDs []string
	// nolint:staticcheck
	Registration webhook.RegistrationV1 `json:"registration_v1"`
}

type RegistryV2 struct {
	PartnerIds   []string
	Registration webhook.RegistrationV2 `json:"registration_v2"`
}

func (v1 *RegistryV1) GetId() string {
	return v1.Registration.Config.ReceiverURL
}

func (v1 *RegistryV1) GetUntil() time.Time {
	return v1.Registration.Until
}

func (v2 *RegistryV2) GetId() string {
	return v2.Registration.CanonicalName
}

func (v2 *RegistryV2) GetUntil() time.Time {
	return v2.Registration.Expires
}

func SchemaToItem(now func() time.Time, iw RegistryManifest) (model.Item, error) {
	encodedWRPEventStream, err := json.Marshal(iw)
	if err != nil {
		return model.Item{}, err
	}
	var data map[string]interface{}

	err = json.Unmarshal(encodedWRPEventStream, &data)
	if err != nil {
		return model.Item{}, err
	}

	SecondsToExpiry := iw.GetUntil().Sub(now()).Seconds()
	TTLSeconds := int64(math.Max(0, SecondsToExpiry))

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(iw.GetId())))

	return model.Item{
		Data: data,
		ID:   checksum,
		TTL:  &TTLSeconds,
	}, nil
}

func ItemToSchema(i model.Item) (RegistryManifest, error) {
	var (
		v1   *RegistryV1
		v2   *RegistryV2
		errs error
	)
	encodedWRPEventStream, err := json.Marshal(i.Data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(encodedWRPEventStream, &v2)
	if err == nil && v2.Registration.CanonicalName != "" {
		return v2, nil
	} else if err != nil {
		errs = errors.Join(errs, fmt.Errorf("RegistryV2 unmarshal error: %s", err))
	}

	err = json.Unmarshal(encodedWRPEventStream, &v1)
	if err == nil {
		return v1, nil
	}

	errs = errors.Join(errs, fmt.Errorf("RegistryV1 unmarshal error: %s", err))

	return nil, fmt.Errorf("could not unmarshal data into either RegistryV1 or RegistryV2: %s", errs)
}

func ItemsToSchemas(items []model.Item) ([]RegistryManifest, error) {
	iws := []RegistryManifest{}
	for _, item := range items {
		iw, err := ItemToSchema(item)
		if err != nil {
			return nil, err
		}
		iws = append(iws, iw)
	}
	return iws, nil
}

func SchemasToWRPEventStreams(iws []RegistryManifest) []any {
	w := make([]any, 0, len(iws))
	for _, iw := range iws {
		switch r := iw.(type) {
		case *RegistryV1:
			w = append(w, r.Registration)
		case *RegistryV2:
			w = append(w, r.Registration)
		}
	}
	return w
}
