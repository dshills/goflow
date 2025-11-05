package execution

import (
	"fmt"
	"log"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/storage"
)

// Logger handles execution logging to persistent storage.
type Logger struct {
	repository *storage.SQLiteExecutionRepository
}

// NewLogger creates a new execution logger.
func NewLogger(repo *storage.SQLiteExecutionRepository) *Logger {
	return &Logger{
		repository: repo,
	}
}

// LogExecutionStart logs the start of a workflow execution.
func (l *Logger) LogExecutionStart(exec *execution.Execution) {
	if l.repository == nil {
		// No repository configured, skip logging
		return
	}

	// Create a minimal execution record for the start event
	if err := l.repository.Save(exec); err != nil {
		// Log error but don't fail execution
		log.Printf("Warning: failed to log execution start: %v", err)
	}
}

// LogExecutionComplete logs the completion of a workflow execution.
func (l *Logger) LogExecutionComplete(exec *execution.Execution) {
	if l.repository == nil {
		// No repository configured, skip logging
		return
	}

	// Save final execution state
	if err := l.repository.Save(exec); err != nil {
		// Log error but don't fail execution
		log.Printf("Warning: failed to log execution completion: %v", err)
	}
}

// LogNodeExecution logs a node execution record.
func (l *Logger) LogNodeExecution(nodeExec *execution.NodeExecution) {
	if l.repository == nil {
		// No repository configured, skip logging
		return
	}

	// Node executions are saved as part of the execution entity
	// This method is primarily for real-time monitoring/streaming
	// For now, we'll just log to stdout for debugging
	log.Printf("Node %s: %s -> %s (duration: %v)",
		nodeExec.NodeID,
		nodeExec.NodeType,
		nodeExec.Status,
		nodeExec.Duration(),
	)
}

// LogVariableChange logs a variable value change.
func (l *Logger) LogVariableChange(snapshot *execution.VariableSnapshot) {
	if l.repository == nil {
		// No repository configured, skip logging
		return
	}

	// Variable snapshots are saved as part of the execution context
	// This method is for real-time monitoring
	log.Printf("Variable changed: %s = %v (node: %s)",
		snapshot.VariableName,
		snapshot.NewValue,
		snapshot.NodeExecutionID,
	)
}

// GetExecutionLogs retrieves execution logs from storage.
func (l *Logger) GetExecutionLogs(executionID types.ExecutionID) (*execution.Execution, error) {
	if l.repository == nil {
		return nil, fmt.Errorf("no repository configured")
	}

	return l.repository.Load(executionID)
}

// ListExecutions retrieves all execution records.
// Note: This requires iterating through all workflows. For now, return empty list.
func (l *Logger) ListExecutions() ([]*execution.Execution, error) {
	if l.repository == nil {
		return nil, fmt.Errorf("no repository configured")
	}

	// TODO: Implement when we have a ListAll method or iterate through workflows
	return []*execution.Execution{}, nil
}

// ListExecutionsByWorkflow retrieves all executions for a specific workflow.
func (l *Logger) ListExecutionsByWorkflow(workflowID types.WorkflowID) ([]*execution.Execution, error) {
	if l.repository == nil {
		return nil, fmt.Errorf("no repository configured")
	}

	return l.repository.ListByWorkflow(workflowID)
}

// DeleteExecution removes an execution record from storage.
func (l *Logger) DeleteExecution(executionID types.ExecutionID) error {
	if l.repository == nil {
		return fmt.Errorf("no repository configured")
	}

	return l.repository.Delete(executionID)
}
