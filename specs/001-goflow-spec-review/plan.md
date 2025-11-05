# Implementation Plan: GoFlow - Visual MCP Workflow Orchestrator

**Branch**: `001-goflow-spec-review` | **Date**: 2025-11-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-goflow-spec-review/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

GoFlow is a workflow orchestration system for Model Context Protocol (MCP) servers enabling developers to chain multiple MCP tools into reusable workflows with conditional logic, data transformation, and parallel execution. The system provides both a terminal user interface (TUI) for visual workflow building and a CLI for programmatic execution. Core technical approach follows Domain-Driven Design with three aggregates (Workflow, Execution, MCP Server Registry), built as a single Go binary using goterm for TUI, YAML for workflow definitions, and SQLite for execution history.

## Technical Context

**Language/Version**: Go 1.21+ (per constitution: exclusive language, no CGO dependencies)
**Primary Dependencies**:
- github.com/dshills/goterm (TUI framework)
- gopkg.in/yaml.v3 (workflow YAML parsing)
- github.com/tidwall/gjson (JSON path queries for transformations)
- github.com/expr-lang/expr (sandboxed expression evaluation)
- golang.org/x/sync/errgroup (parallel execution coordination)
- modernc.org/sqlite (embedded database for execution history)

**Storage**:
- Local filesystem for workflow definitions (YAML format)
- SQLite for execution history and metadata
- System keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service) for MCP server credentials

**Testing**: go test with standard library testing package, table-driven tests for domain logic, integration tests for MCP protocol interactions, TUI interaction tests via goterm testing facilities

**Target Platform**: macOS, Linux, Windows (cross-platform terminal support required)

**Project Type**: Single CLI binary with embedded TUI

**Performance Goals**:
- Workflow validation < 100ms for workflows with < 100 nodes
- Execution startup < 500ms from command to first node execution
- Per-node overhead < 10ms (excluding MCP tool execution time)
- TUI rendering < 16ms frame time (60 FPS)
- Parallel execution supporting 50+ concurrent branches
- Memory < 100MB base + 10MB per active MCP server

**Constraints**:
- Single static binary (no runtime dependencies beyond OS standard library)
- Binary size target < 50MB
- No cloud dependencies (fully local operation)
- Workflows must be shareable without exposing secrets
- Full MCP protocol compliance (stdio, SSE, HTTP transports)

**Scale/Scope**:
- Support workflows with 1000+ nodes without degradation
- Optimize for 10-50 node workflows (typical use case)
- Handle 50+ concurrent workflow executions
- Store unlimited execution history (limited by disk space)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Domain-Driven Design (DDD) - ✅ PASS

**Requirement**: System organized around three core aggregates (Workflow, Execution, MCP Server Registry) with clear boundaries. Cross-aggregate communication via IDs only.

**Status**: COMPLIANT
- Plan defines three distinct aggregates per constitution
- Each aggregate will maintain own invariants
- No direct object references between aggregates planned

### II. Test-First Development (NON-NEGOTIABLE) - ✅ PASS

**Requirement**: Tests written before implementation. Red-Green-Refactor cycle enforced. Comprehensive test coverage required.

**Status**: COMPLIANT
- Testing approach defined (table-driven tests, integration tests, TUI tests)
- Test-first methodology will be enforced in implementation phase
- Constitution mandates pre-commit testing gates

### III. Performance Consciousness - ✅ PASS

**Requirement**: Must meet defined performance targets (< 100ms validation, < 500ms startup, < 10ms node overhead, < 16ms frame time, 50+ parallel branches, < 100MB + 10MB/server memory)

**Status**: COMPLIANT
- All performance targets explicitly documented in Technical Context
- Targets align with constitution requirements
- Performance tests will validate each target

### IV. Security by Design - ✅ PASS

**Requirement**: Keyring-only credential storage, sandboxed expression evaluation, no arbitrary code execution, shareable workflows without secrets, validated user input, filtered logs.

**Status**: COMPLIANT
- System keyring specified for credentials (never in workflow files)
- expr-lang/expr chosen for sandboxed expression evaluation
- Workflows designed to reference servers by ID (secrets separate)
- Input validation planned for all user data

### V. Observable and Debuggable - ✅ PASS

**Requirement**: Complete audit trail (node I/O, variable changes, MCP logs, error context, timing, resource metrics). Real-time TUI visualization. Structured, parseable logs.

**Status**: COMPLIANT
- SQLite execution history stores complete audit trail
- User Story 4 (P4) dedicated to execution monitoring and debugging
- Real-time TUI visualization specified
- Structured logging planned

### Pre-Phase 0 Gate Decision: ✅ PROCEED

All five constitutional principles are satisfied. No violations require justification. Complexity Tracking table remains empty.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── goflow/
    └── main.go              # CLI entry point

pkg/
├── workflow/                # Workflow aggregate
│   ├── workflow.go          # Workflow root entity
│   ├── node.go              # Node value objects
│   ├── edge.go              # Edge value objects
│   ├── variable.go          # Variable value objects
│   ├── parser.go            # YAML workflow parser
│   ├── validator.go         # Workflow validation logic
│   └── repository.go        # Workflow persistence interface
│
├── execution/               # Execution aggregate
│   ├── execution.go         # Execution root entity
│   ├── context.go           # Execution context
│   ├── runtime.go           # Workflow execution engine
│   ├── node_executor.go     # Node execution logic
│   └── repository.go        # Execution persistence interface
│
├── mcpserver/               # MCP Server Registry aggregate
│   ├── server.go            # Server root entity
│   ├── tool.go              # Tool schema value objects
│   ├── client.go            # MCP protocol client
│   ├── connection.go        # Connection management
│   └── repository.go        # Server registry persistence interface
│
├── transform/               # Data transformation engine
│   ├── expression.go        # Expression evaluator (uses expr-lang/expr)
│   ├── jsonpath.go          # JSONPath queries (uses gjson)
│   └── template.go          # Template string interpolation
│
├── storage/                 # Infrastructure layer
│   ├── sqlite.go            # SQLite implementation
│   ├── filesystem.go        # YAML file storage
│   └── keyring.go           # Credential storage (OS keyring)
│
├── tui/                     # Terminal user interface
│   ├── app.go               # TUI application root (uses goterm)
│   ├── workflow_explorer.go # Workflow list view
│   ├── workflow_builder.go  # Visual workflow editor
│   ├── execution_monitor.go # Execution visualization
│   ├── server_registry.go   # Server management UI
│   └── components/          # Reusable TUI components
│
└── cli/                     # Command-line interface
    ├── root.go              # Root command
    ├── init.go              # Initialize workflow
    ├── run.go               # Execute workflow
    ├── validate.go          # Validate workflow
    ├── server.go            # Server management commands
    └── edit.go              # Launch TUI editor

tests/
├── unit/                    # Unit tests for domain logic
│   ├── workflow/
│   ├── execution/
│   └── mcpserver/
│
├── integration/             # Integration tests
│   ├── mcp_protocol_test.go
│   ├── workflow_execution_test.go
│   └── storage_test.go
│
└── tui/                     # TUI interaction tests
    └── builder_test.go

internal/
└── testutil/                # Test utilities and fixtures
    ├── fixtures/
    └── mocks/
```

**Structure Decision**: Single project layout chosen. This is a CLI application with embedded TUI, not a web or mobile app. The structure follows hexagonal architecture with clear separation between domain logic (pkg/workflow, pkg/execution, pkg/mcpserver), infrastructure (pkg/storage), and interfaces (pkg/tui, pkg/cli). The pkg/ directory contains public packages that could be imported by other projects if needed.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
