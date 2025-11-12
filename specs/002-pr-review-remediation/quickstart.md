# Quickstart Guide: Code Review Remediation

**Feature**: 002-pr-review-remediation
**Audience**: Developers implementing the remediation fixes
**Estimated Time**: 2-4 hours for critical fixes, 8-12 hours total

## Overview

This guide helps you quickly implement the critical security and compilation fixes identified in the code review. Work through fixes in priority order (P1 first) to unblock compilation and close security holes immediately.

---

## Prerequisites

- [ ] Go 1.21+ installed
- [ ] GoFlow repository cloned
- [ ] Branch `002-pr-review-remediation` checked out
- [ ] Familiarity with Go testing (`go test`)
- [ ] Read `research.md` (understand security requirements)

---

## Priority 1: Critical Fixes (Do These First)

These fixes are required for the code to compile and close critical security holes.

### 1.1 File Path Validation (Security Critical) ⏱️ ~2 hours

**Objective**: Prevent directory traversal attacks in test server

**Steps**:

1. **Create validation package**:
```bash
mkdir -p pkg/validation
touch pkg/validation/filepath.go
touch pkg/validation/filepath_test.go
touch pkg/validation/filepath_fuzz_test.go
```

2. **Implement `filepath.go`**:
   - Copy implementation pattern from `research.md` Section 7
   - Implement `PathValidator` struct
   - Implement `NewPathValidator(basePath)`
   - Implement `Validate(userPath)` with 6 layers:
     1. Empty check
     2. `filepath.IsLocal()` lexical validation
     3. `filepath.Clean()` + `filepath.Join()`
     4. `filepath.EvalSymlinks()` (CRITICAL for symlink attacks)
     5. `filepath.Rel()` containment check
     6. Windows reserved name check
   - Implement `ValidationError` type

3. **Write tests (`filepath_test.go`)**:
   - Table-driven tests for known attacks (from `research.md` Section 3)
   - Test cases:
     - `../../etc/passwd` → reject
     - `/etc/passwd` → reject
     - `CON` (Windows) → reject
     - `valid/file.txt` → accept
     - Empty path → reject
   - Symlink tests (Unix only)

4. **Write fuzz tests (`filepath_fuzz_test.go`)**:
```go
func FuzzValidateSecurePath(f *testing.F) {
    f.Add("../../etc/passwd")
    f.Add("/etc/shadow")
    f.Add("CON")
    f.Add("")

    f.Fuzz(func(t *testing.T, userPath string) {
        basePath := t.TempDir()
        result, err := ValidateSecurePath(basePath, userPath)

        if err == nil {
            // Verify accepted path is truly safe
            rel, _ := filepath.Rel(basePath, result)
            if strings.HasPrefix(rel, "..") {
                t.Errorf("Accepted path escapes: %q → %q", userPath, result)
            }
        }
    })
}
```

5. **Run tests**:
```bash
go test ./pkg/validation
go test -fuzz=FuzzValidateSecurePath -fuzztime=30s ./pkg/validation
```

**Validation**: All tests pass, fuzz finds no crashes

---

### 1.2 Apply Validation to Test Server ⏱️ ~30 minutes

**Objective**: Secure file operations in test server

**Steps**:

1. **Update `internal/testutil/testserver/main.go`**:
```go
import "github.com/dshills/goflow/pkg/validation"

type Server struct {
    validator *validation.PathValidator
    config    *ServerConfig
    // ... existing fields
}

func NewServer(config *ServerConfig) (*Server, error) {
    validator, err := validation.NewPathValidator(config.AllowedDirectory)
    if err != nil {
        return nil, fmt.Errorf("create path validator: %w", err)
    }
    return &Server{
        validator: validator,
        config:    config,
        // ... initialize other fields
    }, nil
}
```

2. **Fix `handleReadFile`**:
```go
func (s *Server) handleReadFile(path string) (string, error) {
    validPath, err := s.validator.Validate(path)
    if err != nil {
        s.logSecurityViolation("read", path, err)
        return "", fmt.Errorf("invalid file path: %w", err)
    }

    content, err := os.ReadFile(validPath)
    if err != nil {
        return "", fmt.Errorf("read file: %w", err)
    }
    return string(content), nil
}
```

3. **Fix `handleWriteFile`** (similar pattern)

4. **Add security logging**:
```go
func (s *Server) logSecurityViolation(operation, path string, err error) {
    if !s.config.LogSecurityEvents {
        return
    }
    log.Printf("SECURITY [testserver] Rejected %s: input=%q error=%v",
        operation, path, err)
}
```

