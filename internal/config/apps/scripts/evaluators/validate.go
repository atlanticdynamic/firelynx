package evaluators

// ValidateEvaluatorType validates that the provided evaluator type is valid.
func ValidateEvaluatorType(typ EvaluatorType) error {
	switch typ {
	case EvaluatorTypeRisor, EvaluatorTypeStarlark, EvaluatorTypeExtism:
		return nil
	default:
		return NewInvalidEvaluatorTypeError(typ)
	}
}
