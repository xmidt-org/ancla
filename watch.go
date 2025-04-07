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
	Update([]schema.RegistryManifest)
}

// WatchFunc allows bare functions to pass as Watches.
type WatchFunc func([]schema.RegistryManifest)

func (f WatchFunc) Update(update []schema.RegistryManifest) {
	f(update)
}

func wrpEventStreamListSizeWatch(s prometheus.Gauge) Watch {
	return WatchFunc(func(wrpEventStreams []schema.RegistryManifest) {
		s.Set(float64(len(wrpEventStreams)))
	})
}
