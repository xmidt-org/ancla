// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoratorFunc(t *testing.T) {
	assert := assert.New(t)
	errFake := errors.New("decorate failed")
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	require.NoError(t, err)

	d := DecoratorFunc(func(_ context.Context, r *http.Request) error {
		r.Header.Set("Authorization", "Bearer token")
		return errFake
	})

	assert.ErrorIs(d.Decorate(context.Background(), req), errFake)
	assert.Equal("Bearer token", req.Header.Get("Authorization"))
}

func TestNop(t *testing.T) {
	assert := assert.New(t)
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	require.NoError(t, err)

	assert.NoError(Nop.Decorate(context.Background(), req))
	assert.Empty(req.Header)
}

func TestMockDecorator(t *testing.T) {
	assert := assert.New(t)
	errFake := errors.New("mock failed")
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	require.NoError(t, err)

	m := new(MockDecorator)
	// nolint:typecheck
	m.On("Decorate").Return(errFake)

	assert.ErrorIs(m.Decorate(context.Background(), req), errFake)
	assert.Equal(MockAuthHeaderValue, req.Header.Get(MockAuthHeaderName))
	// nolint:typecheck
	m.AssertExpectations(t)
}
