// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
)

// Names
const (
	WebhookListSizeGaugeName = "webhook_list_size"
	WebhookListSizeGaugeHelp = "Size of the current list of webhooks."
	PollsTotalCounterName    = "chrysom_polls_total"
	PollsTotalCounterHelp    = "Counter for the number of polls (and their success/failure outcomes) to fetch new items."
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Label Values
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)

func ProvideMetrics() fx.Option {
	return fx.Options(
		touchstone.Gauge(
			prometheus.GaugeOpts{
				Name: WebhookListSizeGaugeName,
				Help: WebhookListSizeGaugeHelp,
			}),
		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: PollsTotalCounterName,
				Help: PollsTotalCounterHelp,
			},
			OutcomeLabel,
		),
	)
}
