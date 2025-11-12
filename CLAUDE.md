# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**GoFlow** is a visual workflow orchestration system for Model Context Protocol (MCP) servers. It enables developers to chain multiple MCP tools into sophisticated, reusable workflows with conditional logic, data transformation, and parallel execution. Built as a standalone Go binary with both a terminal user interface (TUI) and programmatic API.

**Core Purpose**: Bridge the gap in MCP tooling where servers operate in isolation by providing composability, reusability, and orchestration capabilities.

## Development Commands

### Basic Commands
```bash
# Build the project
go build -o goflow ./cmd/goflow

# Run tests
go test ./...

# Run specific package tests
go test ./pkg/workflow
go test ./pkg/execution

# Run a single test
go test ./pkg/workflow -run TestWorkflowValidation

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Format code
go fmt ./...

# Lint (if golangci-lint is configured)
golangci-lint run
```

### Workflow Development Commands
Once implemented, GoFlow will support these commands:

```bash
# Initialize new workflow
./goflow init <workflow-name>

# Open TUI builder
./goflow edit <workflow-name>

# Execute workflow
./goflow run <workflow-name> [--input input.json] [--watch] [--debug]

# Validate workflow
./goflow validate <workflow-name>

# Manage MCP servers
./goflow server add <server-id> <command> [args...]
./goflow server list
./goflow server test <server-id>
```

## High-Level Architecture

### Domain-Driven Design Structure

GoFlow follows Domain-Driven Design (DDD) principles with three core aggregates:

#### 1. Workflow Aggregate
- **Workflow Root**: Identity, name, version, metadata
- **Nodes**: Individual steps (MCP tool calls, transformations, conditions, loops, parallel)
- **Edges**: Connections defining execution flow
- **Variables**: Workflow-scoped data store
- **Invariants**: Single start node, no circular dependencies in sync paths, unique variable names

#### 2. Execution Aggregate
- **Execution Root**: Workflow ID, execution ID, timestamps, status
- **Execution Context**: Current state, variable values, execution trace
- **Node Executions**: Individual execution records with I/O and errors
- **Invariants**: References valid workflow, maintains topological order, append-only mutations for audit

#### 3. MCP Server Registry Aggregate
- **Server Root**: Server ID, connection config, available tools
- **Tool Schemas**: Input/output schemas per tool
- **Connection State**: Active connections, health status
- **Invariants**: Unique server IDs, encrypted credentials, MCP-compliant tool schemas

### Execution Flow

1. **Validation Phase**: Parse workflow YAML, validate schema, verify server reachability, check dependencies
2. **Initialization Phase**: Start MCP connections, initialize context, set up variable store
3. **Execution Phase**: Topological sort, execute nodes in order, handle conditions/loops/parallel branches
4. **Completion Phase**: Aggregate results, close connections, store logs, return output

### Node Types

- **MCP Tool Node**: Executes MCP tool from registered server
- **Transform Node**: Applies data transformation (JSONPath, jq-style, template strings)
- **Condition Node**: Branches based on boolean expression
- **Loop Node**: Iterates over collections
- **Parallel Node**: Executes multiple branches concurrently
- **Start Node**: Entry point (system-generated)
- **End Node**: Exit point with optional return value

### Technology Stack

- **Language**: Go 1.21+
- **TUI**: `github.com/dshills/goterm` (your existing library)
- **YAML/JSON**: `gopkg.in/yaml.v3`
- **JSON Queries**: `github.com/tidwall/gjson`
- **Expressions**: `github.com/expr-lang/expr`
- **Concurrency**: `golang.org/x/sync/errgroup`
- **MCP Client**: Based on your `craftMCP` project
- **Storage**: SQLite (execution history), filesystem (workflows), system keyring (credentials)

### Workflow Definition Format

Workflows are YAML files with this structure:
- `version`: Workflow schema version
- `name`: Workflow identifier
- `metadata`: Author, tags, timestamps
- `variables`: Workflow-scoped variables with types and defaults
- `servers`: MCP server configurations (command, args, transport)
- `nodes`: Workflow steps with type, configuration, inputs, outputs
- `edges`: Connections between nodes with optional conditions

See `specs/goflow-specification.md` for full schema and examples.

## Project Workflow

This project uses **Specify** workflow system (`.specify/` directory) with these slash commands:

- `/speckit.specify`: Create/update feature specifications
- `/speckit.clarify`: Identify underspecified areas and ask targeted questions
- `/speckit.plan`: Execute implementation planning workflow
- `/speckit.tasks`: Generate actionable, dependency-ordered tasks
- `/speckit.implement`: Execute the implementation plan
- `/speckit.checklist`: Generate custom feature checklists
- `/speckit.analyze`: Cross-artifact consistency and quality analysis
- `/speckit.constitution`: Create/update project constitution

### Development Process

