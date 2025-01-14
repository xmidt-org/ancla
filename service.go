// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xmidt-org/ancla/chrysom"
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
	Add(ctx context.Context, owner string, iw InternalWebhook) error

	// GetAll lists all the current registered webhooks.
	GetAll(ctx context.Context) ([]InternalWebhook, error)
}

// Config contains information needed to initialize the Argus Client service.
type Config struct {
	BasicClientConfig chrysom.BasicClientConfig

	// Logger for this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op logger.
	Logger *zap.Logger

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

// ListenerConfig contains information needed to initialize the Listener Client service.
type ListenerConfig struct {
	Config chrysom.ListenerClientConfig

	// Logger for this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op logger.
	Logger *zap.Logger

	// Measures for instrumenting this package.
	// Gets passed to Argus config before initializing the client.
	Measures Measures
}

type service struct {
	argus  chrysom.PushReader
	logger *zap.Logger
	config Config
	now    func() time.Time
}

// NewService builds the Argus client service from the given configuration.
func NewService(cfg Config, getLogger func(context.Context) *zap.Logger) (*service, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	basic, err := chrysom.NewBasicClient(cfg.BasicClientConfig, getLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create chrysom basic client: %v", err)
	}
	svc := &service{
		logger: cfg.Logger,
		argus:  basic,
		config: cfg,
		now:    time.Now,
	}
	return svc, nil
}

// StartListener builds the Argus listener client service from the given configuration.
// It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func (s *service) StartListener(cfg ListenerConfig, setLogger func(context.Context, *zap.Logger) context.Context, watches ...Watch) (func(), error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	prepArgusListenerClientConfig(&cfg, watches...)
	m := &chrysom.Measures{
		Polls: cfg.Measures.ChrysomPollsTotalCounterName,
	}
	listener, err := chrysom.NewListenerClient(cfg.Config, setLogger, m, s.argus)
	if err != nil {
		return nil, fmt.Errorf("failed to create chrysom listener client: %v", err)
	}

	listener.Start(context.Background())
	return func() { listener.Stop(context.Background()) }, nil
}

func (s *service) Add(ctx context.Context, owner string, iw InternalWebhook) error {
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
func (s *service) GetAll(ctx context.Context) ([]InternalWebhook, error) {
	items, err := s.argus.GetItems(ctx, "")
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWebhooksFetch, err)
	}

	iws := make([]InternalWebhook, len(items))

	for i, item := range items {
		webhook, err := ItemToInternalWebhook(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		iws[i] = webhook
	}

	return iws, nil
}

func prepArgusListenerClientConfig(cfg *ListenerConfig, watches ...Watch) {
	logger := cfg.Logger
	watches = append(watches, webhookListSizeWatch(cfg.Measures.WebhookListSizeGaugeName))
	cfg.Config.Listener = chrysom.ListenerFunc(func(items chrysom.Items) {
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
