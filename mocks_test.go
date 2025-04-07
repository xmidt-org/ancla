// SPDX-FileCopyrightText: 2022 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/ancla/schema"
	"github.com/xmidt-org/webhook-schema"
)

var (
	errReadBodyFail      = errors.New("read test error")
	errMockValidatorFail = errors.New("validation error")
)

type mockPushReader struct {
	mock.Mock
}

func (m *mockPushReader) GetItems(ctx context.Context, owner string) (chrysom.Items, error) {
	// nolint:typecheck
	args := m.Called(ctx, owner)
	return args.Get(0).(chrysom.Items), args.Error(1)
}

func (m *mockPushReader) Start(ctx context.Context) error {
	// nolint:typecheck
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPushReader) Stop(ctx context.Context) error {
	// nolint:typecheck
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockPushReader) PushItem(ctx context.Context, owner string, item model.Item) (chrysom.PushResult, error) {
	// nolint:typecheck
	args := m.Called(ctx, owner, item)
	return args.Get(0).(chrysom.PushResult), args.Error(1)
}

func (m *mockPushReader) RemoveItem(ctx context.Context, id, owner string) (model.Item, error) {
	// nolint:typecheck
	args := m.Called(ctx, id, owner)
	return args.Get(0).(model.Item), args.Error(0)
}

type mockService struct {
	mock.Mock
}

func (m *mockService) Add(ctx context.Context, owner string, iw schema.RegistryManifest) error {
	// nolint:typecheck
	args := m.Called(ctx, owner, iw)
	return args.Error(0)
}

func (m *mockService) GetAll(ctx context.Context) ([]schema.RegistryManifest, error) {
	// nolint:typecheck
	args := m.Called(ctx)
	return args.Get(0).([]schema.RegistryManifest), args.Error(1)
}

type mockCounter struct {
	mock.Mock
}

func (m *mockCounter) With(labelValues ...string) prometheus.Counter {
	// nolint:typecheck
	m.Called(interfacify(labelValues)...)
	return m
}

func (m *mockCounter) Add(delta float64) {
	// nolint:typecheck
	m.Called(delta)
}

func (m *mockCounter) Inc() {
	// nolint:typecheck
	m.Called()
}

func (m *mockCounter) Write(out *dto.Metric) error {
	// nolint:typecheck
	m.Called()

	return nil
}

func (m *mockCounter) Desc() *prometheus.Desc {
	// nolint:typecheck
	m.Called()

	return &prometheus.Desc{}
}

func (m *mockCounter) Collect(ch chan<- prometheus.Metric) {
	// nolint:typecheck
	m.Called()
}

func (m *mockCounter) Describe(ch chan<- *prometheus.Desc) {
	// nolint:typecheck
	m.Called()
}

type mockGauge struct {
	mock.Mock
}

func (m *mockGauge) With(labelValues ...string) prometheus.Gauge {
	// nolint:typecheck
	m.Called(interfacify(labelValues)...)
	return m
}

func (m *mockGauge) Set(value float64) {
	// nolint:typecheck
	m.Called(value)
}

func (m *mockGauge) Add(delta float64) {
	// nolint:typecheck
	m.Called(delta)
}

func (m *mockGauge) Sub(value float64) {
	// nolint:typecheck
	m.Called(value)
}

func (m *mockGauge) SetToCurrentTime() {
	// nolint:typecheck
	m.Called()
}

func (m *mockGauge) Inc() {
	// nolint:typecheck
	m.Called()
}

func (m *mockGauge) Dec() {
	// nolint:typecheck
	m.Called()
}

func (m *mockGauge) Write(out *dto.Metric) error {
	// nolint:typecheck
	m.Called()

	return nil
}

func (m *mockGauge) Desc() *prometheus.Desc {
	// nolint:typecheck
	m.Called()

	return &prometheus.Desc{}
}

func (m *mockGauge) Collect(ch chan<- prometheus.Metric) {
	// nolint:typecheck
	m.Called()
}

func (m *mockGauge) Describe(ch chan<- *prometheus.Desc) {
	// nolint:typecheck
	m.Called()
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

func mockValidator() (opts []webhook.Option) {
	opts = append(opts, mockOption{})
	return
}

type mockOption struct{}

func (mockOption) Validate(mock any) error {
	return errMockValidatorFail
}

func (mockOption) String() string {
	return "mockOption"
}
