// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package chrysom

import (
	"github.com/stretchr/testify/mock"
)

type MockAquirer struct {
	mock.Mock
}

func (m *MockAquirer) GetAuth() (string, error) {
	args := m.Called()

	return args.String(0), args.Error(1)
}
