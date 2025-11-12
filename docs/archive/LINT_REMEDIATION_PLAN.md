# Lint Remediation Plan

**Date**: 2025-11-12
**Total Issues**: 121
**Strategy**: 4-Phase systematic remediation with test validation between phases

## Issue Breakdown

| Category | Count | Priority | Risk Level |
|----------|-------|----------|------------|
| unused | 38 | High | Low (safe removal) |
| errcheck | 50 | Critical | Medium (requires error handling) |
| staticcheck | 28 | Medium | Low (code quality) |
| govet | 1 | High | Medium (nil map) |
| ineffassign | 4 | Low | Low (cleanup) |

## Phase 1: Remove Unused Code (38 issues) ⚡ Quick Wins

**Rationale**: Start with safest changes that have highest impact on reducing noise.

### 1.1 Unused Struct Fields (10 fields)
```go
// pkg/tui/execution_monitor.go
scrollOffset int                              // Line 488

// pkg/tui/execution_monitor_panels.go
filterLevel string                            // Line 20
enhancedError *execpkg.EnhancedExecutionError // Line 249

// pkg/tui/profiler.go
memProfileFile *os.File                       // Line 100

// pkg/tui/server_registry.go
headers string                                // Line 48

// pkg/tui/view_explorer.go
searchQuery string                            // Line 14

// pkg/tui/workflow_builder.go
offsetX, offsetY int                          // Lines 48-49
editMode bool                                 // Line 80
keyBindings []HelpKeyBinding                  // Line 98

// pkg/validation/filepath.go
mu sync.RWMutex                              // Line 29

// pkg/cli/run.go
recentLogs []string                          // Line 383
```

**Action**: Remove fields and ensure no references exist in tests.

### 1.2 Unused Functions (24 functions)

#### Example/Tutorial Code (pkg/execution/error_integration_example.go)
```go
exampleMCPToolNodeWithEnhancedErrors()        // Line 17
exampleBuildComplexError()                    // Line 67
exampleErrorRecoveryStrategy()                // Line 99
newExampleLogCollector()                      // Line 147
exampleErrorMonitoring()                      // Line 192
recordErrorMetric()                           // Line 213
sendCriticalAlert()                           // Line 217
storeErrorForDebugging()                      // Line 221
logStructuredError()                          // Line 225
retryWithBackoff()                            // Line 230
retryWithIncreasedTimeout()                   // Line 235
exampleCompleteNodeExecutionFlow()            // Line 241
executeNodeSimulated()                        // Line 280
exampleErrorReporting()                       // Line 286
formatExecutionPath()                         // Line 304
```

**Decision**: Move to `examples/` directory or delete if redundant with real code.

#### Production Code
```go
// pkg/execution/runtime.go
(*Engine).topologicalSort()                   // Line 517 - replaced by workflow.TopologicalSort()

// pkg/mcp/connection_pool.go
(*ConnectionPool).cleanupIdleConnections()    // Line 356 - intended for future use?

// pkg/transform/jsonpath.go
parseFilterValue()                            // Line 990
compareNumeric()                              // Line 1038

// pkg/tui/execution_monitor_panels.go
screenContainsText()                          // Line 617
containsSubstring()                           // Line 634

// pkg/tui/workflow_explorer.go
sortWorkflowsByName()                         // Line 652
```

**Action**: Remove or move to examples. Verify with `git grep` before deletion.

### 1.3 Unused Types (5 items)
```go
// pkg/execution/error_integration_example.go
type exampleMCPLogCollectorImpl              // Line 143
(*exampleMCPLogCollectorImpl).CollectLogs()  // Line 153
(*exampleMCPLogCollectorImpl).CollectLogsForExecution() // Line 177
(*exampleMCPLogCollectorImpl).addLog()       // Line 187
```

**Action**: Move entire type to examples or delete.

---

## Phase 2: Fix Error Handling (50 errcheck issues) 🔥 Critical

