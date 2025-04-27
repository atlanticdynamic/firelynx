package config

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// stringPtr returns a pointer to a string value
func stringPtr(s string) *string {
	return &s
}

// mergeModePtrValue converts a StaticDataMergeMode enum to a pointer
func mergeModePtrValue(mode pb.StaticDataMergeMode) *pb.StaticDataMergeMode {
	m := mode
	return &m
}

// mustStructValue creates a structpb.Value from an interface{} and fails the test on error
func mustStructValue(t *testing.T, v interface{}) *structpb.Value {
	t.Helper()
	val, err := structpb.NewValue(v)
	require.NoError(t, err, "Failed to create structpb.Value")
	return val
}

func TestConfig_ToProto(t *testing.T) {
	// Create a comprehensive domain config with all types of elements
	cfg := &Config{
		Version: "v1",
		Logging: LoggingConfig{
			Format: LogFormatJSON,
			Level:  LogLevelInfo,
		},
		Listeners: []Listener{
			{
				ID:      "http-listener",
				Address: ":8080",
				Type:    ListenerTypeHTTP,
				Options: HTTPListenerOptions{
					ReadTimeout:  durationpb.New(30 * time.Second),
					WriteTimeout: durationpb.New(45 * time.Second),
					DrainTimeout: durationpb.New(60 * time.Second),
				},
			},
			{
				ID:      "grpc-listener",
				Address: ":50051",
				Type:    ListenerTypeGRPC,
				Options: GRPCListenerOptions{
					MaxConnectionIdle:    durationpb.New(10 * time.Minute),
					MaxConnectionAge:     durationpb.New(1 * time.Hour),
					MaxConcurrentStreams: 100,
				},
			},
		},
		Endpoints: []Endpoint{
			{
				ID:          "endpoint1",
				ListenerIDs: []string{"http-listener"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/resource",
						},
						StaticData: map[string]any{
							"string_value": "value",
							"int_value":    123,
							"bool_value":   true,
							"null_value":   nil,
							"list_value":   []any{"item1", 42, true},
							"map_value": map[string]any{
								"nested": "object",
							},
						},
					},
				},
			},
			{
				ID:          "endpoint2",
				ListenerIDs: []string{"grpc-listener"},
				Routes: []Route{
					{
						AppID: "app2",
						Condition: GRPCServiceCondition{
							Service: "example.Service",
						},
					},
				},
			},
		},
		Apps: []App{
			{
				ID: "app1",
				Config: ScriptApp{
					StaticData: StaticData{
						Data: map[string]any{
							"app_name": "Test App",
							"version":  "1.0",
						},
						MergeMode: StaticDataMergeModeLast,
					},
					Evaluator: RisorEvaluator{
						Code:    "function handler() { return true; }",
						Timeout: durationpb.New(5 * time.Second),
					},
				},
			},
			{
				ID: "app2",
				Config: ScriptApp{
					StaticData: StaticData{
						Data: map[string]any{
							"mode": "test",
						},
						MergeMode: StaticDataMergeModeUnique,
					},
					Evaluator: StarlarkEvaluator{
						Code:    "def handler(req): return True",
						Timeout: durationpb.New(3 * time.Second),
					},
				},
			},
			{
				ID: "app3",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "(module)",
						Entrypoint: "handle_request",
					},
				},
			},
			{
				ID: "app4",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"app1", "app2"},
					StaticData: StaticData{
						Data: map[string]any{
							"composite": true,
							"priority":  1,
						},
						MergeMode: StaticDataMergeModeUnique,
					},
				},
			},
		},
	}

	// Convert to protobuf
	pbConfig := cfg.ToProto()
	require.NotNil(t, pbConfig, "Protobuf config should not be nil")

	// Verify top-level fields
	assert.Equal(t, "v1", *pbConfig.Version, "Version should match")

	// Verify logging
	require.NotNil(t, pbConfig.Logging, "Logging should not be nil")
	assert.Equal(
		t,
		int32(pb.LogFormat_LOG_FORMAT_JSON),
		int32(pbConfig.Logging.GetFormat()),
		"Log format should be JSON",
	)
	assert.Equal(
		t,
		int32(pb.LogLevel_LOG_LEVEL_INFO),
		int32(pbConfig.Logging.GetLevel()),
		"Log level should be INFO",
	)

	// Verify listeners
	require.Len(t, pbConfig.Listeners, 2, "Should have 2 listeners")

	// Verify HTTP listener
	httpListener := pbConfig.Listeners[0]
	assert.Equal(t, "http-listener", *httpListener.Id, "HTTP listener ID should match")
	assert.Equal(t, ":8080", *httpListener.Address, "HTTP listener address should match")
	httpOpts := httpListener.GetHttp()
	require.NotNil(t, httpOpts, "HTTP options should not be nil")
	assert.Equal(t, int64(30), httpOpts.ReadTimeout.GetSeconds(), "HTTP read timeout should match")
	assert.Equal(
		t,
		int64(45),
		httpOpts.WriteTimeout.GetSeconds(),
		"HTTP write timeout should match",
	)
	assert.Equal(
		t,
		int64(60),
		httpOpts.DrainTimeout.GetSeconds(),
		"HTTP drain timeout should match",
	)

	// Verify gRPC listener
	grpcListener := pbConfig.Listeners[1]
	assert.Equal(t, "grpc-listener", *grpcListener.Id, "gRPC listener ID should match")
	assert.Equal(t, ":50051", *grpcListener.Address, "gRPC listener address should match")
	grpcOpts := grpcListener.GetGrpc()
	require.NotNil(t, grpcOpts, "gRPC options should not be nil")
	assert.Equal(
		t,
		int64(600),
		grpcOpts.MaxConnectionIdle.GetSeconds(),
		"gRPC max idle should match",
	)
	assert.Equal(
		t,
		int64(3600),
		grpcOpts.MaxConnectionAge.GetSeconds(),
		"gRPC max age should match",
	)
	assert.Equal(t, int32(100), *grpcOpts.MaxConcurrentStreams, "gRPC max streams should match")

	// Verify endpoints
	require.Len(t, pbConfig.Endpoints, 2, "Should have 2 endpoints")

	// Verify first endpoint
	endpoint1 := pbConfig.Endpoints[0]
	assert.Equal(t, "endpoint1", *endpoint1.Id, "Endpoint 1 ID should match")
	assert.Equal(
		t,
		[]string{"http-listener"},
		endpoint1.ListenerIds,
		"Endpoint 1 listener IDs should match",
	)
	require.Len(t, endpoint1.Routes, 1, "Endpoint 1 should have 1 route")

	// Verify first route
	route1 := endpoint1.Routes[0]
	assert.Equal(t, "app1", *route1.AppId, "Route 1 app ID should match")
	assert.Equal(t, "/api/resource", route1.GetHttpPath(), "Route 1 HTTP path should match")

	// Verify route static data
	require.NotNil(t, route1.StaticData, "Route 1 static data should not be nil")
	assert.Equal(
		t,
		"value",
		route1.StaticData.Data["string_value"].GetStringValue(),
		"String value should match",
	)
	assert.Equal(
		t,
		float64(123),
		route1.StaticData.Data["int_value"].GetNumberValue(),
		"Number value should match",
	)
	assert.Equal(
		t,
		true,
		route1.StaticData.Data["bool_value"].GetBoolValue(),
		"Bool value should match",
	)
	assert.NotNil(
		t,
		route1.StaticData.Data["null_value"].GetNullValue(),
		"Null value should be set",
	)

	// Verify list value
	listValue := route1.StaticData.Data["list_value"].GetListValue()
	require.NotNil(t, listValue, "List value should not be nil")
	require.Len(t, listValue.Values, 3, "List should have 3 items")
	assert.Equal(t, "item1", listValue.Values[0].GetStringValue(), "First list item should match")
	assert.Equal(
		t,
		float64(42),
		listValue.Values[1].GetNumberValue(),
		"Second list item should match",
	)
	assert.Equal(t, true, listValue.Values[2].GetBoolValue(), "Third list item should match")

	// Verify map value
	mapValue := route1.StaticData.Data["map_value"].GetStructValue()
	require.NotNil(t, mapValue, "Map value should not be nil")
	assert.Equal(
		t,
		"object",
		mapValue.Fields["nested"].GetStringValue(),
		"Nested map value should match",
	)

	// Verify second endpoint
	endpoint2 := pbConfig.Endpoints[1]
	assert.Equal(t, "endpoint2", *endpoint2.Id, "Endpoint 2 ID should match")
	assert.Equal(
		t,
		[]string{"grpc-listener"},
		endpoint2.ListenerIds,
		"Endpoint 2 listener IDs should match",
	)
	require.Len(t, endpoint2.Routes, 1, "Endpoint 2 should have 1 route")

	// Verify second route
	route2 := endpoint2.Routes[0]
	assert.Equal(t, "app2", *route2.AppId, "Route 2 app ID should match")
	assert.Equal(t, "example.Service", route2.GetGrpcService(), "Route 2 gRPC service should match")

	// Verify apps
	require.Len(t, pbConfig.Apps, 4, "Should have 4 apps")

	// Verify Risor app
	app1 := pbConfig.Apps[0]
	assert.Equal(t, "app1", *app1.Id, "App 1 ID should match")
	app1Script := app1.GetScript()
	require.NotNil(t, app1Script, "App 1 script should not be nil")
	app1Risor := app1Script.GetRisor()
	require.NotNil(t, app1Risor, "App 1 Risor evaluator should not be nil")
	assert.Equal(
		t,
		"function handler() { return true; }",
		*app1Risor.Code,
		"App 1 code should match",
	)
	assert.Equal(t, int64(5), app1Risor.Timeout.GetSeconds(), "App 1 timeout should match")

	// Verify static data and merge mode
	require.NotNil(t, app1Script.StaticData, "App 1 static data should not be nil")
	assert.Equal(
		t,
		"Test App",
		app1Script.StaticData.Data["app_name"].GetStringValue(),
		"App 1 name should match",
	)
	assert.Equal(
		t,
		"1.0",
		app1Script.StaticData.Data["version"].GetStringValue(),
		"App 1 version should match",
	)
	assert.Equal(
		t,
		int32(pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST),
		int32(app1Script.StaticData.GetMergeMode()),
		"App 1 merge mode should be LAST",
	)

	// Verify Starlark app
	app2 := pbConfig.Apps[1]
	assert.Equal(t, "app2", *app2.Id, "App 2 ID should match")
	app2Script := app2.GetScript()
	require.NotNil(t, app2Script, "App 2 script should not be nil")
	app2Starlark := app2Script.GetStarlark()
	require.NotNil(t, app2Starlark, "App 2 Starlark evaluator should not be nil")
	assert.Equal(t, "def handler(req): return True", *app2Starlark.Code, "App 2 code should match")
	assert.Equal(t, int64(3), app2Starlark.Timeout.GetSeconds(), "App 2 timeout should match")
	assert.Equal(
		t,
		int32(pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE),
		int32(app2Script.StaticData.GetMergeMode()),
		"App 2 merge mode should be UNIQUE",
	)

	// Verify Extism app
	app3 := pbConfig.Apps[2]
	assert.Equal(t, "app3", *app3.Id, "App 3 ID should match")
	app3Script := app3.GetScript()
	require.NotNil(t, app3Script, "App 3 script should not be nil")
	app3Extism := app3Script.GetExtism()
	require.NotNil(t, app3Extism, "App 3 Extism evaluator should not be nil")
	assert.Equal(t, "(module)", *app3Extism.Code, "App 3 code should match")
	assert.Equal(t, "handle_request", *app3Extism.Entrypoint, "App 3 entrypoint should match")

	// Verify composite app
	app4 := pbConfig.Apps[3]
	assert.Equal(t, "app4", *app4.Id, "App 4 ID should match")
	app4Composite := app4.GetCompositeScript()
	require.NotNil(t, app4Composite, "App 4 composite script should not be nil")
	assert.Equal(
		t,
		[]string{"app1", "app2"},
		app4Composite.ScriptAppIds,
		"App 4 script app IDs should match",
	)
	assert.Equal(
		t,
		true,
		app4Composite.StaticData.Data["composite"].GetBoolValue(),
		"App 4 composite flag should match",
	)
	assert.Equal(
		t,
		float64(1),
		app4Composite.StaticData.Data["priority"].GetNumberValue(),
		"App 4 priority should match",
	)
	assert.Equal(
		t,
		int32(pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE),
		int32(app4Composite.StaticData.GetMergeMode()),
		"App 4 merge mode should be UNIQUE",
	)
}

