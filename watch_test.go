// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebhookListSizeWatch(t *testing.T) {
	require := require.New(t)
	gauge := new(mockGauge)
	watch := webhookListSizeWatch(gauge)
	require.NotNil(watch)
	// nolint:typecheck
	gauge.On("Set", float64(2))
	watch.Update([]InternalWebhook{{}, {}})
	// nolint:typecheck
	gauge.AssertExpectations(t)
}
