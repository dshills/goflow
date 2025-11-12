package execution

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// RetryState tracks the current retry attempt state
type RetryState struct {
	Attempt       int
	LastDelay     time.Duration
	TotalDuration time.Duration
	Errors        []error
}

// RetryExecutor handles retry logic for node execution
type RetryExecutor struct {
	policy *workflow.RetryPolicy
}

// NewRetryExecutor creates a new retry executor with the given policy
func NewRetryExecutor(policy *workflow.RetryPolicy) *RetryExecutor {
	if policy == nil {
		return nil
	}
	// Set defaults if not already set
	policyCopy := *policy
	policyCopy.SetDefaults()
	return &RetryExecutor{
		policy: &policyCopy,
	}
}

// Execute attempts to execute the given function with retry logic
func (r *RetryExecutor) Execute(ctx context.Context, fn func() error) error {
	// Validate function is not nil
	if fn == nil {
		return fmt.Errorf("nil function")
	}

	if r == nil || r.policy == nil || !r.policy.IsEnabled() {
		// No retry configured, execute once
		return fn()
	}

	state := &RetryState{
		Errors: make([]error, 0, r.policy.MaxAttempts+1),
	}

	var lastErr error
	maxAttempts := r.policy.MaxAttempts + 1 // +1 for initial attempt

	for attempt := 0; attempt < maxAttempts; attempt++ {
		state.Attempt = attempt

		// Execute the function
		startTime := time.Now()
		err := fn()
		duration := time.Since(startTime)
		state.TotalDuration += duration

		// Success - return immediately
		if err == nil {
			return nil
		}

		// Record the error
		lastErr = err
		state.Errors = append(state.Errors, err)

		// Check if we should retry this error
		if !r.shouldRetry(err) {
			return r.wrapNonRetryableError(err, state)
		}

		// Check if this was the last attempt
		if attempt >= r.policy.MaxAttempts {
			return r.wrapMaxAttemptsError(lastErr, state)
		}

		// Check context cancellation before sleeping
		if ctx.Err() != nil {
			return r.wrapContextError(ctx.Err(), state)
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt)
		state.LastDelay = delay

		// Sleep with context awareness (using timer to avoid leaks)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			// Continue to next attempt
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return r.wrapContextError(ctx.Err(), state)
		}
	}

	// Should not reach here, but handle it gracefully
	return r.wrapMaxAttemptsError(lastErr, state)
}

// calculateDelay computes the delay before the next retry with exponential backoff and jitter
func (r *RetryExecutor) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: initialDelay * (multiplier ^ attempt)
	delay := float64(r.policy.InitialDelay) * math.Pow(r.policy.BackoffMultiplier, float64(attempt))

	// Guard against overflow/Inf/NaN from math.Pow
	if math.IsInf(delay, 0) || math.IsNaN(delay) {
		delay = float64(r.policy.MaxDelay)
	}

	// Add jitter (±25% randomization) to prevent thundering herd
	jitter := delay * 0.25 * (rand.Float64()*2 - 1) // Random value between -0.25 and +0.25
	delay += jitter

	// Apply max delay ceiling AFTER jitter
	if r.policy.MaxDelay > 0 && delay > float64(r.policy.MaxDelay) {
		delay = float64(r.policy.MaxDelay)
	}

	// Ensure delay is non-negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// shouldRetry determines if an error should trigger a retry
func (r *RetryExecutor) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check non-retryable errors first (denylist takes precedence)
	if len(r.policy.NonRetryableErrors) > 0 {
		if matchesErrorPatterns(err, r.policy.NonRetryableErrors) {
			return false
		}
	}

	// If no retryable errors specified, retry all errors (except those in non-retryable list)
	if len(r.policy.RetryableErrors) == 0 {
		return true
	}

	// Check if error matches retryable patterns (allowlist)
	return matchesErrorPatterns(err, r.policy.RetryableErrors)
}

// matchesErrorPatterns checks if an error matches any of the given patterns
func matchesErrorPatterns(err error, patterns []string) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	errType := extractErrorType(err)

	for _, pattern := range patterns {
		// Check for error type match (connection, timeout, rate_limit, etc.)
		if isErrorTypeMatch(pattern, errType) {
			return true
		}

		// Check for regex pattern match against error message
		if matched, _ := regexp.MatchString(pattern, errMsg); matched {
			return true
		}

		// Check for simple substring match (case-insensitive)
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// extractErrorType determines the error type from an error
func extractErrorType(err error) execution.ErrorType {
	// Check if it's an ExecutionError
	if execErr, ok := err.(*execution.ExecutionError); ok {
		return execErr.Type
	}

	// Check specific error types from node_executor.go
	if _, ok := err.(*MCPToolError); ok {
		return execution.ErrorTypeConnection
	}
	if _, ok := err.(*TransformError); ok {
		return execution.ErrorTypeData
	}
	if _, ok := err.(*ConditionError); ok {
		return execution.ErrorTypeData
	}

	// Heuristic-based type detection from error message
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
		return execution.ErrorTypeTimeout
	}
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "dial") || strings.Contains(errMsg, "refused") {
		return execution.ErrorTypeConnection
	}
	if strings.Contains(errMsg, "validation") || strings.Contains(errMsg, "invalid") {
		return execution.ErrorTypeValidation
	}

	return execution.ErrorTypeExecution
}

