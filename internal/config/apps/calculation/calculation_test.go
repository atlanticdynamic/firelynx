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
