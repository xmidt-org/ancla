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
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/provider"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

// NewAddWebhookHandler returns an HTTP handler for adding
// a webhook registration.
func NewAddWebhookHandler(s Service, config HandlerConfig) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		addWebhookRequestDecoder(newTransportConfig(config)),
		encodeAddWebhookResponse,
		kithttp.ServerErrorEncoder(errorEncoder(config.GetLoggerFunc)),
	)
}

// NewGetAllWebhooksHandler returns an HTTP handler for fetching
// all the currently registered webhooks.
func NewGetAllWebhooksHandler(s Service, config HandlerConfig) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		kithttp.NopRequestDecoder,
		encodeGetAllWebhooksResponse,
		kithttp.ServerErrorEncoder(errorEncoder(config.GetLoggerFunc)),
	)
}

// HandlerConfig contains configuration for all components that handlers depend on
// from the service to the transport layers.
type HandlerConfig struct {
	MetricsProvider   provider.Provider
	V                 Validator
	DisablePartnerIDs bool
	GetLoggerFunc     func(context.Context) log.Logger
}

func newTransportConfig(hConfig HandlerConfig) transportConfig {
	if hConfig.MetricsProvider == nil {
		hConfig.MetricsProvider = provider.NewDiscardProvider()
	}
	return transportConfig{
		now:               time.Now,
		v:                 hConfig.V,
		disablePartnerIDs: hConfig.DisablePartnerIDs,
	}
}

func getLogger(ctx context.Context) log.Logger {
	logger := log.With(logging.GetLogger(ctx), "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	return logger
}
