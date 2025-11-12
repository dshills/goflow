package security_test

import (
	"context"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/transform"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestExpressionInjection tests for various expression injection attacks
func TestExpressionInjection(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		shouldFail bool
		reason     string
	}{
		// Code injection attempts
		{
			name:       "os.system injection",
			expression: "os.system('rm -rf /')",
			shouldFail: true,
			reason:     "should detect os.system call",
		},
		{
			name:       "exec injection",
			expression: "exec('malicious code')",
			shouldFail: true,
			reason:     "should detect exec call",
		},
		{
			name:       "eval injection",
			expression: "eval('dangerous code')",
			shouldFail: true,
			reason:     "should detect eval call",
		},
		{
			name:       "__import__ injection",
			expression: "__import__('os').system('ls')",
			shouldFail: true,
			reason:     "should detect Python import injection",
		},
		{
			name:       "subprocess injection",
			expression: "subprocess.call(['rm', '-rf', '/'])",
			shouldFail: true,
			reason:     "should detect subprocess usage",
		},

		// Command injection attempts
		{
			name:       "system command injection",
			expression: "system('cat /etc/passwd')",
			shouldFail: true,
			reason:     "should detect system command",
		},
		{
			name:       "popen injection",
			expression: "popen('whoami')",
			shouldFail: true,
			reason:     "should detect popen call",
		},

		// Script injection attempts
		{
			name:       "javascript protocol injection",
			expression: "javascript:alert('XSS')",
			shouldFail: true,
			reason:     "should detect javascript protocol",
		},
		{
			name:       "vbscript injection",
			expression: "vbscript:msgbox('test')",
			shouldFail: true,
			reason:     "should detect vbscript protocol",
		},
		{
			name:       "data URL injection",
			expression: "data:text/html,<script>alert('XSS')</script>",
			shouldFail: true,
			reason:     "should detect data URL",
		},

		// Event handler injection
		{
			name:       "onload handler injection",
			expression: "<img onload=alert('XSS')>",
			shouldFail: true,
			reason:     "should detect onload handler",
		},
		{
			name:       "onerror handler injection",
			expression: "<img onerror=alert('XSS')>",
			shouldFail: true,
			reason:     "should detect onerror handler",
		},
		{
			name:       "onclick handler injection",
			expression: "<div onclick=malicious()>",
			shouldFail: true,
			reason:     "should detect onclick handler",
		},

		// Environment variable manipulation
		{
			name:       "LD_PRELOAD injection",
			expression: "LD_PRELOAD=/tmp/malicious.so",
			shouldFail: true,
			reason:     "should detect LD_PRELOAD manipulation",
		},
		{
			name:       "LD_LIBRARY_PATH injection",
			expression: "LD_LIBRARY_PATH=/tmp/malicious",
			shouldFail: true,
			reason:     "should detect LD_LIBRARY_PATH manipulation",
		},

		// File system access attempts
		{
			name:       "ReadFile attempt",
			expression: "ReadFile('/etc/passwd')",
			shouldFail: true,
			reason:     "should detect file read attempt",
		},
		{
			name:       "WriteFile attempt",
			expression: "WriteFile('/tmp/bad', 'data')",
			shouldFail: true,
			reason:     "should detect file write attempt",
		},

		// Network access attempts
		{
			name:       "http package access",
			expression: "http.Get('http://evil.com')",
			shouldFail: true,
			reason:     "should detect http package usage",
		},
		{
			name:       "net package access",
			expression: "net.Dial('tcp', 'evil.com:80')",
			shouldFail: true,
			reason:     "should detect net package usage",
		},

		// Unsafe package access
		{
			name:       "unsafe package access",
			expression: "unsafe.Pointer(nil)",
			shouldFail: true,
			reason:     "should detect unsafe package usage",
		},
		{
			name:       "syscall package access",
			expression: "syscall.Exec('/bin/sh', []string{}, []string{})",
			shouldFail: true,
			reason:     "should detect syscall package usage",
		},

		// Prototype pollution attempts
		{
			name:       "__proto__ manipulation",
			expression: "obj.__proto__.admin = true",
			shouldFail: true,
			reason:     "should detect prototype pollution",
		},

		// Valid expressions that should pass
		{
			name:       "simple variable reference",
			expression: "userName",
			shouldFail: false,
			reason:     "should allow valid variable reference",
		},
		{
			name:       "arithmetic expression",
			expression: "count + 1",
			shouldFail: false,
			reason:     "should allow arithmetic",
		},
		{
			name:       "boolean expression",
			expression: "age > 18 && verified == true",
			shouldFail: false,
			reason:     "should allow boolean logic",
		},
		{
			name:       "string concatenation",
			expression: "firstName + ' ' + lastName",
			shouldFail: false,
			reason:     "should allow string operations",
		},
		{
			name:       "array membership check",
			expression: "items[0] == 'apple'",
			shouldFail: false,
			reason:     "should allow array access and comparison",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := workflow.ValidateExpression(tt.expression)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail for %s: %s", tt.name, tt.reason)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass for %s: %s, but got error: %v", tt.name, tt.reason, err)
				}
			}
		})
	}
}

