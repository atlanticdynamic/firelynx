package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpoint_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		endpoint    Endpoint
		errExpected bool
		errContains []string
	}{
		{
			name: "Valid Endpoint",
			endpoint: Endpoint{
				ID:          "valid-endpoint",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
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
			},
			errExpected: true,
			errContains: []string{"empty ID"},
		},
		{
			name: "No Listeners",
			endpoint: Endpoint{
				ID:          "endpoint-no-listeners",
				ListenerIDs: []string{},
			},
			errExpected: true,
			errContains: []string{"no listener IDs"},
		},
		{
			name: "Invalid Route",
			endpoint: Endpoint{
				ID:          "endpoint-invalid-route",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID:     "", // Empty AppID
						Condition: nil,
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty ID", "missing required field"},
		},
		{
			name: "Duplicate Route Conditions",
			endpoint: Endpoint{
				ID:          "endpoint-duplicate-routes",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
					},
					{
						AppID: "app2",
						Condition: HTTPPathCondition{
							Path: "/api/v1", // Same path as above
						},
					},
				},
			},
			errExpected: true,
			errContains: []string{"conflict", "duplicated"},
		},
		{
			name: "Multiple Valid Routes",
			endpoint: Endpoint{
				ID:          "endpoint-valid-routes",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
					},
					{
						AppID: "app2",
						Condition: HTTPPathCondition{
							Path: "/api/v2", // Different path
						},
					},
				},
			},
			errExpected: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.endpoint.Validate()

			if tc.errExpected {
				assert.Error(t, err)
				for _, errStr := range tc.errContains {
					assert.Contains(t, err.Error(), errStr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRoute_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		route       Route
		errExpected bool
		errContains []string
	}{
		{
			name: "Valid HTTP Route",
			route: Route{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/v1",
				},
			},
			errExpected: false,
		},
		{
			name: "Valid gRPC Route",
			route: Route{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "service.v1",
				},
			},
			errExpected: false,
		},
		{
			name: "Empty AppID",
			route: Route{
				AppID: "",
				Condition: HTTPPathCondition{
					Path: "/api/v1",
				},
			},
			errExpected: true,
			errContains: []string{"empty ID"},
		},
		{
			name: "Missing Condition",
			route: Route{
				AppID:     "app1",
				Condition: nil,
			},
			errExpected: true,
			errContains: []string{"missing required field", "condition"},
		},
		{
			name: "Invalid Condition Type",
			route: Route{
				AppID: "app1",
				Condition: &invalidCondition{
					condType: "invalid_type",
					value:    "test",
				},
			},
			errExpected: true,
			errContains: []string{"invalid route type"},
		},
		{
			name: "Empty Condition Value",
			route: Route{
				AppID: "app1",
				Condition: &invalidCondition{
					condType: "http_path",
					value:    "", // Empty value
				},
			},
			errExpected: true,
			errContains: []string{"missing required field", "value"},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.route.Validate()

			if tc.errExpected {
				assert.Error(t, err)
				errStr := err.Error()
				for _, contains := range tc.errContains {
					assert.Contains(t, errStr, contains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// invalidCondition is a test implementation of RouteCondition
type invalidCondition struct {
	condType string
	value    string
}

func (c *invalidCondition) Type() string {
	return c.condType
}

func (c *invalidCondition) Value() string {
	return c.value
}

func TestValidateErrorJoining(t *testing.T) {
	t.Parallel()

	// Test that an endpoint with multiple invalid routes correctly joins all errors
	endpoint := Endpoint{
		ID:          "",         // Invalid: empty ID
		ListenerIDs: []string{}, // Invalid: no listeners
		Routes: []Route{
			{
				AppID:     "",  // Invalid: empty AppID
				Condition: nil, // Invalid: nil condition
			},
			{
				AppID: "app2",
				Condition: &invalidCondition{
					condType: "invalid_type", // Invalid: invalid type
					value:    "",             // Invalid: empty value
				},
			},
		},
	}

	err := endpoint.Validate()

	// Verify that err is not nil and contains multiple error messages
	assert.Error(t, err)

	// Check for specific errors in the joined error
	errorTexts := []string{
		"empty ID",
		"no listener IDs",
		"missing required field",
		"invalid route type",
	}

	for _, text := range errorTexts {
		assert.Contains(t, err.Error(), text)
	}

	// Also test that errors.Is works correctly with the joined errors
	assert.ErrorIs(t, err, ErrEmptyID)
	assert.ErrorIs(t, err, ErrMissingRequiredField)
	assert.ErrorIs(t, err, ErrInvalidRouteType)
}
