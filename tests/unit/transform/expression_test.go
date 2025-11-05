package transform_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/transform"
)

// TestConditionalExpressions tests ternary conditional evaluation: condition ? trueValue : falseValue
func TestConditionalExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "simple numeric comparison - greater than",
			expression: `count > 10 ? "many" : "few"`,
			context:    map[string]interface{}{"count": 15},
			want:       "many",
			wantErr:    false,
		},
		{
			name:       "simple numeric comparison - less than",
			expression: `count > 10 ? "many" : "few"`,
			context:    map[string]interface{}{"count": 5},
			want:       "few",
			wantErr:    false,
		},
		{
			name:       "equality comparison",
			expression: `status == "active" ? "ACTIVE" : "INACTIVE"`,
			context:    map[string]interface{}{"status": "active"},
			want:       "ACTIVE",
			wantErr:    false,
		},
		{
			name:       "inequality comparison",
			expression: `status != "active" ? "unavailable" : "available"`,
			context:    map[string]interface{}{"status": "pending"},
			want:       "unavailable",
			wantErr:    false,
		},
		{
			name:       "nested ternary",
			expression: `score >= 90 ? "A" : score >= 80 ? "B" : "C"`,
			context:    map[string]interface{}{"score": 85},
			want:       "B",
			wantErr:    false,
		},
		{
			name:       "boolean context variable",
			expression: `enabled ? "on" : "off"`,
			context:    map[string]interface{}{"enabled": true},
			want:       "on",
			wantErr:    false,
		},
		{
			name:       "numeric result in ternary",
			expression: `premium ? 100 : 10`,
			context:    map[string]interface{}{"premium": true},
			want:       100,
			wantErr:    false,
		},
		{
			name:       "missing variable in context",
			expression: `missing > 10 ? "yes" : "no"`,
			context:    map[string]interface{}{},
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "invalid syntax - missing colon",
			expression: `count > 10 ? "many"`,
			context:    map[string]interface{}{"count": 15},
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			got, err := evaluator.Evaluate(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBooleanExpressions tests logical operations: AND, OR, NOT
func TestBooleanExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "AND operation - both true",
			expression: `status == "active" && role == "admin"`,
			context:    map[string]interface{}{"status": "active", "role": "admin"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "AND operation - one false",
			expression: `status == "active" && role == "admin"`,
			context:    map[string]interface{}{"status": "active", "role": "user"},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "OR operation - one true",
			expression: `status == "active" || status == "pending"`,
			context:    map[string]interface{}{"status": "active"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "OR operation - both false",
			expression: `status == "active" || status == "pending"`,
			context:    map[string]interface{}{"status": "disabled"},
			want:       false,
			wantErr:    false,
		},
		{
			name:       "NOT operation",
			expression: `!(status == "disabled")`,
			context:    map[string]interface{}{"status": "active"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "complex boolean expression",
			expression: `(age >= 18 && country == "US") || verified`,
			context:    map[string]interface{}{"age": 20, "country": "US", "verified": false},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "complex boolean with numeric comparisons",
			expression: `count > 0 && count < 100 && active`,
			context:    map[string]interface{}{"count": 50, "active": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "string contains check (if supported)",
			expression: `email contains "@" && email contains "."`,
			context:    map[string]interface{}{"email": "user@example.com"},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "parentheses for precedence",
			expression: `(status == "active" || status == "pending") && role == "admin"`,
			context:    map[string]interface{}{"status": "pending", "role": "admin"},
			want:       true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			got, err := evaluator.Evaluate(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotBool, ok := got.(bool)
				if !ok {
					t.Errorf("Evaluate() returned non-boolean value: %v (type %T)", got, got)
					return
				}
				if gotBool != tt.want {
					t.Errorf("Evaluate() = %v, want %v", gotBool, tt.want)
				}
			}
		})
	}
}

// TestSandboxedEvaluation ensures expressions cannot access file system, network, or system resources
func TestSandboxedEvaluation(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		wantErr    bool
		errType    error
	}{
		{
			name:       "attempt to call os package function",
			expression: `os.ReadFile("/etc/passwd")`,
			context:    map[string]interface{}{},
			wantErr:    true,
			errType:    transform.ErrUnsafeOperation,
		},
		{
			name:       "attempt to call net package function",
			expression: `http.Get("https://evil.com")`,
			context:    map[string]interface{}{},
			wantErr:    true,
			errType:    transform.ErrUnsafeOperation,
		},
		{
			name:       "attempt to execute arbitrary code",
			expression: `exec.Command("rm", "-rf", "/")`,
			context:    map[string]interface{}{},
			wantErr:    true,
			errType:    transform.ErrUnsafeOperation,
		},
		{
			name:       "safe arithmetic operations",
			expression: `(a + b) * c`,
			context:    map[string]interface{}{"a": 10, "b": 20, "c": 2},
			wantErr:    false,
		},
		{
			name:       "safe string operations",
			expression: `firstName + " " + lastName`,
			context:    map[string]interface{}{"firstName": "John", "lastName": "Doe"},
			wantErr:    false,
		},
		{
			name:       "attempt to access private fields (should be blocked)",
			expression: `obj.__proto__`,
			context:    map[string]interface{}{"obj": map[string]interface{}{"value": 42}},
			wantErr:    true,
			errType:    transform.ErrUnsafeOperation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			_, err := evaluator.Evaluate(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Evaluate() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// TestTimeoutProtection ensures expressions cannot run indefinitely
func TestTimeoutProtection(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		timeout    time.Duration
		wantErr    bool
		errType    error
	}{
		{
			name:       "infinite loop protection",
			expression: `while(true) { }`, // Syntax may vary based on expr-lang capabilities
			context:    map[string]interface{}{},
			timeout:    100 * time.Millisecond,
			wantErr:    true,
			errType:    transform.ErrEvaluationTimeout,
		},
		{
			name:       "recursive expression (if supported)",
			expression: `factorial(1000000)`, // Would cause stack overflow or long computation
			context:    map[string]interface{}{},
			timeout:    100 * time.Millisecond,
			wantErr:    true,
			errType:    transform.ErrEvaluationTimeout,
		},
		{
			name:       "fast expression completes before timeout",
			expression: `count * 2`,
			context:    map[string]interface{}{"count": 42},
			timeout:    1 * time.Second,
			wantErr:    false,
		},
		{
			name:       "context cancellation propagates",
			expression: `count + 1`,
			context:    map[string]interface{}{"count": 10},
			timeout:    0, // Will use cancelled context instead
			wantErr:    true,
			errType:    context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()

			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			} else if tt.name == "context cancellation propagates" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			_, err := evaluator.Evaluate(ctx, tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Evaluate() error = %v, want error type %v", err, tt.errType)
				}
			}
		})
	}
}

// TestExpressionTypes validates that expressions return expected types
func TestExpressionTypes(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		wantType   string // "bool", "int", "float64", "string"
		wantErr    bool
	}{
		{
			name:       "boolean result",
			expression: `age >= 18`,
			context:    map[string]interface{}{"age": 21},
			wantType:   "bool",
			wantErr:    false,
		},
		{
			name:       "integer result",
			expression: `count + 5`,
			context:    map[string]interface{}{"count": 10},
			wantType:   "int",
			wantErr:    false,
		},
		{
			name:       "float result",
			expression: `price * 1.1`,
			context:    map[string]interface{}{"price": 100.0},
			wantType:   "float64",
			wantErr:    false,
		},
		{
			name:       "string result",
			expression: `firstName + " " + lastName`,
			context:    map[string]interface{}{"firstName": "Jane", "lastName": "Smith"},
			wantType:   "string",
			wantErr:    false,
		},
		{
			name:       "type coercion - string to number fails",
			expression: `"hello" + 42`,
			context:    map[string]interface{}{},
			wantType:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := transform.NewExpressionEvaluator()
			got, err := evaluator.Evaluate(context.Background(), tt.expression, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotType := ""
				switch got.(type) {
				case bool:
					gotType = "bool"
				case int, int64:
					gotType = "int"
				case float64:
					gotType = "float64"
				case string:
					gotType = "string"
				}

				if gotType != tt.wantType {
					t.Errorf("Evaluate() returned type %s, want %s", gotType, tt.wantType)
				}
			}
		})
	}
}
