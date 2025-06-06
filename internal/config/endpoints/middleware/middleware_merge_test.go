package middleware

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareCollection_Merge(t *testing.T) {
	t.Parallel()

	// Create test middlewares
	logger1 := Middleware{
		ID:     "01-logger",
		Config: logger.NewConsoleLogger(),
	}
	logger2 := Middleware{
		ID:     "02-auth",
		Config: logger.NewConsoleLogger(),
	}
	logger3 := Middleware{
		ID:     "00-rate-limit",
		Config: logger.NewConsoleLogger(),
	}
	// Same ID as logger1 but different config (for testing overrides)
	logger1Override := Middleware{
		ID: "01-logger",
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatTxt, // Different from logger1
				Level:  logger.LevelDebug,
			},
		},
	}

	t.Run("Merge empty collections", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{}
		other := MiddlewareCollection{}

		merged := base.Merge(other)
		assert.Empty(t, merged)
	})

	t.Run("Merge with empty base", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{}
		other := MiddlewareCollection{logger1, logger2}

		merged := base.Merge(other)
		require.Len(t, merged, 2)
		assert.Equal(t, "01-logger", merged[0].ID)
		assert.Equal(t, "02-auth", merged[1].ID)
	})

	t.Run("Merge with empty other", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{logger1, logger2}
		other := MiddlewareCollection{}

		merged := base.Merge(other)
		require.Len(t, merged, 2)
		assert.Equal(t, "01-logger", merged[0].ID)
		assert.Equal(t, "02-auth", merged[1].ID)
	})

	t.Run("Merge with no duplicates", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{logger1}
		other := MiddlewareCollection{logger2, logger3}

		merged := base.Merge(other)
		require.Len(t, merged, 3)
		// Check alphabetical ordering
		assert.Equal(t, "00-rate-limit", merged[0].ID)
		assert.Equal(t, "01-logger", merged[1].ID)
		assert.Equal(t, "02-auth", merged[2].ID)
	})

	t.Run("Merge with overrides", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{logger1, logger2}
		other := MiddlewareCollection{logger1Override, logger3}

		merged := base.Merge(other)
		require.Len(t, merged, 3)

		// Check alphabetical ordering
		assert.Equal(t, "00-rate-limit", merged[0].ID)
		assert.Equal(t, "01-logger", merged[1].ID)
		assert.Equal(t, "02-auth", merged[2].ID)

		// Check that other's logger1 overrode base's logger1
		logger1Merged := merged.FindByID("01-logger")
		require.NotNil(t, logger1Merged)
		consoleConfig, ok := logger1Merged.Config.(*logger.ConsoleLogger)
		require.True(t, ok)
		assert.Equal(
			t,
			logger.FormatTxt,
			consoleConfig.Options.Format,
		) // Should be the overridden value
	})

	t.Run("Merge multiple collections", func(t *testing.T) {
		t.Parallel()

		base := MiddlewareCollection{logger1}
		second := MiddlewareCollection{logger2}
		third := MiddlewareCollection{logger3, logger1Override}

		merged := base.Merge(second, third)
		require.Len(t, merged, 3)

		// Check alphabetical ordering
		assert.Equal(t, "00-rate-limit", merged[0].ID)
		assert.Equal(t, "01-logger", merged[1].ID)
		assert.Equal(t, "02-auth", merged[2].ID)

		// Check that third's logger1 overrode base's logger1
		logger1Merged := merged.FindByID("01-logger")
		require.NotNil(t, logger1Merged)
		consoleConfig, ok := logger1Merged.Config.(*logger.ConsoleLogger)
		require.True(t, ok)
		assert.Equal(
			t,
			logger.FormatTxt,
			consoleConfig.Options.Format,
		) // Should be from third collection
	})

	t.Run("Alphabetical ordering", func(t *testing.T) {
		t.Parallel()

		// Create middlewares with IDs that will test alphabetical ordering
		middlewares := []Middleware{
			{ID: "99-last", Config: logger.NewConsoleLogger()},
			{ID: "02-second", Config: logger.NewConsoleLogger()},
			{ID: "01-first", Config: logger.NewConsoleLogger()},
			{ID: "10-tenth", Config: logger.NewConsoleLogger()}, // Tests string sort vs numeric
		}

		base := MiddlewareCollection{}
		other := MiddlewareCollection(middlewares)

		merged := base.Merge(other)

		// Check that ordering is correct (string alphabetical, not numeric)
		require.Len(t, merged, 4)
		assert.Equal(t, "01-first", merged[0].ID)
		assert.Equal(t, "02-second", merged[1].ID)
		assert.Equal(t, "10-tenth", merged[2].ID) // "10" comes before "99" alphabetically
		assert.Equal(t, "99-last", merged[3].ID)
	})

	t.Run("Nil receiver behavior", func(t *testing.T) {
		t.Parallel()

		var nilCollection MiddlewareCollection
		other := MiddlewareCollection{logger1}

		// Nil receiver should still work
		merged := nilCollection.Merge(other)
		require.Len(t, merged, 1)
		assert.Equal(t, "01-logger", merged[0].ID)
	})
}

func TestMiddlewareCollection_Merge_Precedence(t *testing.T) {
	t.Parallel()

	// Create middlewares with same ID but different configurations
	baseLogger := Middleware{
		ID: "logger",
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatJSON,
				Level:  logger.LevelInfo,
			},
		},
	}

	overrideLogger := Middleware{
		ID: "logger", // Same ID
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatTxt,  // Different format
				Level:  logger.LevelDebug, // Different level
			},
		},
	}

	base := MiddlewareCollection{baseLogger}
	override := MiddlewareCollection{overrideLogger}

	merged := base.Merge(override)

	// Should have only 1 middleware (deduplicated)
	require.Len(t, merged, 1)
	assert.Equal(t, "logger", merged[0].ID)

	// Should be the override's version (override takes precedence)
	consoleConfig, ok := merged[0].Config.(*logger.ConsoleLogger)
	require.True(t, ok)
	assert.Equal(t, logger.FormatTxt, consoleConfig.Options.Format)
	assert.Equal(t, logger.LevelDebug, consoleConfig.Options.Level)
}
