/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/v2/logging"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

const errFmt = "%w: %v"

var (
	errNilProvider             = errors.New("provider cannot be nil")
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

	// AllWebhooks lists all the current registered webhooks.
	AllInternalWebhooks(ctx context.Context) ([]InternalWebhook, error)
}

// Config contains information needed to initialize the webhook service.
type Config struct {
	// Argus contains configuration to initialize an Argus client.
	Argus chrysom.ClientConfig

	// Logger for this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op logger.
	Logger log.Logger

	// MetricsProvider for instrumenting this package.
	// Gets passed to Argus config before initializing the client.
	MetricsProvider xmetrics.Registry

	// JWTParserType establishes which parser type will be used by the JWT token
	// acquirer used by Argus. Options include 'simple' and 'raw'.
	// Simple: parser assumes token payloads have the following structure: https://github.com/xmidt-org/bascule/blob/c011b128d6b95fa8358228535c63d1945347adaa/acquire/bearer.go#L77
	// Raw: parser assumes all of the token payload == JWT token
	// (Optional). Defaults to 'simple'
	JWTParserType jwtAcquireParserType
}

type service struct {
	argus  chrysom.PushReader
	logger log.Logger
	config Config
	now    func() time.Time
}

type InternalWebhook struct {
	PartnerIDs []string
	Webhook    Webhook
}

func (s *service) Add(ctx context.Context, owner string, iw InternalWebhook) error {
	item, err := internalWebhookToItem(s.now, iw)
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

// AllInternalWebhooks returns all webhooks found on the configured webhooks partition
// of Argus.
func (s *service) AllInternalWebhooks(ctx context.Context) ([]InternalWebhook, error) {
	items, err := s.argus.GetItems(ctx, "")
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWebhooksFetch, err)
	}

	iws := make([]InternalWebhook, len(items))

	for i, item := range items {
		webhook, err := itemToInternalWebhook(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		iws[i] = webhook
	}

	return iws, nil
}

func internalWebhookToItem(now func() time.Time, iw InternalWebhook) (model.Item, error) {
	encodedWebhook, err := json.Marshal(iw)
	if err != nil {
		return model.Item{}, err
	}
	var data map[string]interface{}

	err = json.Unmarshal(encodedWebhook, &data)
	if err != nil {
		return model.Item{}, err
	}

	SecondsToExpiry := iw.Webhook.Until.Sub(now()).Seconds()
	TTLSeconds := int64(math.Max(0, SecondsToExpiry))

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(iw.Webhook.Config.URL)))

	return model.Item{
		Data: data,
		ID:   checksum,
		TTL:  &TTLSeconds,
	}, nil
}

func itemToInternalWebhook(i model.Item) (InternalWebhook, error) {
	encodedWebhook, err := json.Marshal(i.Data)
	if err != nil {
		return InternalWebhook{}, err
	}
	var iw InternalWebhook
	err = json.Unmarshal(encodedWebhook, &iw)
	if err != nil {
		return InternalWebhook{}, err
	}
	return iw, nil
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

	if cfg.MetricsProvider == nil {
		return nil, nil, errNilProvider
	}
	m := &chrysom.Measures{
		Polls: cfg.MetricsProvider.NewCounterVec(chrysom.PollCounter),
	}
	argus, err := chrysom.NewClient(cfg.Argus, m, getLogger, setLogger)
	if err != nil {
		return nil, nil, err
	}

	svc := &service{
		logger: cfg.Logger,
		argus:  argus,
		config: cfg,
		now:    time.Now,
	}

	argus.Start(context.Background())

	return svc, func() { argus.Stop(context.Background()) }, nil
}

func prepArgusConfig(cfg *Config, watches ...Watch) error {
	watches = append(watches, webhookListSizeWatch(cfg.MetricsProvider.NewGauge(WebhookListSizeGauge)))
	cfg.Argus.Logger = cfg.Logger
	cfg.Argus.Listen.Listener = createArgusListener(cfg.Logger, watches...)
	p, err := newJWTAcquireParser(cfg.JWTParserType)
	if err != nil {
		return err
	}
	cfg.Argus.Auth.JWT.GetToken = p.token
	cfg.Argus.Auth.JWT.GetExpiration = p.expiration
	return nil
}

func createArgusListener(logger log.Logger, watches ...Watch) chrysom.Listener {
	return chrysom.ListenerFunc(func(items chrysom.Items) {
		iws, err := itemsToInternalWebhooks(items)
		if err != nil {
			level.Error(logger).Log(logging.MessageKey(), "Failed to convert items to webhooks", "err", err)
			return
		}
		ws := internalWebhooksToWebhooks(iws)
		for _, watch := range watches {
			watch.Update(ws)
		}
	})
}

func itemsToInternalWebhooks(items []model.Item) ([]InternalWebhook, error) {
	iws := []InternalWebhook{}
	for _, item := range items {
		iw, err := itemToInternalWebhook(item)
		if err != nil {
			return nil, err
		}
		iws = append(iws, iw)
	}
	return iws, nil
}

func internalWebhooksToWebhooks(iws []InternalWebhook) []Webhook {
	w := make([]Webhook, 0, len(iws))
	for _, iw := range iws {
		w = append(w, iw.Webhook)
	}
	return w
}
