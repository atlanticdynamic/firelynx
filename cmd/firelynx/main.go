package main

import (
	"context"
	"fmt"
	"os"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/urfave/cli/v3"
)

// Version is set during build using ldflags
var Version = "dev"

func main() {
	app := &cli.Command{
		Name:    "firelynx",
		Version: Version,
		Usage:   "CLI tool for managing resources",
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Print the version information",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Printf("firelynx version %s\n", cmd.Root().Version)
					return nil
				},
			},
			{
				Name:  "validate",
				Usage: "Validate a configuration file",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.Args().Len() < 1 {
						return fmt.Errorf("config file path required")
					}

					configPath := cmd.Args().Get(0)
					cfg, err := config.NewConfig(configPath)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					if err := cfg.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					// Print validation success message
					fmt.Printf("Configuration file %s is valid\n\n", configPath)
					
					// Print the fancy tree representation of the config
					fmt.Println(cfg)

					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}