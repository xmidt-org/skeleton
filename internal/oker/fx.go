// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package oker

import (
	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type telemetryIn struct {
	fx.In

	Logger   *zap.Logger
	Counter  kit.Counter   `name:"oking_request_count"`
	Duration kit.Histogram `name:"oking_call_duration"`
}

var Module = fx.Module("oker",
	fx.Provide(
		func(in telemetryIn) *telemetry {
			return &telemetry{
				logger:   in.Logger,
				counter:  in.Counter,
				duration: in.Duration,
			}
		}),
	fx.Provide(
		func(cfg Config, t *telemetry) (*Server, error) {
			a, err := New(
				WithConfig(cfg),
				AddOkEventListener(t),
			)

			return a, err
		},
	),
)
