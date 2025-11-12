package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

func TestRetryPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  workflow.RetryPolicy
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid policy",
			policy: workflow.RetryPolicy{
				MaxAttempts:       3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          30 * time.Second,
				BackoffMultiplier: 2.0,
			},
			wantErr: false,
		},
		{
			name: "negative max attempts",
			policy: workflow.RetryPolicy{
				MaxAttempts: -1,
			},
			wantErr: true,
			errMsg:  "max_attempts cannot be negative",
		},
		{
			name: "negative initial delay",
			policy: workflow.RetryPolicy{
				MaxAttempts:  3,
				InitialDelay: -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "initial_delay cannot be negative",
		},
		{
			name: "negative max delay",
			policy: workflow.RetryPolicy{
				MaxAttempts:  3,
				InitialDelay: 1 * time.Second,
				MaxDelay:     -1 * time.Second,
			},
			wantErr: true,
			errMsg:  "max_delay cannot be negative",
		},
		{
			name: "initial delay greater than max delay",
			policy: workflow.RetryPolicy{
				MaxAttempts:  3,
				InitialDelay: 30 * time.Second,
				MaxDelay:     10 * time.Second,
			},
			wantErr: true,
			errMsg:  "initial_delay cannot be greater than max_delay",
		},
		{
			name: "backoff multiplier less than 1",
			policy: workflow.RetryPolicy{
				MaxAttempts:       3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          30 * time.Second,
				BackoffMultiplier: 0.5,
			},
			wantErr: true,
			errMsg:  "backoff_multiplier must be >= 1.0",
		},
		{
			name: "zero max attempts (no retry)",
			policy: workflow.RetryPolicy{
				MaxAttempts:       0,
				BackoffMultiplier: 2.0, // Set valid multiplier even when disabled
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != "retry policy: "+tt.errMsg {
					t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestRetryPolicy_SetDefaults(t *testing.T) {
	tests := []struct {
		name   string
		policy workflow.RetryPolicy
		want   workflow.RetryPolicy
	}{
		{
			name: "sets all defaults",
			policy: workflow.RetryPolicy{
				MaxAttempts: 3,
			},
			want: workflow.RetryPolicy{
				MaxAttempts:       3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          30 * time.Second,
				BackoffMultiplier: 2.0,
			},
		},
		{
			name: "preserves existing values",
			policy: workflow.RetryPolicy{
				MaxAttempts:       5,
				InitialDelay:      2 * time.Second,
				MaxDelay:          60 * time.Second,
				BackoffMultiplier: 3.0,
			},
			want: workflow.RetryPolicy{
				MaxAttempts:       5,
				InitialDelay:      2 * time.Second,
				MaxDelay:          60 * time.Second,
				BackoffMultiplier: 3.0,
			},
		},
		{
			name: "no defaults when max attempts is 0",
			policy: workflow.RetryPolicy{
				MaxAttempts: 0,
			},
			want: workflow.RetryPolicy{
				MaxAttempts: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.policy.SetDefaults()
			if tt.policy.MaxAttempts != tt.want.MaxAttempts ||
				tt.policy.InitialDelay != tt.want.InitialDelay ||
				tt.policy.MaxDelay != tt.want.MaxDelay ||
				tt.policy.BackoffMultiplier != tt.want.BackoffMultiplier {
				t.Errorf("SetDefaults() = %+v, want %+v", tt.policy, tt.want)
			}
		})
	}
}

func TestRetryExecutor_Execute_NoRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("nil executor executes once", func(t *testing.T) {
		var executor *RetryExecutor
		callCount := 0
		fn := func() error {
			callCount++
			return nil
		}

		err := executor.Execute(ctx, fn)
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if callCount != 1 {
			t.Errorf("function called %d times, want 1", callCount)
		}
	})

	t.Run("disabled policy executes once", func(t *testing.T) {
		policy := &workflow.RetryPolicy{MaxAttempts: 0}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			return errors.New("test error")
		}

		err := executor.Execute(ctx, fn)
		if err == nil {
			t.Error("Execute() error = nil, want error")
		}
		if callCount != 1 {
			t.Errorf("function called %d times, want 1", callCount)
		}
	})
}

func TestRetryExecutor_Execute_Success(t *testing.T) {
	ctx := context.Background()

	t.Run("success on first attempt", func(t *testing.T) {
		policy := &workflow.RetryPolicy{
			MaxAttempts:       3,
			InitialDelay:      10 * time.Millisecond,
			MaxDelay:          100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			return nil
		}

		err := executor.Execute(ctx, fn)
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if callCount != 1 {
			t.Errorf("function called %d times, want 1", callCount)
		}
	})

	t.Run("success on second attempt", func(t *testing.T) {
		policy := &workflow.RetryPolicy{
			MaxAttempts:       3,
			InitialDelay:      10 * time.Millisecond,
			MaxDelay:          100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			if callCount == 1 {
				return errors.New("temporary error")
			}
			return nil
		}

		start := time.Now()
		err := executor.Execute(ctx, fn)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
		if callCount != 2 {
			t.Errorf("function called %d times, want 2", callCount)
		}
		// Should have at least one delay (with jitter, it could be slightly less than 10ms)
		// Use 7ms to account for jitter (25% of 10ms = 7.5ms minimum)
		if duration < 7*time.Millisecond {
			t.Errorf("execution took %v, expected at least 7ms for retry delay", duration)
		}
	})
}

func TestRetryExecutor_Execute_MaxAttemptsExhausted(t *testing.T) {
	ctx := context.Background()

	policy := &workflow.RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	executor := NewRetryExecutor(policy)

	callCount := 0
	expectedErr := errors.New("persistent error")
	fn := func() error {
		callCount++
		return expectedErr
	}

	err := executor.Execute(ctx, fn)

	// Should have tried initial + 3 retries = 4 total
	if callCount != 4 {
		t.Errorf("function called %d times, want 4", callCount)
	}

	// Should return RetryExhaustedError
	var retryErr *RetryExhaustedError
	if !errors.As(err, &retryErr) {
		t.Fatalf("Execute() error type = %T, want *RetryExhaustedError", err)
	}

	if retryErr.Attempts != 4 {
		t.Errorf("RetryExhaustedError.Attempts = %d, want 4", retryErr.Attempts)
	}
	if !errors.Is(retryErr.LastError, expectedErr) {
		t.Errorf("RetryExhaustedError.LastError = %v, want %v", retryErr.LastError, expectedErr)
	}
}

func TestRetryExecutor_Execute_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()

	policy := &workflow.RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
	}
	executor := NewRetryExecutor(policy)

	callCount := 0
	fn := func() error {
		callCount++
		return errors.New("test error")
	}

	start := time.Now()
	executor.Execute(ctx, fn)
	duration := time.Since(start)

	// Expected delays (with jitter, so use approximate values):
	// Attempt 0 fails -> delay ~100ms
	// Attempt 1 fails -> delay ~200ms
	// Attempt 2 fails -> delay ~400ms
	// Attempt 3 fails -> done
	// Total: ~700ms (allowing for jitter and execution time, check for at least 500ms)
	minExpectedDuration := 500 * time.Millisecond
	if duration < minExpectedDuration {
		t.Errorf("execution took %v, expected at least %v for exponential backoff", duration, minExpectedDuration)
	}
}

