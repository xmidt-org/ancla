// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package auth

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/mock"
)

const (
	MockAuthHeaderName  = "MockAuthorization"
	MockAuthHeaderValue = "mockAuth"
)

type MockDecorator struct {
	mock.Mock
}

func (m *MockDecorator) Decorate(ctx context.Context, req *http.Request) error {
	req.Header.Set(MockAuthHeaderName, MockAuthHeaderValue)

	return m.Called().Error(0)
}
