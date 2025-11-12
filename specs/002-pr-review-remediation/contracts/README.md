# API Contracts: Code Review Remediation

**Feature**: 002-pr-review-remediation
**Date**: 2025-11-12

## Overview

This directory contains Go interface and type definitions (contracts) for the code review remediation feature. These contracts specify the public API changes and additions without implementation details.

## Contract Files

### 1. `pkg_validation.go`
**Package**: `pkg/validation` (new package)

**Purpose**: Secure file path validation to prevent directory traversal attacks

**Key Types**:
- `PathValidator`: Main validator with multi-layer security checks
- `ValidationError`: Structured error for security logging

**Key Functions**:
- `NewPathValidator(basePath)`: Create validator for directory
- `Validate(userPath)`: Validate user-provided path
- `Stats()`: Get validation statistics

**Performance**: <1ms per validation, ~100μs average

**Security Guarantees**:
- Blocks directory traversal (../../etc/passwd)
- Resolves symbolic links
- Rejects absolute paths
- Handles Windows reserved names
- Cross-platform (Unix, Windows)

---

### 2. `pkg_execution.go`
**Package**: `pkg/execution` (enhanced)

**Purpose**: Add timeout support and enhanced error context to workflow execution

**Key Changes**:
- `ExecutionContext`: Enhanced with context.Context for timeout support
- `ErrorContext`: New type for structured error information
- `Runtime.Execute()`: Now respects context timeout
- `ExecutionStatus`: Added `ExecutionStatusTimedOut` constant

**Key Types**:
- `ErrorContext`: Wraps errors with operational context (workflow ID, node ID, attributes)
- `RuntimeOption`: Functional option for runtime configuration

**Key Functions**:
- `NewErrorContext(operation, workflowID, nodeID, cause)`: Create error context
- `WithTimeout(duration)`: Configure runtime timeout

**Backwards Compatibility**: Fully backwards compatible (new fields optional, timeout defaults to disabled)

---

### 3. `pkg_mcp.go`
**Package**: `pkg/mcp` (API fixes)

**Purpose**: Fix connection pool API signature mismatches and add graceful shutdown

**Key Changes** (BREAKING within package):
- `Get(ctx, serverID)`: Changed from `Get(ctx, *MCPServer)` to `Get(ctx, string)`
- `Release(serverID)`: Changed from `Release(serverID, client)` to `Release(serverID)`
- `Close()`: Enhanced with graceful shutdown (30s grace period)

**Key Enhancements**:
- `PooledConnection.IsClosed()`: Check if connection is still usable
- Shutdown coordination (internal fields)

**Impact**: Only affects `pkg/mcp/health.go` (fixed in same PR)

---

### 4. `pkg_tui.go`
**Package**: `pkg/tui` (API fixes)

**Purpose**: Fix keyboard binding type safety and terminal input handling

**Key Changes**:
- `Mode`: Type-safe mode identifiers (removed string "global", use `ModeGlobal`)
- `KeyBindingRegistry.GetAllBindings()`: Returns `map[Mode]...` (was mixing types)
- Internal: `readKeyboardInput()` fixed to use blocking read (removed invalid `SetReadDeadline()`)

**Key Fixes**:
- Compilation error: `os.Stdin.SetReadDeadline()` doesn't exist → use blocking read + goroutine
- Type mismatch: global bindings now use `ModeGlobal` type (not string)

**Implementation**: Internal changes only (goroutine + blocking read pattern)

---

### 5. `internal_testserver.go`
**Package**: `internal/testutil/testserver` (security enhancements)

**Purpose**: Add secure file operations to test server

**Key Types**:
- `ServerConfig`: Configuration for security policies (allowed directory, max file size, logging)

**Key Changes**:
- `NewServer(config)`: Now accepts configuration with security policies
- `handleReadFile(path)`: Now validates path before reading
- `handleWriteFile(path, content)`: Now validates path before writing
- `logSecurityViolation(operation, path, err)`: Security event logging

