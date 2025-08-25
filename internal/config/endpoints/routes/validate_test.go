package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/require"
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
			errorType:   nil, // No longer checking for specific error type
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
				require.Error(t, err)
				if tc.errorType != nil {
					require.ErrorIs(
						t,
						err, tc.errorType,
						"Expected error to wrap %v",
						tc.errorType,
					)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
