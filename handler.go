// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"net/http"
	"time"

	webhook "github.com/xmidt-org/webhook-schema"
	"go.uber.org/zap"
)

// NewAddWRPEventStreamHandler returns an HTTP handler for adding
// a wrpEventStream registration.
func NewAddWRPEventStreamHandler(s Service, config HandlerConfig) http.Handler {
	return newServer(
		newAddWRPEventStreamEndpoint(s),
		addWRPEventStreamRequestDecoder(newTransportConfig(config)),
		encodeAddWRPEventStreamResponse,
		errorEncoder(config.GetLogger),
	)
}

// NewGetAllWRPEventStreamsHandler returns an HTTP handler for fetching
// all the currently registered wrpEventStreams.
func NewGetAllWRPEventStreamsHandler(s Service, config HandlerConfig) http.Handler {
	return newServer(
		newGetAllWRPEventStreamsEndpoint(s),
		nopRequestDecoder,
		encodeGetAllWRPEventStreamsResponse,
		errorEncoder(config.GetLogger),
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

// server wires a decoder, an endpoint, and an encoder into an http.Handler.
// Any error from the decoder, endpoint, or encoder is passed to the error
// encoder, which is responsible for writing the response.
type server struct {
	e      endpoint
	dec    decodeRequestFunc
	enc    encodeResponseFunc
	errEnc errorEncoderFunc
}

func newServer(e endpoint, dec decodeRequestFunc, enc encodeResponseFunc, errEnc errorEncoderFunc) http.Handler {
	return &server{
		e:      e,
		dec:    dec,
		enc:    enc,
		errEnc: errEnc,
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	request, err := s.dec(ctx, r)
	if err != nil {
		s.errEnc(ctx, err, w)
		return
	}

	response, err := s.e(ctx, request)
	if err != nil {
		s.errEnc(ctx, err, w)
		return
	}

	if err := s.enc(ctx, w, response); err != nil {
		s.errEnc(ctx, err, w)
		return
	}
}
