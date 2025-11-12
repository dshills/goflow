package transform

import (
	"testing"
)

// TestExtractParamGeneric tests the generic parameter extraction helper
func TestExtractParamGeneric(t *testing.T) {
	tests := []struct {
		name      string
		params    []interface{}
		index     int
		paramName string
		testFunc  func(t *testing.T, params []interface{}, index int, paramName string)
	}{
		{
			name:      "extract string success",
			params:    []interface{}{"hello", "world"},
			index:     0,
			paramName: "text",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				result, err := extractParam[string](params, index, paramName)
				if err != nil {
					t.Errorf("extractParam[string]() error = %v, want nil", err)
				}
				if result != "hello" {
					t.Errorf("extractParam[string]() = %v, want hello", result)
				}
			},
		},
		{
			name:      "extract string type mismatch",
			params:    []interface{}{123, "world"},
			index:     0,
			paramName: "text",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				_, err := extractParam[string](params, index, paramName)
				if err == nil {
					t.Error("extractParam[string]() error = nil, want error")
				}
			},
		},
		{
			name:      "extract int success",
			params:    []interface{}{42, "test"},
			index:     0,
			paramName: "count",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				result, err := extractParam[int](params, index, paramName)
				if err != nil {
					t.Errorf("extractParam[int]() error = %v, want nil", err)
				}
				if result != 42 {
					t.Errorf("extractParam[int]() = %d, want 42", result)
				}
			},
		},
		{
			name:      "extract bool success",
			params:    []interface{}{true, false},
			index:     0,
			paramName: "flag",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				result, err := extractParam[bool](params, index, paramName)
				if err != nil {
					t.Errorf("extractParam[bool]() error = %v, want nil", err)
				}
				if !result {
					t.Error("extractParam[bool]() = false, want true")
				}
			},
		},
		{
			name:      "extract slice success",
			params:    []interface{}{[]interface{}{"a", "b", "c"}},
			index:     0,
			paramName: "items",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				result, err := extractParam[[]interface{}](params, index, paramName)
				if err != nil {
					t.Errorf("extractParam[[]interface{}]() error = %v, want nil", err)
				}
				if len(result) != 3 {
					t.Errorf("extractParam[[]interface{}]() length = %d, want 3", len(result))
				}
			},
		},
		{
			name:      "index out of bounds",
			params:    []interface{}{"hello"},
			index:     5,
			paramName: "missing",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				_, err := extractParam[string](params, index, paramName)
				if err == nil {
					t.Error("extractParam[string]() error = nil, want error for out of bounds")
				}
			},
		},
		{
			name:      "extract from empty params",
			params:    []interface{}{},
			index:     0,
			paramName: "missing",
			testFunc: func(t *testing.T, params []interface{}, index int, paramName string) {
				_, err := extractParam[string](params, index, paramName)
				if err == nil {
					t.Error("extractParam[string]() error = nil, want error for empty params")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.params, tt.index, tt.paramName)
		})
	}
}

// TestMultipleParameterExtraction tests extracting multiple parameters
func TestMultipleParameterExtraction(t *testing.T) {
	params := []interface{}{"hello", "world", 42, true}

	// Extract first parameter (string)
	str1, err := extractParam[string](params, 0, "text1")
	if err != nil {
		t.Errorf("failed to extract first parameter: %v", err)
	}
	if str1 != "hello" {
		t.Errorf("first parameter = %s, want hello", str1)
	}

	// Extract second parameter (string)
	str2, err := extractParam[string](params, 1, "text2")
	if err != nil {
		t.Errorf("failed to extract second parameter: %v", err)
	}
	if str2 != "world" {
		t.Errorf("second parameter = %s, want world", str2)
	}

	// Extract third parameter (int)
	num, err := extractParam[int](params, 2, "count")
	if err != nil {
		t.Errorf("failed to extract third parameter: %v", err)
	}
	if num != 42 {
		t.Errorf("third parameter = %d, want 42", num)
	}

	// Extract fourth parameter (bool)
	flag, err := extractParam[bool](params, 3, "flag")
	if err != nil {
		t.Errorf("failed to extract fourth parameter: %v", err)
	}
	if !flag {
		t.Error("fourth parameter = false, want true")
	}
}

// TestExpressionBooleanResult tests boolean result type checking
func TestExpressionBooleanResult(t *testing.T) {
	tests := []struct {
		name    string
		result  interface{}
		wantVal bool
		wantErr bool
	}{
		{
			name:    "true value",
			result:  true,
			wantVal: true,
			wantErr: false,
		},
		{
			name:    "false value",
			result:  false,
			wantVal: false,
			wantErr: false,
		},
		{
			name:    "string instead of bool",
			result:  "true",
			wantVal: false,
			wantErr: true,
		},
		{
			name:    "int instead of bool",
			result:  1,
			wantVal: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the pattern from expression.go:109
			boolResult, ok := tt.result.(bool)

			if ok == tt.wantErr {
				t.Errorf("type assertion ok = %v, wantErr %v", ok, tt.wantErr)
			}

			if !tt.wantErr && boolResult != tt.wantVal {
				t.Errorf("boolean result = %v, want %v", boolResult, tt.wantVal)
			}
		})
	}
}

