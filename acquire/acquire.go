// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package acquire

import (
	"net/http"
	"time"
)

type Acquirer interface {
	AddAuth(*http.Request) error
	Acquire() (string, error)
	ParseToken([]byte) (string, error)
	ParseExpiration([]byte) (time.Time, error)
}
