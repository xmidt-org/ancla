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
