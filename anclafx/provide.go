// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx

import (
	"github.com/xmidt-org/ancla"
	"github.com/xmidt-org/ancla/chrysom"
	"go.uber.org/fx"
)

const Module = "ancla"

func Provide() fx.Option {
	return fx.Module(
		Module,
		fx.Invoke(chrysom.ProvideStartListenerClient),
		fx.Provide(
			ancla.ProvideListener,
			ancla.ProvideService,
			chrysom.ProvideBasicClient,
			chrysom.ProvideDefaultListenerReader,
			chrysom.ProvideListenerClient,
		),
		chrysom.ProvideMetrics(),
	)
}
