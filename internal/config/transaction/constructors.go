package transaction

import (
	"log/slog"
	"path/filepath"

	"github.com/atlanticdynamic/firelynx/internal/config"
)

// FromFile creates a new ConfigTransaction from a configuration file
func FromFile(
	filePath string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	// Use the absolute file path as the source detail
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	return New(SourceFile, absPath, "", cfg, handler)
}

// FromAPI creates a new ConfigTransaction from an API request
func FromAPI(
	requestID string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	return New(SourceAPI, "gRPC API", requestID, cfg, handler)
}

// FromTest creates a new ConfigTransaction for testing
func FromTest(
	testName string,
	cfg *config.Config,
	handler slog.Handler,
) (*ConfigTransaction, error) {
	return New(SourceTest, testName, "", cfg, handler)
}
