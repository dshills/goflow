package tui

import (
	"strings"
	"testing"
)

// TestValidateTextField tests text field validation
func TestValidateTextField(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "empty string",
			value:     "",
			wantError: false,
		},
		{
			name:      "normal text",
			value:     "workflow-step-1",
			wantError: false,
		},
		{
			name:      "text with spaces",
			value:     "My Workflow Name",
			wantError: false,
		},
		{
			name:      "text at max length",
			value:     strings.Repeat("a", 256),
			wantError: false,
		},
		{
			name:      "text exceeds max length",
			value:     strings.Repeat("a", 257),
			wantError: true,
		},
		{
			name:      "text with special characters",
			value:     "workflow_v1.2-beta",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTextField(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateTextField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateExpressionField tests expression field validation
func TestValidateExpressionField(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "empty expression",
			value:     "",
			wantError: false,
		},
		{
			name:      "simple expression",
			value:     "total + 1",
			wantError: false,
		},
		{
			name:      "complex expression",
			value:     `items`,
			wantError: false,
		},
		{
			name:      "expression with variables",
			value:     "user.age * 2",
			wantError: false,
		},
		{
			name:      "expression with function",
			value:     "len(items) > 0",
			wantError: false,
		},
		{
			name:      "invalid syntax",
			value:     "total +",
			wantError: true,
		},
		{
			name:      "unsafe operation - os",
			value:     "os.Exit(1)",
			wantError: true,
		},
		{
			name:      "unsafe operation - exec",
			value:     "exec.Command('rm', '-rf', '/')",
			wantError: true,
		},
		{
			name:      "unsafe operation - http",
			value:     "http.Get('evil.com')",
			wantError: true,
		},
		{
			name:      "unsafe operation - syscall",
			value:     "syscall.Kill(pid, 9)",
			wantError: true,
		},
		{
			name:      "unsafe operation - unsafe",
			value:     "unsafe.Pointer(x)",
			wantError: true,
		},
		{
			name:      "ternary expression",
			value:     "total > 10 ? 'many' : 'few'",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpressionField(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateExpressionField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateConditionField tests condition field validation
func TestValidateConditionField(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "empty condition",
			value:     "",
			wantError: false,
		},
		{
			name:      "boolean literal true",
			value:     "true",
			wantError: false,
		},
		{
			name:      "boolean literal false",
			value:     "false",
			wantError: false,
		},
		{
			name:      "simple comparison",
			value:     "total > 10",
			wantError: false,
		},
		{
			name:      "equality check",
			value:     "status == 'active'",
			wantError: false,
		},
		{
			name:      "logical AND",
			value:     "total > 10 && status == 'active'",
			wantError: false,
		},
		{
			name:      "logical OR",
			value:     "total > 10 || urgent == true",
			wantError: false,
		},
		{
			name:      "complex condition",
			value:     "(total > 10 && status == 'active') || priority == 1",
			wantError: false,
		},
		{
			name:      "negation",
			value:     "!disabled",
			wantError: false,
		},
		{
			name:      "invalid syntax",
			value:     "total >",
			wantError: true,
		},
		{
			name:      "unsafe operation",
			value:     "os.Exit(1) == nil",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConditionField(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateConditionField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateJSONPathField tests JSONPath field validation
func TestValidateJSONPathField(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "empty path",
			value:     "",
			wantError: false,
		},
		{
			name:      "simple path",
			value:     "$.users",
			wantError: false,
		},
		{
			name:      "nested path",
			value:     "$.users[0].email",
			wantError: false,
		},
		{
			name:      "array access",
			value:     "$.items[0]",
			wantError: false,
		},
		{
			name:      "wildcard",
			value:     "$.users[*].name",
			wantError: false,
		},
		{
			name:      "filter expression",
			value:     "$.users[?(@.age > 18)]",
			wantError: false,
		},
		{
			name:      "complex filter",
			value:     "$.users[?(@.age > 18 && @.active == true)].email",
			wantError: false,
		},
		{
			name:      "current object reference",
			value:     "@.field",
			wantError: false,
		},
		{
			name:      "missing $ or @",
			value:     "users.email",
			wantError: true,
		},
		{
			name:      "unbalanced brackets",
			value:     "$.users[0",
			wantError: true,
		},
		{
			name:      "double brackets invalid",
			value:     "$.users[[0]]",
			wantError: true,
		},
		{
			name:      "recursive descent",
			value:     "$..email",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONPathField(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateJSONPathField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateTemplateField tests template field validation
func TestValidateTemplateField(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{
			name:      "empty template",
			value:     "",
			wantError: false,
		},
		{
			name:      "no placeholders",
			value:     "Hello World",
			wantError: false,
		},
		{
			name:      "simple placeholder",
			value:     "Hello ${name}",
			wantError: false,
		},
		{
			name:      "multiple placeholders",
			value:     "Hello ${user.name}, you have ${count} items",
			wantError: false,
		},
		{
			name:      "nested property access",
			value:     "${user.profile.email}",
			wantError: false,
		},
		{
			name:      "array index",
			value:     "${items[0]}",
			wantError: false,
		},
		{
			name:      "complex template",
			value:     "User: ${user.name} (${user.email}), Items: ${items[0].name}",
			wantError: false,
		},
		{
			name:      "unbalanced braces - missing close",
			value:     "Hello ${name",
			wantError: true,
		},
		{
			name:      "unbalanced braces - missing open",
			value:     "Hello name}",
			wantError: false, // Valid - not a placeholder
		},
		{
			name:      "empty placeholder",
			value:     "Hello ${}",
			wantError: true, // Empty placeholder should be invalid
		},
		{
			name:      "invalid variable name - starts with number",
			value:     "${1user}",
			wantError: true,
		},
		{
			name:      "invalid variable name - special chars",
			value:     "${user@name}",
			wantError: true,
		},
		{
			name:      "valid underscore",
			value:     "${user_name}",
			wantError: false,
		},
		{
			name:      "whitespace in placeholder",
			value:     "${ user_name }",
			wantError: true, // Whitespace around variable not allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateField(tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateTemplateField() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestNewPropertyField tests property field creation
func TestNewPropertyField(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		value     string
		fieldType string
		required  bool
		wantValid bool
	}{
		{
			name:      "text field",
			label:     "Name",
			value:     "test-name",
			fieldType: "text",
			required:  true,
			wantValid: false, // not validated yet
		},
		{
			name:      "expression field",
			label:     "Expression",
			value:     "count + 1",
			fieldType: "expression",
			required:  true,
			wantValid: false,
		},
		{
			name:      "condition field",
			label:     "Condition",
			value:     "count > 10",
			fieldType: "condition",
			required:  false,
			wantValid: false,
		},
		{
			name:      "jsonpath field",
			label:     "Path",
			value:     "$.users",
			fieldType: "jsonpath",
			required:  true,
			wantValid: false,
		},
		{
			name:      "template field",
			label:     "Template",
			value:     "Hello ${name}",
			fieldType: "template",
			required:  false,
			wantValid: false,
		},
		{
			name:      "unknown field type falls back to text",
			label:     "Unknown",
			value:     "value",
			fieldType: "unknown",
			required:  false,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := newPropertyField(tt.label, tt.value, tt.fieldType, tt.required)

			if field.label != tt.label {
				t.Errorf("label = %v, want %v", field.label, tt.label)
			}
			if field.value != tt.value {
				t.Errorf("value = %v, want %v", field.value, tt.value)
			}
			if field.required != tt.required {
				t.Errorf("required = %v, want %v", field.required, tt.required)
			}
			if field.valid != tt.wantValid {
				t.Errorf("valid = %v, want %v", field.valid, tt.wantValid)
			}
			if field.validationFn == nil {
				t.Error("validationFn should not be nil")
			}
			if field.helpText == "" {
				t.Error("helpText should not be empty")
			}
		})
	}
}

// TestPropertyFieldValidate tests the validate method
func TestPropertyFieldValidate(t *testing.T) {
	tests := []struct {
		name      string
		field     propertyField
		wantError bool
		wantValid bool
	}{
		{
			name:      "valid text field",
			field:     newPropertyField("Name", "test-name", "text", true),
			wantError: false,
			wantValid: true,
		},
		{
			name:      "invalid text field - too long",
			field:     newPropertyField("Name", strings.Repeat("a", 300), "text", true),
			wantError: true,
			wantValid: false,
		},
		{
			name:      "valid expression",
			field:     newPropertyField("Expr", "total + 1", "expression", true),
			wantError: false,
			wantValid: true,
		},
		{
			name:      "invalid expression - syntax error",
			field:     newPropertyField("Expr", "total +", "expression", true),
			wantError: true,
			wantValid: false,
		},
		{
			name:      "valid condition",
			field:     newPropertyField("Condition", "total > 10", "condition", true),
			wantError: false,
			wantValid: true,
		},
		{
			name:      "valid JSONPath",
			field:     newPropertyField("Path", "$.users[0]", "jsonpath", true),
			wantError: false,
			wantValid: true,
		},
		{
			name:      "invalid JSONPath - no $",
			field:     newPropertyField("Path", "users", "jsonpath", true),
			wantError: true,
			wantValid: false,
		},
		{
			name:      "valid template",
			field:     newPropertyField("Template", "Hello ${name}", "template", true),
			wantError: false,
			wantValid: true,
		},
		{
			name:      "invalid template - unbalanced",
			field:     newPropertyField("Template", "Hello ${name", "template", true),
			wantError: true,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.validate()
			if (err != nil) != tt.wantError {
				t.Errorf("validate() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.field.valid != tt.wantValid {
				t.Errorf("valid = %v, want %v", tt.field.valid, tt.wantValid)
			}
		})
	}
}

// TestHasBalancedBrackets tests bracket balancing helper
func TestHasBalancedBrackets(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", true},
		{"no brackets", "hello", true},
		{"balanced single", "[0]", true},
		{"balanced multiple", "[0][1]", true},
		{"balanced nested", "[[0]]", true},
		{"unbalanced open", "[0", false},
		{"unbalanced close", "0]", false},
		{"unbalanced multiple", "[[0]", false},
		{"complex balanced", "$.users[0].items[1]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBalancedBrackets(tt.input); got != tt.want {
				t.Errorf("hasBalancedBrackets() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasBalancedTemplateBraces tests template brace balancing
func TestHasBalancedTemplateBraces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", true},
		{"no braces", "hello", true},
		{"balanced single", "${name}", true},
		{"balanced multiple", "${name} ${age}", true},
		{"unbalanced open", "${name", false},
		{"unbalanced close", "name}", true},     // } without ${ is not a placeholder
		{"nested balanced", "${${name}}", true}, // `hasBalancedTemplateBraces` only checks ${..} balance, not semantic nesting
		{"complex balanced", "Hello ${user.name}, age: ${user.age}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBalancedTemplateBraces(tt.input); got != tt.want {
				t.Errorf("hasBalancedTemplateBraces() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFindTemplatePlaceholders tests placeholder extraction
func TestFindTemplatePlaceholders(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "no placeholders",
			input: "Hello World",
			want:  []string{},
		},
		{
			name:  "single placeholder",
			input: "Hello ${name}",
			want:  []string{"name"},
		},
		{
			name:  "multiple placeholders",
			input: "Hello ${first} ${last}",
			want:  []string{"first", "last"},
		},
		{
			name:  "nested property",
			input: "${user.profile.email}",
			want:  []string{"user.profile.email"},
		},
		{
			name:  "array index",
			input: "${items[0]}",
			want:  []string{"items[0]"},
		},
		{
			name:  "complex template",
			input: "User: ${user.name} (${user.email}), Item: ${items[0].name}",
			want:  []string{"user.name", "user.email", "items[0].name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTemplatePlaceholders(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("findTemplatePlaceholders() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("findTemplatePlaceholders()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
