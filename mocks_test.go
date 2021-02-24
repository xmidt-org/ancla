package ancla

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
)

type mockPushReader struct {
	mock.Mock
}

func (m *mockPushReader) GetItems(bucket, owner string) (chrysom.Items, error) {
	args := m.Called(bucket, owner)
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

func (m *mockPushReader) PushItem(id, bucket, owner string, item model.Item) (chrysom.PushResult, error) {
	args := m.Called(id, bucket, owner, item)
	return args.Get(0).(chrysom.PushResult), args.Error(1)
}

func (m *mockPushReader) RemoveItem(id, bucket string, owner string) (model.Item, error) {
	args := m.Called(id, bucket, owner)
	return args.Get(0).(model.Item), args.Error(0)
}
