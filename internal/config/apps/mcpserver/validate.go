package mcpserver

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/validation"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// Validate performs validation for a Tool
func (t *Tool) Validate() error {
	var errs []error

	// Validate AppID
	if err := validation.ValidateID(t.AppID, "tool app ID"); err != nil {
		errs = append(errs, err)
	}

	// Tool.ID is optional. When supplied it becomes the registered MCP
	// tool name (see EffectiveID), so apply the same character/length
	// rules as other IDs.
	if t.ID != "" {
		if err := validation.ValidateID(t.ID, "tool ID"); err != nil {
			errs = append(errs, err)
		}
	}

	// Both schemas are optional overrides on top of the provider-defined
	// schemas (auto-generated from typed Go structs). Validate the JSON
	// shape only when supplied.
	if err := t.Schema.ValidateInput(); err != nil {
		errs = append(errs, fmt.Errorf("tool input schema: %w", err))
	}
	if err := t.Schema.ValidateOutput(); err != nil {
		errs = append(errs, fmt.Errorf("tool output schema: %w", err))
	}

	return errors.Join(errs...)
}

// Validate performs validation for a Prompt
func (p *Prompt) Validate() error {
	var errs []error

	// Validate ID
	if err := validation.ValidateID(p.ID, "prompt ID"); err != nil {
		errs = append(errs, err)
	}

	// Validate AppID
	if err := validation.ValidateID(p.AppID, "prompt app ID"); err != nil {
		errs = append(errs, err)
	}

	// Input schema is an optional override; validate JSON shape when supplied.
	if err := p.Schema.ValidateInput(); err != nil {
		errs = append(errs, fmt.Errorf("prompt input schema: %w", err))
	}

	// Prompts return text content, so the output schema field is unused.

	return errors.Join(errs...)
}

// Validate performs validation for a Resource
func (r *Resource) Validate() error {
	var errs []error

	// Validate ID
	if err := validation.ValidateID(r.ID, "resource ID"); err != nil {
		errs = append(errs, err)
	}

	// Validate AppID
	if err := validation.ValidateID(r.AppID, "resource app ID"); err != nil {
		errs = append(errs, err)
	}

	// Validate URITemplate (required for resources)
	if r.URITemplate == "" {
		errs = append(errs, fmt.Errorf("resource uri_template is required"))
	}

	return errors.Join(errs...)
}

// Validate performs validation for an MCP App
func (a *App) Validate() error {
	var errs []error

	// Interpolate all tagged fields first (following config guidelines)
	if err := interpolation.InterpolateStruct(a); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for MCP app: %w", err))
	}

	// Validate ID
	if err := validation.ValidateID(a.ID, "MCP app ID"); err != nil {
		errs = append(errs, err)
	}

	// Validate all tools
	for i, tool := range a.Tools {
		if err := tool.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tool %d: %w", i, err))
		}
	}

	// Validate all prompts
	for i, prompt := range a.Prompts {
		if err := prompt.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("prompt %d (%s): %w", i, prompt.ID, err))
		}
	}

	// Validate all resources
	for i, resource := range a.Resources {
		if err := resource.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("resource %d (%s): %w", i, resource.ID, err))
		}
	}

	// Check for duplicate IDs within each primitive type
	if err := a.validateNoDuplicateIDs(); err != nil {
		errs = append(errs, err)
	}

	// Note: App ID cross-reference validation (checking if referenced app IDs exist)
	// will be implemented once we understand how to access the app registry from validation context

	return errors.Join(errs...)
}

// validateNoDuplicateIDs checks for duplicate IDs within each primitive type
func (a *App) validateNoDuplicateIDs() error {
	var errs []error

	// Tool.ID is an optional override of the registered MCP tool name. When
	// supplied, two tools with the same Tool.ID would collide at MCP
	// registration. Empty Tool.IDs fall back to EffectiveID()/AppID, and
	// multiple tools may legitimately reference the same app, so skip them.
	toolIDs := make(map[string]bool)
	for _, tool := range a.Tools {
		if tool.ID == "" {
			continue
		}
		if toolIDs[tool.ID] {
			errs = append(errs, fmt.Errorf("duplicate tool ID '%s'", tool.ID))
		}
		toolIDs[tool.ID] = true
	}

	// Check for duplicate prompt IDs
	promptIDs := make(map[string]bool)
	for _, prompt := range a.Prompts {
		if promptIDs[prompt.ID] {
			errs = append(errs, fmt.Errorf("duplicate prompt ID '%s'", prompt.ID))
		}
		promptIDs[prompt.ID] = true
	}

	// Check for duplicate resource IDs
	resourceIDs := make(map[string]bool)
	for _, resource := range a.Resources {
		if resourceIDs[resource.ID] {
			errs = append(errs, fmt.Errorf("duplicate resource ID '%s'", resource.ID))
		}
		resourceIDs[resource.ID] = true
	}

	return errors.Join(errs...)
}
