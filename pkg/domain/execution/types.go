// Package execution defines the Execution aggregate for GoFlow workflow execution.
package execution

import (
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// Status represents the current state of a workflow execution.
type Status string

const (
	// StatusPending indicates the execution is created but not yet started.
	StatusPending Status = "pending"
	// StatusRunning indicates the execution is currently in progress.
	StatusRunning Status = "running"
	// StatusCompleted indicates the execution finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed indicates the execution encountered an error and stopped.
	StatusFailed Status = "failed"
	// StatusCancelled indicates the execution was cancelled by the user.
	StatusCancelled Status = "cancelled"
)

// IsTerminal returns true if the status represents a terminal state (execution has finished).
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}

// NodeStatus represents the current state of a node execution.
type NodeStatus string

const (
	// NodeStatusPending indicates the node is waiting to be executed.
	NodeStatusPending NodeStatus = "pending"
	// NodeStatusRunning indicates the node is currently executing.
	NodeStatusRunning NodeStatus = "running"
	// NodeStatusCompleted indicates the node finished successfully.
	NodeStatusCompleted NodeStatus = "completed"
	// NodeStatusFailed indicates the node encountered an error.
	NodeStatusFailed NodeStatus = "failed"
	// NodeStatusSkipped indicates the node was skipped (e.g., conditional branch not taken).
	NodeStatusSkipped NodeStatus = "skipped"
)

// ErrorType categorizes different types of execution errors.
type ErrorType string

const (
	// ErrorTypeValidation indicates an error in workflow validation (schema, parameters, types).
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeConnection indicates an MCP server communication failure.
	ErrorTypeConnection ErrorType = "connection"
	// ErrorTypeExecution indicates a runtime failure (tool errors, timeouts, resources).
	ErrorTypeExecution ErrorType = "execution"
	// ErrorTypeData indicates a transformation failure (invalid JSONPath, type conversions).
	ErrorTypeData ErrorType = "data"
	// ErrorTypeTimeout indicates the execution exceeded its time limit.
	ErrorTypeTimeout ErrorType = "timeout"
)

// ExecutionError represents detailed error information for failed executions.
type ExecutionError struct {
	// Type categorizes the error for appropriate handling.
	Type ErrorType
	// Message is a human-readable error description.
	Message string
	// NodeID identifies where the error occurred (if applicable).
	NodeID types.NodeID
	// StackTrace captures the Go stack trace for debugging.
	StackTrace string
	// Context provides additional error context (e.g., parameter values, server response).
	Context map[string]interface{}
	// Recoverable indicates whether retrying might succeed.
	Recoverable bool
	// Timestamp records when the error occurred.
	Timestamp time.Time
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	if e.NodeID != "" {
		return fmt.Sprintf("[%s] node %s: %s", e.Type, e.NodeID, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NodeError represents error information specific to a node execution failure.
type NodeError struct {
	// Type categorizes the error.
	Type ErrorType
	// Message is a human-readable error description.
	Message string
	// StackTrace captures the Go stack trace for debugging.
	StackTrace string
	// Context provides additional error context.
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *NodeError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// TraceEntry represents a single event in the execution trace.
type TraceEntry struct {
	// NodeID identifies which node this trace entry is for.
	NodeID types.NodeID
	// Event describes what happened (e.g., "started", "completed", "failed").
	Event string
	// Timestamp records when this event occurred.
	Timestamp time.Time
	// Metadata provides additional context about the event.
	Metadata map[string]interface{}
}