5. **Add configuration**:
```go
type ServerConfig struct {
    AllowedDirectory  string
    MaxFileSize       int64
    LogSecurityEvents bool
    // ...
}

func DefaultConfig() *ServerConfig {
    return &ServerConfig{
        AllowedDirectory:  os.TempDir(),
        MaxFileSize:       10 * 1024 * 1024, // 10MB
        LogSecurityEvents: true,
    }
}
```

**Validation**: Security tests pass (attempt malicious paths, verify rejection + logging)

---

### 1.3 Fix TUI Input Handling (Compilation Critical) ⏱️ ~30 minutes

**Objective**: Fix compilation error in `pkg/tui/app.go:218`

**Steps**:

1. **Read current implementation**:
```bash
grep -A 20 "readKeyboardInput" pkg/tui/app.go
```

2. **Remove problematic code** (line 218):
```go
// REMOVE THIS (doesn't compile):
os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
```

3. **Replace with blocking read**:
```go
func (a *App) readKeyboardInput() {
    buf := make([]byte, 32)

    for {
        // Check for cancellation periodically
        select {
        case <-a.ctx.Done():
            return
        default:
        }

        // Blocking read (terminal already in raw mode)
        n, err := os.Stdin.Read(buf)
        if err != nil {
            if err == io.EOF {
                return
            }
            continue
        }

        if n > 0 {
            event := a.parseKeyInput(buf[:n])

            // Send event with cancellation support
            select {
            case a.inputChan <- event:
            case <-a.ctx.Done():
                return
            }
        }
    }
}
```

4. **Test compilation**:
```bash
go build ./pkg/tui
```

5. **Test interactively** (if TUI is functional):
```bash
go run ./cmd/goflow # or whatever runs the TUI
# Press keys, verify input works
# Verify CPU usage is low when idle
```

**Validation**: Code compiles, TUI responds to keyboard input, CPU idle when no input

---

### 1.4 Fix Connection Pool API (Compilation Critical) ⏱️ ~1 hour

**Objective**: Fix API signature mismatch between pool and health check

**Steps**:

1. **Update `pkg/mcp/pool.go` `Get()` signature**:
```go
// BEFORE
func (p *ConnectionPool) Get(ctx context.Context, server *MCPServer) (*PooledConnection, error)

// AFTER
func (p *ConnectionPool) Get(ctx context.Context, serverID string) (*PooledConnection, error) {
    p.mu.RLock()
    server, exists := p.servers[serverID]
    p.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("server not found: %s", serverID)
    }

    // ... rest of implementation
}
```

2. **Update `Release()` signature**:
```go
// BEFORE
func (p *ConnectionPool) Release(serverID string, client protocol.Client) error

// AFTER
func (p *ConnectionPool) Release(serverID string) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    conn, exists := p.connections[serverID]
    if !exists {
        return fmt.Errorf("connection not found: %s", serverID)
    }

    // Mark as released
    conn.LastUsedAt = time.Now()
    // ... rest of implementation
}
```

3. **Update `pkg/mcp/health.go` to use new API**:
```go
func performHealthCheck(ctx context.Context, pool *ConnectionPool, serverID string) error {
    // BEFORE:
    // conn, err := pool.Get(ctx, server) // Type mismatch!
    // defer pool.Release(serverID, conn.Client) // Wrong signature!

    // AFTER:
    conn, err := pool.Get(ctx, serverID)
    if err != nil {
        return fmt.Errorf("get connection: %w", err)
    }
    defer pool.Release(serverID)

    // ... health check logic
}
```

4. **Test compilation**:
```bash
go build ./pkg/mcp
```

5. **Run pool tests**:
```bash
go test ./pkg/mcp
```

**Validation**: Code compiles, all tests pass

---

### 1.5 Fix Keyboard Binding Types (Compilation Critical) ⏱️ ~15 minutes

**Objective**: Fix type mismatch in `pkg/tui/keyboard.go:252`

**Steps**:

1. **Locate the issue**:
```bash
grep -n "GetAllBindings" pkg/tui/keyboard.go
```

2. **Ensure Mode type is used consistently**:
```go
type Mode string

const (
    ModeNormal  Mode = "normal"
    ModeInsert  Mode = "insert"
    ModeVisual  Mode = "visual"
    ModeCommand Mode = "command"
    ModeGlobal  Mode = "global"  // FIX: Use Mode type, not string
)

type KeyBindingRegistry struct {
    bindings map[Mode]map[string]KeyBinding  // FIX: Mode type for keys
    mu       sync.RWMutex
}
```

