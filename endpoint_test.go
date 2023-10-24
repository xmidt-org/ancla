/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ancla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAddWebhookEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newAddWebhookEndpoint(m)
	input := &addWebhookRequest{
		owner:          "owner-val",
		internalWebook: InternalWebhook{},
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

func TestGetAllWebhooksEndpoint(t *testing.T) {
	assert := assert.New(t)
	m := new(mockService)
	endpoint := newGetAllWebhooksEndpoint(m)

	respFake := []InternalWebhook{}
	// nolint:typecheck
	m.On("GetAll", context.Background()).Return(respFake, nil)
	resp, err := endpoint(context.Background(), nil)
	assert.Nil(err)
	assert.Equal(respFake, resp)
	// nolint:typecheck
	m.AssertExpectations(t)
}