// TestJSONPathInjection tests for JSONPath injection attacks
func TestJSONPathInjection(t *testing.T) {
	tests := []struct {
		name       string
		jsonPath   string
		shouldFail bool
		reason     string
	}{
		// Script injection in JSONPath
		{
			name:       "script tag in path",
			jsonPath:   "$.users[?(@.name=='<script>alert(1)</script>')]",
			shouldFail: true,
			reason:     "should detect script tags",
		},
		{
			name:       "javascript protocol in filter",
			jsonPath:   "$.items[?(@.url=='javascript:alert(1)')]",
			shouldFail: true,
			reason:     "should detect javascript protocol",
		},

		// Null byte injection
		{
			name:       "null byte in path",
			jsonPath:   "$.users[0]\x00.admin",
			shouldFail: true,
			reason:     "should detect null bytes",
		},
		{
			name:       "URL encoded null byte",
			jsonPath:   "$.users%00.admin",
			shouldFail: true,
			reason:     "should detect encoded null bytes",
		},

		// Valid JSONPath expressions
		{
			name:       "simple field access",
			jsonPath:   "$.user.name",
			shouldFail: false,
			reason:     "should allow valid field access",
		},
		{
			name:       "array access",
			jsonPath:   "$.items[0]",
			shouldFail: false,
			reason:     "should allow array access",
		},
		{
			name:       "filter expression",
			jsonPath:   "$.users[?(@.age > 18)]",
			shouldFail: false,
			reason:     "should allow valid filters",
		},
		{
			name:       "wildcard access",
			jsonPath:   "$.items[*].price",
			shouldFail: false,
			reason:     "should allow wildcards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First check for security issues in the path string
			secErr := workflow.ValidateDescription(tt.jsonPath)

			// Then validate JSONPath syntax
			syntaxErr := workflow.ValidateJSONPathSyntax(tt.jsonPath)

			// Combine errors - fail if either check fails
			err := secErr
			if err == nil {
				err = syntaxErr
			}

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail for %s: %s", tt.name, tt.reason)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass for %s: %s, but got error: %v", tt.name, tt.reason, err)
				}
			}
		})
	}
}