// TestJSONPathTypeSwitches tests type switches for JSON value handling
func TestJSONPathTypeSwitches(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		wantType string
	}{
		{
			name:     "string value",
			value:    "hello",
			wantType: "string",
		},
		{
			name:     "int value",
			value:    42,
			wantType: "number",
		},
		{
			name:     "float64 value",
			value:    3.14,
			wantType: "number",
		},
		{
			name:     "bool value",
			value:    true,
			wantType: "bool",
		},
		{
			name:     "map value",
			value:    map[string]interface{}{"key": "value"},
			wantType: "map",
		},
		{
			name:     "slice value",
			value:    []interface{}{1, 2, 3},
			wantType: "slice",
		},
		{
			name:     "nil value",
			value:    nil,
			wantType: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotType string

			// Simulate type switch pattern from jsonpath.go
			switch v := tt.value.(type) {
			case string:
				gotType = "string"
				_ = v
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				gotType = "number"
			case bool:
				gotType = "bool"
			case map[string]interface{}:
				gotType = "map"
			case []interface{}:
				gotType = "slice"
			case nil:
				gotType = "nil"
			default:
				t.Errorf("unhandled type: %T", v)
			}

			if gotType != tt.wantType {
				t.Errorf("type = %s, want %s", gotType, tt.wantType)
			}
		})
	}
}

// TestArrayTypeAssertion tests array type assertions for filter operations
func TestArrayTypeAssertion(t *testing.T) {
	tests := []struct {
		name    string
		result  interface{}
		wantLen int
		wantOk  bool
	}{
		{
			name:    "valid array",
			result:  []interface{}{1, 2, 3},
			wantLen: 3,
			wantOk:  true,
		},
		{
			name:    "empty array",
			result:  []interface{}{},
			wantLen: 0,
			wantOk:  true,
		},
		{
			name:    "not an array",
			result:  "not an array",
			wantLen: 0,
			wantOk:  false,
		},
		{
			name:    "nil value",
			result:  nil,
			wantLen: 0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate pattern from jsonpath.go:755
			arr, ok := tt.result.([]interface{})

			if ok != tt.wantOk {
				t.Errorf("type assertion ok = %v, want %v", ok, tt.wantOk)
			}

			if ok && len(arr) != tt.wantLen {
				t.Errorf("array length = %d, want %d", len(arr), tt.wantLen)
			}
		})
	}
}

// BenchmarkParameterExtraction benchmarks parameter extraction methods
func BenchmarkParameterExtraction(b *testing.B) {
	params := []interface{}{"hello", "world", 42}

	b.Run("direct_type_assertion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			str, ok1 := params[0].(string)
			_, ok2 := params[1].(string)
			if !ok1 || !ok2 {
				b.Fatal("type assertion failed")
			}
			_ = str
		}
	})

	b.Run("generic_helper", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			str, err1 := extractParam[string](params, 0, "str1")
			_, err2 := extractParam[string](params, 1, "str2")
			if err1 != nil || err2 != nil {
				b.Fatal("parameter extraction failed")
			}
			_ = str
		}
	})
}

// BenchmarkTypeSwitchVsAssertion benchmarks type switch vs type assertion
func BenchmarkTypeSwitchVsAssertion(b *testing.B) {
	value := interface{}(42)

	b.Run("type_switch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			switch v := value.(type) {
			case int:
				_ = v
			case float64:
				_ = v
			case string:
				_ = v
			}
		}
	})

	b.Run("type_assertion_chain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if v, ok := value.(int); ok {
				_ = v
			} else if v, ok := value.(float64); ok {
				_ = v
			} else if v, ok := value.(string); ok {
				_ = v
			}
		}
	})
}

// TestBehavioralEquivalence verifies that refactored code maintains behavior
func TestBehavioralEquivalence(t *testing.T) {
	params := []interface{}{"hello", "world"}

	// Old pattern (type assertion)
	str1Old, ok1 := params[0].(string)
	str2Old, ok2 := params[1].(string)
	oldSuccess := ok1 && ok2

	// New pattern (generic helper)
	str1New, err1 := extractParam[string](params, 0, "str1")
	str2New, err2 := extractParam[string](params, 1, "str2")
	newSuccess := err1 == nil && err2 == nil

	// Verify behavioral equivalence
	if oldSuccess != newSuccess {
		t.Errorf("behavioral mismatch: old success = %v, new success = %v", oldSuccess, newSuccess)
	}

	if oldSuccess && newSuccess {
		if str1Old != str1New {
			t.Errorf("value mismatch: old = %s, new = %s", str1Old, str1New)
		}
		if str2Old != str2New {
			t.Errorf("value mismatch: old = %s, new = %s", str2Old, str2New)
		}
	}
}
