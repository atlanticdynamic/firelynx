package evaluators

import (
	"testing"
	"time"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

//nolint:dupl
func TestRisorEvaluatorFromProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		proto := (*pbApps.RisorEvaluator)(nil)
		got := RisorEvaluatorFromProto(proto)
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		proto := &pbApps.RisorEvaluator{}
		got := RisorEvaluatorFromProto(proto)
		want := &RisorEvaluator{Code: "", Timeout: 0}
		assert.Equal(t, want, got)
	})

	t.Run("with code", func(t *testing.T) {
		proto := &pbApps.RisorEvaluator{
			Source: &pbApps.RisorEvaluator_Code{Code: "print('hello')"},
		}
		got := RisorEvaluatorFromProto(proto)
		want := &RisorEvaluator{Code: "print('hello')", Timeout: 0}
		assert.Equal(t, want, got)
	})

	t.Run("with timeout", func(t *testing.T) {
		proto := &pbApps.RisorEvaluator{
			Timeout: durationpb.New(5 * time.Second),
		}
		got := RisorEvaluatorFromProto(proto)
		want := &RisorEvaluator{Code: "", Timeout: 5 * time.Second}
		assert.Equal(t, want, got)
	})

	t.Run("with code and timeout", func(t *testing.T) {
		proto := &pbApps.RisorEvaluator{
			Source:  &pbApps.RisorEvaluator_Code{Code: "print('hello')"},
			Timeout: durationpb.New(5 * time.Second),
		}
		got := RisorEvaluatorFromProto(proto)
		want := &RisorEvaluator{Code: "print('hello')", Timeout: 5 * time.Second}
		assert.Equal(t, want, got)
	})
}

func TestRisorEvaluator_ToProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var evaluator *RisorEvaluator = nil
		got := evaluator.ToProto()
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		evaluator := &RisorEvaluator{}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with code", func(t *testing.T) {
		evaluator := &RisorEvaluator{Code: "print('hello')"}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "print('hello')", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with zero timeout", func(t *testing.T) {
		evaluator := &RisorEvaluator{Timeout: 0}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with positive timeout", func(t *testing.T) {
		evaluator := &RisorEvaluator{Timeout: 5 * time.Second}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.NotNil(t, got.Timeout)
		assert.Equal(t, 5*time.Second, got.Timeout.AsDuration())
	})

	t.Run("with code and timeout", func(t *testing.T) {
		evaluator := &RisorEvaluator{Code: "print('hello')", Timeout: 5 * time.Second}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "print('hello')", got.GetCode())
		assert.NotNil(t, got.Timeout)
		assert.Equal(t, 5*time.Second, got.Timeout.AsDuration())
	})
}

//nolint:dupl
func TestStarlarkEvaluatorFromProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		proto := (*pbApps.StarlarkEvaluator)(nil)
		got := StarlarkEvaluatorFromProto(proto)
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		proto := &pbApps.StarlarkEvaluator{}
		got := StarlarkEvaluatorFromProto(proto)
		want := &StarlarkEvaluator{Code: "", Timeout: 0}
		assert.Equal(t, want, got)
	})

	t.Run("with code", func(t *testing.T) {
		proto := &pbApps.StarlarkEvaluator{
			Source: &pbApps.StarlarkEvaluator_Code{Code: "print('hello')"},
		}
		got := StarlarkEvaluatorFromProto(proto)
		want := &StarlarkEvaluator{Code: "print('hello')", Timeout: 0}
		assert.Equal(t, want, got)
	})

	t.Run("with timeout", func(t *testing.T) {
		proto := &pbApps.StarlarkEvaluator{
			Timeout: durationpb.New(5 * time.Second),
		}
		got := StarlarkEvaluatorFromProto(proto)
		want := &StarlarkEvaluator{Code: "", Timeout: 5 * time.Second}
		assert.Equal(t, want, got)
	})

	t.Run("with code and timeout", func(t *testing.T) {
		proto := &pbApps.StarlarkEvaluator{
			Source:  &pbApps.StarlarkEvaluator_Code{Code: "print('hello')"},
			Timeout: durationpb.New(5 * time.Second),
		}
		got := StarlarkEvaluatorFromProto(proto)
		want := &StarlarkEvaluator{Code: "print('hello')", Timeout: 5 * time.Second}
		assert.Equal(t, want, got)
	})
}

