# GoFlow Example Workflows

This directory contains example workflows demonstrating GoFlow's key features and patterns. Each workflow is a complete, documented example that you can use as a starting point for your own workflows.

## Quick Start

1. **Install GoFlow**: Follow the [quickstart guide](../specs/001-goflow-spec-review/quickstart.md)
2. **Register MCP Servers**: Each example lists required servers
3. **Copy Example**: Copy the workflow YAML to `~/.goflow/workflows/`
4. **Run Workflow**: `goflow run <workflow-name>`

## Available Examples

### 1. Simple Pipeline (`simple-pipeline.yaml`)

**Demonstrates**: Basic read-transform-write pattern with filesystem operations

**Use Case**: ETL pipeline that reads JSON, calculates sum of prices, writes result

**Key Features**:
- MCP tool nodes (read_file, write_file)
- Transform node with jq-style expressions
- Variable substitution
- Linear execution flow

**Prerequisites**:
```bash
goflow server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp
echo '{"data": [{"price": 10.5}, {"price": 20.3}, {"price": 5.2}]}' > /tmp/input.json
```

**Run**:
```bash
cp examples/simple-pipeline.yaml ~/.goflow/workflows/data-pipeline.yaml
goflow run data-pipeline
cat /tmp/output.txt  # Should show: Total: 36.0
```

**Related Tutorial**: [Quickstart Guide](../specs/001-goflow-spec-review/quickstart.md)

---

### 2. Conditional Workflow (`conditional-workflow.yaml`)

**Demonstrates**: Conditional branching based on data conditions

**Use Case**: Check file size and compress large files before upload, upload small files directly

**Key Features**:
- Condition node for branching logic
- Multiple execution paths
- Conditional edges with labels
- Path merging

**Prerequisites**:
```bash
goflow server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp
goflow server add compression npx -y @example/compression-server
goflow server add upload npx -y @example/upload-server
```

**Pattern**:
```
Read File → Check Size → [Large File] → Compress → Upload
                      → [Small File] → Upload Direct
```

**When to Use**:
- Different processing based on data values
- Optional workflow steps
- Multi-path workflows with shared endpoints
- Business rule implementation

---

### 3. Error Handling (`error-handling.yaml`)

**Demonstrates**: Retry policies and graceful degradation

**Use Case**: Fetch data from API with automatic retry on connection errors, fallback to backup API if primary fails

**Key Features**:
- Retry policy configuration
- Exponential backoff
- Error-specific retry conditions
- Fallback paths
- Default data when all sources fail

**Prerequisites**:
```bash
goflow server add http npx -y @example/http-client
```

**Retry Configuration**:
```yaml
retry:
  max_attempts: 3
  backoff: "exponential"
  initial_delay: "1s"
  max_delay: "30s"
  on: ["connection_error", "timeout", "service_unavailable"]
  skip_on: ["bad_request", "unauthorized", "not_found"]
```

**When to Use**:
- External API integrations
- Network-dependent operations
- Critical data fetching
- Resilient workflow design

---

### 4. Parallel Batch Processing (`parallel-batch.yaml`)

**Demonstrates**: Concurrent execution for processing multiple items

**Use Case**: Process multiple files in parallel with transformation and validation

**Key Features**:
- Parallel node for concurrent execution
- Loop node with parallel flag
- Max concurrency limit
- Result aggregation
- Merge strategies

**Prerequisites**:
```bash
goflow server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp
goflow server add validator npx -y @example/validator-server
mkdir -p /tmp/batch /tmp/batch-output
```

**Note**: This is a **Phase 8 feature** (not yet implemented). The example serves as:
- Design reference for parallel execution
- Documentation of planned capabilities
- Template for future implementation

**When to Use**:
- Processing multiple independent items
- I/O-bound operations (file reading, API calls)
- Performance optimization
- Batch data processing

---

## Workflow Pattern Categories

### Data Transformation Patterns

| Pattern | Example | Use Case |
|---------|---------|----------|
| Extract-Transform-Load | `simple-pipeline.yaml` | Data pipeline, format conversion |
| Filter and Transform | `parallel-batch.yaml` | Data cleaning, validation |
| Aggregate and Summarize | `parallel-batch.yaml` | Reporting, analytics |

### Control Flow Patterns

| Pattern | Example | Use Case |
|---------|---------|----------|
| Sequential | `simple-pipeline.yaml` | Linear workflows |
| Conditional Branch | `conditional-workflow.yaml` | Decision-based routing |
| Parallel Split | `parallel-batch.yaml` | Concurrent processing |
| Loop | `parallel-batch.yaml` | Iterative processing |

### Error Handling Patterns

| Pattern | Example | Use Case |
|---------|---------|----------|
| Retry with Backoff | `error-handling.yaml` | Network failures |
| Fallback Path | `error-handling.yaml` | Alternative data sources |
| Default Value | `error-handling.yaml` | Graceful degradation |

## Creating Your Own Workflows

### 1. Start with an Example

Copy the example that most closely matches your use case:

```bash
cp examples/simple-pipeline.yaml ~/.goflow/workflows/my-workflow.yaml
```

### 2. Customize for Your Needs

Edit the workflow file:
- Update `name` and `description`
- Modify `variables` with your parameters
- Configure `servers` for your MCP servers
- Adjust `nodes` for your processing steps
- Update `edges` for your execution flow

### 3. Validate Before Running

```bash
goflow validate my-workflow
```

