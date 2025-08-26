package errz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopLevelErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrFailedToLoadConfig",
			err:         ErrFailedToLoadConfig,
			expectedMsg: "failed to load config",
		},
		{
			name:        "ErrFailedToConvertConfig",
			err:         ErrFailedToConvertConfig,
			expectedMsg: "failed to convert config from proto",
		},
		{
			name:        "ErrFailedToValidateConfig",
			err:         ErrFailedToValidateConfig,
			expectedMsg: "failed to validate config",
		},
		{
			name:        "ErrUnsupportedConfigVer",
			err:         ErrUnsupportedConfigVer,
			expectedMsg: "unsupported config version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrDuplicateID",
			err:         ErrDuplicateID,
			expectedMsg: "duplicate ID",
		},
		{
			name:        "ErrEmptyID",
			err:         ErrEmptyID,
			expectedMsg: "empty ID",
		},
		{
			name:        "ErrInvalidReference",
			err:         ErrInvalidReference,
			expectedMsg: "invalid reference",
		},
		{
			name:        "ErrInvalidValue",
			err:         ErrInvalidValue,
			expectedMsg: "invalid value",
		},
		{
			name:        "ErrMissingRequiredField",
			err:         ErrMissingRequiredField,
			expectedMsg: "missing required field",
		},
		{
			name:        "ErrRouteConflict",
			err:         ErrRouteConflict,
			expectedMsg: "route conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestTypeErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrInvalidListenerType",
			err:         ErrInvalidListenerType,
			expectedMsg: "invalid listener type",
		},
		{
			name:        "ErrInvalidRouteType",
			err:         ErrInvalidRouteType,
			expectedMsg: "invalid route type",
		},
		{
			name:        "ErrInvalidAppType",
			err:         ErrInvalidAppType,
			expectedMsg: "invalid app type",
		},
		{
			name:        "ErrInvalidEvaluator",
			err:         ErrInvalidEvaluator,
			expectedMsg: "invalid evaluator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestReferenceErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrListenerNotFound",
			err:         ErrListenerNotFound,
			expectedMsg: "listener not found",
		},
		{
			name:        "ErrAppNotFound",
			err:         ErrAppNotFound,
			expectedMsg: "app not found",
		},
		{
			name:        "ErrEndpointNotFound",
			err:         ErrEndpointNotFound,
			expectedMsg: "endpoint not found",
		},
		{
			name:        "ErrRouteNotFound",
			err:         ErrRouteNotFound,
			expectedMsg: "route not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestScriptErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrMissingEvaluator",
			err:         ErrMissingEvaluator,
			expectedMsg: "missing evaluator",
		},
		{
			name:        "ErrMissingAppConfig",
			err:         ErrMissingAppConfig,
			expectedMsg: "missing app configuration",
		},
		{
			name:        "ErrEmptyCode",
			err:         ErrEmptyCode,
			expectedMsg: "empty code",
		},
		{
			name:        "ErrEmptyEntrypoint",
			err:         ErrEmptyEntrypoint,
			expectedMsg: "empty entrypoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMsg, tt.err.Error())
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that these errors can be properly wrapped and unwrapped
	baseErr := errors.New("base error")
	wrappedErr := errors.Join(ErrFailedToLoadConfig, baseErr)

	require.ErrorIs(t, wrappedErr, ErrFailedToLoadConfig)
	require.ErrorIs(t, wrappedErr, baseErr)

	// Test with multiple errors
	multiErr := errors.Join(ErrDuplicateID, ErrEmptyID, baseErr)
	require.ErrorIs(t, multiErr, ErrDuplicateID)
	require.ErrorIs(t, multiErr, ErrEmptyID)
	require.ErrorIs(t, multiErr, baseErr)
}