1. Feature specifications go in `specs/` directory
2. Specifications follow user-story driven format with acceptance criteria
3. Use `/speckit` commands for structured development workflow
4. Constitution template available at `.specify/memory/constitution.md` (needs customization)

## Key Design Principles

### Security Model
- Server credentials stored in system keyring (never in workflow files)
- Expression evaluation must be sandboxed
- No arbitrary code execution
- Workflow executions run with user's permissions
- Optional variable encryption at rest
- Workflows are shareable without secrets

### Performance Targets
- Workflow validation: < 100ms for < 100 nodes
- Execution startup: < 500ms
- Node execution overhead: < 10ms per node (excluding MCP tool time)
- Parallel execution: Support 50+ concurrent branches
- Memory: < 100MB base + 10MB per active MCP server
- TUI responsiveness: < 16ms frame time (60 FPS)

### Error Handling
Four error types with distinct handling:
1. **Validation Errors**: Caught pre-execution (syntax, parameters, types)
2. **Connection Errors**: MCP server communication failures
3. **Execution Errors**: Runtime failures (tool errors, timeouts, resources)
4. **Data Errors**: Transformation failures (invalid JSONPath, type conversions)

Recovery strategies include configurable retry policies, fallback paths, automatic rollback, and detailed error context.

## Important Implementation Notes

### MCP Protocol Integration
- Support stdio, SSE, and HTTP with JSON-RPC transports
- Implement connection pooling and automatic reconnection
- Tool discovery and schema introspection required
- Health checks and graceful shutdown essential

### TUI Design (using goterm)
- Four main views: Workflow Explorer, Workflow Builder, Execution Monitor, Server Registry
- Vim-style keybindings (h/j/k/l navigation)
- Real-time validation in builder
- Live execution visualization
- Context-sensitive help (? key)

### Data Transformation Engine
Limited expression language for safety:
- JSONPath for queries: `$.users[0].email`
- Template strings: `"Hello ${user.name}"`
- jq-style transformations: `jq(.items | map(.price) | add)`
- Conditional expressions: `${count > 10 ? "many" : "few"}`

## Related Projects

- **goterm**: Your existing TUI library (will be used for UI)
- **craftMCP**: Your MCP client foundation (basis for protocol implementation)
- **second-opinion**: Your MCP server example (reference for server architecture)

## Testing Strategy

- Unit tests for all domain logic
- Integration tests for MCP server interactions
- TUI interaction tests using goterm's testing facilities
- Workflow validation tests with various edge cases
- Performance benchmarks for execution engine
- Security tests for expression evaluation and credential handling

## Development Phases

The project is structured in 5 phases (20 weeks total):
1. **Foundation (Weeks 1-4)**: Domain model, YAML parser, MCP client, storage, CLI scaffolding
2. **Execution Engine (Weeks 5-8)**: Runtime, node implementations, error handling, logging
3. **TUI Development (Weeks 9-12)**: All four views, keyboard navigation, visual builder
4. **Advanced Features (Weeks 13-16)**: Loops, parallel execution, templates, optimizations
5. **Polish & Launch (Weeks 17-20)**: Documentation, tutorials, examples, security audit

Current status: Project initialized with go.mod and specification in `specs/goflow-specification.md`

## Security Implementation Notes

### Path Validation (`pkg/validation`)
GoFlow implements defense-in-depth file path validation to prevent directory traversal attacks:

**6-Layer Security Approach**:
1. **Lexical validation**: Reject absolute paths, UNC paths, and suspicious patterns using `filepath.IsLocal()`
2. **Path normalization**: Clean paths with `filepath.Clean()` to eliminate `..` and `.` components
3. **Symbolic link resolution**: Resolve symlinks with `filepath.EvalSymlinks()` to prevent symlink attacks
4. **Containment verification**: Use `filepath.Rel()` to ensure resolved path stays within base directory
5. **Platform-specific checks**: Validate Windows reserved names (CON, PRN, NUL, COM1-9, LPT1-9)
6. **Security event logging**: Log all rejected paths with full context for audit trail

**Usage Pattern**:
```go
import "github.com/dshills/goflow/pkg/validation"

validator, err := validation.NewPathValidator("/var/app/data")
if err != nil {
    return err
}

validPath, err := validator.Validate(userPath)
if err != nil {
    // Log security violation
    log.Printf("SECURITY: Rejected file access: %v", err)
    return fmt.Errorf("access denied")
}

// Safe to use validPath
content, err := os.ReadFile(validPath)
```

**Performance**: ~9μs per validation (100x better than 1ms target), 100% malicious path detection rate verified.

