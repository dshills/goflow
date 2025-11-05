package transform_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dshills/goflow/pkg/transform"
)

// Note: Use ctx for context.Context parameters, context for map[string]interface{}
// to avoid shadowing the context package

// ============================================================================
// Test Suite 1: Boolean Literals and Direct References
// ============================================================================

// TestBooleanLiterals tests that boolean literal values work correctly
func TestBooleanLiterals(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "literal true",
			expression: "true",
			context:    map[string]interface{}{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "literal false",
			expression: "false",
			context:    map[string]interface{}{},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "boolean variable reference",
			expression: "enabled",
			context:    map[string]interface{}{"enabled": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "boolean variable false",
			expression: "disabled",
			context:    map[string]interface{}{"disabled": false},
			want:       false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 2: Comparison Operators Return Booleans
// ============================================================================

// TestComparisonReturnsBool tests that comparison operators return boolean results
func TestComparisonReturnsBool(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "greater than returns boolean true",
			expression: "a > b",
			context:    map[string]interface{}{"a": 10, "b": 5},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "greater than returns boolean false",
			expression: "a > b",
			context:    map[string]interface{}{"a": 3, "b": 5},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "less than returns boolean",
			expression: "a < b",
			context:    map[string]interface{}{"a": 3, "b": 5},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "equals returns boolean",
			expression: "name == \"test\"",
			context:    map[string]interface{}{"name": "test"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "not equals returns boolean",
			expression: "name != \"test\"",
			context:    map[string]interface{}{"name": "other"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "greater than or equal returns boolean",
			expression: "a >= b",
			context:    map[string]interface{}{"a": 10, "b": 10},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "less than or equal returns boolean",
			expression: "a <= b",
			context:    map[string]interface{}{"a": 5, "b": 10},
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 3: Logical Operators
// ============================================================================

// TestLogicalAND tests the && (AND) operator
func TestLogicalAND(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "AND with both true",
			expression: "a && b",
			context:    map[string]interface{}{"a": true, "b": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "AND with first false",
			expression: "a && b",
			context:    map[string]interface{}{"a": false, "b": true},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "AND with second false",
			expression: "a && b",
			context:    map[string]interface{}{"a": true, "b": false},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "AND with both false",
			expression: "a && b",
			context:    map[string]interface{}{"a": false, "b": false},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "AND with comparisons",
			expression: "x > 5 && y < 10",
			context:    map[string]interface{}{"x": 10, "y": 5},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "AND multiple conditions",
			expression: "a && b && c",
			context:    map[string]interface{}{"a": true, "b": true, "c": true},
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestLogicalOR tests the || (OR) operator
func TestLogicalOR(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "OR with both true",
			expression: "a || b",
			context:    map[string]interface{}{"a": true, "b": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "OR with first true",
			expression: "a || b",
			context:    map[string]interface{}{"a": true, "b": false},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "OR with second true",
			expression: "a || b",
			context:    map[string]interface{}{"a": false, "b": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "OR with both false",
			expression: "a || b",
			context:    map[string]interface{}{"a": false, "b": false},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "OR with comparisons",
			expression: "x > 10 || y < 5",
			context:    map[string]interface{}{"x": 5, "y": 3},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "OR multiple conditions",
			expression: "a || b || c",
			context:    map[string]interface{}{"a": false, "b": false, "c": true},
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestLogicalNOT tests the ! (NOT) operator
func TestLogicalNOT(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "NOT true",
			expression: "!a",
			context:    map[string]interface{}{"a": true},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "NOT false",
			expression: "!a",
			context:    map[string]interface{}{"a": false},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "NOT with comparison",
			expression: "!(x > 5)",
			context:    map[string]interface{}{"x": 3},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "NOT literal true",
			expression: "!true",
			context:    map[string]interface{}{},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "NOT literal false",
			expression: "!false",
			context:    map[string]interface{}{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "double negation",
			expression: "!!a",
			context:    map[string]interface{}{"a": true},
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 4: Operator Precedence
// ============================================================================

// TestOperatorPrecedence tests that operators are evaluated in correct order
func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "AND has higher precedence than OR",
			expression: "a || b && c",
			context:    map[string]interface{}{"a": false, "b": false, "c": true},
			want:       false, // (false) || (false && true) = false || false = false
			wantErr:    false,
		},
		{
			name:       "AND has higher precedence than OR - case 2",
			expression: "a || b && c",
			context:    map[string]interface{}{"a": true, "b": false, "c": false},
			want:       true, // (true) || (false && false) = true || false = true
			wantErr:    false,
		},
		{
			name:       "parentheses override precedence",
			expression: "(a || b) && c",
			context:    map[string]interface{}{"a": false, "b": true, "c": false},
			want:       false, // (false || true) && false = true && false = false
			wantErr:    false,
		},
		{
			name:       "parentheses override precedence - case 2",
			expression: "(a || b) && c",
			context:    map[string]interface{}{"a": false, "b": true, "c": true},
			want:       true, // (false || true) && true = true && true = true
			wantErr:    false,
		},
		{
			name:       "NOT has higher precedence than AND",
			expression: "!a && b",
			context:    map[string]interface{}{"a": true, "b": true},
			want:       false, // (!true) && true = false && true = false
			wantErr:    false,
		},
		{
			name:       "NOT has higher precedence than AND - case 2",
			expression: "!a && b",
			context:    map[string]interface{}{"a": false, "b": true},
			want:       true, // (!false) && true = true && true = true
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 5: Parentheses Control
// ============================================================================

// TestParenthesesControl tests explicit parentheses for grouping
func TestParenthesesControl(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "simple parentheses",
			expression: "(a && b)",
			context:    map[string]interface{}{"a": true, "b": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "nested parentheses",
			expression: "((a && b) || c)",
			context:    map[string]interface{}{"a": false, "b": true, "c": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "deep nesting",
			expression: "(((a || b) && c) || d)",
			context:    map[string]interface{}{"a": true, "b": false, "c": false, "d": false},
			want:       false, // (((true || false) && false) || false) = ((true && false) || false) = (false || false) = false
			wantErr:    false,
		},
		{
			name:       "parentheses with comparisons",
			expression: "(x > 5) && (y < 10)",
			context:    map[string]interface{}{"x": 10, "y": 5},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "parentheses override NOT",
			expression: "!(a && b)",
			context:    map[string]interface{}{"a": true, "b": true},
			want:       false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 6: Helper Functions (not, and, or)
// ============================================================================

// TestBooleanHelperFunctions tests custom safe helper functions
// Note: expr-lang reserves 'and' and 'or' as operators, so only 'not()' is available as a function
func TestBooleanHelperFunctions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "not() function with true",
			expression: "not(true)",
			context:    map[string]interface{}{},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "not() function with false",
			expression: "not(false)",
			context:    map[string]interface{}{},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "not() function with variable",
			expression: "not(enabled)",
			context:    map[string]interface{}{"enabled": false},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "not() function with comparison",
			expression: "not(x > 5)",
			context:    map[string]interface{}{"x": 3},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "not() with less than one argument",
			expression: "not()",
			context:    map[string]interface{}{},
			want:       false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			result, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvaluateBool() = %v, want %v", result, tt.want)
			}
		})
	}
}

// ============================================================================
// Test Suite 7: Type Errors
// ============================================================================

// TestEvaluateBoolTypeErrors tests EvaluateBool type checking
func TestEvaluateBoolTypeErrors(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		wantErr    bool
		errType    error
	}{
		{
			name:       "numeric result causes type error",
			expression: "a + b",
			context:    map[string]interface{}{"a": 5, "b": 10},
			wantErr:    true,
			errType:    transform.ErrTypeMismatch,
		},
		{
			name:       "string result causes type error",
			expression: `firstName + " " + lastName`,
			context:    map[string]interface{}{"firstName": "John", "lastName": "Doe"},
			wantErr:    true,
			errType:    transform.ErrTypeMismatch,
		},
		{
			name:       "undefined variable causes error",
			expression: "undefined_var",
			context:    map[string]interface{}{},
			wantErr:    true,
			errType:    transform.ErrUndefinedVariable,
		},
		{
			name:       "invalid syntax causes error",
			expression: "a > > 5",
			context:    map[string]interface{}{"a": 10},
			wantErr:    true,
			errType:    transform.ErrInvalidExpression,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			_, err := evaluator.EvaluateBool(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("EvaluateBool() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkBooleanEvaluation benchmarks boolean expression evaluation
func BenchmarkBooleanEvaluation(b *testing.B) {
	evaluator := transform.NewExpressionEvaluator()
	expression := "a && b || c"
	ctx := context.Background()
	evalCtx := map[string]interface{}{
		"a": true,
		"b": false,
		"c": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.EvaluateBool(ctx, expression, evalCtx)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

// BenchmarkSimpleComparison benchmarks simple comparison evaluation
func BenchmarkSimpleComparison(b *testing.B) {
	evaluator := transform.NewExpressionEvaluator()
	expression := "x > 10"
	ctx := context.Background()
	evalCtx := map[string]interface{}{"x": 15}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.EvaluateBool(ctx, expression, evalCtx)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

// BenchmarkComplexBoolean benchmarks complex nested boolean evaluation
func BenchmarkComplexBoolean(b *testing.B) {
	evaluator := transform.NewExpressionEvaluator()
	expression := "(a > 5 && b < 10) || (c == 20 && !d)"
	ctx := context.Background()
	evalCtx := map[string]interface{}{
		"a": 7,
		"b": 8,
		"c": 20,
		"d": false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.EvaluateBool(ctx, expression, evalCtx)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}
