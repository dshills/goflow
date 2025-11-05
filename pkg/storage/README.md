# GoFlow Storage Layer

The storage layer provides persistence for workflows, execution history, and credentials in GoFlow. It follows the Repository pattern and Domain-Driven Design principles, with separate implementations for each aggregate root.

## Overview

GoFlow uses a multi-backend storage strategy:

1. **Workflows**: Filesystem (YAML files) - Human-readable, version control friendly
2. **Executions**: SQLite (pure Go) - Efficient querying, ACID transactions
3. **Credentials**: System keyring - OS-provided secure storage

## Storage Locations

All GoFlow data is stored under `~/.goflow/`:

```
~/.goflow/
├── workflows/           # Workflow YAML files
│   └── <workflow-id>.yaml
└── goflow.db           # SQLite database for execution history
```

Credentials are stored in the system keyring:
- **macOS**: Keychain
- **Windows**: Credential Manager
- **Linux**: Secret Service (GNOME Keyring, KWallet)

## Components

### Filesystem Workflow Repository

Implements `workflow.WorkflowRepository` interface using YAML files.

**Features**:
- Atomic writes using temp file + rename
- Human-readable YAML format
- Version control friendly
- Easy sharing and backup

**Usage**:
```go
repo, err := storage.NewFilesystemWorkflowRepository()
if err != nil {
    log.Fatal(err)
}

// Save workflow
wf, _ := workflow.NewWorkflow("my-workflow", "Description")
repo.Save(wf)

// Load workflow
loaded, err := repo.Load(workflow.WorkflowID(wf.ID))

// List all workflows
workflows, err := repo.List()

// Delete workflow
repo.Delete(workflow.WorkflowID(wf.ID))
```

**Custom Path** (useful for testing):
```go
repo, err := storage.NewFilesystemWorkflowRepositoryWithPath("/custom/path")
```

### SQLite Execution Repository

Implements `execution.ExecutionRepository` interface using SQLite.

**Database Schema**:
- `executions`: Workflow execution lifecycle tracking
- `node_executions`: Individual node execution details
- `variable_snapshots`: Append-only audit trail of variable changes
- `migrations`: Schema version tracking for future updates

**Features**:
- Pure Go SQLite (no CGO dependency)
- Efficient indexes for common queries
- Foreign key constraints with cascade delete
- ACID transaction support
- Automatic schema migration

**Usage**:
```go
repo, err := storage.NewSQLiteExecutionRepository()
if err != nil {
    log.Fatal(err)
}
defer repo.Close()

// Create and save execution
exec, _ := execution.NewExecution(workflowID, "1.0.0", inputs)
exec.Start()
repo.Save(exec)

// Load execution
loaded, err := repo.Load(exec.ID)

// Query executions
executions, err := repo.ListByWorkflow(workflowID)
executions, err := repo.ListByStatus(execution.StatusRunning)

// Save node execution
nodeExec := execution.NewNodeExecution(exec.ID, nodeID, "mcp_tool")
repo.SaveNodeExecution(nodeExec)

// Delete execution (cascades to node_executions and variable_snapshots)
repo.Delete(exec.ID)
```

**Custom Database Path** (useful for testing):
```go
repo, err := storage.NewSQLiteExecutionRepositoryWithPath("/custom/path/test.db")
```

### Keyring Credential Store

Implements `CredentialStore` interface using the system keyring.

**Features**:
- OS-provided secure storage
- Automatic encryption at rest
- Per-user isolation
- No credentials in workflow files
- Support for structured credentials (JSON serialization)

**Usage**:
```go
store := storage.NewKeyringCredentialStore()

// Store simple credential
store.Set("api-key", "secret-value")

// Retrieve credential
value, err := store.Get("api-key")

// List credential keys (not values)
keys, err := store.List()

// Delete credential
store.Delete("api-key")

// Store structured credential
serverCreds := map[string]interface{}{
    "host":    "example.com",
    "api_key": "secret-123",
    "port":    5432,
}
store.SetStructured("mcp-server-1", serverCreds)

// Retrieve structured credential
var retrieved map[string]interface{}
store.GetStructured("mcp-server-1", &retrieved)
```

**Credential Index**:
The keyring store maintains an internal index of credential keys (stored as `__goflow_index__` in the keyring) to support the `List()` operation, since keyrings don't provide native enumeration.

## Database Schema

