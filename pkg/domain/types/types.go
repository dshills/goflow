// Package types defines core domain type aliases and identifiers for GoFlow.
package types

import "github.com/google/uuid"

// WorkflowID is a unique identifier for a workflow.
type WorkflowID string

// NodeID is a unique identifier for a node within a workflow.
type NodeID string

// ExecutionID is a unique identifier for a workflow execution.
type ExecutionID string

// NodeExecutionID is a unique identifier for a node execution within a workflow execution.
type NodeExecutionID string

// NewExecutionID generates a new unique execution ID.
func NewExecutionID() ExecutionID {
	return ExecutionID(uuid.NewString())
}

// String returns the string representation of an ExecutionID.
func (id ExecutionID) String() string {
	return string(id)
}

// IsZero returns true if the ExecutionID is the zero value.
func (id ExecutionID) IsZero() bool {
	return id == ""
}

// NewNodeExecutionID generates a new unique node execution ID.
func NewNodeExecutionID() NodeExecutionID {
	return NodeExecutionID(uuid.NewString())
}
