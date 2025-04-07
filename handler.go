// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"net/http"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	webhook "github.com/xmidt-org/webhook-schema"
	"go.uber.org/zap"
)

// NewAddWebhookHandler returns an HTTP handler for adding
// a webhook registration.
func NewAddWebhookHandler(s Service, config HandlerConfig) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		addWebhookRequestDecoder(newTransportConfig(config)),
		encodeAddWebhookResponse,
		kithttp.ServerErrorEncoder(errorEncoder(config.GetLogger)),
	)
}

// NewGetAllWebhooksHandler returns an HTTP handler for fetching
// all the currently registered webhooks.
func NewGetAllWebhooksHandler(s Service, config HandlerConfig) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		kithttp.NopRequestDecoder,
		encodeGetAllWebhooksResponse,
		kithttp.ServerErrorEncoder(errorEncoder(config.GetLogger)),
	)
}

// HandlerConfig contains configuration for all components that handlers depend on
// from the service to the transport layers.
type HandlerConfig struct {
	V                 webhook.Validators
	DisablePartnerIDs bool
	GetLogger         func(context.Context) *zap.Logger
}

func newTransportConfig(hConfig HandlerConfig) transportConfig {
	return transportConfig{
		now:               time.Now,
		v:                 hConfig.V,
		disablePartnerIDs: hConfig.DisablePartnerIDs,
	}
}