### executions Table
```sql
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
);
```

### node_executions Table
```sql
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
);
```

### variable_snapshots Table
```sql
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
);
```

## Performance Characteristics

### Filesystem Repository
- **Save**: O(1) - Single file write
- **Load**: O(1) - Single file read
- **List**: O(n) - Reads all workflow files
- **Delete**: O(1) - Single file delete

### SQLite Repository
- **Save Execution**: O(1) - Single row upsert
- **Load Execution**: O(1) - Indexed lookup by ID
- **ListByWorkflow**: O(log n) - Indexed query on workflow_id
- **ListByStatus**: O(log n) - Indexed query on status
- **SaveNodeExecution**: O(1) - Single row upsert

All common queries use indexes for efficient retrieval.

### Keyring Store
- **Set/Get/Delete**: O(1) - Direct keyring operations
- **List**: O(1) - Read index from keyring

## Concurrency

### Filesystem Repository
- **Atomic writes**: Uses temp file + rename pattern
- **Concurrent reads**: Safe (read-only operations)
- **Concurrent writes**: OS-level file locking prevents corruption
- **Recommendation**: Use external locking for concurrent workflow editing

### SQLite Repository
- **Connection pool**: Single connection (optimal for SQLite)
- **Transactions**: Used for multi-row operations
- **Concurrent reads**: Supported by SQLite
- **Concurrent writes**: Serialized by SQLite (write-ahead logging could be enabled for better concurrency)

### Keyring Store
- **Thread-safe**: OS keyring handles concurrency
- **Process-safe**: System keyring is process-safe

## Migration Strategy

The SQLite schema includes a `migrations` table to track version:

```sql
CREATE TABLE migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version INTEGER NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Current version: **1**

Future schema changes will be applied incrementally:
1. Check current version from `migrations` table
2. Apply unapplied migrations in order
3. Record new version in `migrations` table

## Error Handling

All repository operations return descriptive errors:

- **Not Found**: Resource doesn't exist
- **Already Exists**: Duplicate ID or name
- **Permission Denied**: File system or keyring access denied
- **Invalid Data**: Corrupt YAML or JSON
- **Database Error**: SQLite constraint violations or connection issues

Example:
```go
exec, err := repo.Load(execID)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle not found
    } else {
        // Handle other errors
    }
}
```

## Testing

The storage layer can be tested with custom paths:

```go
// Temporary test directories
testDir := filepath.Join(os.TempDir(), "goflow-test")
defer os.RemoveAll(testDir)

// Custom workflow repository
workflowRepo, _ := storage.NewFilesystemWorkflowRepositoryWithPath(testDir)

// Custom execution repository
dbPath := filepath.Join(testDir, "test.db")
execRepo, _ := storage.NewSQLiteExecutionRepositoryWithPath(dbPath)
defer execRepo.Close()

// Keyring store (uses real system keyring - clean up test credentials)
credStore := storage.NewKeyringCredentialStore()
defer credStore.Delete("test-key")
```

## Security Considerations

### Workflows
- Stored as plain YAML files
- World-readable by default (0644)
- No sensitive data should be in workflow files
- Credentials referenced by ID only

### Executions
- SQLite database with 0644 permissions
- No credentials stored in execution data
- Sensitive inputs/outputs should be avoided or encrypted

### Credentials
- Never stored in workflow files or database
- Always use system keyring
- Per-user isolation
- OS-provided encryption at rest

## Examples

See `/cmd/storage-test/` for comprehensive verification tests.
See `/cmd/storage-example/` for real-world usage patterns.

## Dependencies

- `gopkg.in/yaml.v3` - YAML serialization
- `modernc.org/sqlite` - Pure Go SQLite (no CGO)
- `github.com/zalando/go-keyring` - Cross-platform keyring access

## Future Enhancements

Potential improvements for future versions:

1. **Workflow Repository**:
   - File watching for external changes
   - Workflow versioning and history
   - Template workflow storage

2. **Execution Repository**:
   - Execution retention policies (auto-cleanup old executions)
   - Execution statistics and aggregations
   - Export to JSON/CSV

3. **Credential Store**:
   - Credential expiration tracking
   - Credential rotation support
   - Multi-tenant credential isolation

4. **Performance**:
   - SQLite write-ahead logging (WAL mode)
   - Execution result caching
   - Lazy loading of node executions
