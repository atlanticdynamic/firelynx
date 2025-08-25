package evaluators

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtismEvaluator_Type(t *testing.T) {
	extism := &ExtismEvaluator{}
	assert.Equal(t, EvaluatorTypeExtism, extism.Type())
}

func TestExtismEvaluator_String(t *testing.T) {
	tests := []struct {
		name      string
		evaluator *ExtismEvaluator
		want      string
	}{
		{
			name:      "nil",
			evaluator: nil,
			want:      "Extism(nil)",
		},
		{
			name:      "empty",
			evaluator: &ExtismEvaluator{},
			want:      "Extism(code=0 chars, entrypoint=, timeout=0s)",
		},
		{
			name: "with code and entrypoint",
			evaluator: &ExtismEvaluator{
				Code:       "base64content",
				Entrypoint: "handle_request",
			},
			want: "Extism(code=13 chars, entrypoint=handle_request, timeout=0s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.evaluator.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtismEvaluator_Validate(t *testing.T) {
	t.Run("valid with real WASM module", func(t *testing.T) {
		// Use real WASM module from go-polyscript
		wasmBase64 := base64.StdEncoding.EncodeToString(wasmdata.TestModule)
		evaluator := &ExtismEvaluator{
			Code:       wasmBase64,
			Entrypoint: wasmdata.EntrypointGreet,
		}
		err := evaluator.Validate()
		require.NoError(t, err)
		compiled, err := evaluator.GetCompiledEvaluator()
		require.NoError(t, err)
		assert.NotNil(t, compiled)
	})

	t.Run("neither code nor uri", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "",
			URI:        "",
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingCodeAndURI)
	})

	t.Run("both code and uri", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "base64content",
			URI:        "file://test.wasm",
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBothCodeAndURI)
	})

	t.Run("empty entrypoint", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "base64content",
			Entrypoint: "",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrEmptyEntrypoint)
	})

	t.Run("multiple errors", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "",
			URI:        "",
			Entrypoint: "",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMissingCodeAndURI)
		require.ErrorIs(t, err, ErrEmptyEntrypoint)
	})

	t.Run("negative timeout", func(t *testing.T) {
		wasmBase64 := base64.StdEncoding.EncodeToString(wasmdata.TestModule)
		evaluator := &ExtismEvaluator{
			Code:       wasmBase64,
			Entrypoint: "handle_request",
			Timeout:    -5 * time.Second,
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNegativeTimeout)
	})

	t.Run("uri only valid", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			URI:        "file://test.wasm",
			Entrypoint: "handle_request",
		}
		// This will fail at build stage due to file not existing
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
	})

	t.Run("invalid base64 code", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "invalid base64 !!!",
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
		assert.Contains(t, err.Error(), "failed to decode base64 WASM")
	})

	t.Run("invalid wasm bytes", func(t *testing.T) {
		invalidWasm := base64.StdEncoding.EncodeToString([]byte("not a valid wasm module"))
		evaluator := &ExtismEvaluator{
			Code:       invalidWasm,
			Entrypoint: "handle_request",
		}
		err := evaluator.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
	})
}

func TestExtismEvaluator_GetCompiledEvaluator(t *testing.T) {
	t.Run("build error propagated", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Code:       "invalid base64 !!!",
			Entrypoint: "handle_request",
		}
		result, err := evaluator.GetCompiledEvaluator()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrCompilationFailed)
		assert.Nil(t, result)
	})

	t.Run("successful build returns evaluator", func(t *testing.T) {
		wasmBase64 := base64.StdEncoding.EncodeToString(wasmdata.TestModule)
		evaluator := &ExtismEvaluator{
			Code:       wasmBase64,
			Entrypoint: wasmdata.EntrypointGreet,
		}
		result, err := evaluator.GetCompiledEvaluator()
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestExtismEvaluator_GetTimeout(t *testing.T) {
	t.Run("returns set timeout", func(t *testing.T) {
		timeout := 10 * time.Second
		evaluator := &ExtismEvaluator{
			Timeout: timeout,
		}
		assert.Equal(t, timeout, evaluator.GetTimeout())
	})

	t.Run("returns default when zero", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Timeout: 0,
		}
		assert.Equal(t, DefaultEvalTimeout, evaluator.GetTimeout())
	})

	t.Run("returns default when negative", func(t *testing.T) {
		evaluator := &ExtismEvaluator{
			Timeout: -5 * time.Second,
		}
		assert.Equal(t, DefaultEvalTimeout, evaluator.GetTimeout())
	})
}
