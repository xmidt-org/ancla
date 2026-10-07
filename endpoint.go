// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
)

// endpoint is the business-logic step of a handler.  It receives the decoded
// request and returns a response for the encoder, or an error for the error
// encoder.
type endpoint func(ctx context.Context, request any) (any, error)

func newAddWRPEventStreamEndpoint(s Service) endpoint {
	return func(ctx context.Context, request any) (any, error) {
		r := request.(*addWRPEventStreamRequest)
		return nil, s.Add(ctx, r.owner, r.internalWebook)
	}
}

func newGetAllWRPEventStreamsEndpoint(s Service) endpoint {
	return func(ctx context.Context, _ any) (any, error) {
		return s.GetAll(ctx)
	}
}
