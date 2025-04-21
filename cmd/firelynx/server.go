package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/server/cfgrpc"
	"github.com/atlanticdynamic/firelynx/internal/server/core"
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

		cManager, err := cfgrpc.New(
			cfgrpc.WithLogger(logger.With("component", "config_manager")),
			cfgrpc.WithListenAddr(listenAddr),
			cfgrpc.WithConfigPath(configPath),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create config manager: %w", err), 1)
		}

		serverCore, err := core.New(
			core.WithLogger(logger.With("component", "core")),
			core.WithConfigCallback(cManager.GetConfigClone),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create server core: %w", err), 1)
		}

		// Set up a reload listener
		// TODO: review - this does not look correct
		reloadCh := cManager.GetReloadChannel()
		go func() {
			for {
				select {
				case <-reloadCh:
					logger.Info("Reload notification received")
					serverCore.Reload()
				case <-ctx.Done():
					return
				}
			}
		}()

		// Create a list of runnables to manage, order is important
		runnables := []supervisor.Runnable{
			cManager,
			serverCore,
		}

		super, err := supervisor.New(
			supervisor.WithRunnables(runnables...),
			supervisor.WithLogHandler(slog.Default().Handler()),
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