func TestConfig_ToProto_WithRawProto(t *testing.T) {
	// Test the optimization where a stored raw proto is returned directly
	version := "v2"
	rawProto := &pb.ServerConfig{
		Version: &version,
	}

	cfg := &Config{
		Version:  "v1", // Different from raw proto to verify which one is returned
		rawProto: rawProto,
	}

	result := cfg.ToProto()

	// Should return the stored raw proto, not create a new one
	assert.Same(t, rawProto, result, "Should return the stored raw proto directly")
	assert.Equal(t, "v2", *result.Version, "Should have the version from raw proto")
}

func TestConfig_ToProto_EmptyConfig(t *testing.T) {
	// Test with an empty config
	cfg := &Config{}
	result := cfg.ToProto()

	require.NotNil(t, result, "Result should not be nil even for empty config")
	assert.Equal(t, "", *result.Version, "Empty config should have empty version")
	assert.Nil(t, result.Logging, "Empty config should have nil logging")
	assert.Empty(t, result.Listeners, "Empty config should have no listeners")
	assert.Empty(t, result.Endpoints, "Empty config should have no endpoints")
	assert.Empty(t, result.Apps, "Empty config should have no apps")
}

func TestConvertProtoValueToInterface(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result := convertProtoValueToInterface(nil)
		assert.Nil(t, result, "nil proto value should convert to nil interface")
	})

	t.Run("null value", func(t *testing.T) {
		nullVal, err := structpb.NewValue(nil)
		require.NoError(t, err)

		result := convertProtoValueToInterface(nullVal)
		assert.Nil(t, result, "null proto value should convert to nil interface")
	})

	t.Run("number value", func(t *testing.T) {
		// Test integer
		intVal, err := structpb.NewValue(42)
		require.NoError(t, err)

		result := convertProtoValueToInterface(intVal)
		assert.Equal(t, float64(42), result, "integer proto value should convert to float64")

		// Test float
		floatVal, err := structpb.NewValue(42.5)
		require.NoError(t, err)

		result = convertProtoValueToInterface(floatVal)
		assert.Equal(t, 42.5, result, "float proto value should convert to float64")
	})

	t.Run("string value", func(t *testing.T) {
		strVal, err := structpb.NewValue("test string")
		require.NoError(t, err)

		result := convertProtoValueToInterface(strVal)
		assert.Equal(t, "test string", result, "string proto value should convert to string")

		// Test empty string
		emptyStrVal, err := structpb.NewValue("")
		require.NoError(t, err)

		result = convertProtoValueToInterface(emptyStrVal)
		assert.Equal(t, "", result, "empty string proto value should convert to empty string")
	})

	t.Run("bool value", func(t *testing.T) {
		// Test true
		trueVal, err := structpb.NewValue(true)
		require.NoError(t, err)

		result := convertProtoValueToInterface(trueVal)
		assert.Equal(t, true, result, "bool true proto value should convert to true")

		// Test false
		falseVal, err := structpb.NewValue(false)
		require.NoError(t, err)

		result = convertProtoValueToInterface(falseVal)
		assert.Equal(t, false, result, "bool false proto value should convert to false")
	})

	t.Run("list value", func(t *testing.T) {
		// Test empty list
		emptyList, err := structpb.NewValue([]any{})
		require.NoError(t, err)

		result := convertProtoValueToInterface(emptyList)
		assert.Equal(t, []any{}, result, "empty list proto value should convert to empty slice")

		// Test list with mixed types
		mixedList, err := structpb.NewValue([]any{
			"string",
			42,
			true,
			nil,
			[]any{"nested"},
			map[string]any{"key": "value"},
		})
		require.NoError(t, err)

		result = convertProtoValueToInterface(mixedList)
		resultList, ok := result.([]any)
		require.True(t, ok, "should convert to []any")
		require.Len(t, resultList, 6, "converted list should have 6 elements")

		assert.Equal(t, "string", resultList[0], "first element should be string")
		assert.Equal(t, float64(42), resultList[1], "second element should be float64(42)")
		assert.Equal(t, true, resultList[2], "third element should be true")
		assert.Nil(t, resultList[3], "fourth element should be nil")

		// Check nested list
		nestedList, ok := resultList[4].([]any)
		require.True(t, ok, "fifth element should be []any")
		assert.Equal(t, "nested", nestedList[0], "nested element should be 'nested'")

		// Check nested map
		nestedMap, ok := resultList[5].(map[string]any)
		require.True(t, ok, "sixth element should be map[string]any")
		assert.Equal(t, "value", nestedMap["key"], "nested map value should be 'value'")
	})

	t.Run("struct value", func(t *testing.T) {
		// Test empty struct
		emptyStruct, err := structpb.NewValue(map[string]any{})
		require.NoError(t, err)

		result := convertProtoValueToInterface(emptyStruct)
		assert.Equal(
			t,
			map[string]any{},
			result,
			"empty struct proto value should convert to empty map",
		)

		// Test struct with mixed types
		mixedStruct, err := structpb.NewValue(map[string]any{
			"string": "value",
			"number": 42,
			"bool":   true,
			"null":   nil,
			"list":   []any{"item"},
			"map":    map[string]any{"nested": "object"},
		})
		require.NoError(t, err)

		result = convertProtoValueToInterface(mixedStruct)
		resultMap, ok := result.(map[string]any)
		require.True(t, ok, "should convert to map[string]any")

		assert.Equal(t, "value", resultMap["string"], "string field should be 'value'")
		assert.Equal(t, float64(42), resultMap["number"], "number field should be float64(42)")
		assert.Equal(t, true, resultMap["bool"], "bool field should be true")
		assert.Nil(t, resultMap["null"], "null field should be nil")

		// Check nested list
		nestedList, ok := resultMap["list"].([]any)
		require.True(t, ok, "list field should be []any")
		assert.Equal(t, "item", nestedList[0], "nested list item should be 'item'")

		// Check nested map
		nestedMap, ok := resultMap["map"].(map[string]any)
		require.True(t, ok, "map field should be map[string]any")
		assert.Equal(t, "object", nestedMap["nested"], "nested map value should be 'object'")
	})

	t.Run("deep nesting", func(t *testing.T) {
		// Create a deeply nested structure to test recursion
		deeplyNested, err := structpb.NewValue(map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"value": "deep",
						},
						"array": []any{
							map[string]any{
								"nested": "array item",
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		result := convertProtoValueToInterface(deeplyNested)
		resultMap, ok := result.(map[string]any)
		require.True(t, ok, "should convert to map[string]any")

		// Navigate to deeply nested value
		level1, ok := resultMap["level1"].(map[string]any)
		require.True(t, ok, "level1 should be map")

		level2, ok := level1["level2"].(map[string]any)
		require.True(t, ok, "level2 should be map")

		level3, ok := level2["level3"].(map[string]any)
		require.True(t, ok, "level3 should be map")

		level4, ok := level3["level4"].(map[string]any)
		require.True(t, ok, "level4 should be map")

		assert.Equal(t, "deep", level4["value"], "deeply nested value should be preserved")

		// Check the nested array
		array, ok := level3["array"].([]any)
		require.True(t, ok, "array should be []any")

		arrayItem, ok := array[0].(map[string]any)
		require.True(t, ok, "array item should be map")

		assert.Equal(
			t,
			"array item",
			arrayItem["nested"],
			"nested array item value should be preserved",
		)
	})
}

// deeplyEqualWithNumericNormalization compares two values while handling int/float64 conversions
func deeplyEqualWithNumericNormalization(t *testing.T, a, b any) bool {
	t.Helper()
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch av := a.(type) {
	case int:
		// Convert int to float64 for comparison
		if bv, ok := b.(float64); ok {
			return float64(av) == bv
		}
	case float64:
		// Convert int to float64 for comparison
		if bv, ok := b.(int); ok {
			return av == float64(bv)
		}
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i, v := range av {
			if !deeplyEqualWithNumericNormalization(t, v, bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bval, exists := bv[k]
			if !exists || !deeplyEqualWithNumericNormalization(t, v, bval) {
				return false
			}
		}
		return true
	}

	// Use normal equality for other types
	return a == b
}

func TestConvertProtoValueToInterfaceRoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		value any
	}{
		{"null", nil},
		{"string", "test string"},
		{"empty string", ""},
		{"integer", 42},
		{"float", 42.5},
		{"true", true},
		{"false", false},
		{"empty list", []any{}},
		{"simple list", []any{"item1", "item2"}},
		{"empty map", map[string]any{}},
		{"simple map", map[string]any{"key": "value"}},
		{"complex object", map[string]any{
			"string": "value",
			"number": 42,
			"bool":   true,
			"null":   nil,
			"list":   []any{"item1", 42, true, nil},
			"map":    map[string]any{"nested": "value"},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Go value to proto value
			protoValue, err := structpb.NewValue(tc.value)
			require.NoError(t, err, "Failed to convert to proto value")

			// Proto value back to Go value
			goValue := convertProtoValueToInterface(protoValue)

			// Compare original and round-tripped values
			if tc.value == nil {
				assert.Nil(t, goValue, "Round-tripped nil value should be nil")
			} else {
				// Use deeply equal comparison with numeric type normalization
				assert.True(t, deeplyEqualWithNumericNormalization(t, tc.value, goValue),
					"Round-tripped value should match original after numeric normalization")
			}
		})
	}
}

