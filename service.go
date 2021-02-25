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

var (
	errMigrationOwnerEmpty         = errors.New("owner is required when migration section is provided in config")
	errNonSuccessPushResult        = errors.New("got a push result but was not of success type")
	errFailedWebhookPush           = errors.New("failed to add webhook to registry")
	errFailedWebhookConversion     = errors.New("failed to convert webhook to argus item")
	errFailedItemConversion        = errors.New("failed to convert argus item to webhook")
	errFailedMigratedWebhooksFetch = errors.New("failed to fetch migrated webhooks")
	errFailedWebhooksFetch         = errors.New("failed to fetch webhooks")
)

// Service describes the core operations around webhook subscriptions.
// Initialize() provides a service ready to use and the controls around watching for updates.
type Service interface {
	// Add adds the given owned webhook to the current list of webhooks. If the operation
	// succeeds, a non-nil error is returned.
	Add(owner string, w Webhook) error

	// AllWebhooks lists all the current webhooks for the given owner.
	// If an owner is not provided, all webhooks are returned.
	AllWebhooks(owner string) ([]Webhook, error)
}

// MigrationConfig contains fields to capture webhooks items migrated
// from SNS to Argus.
type MigrationConfig struct {
	// Bucket from which to fetch the webhook items.
	// (Optional). Defaults to 'webhooks'
	Bucket string

	// Owner of the items.
	Owner string
}

// Config contains information needed to initialize the webhook service.
type Config struct {
	// Argus contains configuration to initialize an Argus client.
	Argus chrysom.ClientConfig

	// Bucket is the name of the Argus partition in which the webhook items
	// will be stored.
	// (Optional). Defaults to 'webhooks'
	Bucket string

	// Migration provides info for capturing webhook items that
	// were recently migrated from SNS to Argus. This should
	// match the migration configuration Hecate uses.
	// (Optional)
	// If this config section is left blank, migrated items won't be
	// captured.
	// if the section is specified but owner is not provided, a validation
	// error should be given.
	Migration *MigrationConfig

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
		return fmt.Errorf("%w: %v", errFailedWebhookConversion, err)
	}
	result, err := s.argus.PushItem(item.ID, s.config.Bucket, owner, item)
	if err != nil {
		return fmt.Errorf("%w: %v", errFailedWebhookPush, err)
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return fmt.Errorf("%w: %s", errNonSuccessPushResult, result)
}

// AllWebhooks returns the set of all webhooks associated with the given owner.
// Note: While webhooks stored through Argus have this item to owner relationship
// information, those stored through SNS do not have that piece of information.
// This opens the possibility of not capturing those items that were just migrated
// from SNS to Argus. For this reason, when migration configuration is provided,
// AllWebhooks provides a set with data from both the migrated list from SNS and the
// existing Argus webhooks.
func (s *service) AllWebhooks(owner string) ([]Webhook, error) {
	webhookSet := make(map[string]Webhook)
	if s.config.Migration != nil {
		err := s.captureMigratedWebhooks(webhookSet)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errFailedMigratedWebhooksFetch, err)
		}
	}

	items, err := s.argus.GetItems(s.config.Bucket, owner)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errFailedWebhooksFetch, err)
	}

	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errFailedItemConversion, err)
		}
		webhookSet[item.ID] = webhook
	}

	return toSlice(webhookSet), nil
}

func (s *service) captureMigratedWebhooks(webhookSet map[string]Webhook) error {
	items, err := s.argus.GetItems(s.config.Migration.Bucket, s.config.Migration.Owner)
	if err != nil {
		return err
	}

	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			return err
		}
		webhookSet[item.ID] = webhook
	}

	return nil
}

func toSlice(webhookSet map[string]Webhook) []Webhook {
	webhooks := []Webhook{}

	for _, webhook := range webhookSet {
		webhooks = append(webhooks, webhook)
	}

	return webhooks
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

func validateConfig(cfg *Config) error {
	if cfg.Bucket == "" {
		cfg.Bucket = "webhooks"
	}

	if cfg.Migration != nil {
		if cfg.Migration.Owner == "" {
			return errMigrationOwnerEmpty
		}

		if cfg.Migration.Bucket == "" {
			cfg.Migration.Bucket = "webhooks"
		}
	}

	if cfg.Logger == nil {
		cfg.Logger = log.NewNopLogger()
	}

	if cfg.MetricsProvider == nil {
		cfg.MetricsProvider = provider.NewDiscardProvider()
	}

	return nil
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