3. **Fix `GetAllBindings()`**:
```go
// BEFORE
func (r *KeyBindingRegistry) GetAllBindings() map[Mode]map[string]KeyBinding {
    // ...
    result["global"] = globalBindings  // WRONG: string "global"
}

// AFTER
func (r *KeyBindingRegistry) GetAllBindings() map[Mode]map[string]KeyBinding {
    // ...
    result[ModeGlobal] = globalBindings  // CORRECT: Mode type
}
```

4. **Test compilation**:
```bash
go build ./pkg/tui
```

**Validation**: Code compiles without type errors

---

### 1.6 Workflow Execution Timeout (Reliability Critical) ⏱️ ~1 hour

**Objective**: Add timeout protection to prevent hung workflows

**Steps**:

1. **Enhance `ExecutionContext` in `pkg/execution/runtime.go`**:
```go
type ExecutionContext struct {
    // ... existing fields
    ctx             context.Context
    cancel          context.CancelFunc
    TimeoutDuration time.Duration
    TimedOut        bool
    TimeoutNode     string
}

func (e *ExecutionContext) Context() context.Context {
    return e.ctx
}
```

2. **Update workflow execution**:
```go
func (r *Runtime) executeWorkflow(ctx context.Context, workflow *Workflow) error {
    // Create execution context with timeout
    var execCtx context.Context
    var cancel context.CancelFunc

    if r.defaultTimeout > 0 {
        execCtx, cancel = context.WithTimeout(ctx, r.defaultTimeout)
    } else {
        execCtx, cancel = context.WithCancel(ctx)
    }
    defer cancel()

    exec := &ExecutionContext{
        // ... other fields
        ctx:             execCtx,
        cancel:          cancel,
        TimeoutDuration: r.defaultTimeout,
    }

    // Pass context to node executions
    for _, node := range sortedNodes {
        select {
        case <-execCtx.Done():
            // Timeout occurred
            exec.TimedOut = true
            exec.TimeoutNode = node.ID
            return &ErrorContext{
                Operation:  "executing workflow node",
                WorkflowID: workflow.ID,
                NodeID:     node.ID,
                Timestamp:  time.Now(),
                Cause:      execCtx.Err(),
            }
        default:
        }

        err := r.executeNode(execCtx, node, exec)
        if err != nil {
            return err
        }
    }

    return nil
}
```

3. **Add error context type**:
```go
type ErrorContext struct {
    Operation  string
    WorkflowID string
    NodeID     string
    Timestamp  time.Time
    Attributes map[string]interface{}
    Cause      error
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

4. **Add timeout tests**:
```bash
touch pkg/execution/timeout_test.go
```

```go
func TestExecutionTimeout(t *testing.T) {
    runtime := NewRuntime(WithTimeout(1 * time.Second))

    // Create workflow with blocking node
    workflow := &Workflow{
        Nodes: []Node{
            {
                ID:   "blocking-node",
                Type: "test-block",  // Test node that blocks
            },
        },
    }

    ctx := context.Background()
    _, err := runtime.Execute(ctx, workflow)

    // Verify timeout error
    if err == nil {
        t.Fatal("Expected timeout error")
    }

    var errCtx *ErrorContext
    if !errors.As(err, &errCtx) {
        t.Fatalf("Expected ErrorContext, got %T", err)
    }

    if errCtx.NodeID != "blocking-node" {
        t.Errorf("Wrong timeout node: got %s, want blocking-node", errCtx.NodeID)
    }
}
```

**Validation**: Timeout tests pass, blocking workflows terminate with timeout error

---

## Priority 2: High-Priority Fixes (Do These Next)

### 2.1 Enhanced Error Context ⏱️ ~1 hour

Apply `ErrorContext` wrapping throughout codebase for better debugging.

**Pattern**:
```go
if err != nil {
    return execution.NewErrorContext(
        "operation description",
        workflowID,
        nodeID,
        err,
    )
}
```

**Target areas**:
- Workflow validation errors
- Node execution errors
- MCP connection errors
- Variable resolution errors

---

### 2.2 Connection Pool Cleanup ⏱️ ~1 hour

Add graceful shutdown to connection pool.

**Steps**:
1. Add `closing` chan and `wg` sync.WaitGroup to pool
2. Implement `Close()` with 30s grace period
3. Track active operations with `wg.Add/Done`
4. Test shutdown with active connections

---

### 2.3 Nil/Error Check Fixes ⏱️ ~2 hours

Add missing nil/error checks identified in review.

**Process**:
1. Review code review report for nil dereference locations
2. For each location, add check:
```go
if value == nil {
    return fmt.Errorf("unexpected nil value")
}
```
3. Add regression tests

---

## Running the Full Test Suite

After implementing fixes:

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run security fuzz tests (30 seconds each)
go test -fuzz=FuzzValidateSecurePath -fuzztime=30s ./pkg/validation

# Run benchmarks
go test -bench=. ./pkg/validation
go test -bench=. ./pkg/execution

# Check coverage
go test -cover ./...
```

