package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgfileloader"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr"
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

	// Create transaction storage
	txStorage := txstorage.NewTransactionStorage(
		txstorage.WithAsyncCleanup(true),
	)

	// Create saga orchestrator first
	txmgrOrchestrator := txmgr.NewSagaOrchestrator(txStorage, logHandler)

	// Build list of runnables based on provided arguments
	var runnables []supervisor.Runnable

	// Create cfgfileloader if configPath is provided
	if configPath != "" {
		cfgFileLoader, err := cfgfileloader.NewRunner(
			configPath,
			cfgfileloader.WithLogHandler(logHandler),
			cfgfileloader.WithContext(ctx),
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
			txmgrOrchestrator,
			cfgservice.WithLogHandler(logHandler),
			cfgservice.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("failed to create config service: %w", err)
		}
		runnables = append(runnables, cfgService)
	}

	// Create the core txmgr runner (needs a config provider)
	// TODO: This will need to be updated to get config from the appropriate source
	serverCore, err := txmgr.NewRunner(
		func() config.Config { return config.Config{} }, // Placeholder - needs to be connected to config sources
		txmgr.WithLogHandler(logHandler),
		txmgr.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create server core: %w", err)
	}
	runnables = append(runnables, serverCore)

	// Create an HTTP runner with the logger
	httpLogger := slog.New(logHandler).WithGroup("http")
	httpRunner, err := http.NewRunner(httpLogger)
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
		supervisor.WithLogHandler(logHandler),
		supervisor.WithRunnables(runnables...),
		supervisor.WithContext(ctx),
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
