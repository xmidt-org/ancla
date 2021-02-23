package ancla

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
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

// Config contains information needed to initialize the webhook service.
type Config struct {
	// Argus contains configuration to initialize an Argus client.
	Argus chrysom.ClientConfig

	// Bucket is the name of the Argus partition in which the webhook items
	// will be stored.
	// (Optional). Defaults to 'webhooks'
	Bucket string
}

type loggerGroup struct {
	Error log.Logger
	Debug log.Logger
}

type service struct {
	argus   *chrysom.Client
	loggers *loggerGroup
	config  Config
}

func (s *service) Add(owner string, w Webhook) error {
	item, err := webhookToItem(w)
	if err != nil {
		return err
	}
	result, err := s.argus.PushItem(item.ID, s.config.Bucket, owner, item)
	if err != nil {
		return err
	}

	if result == chrysom.CreatedPushResult || result == chrysom.UpdatedPushResult {
		return nil
	}
	return errors.New("operation to add webhook to db failed")
}

func (s *service) AllWebhooks(owner string) ([]Webhook, error) {
	s.loggers.Debug.Log("msg", "AllWebhooks called", "owner", owner)
	items, err := s.argus.GetItems(s.config.Bucket, owner)
	if err != nil {
		return nil, err
	}
	webhooks := []Webhook{}
	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			continue
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

func newLoggerGroup(root log.Logger) *loggerGroup {
	if root == nil {
		root = log.NewNopLogger()
	}

	return &loggerGroup{
		Debug: log.WithPrefix(root, level.Key(), level.DebugValue()),
		Error: log.WithPrefix(root, level.Key(), level.ErrorValue()),
	}

}
func validateConfig(cfg *Config) {
	if len(strings.TrimSpace(cfg.Bucket)) == 0 {
		cfg.Bucket = "webhooks"
	}
}

// Initialize builds the webhook service from the given configuration. It allows adding watchers for the internal subscription state. Call the returned
// function when you are done watching for updates.
func Initialize(cfg Config, watches ...Watch) (Service, func(), error) {
	validateConfig(&cfg)
	watches = append(watches, webhookListSizeWatch(cfg.Argus.MetricsProvider.NewGauge(WebhookListSizeGauge)))

	cfg.Argus.Listener = createArgusListener(watches...)

	argus, err := chrysom.NewClient(cfg.Argus)
	if err != nil {
		return nil, nil, err
	}

	svc := &service{
		loggers: newLoggerGroup(cfg.Argus.Logger),
		argus:   argus,
	}

	argus.Start(context.Background())

	return svc, func() { argus.Stop(context.Background()) }, nil
}

func createArgusListener(watches ...Watch) chrysom.Listener {
	if len(watches) < 1 {
		return nil
	}
	return chrysom.ListenerFunc(func(items chrysom.Items) {
		webhooks := itemsToWebhooks(items)
		for _, watch := range watches {
			watch.Update(webhooks)
		}
	})
}

func itemsToWebhooks(items []model.Item) []Webhook {
	webhooks := []Webhook{}
	for _, item := range items {
		webhook, err := itemToWebhook(item)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, webhook)
	}
	return webhooks
}
