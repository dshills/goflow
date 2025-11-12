# Data Model: Code Review Remediation

**Feature**: 002-pr-review-remediation
**Date**: 2025-11-12

## Overview

This feature primarily enhances existing domain aggregates rather than introducing new entities. The data model focuses on validation structures, error context enhancements, and configuration entities for security policies.

---

## 1. Existing Aggregates (Enhanced)

### 1.1 Workflow Execution Aggregate (Enhanced)

**Location**: `pkg/execution/`

**Existing Root**: `Execution`

**Enhancements**:

#### ExecutionContext (Enhanced)

Existing context enhanced with timeout support:

```go
type ExecutionContext struct {
    // Existing fields
    WorkflowID   string
    ExecutionID  string
    StartTime    time.Time
    Status       ExecutionStatus
    Variables    map[string]interface{}
    NodeTrace    []NodeExecution

    // ENHANCED: Add timeout context
    ctx          context.Context    // New: context with timeout
    cancel       context.CancelFunc // New: cancellation function

    // ENHANCED: Add timeout tracking
    TimeoutDuration time.Duration   // New: configured timeout
    TimedOut        bool             // New: whether execution timed out
    TimeoutNode     string           // New: node ID that was executing when timeout occurred
}
```

**Validation Rules**:
- `ctx` must be created with `context.WithTimeout()` or `context.WithDeadline()`
- `TimeoutDuration` must be > 0 if timeout is enabled (or 0 for no timeout)
- `TimeoutNode` only set if `TimedOut == true`
- Context must be passed to all node executions

**State Transitions**:
- `Running` → `TimedOut` (when context deadline exceeded)
- `TimedOut` is terminal state (cannot resume)

---

### 1.2 MCP Server Registry Aggregate (Enhanced)

**Location**: `pkg/mcp/`

**Existing Root**: `ConnectionPool`

**Enhancements**:

#### ConnectionPool (API Fixed)

API signatures corrected for consistency:

```go
type ConnectionPool struct {
    mu          sync.RWMutex
    connections map[string]*PooledConnection
    servers     map[string]*MCPServer
    maxIdle     time.Duration
    cleanupTick *time.Ticker

    // ENHANCED: Shutdown coordination
    closing     chan struct{}       // New: signals shutdown in progress
    wg          sync.WaitGroup      // New: tracks active operations
}

// FIXED: API signature consistency
func (p *ConnectionPool) Get(ctx context.Context, serverID string) (*PooledConnection, error)

// FIXED: API signature consistency
func (p *ConnectionPool) Release(serverID string) error

// ENHANCED: Graceful shutdown
func (p *ConnectionPool) Close() error {
    // Coordinate shutdown of all connections
    // Wait for active operations with timeout
    // Force-close after grace period
}
```

**Validation Rules**:
- `Get()` must be called with valid `serverID` that exists in `servers` map
- `Release()` must be called with `serverID` of currently acquired connection
- `Close()` must wait up to 30 seconds for graceful shutdown before force-closing
- `Get()` after `Close()` returns error

**Lifecycle States**:
- `Active`: Normal operation, can acquire/release connections
- `Closing`: Shutdown initiated, no new acquisitions, existing operations complete
- `Closed`: All connections closed, all operations must error

#### PooledConnection (Enhanced)

```go
type PooledConnection struct {
    // Existing fields
    ServerID     string
    Client       protocol.Client
    AcquiredAt   time.Time
    LastUsedAt   time.Time

    // ENHANCED: Lifecycle tracking
    closed       bool                 // New: whether connection is closed
    closeErr     error                // New: error from close operation
    refCount     int32                // New: reference count for leak detection
}
```

**Validation Rules**:
- `refCount` must be 1 when acquired, 0 when released
- `closed` must be false when returned from `Get()`
- Setting `closed = true` is irreversible (terminal state)

---

## 2. New Entities

### 2.1 File Path Validator (New)

**Package**: `pkg/validation`

**Purpose**: Secure validation of user-provided file paths to prevent directory traversal attacks.

#### PathValidator

