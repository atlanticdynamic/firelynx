package middleware

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMiddleware_ProtoRoundTrip(t *testing.T) {
	t.Parallel()

	// Create a middleware with console logger
	original := Middleware{
		ID:     "test-logger",
		Config: logger.NewConsoleLogger(),
	}

	// Convert to proto
	pbMiddleware := original.ToProto()
	require.NotNil(t, pbMiddleware)
	assert.Equal(t, "test-logger", pbMiddleware.GetId())
	assert.Equal(t, pb.Middleware_TYPE_CONSOLE_LOGGER, pbMiddleware.GetType())
	assert.NotNil(t, pbMiddleware.GetConsoleLogger())

	// Convert back from proto
	converted, err := middlewareFromProto(pbMiddleware)
	require.NoError(t, err)
	assert.Equal(t, original.ID, converted.ID)
	assert.Equal(t, original.Config.Type(), converted.Config.Type())
}

func TestMiddlewareCollection_ProtoRoundTrip(t *testing.T) {
	t.Parallel()

	// Create a collection with multiple middlewares
	original := MiddlewareCollection{
		{
			ID:     "logger1",
			Config: logger.NewConsoleLogger(),
		},
		{
			ID: "logger2",
			Config: &logger.ConsoleLogger{
				Options: logger.LogOptionsGeneral{
					Format: logger.FormatTxt,
					Level:  logger.LevelDebug,
				},
				IncludeOnlyPaths: []string{"/api"},
			},
		},
	}

	// Convert to proto
	pbMiddlewares := original.ToProto()
	require.Len(t, pbMiddlewares, 2)

	// Convert back from proto
	converted, err := FromProto(pbMiddlewares)
	require.NoError(t, err)
	require.Len(t, converted, 2)

	// Verify first middleware
	assert.Equal(t, original[0].ID, converted[0].ID)
	assert.Equal(t, original[0].Config.Type(), converted[0].Config.Type())

	// Verify second middleware
	assert.Equal(t, original[1].ID, converted[1].ID)
	assert.Equal(t, original[1].Config.Type(), converted[1].Config.Type())
}

func TestMiddleware_ValidationIntegration(t *testing.T) {
	t.Parallel()

	// Test that validation catches proto conversion errors
	collection := MiddlewareCollection{
		{
			ID: "valid-logger",
			Config: &logger.ConsoleLogger{
				Options: logger.LogOptionsGeneral{
					Format: logger.FormatJSON,
					Level:  logger.LevelInfo,
				},
			},
		},
		{
			ID: "invalid-logger",
			Config: &logger.ConsoleLogger{
				Options: logger.LogOptionsGeneral{
					Format: "invalid-format", // This should cause validation to fail
				},
			},
		},
	}

	err := collection.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestMiddlewareCollection_ToProto_EmptyCollection(t *testing.T) {
	t.Parallel()

	var collection MiddlewareCollection
	result := collection.ToProto()
	assert.Nil(t, result, "Empty collection should return nil")
}

func TestFromProto_EmptySlice(t *testing.T) {
	t.Parallel()

	result, err := FromProto(nil)
	assert.NoError(t, err)
	assert.Nil(t, result)

	result, err = FromProto([]*pb.Middleware{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestMiddlewareFromProto_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("EmptyID", func(t *testing.T) {
		pbMiddleware := &pb.Middleware{
			Id:   proto.String(""),
			Type: pb.Middleware_TYPE_CONSOLE_LOGGER.Enum(),
		}

		_, err := middlewareFromProto(pbMiddleware)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware has empty ID")
	})

	t.Run("ConsoleLoggerMissingConfig", func(t *testing.T) {
		pbMiddleware := &pb.Middleware{
			Id:   proto.String("test-logger"),
			Type: pb.Middleware_TYPE_CONSOLE_LOGGER.Enum(),
		}

		_, err := middlewareFromProto(pbMiddleware)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "console logger middleware missing config")
	})

	t.Run("UnspecifiedType", func(t *testing.T) {
		pbMiddleware := &pb.Middleware{
			Id:   proto.String("test-unspecified"),
			Type: pb.Middleware_TYPE_UNSPECIFIED.Enum(),
		}

		_, err := middlewareFromProto(pbMiddleware)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware type unspecified")
	})

	t.Run("UnknownType", func(t *testing.T) {
		pbMiddleware := &pb.Middleware{
			Id:   proto.String("test-unknown"),
			Type: pb.Middleware_Type(999).Enum(), // Unknown type value
		}

		_, err := middlewareFromProto(pbMiddleware)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown middleware type")
	})
}

func TestFromProto_ErrorPropagation(t *testing.T) {
	t.Parallel()

	pbMiddlewares := []*pb.Middleware{
		{
			Id:   proto.String("valid-logger"),
			Type: pb.Middleware_TYPE_CONSOLE_LOGGER.Enum(),
			Config: &pb.Middleware_ConsoleLogger{
				ConsoleLogger: &pb.ConsoleLoggerConfig{},
			},
		},
		{
			Id:   proto.String(""), // Invalid empty ID
			Type: pb.Middleware_TYPE_CONSOLE_LOGGER.Enum(),
		},
	}

	_, err := FromProto(pbMiddlewares)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "middleware at index 1")
	assert.Contains(t, err.Error(), "middleware has empty ID")
}
