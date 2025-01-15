// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// GetLogger returns a logger from the given context.
type GetLogger func(context.Context) *zap.Logger

// SetLogger embeds the `Listener.logger` in outgoing request contexts for `Listener.Update` calls.
type SetLogger func(context.Context, *zap.Logger) context.Context

type BasicClientIn struct {
	fx.In

	// Ancla Client config.
	Config BasicClientConfig
	// GetLogger returns a logger from the given context.
	GetLogger GetLogger
}

// ProvideBasicClient provides a new BasicClient.
func ProvideBasicClient(in BasicClientIn) (*BasicClient, error) {
	client, err := NewBasicClient(in.Config, in.GetLogger)
	if err != nil {
		return nil, errors.Join(errFailedConfig, err)
	}

	return client, nil
}

// ListenerConfig contains config data for polling the Argus client.
type ListenerClientIn struct {
	fx.In

	// Listener fetches a copy of all items within a bucket on
	// an interval based on `BasicClientConfig.PullInterval`.
	// (Optional). If not provided, listening won't be enabled for this client.
	Listener Listener
	// Config configures the ancla client and its listeners.
	Config BasicClientConfig
	// PollsTotalCounter measures the number of polls (and their success/failure outcomes) to fetch new items.
	PollsTotalCounter *prometheus.CounterVec `name:"chrysom_polls_total"`
	// Reader is the DB interface used to fetch new items using `GeItems`.
	Reader Reader
	// GetLogger returns a logger from the given context.
	GetLogger GetLogger
	// SetLogger embeds the `Listener.logger` in outgoing request contexts for `Listener.Update` calls.
	SetLogger SetLogger
}

// ProvideListenerClient provides a new ListenerClient.
func ProvideListenerClient(in ListenerClientIn) (*ListenerClient, error) {
	client, err := NewListenerClient(in.Listener, in.GetLogger, in.SetLogger, in.Config.PullInterval, in.PollsTotalCounter, in.Reader)
	if err != nil {
		return nil, errors.Join(err, errFailedConfig)
	}

	return client, nil
}

func ProvideDefaultListenerReader(client *BasicClient) Reader {
	return client
}

type StartListenerIn struct {
	fx.In

	Listener *ListenerClient
	LC       fx.Lifecycle
}

// ProvideStartListenerClient starts the Argus listener client service.
func ProvideStartListenerClient(in StartListenerIn) error {
	in.Listener.Start(context.Background())
	in.LC.Append(fx.StopHook(in.Listener.Stop))

	return nil
}

func ProvideMetrics() fx.Option {
	return fx.Options(
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