// TestTemplateInjection tests for template injection attacks
func TestTemplateInjection(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		shouldFail bool
		reason     string
	}{
		// Code injection in templates
		{
			name:       "eval in template",
			template:   "Hello ${eval('malicious')}",
			shouldFail: true,
			reason:     "should detect eval in template",
		},
		{
			name:       "system call in template",
			template:   "User: ${system('whoami')}",
			shouldFail: true,
			reason:     "should detect system call",
		},
		{
			name:       "exec in template",
			template:   "Result: ${exec('rm -rf /')}",
			shouldFail: true,
			reason:     "should detect exec in template",
		},

		// Script injection in templates
		{
			name:       "script tag in template",
			template:   "Welcome <script>alert('XSS')</script>",
			shouldFail: true,
			reason:     "should detect script tags",
		},
		{
			name:       "javascript protocol in template",
			template:   "<a href='javascript:alert(1)'>Click</a>",
			shouldFail: true,
			reason:     "should detect javascript protocol",
		},
		{
			name:       "event handler in template",
			template:   "<img onerror=alert('XSS') src='x'>",
			shouldFail: true,
			reason:     "should detect event handlers",
		},

		// Valid templates
		{
			name:       "simple variable substitution",
			template:   "Hello ${user.name}",
			shouldFail: false,
			reason:     "should allow variable substitution",
		},
		{
			name:       "multiple variables",
			template:   "Order ${orderId} total: ${total}",
			shouldFail: false,
			reason:     "should allow multiple variables",
		},
		{
			name:       "nested field access",
			template:   "Email: ${user.contact.email}",
			shouldFail: false,
			reason:     "should allow nested access",
		},
		{
			name:       "plain text template",
			template:   "No variables here",
			shouldFail: false,
			reason:     "should allow plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate the template syntax
			err := workflow.ValidateTemplateSyntax(tt.template)
			if err != nil && !tt.shouldFail {
				t.Errorf("Template syntax validation failed for %s: %v", tt.name, err)
				return
			}

			// Then validate for security issues
			err = workflow.ValidateDescription(tt.template)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail for %s: %s", tt.name, tt.reason)
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "exceeds maximum length") {
					t.Errorf("Expected validation to pass for %s: %s, but got error: %v", tt.name, tt.reason, err)
				}
			}
		})
	}
}

// TestExpressionSandboxing tests that expressions are properly sandboxed
func TestExpressionSandboxing(t *testing.T) {
	evaluator := transform.NewExpressionEvaluator()
	ctx := context.Background()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		shouldFail bool
		reason     string
	}{
		{
			name:       "access to os package blocked",
			expression: "os.Getenv('SECRET')",
			data:       map[string]interface{}{},
			shouldFail: true,
			reason:     "should not allow access to os package",
		},
		{
			name:       "file system access blocked",
			expression: "ReadFile('/etc/passwd')",
			data:       map[string]interface{}{},
			shouldFail: true,
			reason:     "should not allow file system access",
		},
		{
			name:       "safe arithmetic allowed",
			expression: "a + b",
			data:       map[string]interface{}{"a": 10, "b": 20},
			shouldFail: false,
			reason:     "should allow safe arithmetic",
		},
		{
			name:       "safe comparisons allowed",
			expression: "age > 18",
			data:       map[string]interface{}{"age": 25},
			shouldFail: false,
			reason:     "should allow safe comparisons",
		},
		{
			name:       "safe string operations allowed",
			expression: "name + ' is ' + status",
			data:       map[string]interface{}{"name": "Alice", "status": "active"},
			shouldFail: false,
			reason:     "should allow safe string operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate the expression
			err := workflow.ValidateExpression(tt.expression)

			if tt.shouldFail {
				if err == nil {
					// If validation didn't catch it, try evaluation
					_, evalErr := evaluator.Evaluate(ctx, tt.expression, tt.data)
					if evalErr == nil {
						t.Errorf("Expected %s to fail: %s", tt.name, tt.reason)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Validation failed for %s: %s, error: %v", tt.name, tt.reason, err)
				} else {
					// Verify it can actually evaluate
					_, evalErr := evaluator.Evaluate(ctx, tt.expression, tt.data)
					if evalErr != nil {
						t.Errorf("Evaluation failed for %s: %s, error: %v", tt.name, tt.reason, evalErr)
					}
				}
			}
		})
	}
}