**Rationale**: Unhandled errors can lead to silent failures and production bugs.

### 2.1 Defer Error Checks (20 issues)

**Pattern**: Add error check or document intentional ignore.

```go
// BAD
defer file.Close()

// GOOD (when error is relevant)
defer func() {
    if err := file.Close(); err != nil {
        log.Printf("Failed to close file: %v", err)
    }
}()

// GOOD (when error is irrelevant - document why)
defer file.Close() // Error ignored: file is read-only, close failure is benign
```

**Files Affected**:
- `examples/performance_optimization_example.go:89` - pool.Close()
- `pkg/cli/edit.go:62` - app.Close()
- `pkg/mcp/http_client.go:125,172` - httpResp.Body.Close()
- `pkg/mcp/sse_client.go:201,258` - httpResp.Body.Close()
- `pkg/tui/app.go:58,64,70` - screen.Close()

### 2.2 Workflow Builder Errors (15 issues)

**Problem**: Builder methods return errors that are being ignored.

```go
// BAD
wf.AddVariable(&workflow.Variable{...})

// GOOD
if err := wf.AddVariable(&workflow.Variable{...}); err != nil {
    return fmt.Errorf("failed to add variable: %w", err)
}
```

**Files Affected**:
- `pkg/cli/init.go` - AddVariable (2), AddNode (5), AddEdge (4)
- `pkg/tui/examples/condition_example.go` - AddVariable (3), AddNodeToCanvas (5), CreateEdge (1), CreateConditionalEdge (2)
- `pkg/tui/examples/loop_example.go` - Similar pattern (15 calls)
- `pkg/tui/examples/parallel_example.go` - Similar pattern (8 calls)

### 2.3 Miscellaneous Errors (15 issues)

**Mix of I/O, formatting, and write operations**:
- `pkg/execution/error.go:354` - fmt.Sscanf
- `pkg/cli/run.go` - fmt.Fprintf, Write operations
- `pkg/cli/server.go` - fmt.Fprintf
- `pkg/tui/server_registry.go` - fmt.Fprintf

---

## Phase 3: Apply Staticcheck Improvements (28 issues) 📊 Code Quality

### 3.1 Deprecated Functions (1 issue)

```go
// BAD
strings.Title(s)

// GOOD
import "golang.org/x/text/cases"
import "golang.org/x/text/language"

titleCaser := cases.Title(language.English)
titleCaser.String(s)
```

**File**: `pkg/tui/workflow_builder.go:1290`

### 3.2 Simplifications (10 issues)

#### S1031: Unnecessary nil check around range
```go
// BAD
if initialVars != nil {
    for k, v := range initialVars {
        // ...
    }
}

// GOOD
for k, v := range initialVars {
    // ... (range over nil slice is safe)
}
```

#### S1005: Unnecessary blank identifier assignment
```go
// BAD
oldValue, _ := ctx.Variables[name]

// GOOD
oldValue := ctx.Variables[name]  // If _ is truly unused
```

#### QF1004: Use strings.ReplaceAll
```go
// BAD
strings.Replace(result, placeholder, strValue, -1)

// GOOD
strings.ReplaceAll(result, placeholder, strValue)
```

### 3.3 Tagged Switch (10 issues)

**Pattern**: Convert if-else chains to type switches.

```go
// BAD
if path[j] == '(' {
    // ...
} else if path[j] == '[' {
    // ...
} else if path[j] == '.' {
    // ...
}

// GOOD
switch path[j] {
case '(':
    // ...
case '[':
    // ...
case '.':
    // ...
}
```

**Files**: `pkg/transform/jsonpath.go` (6 locations), `pkg/workflow/expression_validator.go` (3 locations), others (1 location)

### 3.4 Empty Branches & Logic (7 issues)

