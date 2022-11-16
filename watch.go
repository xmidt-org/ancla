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
)

// Watch is the interface for listening for webhook subcription updates.
// Updates represent the latest known list of subscriptions.
type Watch interface {
	Update([]InternalWebhook)
}

// WatchFunc allows bare functions to pass as Watches.
type WatchFunc func([]InternalWebhook)

func (f WatchFunc) Update(update []InternalWebhook) {
	f(update)
}

func webhookListSizeWatch(s prometheus.Gauge) Watch {
	return WatchFunc(func(webhooks []InternalWebhook) {
		s.Set(float64(len(webhooks)))
	})
}
