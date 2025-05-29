package main

import (
	"context"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/cmd/firelynx/server"
	"github.com/urfave/cli/v3"
)

const (
	invalidArgsErrorMsg = "Error: --config or --listen is required.\nSee --help for more info."
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
		if configPath == "" && listenAddr == "" {
			return cli.Exit(invalidArgsErrorMsg, 1)
		}
		return server.Run(ctx, slog.Default(), configPath, listenAddr)
	},
}
