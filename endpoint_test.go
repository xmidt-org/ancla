// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

// import (
// 	"context"
// 	"errors"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestNewAddWebhookEndpoint(t *testing.T) {
// 	assert := assert.New(t)
// 	m := new(mockService)
// 	endpoint := newAddWebhookEndpoint(m)
// 	input := &addWebhookRequest{
// 		owner:          "owner-val",
// 		internalWebook: InternalWebhook{},
// 	}

// 	errFake := errors.New("failed")
// 	// nolint:typecheck
// 	m.On("Add", context.Background(), "owner-val", input.internalWebook).Return(errFake)
// 	resp, err := endpoint(context.Background(), input)
// 	assert.Nil(resp)
// 	assert.Equal(errFake, err)
// 	// nolint:typecheck
// 	m.AssertExpectations(t)
// }

// func TestGetAllWebhooksEndpoint(t *testing.T) {
// 	assert := assert.New(t)
// 	m := new(mockService)
// 	endpoint := newGetAllWebhooksEndpoint(m)

// 	respFake := []InternalWebhook{}
// 	// nolint:typecheck
// 	m.On("GetAll", context.Background()).Return(respFake, nil)
// 	resp, err := endpoint(context.Background(), nil)
// 	assert.Nil(err)
// 	assert.Equal(respFake, resp)
// 	// nolint:typecheck
// 	m.AssertExpectations(t)
// }
