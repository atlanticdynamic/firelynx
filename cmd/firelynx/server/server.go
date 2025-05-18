package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/cfgservice"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	txmgr "github.com/atlanticdynamic/firelynx/internal/server/txmgr"
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
	cManager, err := cfgservice.NewRunner(
		cfgservice.WithLogHandler(logHandler),
		cfgservice.WithListenAddr(listenAddr),
		cfgservice.WithConfigPath(configPath),
	)
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Debug wrapper for the config callback to log what's being passed
	configCallback := func() config.Config {
		cfg := cManager.GetDomainConfig()
		logger.Debug("Core runner config callback",
			"listeners", len(cfg.Listeners),
			"endpoints", len(cfg.Endpoints),
			"apps", len(cfg.Apps))
		return cfg
	}

	serverCore, err := txmgr.NewRunner(
		configCallback,
		txmgr.WithLogHandler(logHandler),
	)
	if err != nil {
		return fmt.Errorf("failed to create server core: %w", err)
	}

	// Create an HTTP runner using the core's config callback
	// The registry is already included in the config returned by GetHTTPConfigCallback
	cfgCallback := serverCore.GetHTTPConfigCallback()
	httpRunner, err := http.NewRunner(
		cfgCallback,
		http.WithManagerLogger(slog.Default().WithGroup("http.Runner")),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP listener runner: %w", err)
	}

	// No need to explicitly pre-load config now that we use proper Stateable interface
	// The supervisor will ensure components are ready before dependent components use them

	// Order is important here- the config manager must be started first,
	// then the server core, and then any others.
	runnables := []supervisor.Runnable{
		cManager,
		serverCore,
		httpRunner,
	}
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
