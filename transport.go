/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/metrics"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/httpaux"

	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/xhttp"
)

const defaultWebhookExpiration time.Duration = time.Minute * 5

var (
	errInvalidConfigURL = errors.New("invalid Config URL")
	errInvalidEvents    = errors.New("invalid events")
)

const (
	contentTypeHeader string = "Content-Type"
	jsonContentType   string = "application/json"
)

type transportConfig struct {
	webhookLegacyDecodeCount metrics.Counter
}

type addWebhookRequest struct {
	owner   string
	webhook Webhook
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	webhooks := response.([]Webhook)
	if webhooks == nil {
		// prefer JSON output to be "[]" instead of "<nil>"
		webhooks = []Webhook{}
	}
	obfuscateSecrets(webhooks)
	encodedWebhooks, err := json.Marshal(&webhooks)
	if err != nil {
		return err
	}

	rw.Header().Set(contentTypeHeader, jsonContentType)
	_, err = rw.Write(encodedWebhooks)
	return err
}

func addWebhookRequestDecoder(config transportConfig) kithttp.DecodeRequestFunc {
	wv := webhookValidator{
		now: time.Now,
	}
	return func(c context.Context, r *http.Request) (request interface{}, err error) {
		requestPayload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var webhook Webhook

		err = json.Unmarshal(requestPayload, &webhook)
		if err != nil {
			webhook, err = getFirstFromList(requestPayload)
			if err != nil {
				return nil, err
			}
			config.webhookLegacyDecodeCount.With(URLLabel, webhook.Config.URL).Add(1)
		}
		err = wv.validateWebhook(&webhook, r.RemoteAddr)
		if err != nil {
			return nil, err
		}

		return &addWebhookRequest{
			owner:   getOwner(r.Context()),
			webhook: webhook,
		}, nil
	}
}

func encodeAddWebhookResponse(ctx context.Context, rw http.ResponseWriter, _ interface{}) error {
	rw.Header().Set(contentTypeHeader, jsonContentType)
	rw.Write([]byte(`{"message": "Success"}`))
	return nil
}

func getOwner(ctx context.Context) string {
	auth, ok := bascule.FromContext(ctx)
	if !ok {
		return ""
	}
	switch auth.Token.Type() {
	case "jwt", "basic":
		return auth.Token.Principal()
	}
	return ""
}

func getFirstFromList(requestPayload []byte) (Webhook, error) {
	var webhooks []Webhook

	err := json.Unmarshal(requestPayload, &webhooks)
	if err != nil {
		return Webhook{}, err
	}

	if len(webhooks) < 1 {
		return Webhook{}, &xhttp.Error{Text: "no webhooks in request data list", Code: http.StatusBadRequest}
	}
	return webhooks[0], nil
}

func obfuscateSecrets(webhooks []Webhook) {
	for i := range webhooks {
		webhooks[i].Config.Secret = "<obfuscated>"
	}
}

type webhookValidator struct {
	now func() time.Time
}

func (wv webhookValidator) validateWebhook(webhook *Webhook, requestOriginAddress string) (err error) {
	if strings.TrimSpace(webhook.Config.URL) == "" {
		return &httpaux.Error{Code: http.StatusBadRequest, Err: errInvalidConfigURL}
	}

	if len(webhook.Events) == 0 {
		return &httpaux.Error{Code: http.StatusBadRequest, Err: errInvalidEvents}
	}

	// TODO Validate content type ?  What about different types?

	if len(webhook.Matcher.DeviceID) == 0 {
		webhook.Matcher.DeviceID = []string{".*"} // match anything
	}

	if webhook.Address == "" && requestOriginAddress != "" {
		host, _, err := net.SplitHostPort(requestOriginAddress)
		if err != nil {
			return err
		}
		webhook.Address = host
	}

	// always set duration to default
	webhook.Duration = defaultWebhookExpiration

	if webhook.Until.Equal(time.Time{}) {
		webhook.Until = wv.now().Add(webhook.Duration)
	}

	return nil
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set(contentTypeHeader, jsonContentType)
	code := http.StatusInternalServerError
	var sc kithttp.StatusCoder
	if errors.As(err, &sc) {
		code = sc.StatusCode()
	}
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(
		map[string]interface{}{
			"message": err.Error(),
		})
}