```go
type PathValidator struct {
    basePath     string          // Absolute path to allowed directory
    resolvedBase string          // Resolved base path (symlinks resolved)
    maxPathLen   int             // Maximum path length (default: 1024)

    // Statistics (for monitoring)
    validations  uint64          // Total validation attempts
    rejections   uint64          // Total rejections
    mu           sync.RWMutex    // Protects statistics
}
```

**Fields**:
- `basePath`: User-configured allowed directory (must be absolute)
- `resolvedBase`: Result of `filepath.EvalSymlinks(basePath)` cached at initialization
- `maxPathLen`: Maximum allowed path length in bytes (prevents DOS)
- `validations`, `rejections`: Counters for monitoring (not persisted)

**Validation Rules**:
- `basePath` must be absolute path (starts with `/` on Unix, drive letter on Windows)
- `basePath` must exist and be a directory
- `maxPathLen` must be >= 256 and <= 4096
- `resolvedBase` computed once at construction, immutable

**Methods**:
```go
// NewPathValidator creates a validator for the given base directory
func NewPathValidator(basePath string) (*PathValidator, error)

// Validate checks if userPath is safe to access within base directory
// Returns validated absolute path or error
func (v *PathValidator) Validate(userPath string) (string, error)

// Stats returns validation statistics
func (v *PathValidator) Stats() (validations, rejections uint64)
```

#### ValidationError (New)

```go
type ValidationError struct {
    UserPath     string          // Original user input
    Reason       string          // Human-readable reason
    ResolvedPath string          // Resolved path (if resolution succeeded)
    Timestamp    time.Time       // When error occurred
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("path validation failed: %s (input: %s)", e.Reason, e.UserPath)
}
```

**Purpose**: Structured error for security logging and audit trails.

**Validation Rules**:
- `UserPath` must be non-empty (the invalid input that was rejected)
- `Reason` must be non-empty (one of predefined reasons)
- `Timestamp` set to `time.Now()` at creation

**Predefined Reasons**:
- `"path is empty"`
- `"path escapes allowed directory"`
- `"resolved path escapes base directory"`
- `"Windows reserved name not allowed"`
- `"path is not relative to base"`
- `"cannot resolve path"`

---

### 2.2 Error Context (New)

**Package**: `pkg/execution`

**Purpose**: Enhanced error information for debugging and observability.

#### ErrorContext

```go
type ErrorContext struct {
    Operation    string                 // What operation was being performed
    WorkflowID   string                 // Which workflow
    NodeID       string                 // Which node (if applicable)
    Timestamp    time.Time              // When error occurred
    Attributes   map[string]interface{} // Additional context key-value pairs
    Cause        error                  // Underlying error (wrapped)
}

func (e *ErrorContext) Error() string {
    return fmt.Sprintf("[%s] %s: workflow=%s node=%s: %v",
        e.Timestamp.Format(time.RFC3339),
        e.Operation,
        e.WorkflowID,
        e.NodeID,
        e.Cause)
}

func (e *ErrorContext) Unwrap() error {
    return e.Cause
}
```

**Fields**:
- `Operation`: Human-readable operation name (e.g., "executing MCP tool", "validating workflow")
- `WorkflowID`: ID of workflow being executed
- `NodeID`: ID of node being executed (empty if not node-specific)
- `Timestamp`: When error occurred
- `Attributes`: Arbitrary key-value pairs for additional context
- `Cause`: The underlying error being wrapped

**Validation Rules**:
- `Operation` must be non-empty
- `Timestamp` set to `time.Now()` at creation
- `Cause` must be non-nil (no wrapping nil errors)
- `Attributes` can be nil (treated as empty map)

**Usage Pattern**:
```go
if err != nil {
    return &ErrorContext{
        Operation:  "executing workflow node",
        WorkflowID: exec.WorkflowID,
        NodeID:     node.ID,
        Timestamp:  time.Now(),
        Attributes: map[string]interface{}{
            "nodeType":  node.Type,
            "serverID":  node.ServerID,
        },
        Cause:      err,
    }
}
```

---

### 2.3 Terminal Input Event (Enhanced)

**Package**: `pkg/tui`

**Existing**: `KeyEvent`

**No changes required** - existing `KeyEvent` structure is sufficient. Terminal input handling changes are purely implementation (goroutine pattern), not data model changes.

