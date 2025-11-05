package execution

import "github.com/dshills/goflow/pkg/domain/types"

// ExecutionRepository defines the interface for persisting and retrieving executions.
// Implementations will typically use SQLite for storage.
type ExecutionRepository interface {
	// Save persists an execution to storage.
	// Updates the execution if it already exists.
	Save(execution *Execution) error

	// Load retrieves an execution by its ID.
	// Returns an error if the execution is not found.
	Load(id types.ExecutionID) (*Execution, error)

	// ListByWorkflow returns all executions for a specific workflow.
	// Results are typically ordered by StartedAt descending (most recent first).
	ListByWorkflow(workflowID types.WorkflowID) ([]*Execution, error)

	// ListByStatus returns all executions with a specific status.
	// Useful for finding running, failed, or pending executions.
	ListByStatus(status Status) ([]*Execution, error)

	// Delete removes an execution and all its related data from storage.
	Delete(id types.ExecutionID) error

	// SaveNodeExecution persists a node execution record.
	// Node executions are typically saved as they complete during workflow execution.
	SaveNodeExecution(nodeExec *NodeExecution) error

	// SaveVariableSnapshot persists a variable snapshot to the audit trail.
	// Snapshots are append-only and never modified.
	SaveVariableSnapshot(snapshot *VariableSnapshot) error
}
