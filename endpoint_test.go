// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/ancla/schema"
)

func TestNewAddWRPEventStreamEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newAddWRPEventStreamEndpoint(m)
	input := &addWRPEventStreamRequest{
		owner:          "owner-val",
		internalWebook: &schema.ManifestV1{},
	}

	errFake := errors.New("failed")
	// nolint:typecheck
	m.On("Add", context.Background(), "owner-val", input.internalWebook).Return(errFake)
	resp, err := endpoint(context.Background(), input)
	assert.Nil(resp)
	assert.Equal(errFake, err)
	// nolint:typecheck
	m.AssertExpectations(t)
}

func TestGetAllWRPEventStreamsEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newGetAllWRPEventStreamsEndpoint(m)

	respFake := []schema.Manifest{}
	// nolint:typecheck
	m.On("GetAll", context.Background()).Return(respFake, nil)
	resp, err := endpoint(context.Background(), nil)
	assert.Nil(err)
	assert.Equal(respFake, resp)
	// nolint:typecheck
	m.AssertExpectations(t)
}
