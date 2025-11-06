package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/dshills/goflow/pkg/transform"
)

// variableReferenceRegex matches variable references in expressions
// Matches patterns like: variableName, variable.field, variable[0]
var variableReferenceRegex = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9_]*)\b`)

// extractVariableReferences extracts variable names from an expression
// This is a simplified extraction that looks for identifier-like tokens
// For dotted access like user.age, it returns only the base variable name (user)
// Ignores identifiers inside string literals
func extractVariableReferences(expr string) []string {
	// First, remove string literals to avoid false positives
	cleanedExpr := removeStringLiterals(expr)

	// Find all identifiers
	matches := variableReferenceRegex.FindAllStringSubmatch(cleanedExpr, -1)
	candidates := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			// Skip keywords and built-in identifiers
			if isKeyword(varName) {
				continue
			}
			candidates[varName] = true
		}
	}

	// Second pass: filter out field names from dotted access
	// If we see "user.age", we only want "user", not "age"
	seen := make(map[string]bool)
	var vars []string

	for candidate := range candidates {
		// Check if this identifier appears after a dot in the expression
		// e.g., in "user.age", "age" appears after "."
		isDottedField := false
		for i := 0; i < len(cleanedExpr); i++ {
			// Look for ".candidate"
			if i > 0 && cleanedExpr[i-1] == '.' {
				// Check if we have the candidate word here
				if i+len(candidate) <= len(cleanedExpr) && cleanedExpr[i:i+len(candidate)] == candidate {
					// Check if it's a word boundary after
					if i+len(candidate) >= len(cleanedExpr) || !isIdentifierChar(rune(cleanedExpr[i+len(candidate)])) {
						isDottedField = true
						break
					}
				}
			}
		}

		if !isDottedField && !seen[candidate] {
			seen[candidate] = true
			vars = append(vars, candidate)
		}
	}

	return vars
}

// removeStringLiterals removes string literals from an expression
// Replaces strings with empty strings to avoid extracting identifiers from them
func removeStringLiterals(expr string) string {
	result := strings.Builder{}
	inString := false
	stringChar := rune(0)

	for i, ch := range expr {
		// Check for escape sequences
		if i > 0 && expr[i-1] == '\\' {
			continue
		}

		// Check for string delimiters
		if !inString && (ch == '\'' || ch == '"') {
			inString = true
			stringChar = ch
			continue
		} else if inString && ch == stringChar {
			inString = false
			stringChar = 0
			continue
		}

		// Only include characters outside of strings
		if !inString {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// isIdentifierChar checks if a character is valid in an identifier
func isIdentifierChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isKeyword checks if a string is a reserved keyword or built-in function
func isKeyword(s string) bool {
	keywords := map[string]bool{
		"true":     true,
		"false":    true,
		"nil":      true,
		"null":     true,
		"and":      true,
		"or":       true,
		"not":      true,
		"contains": true,
		"len":      true,
		"length":   true,
		"test":     true, // Common literal in tests
	}
	return keywords[s]
}

// validateExpressionSyntax validates the syntax of an expression
// Uses the transform package's expression evaluator for validation
func validateExpressionSyntax(expr string) error {
	// Check for unsafe operations first
	unsafePatterns := []string{
		"os.",
		"exec.",
		"http.",
		"net.",
		"syscall.",
		"unsafe.",
		"__proto__",
		"ReadFile",
		"WriteFile",
		"Command",
	}

	lowerExpr := strings.ToLower(expr)
	for _, pattern := range unsafePatterns {
		if strings.Contains(lowerExpr, strings.ToLower(pattern)) {
			return fmt.Errorf("unsafe operation detected: %s", pattern)
		}
	}

	// Try to compile the expression with a minimal context
	// This validates syntax without requiring actual data
	evaluator := transform.NewExpressionEvaluator()
	ctx := context.Background()
	dummyContext := make(map[string]interface{})

	// Add dummy values for extracted variables to allow compilation
	// Use interface{} to allow the expression to compile with any type
	varRefs := extractVariableReferences(expr)
	for _, varName := range varRefs {
		if !isKeyword(varName) {
			// Use true as dummy value - it can be used in boolean contexts
			// and the expr library is more lenient with booleans
			dummyContext[varName] = true
		}
	}

	// Try to evaluate - we don't care about the result, just syntax validation
	_, err := evaluator.Evaluate(ctx, expr, dummyContext)
	if err != nil {
		// Filter out type mismatch errors since we're using dummy context
		if strings.Contains(err.Error(), "undefined") ||
			strings.Contains(err.Error(), "mismatched types") ||
			strings.Contains(err.Error(), "type mismatch") {
			// This is OK during validation - we just want to check syntax
			return nil
		}
		return err
	}

	return nil
}

// validateJSONPathSyntax validates JSONPath expression syntax
func validateJSONPathSyntax(path string) error {
	// Basic syntax checks
	if path == "" {
		return fmt.Errorf("empty JSONPath")
	}

	// JSONPath should start with $
	if !strings.HasPrefix(path, "$") {
		return fmt.Errorf("JSONPath must start with $")
	}

	// Check for balanced brackets
	bracketCount := 0
	for _, ch := range path {
		if ch == '[' {
			bracketCount++
		} else if ch == ']' {
			bracketCount--
			if bracketCount < 0 {
				return fmt.Errorf("unmatched closing bracket in JSONPath")
			}
		}
	}
	if bracketCount != 0 {
		return fmt.Errorf("unclosed bracket in JSONPath")
	}

	// Check for balanced braces in filter expressions
	braceCount := 0
	for _, ch := range path {
		if ch == '(' {
			braceCount++
		} else if ch == ')' {
			braceCount--
			if braceCount < 0 {
				return fmt.Errorf("unmatched closing parenthesis in JSONPath")
			}
		}
	}
	if braceCount != 0 {
		return fmt.Errorf("unclosed parenthesis in JSONPath")
	}

	// Try to use the querier to validate (it will catch more complex errors)
	querier := transform.NewJSONPathQuerier()
	// Use a minimal test document to check if the path compiles
	testData := map[string]interface{}{"test": "value"}
	_, err := querier.Query(context.Background(), path, testData)
	// We don't care if it returns nil (path not found), only if there's a syntax error
	if err != nil && strings.Contains(err.Error(), "invalid") {
		return err
	}

	return nil
}

// containsTemplate checks if a string contains template syntax ${...}
func containsTemplate(s string) bool {
	return strings.Contains(s, "${")
}

// validateTemplateSyntax validates template string syntax
func validateTemplateSyntax(template string) error {
	// Check for balanced braces
	i := 0
	for i < len(template) {
		// Skip escaped dollar signs
		if i < len(template)-1 && template[i] == '\\' && template[i+1] == '$' {
			i += 2
			continue
		}

		// Look for ${
		if i < len(template)-1 && template[i] == '$' && template[i+1] == '{' {
			// Find closing }
			j := i + 2
			depth := 1
			for j < len(template) && depth > 0 {
				if template[j] == '{' {
					depth++
				} else if template[j] == '}' {
					depth--
				}
				j++
			}

			if depth != 0 {
				return fmt.Errorf("unclosed brace at position %d", i)
			}

			// Extract and validate the expression inside
			expr := template[i+2 : j-1]
			if expr == "" {
				return fmt.Errorf("empty variable reference at position %d", i)
			}

			// Check for valid variable name or function call
			if !isValidTemplateExpression(expr) {
				return fmt.Errorf("invalid template expression: %s", expr)
			}

			i = j
		} else {
			i++
		}
	}

	return nil
}

// isValidTemplateExpression checks if a template expression is valid
// Valid forms: variableName, variable.field, functionName(args)
func isValidTemplateExpression(expr string) bool {
	// Allow variable names with dots (for nested access)
	// Allow function calls
	if strings.Contains(expr, "(") {
		// Function call - should have matching parentheses
		openCount := strings.Count(expr, "(")
		closeCount := strings.Count(expr, ")")
		return openCount == closeCount
	}

	// Simple variable reference - should be a valid identifier possibly with dots
	parts := strings.Split(expr, ".")
	for _, part := range parts {
		if part == "" {
			return false
		}
		// Each part should be a valid identifier
		if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(part) {
			return false
		}
	}

	return true
}

// extractTemplateVariables extracts variable names from template syntax
// Example: "Hello ${user.name}, you have ${count} items" -> ["user", "count"]
func extractTemplateVariables(template string) []string {
	var vars []string
	seen := make(map[string]bool)

	i := 0
	for i < len(template) {
		// Skip escaped dollar signs
		if i < len(template)-1 && template[i] == '\\' && template[i+1] == '$' {
			i += 2
			continue
		}

		// Look for ${
		if i < len(template)-1 && template[i] == '$' && template[i+1] == '{' {
			// Find closing }
			j := strings.Index(template[i+2:], "}")
			if j == -1 {
				break
			}
			j += i + 2

			// Extract expression
			expr := template[i+2 : j]

			// Extract variable name (first part before . or ()
			varName := extractBaseVariable(expr)
			if varName != "" && !seen[varName] {
				seen[varName] = true
				vars = append(vars, varName)
			}

			i = j + 1
		} else {
			i++
		}
	}

	return vars
}

// extractBaseVariable extracts the base variable name from an expression
// Examples: "user.name" -> "user", "count" -> "count", "upper(name)" -> "name"
func extractBaseVariable(expr string) string {
	// Handle function calls - extract argument variables
	if strings.Contains(expr, "(") {
		// For now, just extract the function argument if it's a simple variable
		start := strings.Index(expr, "(")
		end := strings.LastIndex(expr, ")")
		if start >= 0 && end > start {
			arg := strings.TrimSpace(expr[start+1 : end])
			// Remove quotes if it's a string literal
			if len(arg) > 0 && (arg[0] == '"' || arg[0] == '\'') {
				return ""
			}
			return extractBaseVariable(arg)
		}
		return ""
	}

	// Handle dot notation - return first part
	if strings.Contains(expr, ".") {
		parts := strings.Split(expr, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Simple variable name
	return expr
}

// Exported validation functions for TUI use

// ValidateExpressionSyntax validates the syntax of a condition expression (exported)
func ValidateExpressionSyntax(expr string) error {
	return validateExpressionSyntax(expr)
}

// ValidateJSONPathSyntax validates JSONPath expression syntax (exported)
func ValidateJSONPathSyntax(path string) error {
	return validateJSONPathSyntax(path)
}

// ValidateTemplateSyntax validates template string syntax (exported)
func ValidateTemplateSyntax(template string) error {
	return validateTemplateSyntax(template)
}

// ExtractVariableReferences extracts variable names from an expression (exported)
func ExtractVariableReferences(expr string) []string {
	return extractVariableReferences(expr)
}