// isErrorTypeMatch checks if a pattern matches an error type
func isErrorTypeMatch(pattern string, errType execution.ErrorType) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Map common error type names to ErrorType constants
	switch pattern {
	case "connection", "connection_error":
		return errType == execution.ErrorTypeConnection
	case "timeout", "timeout_error":
		return errType == execution.ErrorTypeTimeout
	case "validation", "validation_error":
		return errType == execution.ErrorTypeValidation
	case "data", "data_error", "transform", "transform_error":
		return errType == execution.ErrorTypeData
	case "execution", "execution_error":
		return errType == execution.ErrorTypeExecution
	case "rate_limit", "rate_limited", "throttle", "throttled":
		// Rate limiting typically manifests as connection or execution errors
		// Check error message for rate limit indicators
		return false // Let regex/substring matching handle this
	}

	return false
}

// wrapNonRetryableError wraps an error that should not be retried
func (r *RetryExecutor) wrapNonRetryableError(err error, state *RetryState) error {
	return &RetryExhaustedError{
		Reason:        "non-retryable error",
		Attempts:      state.Attempt + 1,
		LastError:     err,
		AllErrors:     state.Errors,
		TotalDuration: state.TotalDuration,
	}
}

// wrapMaxAttemptsError wraps an error when max attempts are reached
func (r *RetryExecutor) wrapMaxAttemptsError(err error, state *RetryState) error {
	return &RetryExhaustedError{
		Reason:        fmt.Sprintf("max attempts (%d) reached", r.policy.MaxAttempts+1),
		Attempts:      state.Attempt + 1,
		LastError:     err,
		AllErrors:     state.Errors,
		TotalDuration: state.TotalDuration,
	}
}

// wrapContextError wraps an error when context is cancelled
func (r *RetryExecutor) wrapContextError(ctxErr error, state *RetryState) error {
	var lastErr error
	if len(state.Errors) > 0 {
		lastErr = state.Errors[len(state.Errors)-1]
	}

	return &RetryExhaustedError{
		Reason:        fmt.Sprintf("context cancelled: %v", ctxErr),
		Attempts:      state.Attempt + 1,
		LastError:     lastErr,
		AllErrors:     state.Errors,
		TotalDuration: state.TotalDuration,
	}
}

// RetryExhaustedError is returned when retries are exhausted
type RetryExhaustedError struct {
	Reason        string
	Attempts      int
	LastError     error
	AllErrors     []error
	TotalDuration time.Duration
}

// Error implements the error interface
func (e *RetryExhaustedError) Error() string {
	return fmt.Sprintf("retry exhausted: %s after %d attempts (took %v): %v",
		e.Reason, e.Attempts, e.TotalDuration, e.LastError)
}

// Unwrap returns the last error for error chain unwrapping
func (e *RetryExhaustedError) Unwrap() error {
	return e.LastError
}

// RetryMetrics provides metrics about retry execution
type RetryMetrics struct {
	Attempts      int
	TotalDuration time.Duration
	Delays        []time.Duration
	Errors        []error
	Success       bool
}

// ExecuteWithMetrics executes with retry and returns detailed metrics
func (r *RetryExecutor) ExecuteWithMetrics(ctx context.Context, fn func() error) (*RetryMetrics, error) {
	if r == nil || r.policy == nil || !r.policy.IsEnabled() {
		// No retry configured
		err := fn()
		return &RetryMetrics{
			Attempts:      1,
			TotalDuration: 0,
			Delays:        []time.Duration{},
			Errors:        []error{err},
			Success:       err == nil,
		}, err
	}

	metrics := &RetryMetrics{
		Delays: make([]time.Duration, 0, r.policy.MaxAttempts),
		Errors: make([]error, 0, r.policy.MaxAttempts+1),
	}

	startTime := time.Now()
	state := &RetryState{
		Errors: make([]error, 0, r.policy.MaxAttempts+1),
	}

	var lastErr error
	maxAttempts := r.policy.MaxAttempts + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		state.Attempt = attempt
		metrics.Attempts = attempt + 1

		// Execute the function
		err := fn()
		metrics.Errors = append(metrics.Errors, err)

		// Success
		if err == nil {
			metrics.Success = true
			metrics.TotalDuration = time.Since(startTime)
			return metrics, nil
		}

		lastErr = err
		state.Errors = append(state.Errors, err)

		// Check if we should retry
		if !r.shouldRetry(err) || attempt >= r.policy.MaxAttempts {
			metrics.TotalDuration = time.Since(startTime)
			if !r.shouldRetry(err) {
				return metrics, r.wrapNonRetryableError(err, state)
			}
			return metrics, r.wrapMaxAttemptsError(lastErr, state)
		}

		// Context check
		if ctx.Err() != nil {
			metrics.TotalDuration = time.Since(startTime)
			return metrics, r.wrapContextError(ctx.Err(), state)
		}

		// Calculate and record delay
		delay := r.calculateDelay(attempt)
		metrics.Delays = append(metrics.Delays, delay)

		// Sleep with context awareness
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			metrics.TotalDuration = time.Since(startTime)
			return metrics, r.wrapContextError(ctx.Err(), state)
		}
	}

	metrics.TotalDuration = time.Since(startTime)
	return metrics, r.wrapMaxAttemptsError(lastErr, state)
}
