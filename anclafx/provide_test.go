// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx_test

import (
	"context"
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

	Factory           *touchstone.Factory
	BasicClientConfig chrysom.BasicClientConfig
	GetLogger         chrysom.GetLogger
	SetLogger         chrysom.SetLogger
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
		BasicClientConfig: chrysom.BasicClientConfig{
			Address: "example.com",
			Bucket:  "bucket-name",
		},
		GetLogger: func(context.Context) *zap.Logger { return zap.NewNop() },
		SetLogger: func(context.Context, *zap.Logger) context.Context { return context.Background() },
	}, nil
}

func TestProvide(t *testing.T) {
	t.Run("Test anclafx.Provide() defaults", func(t *testing.T) {
		var (
			svc ancla.Service
			bc  *chrysom.BasicClient
			l   *chrysom.ListenerClient
		)

		app := fxtest.New(t,
			anclafx.Provide(),
			fx.Provide(
				provideDefaults,
			),
			fx.Populate(
				&svc,
				&bc,
				&l,
			),
		)

		require := require.New(t)
		require.NotNil(app)
		require.NoError(app.Err())
		app.RequireStart()
		require.NotNil(svc)
		require.NotNil(bc)
		require.NotNil(l)
		app.RequireStop()
	})
}