func TestRetryExecutor_Execute_ContextCancellation(t *testing.T) {
	t.Run("context cancelled before first retry", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		policy := &workflow.RetryPolicy{
			MaxAttempts:       3,
			InitialDelay:      100 * time.Millisecond,
			MaxDelay:          1 * time.Second,
			BackoffMultiplier: 2.0,
		}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			if callCount == 1 {
				// Cancel context after first failure
				cancel()
			}
			return errors.New("test error")
		}

		err := executor.Execute(ctx, fn)

		// Should have only tried twice (first attempt + cancelled before retry)
		if callCount != 1 {
			t.Errorf("function called %d times, want 1", callCount)
		}

		var retryErr *RetryExhaustedError
		if !errors.As(err, &retryErr) {
			t.Fatalf("Execute() error type = %T, want *RetryExhaustedError", err)
		}
		if retryErr.Reason != "context cancelled: context canceled" {
			t.Errorf("RetryExhaustedError.Reason = %v, want context cancellation", retryErr.Reason)
		}
	})
}

func TestRetryExecutor_ErrorMatching(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		retryableErrors []string
		errorToTest     error
		shouldRetry     bool
	}{
		{
			name:            "retry connection error by type",
			retryableErrors: []string{"connection"},
			errorToTest: &execution.ExecutionError{
				Type:    execution.ErrorTypeConnection,
				Message: "failed to connect",
			},
			shouldRetry: true,
		},
		{
			name:            "retry timeout error by type",
			retryableErrors: []string{"timeout"},
			errorToTest: &execution.ExecutionError{
				Type:    execution.ErrorTypeTimeout,
				Message: "operation timed out",
			},
			shouldRetry: true,
		},
		{
			name:            "retry by regex pattern",
			retryableErrors: []string{"temporary.*error"},
			errorToTest:     errors.New("temporary network error"),
			shouldRetry:     true,
		},
		{
			name:            "retry by substring match",
			retryableErrors: []string{"rate limit"},
			errorToTest:     errors.New("rate limit exceeded"),
			shouldRetry:     true,
		},
		{
			name:            "don't retry validation error",
			retryableErrors: []string{"connection", "timeout"},
			errorToTest: &execution.ExecutionError{
				Type:    execution.ErrorTypeValidation,
				Message: "invalid input",
			},
			shouldRetry: false,
		},
		{
			name:            "retry all when no patterns specified",
			retryableErrors: []string{},
			errorToTest:     errors.New("any error"),
			shouldRetry:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &workflow.RetryPolicy{
				MaxAttempts:       2,
				InitialDelay:      10 * time.Millisecond,
				RetryableErrors:   tt.retryableErrors,
				BackoffMultiplier: 2.0,
			}
			executor := NewRetryExecutor(policy)

			callCount := 0
			fn := func() error {
				callCount++
				return tt.errorToTest
			}

			executor.Execute(ctx, fn)

			if tt.shouldRetry {
				// Should try initial + retries
				if callCount <= 1 {
					t.Errorf("function called %d times, expected retries", callCount)
				}
			} else {
				// Should only try once
				if callCount != 1 {
					t.Errorf("function called %d times, want 1 (no retry)", callCount)
				}
			}
		})
	}
}

