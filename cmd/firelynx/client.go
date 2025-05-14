package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/urfave/cli/v3"
)

var clientCmd = &cli.Command{
	Name:  "client",
	Usage: "Client operations for firelynx server",
	Commands: []*cli.Command{
		{
			Name:  "apply",
			Usage: "Apply configuration to the server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "config",
					Usage:    "Path to TOML configuration file",
					Aliases:  []string{"c"},
					Required: true,
				},
				&cli.StringFlag{
					Name:     "server",
					Usage:    "Server address (tcp://host:port or unix:///path/to/socket)",
					Aliases:  []string{"s"},
					Required: true,
					Value:    "localhost:8080",
				},
				&cli.IntFlag{
					Name:    "timeout",
					Usage:   "Timeout for the operation in seconds",
					Aliases: []string{"t"},
					Value:   5,
				},
			},
			Action: clientApplyAction,
		},
		{
			Name:  "get",
			Usage: "Get current configuration from the server",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "server",
					Usage:    "Server address (tcp://host:port or unix:///path/to/socket)",
					Aliases:  []string{"s"},
					Required: true,
					Value:    "localhost:8080",
				},
				&cli.StringFlag{
					Name:    "output",
					Usage:   "Path to save configuration (if not provided, print to stdout)",
					Aliases: []string{"o"},
				},
			},
			Action: clientGetAction,
		},
	},
}

func clientApplyAction(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")
	serverAddr := cmd.String("server")

	t := cmd.Int("timeout")
	var cancel context.CancelFunc
	if t > 0 {
		timeout := time.Duration(t) * time.Second
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	logger := slog.Default()

	// Create a loader for the configuration
	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Create a new client using the internal client package
	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	// Apply the configuration
	if err := firelynxClient.ApplyConfig(ctx, configLoader); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func clientGetAction(ctx context.Context, cmd *cli.Command) error {
	serverAddr := cmd.String("server")
	outputPath := cmd.String("output")

	logger := slog.Default()

	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	// Get the configuration using the internal client package
	config, err := firelynxClient.GetConfig(ctx)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	if outputPath != "" {
		if err := firelynxClient.SaveConfig(config, outputPath); err != nil {
			return cli.Exit(err.Error(), 1)
		}
		return nil
	}

	// Format and print the configuration to stdout if no output path was provided
	configStr, err := firelynxClient.FormatConfig(config)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	fmt.Println(configStr)

	return nil
}
