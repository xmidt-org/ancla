// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package anclafx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/ancla"
	"github.com/xmidt-org/ancla/anclafx"
	"github.com/xmidt-org/ancla/chrysom"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type out struct {
	fx.Out

	Factory           *touchstone.Factory
	BasicClientConfig chrysom.BasicClientConfig
	Options           chrysom.Options
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
		Factory: touchstone.NewFactory(cfg, sallust.Default(), pr),
		BasicClientConfig: chrysom.BasicClientConfig{
			Address: "example.com",
			Bucket:  "bucket-name",
		},
		Options: chrysom.Options{
			chrysom.Address("example.com"),
			chrysom.Bucket("bucket-name"),
		},
	}, nil
}

func TestProvide(t *testing.T) {
	t.Run("Test anclafx.Provide() defaults", func(t *testing.T) {
		var (
			svc *ancla.ClientService
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
