// Package evaluators provides types and utilities for various script evaluators used in firelynx.
package evaluators

import (
	"time"
)

// EvaluatorType represents the type of script evaluator.
type EvaluatorType int

// EvaluatorType enum values - must match the protobuf definition.
const (
	// EvaluatorTypeUnspecified is the default evaluator type (invalid).
	EvaluatorTypeUnspecified EvaluatorType = iota
	// EvaluatorTypeRisor represents a Risor script evaluator.
	EvaluatorTypeRisor
	// EvaluatorTypeStarlark represents a Starlark script evaluator.
	EvaluatorTypeStarlark
	// EvaluatorTypeExtism represents an Extism WASM evaluator.
	EvaluatorTypeExtism
)

// Evaluator is the common interface for all script evaluators.
type Evaluator interface {
	// Type returns the type of this evaluator.
	Type() EvaluatorType
	// Validate validates the evaluator configuration.
	Validate() error
}

// RisorEvaluator represents a Risor script evaluator.
type RisorEvaluator struct {
	// Code contains the Risor script source code.
	Code string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration
}

// Type returns the type of this evaluator.
func (r *RisorEvaluator) Type() EvaluatorType {
	return EvaluatorTypeRisor
}

// StarlarkEvaluator represents a Starlark script evaluator.
type StarlarkEvaluator struct {
	// Code contains the Starlark script source code.
	Code string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration
}

// Type returns the type of this evaluator.
func (s *StarlarkEvaluator) Type() EvaluatorType {
	return EvaluatorTypeStarlark
}

// ExtismEvaluator represents an Extism WASM evaluator.
type ExtismEvaluator struct {
	// Code contains the WASM binary encoded as base64.
	Code string
	// Entrypoint is the name of the function to call within the WASM module.
	Entrypoint string
}

// Type returns the type of this evaluator.
func (e *ExtismEvaluator) Type() EvaluatorType {
	return EvaluatorTypeExtism
}
