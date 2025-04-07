// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/schema"
)

func TestWRPEventStreamListSizeWatch(t *testing.T) {
	require := require.New(t)
	gauge := new(mockGauge)
	watch := wrpEventStreamListSizeWatch(gauge)
	require.NotNil(watch)
	// nolint:typecheck
	gauge.On("Set", float64(2))
	watch.Update([]schema.RegistryManifest{&schema.RegistryV1{}, &schema.RegistryV2{}})
	// nolint:typecheck
	gauge.AssertExpectations(t)
}
