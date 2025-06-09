package client

import (
	_ "embed"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/test_config.toml
var testConfigContent string

//go:embed testdata/invalid_config.toml
var invalidConfigContent string

func TestApplyConfigFromPath(t *testing.T) {
	ctx := t.Context()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.toml")

	err := os.WriteFile(configPath, []byte(testConfigContent), 0o644)
	require.NoError(t, err)

	// Create a client with an invalid address (this will fail at connection time)
	client := New(Config{
		ServerAddr: "invalid-host:-1", // This will force connection error
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// Test should fail at connection time, not at config loading
	err = client.ApplyConfigFromPath(ctx, configPath)
	assert.Error(t, err)

	// The error should be a connection failure
	assert.ErrorIs(t, err, ErrConnectionFailed)
}

func TestApplyConfigFromPath_BadFile(t *testing.T) {
	ctx := t.Context()

	// Create a client
	client := New(Config{
		ServerAddr: "localhost:8080",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// Test with non-existent file
	err := client.ApplyConfigFromPath(ctx, "/non/existent/file.toml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

func TestApplyConfigFromPath_InvalidConfig(t *testing.T) {
	ctx := t.Context()

	// Create a temporary config file with invalid content
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_config.toml")

	err := os.WriteFile(configPath, []byte(invalidConfigContent), 0o644)
	require.NoError(t, err)

	// Create a client
	client := New(Config{
		ServerAddr: "localhost:8080",
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	// Test should fail at config parsing
	err = client.ApplyConfigFromPath(ctx, configPath)
	assert.Error(t, err)
	// Should be parsing error, not connection error
	assert.NotContains(t, err.Error(), "dial")
}