---

### 2.4 Test Server Configuration (New)

**Package**: `internal/testutil/testserver`

**Purpose**: Configuration for test server security policies.

#### ServerConfig

```go
type ServerConfig struct {
    // File operation security
    AllowedDirectory string          // Base directory for file operations
    MaxFileSize      int64            // Maximum file size for read/write (bytes)

    // Logging
    LogSecurityEvents bool            // Whether to log security violations
    LogFilePath       string          // Path to security audit log (empty = stderr)

    // Performance
    ReadTimeout       time.Duration   // Timeout for file read operations
    WriteTimeout      time.Duration   // Timeout for file write operations
}

// Default returns default secure configuration
func DefaultConfig() *ServerConfig {
    return &ServerConfig{
        AllowedDirectory:  os.TempDir(),
        MaxFileSize:       10 * 1024 * 1024, // 10MB
        LogSecurityEvents: true,
        LogFilePath:       "",                // stderr
        ReadTimeout:       5 * time.Second,
        WriteTimeout:      5 * time.Second,
    }
}
```

**Validation Rules**:
- `AllowedDirectory` must be absolute path and exist
- `MaxFileSize` must be > 0 and <= 100MB (prevent DOS)
- `ReadTimeout`, `WriteTimeout` must be > 0

**Configuration Sources** (priority order):
1. Environment variables:
   - `GOFLOW_TESTSERVER_ALLOWED_DIR`
   - `GOFLOW_TESTSERVER_MAX_FILE_SIZE`
   - `GOFLOW_TESTSERVER_LOG_SECURITY`
2. Configuration file: `.goflow/testserver.yaml` (if exists)
3. Defaults: `DefaultConfig()`

---

### 2.5 Keyboard Binding (Type Fixed)

**Package**: `pkg/tui`

**Existing**: `KeyBinding` and related types

**Fix**: Ensure `Mode` type is consistently used throughout binding system.

```go
type Mode string

const (
    ModeNormal   Mode = "normal"
    ModeInsert   Mode = "insert"
    ModeVisual   Mode = "visual"
    ModeCommand  Mode = "command"
    ModeGlobal   Mode = "global"    // FIXED: Use Mode type, not string "global"
)

type KeyBindingRegistry struct {
    bindings map[Mode]map[string]KeyBinding  // FIXED: Mode type for keys
    mu       sync.RWMutex
}

// FIXED: Return type uses Mode consistently
func (r *KeyBindingRegistry) GetAllBindings() map[Mode]map[string]KeyBinding {
    // ...
}
```

**Validation Rules**:
- `Mode` must be one of the defined constants
- Global bindings stored under `ModeGlobal` (not string `"global"`)
- Type-safe at compile time

---

## 3. Relationships

### 3.1 Execution → ErrorContext

- **Type**: Composition (has-a)
- **Cardinality**: 1 Execution : 0..* ErrorContext
- **Lifecycle**: ErrorContext created when errors occur during execution
- **Access**: Through `NodeExecution.Error` field (enhanced to be `*ErrorContext`)

### 3.2 ConnectionPool → PathValidator

- **Type**: Dependency (uses)
- **Cardinality**: N/A (different aggregates)
- **Note**: Test server (user of both) coordinates them, no direct relationship

### 3.3 ExecutionContext → context.Context

- **Type**: Dependency (wraps)
- **Cardinality**: 1 ExecutionContext : 1 context.Context
- **Lifecycle**: Context created at execution start, cancelled at completion/timeout

---

## 4. Validation Summary

### Cross-Entity Invariants

1. **Timeout Consistency**: If `ExecutionContext.TimedOut == true`, then:
   - `ExecutionContext.Status == ExecutionStatusTimedOut`
   - `ExecutionContext.TimeoutNode != ""`
   - Last `NodeExecution` in trace has timeout error

2. **Connection Lifecycle**: If `PooledConnection.closed == true`, then:
   - Connection not in `ConnectionPool.connections` map
   - All operations on connection return error

3. **Path Validation**: If `PathValidator.Validate()` succeeds, then:
   - Returned path is absolute
   - Returned path is within `PathValidator.resolvedBase`
   - Returned path has no ".." components