// TestNullByteInjection tests for null byte injection attacks
func TestNullByteInjection(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		validator func(string) error
		fieldName string
	}{
		{
			name:      "actual null byte in workflow name",
			input:     "workflow\x00admin",
			validator: workflow.ValidateWorkflowName,
			fieldName: "workflow name",
		},
		{
			name:      "URL encoded null pattern in node ID",
			input:     "node%00admin",
			validator: workflow.ValidateNodeID,
			fieldName: "node ID",
		},
		{
			name:      "escaped null pattern in variable name",
			input:     "var\\x00name",
			validator: workflow.ValidateVariableName,
			fieldName: "variable name",
		},
		{
			name:      "unicode null pattern in expression",
			input:     "value == '\\u0000'",
			validator: workflow.ValidateExpression,
			fieldName: "expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.input)
			if err == nil {
				t.Errorf("Expected validation to detect null byte pattern in %s", tt.fieldName)
			}
			if !strings.Contains(err.Error(), "null byte") &&
				!strings.Contains(err.Error(), "invalid") &&
				!strings.Contains(err.Error(), "alphanumeric") {
				t.Errorf("Expected error message to mention validation failure, got: %v", err)
			}
		})
	}
}

// TestPathTraversalInjection tests for path traversal attacks
func TestPathTraversalInjection(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "basic path traversal",
			path: "../../../etc/passwd",
		},
		{
			name: "URL encoded path traversal",
			path: "..%2f..%2f..%2fetc%2fpasswd",
		},
		{
			name: "double encoded path traversal",
			path: "..%252f..%252fetc%252fpasswd",
		},
		{
			name: "windows path traversal",
			path: "..\\..\\..\\windows\\system32",
		},
		{
			name: "mixed separators",
			path: "../..\\/etc/passwd",
		},
		{
			name: "current directory traversal",
			path: "./././etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := workflow.ValidateFilePath(tt.path)
			if err == nil {
				t.Errorf("Expected validation to detect path traversal in: %s", tt.path)
			}
		})
	}
}

// TestControlCharacterInjection tests for control character injection
func TestControlCharacterInjection(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "bell character in name",
			input: "workflow\aname",
		},
		{
			name:  "escape character in name",
			input: "workflow\x1bname",
		},
		{
			name:  "backspace character in name",
			input: "workflow\bname",
		},
		{
			name:  "null character in name",
			input: "workflow\x00name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := workflow.ValidateWorkflowName(tt.input)
			if err == nil {
				t.Errorf("Expected validation to detect control characters in: %s", tt.name)
			}
		})
	}
}

// TestReservedWordValidation tests that reserved words cannot be used as variable names
func TestReservedWordValidation(t *testing.T) {
	reservedWords := []string{
		"true", "false", "nil", "null",
		"and", "or", "not",
		"if", "else", "then",
		"for", "while", "break", "continue", "return",
		"function", "var", "let", "const",
	}

	for _, word := range reservedWords {
		t.Run(word, func(t *testing.T) {
			err := workflow.ValidateVariableName(word)
			if err == nil {
				t.Errorf("Expected validation to reject reserved word: %s", word)
			}
			if !strings.Contains(err.Error(), "reserved word") {
				t.Errorf("Expected error message to mention reserved word, got: %v", err)
			}
		})
	}
}

// TestLengthValidation tests that excessively long inputs are rejected
func TestLengthValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		validator func(string) error
		maxLength int
	}{
		{
			name:      "very long workflow name",
			input:     strings.Repeat("a", 300),
			validator: workflow.ValidateWorkflowName,
			maxLength: 256,
		},
		{
			name:      "very long node ID",
			input:     "n" + strings.Repeat("a", 200),
			validator: workflow.ValidateNodeID,
			maxLength: 128,
		},
		{
			name:      "very long expression",
			input:     "value == '" + strings.Repeat("a", 10000) + "'",
			validator: workflow.ValidateExpression,
			maxLength: 8192,
		},
		{
			name:      "very long file path",
			input:     "/" + strings.Repeat("a/", 2500), // 5001 characters total
			validator: workflow.ValidateFilePath,
			maxLength: 4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.input)
			if err == nil {
				t.Errorf("Expected validation to reject input exceeding %d characters", tt.maxLength)
				return
			}
			if !strings.Contains(err.Error(), "exceeds maximum length") &&
				!strings.Contains(err.Error(), "too long") &&
				!strings.Contains(err.Error(), "path traversal") {
				t.Errorf("Expected error message to mention length limit, got: %v", err)
			}
		})
	}
}
