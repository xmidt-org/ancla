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
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// Names
const (
	WebhookListSizeGauge     = "webhook_list_size_value"
	WebhookLegacyDecodeCount = "webhook_legacy_decodings_total"
)

// Labels
const (
	OutcomeLabel = "outcome"
	URLLabel     = "url"
)

// Label Values
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)

// Metrics returns the Metrics relevant to this package.
func Metrics() []xmetrics.Metric {
	metrics := []xmetrics.Metric{
		{
			Name: WebhookListSizeGauge,
			Type: xmetrics.GaugeType,
			Help: "Size of the current list of webhooks.",
		},
		{
			Name:       WebhookLegacyDecodeCount,
			Type:       xmetrics.CounterType,
			Help:       "Number of times a webhook is registered with a legacy decoding strategy.",
			LabelNames: []string{URLLabel},
		},
		{
			Name:       chrysom.PollCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			LabelNames: []string{chrysom.OutcomeLabel},
		},
	}
	return metrics
}