#### SA9003: Empty branch
```go
// BAD
if condition {
    // TODO: implement
}

// GOOD
if condition {
    // TODO: Implement feature X in Phase 4
    _ = condition // Explicitly acknowledge intentional no-op
}

// OR just remove the branch
```

#### QF1001: De Morgan's law
```go
// BAD
if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'))

// GOOD
if !isAlpha(ch)  // Extract to named function for clarity
```

---

## Phase 4: Fix Remaining Issues (5 issues) 🎯 Final Cleanup

### 4.1 Govet: Nil Map Assignment (1 issue)

```go
// internal/testutil/testserver/config.go - Need to see the code
entry, found = cache.Get(nodeID, nodeType, inputs)
```

**Action**: Initialize map before use or check for nil.

### 4.2 Ineffassign: Ineffectual Assignments (4 issues)

**Pattern**: Variable assigned but never read.

```go
// BAD
result := calculate()
result = otherValue  // First assignment wasted

// GOOD
result := otherValue  // Skip redundant assignment
```

**Files**: Need to identify specific locations (not shown in truncated output).

---

## Testing Strategy

### After Each Phase
```bash
# 1. Run affected package tests
go test ./pkg/tui/...
go test ./pkg/execution/...

# 2. Run full test suite
go test ./...

# 3. Run with race detector
go test -race ./...

# 4. Verify lint improvements
golangci-lint run

# 5. Check test coverage
go test -cover ./...
```

### Test Coverage Targets
- Maintain or improve current 96% test pass rate
- Zero race conditions (validated with -race flag)
- Zero new test failures introduced

---

## Risk Assessment

| Phase | Risk | Mitigation |
|-------|------|------------|
| Phase 1: Unused Code | **LOW** | Git grep before deletion, incremental commits |
| Phase 2: Error Handling | **MEDIUM** | Careful review of error semantics, test each file |
| Phase 3: Staticcheck | **LOW** | Mostly mechanical refactoring, well-tested patterns |
| Phase 4: Final Cleanup | **MEDIUM** | Requires understanding context of each issue |

---

## Implementation Timeline

| Phase | Estimated Time | Complexity |
|-------|----------------|------------|
| Phase 1: Unused Code (38) | 30 min | Low (mechanical deletion) |
| Phase 2: Error Handling (50) | 90 min | Medium (requires judgment) |
| Phase 3: Staticcheck (28) | 45 min | Low (mechanical refactoring) |
| Phase 4: Final Cleanup (5) | 20 min | Medium (requires analysis) |
| **Testing & Validation** | 30 min | N/A |
| **Documentation** | 15 min | N/A |
| **Total** | ~4 hours | Mixed |

---

## Success Criteria

- ✅ `golangci-lint run` reports 0 issues
- ✅ All tests pass: `go test ./...`
- ✅ No race conditions: `go test -race ./...`
- ✅ Test coverage unchanged or improved
- ✅ No new test failures introduced
- ✅ Comprehensive documentation in LINT_FIX_SUMMARY.md
- ✅ Clean git commit history with atomic changes per phase

---

## Execution Strategy

1. **Incremental commits**: One commit per phase for easy rollback
2. **Atomic changes**: Group related fixes together (e.g., all errcheck in one file)
3. **Test between phases**: Never proceed if tests fail
4. **Document decisions**: Comment non-obvious error ignores
5. **Preserve history**: Use `git mv` for file moves, maintain git blame

---

## Notes

- **Pre-existing issues**: All 121 issues existed before recent test fixes and code review remediation
- **No regressions**: Recent commits (96aea3e, a8e8202) did not introduce lint issues
- **Priority**: Focus on errcheck (safety) before staticcheck (style)
- **Examples folder**: Consider moving error_integration_example.go to examples/ vs deleting

---

## Next Steps

1. Review and approve this plan
2. Execute Phase 1 (unused code removal)
3. Run tests and verify
4. Proceed to Phase 2 if Phase 1 successful
5. Continue until all phases complete
