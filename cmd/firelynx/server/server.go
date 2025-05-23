package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgfileloader"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/robbyt/go-supervisor/supervisor"
)

// Run starts the firelynx server using the provided context, logger, configuration file path, and gRPC listen address.
// It returns an error if the server fails to start.
func Run(
	ctx context.Context,
	logger *slog.Logger,
	configPath string,
	listenAddr string,
) error {
	logHandler := logger.Handler()

	// Transaction storage stores the history of configuration "transactions" or updates/rollbacks
	txStorage := txstorage.NewTransactionStorage(
		txstorage.WithAsyncCleanup(true),
		txstorage.WithLogHandler(logHandler),
	)

	// txmgrOrchestrator coordinates the configuration management rollout transactions with atomic roll-back
	txmgrOrchestrator := orchestrator.NewSagaOrchestrator(txStorage, logHandler)

	// Build list of runnables based on provided arguments
	var runnables []supervisor.Runnable
	var configProviders []txmgr.ConfigChannelProvider

	// Create cfgfileloader if configPath is provided
	if configPath != "" {
		cfgFileLoader, err := cfgfileloader.NewRunner(
			configPath,
			cfgfileloader.WithContext(ctx),
			cfgfileloader.WithLogHandler(logHandler),
		)
		if err != nil {
			return fmt.Errorf("failed to create config file loader: %w", err)
		}
		runnables = append(runnables, cfgFileLoader)
		configProviders = append(configProviders, cfgFileLoader)
	}

	// Create cfgservice if listenAddr is provided
	if listenAddr != "" {
		cfgService, err := cfgservice.NewRunner(
			listenAddr,
			cfgservice.WithContext(ctx),
			cfgservice.WithLogHandler(logHandler),
			cfgservice.WithConfigTransactionStorage(txStorage),
		)
		if err != nil {
			return fmt.Errorf("failed to create config service: %w", err)
		}
		runnables = append(runnables, cfgService)
		configProviders = append(configProviders, cfgService)
	}

	// Ensure at least one config provider is available
	if len(configProviders) == 0 {
		return fmt.Errorf(
			"no configuration source specified: provide either a config file path or a gRPC listen address",
		)
	}

	// combine the config providers into a single channel, if there are more than one
	configProvider, err := fanInOrDirect(ctx, configProviders)
	if err != nil {
		return fmt.Errorf("failed to create config provider: %w", err)
	}

	// Create the core txmgr runner
	txMan, err := txmgr.NewRunner(
		txmgrOrchestrator,
		configProvider,
		txmgr.WithContext(ctx),
		txmgr.WithLogHandler(logHandler),
	)
	if err != nil {
		return fmt.Errorf("failed to create server core: %w", err)
	}
	runnables = append(runnables, txMan)

	// Create an HTTP runner with the logger
	httpRunner, err := http.NewRunner(http.WithLogHandler(logHandler))
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener runner: %w", err)
	}

	// Register the HTTP runner with the transaction manager
	if err := txmgrOrchestrator.RegisterParticipant(httpRunner); err != nil {
		return fmt.Errorf("failed to register HTTP runner with saga orchestrator: %w", err)
	}

	// Add HTTP runner to runnables
	runnables = append(runnables, httpRunner)
	super, err := supervisor.New(
		supervisor.WithContext(ctx),
		supervisor.WithLogHandler(logHandler),
		supervisor.WithRunnables(runnables...),
	)
	if err != nil {
		return fmt.Errorf("failed to create supervisor: %w", err)
	}
	if err := super.Run(); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	logger.Info("Server shutdown complete")
	return nil
}
