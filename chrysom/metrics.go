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

// Measures describes the defined metrics that will be used by clients.
type Measures struct {
	WebhookListSizeGauge prometheus.Gauge       `name:"webhook_list_size"`
	PollsTotalCounter    *prometheus.CounterVec `name:"chrysom_polls_total"`
}

// MeasuresIn is an uber/fx parameter with the webhook registration counter.
type MeasuresIn struct {
	fx.In
	WebhookListSizeGauge prometheus.Gauge       `name:"webhook_list_size"`
	PollsTotalCounter    *prometheus.CounterVec `name:"chrysom_polls_total"`
}

// NewMeasures realizes desired metrics.
func NewMeasures(in MeasuresIn) *Measures {
	return &Measures{
		WebhookListSizeGauge: in.WebhookListSizeGauge,
		PollsTotalCounter:    in.PollsTotalCounter,
	}
}

// Metrics returns the Metrics relevant to this package
func ProvideMetrics() fx.Option {
	return fx.Options(
		fx.Provide(NewMeasures),
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
