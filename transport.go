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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/metrics"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/spf13/cast"
	"github.com/xmidt-org/httpaux/erraux"
	"github.com/xmidt-org/webpa-common/v2/basculechecks"

	"github.com/xmidt-org/bascule"
)

var (
	errNoWebhooksInLegacyDecode  = errors.New("no webhooks to decode in legacy decoding format")
	errFailedWebhookUnmarshal    = errors.New("failed to JSON unmarshal webhook")
	errAuthIsNotOfTypeBasicOrJWT = errors.New("auth is not of type Basic of JWT")
	errGettingPartnerIDs         = errors.New("unable to retrieve PartnerIDs")
	errAuthNotPresent            = errors.New("auth not present")
	errAuthTokenIsNil            = errors.New("auth token is nil")
	errPartnerIDsDoNotExist      = errors.New("partnerIDs do not exist")
	DefaultBasicPartnerIDsHeader = "X-Xmidt-Partner-Ids"
	jwtstr                       = "jwt"
	basicstr                     = "basic"
)

const (
	contentTypeHeader string = "Content-Type"
	jsonContentType   string = "application/json"
)

type transportConfig struct {
	webhookLegacyDecodeCount metrics.Counter
	now                      func() time.Time
	v                        Validator
	basicPartnerIDsHeader    string
	reqPartnerIDs            bool
}

type addWebhookRequest struct {
	owner          string
	internalWebook InternalWebhook
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	iws := response.([]InternalWebhook)
	webhooks := internalWebhooksToWebhooks(iws)
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
		now: config.now,
	}

	if config.basicPartnerIDsHeader == "" {
		config.basicPartnerIDsHeader = DefaultBasicPartnerIDsHeader
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

		err = config.v.Validate(webhook)
		if err != nil {
			return nil, &erraux.Error{Err: err, Message: "failed webhook validation", Code: http.StatusBadRequest}
		}

		wv.setWebhookDefaults(&webhook, r.RemoteAddr)

		var partners []string
		partners, err = extractPartnerIDs(config, c, r)
		if err != nil && config.reqPartnerIDs {
			return nil, &erraux.Error{Err: err, Message: "failed getting partnerIDs", Code: http.StatusBadRequest}
		}

		return &addWebhookRequest{
			owner: getOwner(r.Context()),
			internalWebook: InternalWebhook{
				Webhook:    webhook,
				PartnerIDs: partners,
			},
		}, nil
	}
}

func extractPartnerIDs(config transportConfig, c context.Context, r *http.Request) ([]string, error) {
	auth, present := bascule.FromContext(c)
	if !present {
		return nil, errAuthNotPresent
	}
	if auth.Token == nil {
		return nil, errAuthTokenIsNil
	}

	var partners []string

	switch auth.Token.Type() {
	case basicstr:
		authHeader := r.Header[config.basicPartnerIDsHeader]
		for _, value := range authHeader {
			fields := strings.Split(value, ",")
			for i := 0; i < len(fields); i++ {
				fields[i] = strings.TrimSpace(fields[i])
			}
			partners = append(partners, fields...)
		}
		return partners, nil
	case jwtstr:
		authToken := auth.Token
		partnersInterface, attrExist := bascule.GetNestedAttribute(authToken.Attributes(), basculechecks.PartnerKeys()...)
		if !attrExist {
			return nil, errPartnerIDsDoNotExist
		}
		vals, err := cast.ToStringSliceE(partnersInterface)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errGettingPartnerIDs, err)
		}
		partners = vals
		return partners, nil
	}
	return nil, errAuthIsNotOfTypeBasicOrJWT
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
	case jwtstr, basicstr:
		return auth.Token.Principal()
	}
	return ""
}

func getFirstFromList(requestPayload []byte) (Webhook, error) {
	var webhooks []Webhook

	err := json.Unmarshal(requestPayload, &webhooks)
	if err != nil {
		return Webhook{}, &erraux.Error{Err: errFailedWebhookUnmarshal, Code: http.StatusBadRequest}
	}

	if len(webhooks) < 1 {
		return Webhook{}, &erraux.Error{Err: errNoWebhooksInLegacyDecode, Code: http.StatusBadRequest}
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

func (wv webhookValidator) setWebhookDefaults(webhook *Webhook, requestOriginHost string) {
	if len(webhook.Matcher.DeviceID) == 0 {
		webhook.Matcher.DeviceID = []string{".*"} // match anything
	}

	if webhook.Address == "" && requestOriginHost != "" {
		webhook.Address = requestOriginHost
	}

	if webhook.Until.IsZero() {
		webhook.Until = wv.now().Add(webhook.Duration)
	}
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