func TestStarlarkEvaluator_ToProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var evaluator *StarlarkEvaluator = nil
		got := evaluator.ToProto()
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with code", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{Code: "print('hello')"}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "print('hello')", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with zero timeout", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{Timeout: 0}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Nil(t, got.Timeout)
	})

	t.Run("with positive timeout", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{Timeout: 5 * time.Second}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.NotNil(t, got.Timeout)
		assert.Equal(t, 5*time.Second, got.Timeout.AsDuration())
	})

	t.Run("with code and timeout", func(t *testing.T) {
		evaluator := &StarlarkEvaluator{Code: "print('hello')", Timeout: 5 * time.Second}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "print('hello')", got.GetCode())
		assert.NotNil(t, got.Timeout)
		assert.Equal(t, 5*time.Second, got.Timeout.AsDuration())
	})
}

func TestExtismEvaluatorFromProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		proto := (*pbApps.ExtismEvaluator)(nil)
		got := ExtismEvaluatorFromProto(proto)
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		proto := &pbApps.ExtismEvaluator{}
		got := ExtismEvaluatorFromProto(proto)
		want := &ExtismEvaluator{Code: "", Entrypoint: ""}
		assert.Equal(t, want, got)
	})

	t.Run("with code", func(t *testing.T) {
		proto := &pbApps.ExtismEvaluator{
			Source: &pbApps.ExtismEvaluator_Code{Code: "base64content"},
		}
		got := ExtismEvaluatorFromProto(proto)
		want := &ExtismEvaluator{Code: "base64content", Entrypoint: ""}
		assert.Equal(t, want, got)
	})

	t.Run("with entrypoint", func(t *testing.T) {
		proto := &pbApps.ExtismEvaluator{
			Entrypoint: proto.String("handle_request"),
		}
		got := ExtismEvaluatorFromProto(proto)
		want := &ExtismEvaluator{Code: "", Entrypoint: "handle_request"}
		assert.Equal(t, want, got)
	})

	t.Run("with code and entrypoint", func(t *testing.T) {
		proto := &pbApps.ExtismEvaluator{
			Source:     &pbApps.ExtismEvaluator_Code{Code: "base64content"},
			Entrypoint: proto.String("handle_request"),
		}
		got := ExtismEvaluatorFromProto(proto)
		want := &ExtismEvaluator{Code: "base64content", Entrypoint: "handle_request"}
		assert.Equal(t, want, got)
	})
}

func TestExtismEvaluator_ToProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var evaluator *ExtismEvaluator = nil
		got := evaluator.ToProto()
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		evaluator := &ExtismEvaluator{}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Equal(t, "", got.GetEntrypoint())
	})

	t.Run("with code", func(t *testing.T) {
		evaluator := &ExtismEvaluator{Code: "base64content"}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "base64content", got.GetCode())
		assert.Equal(t, "", got.GetEntrypoint())
	})

	t.Run("with entrypoint", func(t *testing.T) {
		evaluator := &ExtismEvaluator{Entrypoint: "handle_request"}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "", got.GetCode())
		assert.Equal(t, "handle_request", got.GetEntrypoint())
	})

	t.Run("with code and entrypoint", func(t *testing.T) {
		evaluator := &ExtismEvaluator{Code: "base64content", Entrypoint: "handle_request"}
		got := evaluator.ToProto()
		assert.NotNil(t, got)
		assert.Equal(t, "base64content", got.GetCode())
		assert.Equal(t, "handle_request", got.GetEntrypoint())
	})
}

func TestEvaluatorFromProto(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		proto := (*pbApps.ScriptApp)(nil)
		got, err := EvaluatorFromProto(proto)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		proto := &pbApps.ScriptApp{}
		got, err := EvaluatorFromProto(proto)
		require.Error(t, err)
		assert.Nil(t, got)
	})

	t.Run("risor", func(t *testing.T) {
		proto := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{
					Source: &pbApps.RisorEvaluator_Code{Code: "print('hello')"},
				},
			},
		}
		got, err := EvaluatorFromProto(proto)
		require.NoError(t, err)
		want := &RisorEvaluator{
			Code: "print('hello')",
		}
		assert.Equal(t, want, got)
	})

	t.Run("starlark", func(t *testing.T) {
		proto := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Starlark{
				Starlark: &pbApps.StarlarkEvaluator{
					Source: &pbApps.StarlarkEvaluator_Code{Code: "print('hello')"},
				},
			},
		}
		got, err := EvaluatorFromProto(proto)
		require.NoError(t, err)
		want := &StarlarkEvaluator{
			Code: "print('hello')",
		}
		assert.Equal(t, want, got)
	})

	t.Run("extism", func(t *testing.T) {
		proto := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Extism{
				Extism: &pbApps.ExtismEvaluator{
					Source:     &pbApps.ExtismEvaluator_Code{Code: "base64content"},
					Entrypoint: proto.String("handle_request"),
				},
			},
		}
		got, err := EvaluatorFromProto(proto)
		require.NoError(t, err)
		want := &ExtismEvaluator{
			Code:       "base64content",
			Entrypoint: "handle_request",
		}
		assert.Equal(t, want, got)
	})
}
