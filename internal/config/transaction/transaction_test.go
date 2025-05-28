package transaction

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*ConfigTransaction, slog.Handler) {
	t.Helper()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	// Create a valid config with the current version
	cfg := &config.Config{
		Version: config.VersionLatest,
	}

	tx, err := FromTest("test_transaction", cfg, handler)
	require.NoError(t, err)
	require.NotNil(t, tx)

	return tx, handler
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates new transaction with correct initial state", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.GetState())
		assert.Equal(t, SourceTest, tx.Source)
		assert.Equal(t, "test_transaction", tx.SourceDetail)
		assert.NotEmpty(t, tx.ID)
		assert.NotNil(t, tx.logger)
		assert.NotNil(t, tx.logCollector)
		assert.False(t, tx.IsValid.Load())
		assert.Empty(t, tx.terminalErrors)
	})
}

func TestTransactionLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("validates happy path lifecycle", func(t *testing.T) {
		tx, _ := setupTest(t)

		assert.Equal(t, finitestate.StateCreated, tx.GetState())

		require.NoError(t, tx.RunValidation())
		assert.Equal(t, finitestate.StateValidated, tx.GetState())
		assert.True(t, tx.IsValid.Load())

		require.NoError(t, tx.BeginExecution())
		assert.Equal(t, finitestate.StateExecuting, tx.GetState())

		require.NoError(t, tx.MarkSucceeded())
		assert.Equal(t, finitestate.StateSucceeded, tx.GetState())

		require.NoError(t, tx.BeginReload())
		assert.Equal(t, finitestate.StateReloading, tx.GetState())

		require.NoError(t, tx.MarkCompleted())
		assert.Equal(t, finitestate.StateCompleted, tx.GetState())
	})

	t.Run("validates failed validation path", func(t *testing.T) {
		tx, _ := setupTest(t)

		invalidCfg := &config.Config{}
		tx.domainConfig = invalidCfg

		validationErrs := []error{
			errors.New("validation error 1"),
			errors.New("validation error 2"),
		}

		require.NoError(t, tx.fsm.Transition(finitestate.StateValidating))
		require.NoError(t, tx.setStateInvalid(validationErrs))
		assert.Equal(t, finitestate.StateInvalid, tx.GetState())
		assert.False(t, tx.IsValid.Load())
		assert.Len(t, tx.terminalErrors, 2)

		err := tx.BeginExecution()
		assert.ErrorIs(t, err, ErrNotValidated)
	})

	t.Run("validates compensation path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())

		require.NoError(t, tx.BeginCompensation())
		assert.Equal(t, finitestate.StateCompensating, tx.GetState())

		require.NoError(t, tx.MarkCompensated())
		assert.Equal(t, finitestate.StateCompensated, tx.GetState())
	})

	t.Run("validates failure path", func(t *testing.T) {
		tx, _ := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())

		testErr := errors.New("something bad happened")
		require.NoError(t, tx.MarkFailed(testErr))
		assert.Equal(t, finitestate.StateFailed, tx.GetState())
		assert.Contains(t, tx.terminalErrors, testErr)
	})
}

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
}

func TestLogCollection(t *testing.T) {
	t.Parallel()

	t.Run("collects and plays back logs", func(t *testing.T) {
		tx, handler := setupTest(t)

		require.NoError(t, tx.RunValidation())
		require.NoError(t, tx.BeginExecution())
		require.NoError(t, tx.MarkSucceeded())
		require.NoError(t, tx.BeginReload())
		require.NoError(t, tx.MarkCompleted())

		err := tx.PlaybackLogs(handler)
		require.NoError(t, err)
	})
}

func TestGetDuration(t *testing.T) {
	t.Parallel()

	t.Run("reports transaction duration", func(t *testing.T) {
		tx, _ := setupTest(t)

		time.Sleep(10 * time.Millisecond)

		duration := tx.GetTotalDuration()
		assert.Greater(t, duration, 0*time.Millisecond)
	})
}

func TestConvertToAppDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		input    apps.AppCollection
		expected []serverApps.AppDefinition
	}{
		{
			name:     "empty collection",
			input:    apps.AppCollection{},
			expected: []serverApps.AppDefinition{},
		},
		{
			name: "single echo app",
			input: apps.AppCollection{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
		},
		{
			name: "multiple apps",
			input: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToAppDefinitions(tt.input)

			require.Len(t, result, len(tt.expected))

			for i, def := range result {
				assert.Equal(t, tt.expected[i].ID, def.ID)
				assert.Equal(t, tt.expected[i].Config, def.Config)
			}
		})
	}
}

func TestAppFactoryIntegration(t *testing.T) {
	t.Run("creates app collection from config", func(t *testing.T) {
		// Create a config with echo apps
		cfg := &config.Config{
			Apps: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "Hello 1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "Hello 2"},
				},
			},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify apps exist
		app1, exists1 := collection.GetApp("echo1")
		assert.True(t, exists1)
		assert.Equal(t, "echo1", app1.String())

		app2, exists2 := collection.GetApp("echo2")
		assert.True(t, exists2)
		assert.Equal(t, "echo2", app2.String())
	})

	t.Run("handles empty config", func(t *testing.T) {
		cfg := &config.Config{
			Apps: apps.AppCollection{},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify no apps exist
		_, exists := collection.GetApp("nonexistent")
		assert.False(t, exists)
	})
}
