package headers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"golang.org/x/net/http/httpguts"
)

const HeadersType = "headers"

// Headers represents a headers middleware configuration
type Headers struct {
	// Headers to set (replace existing values)
	SetHeaders map[string]string `json:"setHeaders" toml:"set_headers"`

	// Headers to add (append to existing values)
	AddHeaders map[string]string `json:"addHeaders" toml:"add_headers"`

	// Header names to remove
	RemoveHeaders []string `json:"removeHeaders" toml:"remove_headers"`
}

// NewHeaders creates a new headers middleware configuration with default settings
func NewHeaders() *Headers {
	return &Headers{
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: []string{},
	}
}

// Type returns the middleware type
func (h *Headers) Type() string {
	return HeadersType
}

// Validate validates the headers configuration
func (h *Headers) Validate() error {
	var errs []error

	// Validate set headers
	for key, value := range h.SetHeaders {
		if err := validateHeader(key, value); err != nil {
			errs = append(errs, fmt.Errorf("invalid set header '%s': %w", key, err))
		}
	}

	// Validate add headers
	for key, value := range h.AddHeaders {
		if err := validateHeader(key, value); err != nil {
			errs = append(errs, fmt.Errorf("invalid add header '%s': %w", key, err))
		}
	}

	// Validate remove headers
	for _, key := range h.RemoveHeaders {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, errors.New("remove header name cannot be empty"))
		}
	}

	return errors.Join(errs...)
}

// String returns a string representation of the headers configuration
func (h *Headers) String() string {
	var parts []string

	if len(h.SetHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Set: %d headers", len(h.SetHeaders)))
	}

	if len(h.AddHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Add: %d headers", len(h.AddHeaders)))
	}

	if len(h.RemoveHeaders) > 0 {
		parts = append(parts, fmt.Sprintf("Remove: %d headers", len(h.RemoveHeaders)))
	}

	if len(parts) == 0 {
		return "No header operations configured"
	}

	return strings.Join(parts, ", ")
}

// ToTree returns a tree representation of the headers configuration
func (h *Headers) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree(fancy.MiddlewareText("Headers Middleware"))

	// Set headers
	if len(h.SetHeaders) > 0 {
		setNode := tree.AddChild("Set Headers:")
		for key, value := range h.SetHeaders {
			setNode.Child(fmt.Sprintf("%s: %s", key, value))
		}
	}

	// Add headers
	if len(h.AddHeaders) > 0 {
		addNode := tree.AddChild("Add Headers:")
		for key, value := range h.AddHeaders {
			addNode.Child(fmt.Sprintf("%s: %s", key, value))
		}
	}

	// Remove headers
	if len(h.RemoveHeaders) > 0 {
		removeNode := tree.AddChild("Remove Headers:")
		for _, key := range h.RemoveHeaders {
			removeNode.Child(key)
		}
	}

	if len(h.SetHeaders) == 0 && len(h.AddHeaders) == 0 && len(h.RemoveHeaders) == 0 {
		tree.AddChild("No header operations configured")
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
