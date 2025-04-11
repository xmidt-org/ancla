// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func newAddWRPEventStreamEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(*addWRPEventStreamRequest)
		return nil, s.Add(ctx, r.owner, r.internalWebook)
	}
}

func newGetAllWRPEventStreamsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		return s.GetAll(ctx)
	}
}
