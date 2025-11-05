# GoFlow Quickstart Guide

**Version**: 1.0 | **Last Updated**: 2025-11-05 | **For**: GoFlow v1.0.0

Welcome to GoFlow! This guide will get you up and running with workflow orchestration for Model Context Protocol (MCP) servers in under 10 minutes.

## What is GoFlow?

GoFlow lets you chain multiple MCP tools into automated workflows with conditional logic, data transformation, and parallel execution - all without writing code. Think of it as a visual workflow builder and execution engine for MCP servers.

**Key Benefits**:
- **Composability**: Chain tools from multiple MCP servers
- **Reusability**: Save workflows as YAML files and share them
- **Observability**: Complete execution history with debugging tools
- **No Code Required**: Visual builder or simple YAML syntax

## Prerequisites

- **Operating System**: macOS, Linux, or Windows
- **MCP Servers**: At least one MCP server installed (we'll use filesystem server in examples)
- **Go 1.21+**: Only required if building from source (not needed for binary installation)

## Installation

### Option 1: Download Binary (Recommended)

```bash
# macOS (Intel)
curl -L https://github.com/dshills/goflow/releases/latest/download/goflow-darwin-amd64 -o goflow
chmod +x goflow
sudo mv goflow /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/dshills/goflow/releases/latest/download/goflow-darwin-arm64 -o goflow
chmod +x goflow
sudo mv goflow /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/dshills/goflow/releases/latest/download/goflow-linux-amd64 -o goflow
chmod +x goflow
sudo mv goflow /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/dshills/goflow/releases/latest/download/goflow-windows-amd64.exe" -OutFile "goflow.exe"
# Move to a directory in your PATH
```

### Option 2: Build from Source

```bash
git clone https://github.com/dshills/goflow.git
cd goflow
go build -o goflow ./cmd/goflow
sudo mv goflow /usr/local/bin/
```

### Verify Installation

```bash
goflow --version
# Expected output: goflow v1.0.0
```

## Your First Workflow

Let's create a simple workflow that reads a file, transforms the data, and writes the result to a new file.

### Step 1: Register an MCP Server

First, register the filesystem MCP server:

```bash
goflow server add filesystem \
  npx \
  -y @modelcontextprotocol/server-filesystem /tmp

# Verify registration
goflow server list
# Expected output:
# ID           | Name                    | Status  | Tools
# -------------|-------------------------|---------|-------
# filesystem   | Filesystem Server       | Healthy | 3
```

### Step 2: Test Server Connection

```bash
goflow server test filesystem
# Expected output:
# ✓ Connection successful
# ✓ Discovered 3 tools: read_file, write_file, list_directory
```

### Step 3: Create a Workflow

Create a new workflow using the interactive builder:

```bash
goflow init data-pipeline
```

This creates a workflow file at `~/.goflow/workflows/data-pipeline.yaml`. Edit it with your favorite editor:

```yaml
version: "1.0"
name: "data-pipeline"
description: "Read file, transform data, write output"

metadata:
  author: "your-name"
  created: "2025-11-05T12:00:00Z"
  tags: ["etl", "data"]

variables:
  - name: "input_file"
    type: "string"
    default: "/tmp/input.json"
  - name: "output_file"
    type: "string"
    default: "/tmp/output.txt"

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
    output: "file_contents"

  - id: "transform"
    type: "transform"
    input: "${file_contents}"
    expression: "jq(.data | map(.price) | add)"
    output: "total_price"

  - id: "write"
    type: "mcp_tool"
    server: "filesystem"
    tool: "write_file"
    parameters:
      path: "${output_file}"
      content: "Total: ${total_price}"
    output: "write_result"

  - id: "end"
    type: "end"
    return: "${write_result}"

edges:
  - from: "start"
    to: "read"

  - from: "read"
    to: "transform"

  - from: "transform"
    to: "write"

  - from: "write"
    to: "end"
```

### Step 4: Validate the Workflow

```bash
goflow validate data-pipeline
# Expected output:
# ✓ Workflow structure valid
# ✓ All nodes reachable
# ✓ All servers registered
# ✓ No circular dependencies
# ✓ Variable types consistent
```

### Step 5: Create Test Data

```bash
echo '{"data": [{"price": 10.5}, {"price": 20.3}, {"price": 5.2}]}' > /tmp/input.json
```

### Step 6: Execute the Workflow

```bash
goflow run data-pipeline
# Expected output:
# ✓ Started execution (ID: exec-1234567890)
# ✓ Node 'read' completed (0.05s)
# ✓ Node 'transform' completed (0.01s)
# ✓ Node 'write' completed (0.03s)
# ✓ Workflow completed successfully (0.12s)
# Return value: {"success": true}
```

### Step 7: Verify Output

```bash
cat /tmp/output.txt
# Expected output: Total: 36.0
```

**Congratulations!** You've created and executed your first GoFlow workflow.

## Visual Builder (TUI)

For a more interactive experience, launch the visual workflow builder:

```bash
goflow edit data-pipeline
```

### TUI Navigation

**Workflow Explorer View**:
- `j/k` or `↓/↑`: Navigate workflow list
- `Enter`: Open workflow in builder
- `n`: Create new workflow
- `d`: Delete selected workflow
- `r`: Rename workflow
- `Tab`: Switch to Server Registry view
- `q`: Quit

**Workflow Builder View**:
- `hjkl` or `←↓↑→`: Navigate canvas
- `a`: Add new node
- `e`: Edit selected node
- `d`: Delete selected node
- `c`: Connect two nodes (create edge)
- `v`: Validate workflow
- `x`: Execute workflow
- `s`: Save workflow
- `Tab`: Switch to Execution Monitor view
- `Esc`: Return to Explorer

**Execution Monitor View**:
- `j/k` or `↓/↑`: Navigate execution list
- `Enter`: View execution details
- `l`: View logs for selected execution
- `f`: Filter by status (running/completed/failed)
- `Tab`: Switch to Explorer
- `q`: Quit

**Server Registry View**:
- `j/k` or `↓/↑`: Navigate server list
- `a`: Add new server
- `t`: Test selected server connection
- `e`: Edit server configuration
- `d`: Remove server
- `i`: View server tools and schemas
- `Tab`: Switch to Explorer
- `q`: Quit

**Help**: Press `?` in any view for context-sensitive help.

## Common Workflow Patterns

### Pattern 1: Conditional Execution

Execute different paths based on data:

```yaml
nodes:
  - id: "check_size"
    type: "condition"
    condition: "file_size > 1000000"

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

### Pattern 2: Parallel Execution

Run multiple branches concurrently:

```yaml
nodes:
  - id: "parallel_processing"
    type: "parallel"
    branches:
      - ["validate_schema", "check_duplicates"]
      - ["enrich_data", "normalize_dates"]
    merge: "wait_all"
```

### Pattern 3: Loop Over Collection

Process each item in an array:

```yaml
nodes:
  - id: "process_files"
    type: "loop"
    collection: "${file_list}"
    item: "current_file"
    body: ["read_file", "transform_file", "write_file"]
    break_condition: "error_count > 5"
```

### Pattern 4: Error Handling with Retry

Automatically retry failed operations:

```yaml
nodes:
  - id: "fetch_data"
    type: "mcp_tool"
    server: "http_client"
    tool: "get"
    parameters:
      url: "${api_endpoint}"
    output: "api_response"
    retry:
      max_attempts: 3
      backoff: "exponential"
      initial_delay: "1s"
      max_delay: "30s"
      on: ["connection_error", "timeout"]
```

### Pattern 5: Data Transformation

Transform data between nodes:

```yaml
nodes:
  # JSONPath query
  - id: "extract_emails"
    type: "transform"
    input: "${users}"
    expression: "$.users[*].email"
    output: "email_list"

  # Template string
  - id: "format_message"
    type: "transform"
    input: "${user}"
    expression: "Hello ${user.name}, your order ${order.id} is ready"
    output: "message"

  # Conditional expression
  - id: "categorize"
    type: "transform"
    input: "${count}"
    expression: "${count > 100 ? 'high' : count > 10 ? 'medium' : 'low'}"
    output: "category"
```

## CLI Command Reference

### Workflow Management

```bash
# Initialize new workflow
goflow init <workflow-name>

# Validate workflow
goflow validate <workflow-name>

# Execute workflow
goflow run <workflow-name> [options]
  --input <file>      # Provide input variables from JSON file
  --watch             # Monitor execution in real-time
  --debug             # Enable debug logging

# Export workflow (shareable without secrets)
goflow export <workflow-name> [--output file.yaml]

# Import workflow
goflow import <file.yaml>

# List all workflows
goflow list

# Delete workflow
goflow delete <workflow-name>

# Open visual editor
goflow edit <workflow-name>
```

### Server Management

```bash
# Add MCP server
goflow server add <server-id> <command> [args...]
  --transport stdio|sse|http  # Transport type (default: stdio)
  --env KEY=VALUE             # Environment variables
  --credential-ref <ref>      # Reference to keyring entry

# List registered servers
goflow server list

# Test server connection
goflow server test <server-id>

# Remove server
goflow server remove <server-id>

# View server details and tools
goflow server info <server-id>
```

### Execution History

```bash
# List executions
goflow executions [--workflow <name>] [--status running|completed|failed]

# View execution details
goflow execution <execution-id>

# View execution logs
goflow logs <execution-id>

# Cancel running execution
goflow cancel <execution-id>
```

### Credential Management

```bash
# Store credential in system keyring
goflow credential add <credential-id>
# Prompts for secret value (not shown in terminal)

# List credential references (not values)
goflow credential list

# Remove credential
goflow credential remove <credential-id>
```

## Troubleshooting

### Issue: "Server not found"

**Symptom**: `Error: MCP server 'myserver' not registered`

**Solution**:
```bash
# List registered servers
goflow server list

# Add the missing server
goflow server add myserver <command> [args...]

# Verify registration
goflow server test myserver
```

### Issue: "Connection timeout"

**Symptom**: `Error: Connection to server 'myserver' timed out`

**Solution**:
1. Test server independently:
   ```bash
   goflow server test myserver
   ```

2. Check server logs:
   ```bash
   goflow logs <execution-id> --server myserver
   ```

3. Verify server command is correct:
   ```bash
   # Test command manually
   npx -y @modelcontextprotocol/server-filesystem /tmp
   ```

### Issue: "Validation failed"

**Symptom**: `Error: Workflow validation failed: circular dependency detected`

**Solution**:
```bash
# View detailed validation errors
goflow validate <workflow-name> --verbose

# Common validation issues:
# - Circular edges (A → B → C → A)
# - Missing start or end node
# - Disconnected nodes
# - Invalid variable references
# - Type mismatches
```

### Issue: "Transformation error"

**Symptom**: `Error: Transform failed: invalid JSONPath expression`

**Solution**:
1. Test expression in isolation:
   ```bash
   echo '{"users": [{"name": "Alice"}]}' | \
   goflow test-expression '$.users[0].name'
   ```

2. Check variable types match expected input
3. Verify JSONPath syntax (see [JSONPath documentation](https://goessner.net/articles/JsonPath/))

### Issue: "Execution stuck"

**Symptom**: Workflow execution appears frozen

**Solution**:
1. Check execution status:
   ```bash
   goflow execution <execution-id>
   ```

2. View real-time logs:
   ```bash
   goflow logs <execution-id> --follow
   ```

3. If truly stuck, cancel and retry:
   ```bash
   goflow cancel <execution-id>
   goflow run <workflow-name> --debug
   ```

### Issue: "Permission denied"

**Symptom**: `Error: Permission denied accessing /path/to/file`

**Solution**:
1. Verify file permissions:
   ```bash
   ls -l /path/to/file
   ```

2. Ensure MCP server has access to the directory:
   ```bash
   # Filesystem server only accesses allowed directories
   # Example: npx @modelcontextprotocol/server-filesystem /tmp
   # Can only access files under /tmp
   ```

3. Check workflow working directory configuration

## Performance Tips

### Tip 1: Connection Pooling

Reuse MCP server connections across workflow executions:

```yaml
servers:
  - id: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    # Connection is pooled automatically - no configuration needed
```

### Tip 2: Parallel Execution

Use parallel nodes for independent operations:

```yaml
# Sequential (slow)
edges:
  - from: "fetch_user"
    to: "fetch_orders"
  - from: "fetch_orders"
    to: "fetch_products"

# Parallel (fast)
nodes:
  - id: "fetch_all"
    type: "parallel"
    branches:
      - ["fetch_user"]
      - ["fetch_orders"]
      - ["fetch_products"]
    merge: "wait_all"
```

### Tip 3: Minimize Transformations

Prefer native MCP tool outputs over complex transformations:

```yaml
# Less efficient
nodes:
  - id: "read"
    type: "mcp_tool"
    tool: "read_file"
    output: "raw_data"

  - id: "transform"
    type: "transform"
    input: "${raw_data}"
    expression: "jq(.data)"
    output: "parsed_data"

# More efficient (if MCP tool supports it)
nodes:
  - id: "read"
    type: "mcp_tool"
    tool: "read_file"
    parameters:
      format: "json"
      extract: "$.data"
    output: "parsed_data"
```

### Tip 4: Workflow Caching

Enable caching for workflows that process the same data:

```bash
goflow run data-pipeline --cache
# Subsequent runs with same input skip unchanged nodes
```

## Next Steps

Now that you're familiar with GoFlow basics, explore these resources:

1. **Example Workflows**: Browse pre-built workflows in the [examples directory](https://github.com/dshills/goflow/tree/main/examples)
2. **Node Type Reference**: Learn about all available node types in [docs/nodes.md](https://github.com/dshills/goflow/blob/main/docs/nodes.md)
3. **Expression Language**: Master data transformations in [docs/expressions.md](https://github.com/dshills/goflow/blob/main/docs/expressions.md)
4. **Advanced Patterns**: See complex workflow patterns in [docs/patterns.md](https://github.com/dshills/goflow/blob/main/docs/patterns.md)
5. **MCP Server Development**: Create custom MCP servers for GoFlow in [docs/mcp-servers.md](https://github.com/dshills/goflow/blob/main/docs/mcp-servers.md)

## Getting Help

- **Documentation**: https://github.com/dshills/goflow/tree/main/docs
- **Issue Tracker**: https://github.com/dshills/goflow/issues
- **Discussions**: https://github.com/dshills/goflow/discussions
- **Built-in Help**: `goflow help <command>`

## Example Workflows Library

Check out the [examples directory](https://github.com/dshills/goflow/tree/main/examples) for complete workflow examples:

- **ETL Pipeline** (`examples/etl-pipeline.yaml`): Read, transform, load data
- **Multi-Server Integration** (`examples/multi-server.yaml`): Combine tools from different servers
- **Error Handling** (`examples/error-handling.yaml`): Retry logic and fallback paths
- **Parallel Processing** (`examples/parallel-batch.yaml`): Process multiple items concurrently
- **Conditional Logic** (`examples/conditional-workflow.yaml`): Branch based on data conditions
- **API Integration** (`examples/api-workflow.yaml`): Call external APIs and process responses

Happy workflow building! 🚀
