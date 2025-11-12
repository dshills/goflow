package execution

import (
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// Execution represents a single run of a workflow with specific inputs.
// It is the root entity of the Execution aggregate.
type Execution struct {
	// ID is the unique identifier for this execution.
	ID types.ExecutionID
	// WorkflowID references the workflow being executed.
	WorkflowID types.WorkflowID
	// WorkflowVersion captures the workflow version at execution time.
	WorkflowVersion string
	// Status is the current execution state.
	Status Status
	// StartedAt is when the execution was created/initialized.
	StartedAt time.Time
	// CompletedAt is when the execution finished (nil if still running).
	CompletedAt time.Time
	// Error contains error details if the execution failed.
	Error *ExecutionError
	// Context holds the runtime state during execution.
	Context *ExecutionContext
	// NodeExecutions tracks the history of node executions in order.
	NodeExecutions []*NodeExecution
	// ReturnValue is the final output from the End node.
	ReturnValue interface{}
}

// NewExecution creates a new execution for a workflow.
// The execution starts in Pending status with an initialized context.
func NewExecution(workflowID types.WorkflowID, workflowVersion string, inputs map[string]interface{}) (*Execution, error) {
	if workflowID == "" {
		return nil, fmt.Errorf("workflow ID cannot be empty")
	}
	if workflowVersion == "" {
		return nil, fmt.Errorf("workflow version cannot be empty")
	}

	ctx, err := NewExecutionContext(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution context: %w", err)
	}

	return &Execution{
		ID:              types.NewExecutionID(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflowVersion,
		Status:          StatusPending,
		StartedAt:       time.Now(),
		Context:         ctx,
		NodeExecutions:  []*NodeExecution{},
	}, nil
}

// Start transitions the execution from Pending to Running.
// Returns an error if the execution is not in Pending status.
func (e *Execution) Start() error {
	if e.Status != StatusPending {
		return fmt.Errorf("cannot start execution: expected status Pending, got %s", e.Status)
	}

	e.Status = StatusRunning
	e.StartedAt = time.Now()
	return nil
}

// Complete marks the execution as successfully completed with an optional return value.
// Returns an error if the execution is not in Running status.
func (e *Execution) Complete(returnValue interface{}) error {
	if e.Status != StatusRunning {
		return fmt.Errorf("cannot complete execution: expected status Running, got %s", e.Status)
	}

	e.Status = StatusCompleted
	e.CompletedAt = time.Now()
	e.ReturnValue = returnValue
	return nil
}

// Fail marks the execution as failed with error details.
// Returns an error if the execution is not in Running status.
func (e *Execution) Fail(err *ExecutionError) error {
	if e.Status != StatusRunning {
		return fmt.Errorf("cannot fail execution: expected status Running, got %s", e.Status)
	}

	if err != nil {
		err.Timestamp = time.Now()
	}

	e.Status = StatusFailed
	e.CompletedAt = time.Now()
	e.Error = err
	return nil
}

// Cancel marks the execution as cancelled.
// Returns an error if the execution is not in Running status.
func (e *Execution) Cancel() error {
	if e.Status != StatusRunning {
		return fmt.Errorf("cannot cancel execution: expected status Running, got %s", e.Status)
	}

	e.Status = StatusCancelled
	e.CompletedAt = time.Now()
	return nil
}

// Timeout marks the execution as timed out with error details.
// Returns an error if the execution is not in Running status.
func (e *Execution) Timeout(timeoutNode string, err *ExecutionError) error {
	if e.Status != StatusRunning {
		return fmt.Errorf("cannot timeout execution: expected status Running, got %s", e.Status)
	}

	if err != nil {
		err.Timestamp = time.Now()
	}

	e.Status = StatusTimedOut
	e.CompletedAt = time.Now()
	e.Error = err
	e.Context.TimedOut = true
	e.Context.TimeoutNode = timeoutNode
	return nil
}

// AddNodeExecution appends a node execution to the history.
// Node executions maintain topological order.
func (e *Execution) AddNodeExecution(nodeExec *NodeExecution) error {
	if nodeExec == nil {
		return fmt.Errorf("node execution cannot be nil")
	}

	// Assign ExecutionID if not set
	if nodeExec.ExecutionID.IsZero() {
		nodeExec.ExecutionID = e.ID
	}

	// Generate ID if not set
	if nodeExec.ID == "" {
		nodeExec.ID = types.NewNodeExecutionID()
	}

	e.NodeExecutions = append(e.NodeExecutions, nodeExec)
	return nil
}

// Duration returns the total execution time.
// Returns 0 if the execution hasn't completed yet.
func (e *Execution) Duration() time.Duration {
	if e.CompletedAt.IsZero() {
		return 0
	}
	return e.CompletedAt.Sub(e.StartedAt)
}

// SetStatusForTest is a test helper to set the status directly, bypassing state machine validation.
// This should ONLY be used in tests to set up initial state.
func (e *Execution) SetStatusForTest(status Status) {
	e.Status = status
}
