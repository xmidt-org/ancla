package ancla

import (
	"context"

	"github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
)

type mockPushReader struct {
	mock.Mock
}

func (m *mockPushReader) GetItems(ctx context.Context, owner string) (chrysom.Items, error) {
	args := m.Called(ctx, owner)
	return args.Get(0).(chrysom.Items), args.Error(1)
}

func (m *mockPushReader) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPushReader) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPushReader) PushItem(ctx context.Context, owner string, item model.Item) (chrysom.PushResult, error) {
	args := m.Called(ctx, owner, item)
	return args.Get(0).(chrysom.PushResult), args.Error(1)
}

func (m *mockPushReader) RemoveItem(ctx context.Context, id, owner string) (model.Item, error) {
	args := m.Called(ctx, id, owner)
	return args.Get(0).(model.Item), args.Error(0)
}

type mockService struct {
	mock.Mock
}

func (m *mockService) Add(ctx context.Context, owner string, w Webhook) error {
	args := m.Called(ctx, owner, w)
	return args.Error(0)
}

func (m *mockService) AllWebhooks(ctx context.Context) ([]Webhook, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Webhook), args.Error(1)
}

type mockCounter struct {
	mock.Mock
}

func (m *mockCounter) With(labelValues ...string) metrics.Counter {
	m.Called(interfacify(labelValues)...)
	return m
}

func (m *mockCounter) Add(delta float64) {
	m.Called(delta)
}

type mockGauge struct {
	mock.Mock
}

func (m *mockGauge) With(labelValues ...string) metrics.Gauge {
	m.Called(interfacify(labelValues)...)
	return m
}

func (m *mockGauge) Set(value float64) {
	m.Called(value)
}

func (m *mockGauge) Add(delta float64) {
	m.Called(delta)
}

func interfacify(vals []string) []interface{} {
	transformed := make([]interface{}, len(vals))
	for i, val := range vals {
		transformed[i] = val
	}
	return transformed
}
