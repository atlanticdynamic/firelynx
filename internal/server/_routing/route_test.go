package routing

import (
	"reflect"
	"testing"
)

func TestRoute_Clone(t *testing.T) {
	original := Route{
		Path:  "/api/test",
		AppID: "test-app",
		StaticData: map[string]any{
			"version": "1.0",
			"debug":   true,
		},
	}

	// Clone the route
	clone := original.Clone()

	// Check fields match
	if clone.Path != original.Path {
		t.Errorf("Clone path = %v, want %v", clone.Path, original.Path)
	}

	if clone.AppID != original.AppID {
		t.Errorf("Clone appID = %v, want %v", clone.AppID, original.AppID)
	}

	// Check that StaticData has same content but is a different map
	if !reflect.DeepEqual(clone.StaticData, original.StaticData) {
		t.Errorf("Clone staticData = %v, want %v", clone.StaticData, original.StaticData)
	}

	// Verify that modifying clone's StaticData doesn't affect original
	clone.StaticData["version"] = "2.0"
	if original.StaticData["version"] == "2.0" {
		t.Error("Modifying clone's StaticData affected original")
	}
}

func TestEndpointRoutes_Clone(t *testing.T) {
	original := EndpointRoutes{
		EndpointID: "test-endpoint",
		Routes: []Route{
			{
				Path:  "/api/users",
				AppID: "user-app",
				StaticData: map[string]any{
					"version": "1.0",
				},
			},
			{
				Path:  "/api/products",
				AppID: "product-app",
				StaticData: map[string]any{
					"version": "2.0",
				},
			},
		},
	}

	// Clone the endpoint routes
	clone := original.Clone()

	// Check fields match
	if clone.EndpointID != original.EndpointID {
		t.Errorf("Clone endpointID = %v, want %v", clone.EndpointID, original.EndpointID)
	}

	// Check that Routes has same content but is a different slice
	if !reflect.DeepEqual(clone.Routes, original.Routes) {
		t.Errorf("Clone routes = %v, want %v", clone.Routes, original.Routes)
	}

	if len(clone.Routes) != len(original.Routes) {
		t.Errorf("Clone routes length = %v, want %v", len(clone.Routes), len(original.Routes))
		return
	}

	// Verify that modifying clone's Routes doesn't affect original
	clone.Routes[0].Path = "/api/users/modified"
	if original.Routes[0].Path == "/api/users/modified" {
		t.Error("Modifying clone's Routes affected original")
	}
}

func TestRoutingConfig_Clone(t *testing.T) {
	original := &RoutingConfig{
		EndpointRoutes: []EndpointRoutes{
			{
				EndpointID: "test-endpoint-1",
				Routes: []Route{
					{
						Path:  "/api/users",
						AppID: "user-app",
					},
				},
			},
			{
				EndpointID: "test-endpoint-2",
				Routes: []Route{
					{
						Path:  "/api/products",
						AppID: "product-app",
					},
				},
			},
		},
	}

	// Clone the config
	clone := original.Clone()

	// Check that EndpointRoutes has same content but is a different slice
	if !reflect.DeepEqual(clone.EndpointRoutes, original.EndpointRoutes) {
		t.Errorf(
			"Clone endpointRoutes = %v, want %v",
			clone.EndpointRoutes,
			original.EndpointRoutes,
		)
	}

	// Verify that modifying clone's EndpointRoutes doesn't affect original
	clone.EndpointRoutes[0].EndpointID = "modified-endpoint"
	if original.EndpointRoutes[0].EndpointID == "modified-endpoint" {
		t.Error("Modifying clone's EndpointRoutes affected original")
	}
}

func TestRoutingConfig_Clone_Nil(t *testing.T) {
	var original *RoutingConfig = nil
	clone := original.Clone()

	if clone != nil {
		t.Errorf("Clone of nil RoutingConfig should be nil, got %v", clone)
	}
}
