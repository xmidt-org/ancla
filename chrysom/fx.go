// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
)

var (
	ErrMisconfiguredListener = errors.New("ancla listener configuration error")
)

// ListenerConfig contains config data for polling the Argus client.
type ListenerClientIn struct {
	fx.In

	// PollsTotalCounter measures the number of polls (and their success/failure outcomes) to fetch new items.
	PollsTotalCounter *prometheus.CounterVec `name:"chrysom_polls_total"`
	Options           ListenerOptions        `group:"listener_options"`
}

// ProvideListenerClient provides a new ListenerClient.
func ProvideListenerClient(in ListenerClientIn) (*ListenerClient, error) {
	client, err := NewListenerClient(in.PollsTotalCounter, in.Options)
	if err != nil {
		return nil, errors.Join(err, ErrMisconfiguredListener)
	}

	return client, nil
}

// ReaderOptionOut contains options data for Listener client's reader.
type ReaderOptionOut struct {
	fx.Out

	Option ListenerOption `group:"listener_options"`
}

// ProvideReaderOption provides Listener client's read option.
// ancla.ProvideService determines which Reader is used, i.e.:
// the default Argus db client or a user provided db client.
func ProvideReaderOption(r Reader) (ReaderOptionOut, error) {
	return ReaderOptionOut{
		Option: reader(r),
	}, nil
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
