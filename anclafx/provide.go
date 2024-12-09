// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/ancla"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/touchstone"
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
		touchstone.Gauge(
			prometheus.GaugeOpts{
				Name: chrysom.WebhookListSizeGaugeName,
				Help: chrysom.WebhookListSizeGaugeHelp,
			}),
		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: chrysom.PollsTotalCounterName,
				Help: chrysom.PollsTotalCounterHelp,
			},
			chrysom.OutcomeLabel,
		),
	)
}
