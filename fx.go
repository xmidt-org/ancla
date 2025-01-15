// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"github.com/xmidt-org/ancla/chrysom"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

type ServiceIn struct {
	fx.In

	// Ancla Client.
	BasicClient *chrysom.BasicClient
}

// ProvideService builds the Argus client service from the given configuration.
func ProvideService(in ServiceIn) Service {
	return NewService(in.BasicClient)
}

// TODO: Refactor and move Watch and Listener related code to chrysom.
type DefaultListenersIn struct {
	fx.In

	// Metric for webhook list size, used by the webhook list size watcher listener.
	WebhookListSizeGauge prometheus.Gauge `name:"webhook_list_size"`
}

type DefaultListenerOut struct {
	fx.Out

	Watchers []Watch `group:"watchers,flatten"`
}

func ProvideDefaultListenerWatchers(in DefaultListenersIn) DefaultListenerOut {
	var watchers []Watch

	watchers = append(watchers, webhookListSizeWatch(in.WebhookListSizeGauge))

	return DefaultListenerOut{
		Watchers: watchers,
	}
}

type ListenerIn struct {
	fx.In

	Shutdowner fx.Shutdowner
	// Watchers are called by the Listener when new webhooks are fetched.
	Watchers []Watch `group:"watchers"`
}

func ProvideListener(in ListenerIn) chrysom.Listener {
	return chrysom.ListenerFunc(func(items chrysom.Items) {
		iws, err := ItemsToInternalWebhooks(items)
		if err != nil {
			in.Shutdowner.Shutdown(fx.ExitCode(1))

			return
		}

		for _, watch := range in.Watchers {
			watch.Update(iws)
		}
	})
}
