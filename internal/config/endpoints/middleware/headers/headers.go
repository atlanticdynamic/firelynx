package headers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"golang.org/x/net/http/httpguts"
)

const HeadersType = "headers"

// HeaderOperations represents header manipulation operations
type HeaderOperations struct {
	// Headers to set (replace existing values)
	SetHeaders map[string]string `json:"setHeaders" toml:"set_headers" env_interpolation:"yes"`

	// Headers to add (append to existing values)
	AddHeaders map[string]string `json:"addHeaders" toml:"add_headers" env_interpolation:"yes"`

	// Header names to remove
	RemoveHeaders []string `json:"removeHeaders" toml:"remove_headers" env_interpolation:"yes"`
}

// Headers represents a headers middleware configuration
type Headers struct {
	// Operations to perform on request headers
	Request *HeaderOperations `json:"request,omitempty" toml:"request,omitempty"`

	// Operations to perform on response headers
	Response *HeaderOperations `json:"response,omitempty" toml:"response,omitempty"`
}

// NewHeaderOperations creates a new header operations configuration
func NewHeaderOperations() *HeaderOperations {
	return &HeaderOperations{
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: []string{},
	}
}

// NewHeaders creates a new headers middleware configuration with default settings
func NewHeaders() *Headers {
	return &Headers{
		Request:  nil, // No request operations by default
		Response: nil, // No response operations by default
	}
}

// Type returns the middleware type
func (h *Headers) Type() string {
	return HeadersType
}

// Validate validates the header operations
func (ho *HeaderOperations) Validate() error {
	if ho == nil {
		return nil
	}

	var errs []error

	// Interpolate all tagged fields (map values)
	if err := interpolation.InterpolateStruct(ho); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed: %w", err))
	}

	// Validate set headers
	for key, value := range ho.SetHeaders {
		if err := validateHeader(key, value); err != nil {
			errs = append(errs, fmt.Errorf("invalid set header '%s': %w", key, err))
		}
	}

	// Validate add headers
	for key, value := range ho.AddHeaders {
		if err := validateHeader(key, value); err != nil {
			errs = append(errs, fmt.Errorf("invalid add header '%s': %w", key, err))
		}
	}

	// Validate remove headers
	for _, key := range ho.RemoveHeaders {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, errors.New("remove header name cannot be empty"))
		}
	}

	return errors.Join(errs...)
}

// Validate validates the headers configuration
func (h *Headers) Validate() error {
	var errs []error

	// Validate request operations
	if err := h.Request.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid request header operations: %w", err))
	}

	// Validate response operations
	if err := h.Response.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid response header operations: %w", err))
	}

	// At least one operation must be configured
	if h.Request == nil && h.Response == nil {
		errs = append(
			errs,
			errors.New("at least one of request or response operations must be configured"),
		)
	}

	return errors.Join(errs...)
}

// String returns a string representation of the header operations
func (ho *HeaderOperations) String() string {
	if ho == nil {
		return "No operations"
	}

	var parts []string

	if len(ho.SetHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Set: %d headers", len(ho.SetHeaders)))
	}

	if len(ho.AddHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Add: %d headers", len(ho.AddHeaders)))
	}

	if len(ho.RemoveHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Remove: %d headers", len(ho.RemoveHeaders)))
	}

	if len(parts) == 0 {
		return "No operations"
	}

	return strings.Join(parts, ", ")
}

// String returns a string representation of the headers configuration
func (h *Headers) String() string {
	var parts []string

	if h.Request != nil {
		parts = append(parts, fmt.Sprintf("Request: %s", h.Request.String()))
	}

	if h.Response != nil {
		parts = append(parts, fmt.Sprintf("Response: %s", h.Response.String()))
	}

	if len(parts) == 0 {
		return "No header operations configured"
	}

	return strings.Join(parts, ", ")
}

// addOperationsToTree adds header operations to a tree node
func addOperationsToTree(tree *fancy.ComponentTree, prefix string, ho *HeaderOperations) {
	if ho == nil {
		tree.AddChild(fmt.Sprintf("%s: No operations", prefix))
		return
	}

	hasOperations := false

	// Set headers
	if len(ho.SetHeaders) > 0 {
		setNode := tree.AddChild(fmt.Sprintf("%s Set Headers:", prefix))
		for key, value := range ho.SetHeaders {
			setNode.Child(fmt.Sprintf("%s: %s", key, value))
		}
		hasOperations = true
	}

	// Add headers
	if len(ho.AddHeaders) > 0 {
		addNode := tree.AddChild(fmt.Sprintf("%s Add Headers:", prefix))
		for key, value := range ho.AddHeaders {
			addNode.Child(fmt.Sprintf("%s: %s", key, value))
		}
		hasOperations = true
	}

	// Remove headers
	if len(ho.RemoveHeaders) > 0 {
		removeNode := tree.AddChild(fmt.Sprintf("%s Remove Headers:", prefix))
		for _, key := range ho.RemoveHeaders {
			removeNode.Child(key)
		}
		hasOperations = true
	}

	if !hasOperations {
		tree.AddChild(fmt.Sprintf("%s: No operations", prefix))
	}
}

// ToTree returns a tree representation of the headers configuration
func (h *Headers) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree(fancy.MiddlewareText("Headers Middleware"))

	// Request operations
	if h.Request != nil {
		addOperationsToTree(tree, "Request", h.Request)
	} else {
		tree.AddChild("Request: No operations")
	}

	// Response operations
	if h.Response != nil {
		addOperationsToTree(tree, "Response", h.Response)
	} else {
		tree.AddChild("Response: No operations")
	}

	return tree
}

// validateHeader validates a header key-value pair using httpguts
func validateHeader(key, value string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("header name cannot be empty")
	}

	if !httpguts.ValidHeaderFieldName(key) {
		return fmt.Errorf("invalid header name: %s", key)
	}

	if !httpguts.ValidHeaderFieldValue(value) {
		return fmt.Errorf("invalid header value for %s", key)
	}

	return nil
}
