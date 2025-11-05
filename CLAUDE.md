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

## Active Technologies
- Go 1.21+ (per constitution: exclusive language, no CGO dependencies) (001-goflow-spec-review)

## Recent Changes
- 001-goflow-spec-review: Added Go 1.21+ (per constitution: exclusive language, no CGO dependencies)
