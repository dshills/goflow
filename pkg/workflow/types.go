package workflow

import (
	"errors"

	"github.com/google/uuid"
)

// Common workflow errors
var (
	// ErrWorkflowNotFound is returned when a workflow cannot be found
	ErrWorkflowNotFound = errors.New("workflow not found")
)

// WorkflowID is a unique identifier for a workflow
type WorkflowID string

// String returns the string representation of the WorkflowID
func (w WorkflowID) String() string {
	return string(w)
}

// NewWorkflowID generates a new unique WorkflowID
func NewWorkflowID() WorkflowID {
	return WorkflowID(uuid.New().String())
}

// NodeID is a unique identifier for a node within a workflow
type NodeID string

// String returns the string representation of the NodeID
func (n NodeID) String() string {
	return string(n)
}

// EdgeID is a unique identifier for an edge within a workflow
type EdgeID string

// String returns the string representation of the EdgeID
func (e EdgeID) String() string {
	return string(e)
}

// NewEdgeID generates a new unique EdgeID
func NewEdgeID() EdgeID {
	return EdgeID(uuid.New().String())
}
