package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		endpoint    Endpoint
		errExpected bool
		errContains string // substring that should be in the error message
	}{
		{
			name: "Valid endpoint - no routes",
			endpoint: Endpoint{
				ID:         "endpoint1",
				ListenerID: "listener1",
				Routes:     []routes.Route{},
			},
			errExpected: false,
		},
		{
			name: "Valid endpoint - with routes",
			endpoint: Endpoint{
				ID:         "endpoint2",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Empty ID",
			endpoint: Endpoint{
				ID:         "",
				ListenerID: "listener1",
				Routes:     []routes.Route{},
			},
			errExpected: true,
			errContains: "endpoint ID cannot be empty",
		},
		{
			name: "Empty listener ID",
			endpoint: Endpoint{
				ID:         "endpoint3",
				ListenerID: "",
				Routes:     []routes.Route{},
			},
			errExpected: true,
			errContains: "listener ID cannot be empty",
		},
		{
			name: "Route with missing app ID",
			endpoint: Endpoint{
				ID:         "endpoint4",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
				},
			},
			errExpected: true,
			errContains: "route app ID cannot be empty",
		},
		{
			name: "Route with missing condition",
			endpoint: Endpoint{
				ID:         "endpoint5",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: nil,
					},
				},
			},
			errExpected: true,
			errContains: "route condition",
		},
		{
			name: "Duplicate route conditions",
			endpoint: Endpoint{
				ID:         "endpoint6",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
				},
			},
			errExpected: true,
			errContains: "duplicated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.endpoint.Validate()

			if tc.errExpected {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
