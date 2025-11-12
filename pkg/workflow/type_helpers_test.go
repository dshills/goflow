package workflow

import (
	"testing"
)

// TestValidateTypeGeneric tests the generic type validation helper
func TestValidateTypeGeneric(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		typeName string
		testFunc func(t *testing.T, value interface{})
	}{
		{
			name:     "string validation success",
			value:    "hello",
			typeName: "string",
			testFunc: func(t *testing.T, value interface{}) {
				result, err := validateType[string](value, "string")
				if err != nil {
					t.Errorf("validateType[string]() error = %v, want nil", err)
				}
				if result != "hello" {
					t.Errorf("validateType[string]() = %v, want hello", result)
				}
			},
		},
		{
			name:     "string validation failure",
			value:    123,
			typeName: "string",
			testFunc: func(t *testing.T, value interface{}) {
				_, err := validateType[string](value, "string")
				if err == nil {
					t.Error("validateType[string]() error = nil, want error")
				}
			},
		},
		{
			name:     "bool validation success",
			value:    true,
			typeName: "bool",
			testFunc: func(t *testing.T, value interface{}) {
				result, err := validateType[bool](value, "bool")
				if err != nil {
					t.Errorf("validateType[bool]() error = %v, want nil", err)
				}
				if result != true {
					t.Errorf("validateType[bool]() = %v, want true", result)
				}
			},
		},
		{
			name:     "bool validation failure",
			value:    "true",
			typeName: "bool",
			testFunc: func(t *testing.T, value interface{}) {
				_, err := validateType[bool](value, "bool")
				if err == nil {
					t.Error("validateType[bool]() error = nil, want error")
				}
			},
		},
		{
			name:     "map validation success",
			value:    map[string]interface{}{"key": "value"},
			typeName: "map",
			testFunc: func(t *testing.T, value interface{}) {
				result, err := validateType[map[string]interface{}](value, "map")
				if err != nil {
					t.Errorf("validateType[map]() error = %v, want nil", err)
				}
				if len(result) != 1 {
					t.Errorf("validateType[map]() length = %d, want 1", len(result))
				}
			},
		},
		{
			name:     "map validation failure",
			value:    []string{"not", "a", "map"},
			typeName: "map",
			testFunc: func(t *testing.T, value interface{}) {
				_, err := validateType[map[string]interface{}](value, "map")
				if err == nil {
					t.Error("validateType[map]() error = nil, want error")
				}
			},
		},
		{
			name:     "int validation success",
			value:    42,
			typeName: "int",
			testFunc: func(t *testing.T, value interface{}) {
				result, err := validateType[int](value, "int")
				if err != nil {
					t.Errorf("validateType[int]() error = %v, want nil", err)
				}
				if result != 42 {
					t.Errorf("validateType[int]() = %d, want 42", result)
				}
			},
		},
		{
			name:     "float64 validation success",
			value:    3.14,
			typeName: "float64",
			testFunc: func(t *testing.T, value interface{}) {
				result, err := validateType[float64](value, "float64")
				if err != nil {
					t.Errorf("validateType[float64]() error = %v, want nil", err)
				}
				if result != 3.14 {
					t.Errorf("validateType[float64]() = %f, want 3.14", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.value)
		})
	}
}

// TestVariableTypeValidation tests variable type validation patterns
func TestVariableTypeValidation(t *testing.T) {
	tests := []struct {
		name         string
		varType      string
		defaultValue interface{}
		wantErr      bool
	}{
		{
			name:         "string type with string value",
			varType:      "string",
			defaultValue: "test",
			wantErr:      false,
		},
		{
			name:         "string type with int value",
			varType:      "string",
			defaultValue: 123,
			wantErr:      true,
		},
		{
			name:         "boolean type with bool value",
			varType:      "boolean",
			defaultValue: true,
			wantErr:      false,
		},
		{
			name:         "boolean type with string value",
			varType:      "boolean",
			defaultValue: "true",
			wantErr:      true,
		},
		{
			name:         "object type with map value",
			varType:      "object",
			defaultValue: map[string]interface{}{"key": "value"},
			wantErr:      false,
		},
		{
			name:         "object type with string value",
			varType:      "object",
			defaultValue: "not an object",
			wantErr:      true,
		},
		{
			name:         "number type with int",
			varType:      "number",
			defaultValue: 42,
			wantErr:      false,
		},
		{
			name:         "number type with float64",
			varType:      "number",
			defaultValue: 3.14,
			wantErr:      false,
		},
		{
			name:         "number type with string",
			varType:      "number",
			defaultValue: "42",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Variable{
				Name:         "test_var",
				Type:         tt.varType,
				DefaultValue: tt.defaultValue,
			}

			err := v.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Variable.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTypeSwitchExhaustiveness tests that type switches handle all expected cases
func TestTypeSwitchExhaustiveness(t *testing.T) {
	// Test that all variable types are handled
	varTypes := []string{"string", "number", "boolean", "object", "array"}

	for _, varType := range varTypes {
		t.Run("variable_type_"+varType, func(t *testing.T) {
			var defaultValue interface{}

			switch varType {
			case "string":
				defaultValue = "test"
			case "number":
				defaultValue = 42
			case "boolean":
				defaultValue = true
			case "object":
				defaultValue = map[string]interface{}{}
			case "array":
				defaultValue = []interface{}{}
			default:
				t.Errorf("unhandled variable type: %s", varType)
			}

			v := &Variable{
				Name:         "test",
				Type:         varType,
				DefaultValue: defaultValue,
			}

			if err := v.Validate(); err != nil {
				t.Errorf("Variable.Validate() failed for type %s: %v", varType, err)
			}
		})
	}
}

// BenchmarkTypeValidation benchmarks different type validation approaches
func BenchmarkTypeValidation(b *testing.B) {
	value := "test string"

	b.Run("direct_type_assertion", func(b *testing.B) {
		var val interface{} = value
		for i := 0; i < b.N; i++ {
			if _, ok := val.(string); !ok {
				b.Fatal("should not fail")
			}
		}
	})

	b.Run("generic_helper", func(b *testing.B) {
		var val interface{} = value
		for i := 0; i < b.N; i++ {
			if _, err := validateType[string](val, "string"); err != nil {
				b.Fatal("should not fail")
			}
		}
	})
}

// TestTemplateTypeConversion tests template value type conversions
func TestTemplateTypeConversion(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantType  string
		assertion func(t *testing.T, value interface{})
	}{
		{
			name:     "string value",
			value:    "hello",
			wantType: "string",
			assertion: func(t *testing.T, value interface{}) {
				s, ok := value.(string)
				if !ok {
					t.Error("value is not a string")
				}
				if s != "hello" {
					t.Errorf("value = %s, want hello", s)
				}
			},
		},
		{
			name:     "bool value",
			value:    true,
			wantType: "bool",
			assertion: func(t *testing.T, value interface{}) {
				b, ok := value.(bool)
				if !ok {
					t.Error("value is not a bool")
				}
				if !b {
					t.Error("value = false, want true")
				}
			},
		},
		{
			name:     "int value",
			value:    42,
			wantType: "int",
			assertion: func(t *testing.T, value interface{}) {
				// Test that int is compatible with number types
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					// Success
				default:
					t.Errorf("value type %T is not a number", value)
				}
			},
		},
		{
			name:     "float64 value",
			value:    3.14,
			wantType: "float64",
			assertion: func(t *testing.T, value interface{}) {
				f, ok := value.(float64)
				if !ok {
					t.Error("value is not a float64")
				}
				if f != 3.14 {
					t.Errorf("value = %f, want 3.14", f)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertion(t, tt.value)
		})
	}
}
