// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrincipal(t *testing.T) {
	t.Run("Test SetPartnerIDs, GetPartnerIDs", func(t *testing.T) {
		assert := assert.New(t)
		partnerIDs := []string{"foo", "bar"}
		ctx := SetPartnerIDs(context.Background(), partnerIDs)
		actualPartnerIDs, ok := GetPartnerIDs(ctx)
		assert.True(ok)
		assert.Equal(partnerIDs, actualPartnerIDs)
		actualPartnerIDs, ok = GetPartnerIDs(context.Background())
		assert.False(ok)
		var empty []string
		assert.Equal(empty, actualPartnerIDs)
	})
	t.Run("Test SetPrincipal, GetPrincipal", func(t *testing.T) {
		assert := assert.New(t)
		principal := "foo"
		ctx := SetPrincipal(context.Background(), principal)
		actualPrincipal, ok := GetPrincipal(ctx)
		assert.True(ok)
		assert.Equal(principal, actualPrincipal)
		actualPrincipal, ok = GetPrincipal(context.Background())
		assert.False(ok)
		assert.Equal("", actualPrincipal)
	})
}
