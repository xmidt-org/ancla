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
	"net/http"
	"time"

	"github.com/go-kit/kit/metrics/provider"
	kithttp "github.com/go-kit/kit/transport/http"
)

// NewAddWebhookHandler returns an HTTP handler for adding
// a webhook registration.
func NewAddWebhookHandler(s Service, config HandlerConfig) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		addWebhookRequestDecoder(newTransportConfig(config)),
		encodeAddWebhookResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}

// NewGetAllWebhooksHandler returns an HTTP handler for fetching
// all the currently registered webhooks.
func NewGetAllWebhooksHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		kithttp.NopRequestDecoder,
		encodeGetAllWebhooksResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}

// HandlerConfig contains configuration for all components that handlers depend on
// from the service to the transport layers.
type HandlerConfig struct {
	MetricsProvider provider.Provider
}

func newTransportConfig(hConfig HandlerConfig) transportConfig {
	if hConfig.MetricsProvider == nil {
		hConfig.MetricsProvider = provider.NewDiscardProvider()
	}
	return transportConfig{
		webhookLegacyDecodeCount: hConfig.MetricsProvider.NewCounter(WebhookLegacyDecodeCount),
		now:                      time.Now,
	}
}
