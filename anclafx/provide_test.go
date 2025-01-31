// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla"
	"github.com/xmidt-org/ancla/anclafx"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type out struct {
	fx.Out

	Factory         *touchstone.Factory
	ClientOptions   chrysom.ClientOptions   `group:"client_options,flatten"`
	ListenerOptions chrysom.ListenerOptions `group:"listener_options,flatten"`
}

func provideDefaults() (out, error) {
	cfg := touchstone.Config{
		DefaultNamespace: "n",
		DefaultSubsystem: "s",
	}
	_, pr, err := touchstone.New(cfg)
	if err != nil {
		return out{}, err
	}

	return out{
		Factory: touchstone.NewFactory(cfg, zap.NewNop(), pr),
		ClientOptions: chrysom.ClientOptions{
			chrysom.Bucket("bucket-name"),
		},
		// Listener has no required options
		ListenerOptions: chrysom.ListenerOptions{},
	}, nil
}

func TestProvide(t *testing.T) {
	t.Run("Test anclafx.Provide() defaults", func(t *testing.T) {
		var (
			svc      ancla.Service
			reader   chrysom.Reader
			listener *chrysom.ListenerClient
		)

		app := fxtest.New(t,
			anclafx.Provide(),
			fx.Provide(
				provideDefaults,
			),
			fx.Populate(
				&svc,
				&reader,
				&listener,
			),
		)

		require := require.New(t)
		require.NotNil(app)
		require.NoError(app.Err())
		app.RequireStart()
		require.NotNil(svc)
		require.NotNil(reader)
		require.NotNil(listener)
		app.RequireStop()
	})
}
