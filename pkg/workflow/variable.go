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
	Required     bool        `json:"required,omitempty" yaml:"required,omitempty"`
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

	// If variable is required, it should not have a default value
	// (having both doesn't make semantic sense - if it's required, user must provide it)
	if v.Required && v.DefaultValue != nil {
		return fmt.Errorf("variable: required variable %s cannot have a default value", v.Name)
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
		if _, err := validateType[string](v.DefaultValue, v.Name); err != nil {
			return err
		}
	case "number":
		// Accept both int and float types as numbers using helper function
		if !isNumericType(v.DefaultValue) {
			return fmt.Errorf("variable: default value type mismatch for %s: expected number, got %T", v.Name, v.DefaultValue)
		}
	case "boolean":
		if _, err := validateType[bool](v.DefaultValue, v.Name); err != nil {
			return err
		}
	case "object":
		if _, err := validateType[map[string]interface{}](v.DefaultValue, v.Name); err != nil {
			return err
		}
	case "array":
		// Check for array types using helper function
		if !isArrayType(v.DefaultValue) {
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
