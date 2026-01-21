// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package loggerfx

import (
	"errors"
	"fmt"

	"github.com/goschtalt/goschtalt"
	"github.com/xmidt-org/sallust"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	deaultConfigPath = "logging"
)

var (
	ErrInvalidConfigPath = errors.New("configuration structure path like 'logging' or 'foo.logging' is required")
)

// Module is function that builds the loggerfx module based on the inputs.  If
// the configPath is not provided then the default path is used.
func Module(configPath ...string) fx.Option {
	configPath = append(configPath, deaultConfigPath)

	var path string
	for _, cp := range configPath {
		if cp != "" {
			path = cp
			break
		}
	}

	// Why not use sallust.WithLogger()?  It is because we want to provide the
	// developer configuration based on the sallust configuration, not the zap
	// options.  This makes the configuration consistent between the two modes.
	return fx.Options(
		// Provide the logger configuration based on the input path.
		fx.Provide(
			goschtalt.UnmarshalFunc[sallust.Config](path),
			provideLogger,
		),

		// Inform fx that we are providing a logger for it.
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	)
}

// DefaultConfig is a helper function that creates a default sallust configuration
// for the logger based on the application name.
func DefaultConfig(appName string) sallust.Config {
	return sallust.Config{
		// Use the default zap logger configuration for most of the settings.
		OutputPaths: []string{
			fmt.Sprintf("/var/log/%s/%s.log", appName, appName),
		},
		ErrorOutputPaths: []string{
			fmt.Sprintf("/var/log/%s/%s.log", appName, appName),
		},
		Rotation: &sallust.Rotation{
			MaxSize:    50,
			MaxBackups: 10,
			MaxAge:     2,
		},
	}
}

// SyncOnShutdown is a helper function that returns an fx option that will
// sync the logger on shutdown.
//
// Make sure to include this option last in the fx.Options list, so that the
// logger is the last component to be shutdown.
func SyncOnShutdown() fx.Option {
	return sallust.SyncOnShutdown()
}

// LoggerIn describes the dependencies used to bootstrap a zap logger within
// an fx application.
type LoggerIn struct {
	fx.In

	// Config is the sallust configuration for the logger.  This component is optional,
	// and if not supplied a default zap logger will be created.
	Config sallust.Config `optional:"true"`

	// DevMode is a flag that indicates if the logger should be in debug mode.
	DevMode bool `name:"loggerfx.dev_mode" optional:"true"`
}

// Create the logger and configure it based on if the program is in
// debug mode or normal mode.
func provideLogger(in LoggerIn) (*zap.Logger, error) {
	if in.DevMode {
		in.Config.Level = "DEBUG"
		in.Config.Development = true
		in.Config.Encoding = "console"
		in.Config.EncoderConfig = sallust.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    "capitalColor",
			EncodeTime:     "RFC3339",
			EncodeDuration: "string",
			EncodeCaller:   "short",
		}
		in.Config.OutputPaths = []string{"stderr"}
		in.Config.ErrorOutputPaths = []string{"stderr"}
	}
	return in.Config.Build()
}
