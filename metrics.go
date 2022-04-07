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
	"github.com/go-kit/kit/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchkit"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
	"go.uber.org/fx"
)

// Names
const (
	WebhookListSizeGauge     = "webhook_list_size_value"
	ChrysomPollsTotalCounter = chrysom.PollCounter
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

// Metrics returns the Metrics relevant to this package.
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name: WebhookListSizeGauge,
			Type: xmetrics.GaugeType,
			Help: "Size of the current list of webhooks.",
		},
		{
			Name:       ChrysomPollsTotalCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			LabelNames: []string{OutcomeLabel},
		},
	}
}

// Measures describes the defined metrics that will be used by clients.
type Measures struct {
	WebhookListSizeGauge     metrics.Gauge
	ChrysomPollsTotalCounter *prometheus.CounterVec
}

// MeasuresIn is an uber/fx parameter with the webhook registration counter.
type MeasuresIn struct {
	fx.In
	WebhookListSizeGauge     metrics.Gauge          `name:"webhook_list_size"`
	ChrysomPollsTotalCounter *prometheus.CounterVec `name:"chrysom_polls_total"`
}

// NewMeasures realizes desired metrics.
func NewMeasures(p xmetrics.Registry) *Measures {
	return &Measures{
		WebhookListSizeGauge:     p.NewGauge(WebhookListSizeGauge),
		ChrysomPollsTotalCounter: p.NewCounterVec(ChrysomPollsTotalCounter),
	}
}

// ProvideMetrics provides the metrics relevant to this package as uber/fx options.
func ProvideMetrics() fx.Option {
	return fx.Options(
		touchkit.Gauge(
			prometheus.GaugeOpts{
				Name: WebhookListSizeGauge,
				Help: "Size of the current list of webhooks.",
			},
		),
		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: ChrysomPollsTotalCounter,
				Help: "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			}, OutcomeLabel,
		),
		fx.Provide(
			func(in MeasuresIn) *Measures {
				return &Measures{
					WebhookListSizeGauge:     in.WebhookListSizeGauge,
					ChrysomPollsTotalCounter: in.ChrysomPollsTotalCounter,
				}
			},
		),
	)
}
