package middleware

import (
	"fmt"
	"strings"
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
		{
			name: "Invalid ID with spaces",
			middleware: Middleware{
				ID:     "invalid id with spaces",
				Config: logger.NewConsoleLogger(),
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
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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

func TestMiddleware_String(t *testing.T) {
	t.Parallel()

	t.Run("Console logger middleware", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		middleware := Middleware{
			ID:     "test-logger",
			Config: config,
		}

		result := middleware.String()
		expected := "Middleware(test-logger: " + config.String() + ")"
		assert.Equal(t, expected, result)
	})

	t.Run("Empty ID", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		middleware := Middleware{
			ID:     "",
			Config: config,
		}

		result := middleware.String()
		expected := "Middleware(: " + config.String() + ")"
		assert.Equal(t, expected, result)
	})

	t.Run("Special characters in ID", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		middleware := Middleware{
			ID:     "test-logger-with-dashes_and_underscores.and.dots",
			Config: config,
		}

		result := middleware.String()
		assert.Contains(t, result, "test-logger-with-dashes_and_underscores.and.dots")
		assert.Contains(t, result, "Middleware(")
		assert.Contains(t, result, config.String())
	})
}

func TestMiddlewareCollection_String(t *testing.T) {
	t.Parallel()

	t.Run("Empty collection", func(t *testing.T) {
		t.Parallel()

		collection := MiddlewareCollection{}
		result := collection.String()
		assert.Equal(t, "MiddlewareCollection(empty)", result)
	})

	t.Run("Single middleware", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		collection := MiddlewareCollection{
			{
				ID:     "logger1",
				Config: config,
			},
		}

		result := collection.String()
		expected := "MiddlewareCollection[Middleware(logger1: " + config.String() + ")]"
		assert.Equal(t, expected, result)
	})

	t.Run("Multiple middlewares", func(t *testing.T) {
		t.Parallel()

		config1 := logger.NewConsoleLogger()
		config2 := logger.NewConsoleLogger()
		collection := MiddlewareCollection{
			{
				ID:     "logger1",
				Config: config1,
			},
			{
				ID:     "logger2",
				Config: config2,
			},
		}

		result := collection.String()

		// Verify structure
		assert.Contains(t, result, "MiddlewareCollection[")
		assert.Contains(t, result, "Middleware(logger1:")
		assert.Contains(t, result, "Middleware(logger2:")
		assert.Contains(t, result, config1.String())
		assert.Contains(t, result, config2.String())

		// Verify comma separation
		assert.Contains(t, result, ", ")

		// Verify it ends properly
		assert.True(t, strings.HasSuffix(result, "]"))
	})

	t.Run("Large collection", func(t *testing.T) {
		t.Parallel()

		var collection MiddlewareCollection
		for i := 0; i < 5; i++ {
			collection = append(collection, Middleware{
				ID:     fmt.Sprintf("logger%d", i),
				Config: logger.NewConsoleLogger(),
			})
		}

		result := collection.String()

		// Verify all middlewares are included
		for i := 0; i < 5; i++ {
			assert.Contains(t, result, fmt.Sprintf("logger%d", i))
		}

		// Count middleware separators (should be n-1 for n items)
		separatorCount := strings.Count(result, "), Middleware(")
		assert.Equal(t, 4, separatorCount, "should have 4 separators between 5 middleware items")
	})
}

func TestMiddlewareCollection_ToTree(t *testing.T) {
	t.Parallel()

	t.Run("Empty collection", func(t *testing.T) {
		t.Parallel()

		collection := MiddlewareCollection{}
		tree := collection.ToTree()

		require.NotNil(t, tree)
		// For empty collection, verify the tree can be rendered
		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "Middlewares (0)")
	})

	t.Run("Single middleware", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		collection := MiddlewareCollection{
			{
				ID:     "test-logger",
				Config: config,
			},
		}

		tree := collection.ToTree()

		require.NotNil(t, tree)
		// Verify the tree structure by checking its string output
		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "Middlewares (1)")
		assert.Contains(t, treeString, "test-logger")
	})

	t.Run("Multiple middlewares", func(t *testing.T) {
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
			{
				ID:     "logger3",
				Config: logger.NewConsoleLogger(),
			},
		}

		tree := collection.ToTree()

		require.NotNil(t, tree)
		// Verify the tree structure by checking its string output
		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "Middlewares (3)")
		assert.Contains(t, treeString, "logger1")
		assert.Contains(t, treeString, "logger2")
		assert.Contains(t, treeString, "logger3")
	})

	t.Run("Tree structure integrity", func(t *testing.T) {
		t.Parallel()

		config := logger.NewConsoleLogger()
		collection := MiddlewareCollection{
			{
				ID:     "test-logger",
				Config: config,
			},
		}

		tree := collection.ToTree()

		// Verify the tree can be traversed without panics
		require.NotNil(t, tree)
		require.NotNil(t, tree.Tree())

		// Check the structure is consistent by verifying content
		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "test-logger")

		// Verify config content appears in the tree
		assert.Contains(t, treeString, "Config:")
	})
}
