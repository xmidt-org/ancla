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
	webhook "github.com/xmidt-org/webhook-schema"
	"go.uber.org/zap"
)

var (
	errFailedWebhookUnmarshal    = errors.New("failed to JSON unmarshal webhook")
	errGettingPartnerIDs         = errors.New("unable to retrieve PartnerIDs")
	DefaultBasicPartnerIDsHeader = "X-Xmidt-Partner-Ids"
)

const (
	contentTypeHeader string = "Content-Type"
	jsonContentType   string = "application/json"
)

type transportConfig struct {
	now                   func() time.Time
	v                     webhook.Validators
	basicPartnerIDsHeader string
	disablePartnerIDs     bool
}

type addWebhookRequest struct {
	owner          string
	internalWebook Register
}

func encodeGetAllWebhooksResponse(ctx context.Context, rw http.ResponseWriter, response interface{}) error {
	iws := response.([]Register)
	webhooks := InternalWebhooksToWebhooks(iws)
	if webhooks == nil {
		// prefer JSON output to be "[]" instead of "<nil>"
		webhooks = []any{}
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

	opts := config.v
	// if no validators are given, we accept anything.
	if opts == nil {
		opts = append(opts, webhook.AlwaysValid())
	}

	return func(c context.Context, r *http.Request) (request interface{}, err error) {
		requestPayload, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		var (
			v1       webhook.RegistrationV1
			v2       webhook.RegistrationV2
			errs     error
			partners []string
		)

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

		err = json.Unmarshal(requestPayload, &v1)
		if err == nil {
			if err = opts.Validate(&v1); err != nil {
				return nil, &erraux.Error{Err: err, Message: "failed webhook validation", Code: http.StatusBadRequest}
			}
			wv.setV1Defaults(&v1, r.RemoteAddr)
			reg := &RegistryV1{
				PartnerIDs:   partners,
				Registration: v1,
			}

			return &addWebhookRequest{
				owner:          owner,
				internalWebook: reg,
			}, nil
		}

		errs = errors.Join(errs, err)
		err = json.Unmarshal(requestPayload, &v2)
		if err == nil {
			if err = opts.Validate(&v2); err != nil {
				return nil, &erraux.Error{Err: err, Message: "failed webhook validation", Code: http.StatusBadRequest}
			}

			reg := &RegistryV2{
				PartnerIds:   partners,
				Registration: v2,
			}

			return &addWebhookRequest{
				owner:          owner,
				internalWebook: reg,
			}, nil

		}

		errs = errors.Join(errs, err)

		return nil, &erraux.Error{Err: fmt.Errorf("%w: %v", errFailedWebhookUnmarshal, errs), Code: http.StatusBadRequest}
	}
}

func encodeAddWebhookResponse(ctx context.Context, rw http.ResponseWriter, _ interface{}) error {
	rw.Header().Set(contentTypeHeader, jsonContentType)
	rw.Write([]byte(`{"message": "Success"}`))
	return nil
}

func obfuscateSecrets(webhooks []any) {
	for i, v := range webhooks {
		switch r := v.(type) {
		case webhook.RegistrationV1:
			r.Config.Secret = "<obfuscated>"
			webhooks[i] = r
		case webhook.RegistrationV2:
			for i := range r.Webhooks {
				r.Webhooks[i].Secret = "<obfuscated>"
			}
			webhooks[i] = r
		}
	}
}

type webhookValidator struct {
	now func() time.Time
}

func (wv webhookValidator) setV1Defaults(r *webhook.RegistrationV1, requestOriginHost string) {
	if len(r.Matcher.DeviceID) == 0 {
		r.Matcher.DeviceID = []string{".*"} // match anything
	}
	if r.Until.IsZero() {
		r.Until = wv.now().Add(time.Duration(r.Duration))
	}
	if requestOriginHost != "" {
		r.Address = requestOriginHost
	}
}

func (wv webhookValidator) setV2Defaults(r *webhook.RegistrationV2) {
	//TODO: need to get registrationV2 defaults
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
