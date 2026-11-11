// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"

	"github.com/xmidt-org/ancla/schema"
)

// endpoint is the business-logic step of a handler.  It receives the decoded
// request and returns a response for the encoder, or an error for the error
// encoder.
type endpoint func(ctx context.Context, request any) (any, error)

func newAddWRPEventStreamEndpoint(add func(ctx context.Context, owner string, m schema.Manifest) error) endpoint {
	return func(ctx context.Context, request any) (any, error) {
		r := request.(*addWRPEventStreamRequest)
		return nil, add(ctx, r.owner, r.manifest)
	}
}

func newGetAllWRPEventStreamsEndpoint(list func(ctx context.Context) ([]schema.Manifest, error)) endpoint {
	return func(ctx context.Context, _ any) (any, error) {
		return list(ctx)
	}
}
