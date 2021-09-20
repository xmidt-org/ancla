package ancla

import (
	"context"
	"errors"

	"github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
)

var (
	errReadBodyFail      = errors.New("read test error")
	errMockValidatorFail = errors.New("validation error")
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

func (m *mockService) Add(ctx context.Context, owner string, iw InternalWebhook) error {
	args := m.Called(ctx, owner, iw)
	return args.Error(0)
}

func (m *mockService) GetAll(ctx context.Context) ([]InternalWebhook, error) {
	args := m.Called(ctx)
	return args.Get(0).([]InternalWebhook), args.Error(1)
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

type errReader struct {
}

func (e errReader) Read(p []byte) (n int, err error) {
	return 0, errReadBodyFail
}

func (e errReader) Close() error {
	return errors.New("close test error")
}

func mockValidator() ValidatorFunc {
	return func(w Webhook) error {
		return errMockValidatorFail
	}
}
