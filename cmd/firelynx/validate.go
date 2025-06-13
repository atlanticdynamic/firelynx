package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/client"
	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/urfave/cli/v3"
)

// ValidationResult represents the outcome of validating a single configuration file
type ValidationResult struct {
	Path   string
	Valid  bool
	Error  error
	Config *config.Config // Only populated if validation succeeded
	Remote bool           // Whether validation was done remotely
}

// Use existing styles from fancy package for validation output

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
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Suppress per-file success messages, only show errors and summary",
		},
		&cli.BoolFlag{
			Name:  "summary",
			Usage: "Show only summary statistics, suppress all per-file output",
		},
		&cli.BoolFlag{
			Name:  "no-color",
			Usage: "Disable colored output",
		},
	},
	Suggest:           true,
	ReadArgsFromStdin: true,
	Action:            validateAction,
}

// colorEnabled checks if color output should be enabled
func colorEnabled(noColorFlag bool) bool {
	// Check --no-color flag first
	if noColorFlag {
		return false
	}
	// Check NO_COLOR environment variable (standard convention)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

// formatValidResult formats a valid validation result
func formatValidResult(result ValidationResult, treeView, noColor bool) string {
	if treeView {
		if noColor {
			return fmt.Sprintf("%s: valid", result.Path)
		}
		path := fancy.PathText(result.Path)
		validText := fancy.ValidText("valid")
		return fmt.Sprintf("%s: %s", path, validText)
	}

	details := fmt.Sprintf("(%s, %d listeners, %d endpoints, %d apps)",
		result.Config.Version, len(result.Config.Listeners),
		len(result.Config.Endpoints), len(result.Config.Apps))

	if noColor {
		return fmt.Sprintf("%s: valid %s", result.Path, details)
	}

	path := fancy.PathText(result.Path)
	validText := fancy.ValidText("valid")
	detailsText := fancy.SummaryText(details)
	return fmt.Sprintf("%s: %s %s", path, validText, detailsText)
}

// formatInvalidResult formats an invalid validation result
func formatInvalidResult(result ValidationResult, noColor bool) string {
	if noColor {
		return fmt.Sprintf("%s: %v", result.Path, result.Error)
	}

	path := fancy.PathText(result.Path)
	errorMsg := fancy.ErrorText(result.Error.Error())
	return fmt.Sprintf("%s: %s", path, errorMsg)
}

// formatSummary formats the summary line
func formatSummary(
	totalFiles, passedCount, failedCount int,
	duration time.Duration,
	noColor bool,
) string {
	if noColor {
		if failedCount > 0 {
			return fmt.Sprintf("%d files validated: %d passed, %d failed (%v)",
				totalFiles, passedCount, failedCount, duration)
		} else {
			return fmt.Sprintf("%d files validated: all passed (%v)",
				totalFiles, duration)
		}
	}

	baseText := fancy.SummaryText("files validated:")
	timing := fancy.SummaryText(fmt.Sprintf("(%v)", duration))

	if failedCount > 0 {
		total := fancy.CountText(fmt.Sprintf("%d", totalFiles))
		passed := fancy.ValidText(fmt.Sprintf("%d passed", passedCount))
		failed := fancy.ErrorText(fmt.Sprintf("%d failed", failedCount))
		return fmt.Sprintf("%s %s %s, %s %s", total, baseText, passed, failed, timing)
	} else {
		total := fancy.CountText(fmt.Sprintf("%d", totalFiles))
		allPassed := fancy.ValidText("all passed")
		return fmt.Sprintf("%s %s %s %s", total, baseText, allPassed, timing)
	}
}

func validateAction(ctx context.Context, cmd *cli.Command) error {
	startTime := time.Now()

	// Extract CLI flags
	configPath := cmd.String("config")
	serverAddr := cmd.String("server")
	treeView := cmd.Bool("tree")
	quiet := cmd.Bool("quiet")
	summaryOnly := cmd.Bool("summary")
	noColor := !colorEnabled(cmd.Bool("no-color"))

	var configPaths []string

	if configPath != "" {
		// Single file via --config flag
		configPaths = []string{configPath}
	} else {
		// Multiple files via positional arguments
		if cmd.Args().Len() < 1 {
			return fmt.Errorf(
				"config file path required (use the --config flag, or provide config files as positional arguments)",
			)
		}
		configPaths = cmd.Args().Slice()
	}

	// Validate all files and collect results
	var results []ValidationResult
	if serverAddr != "" {
		results = validateRemote(ctx, configPaths, serverAddr)
	} else {
		results = validateLocal(ctx, configPaths)
	}

	// Count results
	var passedCount, failedCount int
	for _, result := range results {
		if result.Valid {
			passedCount++
		} else {
			failedCount++
		}
	}

	// Output results based on flags
	if !summaryOnly {
		// Print per-file results
		for _, result := range results {
			if !result.Valid {
				// Always show errors with consistent format
				fmt.Println(formatInvalidResult(result, noColor))
			} else if !quiet {
				// Show success with config summary on single line
				if treeView {
					fmt.Println(formatValidResult(result, true, noColor))
					fmt.Println(result.Config)
				} else {
					fmt.Println(formatValidResult(result, false, noColor))
				}
			}
		}
	}

	// Show summary
	duration := time.Since(startTime)
	totalFiles := len(results)

	if summaryOnly || totalFiles > 1 || failedCount > 0 {
		fmt.Println(formatSummary(totalFiles, passedCount, failedCount, duration, noColor))
	}

	// Return appropriate exit code
	if failedCount > 0 {
		return fmt.Errorf("validation failed")
	}

	return nil
}

func validateRemote(
	ctx context.Context,
	configPaths []string,
	serverAddr string,
) []ValidationResult {
	logger := slog.Default()

	// Create a single client for remote validation (reuse connection)
	firelynxClient := client.New(client.Config{
		Logger:     logger,
		ServerAddr: serverAddr,
	})

	var results []ValidationResult

	// Validate each file using the same gRPC connection
	for _, configPath := range configPaths {
		result := ValidationResult{
			Path:   configPath,
			Remote: true,
		}

		// Check if context has been canceled
		if ctx.Err() != nil {
			result.Error = fmt.Errorf("validation canceled: %w", ctx.Err())
			results = append(results, result)
			break
		}

		// Create a loader for the configuration
		configLoader, err := loader.NewLoaderFromFilePath(configPath)
		if err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		// Load the protobuf config for remote validation
		pbConfig, err := configLoader.LoadProto()
		if err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		// Validate remotely using the gRPC service
		isValid, err := firelynxClient.ValidateConfig(ctx, pbConfig)
		if err != nil {
			result.Error = fmt.Errorf("remote validation failed: %w", err)
			results = append(results, result)
			continue
		}

		if !isValid {
			result.Error = fmt.Errorf("configuration validation failed on server")
			results = append(results, result)
			continue
		}

		// Validation succeeded - load config for display purposes
		cfg, err := config.NewConfig(configPath)
		if err != nil {
			// This shouldn't happen since we already validated successfully
			result.Error = fmt.Errorf("failed to load config for display: %w", err)
			results = append(results, result)
			continue
		}

		result.Valid = true
		result.Config = cfg
		results = append(results, result)
	}

	return results
}

func validateLocal(ctx context.Context, configPaths []string) []ValidationResult {
	var results []ValidationResult

	for _, configPath := range configPaths {
		result := ValidationResult{
			Path:   configPath,
			Remote: false,
		}

		// Check if context has been canceled
		if ctx.Err() != nil {
			result.Error = fmt.Errorf("validation canceled: %w", ctx.Err())
			results = append(results, result)
			break
		}

		cfg, err := config.NewConfig(configPath)
		if err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		if err := cfg.Validate(); err != nil {
			result.Error = err
			results = append(results, result)
			continue
		}

		result.Valid = true
		result.Config = cfg
		results = append(results, result)
	}

	return results
}
