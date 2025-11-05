package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLiteExecutionRepository implements ExecutionRepository using SQLite storage.
// Provides persistent storage for execution history with efficient querying.
type SQLiteExecutionRepository struct {
	db *sql.DB
}

// NewSQLiteExecutionRepository creates a new SQLite-based execution repository.
// Database location: ~/.goflow/goflow.db
func NewSQLiteExecutionRepository() (*SQLiteExecutionRepository, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".goflow")
	dbPath := filepath.Join(baseDir, "goflow.db")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .goflow directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)

	// Initialize database schema
	if err := InitializeDatabase(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &SQLiteExecutionRepository{db: db}, nil
}

// NewSQLiteExecutionRepositoryWithPath creates a repository with a custom database path.
// Useful for testing.
func NewSQLiteExecutionRepositoryWithPath(dbPath string) (*SQLiteExecutionRepository, error) {
	// Create directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Initialize database schema
	if err := InitializeDatabase(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &SQLiteExecutionRepository{db: db}, nil
}

// Close closes the database connection.
func (r *SQLiteExecutionRepository) Close() error {
	return r.db.Close()
}

// Save persists an execution to the database.
// Updates the execution if it already exists (based on ID).
func (r *SQLiteExecutionRepository) Save(exec *execution.Execution) error {
	if exec == nil {
		return fmt.Errorf("cannot save nil execution")
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Serialize error context and return value as JSON
	var errorType, errorMessage, errorNodeID, errorContext sql.NullString
	if exec.Error != nil {
		errorType.Valid = true
		errorType.String = string(exec.Error.Type)
		errorMessage.Valid = true
		errorMessage.String = exec.Error.Message
		if exec.Error.NodeID != "" {
			errorNodeID.Valid = true
			errorNodeID.String = string(exec.Error.NodeID)
		}
		if len(exec.Error.Context) > 0 {
			ctxData, err := json.Marshal(exec.Error.Context)
			if err == nil {
				errorContext.Valid = true
				errorContext.String = string(ctxData)
			}
		}
	}

	var returnValue sql.NullString
	if exec.ReturnValue != nil {
		retData, err := json.Marshal(exec.ReturnValue)
		if err == nil {
			returnValue.Valid = true
			returnValue.String = string(retData)
		}
	}

	var completedAt sql.NullTime
	if !exec.CompletedAt.IsZero() {
		completedAt.Valid = true
		completedAt.Time = exec.CompletedAt
	}

	// Upsert execution record
	query := `
		INSERT INTO executions (
			id, workflow_id, workflow_version, status, started_at, completed_at,
			error_type, error_message, error_node_id, error_context, return_value
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			completed_at = excluded.completed_at,
			error_type = excluded.error_type,
			error_message = excluded.error_message,
			error_node_id = excluded.error_node_id,
			error_context = excluded.error_context,
			return_value = excluded.return_value
	`

	_, err = tx.Exec(query,
		exec.ID.String(),
		string(exec.WorkflowID),
		exec.WorkflowVersion,
		string(exec.Status),
		exec.StartedAt,
		completedAt,
		errorType,
		errorMessage,
		errorNodeID,
		errorContext,
		returnValue,
	)
	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Load retrieves an execution by its ID.
func (r *SQLiteExecutionRepository) Load(id types.ExecutionID) (*execution.Execution, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("execution ID cannot be empty")
	}

	query := `
		SELECT id, workflow_id, workflow_version, status, started_at, completed_at,
		       error_type, error_message, error_node_id, error_context, return_value
		FROM executions
		WHERE id = ?
	`

	var exec execution.Execution
	var completedAt sql.NullTime
	var errorType, errorMessage, errorNodeID, errorContext, returnValue sql.NullString

	err := r.db.QueryRow(query, id.String()).Scan(
		&exec.ID,
		&exec.WorkflowID,
		&exec.WorkflowVersion,
		&exec.Status,
		&exec.StartedAt,
		&completedAt,
		&errorType,
		&errorMessage,
		&errorNodeID,
		&errorContext,
		&returnValue,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load execution: %w", err)
	}

	// Deserialize optional fields
	if completedAt.Valid {
		exec.CompletedAt = completedAt.Time
	}

	if errorType.Valid && errorMessage.Valid {
		exec.Error = &execution.ExecutionError{
			Type:    execution.ErrorType(errorType.String),
			Message: errorMessage.String,
		}
		if errorNodeID.Valid {
			exec.Error.NodeID = types.NodeID(errorNodeID.String)
		}
		if errorContext.Valid {
			var ctx map[string]interface{}
			if err := json.Unmarshal([]byte(errorContext.String), &ctx); err == nil {
				exec.Error.Context = ctx
			}
		}
	}

	if returnValue.Valid {
		var ret interface{}
		if err := json.Unmarshal([]byte(returnValue.String), &ret); err == nil {
			exec.ReturnValue = ret
		}
	}

	// Load node executions
	nodeExecs, err := r.loadNodeExecutions(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load node executions: %w", err)
	}
	exec.NodeExecutions = nodeExecs

	// Initialize context (will be populated by execution engine)
	exec.Context, _ = execution.NewExecutionContext(nil)

	return &exec, nil
}

// loadNodeExecutions retrieves all node executions for an execution.
func (r *SQLiteExecutionRepository) loadNodeExecutions(execID types.ExecutionID) ([]*execution.NodeExecution, error) {
	query := `
		SELECT id, execution_id, node_id, node_type, status, started_at, completed_at,
		       inputs, outputs, error_type, error_message, error_context, retry_count
		FROM node_executions
		WHERE execution_id = ?
		ORDER BY started_at
	`

	rows, err := r.db.Query(query, execID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query node executions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	nodeExecs := make([]*execution.NodeExecution, 0)

	for rows.Next() {
		var ne execution.NodeExecution
		var completedAt sql.NullTime
		var inputs, outputs, errorType, errorMessage, errorContext sql.NullString

		err := rows.Scan(
			&ne.ID,
			&ne.ExecutionID,
			&ne.NodeID,
			&ne.NodeType,
			&ne.Status,
			&ne.StartedAt,
			&completedAt,
			&inputs,
			&outputs,
			&errorType,
			&errorMessage,
			&errorContext,
			&ne.RetryCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node execution: %w", err)
		}

		if completedAt.Valid {
			ne.CompletedAt = completedAt.Time
		}

		// Deserialize JSON fields
		if inputs.Valid {
			var inp map[string]interface{}
			if err := json.Unmarshal([]byte(inputs.String), &inp); err == nil {
				ne.Inputs = inp
			}
		}
		if outputs.Valid {
			var out map[string]interface{}
			if err := json.Unmarshal([]byte(outputs.String), &out); err == nil {
				ne.Outputs = out
			}
		}

		if errorType.Valid && errorMessage.Valid {
			ne.Error = &execution.NodeError{
				Type:    execution.ErrorType(errorType.String),
				Message: errorMessage.String,
			}
			if errorContext.Valid {
				var ctx map[string]interface{}
				if err := json.Unmarshal([]byte(errorContext.String), &ctx); err == nil {
					ne.Error.Context = ctx
				}
			}
		}

		nodeExecs = append(nodeExecs, &ne)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating node executions: %w", err)
	}

	return nodeExecs, nil
}

// ListByWorkflow returns all executions for a specific workflow.
// Results are ordered by StartedAt descending (most recent first).
func (r *SQLiteExecutionRepository) ListByWorkflow(workflowID types.WorkflowID) ([]*execution.Execution, error) {
	query := `
		SELECT id, workflow_id, workflow_version, status, started_at, completed_at,
		       error_type, error_message, error_node_id, error_context, return_value
		FROM executions
		WHERE workflow_id = ?
		ORDER BY started_at DESC
	`

	return r.queryExecutions(query, string(workflowID))
}

// ListByStatus returns all executions with a specific status.
func (r *SQLiteExecutionRepository) ListByStatus(status execution.Status) ([]*execution.Execution, error) {
	query := `
		SELECT id, workflow_id, workflow_version, status, started_at, completed_at,
		       error_type, error_message, error_node_id, error_context, return_value
		FROM executions
		WHERE status = ?
		ORDER BY started_at DESC
	`

	return r.queryExecutions(query, string(status))
}

// queryExecutions is a helper function to execute queries that return multiple executions.
func (r *SQLiteExecutionRepository) queryExecutions(query string, args ...interface{}) ([]*execution.Execution, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	executions := make([]*execution.Execution, 0)

	for rows.Next() {
		var exec execution.Execution
		var completedAt sql.NullTime
		var errorType, errorMessage, errorNodeID, errorContext, returnValue sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.WorkflowID,
			&exec.WorkflowVersion,
			&exec.Status,
			&exec.StartedAt,
			&completedAt,
			&errorType,
			&errorMessage,
			&errorNodeID,
			&errorContext,
			&returnValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		// Deserialize optional fields
		if completedAt.Valid {
			exec.CompletedAt = completedAt.Time
		}

		if errorType.Valid && errorMessage.Valid {
			exec.Error = &execution.ExecutionError{
				Type:    execution.ErrorType(errorType.String),
				Message: errorMessage.String,
			}
			if errorNodeID.Valid {
				exec.Error.NodeID = types.NodeID(errorNodeID.String)
			}
			if errorContext.Valid {
				var ctx map[string]interface{}
				if err := json.Unmarshal([]byte(errorContext.String), &ctx); err == nil {
					exec.Error.Context = ctx
				}
			}
		}

		if returnValue.Valid {
			var ret interface{}
			if err := json.Unmarshal([]byte(returnValue.String), &ret); err == nil {
				exec.ReturnValue = ret
			}
		}

		// Initialize context
		exec.Context, _ = execution.NewExecutionContext(nil)

		// Note: Node executions are not loaded in list operations for performance
		// Use Load() to get full execution details with node executions

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating executions: %w", err)
	}

	return executions, nil
}

// Delete removes an execution and all its related data from storage.
func (r *SQLiteExecutionRepository) Delete(id types.ExecutionID) error {
	if id.IsZero() {
		return fmt.Errorf("execution ID cannot be empty")
	}

	// Foreign key cascade will handle node_executions and variable_snapshots
	result, err := r.db.Exec("DELETE FROM executions WHERE id = ?", id.String())
	if err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}

// SaveNodeExecution persists a node execution record.
func (r *SQLiteExecutionRepository) SaveNodeExecution(nodeExec *execution.NodeExecution) error {
	if nodeExec == nil {
		return fmt.Errorf("cannot save nil node execution")
	}

	// Serialize inputs, outputs, and error context as JSON
	var inputs, outputs sql.NullString
	if len(nodeExec.Inputs) > 0 {
		inpData, err := json.Marshal(nodeExec.Inputs)
		if err == nil {
			inputs.Valid = true
			inputs.String = string(inpData)
		}
	}
	if len(nodeExec.Outputs) > 0 {
		outData, err := json.Marshal(nodeExec.Outputs)
		if err == nil {
			outputs.Valid = true
			outputs.String = string(outData)
		}
	}

	var errorType, errorMessage, errorContext sql.NullString
	if nodeExec.Error != nil {
		errorType.Valid = true
		errorType.String = string(nodeExec.Error.Type)
		errorMessage.Valid = true
		errorMessage.String = nodeExec.Error.Message
		if len(nodeExec.Error.Context) > 0 {
			ctxData, err := json.Marshal(nodeExec.Error.Context)
			if err == nil {
				errorContext.Valid = true
				errorContext.String = string(ctxData)
			}
		}
	}

	var completedAt sql.NullTime
	if !nodeExec.CompletedAt.IsZero() {
		completedAt.Valid = true
		completedAt.Time = nodeExec.CompletedAt
	}

	query := `
		INSERT INTO node_executions (
			id, execution_id, node_id, node_type, status, started_at, completed_at,
			inputs, outputs, error_type, error_message, error_context, retry_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			completed_at = excluded.completed_at,
			outputs = excluded.outputs,
			error_type = excluded.error_type,
			error_message = excluded.error_message,
			error_context = excluded.error_context,
			retry_count = excluded.retry_count
	`

	_, err := r.db.Exec(query,
		string(nodeExec.ID),
		nodeExec.ExecutionID.String(),
		string(nodeExec.NodeID),
		nodeExec.NodeType,
		string(nodeExec.Status),
		nodeExec.StartedAt,
		completedAt,
		inputs,
		outputs,
		errorType,
		errorMessage,
		errorContext,
		nodeExec.RetryCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save node execution: %w", err)
	}

	return nil
}

// SaveVariableSnapshot persists a variable snapshot to the audit trail.
func (r *SQLiteExecutionRepository) SaveVariableSnapshot(snapshot *execution.VariableSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("cannot save nil variable snapshot")
	}

	// Serialize old and new values as JSON
	var oldValue, newValue sql.NullString
	if snapshot.OldValue != nil {
		oldData, err := json.Marshal(snapshot.OldValue)
		if err == nil {
			oldValue.Valid = true
			oldValue.String = string(oldData)
		}
	}
	if snapshot.NewValue != nil {
		newData, err := json.Marshal(snapshot.NewValue)
		if err == nil {
			newValue.Valid = true
			newValue.String = string(newData)
		}
	}

	var nodeExecID sql.NullString
	if snapshot.NodeExecutionID != "" {
		nodeExecID.Valid = true
		nodeExecID.String = string(snapshot.NodeExecutionID)
	}

	// Extract execution ID from context (would need to be passed or stored in snapshot)
	// For now, we'll need to enhance the snapshot to include execution_id
	// This is a placeholder - actual implementation would need the execution_id
	query := `
		INSERT INTO variable_snapshots (
			execution_id, node_execution_id, variable_name, old_value, new_value, timestamp
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	// Note: This needs execution_id which isn't currently in VariableSnapshot
	// We'll address this in a future iteration when the execution context is better defined
	_ = query

	return fmt.Errorf("SaveVariableSnapshot not fully implemented - needs execution_id in snapshot")
}
