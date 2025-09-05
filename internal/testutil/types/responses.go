package types

// ScriptResponse represents a common response structure from script evaluations
type ScriptResponse struct {
	Message     string      `json:"message"`
	Service     string      `json:"service"`
	Version     string      `json:"version"`
	Environment string      `json:"environment,omitempty"`
	Features    []string    `json:"features,omitempty"`
	RequestInfo RequestInfo `json:"requestInfo,omitempty"`
	Timestamp   string      `json:"timestamp,omitempty"`
}

// RequestInfo represents request information in script responses
type RequestInfo struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	UserAgent string `json:"userAgent,omitempty"`
}

// ScriptRisorBasicResponse represents the expected response structure from Risor scripts
type ScriptRisorBasicResponse struct {
	Message     string      `json:"message"`
	Service     string      `json:"service"`
	Version     string      `json:"version"`
	Environment string      `json:"environment"`
	RequestInfo RequestInfo `json:"requestInfo"`
	Timestamp   string      `json:"timestamp"`
}

// ScriptStarlarkBasicResponse represents the expected response structure from Starlark scripts
type ScriptStarlarkBasicResponse struct {
	Message        string      `json:"message"`
	Service        string      `json:"service"`
	Version        string      `json:"version"`
	Features       []string    `json:"features"`
	RequestInfo    RequestInfo `json:"requestInfo"`
	ScriptLanguage string      `json:"scriptLanguage"`
}

// ScriptWithDataResponse represents the expected response structure from advanced data processing scripts
type ScriptWithDataResponse struct {
	Service           string                 `json:"service"`
	ProcessingSummary map[string]interface{} `json:"processingSummary"`
	ProcessedItems    []interface{}          `json:"processedItems"`
	Errors            []string               `json:"errors"`
}

// ProcessedItem represents a single processed item structure
type ProcessedItem struct {
	Original    map[string]interface{} `json:"original"`
	Index       int                    `json:"index"`
	Transformed map[string]interface{} `json:"transformed,omitempty"`
	Error       string                 `json:"error,omitempty"`
}
