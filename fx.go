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

	// PushReader is the user provided db client.
	// (Optional)
	// If provided, Ancla will not use default Argus db client.
	PushReader chrysom.PushReader    `optional:"true"`
	Options    chrysom.ClientOptions `group:"client_options"`
}

type ProvideServiceOut struct {
	fx.Out

	// Ancla service.
	Service Service
	// Ancla listener's db client.
	Reader chrysom.Reader
}

// ProvideService provides the Argus client service from the given configuration and client options.
func ProvideService(in ServiceIn) (ProvideServiceOut, error) {
	// If the user provides a non-nil chrysom.PushReader (their own db client), then use that instead
	// of an Argus db client.
	// Otherwise, create and use a new Argus db client.
	if in.PushReader != nil {
		return ProvideServiceOut{
			Service: NewService(in.PushReader),
			Reader:  in.PushReader,
		}, nil
	}

	client, err := chrysom.NewBasicClient(in.Options)
	if err != nil {
		return ProvideServiceOut{}, err
	}

	return ProvideServiceOut{
		Service: NewService(in.PushReader),
		Reader:  client,
	}, nil
}

// TODO: Refactor and move Watch and ListenerInterface related code to chrysom.
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

// ListenerOut contains options data for Listener client's reader.
type ListenerOut struct {
	fx.Out

	Option chrysom.ListenerOption `group:"listener_options"`
}

func ProvideListener(in ListenerIn) ListenerOut {
	return ListenerOut{
		Option: chrysom.Listener(chrysom.ListenerFunc(func(items chrysom.Items) {
			iws, err := ItemsToInternalWebhooks(items)
			if err != nil {
				in.Shutdowner.Shutdown(fx.ExitCode(1))

				return
			}

			for _, watch := range in.Watchers {
				watch.Update(iws)
			}
		})),
	}
}