### 4. Test with Sample Data

```bash
goflow run my-workflow --input test-data.json --debug
```

## Common Modifications

### Adding a New Node

```yaml
nodes:
  - id: "my_new_node"
    type: "mcp_tool"  # or "transform", "condition", etc.
    server: "server-id"
    tool: "tool-name"
    parameters:
      param1: "${variable1}"
    output: "result_variable"
    description: "What this node does"
```

Don't forget to add edges:
```yaml
edges:
  - from: "previous_node"
    to: "my_new_node"
  - from: "my_new_node"
    to: "next_node"
```

### Adding Variables

```yaml
variables:
  - name: "my_variable"
    type: "string"  # or "number", "boolean", "object"
    default: "default_value"
    description: "What this variable is for"
```

Use in nodes: `"${my_variable}"`

### Adding Conditions

```yaml
nodes:
  - id: "check_condition"
    type: "condition"
    condition: "${value} > 100"

edges:
  - from: "check_condition"
    to: "true_path"
    condition: "true"
  - from: "check_condition"
    to: "false_path"
    condition: "false"
```

### Adding Retry Logic

```yaml
nodes:
  - id: "risky_operation"
    type: "mcp_tool"
    # ... node configuration ...
    retry:
      max_attempts: 3
      backoff: "exponential"
      initial_delay: "1s"
      on: ["connection_error", "timeout"]
```

## Testing Your Workflows

### 1. Syntax Validation

```bash
# Check YAML syntax
yamllint my-workflow.yaml

# Validate workflow structure
goflow validate my-workflow
```

### 2. Dry Run

```bash
# Run with debug output
goflow run my-workflow --debug

# Watch execution in real-time
goflow run my-workflow --watch
```

### 3. Variable Testing

```bash
# Override variables
goflow run my-workflow --var input_file=/tmp/test.json --var output_file=/tmp/result.txt

# Use input file
goflow run my-workflow --input test-variables.json
```

## Best Practices

### 1. Workflow Design

- ✅ **Use descriptive IDs**: `read_user_data` not `node1`
- ✅ **Add descriptions**: Help others (and future you) understand the workflow
- ✅ **Single responsibility**: Each node should do one thing well
- ✅ **Fail fast**: Validate inputs early in the workflow
- ✅ **Handle errors**: Add retry logic and fallback paths

### 2. Variables

- ✅ **Provide defaults**: Make workflows runnable out-of-the-box
- ✅ **Use meaningful names**: `api_endpoint` not `url1`
- ✅ **Document expected types**: Use type field and description
- ✅ **Group related variables**: Keep input/output variables together

### 3. Server Configuration

- ✅ **Document prerequisites**: List required servers in comments
- ✅ **Use version pins**: Specify exact MCP server versions in production
- ✅ **Test connections**: `goflow server test` before running workflows
- ✅ **Store credentials safely**: Use keyring for sensitive data

### 4. Error Handling

- ✅ **Add retry policies**: For network-dependent operations
- ✅ **Provide fallbacks**: Alternative data sources or default values
- ✅ **Use specific error types**: Retry only on retryable errors
- ✅ **Log failures**: Capture error context for debugging

### 5. Performance

- ✅ **Use parallel execution**: When operations are independent
- ✅ **Limit concurrency**: Set max_parallel to prevent resource exhaustion
- ✅ **Minimize transformations**: Use native MCP tool outputs when possible
- ✅ **Cache results**: Enable caching for idempotent workflows

## Troubleshooting

### Workflow Won't Validate

```bash
# Get detailed validation errors
goflow validate my-workflow --verbose
```

Common issues:
- Circular dependencies (A → B → C → A)
- Orphaned nodes (no edges connecting)
- Invalid variable references
- Missing start or end node

### Workflow Execution Fails

```bash
# Run with debug logging
goflow run my-workflow --debug

# View execution logs
goflow logs <execution-id>
```

Common issues:
- Server not registered: `goflow server add ...`
- Server connection failed: `goflow server test ...`
- Invalid tool parameters
- Variable type mismatch

### Transform Errors

```bash
# Test expression in isolation
echo '{"data": [1,2,3]}' | goflow test-expression 'jq(.data | add)'
```

Common issues:
- Invalid JSONPath syntax
- Type mismatches in expressions
- Undefined variables in templates

## Contributing Examples

Have a useful workflow pattern? We'd love to include it!

1. Create a well-documented workflow YAML
2. Test thoroughly with `scripts/test-quickstart.sh`
3. Add usage instructions to this README
4. Submit a pull request

Example should include:
- Clear description and use case
- Complete prerequisites
- Working server configurations
- Sample input/output
- Comments explaining key sections

## Resources

- **Full Documentation**: [docs/](../docs/)
- **Quickstart Guide**: [quickstart.md](../specs/001-goflow-spec-review/quickstart.md)
- **Node Type Reference**: [docs/nodes.md](../docs/nodes.md) (coming soon)
- **Expression Language**: [docs/expressions.md](../docs/expressions.md) (coming soon)
- **CLI Reference**: [CLI commands in quickstart](../specs/001-goflow-spec-review/quickstart.md#cli-command-reference)

## Need Help?

- **Issue Tracker**: https://github.com/dshills/goflow/issues
- **Discussions**: https://github.com/dshills/goflow/discussions
- **Built-in Help**: `goflow help <command>`

---

**Happy workflow building!** 🚀