4. **Error Context Chain**: `ErrorContext.Cause` can be another `ErrorContext`:
   - Forms a chain of wrapped errors
   - Can use `errors.Unwrap()` to traverse chain
   - Deepest cause is non-ErrorContext error

### State Transitions

#### ExecutionContext States

```
┌─────────┐
│ Created │
└────┬────┘
     │ Start execution
     ↓
┌─────────┐
│ Running │──────────┐ Timeout
└────┬────┘          ↓
     │ Complete   ┌──────────┐
     ↓            │ TimedOut │ (terminal)
┌───────────┐     └──────────┘
│ Completed │ (terminal)
└───────────┘
```

#### ConnectionPool States

```
┌────────┐
│ Active │
└───┬────┘
    │ Close()
    ↓
┌─────────┐ Wait for operations
│ Closing │ (30s timeout)
└────┬────┘
     │ All operations done
     ↓
┌────────┐
│ Closed │ (terminal)
└────────┘
```

#### PooledConnection States

```
┌──────────┐
│ Released │ (in pool)
└─────┬────┘
      │ Get()
      ↓
┌──────────┐
│ Acquired │
└─────┬────┘
      │ Release()
      ↓
┌──────────┐    Close()
│ Released │────────────→ ┌────────┐
└──────────┘              │ Closed │ (terminal)
                          └────────┘
```

---

## 5. Migration Strategy

### Backwards Compatibility

1. **ExecutionContext**: New fields are optional:
   - If `ctx == nil`, no timeout (existing behavior)
   - If `TimeoutDuration == 0`, no timeout
   - Existing code continues to work unchanged

2. **ConnectionPool**: API changes are breaking but internal:
   - Only `pkg/mcp/health.go` calls these methods
   - Fixed in same PR as pool changes
   - External users (via MCP protocol) unaffected

3. **ErrorContext**: Wraps existing errors:
   - Can gradually adopt (wrap more errors over time)
   - Code using `errors.Is()` and `errors.As()` works unchanged
   - Logging automatically includes enhanced context

4. **PathValidator**: New functionality:
   - Test server previously had no validation (security hole)
   - Adding validation is pure security enhancement
   - No breaking changes (invalid paths now correctly rejected)

### Data Persistence

**None of these entities are persisted**:
- `ExecutionContext`: Ephemeral, exists only during execution (execution logs are persisted separately)
- `ConnectionPool`: Runtime only, connections recreated on restart
- `PathValidator`: Stateless validator, no data to persist
- `ErrorContext`: Logged as text, not persisted as structured data
- `ServerConfig`: Read from environment/config file, not written

---

## 6. Performance Considerations

### Memory Footprint

**Per Execution**:
- `ExecutionContext`: +24 bytes (context pointer + timeout fields)
- `ErrorContext`: +120 bytes (when errors occur) × number of errors
- **Total**: Negligible (<1KB per execution)

**Per Connection**:
- `PooledConnection`: +16 bytes (lifecycle tracking fields)
- **Total**: Negligible (<100 bytes per connection)

**PathValidator**:
- Per instance: ~256 bytes (cached base path + statistics)
- Per validation: 0 bytes allocated (stack only)

### Computation Overhead

**Path Validation**:
- Target: <1ms per call
- Expected: ~100μs average
- Bottleneck: `filepath.EvalSymlinks()` (system call)

**Timeout Checking**:
- Target: Zero overhead during normal execution
- Implementation: Context-based (Go runtime handles checking)
- Only overhead on timeout: cleanup and error creation

**Error Context**:
- Allocation: ~120 bytes per error
- String formatting: Deferred to `Error()` call (only when needed)
- Minimal impact (errors are exceptional path)

---

## Conclusion

This data model enhances existing aggregates while maintaining their invariants and adding minimal new entities focused on security and observability. All changes are backwards compatible or internal-only (breaking changes isolated to single package).

**Key Design Principles**:
- **Minimal disruption**: Enhance existing structures where possible
- **Security first**: Validation and audit trail built in
- **Observable**: Enhanced error context aids debugging
- **Performant**: All additions have negligible overhead
- **Type-safe**: Compilation catches API mismatches