**Acceptance Criteria**:
- ✅ All tests pass
- ✅ No race conditions detected
- ✅ Fuzz tests find no crashes
- ✅ Benchmarks meet performance targets
- ✅ Coverage >= 80% for remediated packages

---

## Verification Checklist

Before submitting PR, verify:

### Compilation
- [ ] `go build ./...` succeeds on macOS
- [ ] `go build ./...` succeeds on Linux (if available)
- [ ] `go build ./...` succeeds on Windows (if available)

### Security
- [ ] Path validation blocks `../../etc/passwd`
- [ ] Path validation blocks `/etc/passwd`
- [ ] Path validation blocks `CON` (Windows)
- [ ] Security violations are logged
- [ ] Test server rejects paths outside allowed directory

### Reliability
- [ ] Blocking workflow times out correctly
- [ ] Timeout error includes node context
- [ ] Connection pool closes gracefully
- [ ] No goroutine leaks (run with `-race`)

### Performance
- [ ] Path validation < 1ms (run benchmarks)
- [ ] TUI responsive (< 16ms frame time)
- [ ] Execution overhead < 5% (compare before/after)

### Code Quality
- [ ] No nil dereferences in fixed paths
- [ ] All errors checked before use
- [ ] Error context includes useful information
- [ ] Code follows Go idioms

---

## Common Issues & Solutions

### Issue: Fuzz test finds crash
**Solution**: The crash reveals an edge case. Add it to table-driven tests, fix validation logic.

### Issue: Path validation too slow
**Solution**: `EvalSymlinks()` is the bottleneck. This is expected (system call). Verify < 1ms p99.

### Issue: TUI still has input lag
**Solution**: Increase `inputChan` buffer size, or check for blocking operations in event loop.

### Issue: Timeout doesn't trigger
**Solution**: Ensure context is passed to blocking operations. Check `select` has `<-ctx.Done()` case.

### Issue: Connection pool deadlock
**Solution**: Review lock ordering. Use `defer` for unlocks. Run with `-race` detector.

---

## Next Steps

After completing fixes:

1. **Run full verification** (checklist above)
2. **Generate tasks**: `/speckit.tasks` to create task breakdown
3. **Review tasks**: Ensure all issues from code review are covered
4. **Implement**: `/speckit.implement` to execute tasks
5. **Test**: Run comprehensive test suite
6. **Review**: Use `mcp-pr` for pre-commit code review
7. **Commit**: Follow commit message format in constitution
8. **PR**: Create pull request with summary of fixes

---

## Estimated Timeline

**Critical Fixes (P1)**: 4-6 hours
- Path validation: 2h
- Test server integration: 0.5h
- TUI input fix: 0.5h
- Connection pool API: 1h
- Keyboard binding types: 0.25h
- Workflow timeout: 1h

**High-Priority Fixes (P2)**: 4-6 hours
- Error context: 1h
- Connection pool cleanup: 1h
- Nil/error checks: 2h

**Testing & Verification**: 2-3 hours
- Write comprehensive tests
- Run full test suite
- Performance benchmarking
- Security validation

**Total**: 10-15 hours for complete implementation

---

## Resources

- **Research**: `research.md` - Detailed technical approach
- **Data Model**: `data-model.md` - Entity definitions and relationships
- **Contracts**: `contracts/` - API definitions
- **Specification**: `spec.md` - Requirements and success criteria
- **Code Review**: `review-results/review-report-20251112-063151.md` - Original findings

---

## Support

If blocked or need clarification:
1. Review research.md for detailed technical background
2. Check contracts/ for exact API signatures
3. Refer to spec.md for requirements
4. Consult CLAUDE.md for development conventions
