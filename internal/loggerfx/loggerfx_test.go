// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package loggerfx

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/goschtalt/goschtalt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

/*
func TestModule(t *testing.T) {
	app := fxtest.New(
		t,
		Module(),
		fx.Invoke(func(logger *zap.Logger) {
			assert.NotNil(t, logger)
		}),
	)
	defer app.RequireStart().RequireStop()
}
*/

func TestDefaultConfig(t *testing.T) {
	appName := "testapp"
	config := DefaultConfig(appName)

	expectedConfig := sallust.Config{
		OutputPaths: []string{
			"/var/log/testapp/testapp.log",
		},
		ErrorOutputPaths: []string{
			"/var/log/testapp/testapp.log",
		},
		Rotation: &sallust.Rotation{
			MaxSize:    50,
			MaxBackups: 10,
			MaxAge:     2,
		},
	}

	assert.Equal(t, expectedConfig, config)
}

func TestLoggerFX_EndToEnd(t *testing.T) {
	// Create a temporary configuration file
	loggerConfig := struct {
		Logger sallust.Config
	}{
		Logger: sallust.Config{
			Level:            "info",
			OutputPaths:      []string{},
			ErrorOutputPaths: []string{},
		},
	}
	loggingConfig := struct {
		Logging sallust.Config
	}{
		Logging: sallust.Config{
			Level:            "info",
			OutputPaths:      []string{},
			ErrorOutputPaths: []string{},
		},
	}

	tests := []struct {
		name  string
		input any
		label string
		dev   bool
		err   bool
	}{
		{
			name:  "empty path, no error",
			input: loggingConfig,
		}, {
			name:  "specific path, no error",
			input: loggerConfig,
			label: "logger",
		}, {
			name:  "empty path, no error, dev mode",
			input: loggingConfig,
			dev:   true,
		}, {
			name:  "specific path, no error, dev mode",
			input: loggerConfig,
			label: "logger",
			dev:   true,
		}, {
			name:  "specific path, error",
			input: loggerConfig,
			label: "invalid",
			err:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Initialize goschtalt with the configuration file
			config, err := goschtalt.New(
				goschtalt.ConfigIs("two_words"),
				goschtalt.AddValue("built-in", goschtalt.Root, test.input, goschtalt.AsDefault()),
			)
			require.NoError(t, err)

			opt := Module()
			if test.label != "" {
				opt = Module(test.label)
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			var stderr bytes.Buffer
			done := make(chan struct{})
			go func() {
				io.Copy(&stderr, r)
				close(done)
			}()

			opts := []fx.Option{
				fx.Supply(config),
				fx.Supply(fx.Annotate(test.dev, fx.ResultTags(`name:"loggerfx.dev_mode"`))),
				opt,
				fx.Invoke(func(logger *zap.Logger) {
					assert.NotNil(t, logger)
					logger.Info("End-to-end test log message")
				}),
				SyncOnShutdown(),
			}

			if test.err {
				app := fx.New(opts...)
				require.NotNil(t, app)
				assert.Error(t, app.Err())
			} else {
				app := fxtest.New(t, opts...)
				require.NotNil(t, app)
				assert.NoError(t, app.Err())
				app.RequireStart().RequireStop()
			}

			// Close the writer and restore stderr
			w.Close()
			os.Stderr = oldStderr
			<-done
		})
	}
}
