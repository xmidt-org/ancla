// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

// Acquirer acquires the credential for http request authorization headers.
type Acquirer interface {
	// Acquire gets a credential string.
	Acquire() (string, error)
}
