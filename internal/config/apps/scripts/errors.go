package scripts

import (
	"errors"
	"fmt"
)

var (
	// ErrAppScript is the base error type for AppScript errors.
	ErrAppScript = errors.New("app script error")

	// ErrMissingEvaluator indicates that no evaluator was provided.
	ErrMissingEvaluator = fmt.Errorf("%w: missing evaluator", ErrAppScript)

	// ErrInvalidEvaluator indicates that the provided evaluator is invalid.
	ErrInvalidEvaluator = fmt.Errorf("%w: invalid evaluator", ErrAppScript)

	// ErrInvalidStaticData indicates that the provided static data is invalid.
	ErrInvalidStaticData = fmt.Errorf("%w: invalid static data", ErrAppScript)

	// ErrProtoConversion indicates an error converting to/from protobuf.
	ErrProtoConversion = fmt.Errorf("%w: proto conversion error", ErrAppScript)
)
