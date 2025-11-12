package workflow

import "fmt"

// validateType is a generic helper for compile-time type-safe validation.
// It checks if a value matches the expected type T and returns the typed value or an error.
// This reduces code duplication and improves type safety over repeated type assertions.
func validateType[T any](value interface{}, fieldName string) (T, error) {
	if v, ok := value.(T); ok {
		return v, nil
	}
	var zero T
	return zero, fmt.Errorf("variable: type mismatch for %s: expected %T, got %T", fieldName, zero, value)
}

// isNumericType checks if a value is any numeric type (int, uint, float variants).
// This helper reduces repetition in number type validation.
func isNumericType(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

// isArrayType checks if a value is any supported array/slice type.
// This helper reduces repetition in array type validation.
func isArrayType(value interface{}) bool {
	switch value.(type) {
	case []interface{}, []string, []int, []float64, []bool, []map[string]interface{}:
		return true
	default:
		return false
	}
}
