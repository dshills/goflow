// Package contracts defines API contracts for the 002-pr-review-remediation feature.
//
// This file documents contract changes (API modifications) for the execution package.
// Implementation will be in pkg/execution/
package contracts

import (
	"context"
	"time"
)

// ExecutionContext represents the state of a workflow execution.
//
// ENHANCED: Now includes timeout support via context.Context
type ExecutionContext struct {
	// Existing fields (unchanged)
	WorkflowID  string
	ExecutionID string
	StartTime   time.Time
	Status      ExecutionStatus
	Variables   map[string]interface{}
	NodeTrace   []NodeExecution

	// NEW: Timeout support
	ctx             context.Context    // Context with timeout/deadline
	cancel          context.CancelFunc // Cancellation function
	TimeoutDuration time.Duration      // Configured timeout (0 = no timeout)
	TimedOut        bool               // Whether execution timed out
	TimeoutNode     string             // Node ID executing when timeout occurred
}

// Context returns the execution context for timeout and cancellation.
//
// NEW METHOD: Returns context for passing to node executions.
//
// Example:
//
//	result, err := node.Execute(exec.Context(), input)
func (e *ExecutionContext) Context() context.Context

// Cancel cancels the execution.
//
// NEW METHOD: Explicitly cancels execution (for user-initiated cancellation).
func (e *ExecutionContext) Cancel()

// ErrorContext represents enhanced error information for debugging.
//
// NEW TYPE: Wraps errors with operational context.
type ErrorContext struct {
	Operation  string                 // What operation was being performed
	WorkflowID string                 // Which workflow
	NodeID     string                 // Which node (if applicable)
	Timestamp  time.Time              // When error occurred
	Attributes map[string]interface{} // Additional context (optional)
	Cause      error                  // Underlying error
}

// Error implements the error interface.
//
// Format: "[timestamp] operation: workflow=ID node=ID: cause"
func (e *ErrorContext) Error() string

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ErrorContext) Unwrap() error

// NewErrorContext creates an ErrorContext wrapping an error.
//
// Example:
//
//	if err != nil {
//	    return NewErrorContext("executing node", workflowID, nodeID, err)
//	}
func NewErrorContext(operation, workflowID, nodeID string, cause error) *ErrorContext

// NewErrorContextWithAttrs creates an ErrorContext with additional attributes.
//
// Example:
//
//	return NewErrorContextWithAttrs(
//	    "validating input",
//	    workflowID,
//	    nodeID,
//	    err,
//	    map[string]interface{}{
//	        "inputSize": len(input),
//	        "schema":    schemaName,
//	    },
//	)
func NewErrorContextWithAttrs(operation, workflowID, nodeID string, cause error, attrs map[string]interface{}) *ErrorContext

// Runtime represents the workflow execution engine.
//
// ENHANCED: Now supports timeout configuration
type Runtime struct {
	// ... existing fields ...
}

// NewRuntime creates a new execution runtime.
//
// ENHANCED: Now accepts timeout option
//
// Example:
//
//	runtime := NewRuntime(WithTimeout(5 * time.Minute))
func NewRuntime(opts ...RuntimeOption) *Runtime

// RuntimeOption is a functional option for runtime configuration.
//
// NEW TYPE: Enables flexible runtime configuration
type RuntimeOption func(*Runtime)

// WithTimeout configures a default timeout for workflow executions.
//
// NEW FUNCTION: Sets execution timeout
//
// Example:
//
//	runtime := NewRuntime(WithTimeout(5 * time.Minute))
func WithTimeout(timeout time.Duration) RuntimeOption

// Execute executes a workflow.
//
// ENHANCED: Now respects context timeout
//
// If ctx has a deadline, execution will timeout when deadline is exceeded.
// If timeout occurs, returns *ErrorContext with TimedOut status.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
//	defer cancel()
//	result, err := runtime.Execute(ctx, workflow)
func (r *Runtime) Execute(ctx context.Context, workflow *Workflow) (*ExecutionResult, error)

// ENHANCED: ExecutionStatus enum adds TimedOut status
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimedOut  ExecutionStatus = "timed_out" // NEW: Timeout status
)
