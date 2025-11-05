package execution

import (
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// NodeExecution represents a record of a single node execution within a workflow run.
// It is a value object that captures the inputs, outputs, and execution details of a node.
type NodeExecution struct {
	// ID is the unique identifier for this node execution.
	ID types.NodeExecutionID
	// ExecutionID is the parent execution reference.
	ExecutionID types.ExecutionID
	// NodeID identifies which node was executed.
	NodeID types.NodeID
	// NodeType is the type of node (e.g., "mcp_tool", "transform", "condition").
	NodeType string
	// Status is the current status of this node execution.
	Status NodeStatus
	// StartedAt is when the node execution began.
	StartedAt time.Time
	// CompletedAt is when the node execution finished (zero if still running).
	CompletedAt time.Time
	// Inputs contains the input values provided to the node.
	Inputs map[string]interface{}
	// Outputs contains the output values produced by the node.
	Outputs map[string]interface{}
	// Error contains error details if the node failed.
	Error *NodeError
	// RetryCount is the number of retries attempted for this node.
	RetryCount int
}

// NewNodeExecution creates a new node execution record.
func NewNodeExecution(executionID types.ExecutionID, nodeID types.NodeID, nodeType string) *NodeExecution {
	return &NodeExecution{
		ID:          types.NewNodeExecutionID(),
		ExecutionID: executionID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Status:      NodeStatusPending,
		Inputs:      make(map[string]interface{}),
		Outputs:     make(map[string]interface{}),
		RetryCount:  0,
	}
}

// Start marks the node execution as started.
func (ne *NodeExecution) Start() {
	ne.Status = NodeStatusRunning
	ne.StartedAt = time.Now()
}

// Complete marks the node execution as successfully completed with outputs.
func (ne *NodeExecution) Complete(outputs map[string]interface{}) {
	ne.Status = NodeStatusCompleted
	ne.CompletedAt = time.Now()
	if outputs != nil {
		ne.Outputs = outputs
	}
}

// Fail marks the node execution as failed with error details.
func (ne *NodeExecution) Fail(err *NodeError) {
	ne.Status = NodeStatusFailed
	ne.CompletedAt = time.Now()
	ne.Error = err
}

// Skip marks the node execution as skipped (e.g., conditional branch not taken).
func (ne *NodeExecution) Skip() {
	ne.Status = NodeStatusSkipped
	ne.CompletedAt = time.Now()
}

// Duration returns the execution time for this node.
// Returns 0 if the node hasn't completed yet.
func (ne *NodeExecution) Duration() time.Duration {
	if ne.CompletedAt.IsZero() {
		return 0
	}
	return ne.CompletedAt.Sub(ne.StartedAt)
}

// IncrementRetry increments the retry count for this node execution.
func (ne *NodeExecution) IncrementRetry() {
	ne.RetryCount++
}
