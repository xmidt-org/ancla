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
	errFake := errors.New("failed")
	input := &addWRPEventStreamRequest{
		owner:    "owner-val",
		manifest: &schema.ManifestV1{},
	}

	var gotOwner string
	var gotManifest schema.Manifest
	endpoint := newAddWRPEventStreamEndpoint(func(_ context.Context, owner string, m schema.Manifest) error {
		gotOwner, gotManifest = owner, m
		return errFake
	})

	resp, err := endpoint(context.Background(), input)
	assert.Nil(resp)
	assert.Equal(errFake, err)
	assert.Equal("owner-val", gotOwner)
	assert.Equal(input.manifest, gotManifest)
}

func TestGetAllWRPEventStreamsEndpoint(t *testing.T) {
	assert := assert.New(t)
	respFake := []schema.Manifest{}
	endpoint := newGetAllWRPEventStreamsEndpoint(func(context.Context) ([]schema.Manifest, error) {
		return respFake, nil
	})

	resp, err := endpoint(context.Background(), nil)
	assert.Nil(err)
	assert.Equal(respFake, resp)
}
