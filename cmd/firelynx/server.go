package main

import (
	"context"
	"log/slog"
	"os"

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
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		configPath := cmd.String("config")
		listenAddr := cmd.String("listen")

		// Require at least one of --config or --listen
		if configPath == "" && listenAddr == "" {
			return cli.Exit("Either --config or --listen flag is required", 1)
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		// Create the config manager
		configManager := config_manager.New(config_manager.Config{
			Logger:     logger.With("component", "config_manager"),
			ListenAddr: listenAddr,
			ConfigPath: configPath,
		})

		// Create the server core
		serverCore := core.New(core.Config{
			Logger:     logger.With("component", "server_core"),
			ConfigFunc: configManager.GetCurrentConfig,
		})

		// Set up a reload listener
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

		// Create a list of runnables to manage
		runnables := []supervisor.Runnable{
			configManager, // Server Core depends on Config Manager, so we'll start Config Manager first
			serverCore,
		}

		// Create a new supervisor to manage our components
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		super, err := supervisor.New(
			supervisor.WithRunnables(runnables...),
			supervisor.WithLogHandler(handler),
			supervisor.WithContext(ctx),
		)
		if err != nil {
			logger.Error("Failed to create supervisor", "error", err)
			return cli.Exit(err.Error(), 1)
		}

		// Start the supervisor
		if err := super.Run(); err != nil {
			logger.Error("Failed to run server", "error", err)
			return cli.Exit(err.Error(), 1)
		}

		logger.Info("Server shutdown complete")
		return nil
	},
}
