package transaction

import (
	"log/slog"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(os.Stdout, nil)
	cfg := &config.Config{}

	t.Run("constructs from file", func(t *testing.T) {
		tx, err := FromFile("testdata/config.toml", cfg, handler)
		require.NoError(t, err)
		assert.Equal(t, SourceFile, tx.Source)
		assert.Contains(t, tx.SourceDetail, "testdata/config.toml")
	})

	t.Run("constructs from invalid file", func(t *testing.T) {
		tx, err := FromFile("testdata/nope.toml", cfg, handler)
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
