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

type HeaderOperationsType string

const (
	RequestHeaderOperationsType  HeaderOperationsType = "Request"
	ResponseHeaderOperationsType HeaderOperationsType = "Response"
)

// HeaderOperations represents header manipulation operations
type HeaderOperations struct {
	// Title identifies the context (e.g., "Request" or "Response")
	Title HeaderOperationsType

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
func NewHeaderOperations(title HeaderOperationsType) *HeaderOperations {
	return &HeaderOperations{
		Title:         title,
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: []string{},
	}
}

// NewHeaders creates a new headers middleware configuration
func NewHeaders(request, response *HeaderOperations) *Headers {
	return &Headers{
		Request:  request,
		Response: response,
	}
}

// HasOperations checks if HeaderOperations has any operations
func (ho *HeaderOperations) HasOperations() bool {
	return len(ho.SetHeaders) > 0 || len(ho.AddHeaders) > 0 || len(ho.RemoveHeaders) > 0
}

// ToTree returns a tree representation using the operation's title
func (ho *HeaderOperations) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree(string(ho.Title) + " Operations:")

	// Set headers
	for key, value := range ho.SetHeaders {
		tree.AddChild(fmt.Sprintf("Set: \"%s: %s\"", key, value))
	}

	// Add headers
	for key, value := range ho.AddHeaders {
		tree.AddChild(fmt.Sprintf("Add: \"%s: %s\"", key, value))
	}

	// Remove headers
	for _, key := range ho.RemoveHeaders {
		tree.AddChild(fmt.Sprintf("Remove: \"%s\"", key))
	}

	return tree
}

// Type returns the middleware type
func (h *Headers) Type() string {
	return HeadersType
}

// Validate validates the header operations
func (ho *HeaderOperations) Validate() error {
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
	if h.Request != nil {
		if err := h.Request.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid request header operations: %w", err))
		}
	}

	// Validate response operations
	if h.Response != nil {
		if err := h.Response.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid response header operations: %w", err))
		}
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
		parts = append(parts, fmt.Sprintf("%s: %s", string(RequestHeaderOperationsType), h.Request.String()))
	}

	if h.Response != nil {
		parts = append(parts, fmt.Sprintf("%s: %s", string(ResponseHeaderOperationsType), h.Response.String()))
	}

	if len(parts) == 0 {
		return "No header operations configured"
	}

	return strings.Join(parts, ", ")
}

// ToTree returns a tree representation of the headers configuration
func (h *Headers) ToTree() *fancy.ComponentTree {
	// Check if we have any operations at all
	hasRequestOps := h.Request != nil && h.Request.HasOperations()
	hasResponseOps := h.Response != nil && h.Response.HasOperations()

	// Return empty tree if no operations exist
	if !hasRequestOps && !hasResponseOps {
		return fancy.NewComponentTree("")
	}

	tree := fancy.NewComponentTree("Config:")

	// Add Request and Response as children
	if hasRequestOps {
		tree.AddChild(h.Request.ToTree().Tree())
	}

	if hasResponseOps {
		tree.AddChild(h.Response.ToTree().Tree())
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
