// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package chrysom

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type MockAquirer struct {
	mock.Mock
}

func (m *MockAquirer) AddAuth(req *http.Request) error {
	args := m.Called(req)

	return args.Error(0)
}
