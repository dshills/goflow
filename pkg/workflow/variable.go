package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Variable represents a workflow-scoped variable
type Variable struct {
	Name         string      `json:"name" yaml:"name"`
	Type         string      `json:"type" yaml:"type"`
	DefaultValue interface{} `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	Description  string      `json:"description,omitempty" yaml:"description,omitempty"`
}

// validVariableNameRegex matches valid variable names (alphanumeric + underscore, not starting with underscore or number)
var validVariableNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// validVariableTypes are the allowed variable types
var validVariableTypes = map[string]bool{
	"string":  true,
	"number":  true,
	"boolean": true,
	"object":  true,
	"array":   true,
	"any":     true,
}

// Validate checks if the variable is valid
func (v *Variable) Validate() error {
	if v.Name == "" {
		return errors.New("variable: empty variable name")
	}

	// Check variable name format
	if !validVariableNameRegex.MatchString(v.Name) {
		return fmt.Errorf("variable: invalid variable name format: %s (must start with letter, contain only alphanumeric and underscore)", v.Name)
	}

	if v.Type == "" {
		return errors.New("variable: empty variable type")
	}

	// Check if type is valid
	if !validVariableTypes[v.Type] {
		return fmt.Errorf("variable: invalid variable type: %s (must be one of: string, number, boolean, object, array, any)", v.Type)
	}

	// Check default value type matches declared type (if provided)
	if v.DefaultValue != nil {
		if err := v.validateDefaultValueType(); err != nil {
			return err
		}
	}

	return nil
}

// validateDefaultValueType checks if the default value matches the declared type
func (v *Variable) validateDefaultValueType() error {
	if v.DefaultValue == nil {
		return nil
	}

	// "any" type accepts any value
	if v.Type == "any" {
		return nil
	}

	switch v.Type {
	case "string":
		if _, ok := v.DefaultValue.(string); !ok {
			return fmt.Errorf("variable: default value type mismatch for %s: expected string, got %T", v.Name, v.DefaultValue)
		}
	case "number":
		// Accept both int and float types as numbers
		switch v.DefaultValue.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			// Valid number type
		default:
			return fmt.Errorf("variable: default value type mismatch for %s: expected number, got %T", v.Name, v.DefaultValue)
		}
	case "boolean":
		if _, ok := v.DefaultValue.(bool); !ok {
			return fmt.Errorf("variable: default value type mismatch for %s: expected boolean, got %T", v.Name, v.DefaultValue)
		}
	case "object":
		if _, ok := v.DefaultValue.(map[string]interface{}); !ok {
			return fmt.Errorf("variable: default value type mismatch for %s: expected object (map), got %T", v.Name, v.DefaultValue)
		}
	case "array":
		switch v.DefaultValue.(type) {
		case []interface{}, []string, []int, []float64, []bool, []map[string]interface{}:
			// Valid array types
		default:
			return fmt.Errorf("variable: default value type mismatch for %s: expected array (slice), got %T", v.Name, v.DefaultValue)
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Variable
func (v *Variable) MarshalJSON() ([]byte, error) {
	type Alias Variable
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(v),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Variable
func (v *Variable) UnmarshalJSON(data []byte) error {
	type Alias Variable
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(v),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
