package storage

import (
	"database/sql"
	"fmt"
)

// MigrationVersion tracks the current database schema version.
const MigrationVersion = 1

// InitializeDatabase creates the SQLite database schema for execution history.
// This includes migration version tracking to support future schema updates.
func InitializeDatabase(db *sql.DB) error {
	// Create migrations table to track schema version
	migrationsTable := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER NOT NULL UNIQUE,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(migrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check current version
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to check migration version: %w", err)
	}

	// Apply migrations
	if currentVersion < 1 {
		if err := applyMigration1(db); err != nil {
			return fmt.Errorf("failed to apply migration 1: %w", err)
		}
	}

	return nil
}

// applyMigration1 creates the initial database schema.
func applyMigration1(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Executions table - tracks workflow execution lifecycle
	executionsTable := `
	CREATE TABLE executions (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		workflow_version TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		error_type TEXT,
		error_message TEXT,
		error_node_id TEXT,
		error_context TEXT,
		return_value TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := tx.Exec(executionsTable); err != nil {
		return fmt.Errorf("failed to create executions table: %w", err)
	}

	// Indexes for common queries
	executionsIndexes := []string{
		// Primary indexes for filtering
		"CREATE INDEX idx_executions_workflow_id ON executions(workflow_id, started_at DESC);",
		"CREATE INDEX idx_executions_status ON executions(status, started_at DESC);",
		"CREATE INDEX idx_executions_started_at ON executions(started_at DESC);",

		// Composite index for common combined queries (workflow + status)
		"CREATE INDEX idx_executions_workflow_status ON executions(workflow_id, status, started_at DESC);",
	}

	for _, idx := range executionsIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create execution index: %w", err)
		}
	}

	// Node executions table - tracks individual node execution details
	nodeExecutionsTable := `
	CREATE TABLE node_executions (
		id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		node_id TEXT NOT NULL,
		node_type TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		inputs TEXT,
		outputs TEXT,
		error_type TEXT,
		error_message TEXT,
		error_context TEXT,
		retry_count INTEGER DEFAULT 0,
		FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE CASCADE
	);`

	if _, err := tx.Exec(nodeExecutionsTable); err != nil {
		return fmt.Errorf("failed to create node_executions table: %w", err)
	}

	// Indexes for node executions
	nodeExecutionsIndexes := []string{
		"CREATE INDEX idx_node_executions_execution_id ON node_executions(execution_id, started_at);",
		"CREATE INDEX idx_node_executions_status ON node_executions(status);",
	}

	for _, idx := range nodeExecutionsIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create node execution index: %w", err)
		}
	}

	// Variable snapshots table - append-only audit trail of variable changes
	variableSnapshotsTable := `
	CREATE TABLE variable_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		execution_id TEXT NOT NULL,
		node_execution_id TEXT,
		variable_name TEXT NOT NULL,
		old_value TEXT,
		new_value TEXT,
		timestamp TIMESTAMP NOT NULL,
		FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE CASCADE,
		FOREIGN KEY (node_execution_id) REFERENCES node_executions(id) ON DELETE SET NULL
	);`

	if _, err := tx.Exec(variableSnapshotsTable); err != nil {
		return fmt.Errorf("failed to create variable_snapshots table: %w", err)
	}

	// Indexes for variable snapshots
	variableSnapshotsIndexes := []string{
		"CREATE INDEX idx_variable_snapshots_execution_id ON variable_snapshots(execution_id, timestamp);",
		"CREATE INDEX idx_variable_snapshots_node_execution_id ON variable_snapshots(node_execution_id);",
	}

	for _, idx := range variableSnapshotsIndexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create variable snapshot index: %w", err)
		}
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO migrations (version) VALUES (?)", 1); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}
