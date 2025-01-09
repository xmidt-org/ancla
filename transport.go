// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/ancla/auth"
	"github.com/xmidt-org/httpaux/erraux"
	"go.uber.org/zap"
)

var (
	errFailedWebhookUnmarshal    = errors.New("failed to JSON unmarshal webhook")
	errAuthIsNotOfTypeBasicOrJWT = errors.New("auth is not of type Basic of JWT")
	errGettingPartnerIDs         = errors.New("unable to retrieve PartnerIDs")
	DefaultBasicPartnerIDsHeader = "X-Xmidt-Partner-Ids"
)

const (
	contentTypeHeader string = "Content-Type"
	jsonContentType   string = "application/json"
)

type transportConfig struct {
	now                   func() time.Time
	v                     Validator
	basicPartnerIDsHeader string
	disablePartnerIDs     bool
	auth                  auth.Acquirer
}

type addWebhookRequest struct {
	owner          string
	internalWebook InternalWebhook
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	iws := response.([]InternalWebhook)
	webhooks := InternalWebhooksToWebhooks(iws)
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

	// if no validators are given, we accept anything.
	if config.v == nil {
		config.v = AlwaysValid()
	}

	return func(c context.Context, r *http.Request) (request interface{}, err error) {
		requestPayload, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var wr WebhookRegistration

		err = json.Unmarshal(requestPayload, &wr)
		if err != nil {
			var e *json.UnmarshalTypeError
			if errors.As(err, &e) {
				return nil, &erraux.Error{Err: fmt.Errorf("%w: %v must be of type %v", errFailedWebhookUnmarshal, e.Field, e.Type), Code: http.StatusBadRequest}
			}
			return nil, &erraux.Error{Err: fmt.Errorf("%w: %v", errFailedWebhookUnmarshal, err), Code: http.StatusBadRequest}
		}

		webhook := wr.ToWebhook()
		err = config.v.Validate(webhook)
		if err != nil {
			return nil, &erraux.Error{Err: err, Message: "failed webhook validation", Code: http.StatusBadRequest}
		}

		wv.setWebhookDefaults(&webhook, r.RemoteAddr)

		partners, ok := auth.GetPartnerIDs(r.Context())
		if !ok {
			if !config.disablePartnerIDs {
				return nil, &erraux.Error{Err: errGettingPartnerIDs, Message: "failed getting partnerIDs", Code: http.StatusBadRequest}
			}
			partners = []string{}
		}

		owner, ok := auth.GetPrincipal(r.Context())
		if !ok {
			owner = ""
		}

		return &addWebhookRequest{
			owner: owner,
			internalWebook: InternalWebhook{
				Webhook:    webhook,
				PartnerIDs: partners,
			},
		}, nil
	}
}

func encodeAddWebhookResponse(ctx context.Context, rw http.ResponseWriter, _ interface{}) error {
	rw.Header().Set(contentTypeHeader, jsonContentType)
	rw.Write([]byte(`{"message": "Success"}`))
	return nil
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
	if webhook.Until.IsZero() {
		webhook.Until = wv.now().Add(webhook.Duration)
	}
	if requestOriginHost != "" {
		webhook.Address = requestOriginHost
	}

}

func errorEncoder(getLogger func(context.Context) *zap.Logger) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		w.Header().Set(contentTypeHeader, jsonContentType)
		code := http.StatusInternalServerError
		var sc kithttp.StatusCoder
		if errors.As(err, &sc) {
			code = sc.StatusCode()
		}

		logger := getLogger(ctx)
		if logger != nil && code != http.StatusNotFound {
			logger.Error("sending non-200, non-404 response", zap.Int("code", code), zap.Error(err))
		}

		w.WriteHeader(code)

		json.NewEncoder(w).Encode(
			map[string]interface{}{
				"message": err.Error(),
			})
	}
}
