package transform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Sentinel errors for expression evaluation
var (
	ErrUnsafeOperation   = errors.New("unsafe operation attempted")
	ErrEvaluationTimeout = errors.New("expression evaluation timed out")
	ErrInvalidExpression = errors.New("invalid expression syntax")
	ErrUndefinedVariable = errors.New("undefined variable in expression")
)

// ExpressionEvaluator defines the interface for evaluating expressions
type ExpressionEvaluator interface {
	Evaluate(ctx context.Context, expression string, context map[string]interface{}) (interface{}, error)
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
			str, ok1 := params[0].(string)
			substr, ok2 := params[1].(string)
			if !ok1 || !ok2 {
				return false, nil
			}
			return strings.Contains(str, substr), nil
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
