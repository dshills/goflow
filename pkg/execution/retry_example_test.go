package execution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/workflow"
)

// Example demonstrating basic retry policy usage
func ExampleRetryExecutor_basic() {
	// Configure a retry policy
	policy := &workflow.RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	executor := NewRetryExecutor(policy)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := executor.Execute(context.Background(), operation)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	} else {
		fmt.Printf("Succeeded after %d attempts\n", attemptCount)
	}

	// Output:
	// Succeeded after 3 attempts
}

// Example demonstrating retry with error type filtering
func ExampleRetryExecutor_errorTypeFiltering() {
	// Only retry connection and timeout errors
	policy := &workflow.RetryPolicy{
		MaxAttempts:       2,
		InitialDelay:      50 * time.Millisecond,
		MaxDelay:          500 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableErrors:   []string{"connection", "timeout"},
	}

	executor := NewRetryExecutor(policy)

	// This will be retried (connection error)
	operation1 := func() error {
		return errors.New("connection refused")
	}

	err1 := executor.Execute(context.Background(), operation1)
	if retryErr, ok := err1.(*RetryExhaustedError); ok {
		fmt.Printf("Connection error exhausted retries after %d attempts\n", retryErr.Attempts)
	}

	// This will NOT be retried (validation error)
	operation2 := func() error {
		return errors.New("invalid parameter")
	}

	err2 := executor.Execute(context.Background(), operation2)
	if retryErr, ok := err2.(*RetryExhaustedError); ok {
		fmt.Printf("Validation error not retried: %d attempts\n", retryErr.Attempts)
	}

	// Output:
	// Connection error exhausted retries after 3 attempts
	// Validation error not retried: 1 attempts
}

// Example demonstrating context cancellation with retry
func ExampleRetryExecutor_contextCancellation() {
	policy := &workflow.RetryPolicy{
		MaxAttempts:       10,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	executor := NewRetryExecutor(policy)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	attemptCount := 0
	operation := func() error {
		attemptCount++
		return errors.New("persistent error")
	}

	err := executor.Execute(ctx, operation)
	if retryErr, ok := err.(*RetryExhaustedError); ok {
		fmt.Printf("Stopped after %d attempts due to context cancellation\n", retryErr.Attempts)
	}

	// Output:
	// Stopped after 2 attempts due to context cancellation
}

// Example demonstrating retry metrics collection
func ExampleRetryExecutor_withMetrics() {
	policy := &workflow.RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	executor := NewRetryExecutor(policy)

	attemptCount := 0
	operation := func() error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	metrics, err := executor.ExecuteWithMetrics(context.Background(), operation)

	if err == nil {
		fmt.Printf("Success: %d attempts, %d delays, took %v\n",
			metrics.Attempts,
			len(metrics.Delays),
			metrics.TotalDuration > 0)
	}

	// Output:
	// Success: 3 attempts, 2 delays, took true
}

// Example demonstrating non-retryable error patterns
func ExampleRetryExecutor_nonRetryablePatterns() {
	// Retry all errors except specific patterns
	policy := &workflow.RetryPolicy{
		MaxAttempts:        3,
		InitialDelay:       50 * time.Millisecond,
		MaxDelay:           500 * time.Millisecond,
		BackoffMultiplier:  2.0,
		RetryableErrors:    []string{".*"}, // Retry everything
		NonRetryableErrors: []string{"fatal", "invalid"},
	}

	executor := NewRetryExecutor(policy)

	// This will NOT be retried (matches non-retryable pattern)
	operation1 := func() error {
		return errors.New("fatal: database corrupted")
	}

	err1 := executor.Execute(context.Background(), operation1)
	if retryErr, ok := err1.(*RetryExhaustedError); ok {
		fmt.Printf("Fatal error not retried: %d attempt\n", retryErr.Attempts)
	}

	// This WILL be retried (doesn't match non-retryable pattern)
	attemptCount := 0
	operation2 := func() error {
		attemptCount++
		if attemptCount < 2 {
			return errors.New("temporary network error")
		}
		return nil
	}

	err2 := executor.Execute(context.Background(), operation2)
	if err2 == nil {
		fmt.Printf("Temporary error succeeded after retry\n")
	}

	// Output:
	// Fatal error not retried: 1 attempt
	// Temporary error succeeded after retry
}

// Example demonstrating exponential backoff calculation
func ExampleRetryPolicy_exponentialBackoff() {
	policy := &workflow.RetryPolicy{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          2 * time.Second,
		BackoffMultiplier: 2.0,
	}
	policy.SetDefaults()

	executor := NewRetryExecutor(policy)

	fmt.Println("Exponential backoff with jitter:")
	for attempt := 0; attempt < 5; attempt++ {
		delay := executor.calculateDelay(attempt)
		// Display approximate delay (actual has jitter)
		baseDelay := policy.InitialDelay * time.Duration(1<<attempt)
		if baseDelay > policy.MaxDelay {
			baseDelay = policy.MaxDelay
		}
		fmt.Printf("Attempt %d: ~%v (base), actual: %v\n", attempt, baseDelay, delay > 0)
	}

	// Output:
	// Exponential backoff with jitter:
	// Attempt 0: ~100ms (base), actual: true
	// Attempt 1: ~200ms (base), actual: true
	// Attempt 2: ~400ms (base), actual: true
	// Attempt 3: ~800ms (base), actual: true
	// Attempt 4: ~1.6s (base), actual: true
}
