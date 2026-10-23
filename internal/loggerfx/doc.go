// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

/*
Package loggerfx provides a function that can be used to create a logger module
for an fx application.  The logger module is based on the sallust, goschtalt,
and zap libraries and provides a simple, tested way to include a standard logger.

Example Usage:

	package main

	import (
		"github.com/goschtalt/goschtalt"
		"github.com/xmidt-org/sallust"
		"github.com/xmidt-org/skeleton/internal/loggerfx"
		"go.uber.org/fx"
		"go.uber.org/zap"
	)

	func main() {
		cfg := struct {
			Logging sallust.Config
		}{
			Logging: sallust.Config{
				Level:            "info",
				OutputPaths:      []string{},
				ErrorOutputPaths: []string{},
			},
		}

		config, err := goschtalt.New(
			goschtalt.ConfigIs("two_words"),
			goschtalt.AddValue("built-in", goschtalt.Root, cfg, goschtalt.AsDefault()),
		)
		if err != nil {
			panic(err)
		}

		app := fx.New(
			fx.Provide(
				func() *goschtalt.Config {
					return config
				}),

			loggerfx.Module(),
			fx.Invoke(func(logger *zap.Logger) {
				logger.Info("Hello, world!")
			}),
			loggerfx.SyncOnShutdown(),
		)

		app.Run()
	}
*/
package loggerfx
