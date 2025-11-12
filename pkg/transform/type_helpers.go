package transform

import "fmt"

// extractParam is a generic helper for type-safe parameter extraction from function parameters.
// It reduces code duplication in function parameter handling and provides better error messages.
//
// Type parameters:
//   - T: The expected type of the parameter
//
// Parameters:
//   - params: The slice of parameters to extract from
//   - index: The index of the parameter to extract
//   - name: A descriptive name for the parameter (used in error messages)
//
// Returns:
//   - The parameter value with type T if successful
//   - An error if the parameter is missing or has the wrong type
//
// Example usage:
//
//	str, err := extractParam[string](params, 0, "text")
//	if err != nil {
//	    return err
//	}
func extractParam[T any](params []interface{}, index int, name string) (T, error) {
	var zero T

	if index >= len(params) {
		return zero, fmt.Errorf("parameter %d (%s) not provided", index, name)
	}

	if v, ok := params[index].(T); ok {
		return v, nil
	}

	return zero, fmt.Errorf("parameter %d (%s) must be %T, got %T", index, name, zero, params[index])
}

// extractBoolResult is a helper for extracting boolean results from expression evaluation.
// It provides consistent error handling for boolean type assertions.
func extractBoolResult(result interface{}, context string) (bool, error) {
	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("%w: %s returned %T, expected bool", ErrTypeMismatch, context, result)
	}
	return boolResult, nil
}
