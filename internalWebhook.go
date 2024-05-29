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
func (v2 *RegistryV2) GetId() string {
	return v2.Registration.CanonicalName
}

func (v2 *RegistryV2) GetUntil() time.Time {
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

// TODO: i don't believe this function will work correctly
func ItemToInternalWebhook(i model.Item) (webhook.Register, error) {
	encodedWebhook, err := json.Marshal(i.Data)
	if err != nil {
		return nil, err
	}
	var iw webhook.Register
	err = json.Unmarshal(encodedWebhook, &iw)
	if err != nil {
		return nil, err
	}
	return iw, nil
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

func InternalWebhooksToWebhooks(iws []webhook.Register) []webhook.RegistrationV1 {
	w := make([]webhook.RegistrationV1, 0, len(iws))
	for _, iw := range iws {
		switch r1 := iw.(type) {
		case RegistryV1:
			w = append(w, r1.Webhook)
		}
	}
	return w
}
