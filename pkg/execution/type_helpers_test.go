package execution

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dshills/goflow/pkg/domain/execution"
)

// TestErrorAsPattern tests the errors.As pattern for error type checking
func TestErrorAsPattern(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantErrorType execution.ErrorType
		wantFound     bool
	}{
		{
			name:          "direct ExecutionError",
			err:           &execution.ExecutionError{Type: execution.ErrorTypeValidation, Message: "test"},
			wantErrorType: execution.ErrorTypeValidation,
			wantFound:     true,
		},
		{
			name:          "wrapped ExecutionError",
			err:           fmt.Errorf("wrapped: %w", &execution.ExecutionError{Type: execution.ErrorTypeConnection, Message: "test"}),
			wantErrorType: execution.ErrorTypeConnection,
			wantFound:     true,
		},
		{
			name:          "non-ExecutionError",
			err:           errors.New("standard error"),
			wantErrorType: "",
			wantFound:     false,
		},
		{
			name:          "nil error",
			err:           nil,
			wantErrorType: "",
			wantFound:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var execErr *execution.ExecutionError
			found := errors.As(tt.err, &execErr)

			if found != tt.wantFound {
				t.Errorf("errors.As() found = %v, want %v", found, tt.wantFound)
			}

			if found && execErr.Type != tt.wantErrorType {
				t.Errorf("ExecutionError.Type = %v, want %v", execErr.Type, tt.wantErrorType)
			}
		})
	}
}

// TestTypeSwitchCompleteness verifies type switches handle all expected types
func TestTypeSwitchCompleteness(t *testing.T) {
	// Test convertToSlice function which uses type switch
	tests := []struct {
		name       string
		collection interface{}
		wantLen    int
		wantErr    bool
	}{
		{
			name:       "[]interface{}",
			collection: []interface{}{1, 2, 3},
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:       "[]string",
			collection: []string{"a", "b", "c"},
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:       "[]int",
			collection: []int{1, 2, 3},
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:       "[]float64",
			collection: []float64{1.1, 2.2, 3.3},
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:       "[]bool",
			collection: []bool{true, false, true},
			wantLen:    3,
			wantErr:    false,
		},
		{
			name:       "unsupported type",
			collection: "not a slice",
			wantLen:    0,
			wantErr:    true,
		},
		{
			name:       "nil collection",
			collection: nil,
			wantLen:    0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToSlice(tt.collection)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertToSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("convertToSlice() returned slice length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestBooleanTypeAssertion tests runtime type checks for boolean values
func TestBooleanTypeAssertion(t *testing.T) {
	tests := []struct {
		name    string
		result  interface{}
		wantVal bool
		wantErr bool
	}{
		{
			name:    "valid true",
			result:  true,
			wantVal: true,
			wantErr: false,
		},
		{
			name:    "valid false",
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
		{
			name:    "nil instead of bool",
			result:  nil,
			wantVal: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This pattern simulates what's in loop.go:170
			broken, ok := tt.result.(bool)

			if ok == tt.wantErr {
				t.Errorf("type assertion ok = %v, wantErr %v", ok, tt.wantErr)
			}

			if !tt.wantErr && broken != tt.wantVal {
				t.Errorf("boolean value = %v, want %v", broken, tt.wantVal)
			}
		})
	}
}

// TestMapTypeAssertion tests runtime type checks for map access
func TestMapTypeAssertion(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantOk  bool
		wantLen int
	}{
		{
			name:    "valid map",
			value:   map[string]interface{}{"key": "value"},
			wantOk:  true,
			wantLen: 1,
		},
		{
			name:    "empty map",
			value:   map[string]interface{}{},
			wantOk:  true,
			wantLen: 0,
		},
		{
			name:    "string instead of map",
			value:   "not a map",
			wantOk:  false,
			wantLen: 0,
		},
		{
			name:    "slice instead of map",
			value:   []string{"a", "b"},
			wantOk:  false,
			wantLen: 0,
		},
		{
			name:    "nil instead of map",
			value:   nil,
			wantOk:  false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This pattern simulates what's in node_executor.go:235
			m, ok := tt.value.(map[string]interface{})

			if ok != tt.wantOk {
				t.Errorf("type assertion ok = %v, want %v", ok, tt.wantOk)
			}

			if ok && len(m) != tt.wantLen {
				t.Errorf("map length = %d, want %d", len(m), tt.wantLen)
			}
		})
	}
}

// BenchmarkTypeAssertion benchmarks type assertion vs type switch
func BenchmarkTypeAssertion(b *testing.B) {
	value := "test string"

	b.Run("type_assertion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if s, ok := interface{}(value).(string); ok {
				_ = s
			}
		}
	})

	b.Run("type_switch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			switch v := interface{}(value).(type) {
			case string:
				_ = v
			}
		}
	})
}

// BenchmarkErrorsAs benchmarks errors.As vs type assertion
func BenchmarkErrorsAs(b *testing.B) {
	// Use interface type to allow type assertions
	var err error = &execution.ExecutionError{Type: execution.ErrorTypeValidation, Message: "test"}

	b.Run("errors.As", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var execErr *execution.ExecutionError
			if errors.As(err, &execErr) {
				_ = execErr.Type
			}
		}
	})

	b.Run("type_assertion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if execErr, ok := err.(*execution.ExecutionError); ok {
				_ = execErr.Type
			}
		}
	})
}
