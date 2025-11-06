package execution

import (
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// ListOptions defines filtering and pagination options for querying executions.
// All filter fields are optional - nil values mean no filtering on that dimension.
type ListOptions struct {
	// Pagination
	Limit  int // Maximum number of results to return (0 = no limit)
	Offset int // Number of results to skip

	// Filtering
	WorkflowID         *types.WorkflowID // Filter by specific workflow ID
	Status             *Status           // Filter by execution status
	StartedAfter       *time.Time        // Filter executions started after this time
	StartedBefore      *time.Time        // Filter executions started before this time
	WorkflowNameSearch *string           // Search workflow names (case-insensitive substring match)
}

// ListResult contains the results of a List query along with metadata.
type ListResult struct {
	// Executions contains the matching execution records
	Executions []*Execution

	// TotalCount is the total number of matching records (without pagination)
	// Useful for calculating total pages in UI
	TotalCount int

	// Limit is the limit value used in the query
	Limit int

	// Offset is the offset value used in the query
	Offset int
}

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

	// List returns executions with advanced filtering and pagination support.
	// Supports filtering by workflow ID, status, date range, and workflow name search.
	// All filters are optional and can be combined.
	// Results are ordered by StartedAt descending (most recent first).
	List(options ListOptions) (*ListResult, error)

	// Delete removes an execution and all its related data from storage.
	Delete(id types.ExecutionID) error

	// SaveNodeExecution persists a node execution record.
	// Node executions are typically saved as they complete during workflow execution.
	SaveNodeExecution(nodeExec *NodeExecution) error

	// SaveVariableSnapshot persists a variable snapshot to the audit trail.
	// Snapshots are append-only and never modified.
	SaveVariableSnapshot(snapshot *VariableSnapshot) error
}