**Key Functions**:
- `DefaultConfig()`: Secure defaults (temp dir, 10MB limit, logging enabled)
- `LoadConfig()`: Load from environment variables and config files

**Environment Variables**:
- `GOFLOW_TESTSERVER_ALLOWED_DIR`: Override allowed directory
- `GOFLOW_TESTSERVER_MAX_FILE_SIZE`: Override max file size
- `GOFLOW_TESTSERVER_LOG_SECURITY`: Enable/disable security logging

---

## Contract Conventions

### Type Definitions
- Interfaces and structs define public API
- Internal fields marked with comments
- Documentation includes examples

### Method Signatures
- Parameters and return types fully specified
- Error conditions documented
- Examples provided for non-trivial APIs

### Backwards Compatibility
- **Breaking changes marked "FIXED"** (internal to package only)
- **Enhancements marked "ENHANCED"** (backwards compatible)
- **New additions marked "NEW"**

### Performance Targets
- Path validation: <1ms per call
- Timeout checking: Zero overhead during normal execution
- Error wrapping: ~120 bytes per error

### Security Properties
- Path validation: Prevents 20+ attack vectors
- Error context: No sensitive data in messages
- Test server: Whitelist-based access only

---

## Implementation Notes

### Dependencies
All contracts use only Go standard library:
- `context` - timeout and cancellation
- `time` - durations and timestamps
- `sync` - thread-safety
- `path/filepath` - path manipulation
- `io` - for interface documentation

### Testing Requirements
Each contract requires:
1. **Unit tests**: Test all public methods
2. **Integration tests**: Test across package boundaries
3. **Security tests**: Property-based testing for validation
4. **Benchmark tests**: Verify performance targets

### Documentation
Each public function requires:
- Purpose description
- Parameter documentation
- Return value documentation
- Error conditions
- Example usage
- Performance characteristics (if relevant)

---

## Migration Guide

### For New Code
Import and use the new APIs directly:

```go
import "github.com/dshills/goflow/pkg/validation"

validator, err := validation.NewPathValidator("/var/app/data")
if err != nil {
    return err
}

validPath, err := validator.Validate(userPath)
```

### For Existing Code

**Execution package** (backwards compatible):
```go
// Optional: Add timeout
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
result, err := runtime.Execute(ctx, workflow)

// Optional: Wrap errors with context
if err != nil {
    return execution.NewErrorContext("executing workflow", wf.ID, "", err)
}
```

**MCP package** (internal changes only):
```go
// BEFORE
conn, err := pool.Get(ctx, server)
defer pool.Release(serverID, conn.Client)

// AFTER
conn, err := pool.Get(ctx, serverID)
defer pool.Release(serverID)
```

**TUI package** (no API changes):
```go
// No changes required in calling code
// Internal implementation fixed to compile correctly
app := tui.NewApp(ctx, config)
app.Run()
```

**Test server** (configuration required):
```go
// BEFORE
server := testserver.NewServer()

// AFTER
config := testserver.DefaultConfig()
config.AllowedDirectory = "/var/app/data"
server, err := testserver.NewServer(config)
```

---

## Contract Validation

### Compilation
All contracts are valid Go code that will compile once implemented. Run:
```bash
go build -o /dev/null ./specs/002-pr-review-remediation/contracts/...
```

### Type Checking
Contracts define exact types and signatures. Implementation must match precisely:
- Parameter types must match exactly
- Return types must match exactly
- Method receivers must match exactly
- Struct field types must match exactly

### Documentation
All contracts include godoc comments that will appear in generated documentation.

---

## Status

**Contract Status**: ✅ Complete

All API contracts defined for:
- ✅ Path validation (`pkg/validation`)
- ✅ Execution enhancements (`pkg/execution`)
- ✅ Connection pool fixes (`pkg/mcp`)
- ✅ TUI fixes (`pkg/tui`)
- ✅ Test server security (`internal/testutil/testserver`)

**Next Step**: Implementation (`/speckit.tasks` then `/speckit.implement`)
