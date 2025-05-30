package transaction

import (
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/config.toml
var testConfigContent string

func TestConstructors(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(os.Stdout, nil)
	cfg := &config.Config{
		Version: config.VersionLatest,
	}

	t.Run("constructs from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.toml")

		err := os.WriteFile(configPath, []byte(testConfigContent), 0o644)
		require.NoError(t, err)

		tx, err := FromFile(configPath, cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceFile, tx.Source)
		assert.Contains(t, tx.SourceDetail, configPath)
	})

	t.Run("constructs from invalid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "nope.toml")

		tx, err := FromFile(invalidPath, cfg, handler)
		require.Error(t, err)
		assert.Nil(t, tx)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("constructs from API", func(t *testing.T) {
		tx, err := FromAPI("req-123", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceAPI, tx.Source)
		assert.Equal(t, "gRPC API", tx.SourceDetail)
		assert.Equal(t, "req-123", tx.RequestID)
	})

	t.Run("constructs from test", func(t *testing.T) {
		tx, err := FromTest("unit_test", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceTest, tx.Source)
		assert.Equal(t, "unit_test", tx.SourceDetail)
	})

	t.Run("handles invalid config", func(t *testing.T) {
		// Test with nil config
		tx, err := FromTest("test", nil, handler)
		require.Error(t, err)
		assert.Nil(t, tx)
	})
}
