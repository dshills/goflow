package workflow

import (
	"testing"
)

func TestExtractVariableReferences(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected []string
	}{
		{
			name:     "simple variable",
			expr:     "count > 10",
			expected: []string{"count"},
		},
		{
			name:     "multiple variables",
			expr:     "price > 100 && quantity < 50",
			expected: []string{"price", "quantity"},
		},
		{
			name:     "variable with function",
			expr:     "contains(email, '@example.com')",
			expected: []string{"email"},
		},
		{
			name:     "no variables (only keywords)",
			expr:     "true && false",
			expected: []string{},
		},
		{
			name:     "complex expression",
			expr:     "user.age >= 18 && user.verified == true",
			expected: []string{"user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariableReferences(tt.expr)

			// Convert to map for easier comparison (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, v := range result {
				resultMap[v] = true
			}

			expectedMap := make(map[string]bool)
			for _, v := range tt.expected {
				expectedMap[v] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("extractVariableReferences() got %v, want %v", result, tt.expected)
				return
			}

			for v := range expectedMap {
				if !resultMap[v] {
					t.Errorf("extractVariableReferences() missing variable: %s", v)
				}
			}
		})
	}
}

func TestValidateExpressionSyntax(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "valid comparison",
			expr:    "count > 10",
			wantErr: false,
		},
		{
			name:    "valid logical expression",
			expr:    "active == true && count > 0",
			wantErr: false,
		},
		{
			name:    "valid function call with string contains",
			expr:    "email contains 'test'",
			wantErr: false,
		},
		{
			name:    "invalid syntax - unclosed parenthesis",
			expr:    "contains(email, 'test'",
			wantErr: true,
		},
		{
			name:    "unsafe operation - os package",
			expr:    "os.ReadFile('/etc/passwd')",
			wantErr: true,
		},
		{
			name:    "unsafe operation - exec",
			expr:    "exec.Command('ls')",
			wantErr: true,
		},
		{
			name:    "empty expression",
			expr:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExpressionSyntax(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExpressionSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSONPathSyntax(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid simple path",
			path:    "$.users[0].name",
			wantErr: false,
		},
		{
			name:    "valid wildcard",
			path:    "$.items[*].price",
			wantErr: false,
		},
		{
			name:    "valid filter",
			path:    "$.products[?(@.price < 100)]",
			wantErr: false,
		},
		{
			name:    "missing dollar sign",
			path:    "users[0].name",
			wantErr: true,
		},
		{
			name:    "unclosed bracket",
			path:    "$.users[0.name",
			wantErr: true,
		},
		{
			name:    "unclosed parenthesis in filter",
			path:    "$.products[?(@.price < 100]",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "unmatched closing bracket",
			path:    "$.users]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONPathSyntax(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJSONPathSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainsTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "contains template",
			input:    "Hello ${name}",
			expected: true,
		},
		{
			name:     "no template",
			input:    "Hello world",
			expected: false,
		},
		{
			name:     "escaped template",
			input:    "Hello \\${name}",
			expected: true, // Still contains ${ even if escaped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsTemplate(tt.input)
			if result != tt.expected {
				t.Errorf("containsTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateTemplateSyntax(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		wantErr bool
	}{
		{
			name:    "valid simple template",
			tmpl:    "Hello ${name}",
			wantErr: false,
		},
		{
			name:    "valid multiple variables",
			tmpl:    "Hello ${firstName} ${lastName}",
			wantErr: false,
		},
		{
			name:    "valid function call",
			tmpl:    "Hello ${upper(name)}",
			wantErr: false,
		},
		{
			name:    "valid nested access",
			tmpl:    "Hello ${user.name}",
			wantErr: false,
		},
		{
			name:    "unclosed brace",
			tmpl:    "Hello ${name",
			wantErr: true,
		},
		{
			name:    "empty variable reference",
			tmpl:    "Hello ${}",
			wantErr: true,
		},
		{
			name:    "no template",
			tmpl:    "Hello world",
			wantErr: false,
		},
		{
			name:    "escaped template",
			tmpl:    "Hello \\${name}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateSyntax(tt.tmpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTemplateSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractTemplateVariables(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		expected []string
	}{
		{
			name:     "simple variable",
			tmpl:     "Hello ${name}",
			expected: []string{"name"},
		},
		{
			name:     "multiple variables",
			tmpl:     "Hello ${firstName} ${lastName}",
			expected: []string{"firstName", "lastName"},
		},
		{
			name:     "nested access",
			tmpl:     "Hello ${user.name}",
			expected: []string{"user"},
		},
		{
			name:     "function with variable",
			tmpl:     "Hello ${upper(name)}",
			expected: []string{"name"},
		},
		{
			name:     "no variables",
			tmpl:     "Hello world",
			expected: []string{},
		},
		{
			name:     "duplicate variables",
			tmpl:     "Hello ${name}, goodbye ${name}",
			expected: []string{"name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTemplateVariables(tt.tmpl)

			// Convert to map for easier comparison
			resultMap := make(map[string]bool)
			for _, v := range result {
				resultMap[v] = true
			}

			expectedMap := make(map[string]bool)
			for _, v := range tt.expected {
				expectedMap[v] = true
			}

			if len(resultMap) != len(expectedMap) {
				t.Errorf("extractTemplateVariables() got %v, want %v", result, tt.expected)
				return
			}

			for v := range expectedMap {
				if !resultMap[v] {
					t.Errorf("extractTemplateVariables() missing variable: %s", v)
				}
			}
		})
	}
}

func TestWorkflowValidateConditionExpression(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *Workflow
		node      *ConditionNode
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid condition with defined variable",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "count", Type: "number"},
				},
			},
			node: &ConditionNode{
				ID:        "cond1",
				Condition: "count > 10",
			},
			wantErr: false,
		},
		{
			name: "undefined variable in condition",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "count", Type: "number"},
				},
			},
			node: &ConditionNode{
				ID:        "cond1",
				Condition: "price > 100",
			},
			wantErr:   true,
			errSubstr: "undefined variable",
		},
		{
			name: "empty condition",
			workflow: &Workflow{
				Variables: []*Variable{},
			},
			node: &ConditionNode{
				ID:        "cond1",
				Condition: "",
			},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name: "unsafe operation",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "file", Type: "string"},
				},
			},
			node: &ConditionNode{
				ID:        "cond1",
				Condition: "os.ReadFile(file)",
			},
			wantErr:   true,
			errSubstr: "unsafe operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.validateConditionExpression(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConditionExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errSubstr != "" {
				if !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("validateConditionExpression() error = %v, should contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

func TestWorkflowValidateTransformConfig(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *Workflow
		node      *TransformNode
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid JSONPath transform",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "data", Type: "object"},
					{Name: "result", Type: "any"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "data",
				Expression:     "$.users[0].name",
				OutputVariable: "result",
			},
			wantErr: false,
		},
		{
			name: "valid template transform",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "name", Type: "string"},
					{Name: "greeting", Type: "string"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "name",
				Expression:     "Hello ${name}",
				OutputVariable: "greeting",
			},
			wantErr: false,
		},
		{
			name: "undefined input variable",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "result", Type: "string"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "data",
				Expression:     "$.name",
				OutputVariable: "result",
			},
			wantErr:   true,
			errSubstr: "undefined input variable",
		},
		{
			name: "undefined template variable",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "data", Type: "object"},
					{Name: "result", Type: "string"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "data",
				Expression:     "Hello ${name}",
				OutputVariable: "result",
			},
			wantErr:   true,
			errSubstr: "undefined variable in template",
		},
		{
			name: "invalid JSONPath syntax",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "data", Type: "object"},
					{Name: "result", Type: "any"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "data",
				Expression:     "$.users[0.name",
				OutputVariable: "result",
			},
			wantErr:   true,
			errSubstr: "invalid JSONPath",
		},
		{
			name: "empty expression",
			workflow: &Workflow{
				Variables: []*Variable{
					{Name: "data", Type: "object"},
				},
			},
			node: &TransformNode{
				ID:             "trans1",
				InputVariable:  "data",
				Expression:     "",
				OutputVariable: "result",
			},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.validateTransformConfig(tt.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTransformConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errSubstr != "" {
				if !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("validateTransformConfig() error = %v, should contain %q", err, tt.errSubstr)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
