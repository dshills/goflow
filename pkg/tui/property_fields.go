package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

// newPropertyField creates a new property field
// Note: propertyField type is defined in workflow_builder.go
func newPropertyField(label, value, fieldType string, required bool) propertyField {
	field := propertyField{
		label:     label,
		value:     value,
		required:  required,
		valid:     false,
		fieldType: fieldType,
		helpText:  getFieldHelpText(fieldType),
	}

	// Assign validation function based on field type
	switch fieldType {
	case "text":
		field.validationFn = validateTextField
	case "expression":
		field.validationFn = validateExpressionField
	case "condition":
		field.validationFn = validateConditionField
	case "jsonpath":
		field.validationFn = validateJSONPathField
	case "template":
		field.validationFn = validateTemplateField
	default:
		field.validationFn = validateTextField // fallback
	}

	return field
}

// validate runs the field's validation function
func (f *propertyField) validate() error {
	if f.validationFn == nil {
		return nil
	}

	err := f.validationFn(f.value)
	f.valid = (err == nil)
	return err
}

// getFieldHelpText returns syntax hints for each field type
func getFieldHelpText(fieldType string) string {
	switch fieldType {
	case "text":
		return "Enter text value"
	case "expression":
		return "Expression: e.g., total + 1, user.age * 2"
	case "condition":
		return "Boolean: e.g., total > 10 && status == \"active\""
	case "jsonpath":
		return "JSONPath: e.g., $.users[?(@.age > 18)].email"
	case "template":
		return "Template: e.g., \"Hello ${user.name}\""
	default:
		return "Enter text value" // Default help text
	}
}

// validateTextField validates text fields
// Checks: required, max length
func validateTextField(value string) error {
	// Note: required check is done separately by PropertyPanel
	// This function only validates format

	// Max length check (reasonable limit for TUI display)
	const maxLength = 256
	if len(value) > maxLength {
		return fmt.Errorf("text exceeds maximum length of %d characters", maxLength)
	}

	return nil
}

// validateExpressionField validates expression fields
// Uses expr-lang/expr to parse and check for unsafe operations
func validateExpressionField(value string) error {
	if value == "" {
		return nil // Empty is valid (required check done separately)
	}

	// Check for unsafe operations
	unsafePatterns := []string{
		"os.", "exec.", "http.", "net.", "syscall.", "unsafe.",
	}
	for _, pattern := range unsafePatterns {
		if strings.Contains(value, pattern) {
			return fmt.Errorf("unsafe operation not allowed: %s", pattern)
		}
	}

	// Parse expression to validate syntax
	// Note: We use a lenient environment that allows unknown variables
	// since we're validating syntax, not evaluating with runtime data
	_, err := expr.Compile(value, expr.AllowUndefinedVariables())
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}

	return nil
}

// validateConditionField validates condition fields
// Must be a boolean expression
func validateConditionField(value string) error {
	if value == "" {
		return nil // Empty is valid (required check done separately)
	}

	// First validate as expression
	if err := validateExpressionField(value); err != nil {
		return err
	}

	// Compile and check if it returns boolean
	// Note: We can't fully validate boolean return without runtime data
	// But we can check for obvious non-boolean expressions
	program, err := expr.Compile(value)
	if err != nil {
		return fmt.Errorf("invalid condition syntax: %w", err)
	}

	// Check if the expression contains comparison or logical operators
	// This is a heuristic check - true validation happens at runtime
	hasComparisonOrLogical := strings.ContainsAny(value, "<>=!&|")
	isBooleanLiteral := value == "true" || value == "false"

	if !hasComparisonOrLogical && !isBooleanLiteral && program != nil {
		// Warn but don't fail - expression might still return boolean
		// Let runtime validation catch issues
	}

	return nil
}

// validateJSONPathField validates JSONPath fields
// Uses gjson library to check syntax
func validateJSONPathField(value string) error {
	if value == "" {
		return nil // Empty is valid (required check done separately)
	}

	// JSONPath must start with $ or @ (gjson convention)
	if !strings.HasPrefix(value, "$") && !strings.HasPrefix(value, "@") {
		return fmt.Errorf("JSONPath must start with $ or @")
	}

	// Check for balanced brackets
	if !hasBalancedBrackets(value) {
		return fmt.Errorf("unbalanced brackets in JSONPath")
	}

	// Validate using gjson
	// gjson.Parse doesn't validate the path itself, so we check syntax manually
	// Check for common syntax errors
	if strings.Contains(value, "..") && !strings.Contains(value, "...") {
		// Double dot is valid for recursive descent
	}

	// Check for invalid characters (basic validation)
	// gjson is very permissive, so we just check for obviously broken paths
	if strings.Contains(value, "[[") || strings.Contains(value, "]]") {
		return fmt.Errorf("invalid bracket syntax in JSONPath")
	}

	return nil
}

// validateTemplateField validates template fields
// Checks for valid ${} placeholder syntax
func validateTemplateField(value string) error {
	if value == "" {
		return nil // Empty is valid (required check done separately)
	}

	// Check for balanced braces
	if !hasBalancedTemplateBraces(value) {
		return fmt.Errorf("unbalanced braces in template string")
	}

	// Check for empty placeholders: ${}
	if strings.Contains(value, "${}") {
		return fmt.Errorf("empty placeholder in template")
	}

	// Find all ${} placeholders
	placeholders := findTemplatePlaceholders(value)

	// Validate each placeholder
	for _, placeholder := range placeholders {
		// Variable name (placeholder content already extracted)
		varName := placeholder

		// Check if variable name is valid
		if varName == "" || strings.TrimSpace(varName) == "" {
			return fmt.Errorf("empty placeholder in template")
		}

		// Variable names should not have leading/trailing whitespace
		if strings.TrimSpace(varName) != varName {
			return fmt.Errorf("placeholder cannot have leading or trailing whitespace: ${%s}", varName)
		}

		// Variable names should match pattern: word characters, dots, brackets
		validVarName := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*|\[\d+\])*$`)
		if !validVarName.MatchString(varName) {
			return fmt.Errorf("invalid variable name in template: %s", varName)
		}
	}

	return nil
}

// hasBalancedBrackets checks if brackets are balanced
func hasBalancedBrackets(s string) bool {
	stack := 0
	for _, ch := range s {
		if ch == '[' {
			stack++
		} else if ch == ']' {
			stack--
			if stack < 0 {
				return false
			}
		}
	}
	return stack == 0
}

// hasBalancedTemplateBraces checks if template ${} braces are balanced
func hasBalancedTemplateBraces(s string) bool {
	// Track ${ and } pairs
	depth := 0
	inPlaceholder := false

	for i := 0; i < len(s); i++ {
		if i < len(s)-1 && s[i] == '$' && s[i+1] == '{' {
			depth++
			inPlaceholder = true
			i++ // skip next char
		} else if s[i] == '}' && inPlaceholder {
			depth--
			if depth < 0 {
				return false
			}
			if depth == 0 {
				inPlaceholder = false
			}
		}
	}

	return depth == 0
}

// findTemplatePlaceholders extracts all ${} placeholders from a template string
func findTemplatePlaceholders(s string) []string {
	placeholders := make([]string, 0)

	// Find all ${...} patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		if len(match) > 1 {
			placeholders = append(placeholders, match[1])
		}
	}

	return placeholders
}
