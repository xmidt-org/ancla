// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushResultString(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("", NilPushResult.String())
	assert.Equal("created", CreatedPushResult.String())
	assert.Equal("ok", UpdatedPushResult.String())
	assert.Equal("unknown", UnknownPushResult.String())
	assert.Equal("unknown", PushResult(42).String())
}
