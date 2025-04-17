package config

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDomainConfig_Helpers(t *testing.T) {
	// Create a sample config using domain model
	config := &Config{
		Version: "v1",
		Listeners: []Listener{
			{
				ID: "test_listener",
			},
		},
		Endpoints: []Endpoint{
			{
				ID: "test_endpoint",
			},
		},
		Apps: []App{
			{
				ID: "test_app",
			},
		},
	}

	// Test FindListener
	listener := config.FindListener("test_listener")
	if listener == nil {
		t.Error("FindListener returned nil for existing listener")
	}

	listener = config.FindListener("nonexistent")
	if listener != nil {
		t.Error("FindListener returned non-nil for nonexistent listener")
	}

	// Test FindEndpoint
	endpoint := config.FindEndpoint("test_endpoint")
	if endpoint == nil {
		t.Error("FindEndpoint returned nil for existing endpoint")
	}

	endpoint = config.FindEndpoint("nonexistent")
	if endpoint != nil {
		t.Error("FindEndpoint returned non-nil for nonexistent endpoint")
	}

	// Test FindApp
	app := config.FindApp("test_app")
	if app == nil {
		t.Error("FindApp returned nil for existing app")
	}

	app = config.FindApp("nonexistent")
	if app != nil {
		t.Error("FindApp returned non-nil for nonexistent app")
	}
}

func TestDomainModelConversion(t *testing.T) {
	// Create a domain model config
	domainConfig := createValidDomainConfig()

	// Convert to protobuf
	pbConfig := domainConfig.ToProto()

	// Convert back to domain model
	roundTripConfig := FromProto(pbConfig)

	// Check that the round-trip conversion preserves data
	if roundTripConfig.Version != domainConfig.Version {
		t.Errorf(
			"Version not preserved: got %s, want %s",
			roundTripConfig.Version,
			domainConfig.Version,
		)
	}

	if len(roundTripConfig.Listeners) != len(domainConfig.Listeners) {
		t.Errorf("Listener count not preserved: got %d, want %d",
			len(roundTripConfig.Listeners), len(domainConfig.Listeners))
	}

	if len(roundTripConfig.Endpoints) != len(domainConfig.Endpoints) {
		t.Errorf("Endpoint count not preserved: got %d, want %d",
			len(roundTripConfig.Endpoints), len(domainConfig.Endpoints))
	}

	if len(roundTripConfig.Apps) != len(domainConfig.Apps) {
		t.Errorf("App count not preserved: got %d, want %d",
			len(roundTripConfig.Apps), len(domainConfig.Apps))
	}

	// Check first listener details
	if roundTripConfig.Listeners[0].ID != domainConfig.Listeners[0].ID {
		t.Errorf("Listener ID not preserved: got %s, want %s",
			roundTripConfig.Listeners[0].ID, domainConfig.Listeners[0].ID)
	}

	if roundTripConfig.Listeners[0].Address != domainConfig.Listeners[0].Address {
		t.Errorf("Listener Address not preserved: got %s, want %s",
			roundTripConfig.Listeners[0].Address, domainConfig.Listeners[0].Address)
	}
}

func TestEnumConversion(t *testing.T) {
	// Test LogFormat conversions
	testCases := []struct {
		domainFormat LogFormat
		pbFormat     pb.LogFormat
		strFormat    string
	}{
		{LogFormatJSON, pb.LogFormat_LOG_FORMAT_JSON, "json"},
		{LogFormatText, pb.LogFormat_LOG_FORMAT_TXT, "text"},
		{LogFormatUnspecified, pb.LogFormat_LOG_FORMAT_UNSPECIFIED, ""},
	}

	for _, tc := range testCases {
		t.Run(string(tc.domainFormat), func(t *testing.T) {
			// Domain to protobuf
			pbFormat := logFormatToProto(tc.domainFormat)
			if pbFormat != tc.pbFormat {
				t.Errorf(
					"logFormatToProto(%s) = %v, want %v",
					tc.domainFormat,
					pbFormat,
					tc.pbFormat,
				)
			}

			// Protobuf to domain
			domainFormat := protoFormatToLogFormat(tc.pbFormat)
			if domainFormat != tc.domainFormat {
				t.Errorf(
					"protoFormatToLogFormat(%v) = %s, want %s",
					tc.pbFormat,
					domainFormat,
					tc.domainFormat,
				)
			}

			// String to domain
			format, err := LogFormatFromString(tc.strFormat)
			if err != nil {
				t.Errorf("LogFormatFromString(%s) error: %v", tc.strFormat, err)
			}
			if format != tc.domainFormat {
				t.Errorf(
					"LogFormatFromString(%s) = %s, want %s",
					tc.strFormat,
					format,
					tc.domainFormat,
				)
			}
		})
	}

	// Test LogLevel conversions
	levelTestCases := []struct {
		domainLevel LogLevel
		pbLevel     pb.LogLevel
		strLevel    string
	}{
		{LogLevelDebug, pb.LogLevel_LOG_LEVEL_DEBUG, "debug"},
		{LogLevelInfo, pb.LogLevel_LOG_LEVEL_INFO, "info"},
		{LogLevelWarn, pb.LogLevel_LOG_LEVEL_WARN, "warn"},
		{LogLevelError, pb.LogLevel_LOG_LEVEL_ERROR, "error"},
		{LogLevelFatal, pb.LogLevel_LOG_LEVEL_FATAL, "fatal"},
		{LogLevelUnspecified, pb.LogLevel_LOG_LEVEL_UNSPECIFIED, ""},
	}

	for _, tc := range levelTestCases {
		t.Run(string(tc.domainLevel), func(t *testing.T) {
			// Domain to protobuf
			pbLevel := logLevelToProto(tc.domainLevel)
			if pbLevel != tc.pbLevel {
				t.Errorf("logLevelToProto(%s) = %v, want %v", tc.domainLevel, pbLevel, tc.pbLevel)
			}

			// Protobuf to domain
			domainLevel := protoLevelToLogLevel(tc.pbLevel)
			if domainLevel != tc.domainLevel {
				t.Errorf(
					"protoLevelToLogLevel(%v) = %s, want %s",
					tc.pbLevel,
					domainLevel,
					tc.domainLevel,
				)
			}

			// String to domain
			level, err := LogLevelFromString(tc.strLevel)
			if err != nil {
				t.Errorf("LogLevelFromString(%s) error: %v", tc.strLevel, err)
			}
			if level != tc.domainLevel {
				t.Errorf("LogLevelFromString(%s) = %s, want %s", tc.strLevel, level, tc.domainLevel)
			}
		})
	}
}

// Helper function to create a valid config for testing
func createValidDomainConfig() *Config {
	return &Config{
		Version: "v1",
		Logging: LoggingConfig{
			Format: LogFormatJSON,
			Level:  LogLevelInfo,
		},
		Listeners: []Listener{
			{
				ID:      "listener1",
				Address: ":8080",
				Type:    ListenerTypeHTTP,
				Options: HTTPListenerOptions{
					ReadTimeout:  durationpb.New(time.Second * 30),
					WriteTimeout: durationpb.New(time.Second * 30),
					DrainTimeout: durationpb.New(time.Second * 30),
				},
			},
		},
		Endpoints: []Endpoint{
			{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/test",
						},
					},
				},
			},
		},
		Apps: []App{
			{
				ID: "app1",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "function handle(req) { return req; }",
					},
				},
			},
		},
	}
}
