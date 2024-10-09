// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx

import (
	"github.com/xmidt-org/ancla"
	"go.uber.org/fx"
)

const Module = "ancla"

func Provide() fx.Option {
	return fx.Module(
		Module,
		ancla.ProvideMetrics(),
		ancla.ProvideListener(),
		ancla.ProvideService(),
	)
}
