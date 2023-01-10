/**
 * Copyright 2022 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ancla

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"time"

	// nolint:typecheck
	"github.com/xmidt-org/argus/model"
)

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
