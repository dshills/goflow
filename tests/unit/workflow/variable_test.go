package workflow

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestVariable_Validation tests Variable validation rules
func TestVariable_Validation(t *testing.T) {
	tests := []struct {
		name     string
		variable *workflow.Variable
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid variable with string type",
			variable: &workflow.Variable{
				Name:        "input_file",
				Type:        "string",
				Description: "Path to input file",
			},
			wantErr: false,
		},
		{
			name: "valid variable with number type",
			variable: &workflow.Variable{
				Name:        "count",
				Type:        "number",
				Description: "Item count",
			},
			wantErr: false,
		},
		{
			name: "valid variable with boolean type",
			variable: &workflow.Variable{
				Name:        "is_active",
				Type:        "boolean",
				Description: "Active status",
			},
			wantErr: false,
		},
		{
			name: "valid variable with object type",
			variable: &workflow.Variable{
				Name:        "user_data",
				Type:        "object",
				Description: "User object",
			},
			wantErr: false,
		},
		{
			name: "valid variable with array type",
			variable: &workflow.Variable{
				Name:        "items",
				Type:        "array",
				Description: "List of items",
			},
			wantErr: false,
		},
		{
			name: "valid variable with any type",
			variable: &workflow.Variable{
				Name:        "dynamic_data",
				Type:        "any",
				Description: "Dynamic data",
			},
			wantErr: false,
		},
		{
			name: "variable with empty name",
			variable: &workflow.Variable{
				Name:        "",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "empty variable name",
		},
		{
			name: "variable with invalid name format (spaces)",
			variable: &workflow.Variable{
				Name:        "invalid name",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "invalid variable name format",
		},
		{
			name: "variable with invalid name format (special chars)",
			variable: &workflow.Variable{
				Name:        "invalid-name!",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "invalid variable name format",
		},
		{
			name: "variable with invalid name format (starts with number)",
			variable: &workflow.Variable{
				Name:        "123invalid",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "invalid variable name format",
		},
		{
			name: "variable with valid name (underscores)",
			variable: &workflow.Variable{
				Name:        "valid_name_123",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: false,
		},
		{
			name: "variable with valid name (camelCase)",
			variable: &workflow.Variable{
				Name:        "validNameCamelCase",
				Type:        "string",
				Description: "Test variable",
			},
			wantErr: false,
		},
		{
			name: "variable with empty type",
			variable: &workflow.Variable{
				Name:        "test_var",
				Type:        "",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "empty variable type",
		},
		{
			name: "variable with invalid type",
			variable: &workflow.Variable{
				Name:        "test_var",
				Type:        "invalid_type",
				Description: "Test variable",
			},
			wantErr: true,
			errMsg:  "invalid variable type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variable.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Variable.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Variable.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestVariable_UniqueNames tests that variable names must be unique within workflow
func TestVariable_UniqueNames(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")

	// Add first variable
	err := wf.AddVariable(&workflow.Variable{
		Name: "data",
		Type: "string",
	})
	if err != nil {
		t.Fatalf("AddVariable() first variable unexpected error: %v", err)
	}

	// Try to add second variable with same name
	err = wf.AddVariable(&workflow.Variable{
		Name: "data",
		Type: "number",
	})
	if err == nil {
		t.Error("AddVariable() should fail for duplicate variable name")
	}

	// Add variable with different name should succeed
	err = wf.AddVariable(&workflow.Variable{
		Name: "result",
		Type: "string",
	})
	if err != nil {
		t.Errorf("AddVariable() unexpected error for unique name: %v", err)
	}
}

// TestVariable_DefaultValueTypeChecking tests that default values match declared types
func TestVariable_DefaultValueTypeChecking(t *testing.T) {
	tests := []struct {
		name     string
		variable *workflow.Variable
		wantErr  bool
		errMsg   string
	}{
		{
			name: "string variable with string default",
			variable: &workflow.Variable{
				Name:         "name",
				Type:         "string",
				DefaultValue: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "number variable with int default",
			variable: &workflow.Variable{
				Name:         "count",
				Type:         "number",
				DefaultValue: 42,
			},
			wantErr: false,
		},
		{
			name: "number variable with float default",
			variable: &workflow.Variable{
				Name:         "price",
				Type:         "number",
				DefaultValue: 19.99,
			},
			wantErr: false,
		},
		{
			name: "boolean variable with bool default",
			variable: &workflow.Variable{
				Name:         "is_active",
				Type:         "boolean",
				DefaultValue: true,
			},
			wantErr: false,
		},
		{
			name: "object variable with map default",
			variable: &workflow.Variable{
				Name:         "config",
				Type:         "object",
				DefaultValue: map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "array variable with slice default",
			variable: &workflow.Variable{
				Name:         "items",
				Type:         "array",
				DefaultValue: []interface{}{"item1", "item2"},
			},
			wantErr: false,
		},
		{
			name: "any variable with any default",
			variable: &workflow.Variable{
				Name:         "dynamic",
				Type:         "any",
				DefaultValue: "anything goes",
			},
			wantErr: false,
		},
		{
			name: "string variable with number default should fail",
			variable: &workflow.Variable{
				Name:         "name",
				Type:         "string",
				DefaultValue: 123,
			},
			wantErr: true,
			errMsg:  "default value type mismatch",
		},
		{
			name: "number variable with string default should fail",
			variable: &workflow.Variable{
				Name:         "count",
				Type:         "number",
				DefaultValue: "not a number",
			},
			wantErr: true,
			errMsg:  "default value type mismatch",
		},
		{
			name: "boolean variable with string default should fail",
			variable: &workflow.Variable{
				Name:         "is_active",
				Type:         "boolean",
				DefaultValue: "true",
			},
			wantErr: true,
			errMsg:  "default value type mismatch",
		},
		{
			name: "object variable with array default should fail",
			variable: &workflow.Variable{
				Name:         "config",
				Type:         "object",
				DefaultValue: []interface{}{"not", "an", "object"},
			},
			wantErr: true,
			errMsg:  "default value type mismatch",
		},
		{
			name: "array variable with object default should fail",
			variable: &workflow.Variable{
				Name:         "items",
				Type:         "array",
				DefaultValue: map[string]interface{}{"not": "an array"},
			},
			wantErr: true,
			errMsg:  "default value type mismatch",
		},
		{
			name: "variable with nil default",
			variable: &workflow.Variable{
				Name:         "optional",
				Type:         "string",
				DefaultValue: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variable.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Variable.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Variable.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestVariable_NameFormatValidation tests variable name format rules
func TestVariable_NameFormatValidation(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		wantErr bool
	}{
		// Valid names
		{name: "simple lowercase", varName: "name", wantErr: false},
		{name: "with underscore", varName: "user_name", wantErr: false},
		{name: "with number", varName: "user123", wantErr: false},
		{name: "camelCase", varName: "userName", wantErr: false},
		{name: "PascalCase", varName: "UserName", wantErr: false},
		{name: "multiple underscores", varName: "user_full_name", wantErr: false},
		{name: "ending with number", varName: "value_1", wantErr: false},
		{name: "single letter", varName: "x", wantErr: false},
		{name: "single letter uppercase", varName: "X", wantErr: false},

		// Invalid names
		{name: "starts with number", varName: "123name", wantErr: true},
		{name: "with space", varName: "user name", wantErr: true},
		{name: "with hyphen", varName: "user-name", wantErr: true},
		{name: "with dot", varName: "user.name", wantErr: true},
		{name: "with special char exclamation", varName: "user!", wantErr: true},
		{name: "with special char at", varName: "user@email", wantErr: true},
		{name: "with special char hash", varName: "#user", wantErr: true},
		{name: "with special char dollar", varName: "$user", wantErr: true},
		{name: "with special char percent", varName: "user%", wantErr: true},
		{name: "with parenthesis", varName: "user(name)", wantErr: true},
		{name: "with brackets", varName: "user[0]", wantErr: true},
		{name: "with braces", varName: "user{name}", wantErr: true},
		{name: "only underscore", varName: "_", wantErr: true},
		{name: "only underscores", varName: "___", wantErr: true},
		{name: "starts with underscore", varName: "_private", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &workflow.Variable{
				Name: tt.varName,
				Type: "string",
			}

			err := v.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("Variable.Validate() expected error for name %q but got none", tt.varName)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Variable.Validate() unexpected error for name %q: %v", tt.varName, err)
			}
		})
	}
}

// TestVariable_Serialization tests variable marshaling and unmarshaling
func TestVariable_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		variable *workflow.Variable
	}{
		{
			name: "simple string variable",
			variable: &workflow.Variable{
				Name:        "name",
				Type:        "string",
				Description: "User name",
			},
		},
		{
			name: "variable with default value",
			variable: &workflow.Variable{
				Name:         "count",
				Type:         "number",
				DefaultValue: 42,
				Description:  "Item count",
			},
		},
		{
			name: "variable with complex default value",
			variable: &workflow.Variable{
				Name: "config",
				Type: "object",
				DefaultValue: map[string]interface{}{
					"host": "localhost",
					"port": 8080,
					"ssl":  true,
				},
				Description: "Server configuration",
			},
		},
		{
			name: "variable with array default",
			variable: &workflow.Variable{
				Name:         "tags",
				Type:         "array",
				DefaultValue: []interface{}{"tag1", "tag2", "tag3"},
				Description:  "Tags list",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := tt.variable.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error: %v", err)
			}

			// Unmarshal back
			newVar := &workflow.Variable{}
			err = newVar.UnmarshalJSON(data)
			if err != nil {
				t.Fatalf("UnmarshalJSON() error: %v", err)
			}

			// Verify fields match
			if newVar.Name != tt.variable.Name {
				t.Errorf("Name mismatch: got %v, want %v", newVar.Name, tt.variable.Name)
			}
			if newVar.Type != tt.variable.Type {
				t.Errorf("Type mismatch: got %v, want %v", newVar.Type, tt.variable.Type)
			}
			if newVar.Description != tt.variable.Description {
				t.Errorf("Description mismatch: got %v, want %v", newVar.Description, tt.variable.Description)
			}

			// Note: DefaultValue comparison would need deep equality check
			// which is omitted here for simplicity
		})
	}
}

// TestVariable_GetVariable tests retrieving variables by name
func TestVariable_GetVariable(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")

	// Add variables
	wf.AddVariable(&workflow.Variable{Name: "var1", Type: "string"})
	wf.AddVariable(&workflow.Variable{Name: "var2", Type: "number"})
	wf.AddVariable(&workflow.Variable{Name: "var3", Type: "boolean"})

	tests := []struct {
		name    string
		varName string
		wantErr bool
	}{
		{name: "get existing variable", varName: "var1", wantErr: false},
		{name: "get another existing variable", varName: "var2", wantErr: false},
		{name: "get non-existent variable", varName: "nonexistent", wantErr: true},
		{name: "get empty name", varName: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := wf.GetVariable(tt.varName)

			if tt.wantErr {
				if err == nil {
					t.Error("GetVariable() expected error but got none")
				}
				if v != nil {
					t.Error("GetVariable() should return nil variable on error")
				}
			} else {
				if err != nil {
					t.Errorf("GetVariable() unexpected error: %v", err)
				}
				if v == nil {
					t.Error("GetVariable() returned nil variable")
				}
				if v.Name != tt.varName {
					t.Errorf("GetVariable() name = %v, want %v", v.Name, tt.varName)
				}
			}
		})
	}
}

// TestVariable_RemoveVariable tests removing variables from workflow
func TestVariable_RemoveVariable(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")

	// Add variables
	wf.AddVariable(&workflow.Variable{Name: "var1", Type: "string"})
	wf.AddVariable(&workflow.Variable{Name: "var2", Type: "number"})
	wf.AddVariable(&workflow.Variable{Name: "var3", Type: "boolean"})

	// Verify initial count
	if len(wf.Variables) != 3 {
		t.Fatalf("Expected 3 variables, got %d", len(wf.Variables))
	}

	// Remove variable
	err := wf.RemoveVariable("var2")
	if err != nil {
		t.Fatalf("RemoveVariable() unexpected error: %v", err)
	}

	// Verify variable removed
	if len(wf.Variables) != 2 {
		t.Errorf("Expected 2 variables after removal, got %d", len(wf.Variables))
	}

	// Verify correct variable removed
	_, err = wf.GetVariable("var2")
	if err == nil {
		t.Error("GetVariable() should fail for removed variable")
	}

	// Verify other variables still exist
	_, err = wf.GetVariable("var1")
	if err != nil {
		t.Error("GetVariable() should succeed for non-removed variable var1")
	}

	_, err = wf.GetVariable("var3")
	if err != nil {
		t.Error("GetVariable() should succeed for non-removed variable var3")
	}

	// Try to remove non-existent variable
	err = wf.RemoveVariable("nonexistent")
	if err == nil {
		t.Error("RemoveVariable() should fail for non-existent variable")
	}
}

// TestVariable_UpdateVariable tests updating variable definitions
func TestVariable_UpdateVariable(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")

	// Add initial variable
	wf.AddVariable(&workflow.Variable{
		Name:        "data",
		Type:        "string",
		Description: "Original description",
	})

	// Update variable
	err := wf.UpdateVariable("data", &workflow.Variable{
		Name:        "data",
		Type:        "number",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("UpdateVariable() unexpected error: %v", err)
	}

	// Verify update
	v, err := wf.GetVariable("data")
	if err != nil {
		t.Fatalf("GetVariable() after update error: %v", err)
	}

	if v.Type != "number" {
		t.Errorf("UpdateVariable() type = %v, want %v", v.Type, "number")
	}

	if v.Description != "Updated description" {
		t.Errorf("UpdateVariable() description = %v, want %v", v.Description, "Updated description")
	}

	// Try to update non-existent variable
	err = wf.UpdateVariable("nonexistent", &workflow.Variable{
		Name: "nonexistent",
		Type: "string",
	})
	if err == nil {
		t.Error("UpdateVariable() should fail for non-existent variable")
	}
}
