// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
