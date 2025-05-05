package conditions

import "fmt"

// ValidateType checks if a condition Type is supported
func ValidateType(t Type) error {
	switch t {
	case TypeHTTP, TypeGRPC, TypeMCP:
		return nil
	case Unknown:
		return fmt.Errorf("%w: empty condition type", ErrInvalidConditionType)
	default:
		return fmt.Errorf("%w: unsupported condition type '%s'", ErrInvalidConditionType, t)
	}
}
