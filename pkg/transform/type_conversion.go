package transform

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ToString converts any value to its string representation
// Returns empty string for nil values
func ToString(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}

	switch val := v.(type) {
	case string:
		return val, nil
	case int:
		return strconv.FormatInt(int64(val), 10), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case []byte:
		return string(val), nil
	case json.Number:
		return val.String(), nil
	default:
		// Try to marshal as JSON for complex types
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val), nil
		}
		return string(data), nil
	}
}

// ParseInt parses a string into an int64 value
// Supports decimal, hexadecimal (0x), octal (0o), and binary (0b) formats
func ParseInt(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("cannot parse empty string as int: %w", ErrTypeMismatch)
	}

	s = strings.TrimSpace(s)

	// Try direct parsing with base 10 first
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val, nil
	}

	// Try parsing with auto base detection (0 = auto)
	val, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as int: %w", s, ErrTypeMismatch)
	}

	return val, nil
}

// ParseFloat parses a string into a float64 value
func ParseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("cannot parse empty string as float: %w", ErrTypeMismatch)
	}

	s = strings.TrimSpace(s)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as float: %w", s, ErrTypeMismatch)
	}

	return val, nil
}

// ParseBool parses a string into a bool value
// Accepts: true, false, yes, no, on, off, 1, 0 (case-insensitive)
func ParseBool(s string) (bool, error) {
	if s == "" {
		return false, fmt.Errorf("cannot parse empty string as bool: %w", ErrTypeMismatch)
	}

	s = strings.TrimSpace(strings.ToLower(s))

	switch s {
	case "true", "yes", "on", "1", "t", "y":
		return true, nil
	case "false", "no", "off", "0", "f", "n":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse %q as bool: %w", s, ErrTypeMismatch)
	}
}

// ToInt converts any value to int64
// Supports numeric types, strings, and booleans
func ToInt(v interface{}) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("cannot convert nil to int: %w", ErrTypeMismatch)
	}

	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		if val > 9223372036854775807 { // max int64
			return 0, fmt.Errorf("uint value %d exceeds max int64: %w", val, ErrTypeMismatch)
		}
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > 9223372036854775807 { // max int64
			return 0, fmt.Errorf("uint64 value %d exceeds max int64: %w", val, ErrTypeMismatch)
		}
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		if val > 9223372036854775807 || val < -9223372036854775808 {
			return 0, fmt.Errorf("float64 value %f exceeds int64 range: %w", val, ErrTypeMismatch)
		}
		return int64(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case string:
		return ParseInt(val)
	case json.Number:
		return val.Int64()
	default:
		return 0, fmt.Errorf("cannot convert %T to int: %w", v, ErrTypeMismatch)
	}
}

// ToFloat converts any value to float64
// Supports numeric types and strings
func ToFloat(v interface{}) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("cannot convert nil to float: %w", ErrTypeMismatch)
	}

	switch val := v.(type) {
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case bool:
		if val {
			return 1.0, nil
		}
		return 0.0, nil
	case string:
		return ParseFloat(val)
	case json.Number:
		return val.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float: %w", v, ErrTypeMismatch)
	}
}

// ToArray converts a value to []interface{}
// Works with arrays, slices, and can unmarshal JSON strings
func ToArray(v interface{}) ([]interface{}, error) {
	if v == nil {
		return []interface{}{}, nil
	}

	// Handle string that might be JSON array
	if s, ok := v.(string); ok {
		var arr []interface{}
		err := json.Unmarshal([]byte(s), &arr)
		if err == nil {
			return arr, nil
		}
		// Not a JSON array string, wrap the string itself
		return []interface{}{s}, nil
	}

	// Use reflection to handle any slice or array type
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		result := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	default:
		return []interface{}{}, fmt.Errorf("cannot convert %T to array: %w", v, ErrTypeMismatch)
	}
}

// ToMap converts a value to map[string]interface{}
// Works with JSON objects, maps, and can unmarshal JSON strings
func ToMap(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return make(map[string]interface{}), nil
	}

	// Handle string that might be JSON object
	if s, ok := v.(string); ok {
		var m map[string]interface{}
		err := json.Unmarshal([]byte(s), &m)
		if err != nil {
			return nil, fmt.Errorf("cannot parse string as map: %w", ErrTypeMismatch)
		}
		return m, nil
	}

	// Handle map[string]interface{} directly
	if m, ok := v.(map[string]interface{}); ok {
		return m, nil
	}

	// Use reflection to convert other map types
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map {
		return nil, fmt.Errorf("cannot convert %T to map: %w", v, ErrTypeMismatch)
	}

	// Convert map keys and values to string keys and interface{} values
	result := make(map[string]interface{})
	for _, key := range rv.MapKeys() {
		keyStr, err := ToString(key.Interface())
		if err != nil {
			return nil, fmt.Errorf("map key conversion failed: %w", err)
		}
		result[keyStr] = rv.MapIndex(key).Interface()
	}

	return result, nil
}

// IsNumeric returns true if the value is a numeric type
func IsNumeric(v interface{}) bool {
	if v == nil {
		return false
	}

	switch v.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case json.Number:
		return true
	default:
		return false
	}
}

// IsString returns true if the value is a string type
func IsString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

// IsArray returns true if the value is an array or slice type
func IsArray(v interface{}) bool {
	if v == nil {
		return false
	}

	// Check for string that might be JSON array
	if s, ok := v.(string); ok {
		var arr []interface{}
		return json.Unmarshal([]byte(s), &arr) == nil
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice
}

// IsMap returns true if the value is a map type or a JSON object string
func IsMap(v interface{}) bool {
	if v == nil {
		return false
	}

	// Check for string that might be JSON object
	if s, ok := v.(string); ok {
		var m map[string]interface{}
		return json.Unmarshal([]byte(s), &m) == nil
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Map
}

// IsBool returns true if the value is a boolean type
func IsBool(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}

// IsNil returns true if the value is nil or empty
func IsNil(v interface{}) bool {
	return v == nil
}

// GetType returns the type name of a value as a string
func GetType(v interface{}) string {
	if v == nil {
		return "nil"
	}

	// Special handling for json.Number
	if _, ok := v.(json.Number); ok {
		return "json.Number"
	}

	return reflect.TypeOf(v).String()
}
