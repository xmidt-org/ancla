// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"
)

// Decorator decorates http requests with authorization header(s).
type Decorator interface {
	// Decorate decorates the given http request with authorization header(s).
	Decorate(ctx context.Context, req *http.Request) error
}

type DecoratorFunc func(context.Context, *http.Request) error

func (f DecoratorFunc) Decorate(ctx context.Context, req *http.Request) error { return f(ctx, req) }

var Nop = DecoratorFunc(func(context.Context, *http.Request) error { return nil })
