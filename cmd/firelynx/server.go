package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/server/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/server/core"
	"github.com/robbyt/go-supervisor/runnables/composite"
	"github.com/robbyt/go-supervisor/supervisor"
	"github.com/urfave/cli/v3"
)

var serverCmd = &cli.Command{
	Name:  "server",
	Usage: "Start the firelynx server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Usage:   "Path to TOML configuration file",
			Aliases: []string{"c"},
		},
		&cli.StringFlag{
			Name:    "listen",
			Usage:   "Address to bind gRPC service (tcp://host:port or a local UNIX socket unix:///path/to/socket)",
			Aliases: []string{"l"},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		configPath := cmd.String("config")
		listenAddr := cmd.String("listen")

		// Require at least one of --config or --listen
		if configPath == "" && listenAddr == "" {
			return cli.Exit("Either --config or --listen flag is required", 1)
		}

		logger := slog.Default()
		logHandler := logger.Handler()

		cManager, err := cfgservice.New(
			cfgservice.WithLogHandler(logHandler),
			cfgservice.WithListenAddr(listenAddr),
			cfgservice.WithConfigPath(configPath),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create config manager: %w", err), 1)
		}

		serverCore, err := core.New(
			core.WithLogHandler(logHandler),
			core.WithConfigCallback(cManager.GetConfigClone),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create server core: %w", err), 1)
		}

		// Create an empty configuration for the listeners composite runner
		emptyEntries := []composite.RunnableEntry[supervisor.Runnable]{}

		// Create a configuration callback for the listeners runner
		listenersConfigCallback := func() (*composite.Config[supervisor.Runnable], error) {
			// Initially returns an empty config, but will be updated by the core
			// after we call SetListenersRunner
			config, err := composite.NewConfig("http-listeners", emptyEntries)
			if err != nil {
				return nil, fmt.Errorf("failed to create listeners config: %w", err)
			}
			return config, nil
		}

		// Create a composite runner for HTTP listeners with the configuration callback
		listenersRunner, err := composite.NewRunner(listenersConfigCallback)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create listeners runner: %w", err), 1)
		}

		// Set the listeners runner in the server core
		serverCore.SetListenersRunner(listenersRunner)

		// Create a list of runnables to manage, order is important
		runnables := []supervisor.Runnable{
			cManager,
			serverCore,
			listenersRunner, // Add the composite runner for listeners
		}
		super, err := supervisor.New(
			supervisor.WithLogHandler(logHandler),
			supervisor.WithRunnables(runnables...),
			supervisor.WithContext(ctx),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create supervisor: %w", err), 1)
		}
		if err := super.Run(); err != nil {
			return cli.Exit(fmt.Errorf("failed to run server: %w", err), 1)
		}

		logger.Info("Server shutdown complete")
		return nil
	},
}
