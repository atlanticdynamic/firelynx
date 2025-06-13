package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
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
		&cli.StringFlag{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "Server address for remote validation (tcp://host:port or unix:///path/to/socket). If not provided, validates locally.",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Path to the configuration file",
		},
	},
	Suggest:           true,
	ReadArgsFromStdin: true,
	Action:            validateAction,
}

func validateAction(ctx context.Context, cmd *cli.Command) error {
	// Check for config flag first
	configPath := cmd.String("config")

	// If no config flag, check for positional argument
	if configPath == "" {
		if cmd.Args().Len() < 1 {
			return fmt.Errorf(
				"config file path required (use the --config flag, or provide the config file as positional argument)",
			)
		}
		configPath = cmd.Args().Get(0)
	}

	serverAddr := cmd.String("server")

	// If server address is provided, use remote validation
	if serverAddr != "" {
		return validateRemote(ctx, configPath, serverAddr, cmd.Bool("tree"))
	}

	// Otherwise, validate locally
	return validateLocal(ctx, configPath, cmd.Bool("tree"))
}

// renderConfigSummary creates a formatted summary string for the configuration
func renderConfigSummary(path string, cfg *config.Config) string {
	var summary strings.Builder

	summary.WriteString("\nConfig Summary:\n")
	summary.WriteString(fmt.Sprintf("- Path: %s\n", path))
	summary.WriteString(fmt.Sprintf("- Version: %s\n", cfg.Version))
	summary.WriteString(fmt.Sprintf("- Listeners: %d\n", len(cfg.Listeners)))
	summary.WriteString(fmt.Sprintf("- Endpoints: %d\n", len(cfg.Endpoints)))
	summary.WriteString(fmt.Sprintf("- Apps: %d\n", len(cfg.Apps)))
	summary.WriteString("\nUse --tree for a more detailed view of the config.")

	return summary.String()
}

func validateRemote(ctx context.Context, configPath, serverAddr string, treeView bool) error {
	logger := slog.Default()

	// Create a loader for the configuration
	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Load the protobuf config for remote validation
	pbConfig, err := configLoader.LoadProto()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create a client for remote validation
	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	// Validate remotely using the gRPC service
	isValid, err := firelynxClient.ValidateConfig(ctx, pbConfig)
	if err != nil {
		return fmt.Errorf("remote validation failed: %w", err)
	}

	if !isValid {
		return fmt.Errorf("configuration validation failed on server")
	}

	fmt.Printf("Configuration file %s is valid (validated remotely)\n", configPath)

	// Load the domain config locally for display (needed for both tree view and summary)
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config for display: %w", err)
	}

	if treeView {
		fmt.Println(cfg)
		return nil
	}

	fmt.Println(renderConfigSummary(configPath, cfg))
	return nil
}

func validateLocal(_ context.Context, configPath string, treeView bool) error {
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("Configuration file %s is valid\n", configPath)

	if treeView {
		// Use the Stringer interface to print the config in a fancy tree format
		fmt.Println(cfg)
		return nil
	}

	fmt.Println(renderConfigSummary(configPath, cfg))
	return nil
}
