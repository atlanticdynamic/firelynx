package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

const flagLogLevel = "log-level"

func main() {
	// Initialize logger with default level to ensure it's always configured
	SetupLogger("info")

	app := &cli.Command{
		Name:    "firelynx",
		Version: Version,
		Usage:   "CLI tool for managing resources",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    flagLogLevel,
				Usage:   "Set logging level (debug, info, warn, error)",
				Value:   "info",
				Aliases: []string{"log"},
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Reconfigure logger if log level was provided
			logLevel := cmd.String(flagLogLevel)
			SetupLogger(logLevel)
			return ctx, nil
		},
		Commands: []*cli.Command{
			versionCmd,
			validateCmd,
			serverCmd,
			clientCmd,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
