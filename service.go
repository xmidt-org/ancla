// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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
	errFailedConfig            = errors.New("ancla configuration error")
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
	argus  chrysom.PushReader
	config Config
	now    func() time.Time
}

// NewService builds the Argus client service from the given configuration.
func NewService(cfg Config) (*ClientService, error) {
	basic, err := chrysom.NewBasicClient(cfg.BasicClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create chrysom basic client: %v", err)
	}
	svc := &ClientService{
		argus:  basic,
		config: cfg,
		now:    time.Now,
	}
	return svc, nil
}

// StartListener builds the Argus listener client service from the given configuration.
// It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func (s *ClientService) StartListener(cfg chrysom.ListenerConfig, metrics chrysom.Measures, watches ...Watch) (func(), error) {
	prepArgusListenerConfig(&cfg, metrics, watches...)
	listener, err := chrysom.NewListenerClient(cfg, metrics, s.argus)
	if err != nil {
		return nil, fmt.Errorf("failed to create chrysom listener client: %v", err)
	}

	listener.Start(context.Background())
	return func() { listener.Stop(context.Background()) }, nil
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

func prepArgusListenerConfig(cfg *chrysom.ListenerConfig, metrics chrysom.Measures, watches ...Watch) {
	watches = append(watches, webhookListSizeWatch(metrics.WebhookListSizeGauge))
	cfg.Listener = chrysom.ListenerFunc(func(ctx context.Context, items chrysom.Items) {
		logger := sallust.Get(ctx)
		iws, err := ItemsToInternalWebhooks(items)
		if err != nil {
			logger.Error("Failed to convert items to webhooks", zap.Error(err))
			return
		}
		for _, watch := range watches {
			watch.Update(iws)
		}
	})
}

type ServiceIn struct {
	fx.In

	Config Config
	Client *http.Client
	Auth   chrysom.Acquirer
}

func ProvideService() fx.Option {
	return fx.Provide(
		func(in ServiceIn) (*ClientService, error) {
			svc, err := NewService(in.Config)
			if err != nil {
				return nil, errors.Join(errFailedConfig, err)
			}

			svc.config.BasicClientConfig.HTTPClient = in.Client
			return svc, err
		},
	)
}

type ListenerIn struct {
	fx.In

	Measures       chrysom.Measures
	Svc            *ClientService
	listenerConfig chrysom.ListenerConfig
	Watcher        Watch
	LC             fx.Lifecycle
}

func ProvideListener() fx.Option {
	return fx.Options(
		fx.Provide(
			func(in ListenerIn) (err error) {
				stopWatches, err := in.Svc.StartListener(in.listenerConfig, in.Measures, in.Watcher)
				if err != nil {
					return fmt.Errorf("webhook service start listener error: %v", err)
				}
				in.LC.Append(fx.StopHook(stopWatches))

				return nil
			},
		),
	)
}
