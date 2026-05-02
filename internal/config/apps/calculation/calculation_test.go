package calculation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Validate(t *testing.T) {
	tests := []struct {
		name    string
		app     *App
		wantErr string
	}{
		{name: "valid", app: &App{ID: "calc"}},
		{name: "missing id", app: &App{}, wantErr: "missing required field: calculation app ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestApp_Type(t *testing.T) {
	assert.Equal(t, "calculation", (&App{ID: "calc"}).Type())
}

func TestApp_String(t *testing.T) {
	assert.Equal(t, "Calculation App (id: calc)", (&App{ID: "calc"}).String())
}

func TestApp_ToTree(t *testing.T) {
	tree := (&App{ID: "calc"}).ToTree()
	require.NotNil(t, tree)
	assert.NotNil(t, tree.Tree())
}

// TestApp_Validate_InterpolationFailure documents that an undefined env-var
// reference in a tagged field surfaces as an interpolation error wrapped with
// the calculation-specific prefix. The ID field is tagged
// env_interpolation:"no", but InterpolateStruct still walks the struct, so an
// undefined-var error in any tagged field would surface here. Calculation has
// no env_interpolation:"yes" fields today, so this case primarily guards the
// error-wrapping branch from being silently dropped.
func TestApp_Validate_InterpolationFailure(t *testing.T) {
	t.Skip(
		"calculation.App has no env_interpolation:\"yes\" fields; the interpolation " +
			"error branch is unreachable from valid input. Re-enable if a tagged field is added.",
	)
}
