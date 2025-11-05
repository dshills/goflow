package transform

import (
	"encoding/json"
	"errors"
	"testing"
)

// ============================================================================
// ToString Tests
// ============================================================================

func TestToString(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		// Nil and empty cases
		{
			name:    "nil input",
			input:   nil,
			want:    "",
			wantErr: false,
		},
		// String cases
		{
			name:    "string passthrough",
			input:   "hello",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		// Integer cases
		{
			name:    "int positive",
			input:   42,
			want:    "42",
			wantErr: false,
		},
		{
			name:    "int negative",
			input:   -42,
			want:    "-42",
			wantErr: false,
		},
		{
			name:    "int64",
			input:   int64(9223372036854775807),
			want:    "9223372036854775807",
			wantErr: false,
		},
		{
			name:    "int zero",
			input:   0,
			want:    "0",
			wantErr: false,
		},
		// Float cases
		{
			name:    "float64",
			input:   3.14159,
			want:    "3.14159",
			wantErr: false,
		},
		{
			name:    "float32",
			input:   float32(2.71828),
			want:    "2.71828",
			wantErr: false,
		},
		{
			name:    "float zero",
			input:   0.0,
			want:    "0",
			wantErr: false,
		},
		// Boolean cases
		{
			name:    "bool true",
			input:   true,
			want:    "true",
			wantErr: false,
		},
		{
			name:    "bool false",
			input:   false,
			want:    "false",
			wantErr: false,
		},
		// Byte cases
		{
			name:    "byte slice",
			input:   []byte("hello"),
			want:    "hello",
			wantErr: false,
		},
		// Complex types
		{
			name:    "slice marshals to JSON",
			input:   []int{1, 2, 3},
			want:    "[1,2,3]",
			wantErr: false,
		},
		{
			name:    "map marshals to JSON",
			input:   map[string]int{"a": 1, "b": 2},
			wantErr: false, // JSON marshaling order varies
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.want != "" && got != tt.want {
				t.Errorf("ToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ParseInt Tests
// ============================================================================

func TestParseInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:    "simple decimal",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "negative decimal",
			input:   "-42",
			want:    -42,
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "large number",
			input:   "9223372036854775807",
			want:    9223372036854775807,
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  42  ",
			want:    42,
			wantErr: false,
		},
		{
			name:    "hexadecimal",
			input:   "0xFF",
			want:    255,
			wantErr: false,
		},
		{
			name:    "octal",
			input:   "0o77",
			want:    63,
			wantErr: false,
		},
		{
			name:    "binary",
			input:   "0b1010",
			want:    10,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "non-numeric",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "float string",
			input:   "3.14",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ParseInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ParseFloat Tests
// ============================================================================

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{
			name:    "simple float",
			input:   "3.14",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "negative float",
			input:   "-2.71",
			want:    -2.71,
			wantErr: false,
		},
		{
			name:    "integer string",
			input:   "42",
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "scientific notation",
			input:   "1.23e4",
			want:    12300.0,
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  3.14  ",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "non-numeric",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ParseFloat() = %f, want %f", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ParseBool Tests
// ============================================================================

func TestParseBool(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		// True values
		{
			name:    "true",
			input:   "true",
			want:    true,
			wantErr: false,
		},
		{
			name:    "TRUE uppercase",
			input:   "TRUE",
			want:    true,
			wantErr: false,
		},
		{
			name:    "yes",
			input:   "yes",
			want:    true,
			wantErr: false,
		},
		{
			name:    "on",
			input:   "on",
			want:    true,
			wantErr: false,
		},
		{
			name:    "1",
			input:   "1",
			want:    true,
			wantErr: false,
		},
		{
			name:    "t",
			input:   "t",
			want:    true,
			wantErr: false,
		},
		// False values
		{
			name:    "false",
			input:   "false",
			want:    false,
			wantErr: false,
		},
		{
			name:    "FALSE uppercase",
			input:   "FALSE",
			want:    false,
			wantErr: false,
		},
		{
			name:    "no",
			input:   "no",
			want:    false,
			wantErr: false,
		},
		{
			name:    "off",
			input:   "off",
			want:    false,
			wantErr: false,
		},
		{
			name:    "0",
			input:   "0",
			want:    false,
			wantErr: false,
		},
		{
			name:    "f",
			input:   "f",
			want:    false,
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  yes  ",
			want:    true,
			wantErr: false,
		},
		// Error cases
		{
			name:    "empty string",
			input:   "",
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid value",
			input:   "maybe",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ParseBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ToInt Tests
// ============================================================================

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    int64
		wantErr bool
	}{
		// Nil case
		{
			name:    "nil input",
			input:   nil,
			want:    0,
			wantErr: true,
		},
		// Integer conversions
		{
			name:    "int",
			input:   42,
			want:    42,
			wantErr: false,
		},
		{
			name:    "int64",
			input:   int64(100),
			want:    100,
			wantErr: false,
		},
		{
			name:    "uint",
			input:   uint(50),
			want:    50,
			wantErr: false,
		},
		// Float conversions
		{
			name:    "float64",
			input:   3.14,
			want:    3,
			wantErr: false,
		},
		{
			name:    "float32",
			input:   float32(2.5),
			want:    2,
			wantErr: false,
		},
		// Boolean conversions
		{
			name:    "bool true",
			input:   true,
			want:    1,
			wantErr: false,
		},
		{
			name:    "bool false",
			input:   false,
			want:    0,
			wantErr: false,
		},
		// String conversions
		{
			name:    "string decimal",
			input:   "42",
			want:    42,
			wantErr: false,
		},
		{
			name:    "string hex",
			input:   "0xFF",
			want:    255,
			wantErr: false,
		},
		// Overflow cases
		{
			name:    "uint64 overflow",
			input:   uint64(9223372036854775808),
			want:    0,
			wantErr: true,
		},
		{
			name:    "float overflow",
			input:   1e20,
			want:    0,
			wantErr: true,
		},
		// Invalid conversions
		{
			name:    "invalid type",
			input:   []int{1, 2, 3},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ToInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ToFloat Tests
// ============================================================================

func TestToFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		// Nil case
		{
			name:    "nil input",
			input:   nil,
			want:    0,
			wantErr: true,
		},
		// Float conversions
		{
			name:    "float64",
			input:   3.14,
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "float32",
			input:   float32(2.5),
			want:    2.5,
			wantErr: false,
		},
		// Integer conversions
		{
			name:    "int",
			input:   42,
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "int64",
			input:   int64(100),
			want:    100.0,
			wantErr: false,
		},
		{
			name:    "uint64",
			input:   uint64(200),
			want:    200.0,
			wantErr: false,
		},
		// Boolean conversions
		{
			name:    "bool true",
			input:   true,
			want:    1.0,
			wantErr: false,
		},
		{
			name:    "bool false",
			input:   false,
			want:    0.0,
			wantErr: false,
		},
		// String conversions
		{
			name:    "string float",
			input:   "3.14",
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "string int",
			input:   "42",
			want:    42.0,
			wantErr: false,
		},
		// Invalid conversions
		{
			name:    "invalid type",
			input:   map[string]int{"a": 1},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ToFloat() = %f, want %f", got, tt.want)
			}
		})
	}
}

// ============================================================================
// ToArray Tests
// ============================================================================

func TestToArray(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil input returns empty array",
			input:   nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "slice of ints",
			input:   []int{1, 2, 3},
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "slice of strings",
			input:   []string{"a", "b"},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "array of ints",
			input:   [3]int{1, 2, 3},
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "empty slice",
			input:   []int{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "JSON array string",
			input:   `[1,2,3]`,
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "plain string wraps as single element",
			input:   "hello",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "non-array type fails",
			input:   42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToArray(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToArray() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(got) != tt.wantLen {
				t.Errorf("ToArray() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// ============================================================================
// ToMap Tests
// ============================================================================

func TestToMap(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantKey string // Key to check existence
		wantErr bool
	}{
		{
			name:    "nil input returns empty map",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "map[string]interface{}",
			input:   map[string]interface{}{"a": 1, "b": 2},
			wantKey: "a",
			wantErr: false,
		},
		{
			name:    "JSON object string",
			input:   `{"x":10,"y":20}`,
			wantKey: "x",
			wantErr: false,
		},
		{
			name:    "map[string]string",
			input:   map[string]string{"key": "value"},
			wantKey: "key",
			wantErr: false,
		},
		{
			name:    "invalid JSON string",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "non-map type fails",
			input:   42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.wantKey != "" && got[tt.wantKey] == nil {
				t.Errorf("ToMap() missing expected key %q", tt.wantKey)
			}
		})
	}
}

// ============================================================================
// Type Checking Tests
// ============================================================================

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"int", 42, true},
		{"int64", int64(42), true},
		{"float64", 3.14, true},
		{"float32", float32(2.5), true},
		{"uint", uint(10), true},
		{"json.Number", json.Number("42"), true},
		{"string", "42", false},
		{"bool", true, false},
		{"nil", nil, false},
		{"array", []int{1, 2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNumeric(tt.input)
			if got != tt.want {
				t.Errorf("IsNumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsString(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"string", "hello", true},
		{"empty string", "", true},
		{"int", 42, false},
		{"nil", nil, false},
		{"byte slice", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsString(tt.input)
			if got != tt.want {
				t.Errorf("IsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsArray(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"int slice", []int{1, 2, 3}, true},
		{"string slice", []string{"a", "b"}, true},
		{"array", [3]int{1, 2, 3}, true},
		{"JSON array string", `[1,2,3]`, true},
		{"empty slice", []int{}, true},
		{"string", "hello", false},
		{"int", 42, false},
		{"map", map[string]int{}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsArray(tt.input)
			if got != tt.want {
				t.Errorf("IsArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMap(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"map[string]interface{}", map[string]interface{}{"a": 1}, true},
		{"map[string]string", map[string]string{"key": "val"}, true},
		{"JSON object string", `{"a":1}`, true},
		{"empty map", map[string]int{}, true},
		{"string", "hello", false},
		{"int", 42, false},
		{"array", []int{1, 2}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMap(tt.input)
			if got != tt.want {
				t.Errorf("IsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBool(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"string", "true", false},
		{"int", 1, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBool(tt.input)
			if got != tt.want {
				t.Errorf("IsBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNil(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"nil", nil, true},
		{"string", "hello", false},
		{"int", 0, false},
		{"empty string", "", false},
		{"false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNil(tt.input)
			if got != tt.want {
				t.Errorf("IsNil() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// GetType Tests
// ============================================================================

func TestGetType(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, "nil"},
		{"string", "hello", "string"},
		{"int", 42, "int"},
		{"float64", 3.14, "float64"},
		{"bool", true, "bool"},
		{"int slice", []int{1, 2}, "[]int"},
		{"map", map[string]int{}, "map[string]int"},
		{"json.Number", json.Number("42"), "json.Number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetType(tt.input)
			if got != tt.want {
				t.Errorf("GetType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		fn          func() error
		wantErrType error
	}{
		{
			name: "ToInt from non-numeric returns ErrTypeMismatch",
			fn: func() error {
				_, err := ToInt("not a number")
				return err
			},
			wantErrType: ErrTypeMismatch,
		},
		{
			name: "ToFloat from non-numeric returns ErrTypeMismatch",
			fn: func() error {
				_, err := ToFloat([]int{1, 2})
				return err
			},
			wantErrType: ErrTypeMismatch,
		},
		{
			name: "ToArray from non-array returns ErrTypeMismatch",
			fn: func() error {
				_, err := ToArray(42)
				return err
			},
			wantErrType: ErrTypeMismatch,
		},
		{
			name: "ToMap from invalid JSON returns ErrTypeMismatch",
			fn: func() error {
				_, err := ToMap("not json")
				return err
			},
			wantErrType: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErrType) {
				t.Errorf("error = %v, want wrapped %v", err, tt.wantErrType)
			}
		})
	}
}

func TestSpecialNumericValues(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		fn    func(interface{}) (float64, error)
		want  float64
	}{
		{
			name:  "ToFloat from max int64",
			input: int64(9223372036854775807),
			fn: func(v interface{}) (float64, error) {
				return ToFloat(v)
			},
			want: 9223372036854775807.0,
		},
		{
			name:  "ToFloat from negative zero",
			input: -0.0,
			fn: func(v interface{}) (float64, error) {
				return ToFloat(v)
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

// ============================================================================
// Roundtrip Tests
// ============================================================================

func TestRoundtripConversions(t *testing.T) {
	tests := []struct {
		name  string
		input int64
	}{
		{"zero", 0},
		{"positive", 42},
		{"negative", -42},
		{"large positive", 9223372036854775807},
		{"large negative", -9223372036854775808},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// int64 -> string -> int64
			str, err := ToString(tt.input)
			if err != nil {
				t.Fatalf("ToString failed: %v", err)
			}

			got, err := ParseInt(str)
			if err != nil {
				t.Fatalf("ParseInt failed: %v", err)
			}

			if got != tt.input {
				t.Errorf("roundtrip failed: %d -> %q -> %d", tt.input, str, got)
			}
		})
	}
}
