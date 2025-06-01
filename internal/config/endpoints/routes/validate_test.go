package routes

import (
	"errors"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
)

func TestRoute_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		route       Route
		expectError bool
		errorType   error
	}{
		{
			name: "Valid HTTP route",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			expectError: false,
		},
		{
			name: "Valid HTTP route with method",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v2", "POST"),
			},
			expectError: false,
		},
		{
			name: "Empty app ID",
			route: Route{
				AppID:     "",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			expectError: true,
			errorType:   ErrEmptyID,
		},
		{
			name: "Nil condition",
			route: Route{
				AppID:     "app1",
				Condition: nil,
			},
			expectError: true,
			errorType:   ErrMissingRequiredField,
		},
		{
			name: "Invalid HTTP path",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("", ""),
			},
			expectError: true,
		},
		{
			name: "Invalid HTTP path",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewHTTP("", ""),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.route.Validate()
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorType != nil {
					assert.True(
						t,
						errors.Is(err, tc.errorType),
						"Expected error to wrap %v",
						tc.errorType,
					)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
