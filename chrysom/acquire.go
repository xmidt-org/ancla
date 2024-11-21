// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"net/http"
)

// Acquirer adds an authorization header and value to a given http request.
type Acquirer interface {
	AddAuth(*http.Request) error
}
