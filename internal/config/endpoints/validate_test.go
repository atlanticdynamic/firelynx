package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/stretchr/testify/assert"
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
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes:      []routes.Route{},
			},
			errExpected: false,
		},
		{
			name: "Valid endpoint - with routes",
			endpoint: Endpoint{
				ID:          "endpoint2",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Empty ID",
			endpoint: Endpoint{
				ID:          "",
				ListenerIDs: []string{"listener1"},
				Routes:      []routes.Route{},
			},
			errExpected: true,
			errContains: "empty ID",
		},
		{
			name: "No listener IDs",
			endpoint: Endpoint{
				ID:          "endpoint3",
				ListenerIDs: []string{},
				Routes:      []routes.Route{},
			},
			errExpected: true,
			errContains: "no listener IDs",
		},
		{
			name: "Route with missing app ID",
			endpoint: Endpoint{
				ID:          "endpoint4",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "",
						Condition: conditions.NewHTTP("/api/v1"),
					},
				},
			},
			errExpected: true,
			errContains: "empty ID",
		},
		{
			name: "Route with missing condition",
			endpoint: Endpoint{
				ID:          "endpoint5",
				ListenerIDs: []string{"listener1"},
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
				ID:          "endpoint6",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/api/v1"),
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
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
