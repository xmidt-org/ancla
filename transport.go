// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/spf13/cast"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/httpaux/erraux"
	"github.com/xmidt-org/sallust"
	webhook "github.com/xmidt-org/webhook-schema"
	"go.uber.org/zap"
)

var (
	errFailedWebhookUnmarshal    = errors.New("failed to JSON unmarshal webhook")
	errAuthIsNotOfTypeBasicOrJWT = errors.New("auth is not of type Basic of JWT")
	errGettingPartnerIDs         = errors.New("unable to retrieve PartnerIDs")
	errAuthNotPresent            = errors.New("auth not present")
	errParsingToken              = errors.New("unable to  parse token")
	errPartnerIDsDoNotExist      = errors.New("partnerIDs do not exist")
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

		partners, err = extractPartnerIDs(config, r)
		if err != nil && !config.disablePartnerIDs {
			return nil, &erraux.Error{Err: err, Message: "failed getting partnerIDs", Code: http.StatusBadRequest}
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
				owner:          getOwner(r),
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
				owner:          getOwner(r),
				internalWebook: reg,
			}, nil

		}

		errs = errors.Join(errs, err)

		return nil, &erraux.Error{Err: fmt.Errorf("%w: %v", errFailedWebhookUnmarshal, errs), Code: http.StatusBadRequest}
	}
}

func extractPartnerIDs(config transportConfig, r *http.Request) ([]string, error) {
	var partners []string

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return nil, errAuthNotPresent
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		partnerIdsHeader := r.Header[config.basicPartnerIDsHeader]
		for _, value := range partnerIdsHeader {
			fields := strings.Split(value, ",")
			for i := 0; i < len(fields); i++ {
				fields[i] = strings.TrimSpace(fields[i])
			}
			partners = append(partners, fields...)
		}
		return partners, nil
	} else if strings.HasPrefix(authHeader, "Bearer ") {
		tok, err := jwt.ParseHeader(r.Header, "Authorization", jwt.WithVerify(false))
		if err != nil {
			return nil, fmt.Errorf("%w, %v", errParsingToken, err)
		}
		claimsMap := tok.PrivateClaims()
		if keysI, ok := claimsMap["allowedResources"]; ok {
			keysMap, ok := keysI.(map[string]interface{})
			if !ok {
				return nil, errPartnerIDsDoNotExist
			}
			if i, ok := keysMap["allowedPartners"]; ok {
				partnerIds, err := cast.ToStringSliceE(i)
				if err != nil || len(partnerIds) == 0 {
					return nil, fmt.Errorf("%w: %v", errGettingPartnerIDs, err)

				}
				partners = partnerIds
			} else {
				return nil, errPartnerIDsDoNotExist
			}
		}
	} else {
		return partners, errAuthIsNotOfTypeBasicOrJWT
	}
	return partners, nil
}

func encodeAddWebhookResponse(ctx context.Context, rw http.ResponseWriter, _ interface{}) error {
	rw.Header().Set(contentTypeHeader, jsonContentType)
	rw.Write([]byte(`{"message": "Success"}`))
	return nil
}

func getOwner(r *http.Request) (owner string) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return
	}

	if strings.HasPrefix(auth, "Bearer ") {
		tok, _ := jwt.ParseHeader(r.Header, "Authorization", jwt.WithVerify(false))
		val, ok := tok.Get("sub")
		if !ok {
			return
		}
		owner, ok = val.(string)
		if !ok {
			return ""
		} else {
			return owner
		}
	} else if strings.HasPrefix(auth, "Basic ") {
		i := strings.Index(auth, " ")
		if i < 1 {
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(auth[i+len(" "):])
		if err != nil {
			return
		}
		j := bytes.IndexByte(decoded, ':')
		if i <= 0 {
			return
		}
		owner = string(decoded[:j])
		return

	}
	return
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

func errorEncoder() kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		w.Header().Set(contentTypeHeader, jsonContentType)
		code := http.StatusInternalServerError
		var sc kithttp.StatusCoder
		if errors.As(err, &sc) {
			code = sc.StatusCode()
		}

		logger := sallust.Get(ctx)
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
