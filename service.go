// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/sallust"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const errFmt = "%w: %v"

var (
	errNonSuccessPushResult    = errors.New("got a push result but was not of success type")
	errFailedWebhookPush       = errors.New("failed to add webhook to registry")
	errFailedWebhookConversion = errors.New("failed to convert webhook to argus item")
	errFailedItemConversion    = errors.New("failed to convert argus item to webhook")
	errFailedWebhooksFetch     = errors.New("failed to fetch webhooks")
)

// Service describes the core operations around webhook subscriptions.
// Initialize() provides a service ready to use and the controls around watching for updates.
type Service interface {
	// Add adds the given owned webhook to the current list of webhooks. If the operation
	// succeeds, a non-nil error is returned.
	Add(ctx context.Context, owner string, iw Register) error

	// GetAll lists all the current registered webhooks.
	GetAll(ctx context.Context) ([]Register, error)
}

// Config contains information needed to initialize the Argus Client service.
type Config struct {
	BasicClientConfig chrysom.BasicClientConfig

	// DisablePartnerIDs, if true, will allow webhooks to register without
	// checking the validity of the partnerIDs in the request
	DisablePartnerIDs bool

	// Validation provides options for validating the webhook's URL and TTL
	// related fields. Some validation happens regardless of the configuration:
	// URLs must be a valid URL structure, the Matcher.DeviceID values must
	// compile into regular expressions, and the Events field must have at
	// least one value and all values must compile into regular expressions.
	Validation ValidatorConfig
}

type ClientService struct {
	argus chrysom.PushReader
	now   func() time.Time
}

func (s *ClientService) Add(ctx context.Context, owner string, iw Register) error {
	item, err := InternalWebhookToItem(s.now, iw)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWebhookConversion, err)
	}
	result, err := s.argus.PushItem(ctx, owner, item)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWebhookPush, err)
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return fmt.Errorf("%w: %s", errNonSuccessPushResult, result)
}

// GetAll returns all webhooks found on the configured webhooks partition
// of Argus.
func (s *ClientService) GetAll(ctx context.Context) ([]Register, error) {
	items, err := s.argus.GetItems(ctx, "")
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWebhooksFetch, err)
	}

	iws := make([]Register, len(items))

	for i, item := range items {
		webhook, err := ItemToInternalWebhook(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		iws[i] = webhook
	}

	return iws, nil
}

func NewService(client *chrysom.BasicClient) *ClientService {
	return &ClientService{
		argus: client,
		now:   time.Now,
	}
}

type ClientServiceIn struct {
	fx.In

	BasicClient *chrysom.BasicClient
}

// ProvideService builds the Argus client service from the given configuration.
func ProvideService(in ClientServiceIn) *ClientService {
	return NewService(in.BasicClient)
}

// TODO: Refactor and move Watch and Listener related code to chrysom.
type DefaultListenersIn struct {
	fx.In

	WebhookListSizeGauge prometheus.Gauge `name:"webhook_list_size"`
}

type DefaultListenerOut struct {
	fx.Out

	Watchers []Watch `group:"watchers,flatten"`
}

func ProvideDefaultListeners(in DefaultListenersIn) DefaultListenerOut {
	var watchers []Watch

	watchers = append(watchers, webhookListSizeWatch(in.WebhookListSizeGauge))

	return DefaultListenerOut{
		Watchers: watchers,
	}
}

type ListenerIn struct {
	fx.In

	Shutdowner fx.Shutdowner
	Watchers   []Watch `group:"watchers"`
}

func ProvideListener(in ListenerIn) chrysom.Listener {
	return chrysom.ListenerFunc(func(ctx context.Context, items chrysom.Items) {
		logger := sallust.Get(ctx)
		iws, err := ItemsToInternalWebhooks(items)
		if err != nil {
			logger.Error("failed to convert items to webhooks", zap.Error(err))
			in.Shutdowner.Shutdown(fx.ExitCode(1))

			return
		}

		for _, watch := range in.Watchers {
			watch.Update(iws)
		}
	})

}
