// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"time"

	// nolint:typecheck

	"github.com/xmidt-org/argus/model"
	webhook "github.com/xmidt-org/webhook-schema"
)

type RegistryV1 struct {
	PartnerIDs []string
	Webhook    webhook.RegistrationV1
}
type RegistryV2 struct {
	PartnerIds   []string
	Registration webhook.RegistrationV2
}

func (v1 RegistryV1) GetId() string {
	return v1.Webhook.Config.ReceiverURL
}

func (v1 RegistryV1) GetUntil() time.Time {
	return v1.Webhook.Until
}
func (v2 RegistryV2) GetId() string {
	return v2.Registration.CanonicalName
}

func (v2 RegistryV2) GetUntil() time.Time {
	return v2.Registration.Expires
}

func InternalWebhookToItem(now func() time.Time, iw webhook.Register) (model.Item, error) {
	encodedWebhook, err := json.Marshal(iw)
	if err != nil {
		return model.Item{}, err
	}
	var data map[string]interface{}

	err = json.Unmarshal(encodedWebhook, &data)
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

func ItemToInternalWebhook(i model.Item) (r webhook.Register, e error) {
	var v1 RegistryV1
	var v2 RegistryV2
	encodedWebhook, e := json.Marshal(i.Data)
	if e != nil {
		return nil, e
	}

	for v := range i.Data {
		if v == "PartnerIds" {
			continue
		} else if v == "Registration" {
			e = json.Unmarshal(encodedWebhook, &v2)
			if e != nil {
				return nil, e
			}
			return v2, nil
		} else if v == "Webhook" {
			e = json.Unmarshal(encodedWebhook, &v1)
			if e != nil {
				return nil, e
			}
			return v1, nil
		}
	}

	return nil, fmt.Errorf("could not unmarshal data into either RegistryV1 or RegistryV2")
}

func ItemsToInternalWebhooks(items []model.Item) ([]webhook.Register, error) {
	iws := []webhook.Register{}
	for _, item := range items {
		iw, err := ItemToInternalWebhook(item)
		if err != nil {
			return nil, err
		}
		iws = append(iws, iw)
	}
	return iws, nil
}

func InternalWebhooksToWebhooks(iws []webhook.Register) []any {
	w := make([]any, 0, len(iws))
	for _, iw := range iws {
		switch r := iw.(type) {
		case RegistryV1:
			w = append(w, r.Webhook)
		case RegistryV2:
			w = append(w, r.Registration)
		}
	}
	return w
}
