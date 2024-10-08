// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"github.com/xmidt-org/ancla/chrysom"
	"go.uber.org/fx"
)

// ProvideMetrics provides the metrics relevant to this package as uber/fx options.
func ProvideMetrics() fx.Option {
	return fx.Options(
		chrysom.ProvideMetrics(),
	)
}
