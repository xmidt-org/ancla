/**
 * Copyright 2022 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package ancla

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/v2/logging"
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

// Config contains information needed to initialize the webhook service.
type Config struct {
	BasicClientConfig chrysom.BasicClientConfig `mapstructure:",squash"`

	ListenerClientConfig chrysom.ListenerClientConfig

	// Logger for this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op logger.
	Logger log.Logger

	// Measures for instrumenting this package.
	// Gets passed to Argus config before initializing the client.
	Measures Measures

	// JWTParserType establishes which parser type will be used by the JWT token
	// acquirer used by Argus. Options include 'simple' and 'raw'.
	// Simple: parser assumes token payloads have the following structure: https://github.com/xmidt-org/bascule/blob/c011b128d6b95fa8358228535c63d1945347adaa/acquire/bearer.go#L77
	// Raw: parser assumes all of the token payload == JWT token
	// (Optional). Defaults to 'simple'
	JWTParserType jwtAcquireParserType

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

type service struct {
	argus  chrysom.PushReader
	logger log.Logger
	config Config
	now    func() time.Time
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

func validateConfig(cfg *Config) {
	if cfg.Logger == nil {
		cfg.Logger = log.NewNopLogger()
	}
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func Initialize(cfg Config, getLogger func(ctx context.Context) log.Logger, setLogger func(context.Context, log.Logger) context.Context, watches ...Watch) (Service, func(), error) {
	validateConfig(&cfg)
	prepArgusConfig(&cfg, watches...)

	m := &chrysom.Measures{
		Polls: cfg.Measures.ChrysomPollsTotalCounter,
	}
	basic, err := chrysom.NewBasicClient(cfg.BasicClientConfig, getLogger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chrysom basic client: %v", err)
	}
	listener, err := chrysom.NewListenerClient(cfg.ListenerClientConfig, setLogger, m, basic)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chrysom listener client: %v", err)
	}

	svc := &service{
		logger: cfg.Logger,
		argus:  basic,
		config: cfg,
		now:    time.Now,
	}

	listener.Start(context.Background())

	return svc, func() { listener.Stop(context.Background()) }, nil
}

func prepArgusConfig(cfg *Config, watches ...Watch) error {
	watches = append(watches, webhookListSizeWatch(cfg.Measures.WebhookListSizeGauge))
	cfg.BasicClientConfig.Logger = cfg.Logger
	cfg.ListenerClientConfig.Listener = createArgusListener(cfg.Logger, watches...)
	p, err := newJWTAcquireParser(cfg.JWTParserType)
	if err != nil {
		return err
	}
	cfg.BasicClientConfig.Auth.JWT.GetToken = p.token
	cfg.BasicClientConfig.Auth.JWT.GetExpiration = p.expiration
	return nil
}

func createArgusListener(logger log.Logger, watches ...Watch) chrysom.Listener {
	return chrysom.ListenerFunc(func(items chrysom.Items) {
		iws, err := ItemsToInternalWebhooks(items)
		if err != nil {
			level.Error(logger).Log(logging.MessageKey(), "Failed to convert items to webhooks", "err", err)
			return
		}
		for _, watch := range watches {
			watch.Update(iws)
		}
	})
}
