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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// Names
const (
	WebhookListSizeGaugeName     = "webhook_list_size"
	WebhookListSizeGaugeHelp     = "Size of the current list of webhooks."
	ChrysomPollsTotalCounterName = chrysom.PollCounter
	ChrysomPollsTotalCounterHelp = "Counter for the number of polls (and their success/failure outcomes) to fetch new items."
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Outcomes
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)

// Measures describes the defined metrics that will be used by clients.
type Measures struct {
	WebhookListSizeGaugeName     prometheus.Gauge       `name:"webhook_list_size"`
	ChrysomPollsTotalCounterName *prometheus.CounterVec `name:"chrysom_polls_total"`
}

type MeasuresOut struct {
	fx.Out

	M *Measures
}

// MeasuresIn is an uber/fx parameter with the webhook registration counter.
type MeasuresIn struct {
	fx.In

	Factory *touchstone.Factory `optional:"true"`
}

// NewMeasures realizes desired metrics.
func NewMeasures(in MeasuresIn) (MeasuresOut, error) {
	var metricErr error
	wlm, err := in.Factory.NewGauge(
		prometheus.GaugeOpts{
			Name: WebhookListSizeGaugeName,
			Help: WebhookListSizeGaugeHelp,
		},
	)
	err = multierr.Append(err, metricErr)
	cpm, err := in.Factory.NewCounterVec(
		prometheus.CounterOpts{
			Name: ChrysomPollsTotalCounterName,
			Help: ChrysomPollsTotalCounterHelp,
		},
		OutcomeLabel,
	)

	return MeasuresOut{
		M: &Measures{
			WebhookListSizeGaugeName:     wlm,
			ChrysomPollsTotalCounterName: cpm,
		},
	}, multierr.Append(err, metricErr)
}

// ProvideMetrics provides the metrics relevant to this package as uber/fx options.
func ProvideMetrics() fx.Option {
	return fx.Options(
		fx.Provide(NewMeasures),
	)
}
