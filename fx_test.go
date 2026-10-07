// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package ancla

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/ancla/model"
	"github.com/xmidt-org/ancla/schema"
)

func TestProvideService(t *testing.T) {
	require := require.New(t)
	svc, err := ProvideService(ServiceIn{PushReader: new(mockPushReader)})
	require.NoError(err)
	require.NotNil(svc)
}

func TestProvideDefaultListenerWatchers(t *testing.T) {
	require := require.New(t)
	gauge := new(mockGauge)
	out := ProvideDefaultListenerWatchers(DefaultListenersIn{WRPEventStreamListSizeGauge: gauge})
	require.Len(out.Watchers, 1)

	// nolint:typecheck
	gauge.On("Set", float64(1))
	require.NoError(out.Watchers[0].Update([]schema.Manifest{&schema.ManifestV1{}}))
	// nolint:typecheck
	gauge.AssertExpectations(t)
}

func TestProvideListener(t *testing.T) {
	require := require.New(t)
	out := ProvideListener(ListenerIn{})
	require.NotNil(out.Option)
}

func TestNewListener(t *testing.T) {
	errWatch := errors.New("watch failed")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tcs := []struct {
		desc        string
		ctx         context.Context
		items       chrysom.Items
		watchers    []Watch
		expectedErr error
		expectCalls int
	}{
		{
			desc:        "canceled context",
			ctx:         canceled,
			expectedErr: context.Canceled,
		},
		{
			desc: "bad item",
			ctx:  context.Background(),
			items: chrysom.Items{
				model.Item{Data: map[string]any{"bad": make(chan int)}},
			},
		},
		{
			desc:  "watcher failure",
			ctx:   context.Background(),
			items: getTestItems(),
			watchers: []Watch{
				WatchFunc(func([]schema.Manifest) error { return errWatch }),
				WatchFunc(func([]schema.Manifest) error { return nil }),
			},
			expectedErr: errWatch,
			expectCalls: 2,
		},
		{
			desc:  testSuccess,
			ctx:   context.Background(),
			items: getTestItems(),
			watchers: []Watch{
				WatchFunc(func([]schema.Manifest) error { return nil }),
			},
			expectCalls: 1,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)
			calls := 0
			watchers := make([]Watch, 0, len(tc.watchers))
			for _, w := range tc.watchers {
				w := w
				watchers = append(watchers, WatchFunc(func(m []schema.Manifest) error {
					calls++
					assert.Len(m, len(tc.items))
					return w.Update(m)
				}))
			}

			err := newListener(watchers)(tc.ctx, tc.items)
			assert.Equal(tc.expectCalls, calls)
			switch {
			case tc.expectedErr != nil:
				assert.ErrorIs(err, tc.expectedErr)
			case tc.desc == "bad item":
				assert.Error(err)
			default:
				assert.NoError(err)
			}
		})
	}
}