func TestRetryExecutor_NonRetryableErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("non-retryable errors take precedence", func(t *testing.T) {
		policy := &workflow.RetryPolicy{
			MaxAttempts:        3,
			InitialDelay:       10 * time.Millisecond,
			RetryableErrors:    []string{"connection", "timeout"},
			NonRetryableErrors: []string{"connection refused"},
			BackoffMultiplier:  2.0,
		}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			return errors.New("connection refused by server")
		}

		err := executor.Execute(ctx, fn)

		// Should not retry
		if callCount != 1 {
			t.Errorf("function called %d times, want 1 (no retry due to non-retryable)", callCount)
		}

		var retryErr *RetryExhaustedError
		if !errors.As(err, &retryErr) {
			t.Fatalf("Execute() error type = %T, want *RetryExhaustedError", err)
		}
		if retryErr.Reason != "non-retryable error" {
			t.Errorf("RetryExhaustedError.Reason = %v, want 'non-retryable error'", retryErr.Reason)
		}
	})
}

func TestRetryExecutor_ExecuteWithMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("success metrics", func(t *testing.T) {
		policy := &workflow.RetryPolicy{
			MaxAttempts:       3,
			InitialDelay:      10 * time.Millisecond,
			MaxDelay:          100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		executor := NewRetryExecutor(policy)

		callCount := 0
		fn := func() error {
			callCount++
			if callCount < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		metrics, err := executor.ExecuteWithMetrics(ctx, fn)

		if err != nil {
			t.Errorf("ExecuteWithMetrics() error = %v, want nil", err)
		}
		if !metrics.Success {
			t.Error("metrics.Success = false, want true")
		}
		if metrics.Attempts != 3 {
			t.Errorf("metrics.Attempts = %d, want 3", metrics.Attempts)
		}
		if len(metrics.Errors) != 3 {
			t.Errorf("len(metrics.Errors) = %d, want 3", len(metrics.Errors))
		}
		if len(metrics.Delays) != 2 {
			t.Errorf("len(metrics.Delays) = %d, want 2", len(metrics.Delays))
		}
	})

	t.Run("failure metrics", func(t *testing.T) {
		policy := &workflow.RetryPolicy{
			MaxAttempts:       2,
			InitialDelay:      10 * time.Millisecond,
			MaxDelay:          100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}
		executor := NewRetryExecutor(policy)

		fn := func() error {
			return errors.New("persistent error")
		}

		metrics, err := executor.ExecuteWithMetrics(ctx, fn)

		if err == nil {
			t.Error("ExecuteWithMetrics() error = nil, want error")
		}
		if metrics.Success {
			t.Error("metrics.Success = true, want false")
		}
		if metrics.Attempts != 3 {
			t.Errorf("metrics.Attempts = %d, want 3", metrics.Attempts)
		}
		if metrics.TotalDuration == 0 {
			t.Error("metrics.TotalDuration = 0, want > 0")
		}
	})
}

func TestCalculateDelay(t *testing.T) {
	policy := &workflow.RetryPolicy{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
	}
	executor := NewRetryExecutor(policy)

	tests := []struct {
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{0, 75 * time.Millisecond, 125 * time.Millisecond},   // ~100ms ± 25%
		{1, 150 * time.Millisecond, 250 * time.Millisecond},  // ~200ms ± 25%
		{2, 300 * time.Millisecond, 500 * time.Millisecond},  // ~400ms ± 25%
		{3, 600 * time.Millisecond, 1 * time.Second},         // ~800ms ± 25%, capped at 1s
		{4, 750 * time.Millisecond, 1250 * time.Millisecond}, // would be ~1.6s, but capped at 1s, then jitter can add 25%
	}

	for _, tt := range tests {
		t.Run("attempt "+string(rune(tt.attempt+'0')), func(t *testing.T) {
			// Test multiple times due to jitter
			for i := 0; i < 10; i++ {
				delay := executor.calculateDelay(tt.attempt)
				if delay < tt.minExpected || delay > tt.maxExpected {
					t.Errorf("calculateDelay(%d) = %v, want between %v and %v",
						tt.attempt, delay, tt.minExpected, tt.maxExpected)
				}
			}
		})
	}
}

func TestExtractErrorType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want execution.ErrorType
	}{
		{
			name: "ExecutionError connection type",
			err: &execution.ExecutionError{
				Type: execution.ErrorTypeConnection,
			},
			want: execution.ErrorTypeConnection,
		},
		{
			name: "MCPToolError",
			err:  &MCPToolError{},
			want: execution.ErrorTypeConnection,
		},
		{
			name: "TransformError",
			err:  &TransformError{},
			want: execution.ErrorTypeData,
		},
		{
			name: "timeout heuristic",
			err:  errors.New("operation timeout exceeded"),
			want: execution.ErrorTypeTimeout,
		},
		{
			name: "connection heuristic",
			err:  errors.New("connection refused"),
			want: execution.ErrorTypeConnection,
		},
		{
			name: "validation heuristic",
			err:  errors.New("invalid parameter value"),
			want: execution.ErrorTypeValidation,
		},
		{
			name: "generic error",
			err:  errors.New("something went wrong"),
			want: execution.ErrorTypeExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractErrorType(tt.err)
			if got != tt.want {
				t.Errorf("extractErrorType() = %v, want %v", got, tt.want)
			}
		})
	}
}
