// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package skeleton

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/goschtalt/goschtalt"
	_ "github.com/goschtalt/goschtalt/pkg/typical"
	_ "github.com/goschtalt/yaml-decoder"
	_ "github.com/goschtalt/yaml-encoder"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/candlelight"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/skeleton/internal/apiauth"
	"github.com/xmidt-org/skeleton/internal/metrics"
	"github.com/xmidt-org/skeleton/internal/oker"
	"github.com/xmidt-org/touchstone"
	"github.com/xmidt-org/touchstone/touchhttp"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

const (
	applicationNamespace = "xmidt"
	applicationName      = "skeleton"
)

// These match what goreleaser provides.
var (
	commit  = "undefined"
	version = "undefined"
	date    = "undefined"
	builtBy = "undefined"
)

// CLI is the structure that is used to capture the command line arguments.
type CLI struct {
	Dev   bool     `optional:"" short:"d" help:"Run in development mode."`
	Show  bool     `optional:"" short:"s" help:"Show the configuration and exit."`
	Graph string   `optional:"" short:"g" help:"Output the dependency graph to the specified file."`
	Files []string `optional:"" short:"f" help:"Specific configuration files or directories."`
}

func Main(args []string, run bool) error { // nolint: funlen
	var (
		gscfg *goschtalt.Config

		// Capture the dependency tree in case we need to debug something.
		g fx.DotGraph

		// Capture the command line arguments.
		cli *CLI
	)

	app := fx.New(
		fx.Supply(cliArgs(args)),
		fx.Populate(&g),
		fx.Populate(&gscfg),
		fx.Populate(&cli),

		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		fx.Provide(
			provideCLI,
			provideLogger,
			provideConfig,
			goschtalt.UnmarshalFunc[sallust.Config]("logging"),
			goschtalt.UnmarshalFunc[candlelight.Config]("tracing"),
			goschtalt.UnmarshalFunc[touchstone.Config]("prometheus"),
			goschtalt.UnmarshalFunc[touchhttp.Config]("prometheus_handler"),
			goschtalt.UnmarshalFunc[HealthPath]("servers.health.path", goschtalt.Optional()),
			goschtalt.UnmarshalFunc[MetricsPath]("servers.metrics.path", goschtalt.Optional()),
			goschtalt.UnmarshalFunc[PprofPathPrefix]("servers.pprof.path", goschtalt.Optional()),
			goschtalt.UnmarshalFunc[Routes]("routes"),
			goschtalt.UnmarshalFunc[oker.Config]("oker"),
			goschtalt.UnmarshalFunc[apiauth.Config]("auth", goschtalt.Optional()),
			// fx.Annotated{
			// 	Name:   "tracing_initial_config",
			// 	Target: goschtalt.UnmarshalFunc[candlelight.Config]("tracing"),
			// },
			fx.Annotated{
				Name:   "servers.health.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.health.http", goschtalt.Optional()),
			},
			fx.Annotated{
				Name:   "servers.metrics.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.metrics.http", goschtalt.Optional()),
			},
			fx.Annotated{
				Name:   "servers.pprof.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.pprof.http", goschtalt.Optional()),
			},
			fx.Annotated{
				Name:   "servers.primary.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.primary.http", goschtalt.Optional()),
			},
			fx.Annotated{
				Name:   "servers.alternate.config",
				Target: goschtalt.UnmarshalFunc[arrangehttp.ServerConfig]("servers.alternate.http", goschtalt.Optional()),
			},

			candlelight.New,
		),

		provideCoreEndpoints(),
		provideMetricEndpoint(),
		provideHealthCheck(),
		providePprofEndpoint(),

		arrangehttp.ProvideServer("servers.health"),
		arrangehttp.ProvideServer("servers.metrics"),
		arrangehttp.ProvideServer("servers.pprof"),
		arrangehttp.ProvideServer("servers.primary"),
		arrangehttp.ProvideServer("servers.alternate"),

		apiauth.Module,
		oker.Module,
		touchstone.Provide(),
		touchhttp.Provide(),
		metrics.Provide(),
	)

	if cli != nil && cli.Graph != "" {
		_ = os.WriteFile(cli.Graph, []byte(g), 0600)
	}

	if cli != nil && cli.Dev {
		defer func() {
			if gscfg != nil {
				fmt.Fprintln(os.Stderr, gscfg.Explain().String())
			}
		}()
	}

	if err := app.Err(); err != nil {
		return err
	}

	if run {
		app.Run()
	}

	return nil
}

// Provides a named type so it's a bit easier to flow through & use in fx.
type cliArgs []string

// Handle the CLI processing and return the processed input.
func provideCLI(args cliArgs) (*CLI, error) {
	return provideCLIWithOpts(args, false)
}

func provideCLIWithOpts(args cliArgs, testOpts bool) (*CLI, error) {
	var cli CLI

	// Create a no-op option to satisfy the kong.New() call.
	var opt kong.Option = kong.OptionFunc(
		func(*kong.Kong) error {
			return nil
		},
	)

	if testOpts {
		opt = kong.Writers(nil, nil)
	}

	parser, err := kong.New(&cli,
		kong.Name(applicationName),
		kong.Description("The cpe agent for Xmidt service.\n"+
			fmt.Sprintf("\tVersion:  %s\n", version)+
			fmt.Sprintf("\tDate:     %s\n", date)+
			fmt.Sprintf("\tCommit:   %s\n", commit)+
			fmt.Sprintf("\tBuilt By: %s\n", builtBy),
		),
		kong.UsageOnError(),
		opt,
	)
	if err != nil {
		return nil, err
	}

	if testOpts {
		parser.Exit = func(_ int) { panic("exit") }
	}

	_, err = parser.Parse(args)
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	return &cli, nil
}