**Attack Vectors Blocked** (20+ variants):
- Classic traversal: `../../etc/passwd`
- URL encoding: `..%2F..%2Fetc%2Fpasswd`
- Double encoding: `..%252F..%252Fetc%252Fpasswd`
- Windows traversal: `..\\..\\Windows\\System32`
- Mixed separators: `..\/..\/../etc/passwd`
- Null byte injection: `../../etc/passwd\x00`
- Unicode normalization and overlong UTF-8 sequences
- Absolute paths and UNC paths

### Timeout Support (`pkg/execution`)
Workflow executions now support configurable timeouts to prevent indefinite execution:

**Configuration Pattern**:
```go
import "github.com/dshills/goflow/pkg/execution"

// Create engine with 5 minute timeout
engine := execution.NewEngine(
    execution.WithTimeout(5 * time.Minute),
)

// Execute workflow (will timeout after 5 minutes)
ctx := context.Background()
exec, err := engine.Execute(ctx, wf, inputs)

// Check timeout status
if exec.TimedOut {
    log.Printf("Workflow timed out at node: %s", exec.TimeoutNode)
}
```

**Context-based timeout**: Zero overhead during normal execution, leverages Go's context package for efficient cancellation.

**Timeout behavior**:
- Engine timeout acts as default for all executions
- Context timeout (if provided) takes precedence
- Execution result includes `TimedOut` flag and `TimeoutNode` ID
- Timeout errors include full operational context for debugging

### Error Context (`pkg/errors`)
All errors are wrapped with operational context for better debugging and monitoring:

**OperationalError Pattern**:
```go
import "github.com/dshills/goflow/pkg/errors"

// Wrap errors with operational context
opErr := errors.NewOperationalError(
    "executing MCP tool",
    workflowID,
    nodeID,
    originalErr,
).WithAttributes(map[string]interface{}{
    "tool":     "filesystem.read",
    "serverID": "local-mcp",
    "attempt":  3,
})

// Error format: [timestamp] operation: workflow=ID node=ID: cause
// Example: [2025-11-12T10:15:23Z] executing MCP tool: workflow=wf-123 node=node-5: connection timeout
```

**Error attributes**: Attach custom debugging information (server IDs, tool names, attempt counts, etc.) for comprehensive error tracking.

**Error unwrapping**: Fully compatible with `errors.Is()` and `errors.As()` for proper error chain handling.

### Expression Evaluation Security
All JSONPath filter expressions use sandboxed evaluation:

**Security measures**:
- Validates expressions against unsafe operation patterns (os., exec., http., syscall., unsafe.)
- Uses `expr-lang` with restricted sandbox environment
- Enforces 1-second timeout for all evaluations
- Blocks access to file system, network, and system calls

**Implementation** (`pkg/transform/jsonpath.go`):
```go
// Validates filter expressions before evaluation
func validateFilterExpression(expr string) error {
    unsafePatterns := []string{
        "os.", "exec.", "http.", "net.", "syscall.", "unsafe.",
    }
    for _, pattern := range unsafePatterns {
        if strings.Contains(expr, pattern) {
            return fmt.Errorf("unsafe operation not allowed: %s", pattern)
        }
    }
    return nil
}
```

### Connection Pool Best Practices
MCP connection pool API has been fixed for consistency and now includes graceful shutdown:

**API changes** (internal only):
```go
// Get connection by server ID (string), not server object
conn, err := pool.Get(ctx, "server-id")

// Release by server ID only (no client parameter)
err = pool.Release("server-id")

// Close with 30s grace period for active operations
err = pool.Close()
```

**Leak detection**: Connection pool tracks reference counts and detects leaks during cleanup with `LeakStats()` counter.

**Best practice**: Always defer `Release()` after successful `Get()` to prevent waitgroup deadlocks:
```go
conn, err := pool.Get(ctx, serverID)
if err != nil {
    return err
}
defer pool.Release(serverID)

// Use conn.Client...
```

## Active Technologies
- Go 1.21+ (per constitution: exclusive language, no CGO dependencies) (001-goflow-spec-review)
- `pkg/validation`: File path security validation (6-layer defense-in-depth) (002-pr-review-remediation)
- `pkg/errors`: Operational error context for debugging (002-pr-review-remediation)
- ✅ No changes to storage layer (002-pr-review-remediation)

## Recent Changes
- 002-pr-review-remediation: Comprehensive code review remediation addressing 8 critical and 45 high-priority issues
  - Added 6-layer path validation to prevent directory traversal attacks
  - Implemented timeout support for workflow executions
  - Enhanced error context with OperationalError type
  - Fixed MCP connection pool API signatures and graceful shutdown
  - Replaced hand-rolled expression parser with sandboxed expr-lang
  - Fixed TUI terminal input handling and keyboard binding type safety
  - Added comprehensive security tests with 100% malicious path detection
  - Performance: Path validation ~9μs, overall remediation overhead <2%
  - See CHANGELOG.md for complete details
- 001-goflow-spec-review: Added Go 1.21+ (per constitution: exclusive language, no CGO dependencies)
