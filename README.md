# GoFlow

**Visual workflow orchestration for Model Context Protocol (MCP) servers**

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-pre--release-orange.svg)](https://github.com/dshills/goflow)

GoFlow enables developers to chain multiple MCP tools into sophisticated, reusable workflows with conditional logic, data transformation, and parallel execution - all without writing code.

## Features

- **🔗 Composability**: Chain tools from multiple MCP servers seamlessly
- **♻️ Reusability**: Save workflows as YAML files and share them across projects
- **📊 Observability**: Complete execution history with detailed debugging tools
- **🎨 Visual Builder**: Terminal UI (TUI) for interactive workflow design
- **🚀 No Code Required**: Simple YAML syntax or visual builder
- **🔒 Security First**: Credentials stored in system keyring, never in workflow files
- **⚡ Performance**: Efficient execution with connection pooling and parallel processing

## Quick Start

### Installation

**Build from Source** (Pre-release - no binaries available yet)

```bash
git clone https://github.com/dshills/goflow.git
cd goflow
go build -o goflow ./cmd/goflow

# Optional: Install to system path
sudo mv goflow /usr/local/bin/
```

**Requirements**: Go 1.21 or higher

### Your First Workflow

Create a simple read-transform-write pipeline:

```bash
# 1. Register an MCP server
goflow server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp

# 2. Create test data
echo '{"data": [{"price": 10.5}, {"price": 20.3}, {"price": 5.2}]}' > /tmp/input.json

# 3. Copy example workflow
cp examples/simple-pipeline.yaml ~/.goflow/workflows/data-pipeline.yaml

# 4. Run the workflow
goflow run data-pipeline

# 5. Check output
cat /tmp/output.txt
# Output: Total: 36.0
```

**Result**: GoFlow read the JSON file, calculated the sum of prices (36.0), and wrote it to a text file - all orchestrated through the filesystem MCP server!

## What is GoFlow?

### The Problem

Model Context Protocol (MCP) servers are powerful tools for AI-assisted development, but they operate in isolation. Building sophisticated automations requires:

- Manually chaining tool calls
- Writing glue code for data transformation
- Implementing error handling and retries
- Managing server connections and state

### The Solution

GoFlow provides:

1. **Visual Workflow Builder**: Design workflows with a terminal UI
2. **Declarative YAML Syntax**: Define workflows as code
3. **MCP Server Orchestration**: Automatic connection management
4. **Data Transformation**: Built-in JSONPath, jq, and template expressions
5. **Advanced Control Flow**: Conditions, loops, parallel execution
6. **Error Handling**: Configurable retry policies and fallback paths
7. **Execution History**: Complete audit trail with debugging support

## Core Concepts

### Workflows

Workflows are directed acyclic graphs (DAGs) of nodes connected by edges. Each node performs a specific operation, and edges define execution order.

```yaml
version: "1.0"
name: "my-workflow"
description: "What this workflow does"

variables:
  - name: "input_file"
    type: "string"
    default: "/tmp/data.json"

servers:
  - id: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]

nodes:
  - id: "start"
    type: "start"

  - id: "read"
    type: "mcp_tool"
    server: "filesystem"
    tool: "read_file"
    parameters:
      path: "${input_file}"
    output: "contents"

  - id: "end"
    type: "end"
    return: "${contents}"

edges:
  - from: "start"
    to: "read"
  - from: "read"
    to: "end"
```

### Node Types

| Type | Purpose | Example Use Case |
|------|---------|-----------------|
| **start** | Workflow entry point | Always required |
| **end** | Workflow exit point | Return final result |
| **mcp_tool** | Call MCP server tool | Read file, make API call |
| **transform** | Transform data | Extract fields, calculate values |
| **condition** | Conditional branching | Route based on data |
| **loop** | Iterate over collection | Process multiple items |
| **parallel** | Concurrent execution | Process files in parallel |

### Variables

Variables store workflow state and pass data between nodes:

```yaml
variables:
  - name: "threshold"
    type: "number"
    default: 100

nodes:
  - id: "check"
    type: "condition"
    condition: "${value} > ${threshold}"
```

### Servers

MCP servers provide tools for workflow nodes:

```yaml
servers:
  - id: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]

  - id: "http"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-fetch"]
```

## Examples

### Simple Pipeline

Read JSON, transform data, write output:

```yaml
# See: examples/simple-pipeline.yaml
version: "1.0"
name: "data-pipeline"

nodes:
  - id: "read"
    type: "mcp_tool"
    tool: "read_file"
    output: "data"

  - id: "transform"
    type: "transform"
    input: "${data}"
    expression: "jq(.data | map(.price) | add)"
    output: "total"

  - id: "write"
    type: "mcp_tool"
    tool: "write_file"
    parameters:
      content: "Total: ${total}"
```

### Conditional Branching

Route execution based on data:

```yaml
# See: examples/conditional-workflow.yaml
nodes:
  - id: "check_size"
    type: "condition"
    condition: "${file_size} > 1000000"

edges:
  - from: "check_size"
    to: "compress"
    condition: "true"
    label: "Large file"

  - from: "check_size"
    to: "upload"
    condition: "false"
    label: "Small file"
```

### Error Handling

Retry with exponential backoff:

```yaml
# See: examples/error-handling.yaml
nodes:
  - id: "fetch_api"
    type: "mcp_tool"
    tool: "http_get"
    retry:
      max_attempts: 3
      backoff: "exponential"
      initial_delay: "1s"
      on: ["connection_error", "timeout"]
```

### Parallel Processing

Process multiple items concurrently:

```yaml
# See: examples/parallel-batch.yaml
nodes:
  - id: "process_batch"
    type: "parallel"
    branches:
      - ["read_file1", "transform1", "write1"]
      - ["read_file2", "transform2", "write2"]
      - ["read_file3", "transform3", "write3"]
    merge_strategy: "wait_all"
    max_parallel: 10
```

More examples in [`examples/`](examples/) directory.

## CLI Commands

### Workflow Management

```bash
# Initialize new workflow
goflow init <workflow-name>

# Validate workflow
goflow validate <workflow-name>

# Execute workflow
goflow run <workflow-name> [options]

# Open visual editor
goflow edit <workflow-name>

# List all workflows
goflow list

# Export workflow (shareable)
goflow export <workflow-name>

# Import workflow
goflow import <file.yaml>
```

### Server Management

```bash
# Add MCP server
goflow server add <server-id> <command> [args...]

# List registered servers
goflow server list

# Test server connection
goflow server test <server-id>

# Remove server
goflow server remove <server-id>
```

### Execution History

```bash
# List executions
goflow executions [--workflow <name>]

# View execution details
goflow execution <execution-id>

# View execution logs
goflow logs <execution-id>
```

Full CLI reference: [Quickstart Guide](specs/001-goflow-spec-review/quickstart.md#cli-command-reference)

## Visual Builder (TUI)

Launch the interactive terminal UI:

```bash
goflow edit <workflow-name>
```

**Features**:
- Visual workflow canvas with node management
- Interactive edge creation and editing
- Real-time validation feedback
- Live execution monitoring
- Vim-style keyboard navigation (hjkl)

**Navigation**:
- `hjkl` or arrow keys: Navigate
- `a`: Add node
- `e`: Edit node
- `d`: Delete node
- `c`: Connect nodes
- `v`: Validate workflow
- `x`: Execute workflow
- `?`: Help

Built with [goterm](https://github.com/dshills/goterm) for efficient terminal rendering.

## Architecture

GoFlow follows Domain-Driven Design (DDD) principles:

```
pkg/
├── workflow/          # Workflow aggregate (domain model)
├── execution/         # Execution aggregate (runtime)
├── mcpserver/         # MCP server registry aggregate
├── mcp/               # MCP protocol client
├── transform/         # Data transformation engine
├── storage/           # Persistence layer (SQLite)
└── cli/               # Command-line interface
```

**Key Design Decisions**:
- **Security**: Credentials in system keyring, never in workflow files
- **Performance**: Connection pooling, parallel execution support
- **Portability**: Workflows are shareable across teams and systems
- **Observability**: Complete execution history for debugging

Full architecture: [CLAUDE.md](CLAUDE.md)

## Development Status

**Current Status**: Pre-release (Core features complete, testing & documentation in progress)

**Recent Update** (2025-11-12): Comprehensive security remediation completed, addressing 8 critical vulnerabilities including directory traversal protection, expression sandboxing, and connection pool improvements. See [CHANGELOG.md](CHANGELOG.md) for details.

| Phase | Status | Features |
|-------|--------|----------|
| **Phase 1: Foundation** | ✅ Complete | Domain model, YAML parser, MCP client, storage |
| **Phase 2: Execution** | ✅ Complete | Runtime engine, all node types, error handling, retry logic |
| **Phase 3: CLI** | ✅ Complete | All commands (run, validate, init, edit, server, credential, executions, logs) |
| **Phase 4: TUI** | ✅ Complete | Workflow builder, execution monitor, server registry, explorer views |
| **Phase 5: Advanced** | ✅ Complete | Loop nodes, parallel execution, templates, data transformations |

### Implemented Features

**Core Engine**:
- ✅ Workflow parsing and validation
- ✅ MCP client (stdio transport)
- ✅ Complete execution engine with timeout support
- ✅ All node types: MCP tool, transform, condition, loop, parallel
- ✅ Data transformation (JSONPath, jq-style, templates, boolean expressions)
- ✅ Loop and parallel execution with configurable merge strategies
- ✅ Error handling with retry policies and timeout protection

**User Interface**:
- ✅ Full CLI with all workflow management commands
- ✅ TUI workflow builder and execution monitor
- ✅ Interactive workflow editor with real-time validation
- ✅ Execution history browser with detailed logs

**Storage & Security**:
- ✅ SQLite for execution history, filesystem for workflows
- ✅ 6-layer path validation (100% malicious path detection)
- ✅ Sandboxed expression evaluation (expr-lang with timeout protection)
- ✅ Credential management with system keyring integration
- ✅ Comprehensive error context for debugging

### In Progress

- 🚧 SSE/HTTP MCP transports (currently stdio only)
- 🚧 Documentation and tutorials
- 🚧 Integration testing with real MCP servers
- 🚧 Performance optimization and benchmarking

## Testing

GoFlow has comprehensive test coverage:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test ./tests/integration/...

# Test quickstart tutorial
./scripts/test-quickstart.sh
```

Test suite includes:
- Unit tests for all packages (comprehensive coverage)
- Integration tests for MCP protocol, loops, and parallel execution
- CLI command tests with mocked dependencies
- Security tests with 20+ malicious path attack vectors
- Mock MCP server for testing (internal/testutil/testserver)
- Performance benchmarks for critical paths

## Documentation

- **[Quickstart Guide](specs/001-goflow-spec-review/quickstart.md)**: Get started in 10 minutes
- **[Examples](examples/)**: Complete workflow examples
- **[CLAUDE.md](CLAUDE.md)**: Development guide for contributors
- **[Architecture](CLAUDE.md#high-level-architecture)**: System design and patterns

Coming soon:
- Node type reference
- Expression language guide
- Advanced patterns
- MCP server development guide

## Contributing

We welcome contributions! GoFlow is in active development.

**How to Contribute**:

1. **Try it out**: Follow the quickstart and share feedback
2. **Report bugs**: Open an issue with reproduction steps
3. **Suggest features**: Discuss in GitHub Discussions
4. **Submit PRs**: Check [CLAUDE.md](CLAUDE.md) for development guide

**Development Setup**:

```bash
git clone https://github.com/dshills/goflow.git
cd goflow

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o goflow ./cmd/goflow

# Run test suite
./scripts/test-quickstart.sh
```

## Related Projects

- **[goterm](https://github.com/dshills/goterm)**: Terminal UI library used by GoFlow
- **[craftMCP](https://github.com/dshills/craftmcp)**: MCP client foundation
- **[second-opinion](https://github.com/dshills/second-opinion)**: Example MCP server

## Roadmap

### 2025 Q1 - Initial Release (In Progress)

- ✅ Core workflow engine
- ✅ MCP client (stdio)
- ✅ CLI commands
- ✅ TUI builder
- ✅ Advanced node types (loops, parallel)
- 🚧 Documentation and tutorials
- 🚧 Real-world workflow examples
- 📋 First stable release (v1.0.0)

### 2025 Q2 - Enhanced Features

- Additional MCP transports (SSE, HTTP)
- Workflow templates library
- Performance optimizations
- Enhanced TUI features (drag-and-drop visual builder)
- Workflow debugging tools

### 2025 Q3 - Enterprise Features

- Workflow scheduling and triggers
- Team collaboration features
- Enhanced credential management UI
- Monitoring, metrics, and alerting
- Workflow versioning and rollback

## License

MIT License - see [LICENSE](LICENSE) for details

## Support

- **Documentation**: [quickstart.md](specs/001-goflow-spec-review/quickstart.md)
- **Issue Tracker**: https://github.com/dshills/goflow/issues
- **Discussions**: https://github.com/dshills/goflow/discussions
- **Built-in Help**: `goflow help <command>`

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [goterm](https://github.com/dshills/goterm) - Terminal UI
- [gjson](https://github.com/tidwall/gjson) - JSON queries
- [expr](https://github.com/expr-lang/expr) - Expression evaluation
- [SQLite](https://www.sqlite.org/) - Embedded database

Inspired by workflow orchestration tools like Temporal, Airflow, and n8n, but designed specifically for MCP server composability.

---

**Status**: Pre-release (Core Complete) | **Version**: 0.9.0-rc | **Go**: 1.21+

Made with ❤️ for the MCP community

**Note**: GoFlow is feature-complete but currently undergoing testing and documentation before the 1.0.0 stable release. The core functionality is fully implemented and operational.
