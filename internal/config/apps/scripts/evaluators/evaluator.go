// Package evaluators provides types and utilities for various script evaluators used in firelynx.
package evaluators

import (
	"fmt"
	"time"

	"github.com/robbyt/go-polyscript/platform"
)

type EvaluatorType int

const (
	DefaultEvalTimeout = 1 * time.Minute // Default timeout for script execution
)

// EvaluatorType enum values - must match the protobuf definition.
const (
	EvaluatorTypeUnspecified EvaluatorType = iota
	EvaluatorTypeRisor
	EvaluatorTypeStarlark
	EvaluatorTypeExtism
)

// Evaluator is the common interface for all script evaluators.
type Evaluator interface {
	Type() EvaluatorType
	Validate() error
	GetCompiledEvaluator() (platform.Evaluator, error)
	GetTimeout() time.Duration
}

// String returns a string representation of the EvaluatorType.
func (t EvaluatorType) String() string {
	switch t {
	case EvaluatorTypeRisor:
		return "Risor"
	case EvaluatorTypeStarlark:
		return "Starlark"
	case EvaluatorTypeExtism:
		return "Extism"
	case EvaluatorTypeUnspecified:
		return "Unspecified"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}
