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
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/webhook-schema"
)

type Register interface {
	GetId() string
	GetUntil() time.Time
	GetBachHint() *webhook.BatchHint
}

type RegistryV1 struct {
	PartnerIDs   []string
	Registration webhook.RegistrationV1 `json:"Webhook"`
}

type RegistryV2 struct {
	PartnerIds   []string
	Registration webhook.RegistrationV2
}

func (v1 *RegistryV1) GetId() string {
	return v1.Registration.Config.ReceiverURL
}

func (v1 *RegistryV1) GetUntil() time.Time {
	return v1.Registration.Until
}

// Need this for now - as we won't be checking the type of the Register
// in the caduceus code. This could be removed once the work in caduceus is finalized.
func (v1 *RegistryV1) GetBachHint() *webhook.BatchHint {
	return nil
}

func (v2 *RegistryV2) GetId() string {
	return v2.Registration.CanonicalName
}

func (v2 *RegistryV2) GetUntil() time.Time {
	return v2.Registration.Expires
}

func (v2 *RegistryV2) GetBachHint() *webhook.BatchHint {
	return &v2.Registration.BatchHint
}

type InternalWebhook struct {
	PartnerIDs []string
	Webhook    Webhook
}

func InternalWebhookToItem(now func() time.Time, iw InternalWebhook) (model.Item, error) {
	encodedWebhook, err := json.Marshal(iw)
	if err != nil {
		return model.Item{}, err
	}
	var data map[string]interface{}

	err = json.Unmarshal(encodedWebhook, &data)
	if err != nil {
		return model.Item{}, err
	}

	SecondsToExpiry := iw.Webhook.Until.Sub(now()).Seconds()
	TTLSeconds := int64(math.Max(0, SecondsToExpiry))

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(iw.Webhook.Config.URL)))

	return model.Item{
		Data: data,
		ID:   checksum,
		TTL:  &TTLSeconds,
	}, nil
}

func ItemToInternalWebhook(i model.Item) (InternalWebhook, error) {
	encodedWebhook, err := json.Marshal(i.Data)
	if err != nil {
		return InternalWebhook{}, err
	}
	var iw InternalWebhook
	err = json.Unmarshal(encodedWebhook, &iw)
	if err != nil {
		return InternalWebhook{}, err
	}
	return iw, nil
}

func ItemsToInternalWebhooks(items []model.Item) ([]InternalWebhook, error) {
	iws := []InternalWebhook{}
	for _, item := range items {
		iw, err := ItemToInternalWebhook(item)
		if err != nil {
			return nil, err
		}
		iws = append(iws, iw)
	}
	return iws, nil
}

func InternalWebhooksToWebhooks(iws []InternalWebhook) []Webhook {
	w := make([]Webhook, 0, len(iws))
	for _, iw := range iws {
		w = append(w, iw.Webhook)
	}
	return w
}
