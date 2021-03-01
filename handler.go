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

	kithttp "github.com/go-kit/kit/transport/http"
)

func NewAddWebhookHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newAddWebhookEndpoint(s),
		decodeAddWebhookRequest,
		encodeAddWebhookResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}

func NewGetAllWebhooksHandler(s Service) http.Handler {
	return kithttp.NewServer(
		newGetAllWebhooksEndpoint(s),
		kithttp.NopRequestDecoder,
		encodeGetAllWebhooksResponse,
		kithttp.ServerErrorEncoder(errorEncoder),
	)
}
