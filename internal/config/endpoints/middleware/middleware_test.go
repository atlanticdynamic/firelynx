package middleware

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		middleware  Middleware
		expectError bool
	}{
		{
			name: "Valid console logger middleware",
			middleware: Middleware{
				ID:     "test-logger",
				Config: logger.NewConsoleLogger(),
			},
			expectError: false,
		},
		{
			name: "Missing ID",
			middleware: Middleware{
				Config: logger.NewConsoleLogger(),
			},
			expectError: true,
		},
		{
			name: "Missing config",
			middleware: Middleware{
				ID: "test-logger",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.middleware.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMiddlewareCollection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		collection  MiddlewareCollection
		expectError bool
	}{
		{
			name:        "Empty collection",
			collection:  MiddlewareCollection{},
			expectError: false,
		},
		{
			name: "Valid collection",
			collection: MiddlewareCollection{
				{
					ID:     "logger1",
					Config: logger.NewConsoleLogger(),
				},
				{
					ID:     "logger2",
					Config: logger.NewConsoleLogger(),
				},
			},
			expectError: false,
		},
		{
			name: "Duplicate IDs",
			collection: MiddlewareCollection{
				{
					ID:     "logger1",
					Config: logger.NewConsoleLogger(),
				},
				{
					ID:     "logger1", // Duplicate
					Config: logger.NewConsoleLogger(),
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.collection.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMiddlewareCollection_FindByID(t *testing.T) {
	t.Parallel()

	collection := MiddlewareCollection{
		{
			ID:     "logger1",
			Config: logger.NewConsoleLogger(),
		},
		{
			ID:     "logger2",
			Config: logger.NewConsoleLogger(),
		},
	}

	// Find existing middleware
	found := collection.FindByID("logger1")
	require.NotNil(t, found)
	assert.Equal(t, "logger1", found.ID)

	// Find non-existing middleware
	notFound := collection.FindByID("non-existing")
	assert.Nil(t, notFound)
}
