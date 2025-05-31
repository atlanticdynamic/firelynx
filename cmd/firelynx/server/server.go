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

	// Ensure at least one config provider is available
	if configPath == "" && listenAddr == "" {
		return fmt.Errorf(
			"no configuration source specified: provide either a config file path and/or a gRPC listen address",
		)
	}

	// Transaction storage stores the history of configuration "transactions" or updates/rollbacks
	txStorage := txstorage.NewMemoryStorage(
		txstorage.WithAsyncCleanup(true),
		txstorage.WithLogHandler(logHandler),
	)

	// txmgrOrchestrator coordinates the configuration management rollout transactions with atomic roll-back
	txmgrOrchestrator := orchestrator.NewSagaOrchestrator(txStorage, logHandler)

	// Create the transaction manager, which has a transaction "siphon" channel
	txMan, err := txmgr.NewRunner(
		txmgrOrchestrator,
		txmgr.WithLogHandler(logHandler),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction manager: %w", err)
	}

	// Get the transaction siphon channel, unbuffered and ready immediately
	txSiphon := txMan.GetTransactionSiphon()

	// Build list of runnables based on provided arguments
	var runnables []supervisor.Runnable

	// Create cfgfileloader if configPath is provided
	if configPath != "" {
		cfgFileLoader, err := cfgfileloader.NewRunner(
			configPath,
			txSiphon,
			cfgfileloader.WithLogHandler(logHandler),
		)
		if err != nil {
			return fmt.Errorf("failed to create config file loader: %w", err)
		}
		runnables = append(runnables, cfgFileLoader)
	}

	// Create cfgservice if listenAddr is provided
	if listenAddr != "" {
		cfgService, err := cfgservice.NewRunner(
			listenAddr,
			txSiphon,
			cfgservice.WithLogHandler(logHandler),
			cfgservice.WithConfigTransactionStorage(txStorage),
		)
		if err != nil {
			return fmt.Errorf("failed to create config service: %w", err)
		}
		runnables = append(runnables, cfgService)
	}

	// Order matters: config providers first, then txmgr, then HTTP runner
	runnables = append(runnables, txMan)

	// Create an HTTP runner with the logger
	httpRunner, err := http.NewRunner(
		http.WithLogHandler(logHandler),
		http.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP runner: %w", err)
	}

	// Register the HTTP runner with the transaction manager as a saga participant
	if err := txmgrOrchestrator.RegisterParticipant(httpRunner); err != nil {
		return fmt.Errorf("failed to register HTTP runner with saga orchestrator: %w", err)
	}

	// Add HTTP runner to runnables
	runnables = append(runnables, httpRunner)
	pid0, err := supervisor.New(
		supervisor.WithContext(ctx),
		supervisor.WithLogHandler(logHandler),
		supervisor.WithRunnables(runnables...),
	)
	if err != nil {
		return fmt.Errorf("failed to create supervisor: %w", err)
	}
	if err := pid0.Run(); err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	logger.Debug("Server shutdown complete")
	return nil
}
