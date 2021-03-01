package ancla

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/provider"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/themis/xlog"
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
	Add(owner string, w Webhook) error

	// AllWebhooks lists all the current registered webhooks.
	AllWebhooks() ([]Webhook, error)
}

// Config contains information needed to initialize the webhook service.
type Config struct {
	// Argus contains configuration to initialize an Argus client.
	Argus chrysom.ClientConfig

	// Bucket is the name of the Argus partition in which the webhook items
	// will be stored.
	// (Optional). Defaults to 'webhooks'
	Bucket string

	// Logger for this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op logger.
	Logger log.Logger

	// MetricsProvider for instrumenting this package.
	// Gets passed to Argus config before initializing the client.
	// (Optional). Defaults to a no op provider.
	MetricsProvider provider.Provider
}

type service struct {
	argus  chrysom.PushReader
	logger log.Logger
	config Config
}

func (s *service) Add(owner string, w Webhook) error {
	item, err := webhookToItem(w)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWebhookConversion, err)
	}
	result, err := s.argus.PushItem(item.ID, s.config.Bucket, owner, item)
	if err != nil {
		return fmt.Errorf(errFmt, errFailedWebhookPush, err)
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return fmt.Errorf("%w: %s", errNonSuccessPushResult, result)
}

// AllWebhooks returns all webhooks found on the configured webhooks partition
// of Argus.
func (s *service) AllWebhooks() ([]Webhook, error) {
	items, err := s.argus.GetItems(s.config.Bucket, "")
	if err != nil {
		return nil, fmt.Errorf(errFmt, errFailedWebhooksFetch, err)
	}

	webhooks := []Webhook{}

	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			return nil, fmt.Errorf(errFmt, errFailedItemConversion, err)
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

func webhookToItem(w Webhook) (model.Item, error) {
	encodedWebhook, err := json.Marshal(w)
	if err != nil {
		return model.Item{}, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(encodedWebhook, &data)
	if err != nil {
		return model.Item{}, err
	}

	TTLSeconds := int64(w.Duration.Seconds())

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(w.Config.URL)))

	return model.Item{
		Data: data,
		ID:   checksum,
		TTL:  &TTLSeconds,
	}, nil
}

func itemToWebhook(i model.Item) (Webhook, error) {
	encodedWebhook, err := json.Marshal(i.Data)
	if err != nil {
		return Webhook{}, err
	}
	var w Webhook
	err = json.Unmarshal(encodedWebhook, &w)
	if err != nil {
		return Webhook{}, err
	}
	return w, nil
}

func validateConfig(cfg *Config) {
	if cfg.Bucket == "" {
		cfg.Bucket = "webhooks"
	}

	if cfg.Logger == nil {
		cfg.Logger = log.NewNopLogger()
	}

	if cfg.MetricsProvider == nil {
		cfg.MetricsProvider = provider.NewDiscardProvider()
	}
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func Initialize(cfg Config, watches ...Watch) (Service, func(), error) {
	validateConfig(&cfg)
	watches = append(watches, webhookListSizeWatch(cfg.MetricsProvider.NewGauge(WebhookListSizeGauge)))

	cfg.Argus.Logger = cfg.Logger
	cfg.Argus.MetricsProvider = cfg.MetricsProvider
	cfg.Argus.Listener = createArgusListener(cfg.Logger, watches...)

	argus, err := chrysom.NewClient(cfg.Argus)
	if err != nil {
		return nil, nil, err
	}

	svc := &service{
		logger: cfg.Logger,
		argus:  argus,
		config: cfg,
	}

	argus.Start(context.Background())

	return svc, func() { argus.Stop(context.Background()) }, nil
}

func createArgusListener(logger log.Logger, watches ...Watch) chrysom.Listener {
	return chrysom.ListenerFunc(func(items chrysom.Items) {
		webhooks, err := itemsToWebhooks(items)
		if err != nil {
			level.Error(logger).Log(xlog.MessageKey(), "Failed to convert items to webhooks", "err", err)
			return
		}
		for _, watch := range watches {
			watch.Update(webhooks)
		}
	})
}

func itemsToWebhooks(items []model.Item) ([]Webhook, error) {
	webhooks := []Webhook{}
	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, webhook)
	}
	return webhooks, nil
}
