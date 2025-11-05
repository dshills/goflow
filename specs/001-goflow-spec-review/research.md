# Research: GoFlow Technology Decisions

**Feature**: GoFlow - Visual MCP Workflow Orchestrator
**Date**: 2025-11-05
**Purpose**: Document technology choices and architecture decisions for implementation planning

## Core Technology Stack

### Decision: Go 1.21+ as Primary Language

**Chosen**: Go 1.21+ exclusively

**Rationale**:
- Constitutional requirement (Language & Runtime constraint)
- Excellent concurrency primitives (goroutines, channels) ideal for parallel workflow execution
- Static binary compilation enables single-file distribution
- Strong standard library reduces external dependencies
- Cross-platform support (macOS, Linux, Windows) without modification
- Fast compilation and execution meets performance targets
- No CGO means pure Go compilation for maximum portability

**Alternatives Considered**:
- **Rust**: Better memory safety guarantees, but steeper learning curve and less mature TUI ecosystem
- **Python**: Easier prototyping, but performance concerns and complex distribution (packaging, dependencies)
- **TypeScript/Node.js**: Strong ecosystem, but runtime dependencies and inferior concurrency model

---

## Domain Model Architecture

### Decision: Domain-Driven Design with Three Aggregates

**Chosen**: Hexagonal architecture with DDD aggregates (Workflow, Execution, MCP Server Registry)

**Rationale**:
- Constitutional requirement (DDD principle)
- Clear aggregate boundaries prevent complexity sprawl
- Each aggregate can be tested independently
- Cross-aggregate communication via IDs maintains loose coupling
- Enables parallel development of aggregates
- Repository pattern abstracts storage implementation

**Aggregate Boundaries**:

1. **Workflow Aggregate**:
   - Root: Workflow entity (ID, name, version, metadata)
   - Value Objects: Node, Edge, Variable
   - Invariants: No circular dependencies, unique variable names, single start node
   - Operations: Create, validate, parse YAML, export

2. **Execution Aggregate**:
   - Root: Execution entity (ID, workflow ID, status, timestamps)
   - Value Objects: ExecutionContext, NodeExecution, VariableSnapshot
   - Invariants: References valid workflow, topological execution order, append-only audit log
   - Operations: Start, execute nodes, handle errors, complete

3. **MCP Server Registry Aggregate**:
   - Root: MCPServer entity (ID, connection config, health status)
   - Value Objects: Tool, ToolSchema, Connection
   - Invariants: Unique server IDs, MCP-compliant tool schemas
   - Operations: Register, connect, discover tools, health check

**Alternatives Considered**:
- **Single monolithic model**: Simpler initially, but becomes unmaintainable as complexity grows
- **Microservices**: Over-engineering for a CLI tool, adds deployment complexity
- **Anemic domain model**: Violates DDD principles, leads to business logic scattered across services

---

## Terminal User Interface (TUI)

### Decision: goterm Framework

