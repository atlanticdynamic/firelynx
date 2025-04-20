package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/server/config_manager"
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
		&cli.StringFlag{
			Name:    "log-level",
			Usage:   "Set logging level (debug, info, warn, error)",
			Value:   "info",
			Aliases: []string{"log"},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		configPath := cmd.String("config")
		listenAddr := cmd.String("listen")
		logLevel := cmd.String("log-level")

		// Require at least one of --config or --listen
		if configPath == "" && listenAddr == "" {
			return cli.Exit("Either --config or --listen flag is required", 1)
		}

		setupLogger(logLevel)
		logger := slog.Default()

		configManager := config_manager.New(config_manager.Config{
			Logger:     logger.With("component", "config_manager"),
			ListenAddr: listenAddr,
			ConfigPath: configPath,
		})

		serverCore, err := core.New(
			core.WithConfigCallback(configManager.GetCurrentConfig),
		)
		if err != nil {
			return cli.Exit(fmt.Errorf("failed to create server core: %w", err), 1)
		}

		// Set up a reload listener
		// TODO: review - this does not look correct
		reloadCh := configManager.GetReloadChannel()
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
			configManager,
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
