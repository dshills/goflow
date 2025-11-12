package errors

import (
	"fmt"
	"time"
)

// OperationalError represents enhanced error information for debugging.
//
// It wraps errors with operational context including workflow ID, node ID,
// and timestamp. This enables better error tracking and debugging in workflow
// executions.
type OperationalError struct {
	Operation  string                 // What operation was being performed
	WorkflowID string                 // Which workflow
	NodeID     string                 // Which node (if applicable)
	Timestamp  time.Time              // When error occurred
	Attributes map[string]interface{} // Additional context (optional)
	Cause      error                  // Underlying error
}

// NewOperationalError creates an OperationalError wrapping an error.
//
// Returns nil if cause is nil (no error to wrap).
//
// Example:
//
//	if err != nil {
//	    return NewOperationalError("executing node", workflowID, nodeID, err)
//	}
func NewOperationalError(operation, workflowID, nodeID string, cause error) *OperationalError {
	if cause == nil {
		return nil
	}

	return &OperationalError{
		Operation:  operation,
		WorkflowID: workflowID,
		NodeID:     nodeID,
		Timestamp:  time.Now(),
		Attributes: nil,
		Cause:      cause,
	}
}

// NewOperationalErrorWithAttrs creates an OperationalError with additional attributes.
//
// Returns nil if cause is nil (no error to wrap).
//
// Example:
//
//	return NewOperationalErrorWithAttrs(
//	    "validating input",
//	    workflowID,
//	    nodeID,
//	    err,
//	    map[string]interface{}{
//	        "inputSize": len(input),
//	        "schema":    schemaName,
//	    },
//	)
func NewOperationalErrorWithAttrs(operation, workflowID, nodeID string, cause error, attrs map[string]interface{}) *OperationalError {
	if cause == nil {
		return nil
	}

	return &OperationalError{
		Operation:  operation,
		WorkflowID: workflowID,
		NodeID:     nodeID,
		Timestamp:  time.Now(),
		Attributes: attrs,
		Cause:      cause,
	}
}

// Error implements the error interface.
//
// Format: "[timestamp] operation: workflow={id} node={id}: {cause}"
// If node ID is empty, it's omitted from the message.
func (e *OperationalError) Error() string {
	if e == nil {
		return "<nil OperationalError>"
	}

	timestamp := e.Timestamp.Format(time.RFC3339)

	// Build message components
	var msg string
	if e.NodeID != "" {
		msg = fmt.Sprintf("[%s] %s: workflow=%s node=%s: %v",
			timestamp,
			e.Operation,
			e.WorkflowID,
			e.NodeID,
			e.Cause)
	} else {
		msg = fmt.Sprintf("[%s] %s: workflow=%s: %v",
			timestamp,
			e.Operation,
			e.WorkflowID,
			e.Cause)
	}

	return msg
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *OperationalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
