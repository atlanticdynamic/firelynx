package main

import (
	"context"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/urfave/cli/v3"
)

var validateCmd = &cli.Command{
	Name:    "validate",
	Aliases: []string{"lint"},
	Usage:   "Validate a configuration file",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "tree",
			Aliases: []string{"t"},
			Usage:   "Show detailed tree view of the validated configuration",
		},
	},
	Suggest:           true,
	ReadArgsFromStdin: true,
	Action:            validateAction,
}

func validateAction(ctx context.Context, cmd *cli.Command) error {
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
	fmt.Printf("Configuration file %s is valid\n", configPath)

	treeView := cmd.Bool("tree")
	if treeView {
		// Use the Stringer interface to print the config in a fancy tree format
		fmt.Println(cfg)
		return nil
	}

	// Print a compact representation, by default
	fmt.Printf("\nConfig Summary:\n")
	fmt.Printf("- Version: %s\n", cfg.Version)
	fmt.Printf("- Logging: level=%s format=%s\n", cfg.Logging.Level, cfg.Logging.Format)
	fmt.Printf("- Listeners: %d\n", len(cfg.Listeners))
	fmt.Printf("- Endpoints: %d\n", len(cfg.Endpoints))
	fmt.Printf("- Apps: %d\n", len(cfg.Apps))
	fmt.Println("\nUse --tree for a more detailed view of the config.")

	return nil
}