**Chosen**: github.com/dshills/goterm (project author's existing TUI library)

**Rationale**:
- Author familiarity reduces development time
- Proven in author's other projects
- Provides necessary primitives for workflow visualization
- Supports vim-style keybindings (requirement)
- Cross-platform terminal compatibility
- Integrates cleanly with Go concurrency model

**Implementation Approach**:
- Four main views: Workflow Explorer, Builder, Execution Monitor, Server Registry
- Component-based architecture for reusability
- Event-driven model for keyboard/mouse input
- 60 FPS rendering target (< 16ms frame time) via optimized redraws

**Alternatives Considered**:
- **tview/tcell**: Popular Go TUI library, but less familiar to author
- **bubbletea**: Modern framework with elm-like architecture, but different mental model
- **Web UI (Electron/Tauri)**: Better visuals, but violates constitution constraints (single binary, no dependencies)

---

## Workflow Definition Format

### Decision: YAML with Custom Schema

**Chosen**: YAML v3 for workflow definitions with versioned schema

**Rationale**:
- Human-readable and editable (users can hand-craft workflows)
- Well-supported parsing library (gopkg.in/yaml.v3)
- Industry standard for configuration and declarative formats
- Supports comments for documentation
- Easy to version control (text-based, diff-friendly)
- Can be validated against JSON Schema for tooling support

**Schema Structure**:
```yaml
version: "1.0"              # Workflow schema version
name: string                # Workflow identifier
description: string         # Human-readable description
metadata:
  author: string
  created: timestamp
  tags: [string]

variables: [Variable]       # Workflow-scoped variables
servers: [ServerConfig]     # MCP server configurations
nodes: [Node]               # Workflow steps
edges: [Edge]               # Connections between nodes
```

**Alternatives Considered**:
- **JSON**: More verbose, no comments, harder for humans to write
- **TOML**: Less familiar to developers, weaker nesting support
- **HCL**: Terraform-like, but unnecessary complexity for this use case
- **DSL/Programming Language**: Too powerful, violates security constraints (no arbitrary code execution)

---

## Data Transformation Engine

### Decision: Sandboxed Expression Evaluation

**Chosen**: Multi-layered approach:
1. github.com/tidwall/gjson for JSONPath queries
2. github.com/expr-lang/expr for conditional expressions
3. Go text/template for string interpolation

**Rationale**:
- **gjson**: Fast, zero-allocation JSON querying with JSONPath syntax
- **expr**: Sandboxed expression evaluator (no arbitrary code execution), supports complex boolean logic
- **text/template**: Standard library, trusted for safe string interpolation

**Security Model**:
- Expression evaluation happens in sandboxed environment (no file system, network, or OS access)
- No arbitrary function calls (whitelist safe functions only)
- Type-safe operations prevent injection attacks
- Timeout protection prevents infinite loops

**Supported Transformations**:
- JSONPath: `$.users[0].email`, `$.items[?(@.price < 100)]`
- Expressions: `count > 10 ? "many" : "few"`, `status == "active" && role == "admin"`
- Templates: `"Hello ${user.name}, order ${order.id} ready"`

**Alternatives Considered**:
- **Embedded Lua/JavaScript**: Too powerful, security risk, constitution violation
- **jq binary**: External dependency, cross-platform issues, harder to integrate
- **CEL (Common Expression Language)**: Good option, but expr-lang more Go-idiomatic

---

## Storage Layer

### Decision: Multi-Backend Strategy

**Chosen**:
1. **Workflow definitions**: Filesystem (YAML files)
2. **Execution history**: SQLite (modernc.org/sqlite - pure Go)
3. **Credentials**: System keyring (OS-provided)

**Rationale**:

**Filesystem for workflows**:
- Human-editable YAML files
- Version control friendly (git workflows)
- Easy sharing and backup
- No database required for basic operation
- Supports workflow templates (copy files)

**SQLite for execution history**:
- Pure Go implementation (modernc.org/sqlite) - no CGO
- Efficient querying of execution logs
- ACID transactions for data integrity
- Embedded (no separate database process)
- Supports indexes for fast searches
- Unlimited history limited only by disk space

**System keyring for credentials**:
- OS-provided secure storage (Keychain, Credential Manager, Secret Service)
- Credentials never in workflow files (constitutional requirement)
- Per-user isolation
- Automatic encryption at rest

**Schema Design** (SQLite):
```sql
executions (
  id, workflow_id, status, started_at, completed_at, error
)

node_executions (
  id, execution_id, node_id, status, inputs, outputs, error, started_at, completed_at
)

variable_snapshots (
  id, execution_id, node_execution_id, variable_name, value, timestamp
)
```

**Alternatives Considered**:
- **PostgreSQL**: Overkill, requires separate process, violates portability
- **BoltDB**: Key-value only, harder to query complex relationships
- **BadgerDB**: Similar to Bolt, no SQL querying
- **All-in-filesystem**: Poor query performance, no transaction support

---

## MCP Protocol Client

### Decision: Custom Implementation Based on craftMCP

**Chosen**: Build MCP client based on author's craftMCP project foundation

**Rationale**:
- Author familiarity with MCP protocol from prior work
- Can customize for GoFlow-specific needs (connection pooling, health checks)
- Full control over protocol features and extensions
- Integrated error handling and retry logic

**Transport Support**:
1. **stdio** (Priority 1): Subprocess communication via stdin/stdout
2. **SSE** (Priority 2): Server-Sent Events over HTTP
3. **HTTP with JSON-RPC** (Priority 3): Direct HTTP calls

**Implementation Approach**:
- Transport abstraction layer (interface for stdio/SSE/HTTP)
- Connection pooling (reuse connections across workflow executions)
- Automatic reconnection on failure (with exponential backoff)
- Health check pings to detect stale connections
- Tool discovery via MCP protocol introspection
- Schema caching to avoid repeated discovery

**Alternatives Considered**:
- **Existing Go MCP libraries**: Ecosystem still immature, no dominant library
- **gRPC**: Not part of MCP spec, would break compatibility
- **GraphQL**: Wrong protocol for tool invocation patterns

---

## CLI Framework

### Decision: Cobra-Style Command Structure (Custom)

**Chosen**: Custom CLI routing inspired by Cobra patterns, implemented directly

**Rationale**:
- Keep dependencies minimal (constitution constraint)
- Simple command structure doesn't need heavy framework
- Full control over command parsing and validation
- Can optimize for GoFlow-specific workflows

**Command Structure**:
```
goflow init <name>                    # Create new workflow
goflow edit <name>                    # Launch TUI editor
goflow run <name> [--input file]      # Execute workflow
goflow validate <name>                # Validate workflow
goflow export <name>                  # Export workflow
goflow import <file>                  # Import workflow
goflow server add <id> <cmd> [args]   # Register MCP server
goflow server list                    # List servers
goflow server test <id>               # Test server connection
goflow server remove <id>             # Unregister server
```

**Alternatives Considered**:
- **cobra**: Popular, but adds dependency and complexity
- **urfave/cli**: Good alternative, still external dependency
- **flag package only**: Too primitive for subcommand structure

---

## Concurrency Model

### Decision: goroutines + errgroup for Parallel Execution

**Chosen**: golang.org/x/sync/errgroup for coordinating parallel node execution

**Rationale**:
- Errgroup provides structured concurrency (wait for all goroutines, collect first error)
- Context propagation for cancellation (if one node fails, cancel others)
- Goroutines are lightweight (can handle 50+ parallel branches easily)
- Channels for node-to-node data passing
- Select statements for timeout handling

**Execution Strategy**:
1. Topological sort determines execution order
2. Independent nodes (no dependencies) execute in parallel via errgroup
3. Dependent nodes wait on channels for upstream completion
4. Context timeout prevents infinite executions
5. Graceful shutdown on interrupt (SIGINT/SIGTERM)

**Alternatives Considered**:
- **sync.WaitGroup**: Less structured, manual error collection
- **Worker pool**: Unnecessary complexity, goroutines scale well
- **Third-party DAG execution libraries**: Additional dependencies, less control

---

## Error Handling Strategy

### Decision: Typed Errors with Retry Policies

**Chosen**: Four error types with distinct handling:

1. **Validation Errors** (pre-execution): Block execution, report to user immediately
2. **Connection Errors** (MCP server): Retry with exponential backoff, fail after N attempts
3. **Execution Errors** (runtime): Configurable retry per node, fallback path support
4. **Data Errors** (transformation): Fail fast with detailed error context

**Retry Configuration**:
```yaml
retry:
  max_attempts: 3
  backoff: exponential  # or: constant, linear
  initial_delay: 1s
  max_delay: 30s
  on: [connection_error, timeout]  # Which errors to retry
```

**Error Context**:
- Full execution trace up to failure point
- Node inputs/outputs before error
- Variable state at failure
- Stack trace for unexpected errors
- MCP server logs if available

**Alternatives Considered**:
- **Panic recovery**: Too coarse-grained, loses error context
- **Result types (Go 2 proposal)**: Not available yet
- **Always retry**: Wastes time on unrecoverable errors

---

## Performance Optimization Strategies

### Decision: Targeted Optimizations for Performance Targets

**Approaches**:

1. **Workflow Validation (< 100ms target)**:
   - Cache parsed workflows (avoid re-parsing)
   - Lazy validation (only validate on changes)
   - Parallel validation of independent nodes

2. **Execution Startup (< 500ms target)**:
   - Connection pooling (reuse MCP server connections)
   - Lazy tool discovery (only when needed)
   - Pre-warm frequently used servers

3. **Node Overhead (< 10ms target)**:
   - Minimize allocations in hot path
   - Reuse buffers for data passing
   - Avoid unnecessary serialization

4. **TUI Rendering (< 16ms target)**:
   - Incremental updates (only redraw changed regions)
   - Debounce keyboard events
   - Async execution updates (don't block UI thread)

5. **Memory (< 100MB + 10MB/server target)**:
   - Streaming data through nodes (avoid loading all in memory)
   - Pool goroutines for parallel execution
   - Limit execution history in memory (page from SQLite)

**Measurement**:
- Benchmarks for critical paths (validation, execution, rendering)
- Memory profiling during development
- Performance tests in CI/CD

---

## Testing Strategy

### Decision: Multi-Layer Testing Approach

**Chosen**: Three test layers aligned with constitutional requirements

1. **Unit Tests** (pkg/*/):
   - Table-driven tests for domain logic
   - Test each aggregate independently
   - Mock repository interfaces
   - Focus on invariant validation
   - Coverage target: 80%+ for domain logic

2. **Integration Tests** (tests/integration/):
   - MCP protocol interactions (real server processes)
   - Workflow execution end-to-end
   - Storage layer (SQLite + filesystem)
   - Test cross-aggregate operations
   - Use testutil fixtures for repeatability

3. **TUI Tests** (tests/tui/):
   - Keyboard navigation flows
   - Component rendering
   - State updates
   - Use goterm testing facilities

**Test-First Process**:
1. Write failing test (Red)
2. Implement minimal code to pass (Green)
3. Refactor while keeping tests green
4. Commit only when tests pass (pre-commit gate)

**CI/CD Testing**:
- Run all tests on every commit
- Performance benchmarks tracked over time
- Linting (golangci-lint) enforced
- Test coverage reports

---

## Security Measures

### Decision: Defense in Depth

**Layers**:

1. **Input Validation**:
   - Validate all YAML against schema
   - Sanitize expression inputs
   - Check file paths for directory traversal

2. **Sandboxed Execution**:
   - Expression evaluator has no file/network access
   - Timeout protection (prevent infinite loops)
   - Memory limits on expression evaluation

3. **Credential Isolation**:
   - Keyring-only storage (never in files)
   - Credentials referenced by ID in workflows
   - No credential exposure in logs or errors

4. **MCP Server Sandboxing**:
   - Servers run with user permissions (no elevation)
   - Process isolation (separate OS processes)
   - Network access controlled by server config

5. **Audit Logging**:
   - All workflow executions logged
   - Sensitive data filtered from logs
   - Immutable audit trail in SQLite

**Security Testing**:
- Fuzz testing for YAML parser
- Expression injection attack scenarios
- Credential leak detection in exports

---

## Build and Distribution

### Decision: Single Static Binary via go build

**Chosen**: Standard go build with cross-compilation

**Build Process**:
```bash
# Build for current platform
go build -o goflow ./cmd/goflow

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o goflow-linux-amd64 ./cmd/goflow
GOOS=darwin GOARCH=amd64 go build -o goflow-darwin-amd64 ./cmd/goflow
GOOS=windows GOARCH=amd64 go build -o goflow-windows-amd64.exe ./cmd/goflow
```

**Release Process**:
1. Tag version (semantic versioning)
2. GitHub Actions builds binaries for all platforms
3. Attach binaries to GitHub release
4. Update installation instructions

**Binary Size Target**: < 50MB (achieved via standard Go build, no additional steps needed)

**Alternatives Considered**:
- **Docker container**: Violates portability, requires Docker runtime
- **Package managers**: Additional maintenance burden, doesn't reach all platforms
- **Installer scripts**: Unnecessary complexity for single binary

---

## Development Workflow Integration

### Decision: Pre-Commit Hooks + CI/CD

**Chosen**: Git hooks + GitHub Actions

**Pre-Commit Hook** (enforces constitution):
```bash
#!/bin/sh
# Run linting
golangci-lint run || exit 1

# Run tests
go test ./... || exit 1

# Run mcp-pr code review
mcp-pr review-staged --provider openai || exit 1

# All gates passed
exit 0
```

**CI/CD Pipeline**:
1. Lint check (golangci-lint)
2. Unit tests (go test)
3. Integration tests
4. Performance benchmarks
5. Build for all platforms
6. Security scan (gosec)

**Alternatives Considered**:
- **Manual verification**: Error-prone, constitution requires automation
- **Pre-push hooks only**: Allows bad commits in history
- **CI/CD only**: Too late to catch issues

---

## Summary of Key Decisions

| Area | Decision | Rationale |
|------|----------|-----------|
| Language | Go 1.21+ | Constitutional requirement, concurrency, portability |
| Architecture | DDD with 3 aggregates | Constitutional requirement, complexity management |
| TUI | goterm | Author familiarity, proven, vim keybindings support |
| Workflow Format | YAML | Human-readable, version control friendly |
| Transformations | gjson + expr + text/template | Sandboxed, performant, safe |
| Storage | Filesystem + SQLite + Keyring | Appropriate for each data type, no CGO |
| MCP Client | Custom (based on craftMCP) | Author experience, full control |
| CLI | Custom routing | Minimal dependencies, sufficient for needs |
| Concurrency | errgroup + goroutines | Structured concurrency, error handling |
| Testing | 3-layer (unit/integration/TUI) | Comprehensive coverage, fast feedback |

All decisions align with constitutional principles: DDD, test-first, performance-conscious, secure by design, and observable/debuggable.
