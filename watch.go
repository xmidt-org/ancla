// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/ancla/schema"
)

// Watch is the interface for listening for wrpEventStream subcription updates.
// Updates represent the latest known list of subscriptions.
type Watch interface {
	Update([]schema.Manifest)
}

// WatchFunc allows bare functions to pass as Watches.
type WatchFunc func([]schema.Manifest)

func (f WatchFunc) Update(update []schema.Manifest) {
	f(update)
}

func wrpEventStreamListSizeWatch(s prometheus.Gauge) Watch {
	return WatchFunc(func(streams []schema.Manifest) {
		s.Set(float64(len(streams)))
	})
}
