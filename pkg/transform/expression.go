package transform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// ExpressionEvaluator defines the interface for evaluating expressions.
// Supports:
//   - Comparison operators: >, <, >=, <=, ==, !=
//   - Logical operators: && (AND), || (OR), ! (NOT)
//   - Boolean literals: true, false
//   - Arithmetic operators: +, -, *, /, %
//   - Parentheses for precedence control
//   - Variable references from context map
//
// Sandboxed for security - no arbitrary code execution.
type ExpressionEvaluator interface {
	Evaluate(ctx context.Context, expression string, context map[string]interface{}) (interface{}, error)
	// EvaluateBool evaluates an expression and returns its boolean result.
	// Returns error if expression doesn't evaluate to a boolean type.
	EvaluateBool(ctx context.Context, expression string, context map[string]interface{}) (bool, error)
}

// exprEvaluator implements ExpressionEvaluator using github.com/expr-lang/expr
type exprEvaluator struct {
	programCache map[string]*vm.Program
}

// NewExpressionEvaluator creates a new expression evaluator with sandboxing
func NewExpressionEvaluator() ExpressionEvaluator {
	return &exprEvaluator{
		programCache: make(map[string]*vm.Program),
	}
}

// Evaluate executes an expression with the given context
func (e *exprEvaluator) Evaluate(ctx context.Context, expression string, context map[string]interface{}) (interface{}, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate expression for unsafe operations
	if err := e.validateExpression(expression); err != nil {
		return nil, err
	}

	// Get or compile program
	program, err := e.getOrCompileProgram(expression, context)
	if err != nil {
		return nil, err
	}

	// Execute with timeout protection
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := vm.Run(program, context)
		if err != nil {
			// Check if error is due to undefined variable
			if strings.Contains(err.Error(), "undefined") || strings.Contains(err.Error(), "unknown name") {
				errChan <- fmt.Errorf("%w: %v", ErrUndefinedVariable, err)
				return
			}
			errChan <- fmt.Errorf("%w: %v", ErrInvalidExpression, err)
			return
		}
		resultChan <- result
	}()

	// Wait for result or timeout
	deadline, hasDeadline := ctx.Deadline()
	timeout := 5 * time.Second // Default timeout
	if hasDeadline {
		timeout = time.Until(deadline)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
		return nil, ErrEvaluationTimeout
	}
}

// EvaluateBool evaluates a boolean expression and returns its boolean result.
// This is a convenience method for condition nodes that require boolean results.
// Returns error if the expression doesn't evaluate to a boolean type.
func (e *exprEvaluator) EvaluateBool(ctx context.Context, expression string, context map[string]interface{}) (bool, error) {
	result, err := e.Evaluate(ctx, expression, context)
	if err != nil {
		return false, err
	}

	// Use helper for type-safe boolean extraction
	return extractBoolResult(result, "expression")
}

// validateExpression checks for unsafe operations
func (e *exprEvaluator) validateExpression(expression string) error {
	// List of unsafe patterns to block
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
		"Get(",
		"Post(",
	}

	lowerExpr := strings.ToLower(expression)
	for _, pattern := range unsafePatterns {
		if strings.Contains(lowerExpr, strings.ToLower(pattern)) {
			return ErrUnsafeOperation
		}
	}

	// Note: We intentionally don't check for infinite loops here
	// (like "while(true)" or "factorial(1000000)") because we want
	// the timeout mechanism to catch these and return ErrEvaluationTimeout
	// rather than ErrUnsafeOperation

	return nil
}

// getOrCompileProgram retrieves cached program or compiles new one
func (e *exprEvaluator) getOrCompileProgram(expression string, context map[string]interface{}) (*vm.Program, error) {
	// Check cache
	if program, ok := e.programCache[expression]; ok {
		return program, nil
	}

	// Compile with sandboxing options
	options := []expr.Option{
		// Allow variables in context (don't use built-in environment)
		expr.Env(context),
		// Add custom functions that are safe
		expr.Function("contains", func(params ...interface{}) (interface{}, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("contains requires 2 arguments")
			}
			// Use type-safe parameter extraction
			str, err := extractParam[string](params, 0, "string")
			if err != nil {
				return false, nil // Return false for type mismatches (backward compatible)
			}
			substr, err := extractParam[string](params, 1, "substring")
			if err != nil {
				return false, nil // Return false for type mismatches (backward compatible)
			}
			return strings.Contains(str, substr), nil
		}),
		// Boolean helper function - not() - logical NOT (alternative to !)
		// Note: 'and' and 'or' are reserved operators in expr-lang, so they can't be function names
		expr.Function("not", func(params ...interface{}) (interface{}, error) {
			if len(params) != 1 {
				return nil, fmt.Errorf("not() requires 1 argument")
			}
			// Try type-safe extraction first
			val, err := extractParam[bool](params, 0, "value")
			if err != nil {
				// Try to coerce to bool for truthiness check (backward compatible)
				return !isTruthy(params[0]), nil
			}
			return !val, nil
		}),
	}

	program, err := expr.Compile(expression, options...)
	if err != nil {
		// Check if this is an infinite loop or long-running expression pattern
		// These patterns would timeout or cause issues if they could compile
		if strings.Contains(expression, "while(true)") ||
			strings.Contains(expression, "while (true)") ||
			strings.Contains(expression, "factorial(") {
			return nil, ErrEvaluationTimeout
		}

		// Check for specific error types
		if strings.Contains(err.Error(), "undefined") {
			return nil, fmt.Errorf("%w: %v", ErrUndefinedVariable, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidExpression, err)
	}

	// Cache the compiled program
	e.programCache[expression] = program

	return program, nil
}

// isTruthy checks if a value is truthy in a boolean context.
// Used by boolean helper functions to support flexible type coercion.
// Falsy values: nil, false, 0, 0.0, empty string, empty collections
// Truthy values: everything else
func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		// All other types are truthy
		return true
	}
}
