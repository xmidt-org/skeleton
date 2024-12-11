// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"go.uber.org/fx"
)

type AuthIn struct {
	fx.In
	Config Config
}

type AuthOut struct {
	fx.Out
	Auth *Auth
}

var Module = fx.Module("auth",
	fx.Provide(
		func(in AuthIn) (AuthOut, error) {
			auth, err := New(
				WithConfig(in.Config),
			)

			return AuthOut{
				Auth: auth}, err
		},
	),
)