func TestAppFromProto(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		app, err := appFromProto(nil)
		assert.Error(t, err, "Should return error for nil app")
		assert.ErrorIs(
			t,
			err,
			ErrFailedToConvertConfig,
			"Error should wrap ErrFailedToConvertConfig",
		)
		assert.Empty(t, app.ID, "App ID should be empty")
	})

	t.Run("empty app", func(t *testing.T) {
		pbApp := &pb.AppDefinition{}
		app, err := appFromProto(pbApp)
		assert.Error(t, err, "Should return error for empty app")
		assert.ErrorIs(
			t,
			err,
			ErrFailedToConvertConfig,
			"Error should wrap ErrFailedToConvertConfig",
		)
		assert.Empty(t, app.ID, "App ID should be empty")
		assert.Contains(
			t,
			err.Error(),
			"unknown or empty config type",
			"Error message should mention empty config type",
		)
	})

	t.Run("script with missing evaluator", func(t *testing.T) {
		// Create protobuf app with script but no evaluator
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-missing-eval"),
			AppConfig: &pb.AppDefinition_Script{
				Script: &pb.AppScript{
					StaticData: &pb.StaticData{
						Data: map[string]*structpb.Value{},
					},
					// No evaluator defined
				},
			},
		}

		// Convert to domain model
		app, err := appFromProto(pbApp)
		assert.Error(t, err, "Should return error for missing evaluator")
		assert.ErrorIs(
			t,
			err,
			ErrFailedToConvertConfig,
			"Error should wrap ErrFailedToConvertConfig",
		)
		assert.Contains(
			t,
			err.Error(),
			"no evaluator defined",
			"Error message should mention missing evaluator",
		)
		assert.Empty(
			t,
			app.ID,
			"App ID should be empty as appFromProto returns an empty App on error",
		)
		assert.Contains(
			t,
			err.Error(),
			"test-missing-eval",
			"Error message should contain the app ID",
		)
	})

	t.Run("composite script app", func(t *testing.T) {
		// Create protobuf app with composite script
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-composite"),
			AppConfig: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
					ScriptAppIds: []string{"app1", "app2"},
					StaticData: &pb.StaticData{
						Data: map[string]*structpb.Value{
							"composite": mustStructValue(t, true),
							"priority":  mustStructValue(t, 1),
						},
						MergeMode: mergeModePtrValue(
							pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
						),
					},
				},
			},
		}

		// Convert to domain model
		app, err := appFromProto(pbApp)
		require.NoError(t, err, "Should convert composite app without error")
		assert.Equal(t, "test-composite", app.ID, "App ID should match")

		// Verify CompositeScriptApp type
		compositeApp, ok := app.Config.(CompositeScriptApp)
		require.True(t, ok, "App config should be CompositeScriptApp")

		// Verify script app IDs
		assert.Equal(
			t,
			[]string{"app1", "app2"},
			compositeApp.ScriptAppIDs,
			"Script app IDs should match",
		)

		// Verify static data
		assert.Equal(
			t,
			true,
			compositeApp.StaticData.Data["composite"],
			"Static data should be converted correctly",
		)
		assert.Equal(
			t,
			float64(1),
			compositeApp.StaticData.Data["priority"],
			"Static data should be converted correctly",
		)
		assert.Equal(
			t,
			StaticDataMergeModeUnique,
			compositeApp.StaticData.MergeMode,
			"Merge mode should be UNIQUE",
		)
	})

	t.Run("composite script with nil static data", func(t *testing.T) {
		// Create protobuf app with composite script but nil static data
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-composite-nil-static"),
			AppConfig: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
					ScriptAppIds: []string{"app1", "app2"},
					// No static data defined
				},
			},
		}

		// Convert to domain model
		app, err := appFromProto(pbApp)
		require.NoError(t, err, "Should convert composite app without error")

		// Verify CompositeScriptApp type
		compositeApp, ok := app.Config.(CompositeScriptApp)
		require.True(t, ok, "App config should be CompositeScriptApp")

		// Verify no static data
		assert.Nil(t, compositeApp.StaticData.Data, "Static data should be nil")
		assert.Equal(
			t,
			StaticDataMergeModeUnspecified,
			compositeApp.StaticData.MergeMode,
			"Default merge mode should be UNSPECIFIED",
		)
	})

	// Table-driven test for script evaluators
	scriptEvalTests := []struct {
		name              string
		appID             string
		evalType          string
		code              string
		entrypoint        string // Only used for Extism
		timeout           time.Duration
		staticDataKey     string
		staticDataValue   any
		mergeMode         pb.StaticDataMergeMode
		expectedMergeMode StaticDataMergeMode
	}{
		{
			name:              "risor evaluator",
			appID:             "test-risor",
			evalType:          "risor",
			code:              "function handler() { return true; }",
			timeout:           5 * time.Second,
			staticDataKey:     "key",
			staticDataValue:   "value",
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
			expectedMergeMode: StaticDataMergeModeLast,
		},
		{
			name:              "starlark evaluator",
			appID:             "test-starlark",
			evalType:          "starlark",
			code:              "def handler(req): return True",
			timeout:           3 * time.Second,
			staticDataKey:     "mode",
			staticDataValue:   "test",
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
			expectedMergeMode: StaticDataMergeModeUnique,
		},
		{
			name:              "extism evaluator",
			appID:             "test-extism",
			evalType:          "extism",
			code:              "(module)",
			entrypoint:        "handle_request",
			staticDataKey:     "format",
			staticDataValue:   "wasm",
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
			expectedMergeMode: StaticDataMergeModeUnspecified,
		},
	}

	for _, tt := range scriptEvalTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create protobuf app with appropriate evaluator
			pbApp := &pb.AppDefinition{
				Id: stringPtr(tt.appID),
				AppConfig: &pb.AppDefinition_Script{
					Script: &pb.AppScript{
						StaticData: &pb.StaticData{
							Data: map[string]*structpb.Value{
								tt.staticDataKey: mustStructValue(t, tt.staticDataValue),
							},
							MergeMode: mergeModePtrValue(tt.mergeMode),
						},
					},
				},
			}

			// Set the evaluator based on the test case type
			script := pbApp.GetScript()
			switch tt.evalType {
			case "risor":
				code := tt.code
				script.Evaluator = &pb.AppScript_Risor{
					Risor: &pb.RisorEvaluator{
						Code:    &code,
						Timeout: durationpb.New(tt.timeout),
					},
				}
			case "starlark":
				code := tt.code
				script.Evaluator = &pb.AppScript_Starlark{
					Starlark: &pb.StarlarkEvaluator{
						Code:    &code,
						Timeout: durationpb.New(tt.timeout),
					},
				}
			case "extism":
				code := tt.code
				entrypoint := tt.entrypoint
				script.Evaluator = &pb.AppScript_Extism{
					Extism: &pb.ExtismEvaluator{
						Code:       &code,
						Entrypoint: &entrypoint,
					},
				}
			}

			// Convert to domain model
			app, err := appFromProto(pbApp)
			require.NoError(t, err, "Should convert %s app without error", tt.evalType)
			assert.Equal(t, tt.appID, app.ID, "App ID should match")

			// Verify ScriptApp type
			scriptApp, ok := app.Config.(ScriptApp)
			require.True(t, ok, "App config should be ScriptApp")

			// Verify the evaluator based on the type
			switch tt.evalType {
			case "risor":
				eval, ok := scriptApp.Evaluator.(RisorEvaluator)
				require.True(t, ok, "Evaluator should be RisorEvaluator")
				assert.Equal(t, tt.code, eval.Code, "Code should match")
				assert.Equal(t, durationpb.New(tt.timeout), eval.Timeout, "Timeout should match")
			case "starlark":
				eval, ok := scriptApp.Evaluator.(StarlarkEvaluator)
				require.True(t, ok, "Evaluator should be StarlarkEvaluator")
				assert.Equal(t, tt.code, eval.Code, "Code should match")
				assert.Equal(t, durationpb.New(tt.timeout), eval.Timeout, "Timeout should match")
			case "extism":
				eval, ok := scriptApp.Evaluator.(ExtismEvaluator)
				require.True(t, ok, "Evaluator should be ExtismEvaluator")
				assert.Equal(t, tt.code, eval.Code, "Code should match")
				assert.Equal(t, tt.entrypoint, eval.Entrypoint, "Entrypoint should match")
			}

			// Verify static data
			assert.Equal(
				t,
				tt.staticDataValue, scriptApp.StaticData.Data[tt.staticDataKey],
				"Static data should be converted correctly",
			)
			assert.Equal(
				t,
				tt.expectedMergeMode, scriptApp.StaticData.MergeMode,
				"Merge mode should match expected value",
			)
		})
	}
}
