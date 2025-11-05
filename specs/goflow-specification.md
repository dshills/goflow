# GoFlow: Visual MCP Server Orchestrator

**Status:** Draft  
**Author:** Darrell Hills  
**Created:** 2025-11-05  
**Last Updated:** 2025-11-05  

---

## Executive Summary

GoFlow is a visual workflow orchestration system for Model Context Protocol (MCP) servers that enables developers and AI systems to chain multiple MCP tools into sophisticated, reusable workflows with conditional logic, data transformation, and parallel execution capabilities. Built as a standalone Go binary with both a terminal user interface (TUI) and programmatic API, GoFlow addresses the current gap in MCP tooling where each server operates in isolation without composability.

---

## Problem Statement

### Current State

The Model Context Protocol ecosystem is rapidly growing, with hundreds of MCP servers providing diverse capabilities (database access, API integration, file systems, AI tools, etc.). However, several critical limitations exist:

1. **Isolation**: Each MCP server operates independently with no standard mechanism for composition
2. **Manual Orchestration**: Developers must manually coordinate multiple MCP calls through LLM prompting or custom code
3. **No Reusability**: Common multi-step workflows must be recreated each time they're needed
4. **Limited Logic**: No declarative way to express conditional flows, loops, or error handling across MCP servers
5. **Poor Observability**: No centralized view of multi-server operations or debugging capabilities
6. **Context Loss**: Data transformation between MCP calls requires LLM intervention, wasting tokens and time

### Impact

- **Developer Productivity**: Complex tasks requiring multiple MCP servers take 5-10x longer than necessary
- **Reliability**: Manual orchestration introduces human error and inconsistent results
- **Cost**: Excessive LLM token usage for simple data passing between MCP servers
- **Adoption**: Steep learning curve prevents non-technical users from leveraging MCP ecosystems

### Success Criteria

GoFlow will be considered successful when:

1. Users can create multi-server workflows in under 5 minutes
2. Workflow execution is 10x faster than manual LLM-mediated orchestration
3. 80% reduction in LLM tokens for multi-step MCP operations
4. Non-technical users can successfully create and run basic workflows
5. The project achieves 500+ GitHub stars within 6 months of launch

---

## Goals and Non-Goals

### Goals

**Primary Goals:**
1. Enable visual composition of MCP servers into workflows
2. Provide conditional branching, loops, and error handling
3. Support data transformation between MCP tool calls without LLM intervention
4. Offer both interactive TUI and programmatic execution modes
5. Maintain full compatibility with the MCP protocol specification
6. Enable workflow sharing and reuse across teams

**Secondary Goals:**
1. Provide real-time execution monitoring and debugging
2. Support workflow versioning and rollback
3. Enable parallel execution of independent workflow branches
4. Offer workflow templates for common use cases
5. Integrate with existing CI/CD pipelines

### Non-Goals

1. **Not a replacement for MCP servers** - GoFlow orchestrates existing MCP servers, it doesn't provide tools itself
2. **Not an LLM framework** - GoFlow focuses on deterministic workflow execution, not AI model management
3. **Not a cloud service** - GoFlow runs locally as a CLI tool (though workflows can reference remote MCP servers)
4. **Not a general workflow engine** - GoFlow is purpose-built for MCP; it won't orchestrate non-MCP systems
5. **Not a visual programming IDE** - The TUI provides workflow building, not full code development

---

## Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                      GoFlow CLI                          │
│  ┌────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │    TUI     │  │  Workflow    │  │   Execution    │  │
│  │  Builder   │◄─┤   Engine     │◄─┤    Runtime     │  │
│  └────────────┘  └──────────────┘  └────────────────┘  │
│         │               │                    │           │
│         ▼               ▼                    ▼           │
│  ┌────────────────────────────────────────────────────┐ │
│  │           Workflow Definition Storage              │ │
│  │              (YAML/JSON formats)                   │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
         ┌────────────────────────────────────┐
         │      MCP Protocol Layer            │
         │  (stdio, SSE, HTTP transports)     │
         └────────────────────────────────────┘
                          │
         ┌────────────────┼────────────────────┐
         ▼                ▼                    ▼
   ┌─────────┐      ┌─────────┐         ┌─────────┐
   │  MCP    │      │  MCP    │         │  MCP    │
   │ Server  │      │ Server  │   ...   │ Server  │
   │    A    │      │    B    │         │    N    │
   └─────────┘      └─────────┘         └─────────┘
```

### Domain Model

#### Core Aggregates

**1. Workflow Aggregate**
- **Workflow Root**: Unique identifier, name, version, metadata
- **Nodes**: Individual steps in the workflow (MCP tool calls, transformations, conditions)
- **Edges**: Connections between nodes defining execution flow
- **Variables**: Workflow-scoped data store for passing values between nodes
- **Invariants**: 
  - Must have exactly one start node
  - All nodes except terminal nodes must have outgoing edges
  - No circular dependencies in synchronous paths
  - Variable names must be unique within workflow scope

**2. Execution Aggregate**
- **Execution Root**: Workflow ID, execution ID, start time, status
- **Execution Context**: Current state, variable values, execution trace
- **Node Executions**: Individual node execution records with inputs/outputs/errors
- **Invariants**:
  - Execution must reference valid workflow
  - Node executions maintain topological order
  - Variable mutations are append-only for audit trail

**3. MCP Server Registry Aggregate**
- **Server Root**: Server identifier, connection configuration, available tools
- **Tool Schemas**: Input/output schemas for each tool
- **Connection State**: Active connections, health status
- **Invariants**:
  - Server identifiers must be unique
  - Connection credentials are encrypted at rest
  - Tool schemas match MCP protocol specification

#### Value Objects

- **NodeConfiguration**: Immutable configuration for each node type
- **DataTransformation**: JSONPath or template-based transformations
- **ExecutionResult**: Output data, status, execution time, error details
- **WorkflowTemplate**: Reusable workflow patterns with parameterization

### Node Types

GoFlow supports the following node types in workflows:

1. **MCP Tool Node**: Executes a specific MCP tool
   - Input: Tool name, parameters (can reference variables)
   - Output: Tool result stored in variable
   
2. **Transform Node**: Applies data transformation
   - Input: Source variable, transformation expression
   - Output: Transformed data in target variable
   
3. **Condition Node**: Branches based on boolean expression
   - Input: Condition expression (supports comparisons, logical operators)
   - Output: Two paths (true/false)
   
4. **Loop Node**: Iterates over collection
   - Input: Collection variable, loop body subgraph
   - Output: Aggregated results
   
5. **Parallel Node**: Executes multiple branches concurrently
   - Input: Multiple subgraph branches
   - Output: Synchronized results from all branches
   
6. **Start Node**: Entry point (system-generated)
   
7. **End Node**: Exit point with optional return value

### Workflow Definition Format

Workflows are stored as YAML files following this schema:

```yaml
version: "1.0"
name: "workflow-name"
description: "Workflow description"
metadata:
  author: "author-name"
  created: "2025-11-05T00:00:00Z"
  tags: ["tag1", "tag2"]

variables:
  - name: "var1"
    type: "string"
    default: "default-value"

servers:
  - id: "server1"
    name: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    
nodes:
  - id: "start"
    type: "start"
    
  - id: "read_file"
    type: "mcp_tool"
    server: "server1"
    tool: "read_file"
    parameters:
      path: "${workflow.input.file_path}"
    output: "file_contents"
    
  - id: "transform"
    type: "transform"
    input: "${file_contents}"
    expression: "jq(.content | fromjson)"
    output: "parsed_json"
    
  - id: "condition"
    type: "condition"
    condition: "${parsed_json.status} == 'active'"
    
  - id: "end_success"
    type: "end"
    return: "${parsed_json}"
    
  - id: "end_failure"
    type: "end"
    return: 
      error: "Invalid status"

edges:
  - from: "start"
    to: "read_file"
    
  - from: "read_file"
    to: "transform"
    
  - from: "transform"
    to: "condition"
    
  - from: "condition"
    to: "end_success"
    when: "true"
    
  - from: "condition"
    to: "end_failure"
    when: "false"
```

### Execution Model

**Execution Phases:**

1. **Validation**: 
   - Parse workflow definition
   - Validate against schema
   - Verify all referenced servers are reachable
   - Check for circular dependencies
   
2. **Initialization**:
   - Start MCP server connections
   - Initialize execution context
   - Set up variable store
   
3. **Execution**:
   - Topological sort of nodes
   - Execute nodes in order
   - Handle conditions and loops
   - Manage parallel branches
   - Transform and pass data
   
4. **Completion**:
   - Aggregate results
   - Close server connections
   - Store execution logs
   - Return final output

**Execution Guarantees:**

- **Atomicity**: Workflow executions are atomic units (succeed or fail completely)
- **Idempotency**: Safe to retry failed executions with same inputs
- **Observability**: Full execution trace with inputs/outputs for each node
- **Error Handling**: Graceful failure with detailed error context

### Terminal User Interface (TUI)

Built using your existing `goterm` library, the TUI provides:

**Main Views:**

1. **Workflow Explorer**
   - List all available workflows
   - Search and filter
   - Quick execute or edit

2. **Workflow Builder**
   - Visual graph editor
   - Drag-and-drop node placement
   - Connection drawing
   - Property inspector panel
   - Real-time validation

3. **Execution Monitor**
   - Live execution visualization
   - Node-by-node progress
   - Variable inspector
   - Error highlighting
   - Execution logs

4. **Server Registry**
   - List configured MCP servers
   - Test connections
   - Browse available tools
   - View tool schemas

**TUI Navigation:**
- Vim-style keybindings (h/j/k/l navigation)
- Tab to switch between panels
- Enter to select/edit
- ESC to go back
- ? for context-sensitive help

### CLI Interface

```bash
# Initialize new workflow
goflow init <workflow-name>

# Open TUI builder
goflow edit <workflow-name>

# Execute workflow
goflow run <workflow-name> [--input input.json] [--watch] [--debug]

# List workflows
goflow list [--tags tag1,tag2]

# Validate workflow
goflow validate <workflow-name>

# Export workflow
goflow export <workflow-name> [--format yaml|json]

# Import workflow
goflow import <file-path>

# Manage servers
goflow server add <server-id> <command> [args...]
goflow server list
goflow server test <server-id>
goflow server remove <server-id>

# Execute workflow from stdin (for CI/CD)
cat workflow.yaml | goflow run --stdin

# Generate template
goflow template <template-name>
```

### Data Transformation Engine

GoFlow includes a lightweight data transformation system:

**Supported Expressions:**

1. **JSONPath**: Query and extract from JSON
   ```
   $.users[0].email
   $.items[?(@.price < 100)]
   ```

2. **Template Strings**: String interpolation
   ```
   "Hello ${user.name}, your order ${order.id} is ready"
   ```

3. **jq-style**: JSON transformation (subset of jq)
   ```
   jq(.items | map(.price) | add)
   ```

4. **Conditional**: Ternary operations
   ```
   ${count > 10 ? "many" : "few"}
   ```

### Error Handling Strategy

**Error Types:**

1. **Validation Errors**: Caught before execution starts
   - Invalid workflow syntax
   - Missing required parameters
   - Type mismatches

2. **Connection Errors**: MCP server communication failures
   - Server unreachable
   - Authentication failures
   - Protocol errors

3. **Execution Errors**: Runtime failures
   - Tool execution errors
   - Timeout exceeded
   - Resource exhaustion

4. **Data Errors**: Transformation or type errors
   - Invalid JSONPath
   - Type conversion failures
   - Missing variables

**Error Recovery:**

- Configurable retry policies per node
- Fallback paths for error conditions
- Automatic rollback for transactional operations
- Detailed error context for debugging

### Security Model

1. **Server Credentials**:
   - Stored in system keyring (keychain on macOS)
   - Never in workflow definitions
   - Referenced by ID only

2. **Workflow Execution**:
   - Runs with user's permissions
   - No privilege escalation
   - Sandboxed MCP server processes

3. **Data Privacy**:
   - Execution logs can be filtered for sensitive data
   - Variable encryption at rest (optional)
   - Secure cleanup of temporary data

4. **Workflow Sharing**:
   - Workflows contain no secrets
   - Server configurations separated from workflow logic
   - Clear warnings about external dependencies

---

## Technical Specifications

### Technology Stack

**Core Language**: Go 1.21+

**Key Libraries**:
- `github.com/dshills/goterm` - Terminal UI framework
- `gopkg.in/yaml.v3` - Workflow definition parsing
- `github.com/tidwall/gjson` - JSON path queries
- `github.com/expr-lang/expr` - Expression evaluation
- `golang.org/x/sync/errgroup` - Parallel execution
- Native MCP client (based on your `craftMCP` project)

**Storage**:
- Local filesystem for workflows (YAML/JSON)
- SQLite for execution history and metadata
- System keyring for credentials

### MCP Protocol Integration

GoFlow implements a full MCP client supporting:

1. **Transport Protocols**:
   - stdio (subprocess communication)
   - Server-Sent Events (SSE)
   - HTTP with JSON-RPC

2. **Protocol Features**:
   - Tool discovery and schema introspection
   - Resource management
   - Prompt templates
   - Sampling/completion (when applicable)

3. **Connection Management**:
   - Connection pooling
   - Automatic reconnection
   - Health checks
   - Graceful shutdown

### Performance Targets

- **Workflow Validation**: < 100ms for workflows with < 100 nodes
- **Execution Startup**: < 500ms to begin execution
- **Node Execution Overhead**: < 10ms per node (excluding MCP tool time)
- **Parallel Execution**: Support 50+ concurrent branches
- **Memory Usage**: < 100MB base + 10MB per active MCP server
- **Workflow Storage**: < 1KB per workflow definition

### Scalability Considerations

1. **Large Workflows**: Support workflows with 1000+ nodes
2. **High-Frequency Execution**: Handle 100+ concurrent workflow executions
3. **Long-Running Workflows**: Support executions lasting hours with checkpointing
4. **Large Data Sets**: Stream data through nodes to avoid memory limits

---

## User Experience

### Target Personas

**Primary Persona - AI Developer**
- Builds MCP servers and tooling
- Needs to test multi-server interactions
- Values speed and flexibility
- Comfortable with terminal interfaces

**Secondary Persona - Automation Engineer**
- Creates workflows for team productivity
- May not write MCP servers but uses many
- Needs reliability and observability
- Prefers visual tools but accepts TUI

**Tertiary Persona - Power User**
- Non-developer using AI tools
- Wants to automate repetitive tasks
- Needs templates and examples
- Requires gentle learning curve

### User Journeys

**Journey 1: Creating First Workflow**

1. User installs GoFlow: `go install github.com/dshills/goflow@latest`
2. User adds their first MCP server: `goflow server add myserver npx @vendor/server`
3. User creates workflow from template: `goflow template data-pipeline`
4. User opens TUI editor: `goflow edit data-pipeline`
5. User customizes nodes and connections visually
6. User tests workflow: `goflow run data-pipeline --debug`
7. User shares workflow: `goflow export data-pipeline > workflow.yaml`

**Journey 2: Debugging Failed Workflow**

1. Workflow execution fails
2. User sees error in terminal with node ID
3. User opens TUI in debug mode: `goflow edit workflow --exec last`
4. Failed node is highlighted in red
5. User inspects node input/output in property panel
6. User identifies issue in data transformation
7. User fixes expression and re-runs
8. Workflow succeeds

**Journey 3: Production Automation**

1. Developer creates workflow in TUI
2. Developer validates: `goflow validate production-sync`
3. Developer exports: `goflow export production-sync > prod.yaml`
4. Commit workflow to git repository
5. CI/CD pipeline runs: `goflow run prod.yaml --input env.json`
6. Workflow executes successfully in automated environment

### Documentation Strategy

1. **README**: Quick start, installation, basic concepts
2. **User Guide**: Comprehensive TUI and CLI documentation
3. **Workflow Cookbook**: 20+ example workflows with explanations
4. **MCP Integration Guide**: How to make MCP servers work with GoFlow
5. **API Reference**: For programmatic usage
6. **Video Tutorials**: Screen recordings of common workflows

---

## Implementation Plan

### Phase 1: Foundation (Weeks 1-4)

**Deliverables:**
- Core workflow domain model
- YAML workflow parser and validator
- Basic MCP client implementation
- SQLite storage layer
- CLI scaffolding

**Success Criteria:**
- Can parse and validate workflow YAML
- Can connect to MCP servers via stdio
- Can store workflows in database

### Phase 2: Execution Engine (Weeks 5-8)

**Deliverables:**
- Workflow execution runtime
- Node execution implementations (MCP tool, transform, condition)
- Variable store and context management
- Error handling and retry logic
- Execution logging

**Success Criteria:**
- Can execute linear workflows (no loops/parallel)
- Can transform data between nodes
- Can handle errors gracefully
- Full execution trace available

### Phase 3: TUI Development (Weeks 9-12)

**Deliverables:**
- Workflow explorer view
- Visual workflow builder
- Execution monitor
- Server registry UI
- Keyboard navigation

**Success Criteria:**
- Can create simple workflows in TUI
- Can execute and monitor workflows
- Can manage MCP servers
- Vim-style navigation works

### Phase 4: Advanced Features (Weeks 13-16)

**Deliverables:**
- Loop and parallel node types
- Advanced data transformations
- Workflow templates
- CI/CD integration examples
- Performance optimizations

**Success Criteria:**
- Can execute complex branching workflows
- Parallel execution works correctly
- Performance targets met
- Template library with 10+ examples

### Phase 5: Polish & Launch (Weeks 17-20)

**Deliverables:**
- Comprehensive documentation
- Video tutorials
- Example workflow repository
- Performance testing and optimization
- Security audit
- Public release

**Success Criteria:**
- Documentation complete
- Zero critical bugs
- Security review passed
- Ready for public announcement

---

## Alternatives Considered

### Alternative 1: Web-Based UI Instead of TUI

**Pros:**
- More familiar to non-technical users
- Richer visual capabilities
- Easier to share screenshots/demos

**Cons:**
- Requires running a server process
- Adds complexity and dependencies
- Slower to start and use
- Not suitable for SSH/remote usage

**Decision**: TUI chosen for simplicity, speed, and alignment with developer workflows

### Alternative 2: Embed Scripting Language (Lua/JavaScript)

**Pros:**
- More powerful transformations
- Familiar to developers
- Extensive standard libraries

**Cons:**
- Security concerns with arbitrary code execution
- Harder to validate and analyze workflows
- Steeper learning curve
- Potential for workflow portability issues

**Decision**: Limited expression language chosen for safety and simplicity

### Alternative 3: Build as MCP Server Instead of Orchestrator

**Pros:**
- Fits naturally into existing MCP ecosystem
- Could be called by LLMs directly

**Cons:**
- Doesn't solve the composition problem
- Adds another layer of indirection
- Limits standalone utility

**Decision**: Standalone orchestrator chosen to solve composition directly

### Alternative 4: Use Existing Workflow Engines (Temporal, Airflow)

**Pros:**
- Proven at scale
- Rich feature sets
- Large communities

**Cons:**
- Heavy infrastructure requirements
- Not MCP-native
- Complex setup for simple use cases
- Poor developer experience for MCP workflows

**Decision**: Purpose-built tool chosen for optimal MCP developer experience

---

## Dependencies and Integration

### Internal Dependencies

- Your existing `goterm` library for TUI
- Your `craftMCP` project as MCP client foundation
- Your experience with `second-opinion` MCP server architecture

### External Dependencies

**Runtime Dependencies:**
- Go 1.21+ runtime
- SQLite (embedded, no separate install)
- System keyring (OS-provided)

**MCP Server Dependencies:**
- Users must install their own MCP servers
- GoFlow doesn't bundle any MCP servers
- Recommendation: Start with popular servers like filesystem, fetch, sqlite

### Integration Points

1. **CI/CD Systems**: 
   - GitHub Actions
   - GitLab CI
   - Jenkins
   - Any system that can run CLI tools

2. **Development Tools**:
   - Claude Code (can generate workflows)
   - VS Code (workflow YAML editing)
   - Git (version control for workflows)

3. **MCP Ecosystem**:
   - All compliant MCP servers
   - MCP specification updates
   - Community server registry

---

## Risk Assessment

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| MCP protocol changes breaking compatibility | Medium | High | Track spec closely, version workflows, maintain backward compatibility |
| Performance issues with large workflows | Low | Medium | Early performance testing, streaming data, lazy evaluation |
| TUI complexity impacts usability | Medium | High | Extensive user testing, simple default view, progressive disclosure |
| Data transformation security vulnerabilities | Low | High | Sandboxed evaluation, expression allowlist, security audit |

### Product Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| MCP ecosystem not mature enough | Low | High | Works with existing servers, provides value even with few servers |
| Users prefer web UI over TUI | Medium | Medium | Excellent TUI experience, target developer audience first |
| Competition from other orchestrators | Medium | Medium | Be first to market, focus on developer experience |
| Insufficient adoption | Medium | High | Strong launch strategy, integrate with popular MCP servers |

### Mitigation Strategies

1. **Early User Feedback**: Alpha release to MCP community after Phase 3
2. **Incremental Development**: Each phase delivers usable functionality
3. **Community Building**: Engage with MCP Discord, Twitter, GitHub
4. **Documentation First**: Write docs during development, not after
5. **Performance Budget**: Set and monitor performance targets from day 1

---

## Success Metrics

### Quantitative Metrics

**Adoption Metrics:**
- GitHub stars: Target 500 in 6 months
- Downloads: Target 5,000 in first year
- Active users: Target 1,000 monthly active users
- Workflow repositories: Target 50 public workflow repos

**Usage Metrics:**
- Average workflow size: 10-20 nodes
- Average execution time: < 30 seconds
- Success rate: > 95% of executions complete successfully
- Workflow reuse: 30% of executions use shared workflows

**Performance Metrics:**
- TUI responsiveness: < 16ms frame time (60 FPS)
- Startup time: < 500ms cold start
- Memory usage: < 200MB for typical workflows
- CPU usage: < 10% during monitoring

### Qualitative Metrics

- User testimonials and case studies
- Integration into popular projects
- Conference talks and blog posts
- Community contributions (PRs, issues, discussions)
- Mentioned in MCP ecosystem documentation

---

## Open Questions

1. **Workflow Sharing**: Should we build a central registry for workflows, or rely on GitHub/gists?

2. **Visual Complexity**: At what node count should we warn users the workflow is too complex?

3. **Debugging Depth**: Should we support stepping through workflows node-by-node like a debugger?

4. **Type System**: Should workflows have a stronger type system for variables, or keep it dynamic?

5. **Versioning**: How should we handle workflow versioning and breaking changes?

6. **Cloud Integration**: Should Phase 6 add cloud execution capabilities, or keep it strictly local?

7. **LLM Integration**: Should GoFlow include a mode where an LLM can generate workflows from natural language?

---

## Appendices

### Appendix A: Workflow YAML JSON Schema

(Full JSON Schema definition would go here for validation)

### Appendix B: Example Workflows

**Example 1: ETL Pipeline**
```yaml
name: "etl-pipeline"
description: "Extract data from API, transform, load to database"
# ... (full workflow definition)
```

**Example 2: Code Review Automation**
```yaml
name: "code-review"
description: "Fetch PR, analyze with LLM, post comments"
# ... (full workflow definition)
```

**Example 3: Multi-Source Data Aggregation**
```yaml
name: "data-aggregator"
description: "Query multiple APIs in parallel, merge results"
# ... (full workflow definition)
```

### Appendix C: MCP Server Compatibility Matrix

| Server | Status | Notes |
|--------|--------|-------|
| @modelcontextprotocol/server-filesystem | ✅ Tested | Full compatibility |
| @modelcontextprotocol/server-fetch | ✅ Tested | Full compatibility |
| @modelcontextprotocol/server-sqlite | ✅ Tested | Full compatibility |
| second-opinion | ✅ Tested | Your own server |
| ... | ... | ... |

### Appendix D: Performance Benchmarks

(Would include detailed performance test results once implemented)

### Appendix E: Security Considerations Checklist

- [ ] Credentials never stored in workflow files
- [ ] Expression evaluation sandboxed
- [ ] No arbitrary code execution
- [ ] Input validation on all user data
- [ ] Secure cleanup of temporary files
- [ ] Audit logging for sensitive operations
- [ ] Rate limiting on MCP calls
- [ ] Timeout protection against infinite loops

---

## Glossary

- **Aggregate**: A cluster of domain objects treated as a single unit (DDD concept)
- **Edge**: A connection between two nodes in a workflow graph
- **MCP**: Model Context Protocol - standard for AI-tool integration
- **Node**: A single step in a workflow (tool call, transformation, condition, etc.)
- **Orchestration**: Coordination of multiple services/tools to accomplish a task
- **TUI**: Terminal User Interface - text-based interactive interface
- **Workflow**: A directed graph of operations to accomplish a task

---

## References

1. Model Context Protocol Specification: https://modelcontextprotocol.io/
2. Domain-Driven Design (Eric Evans)
3. Your existing projects: goterm, craftMCP, second-opinion
4. GitHub spec-kit: https://github.com/github/spec-kit

---

## Changelog

- **2025-11-05**: Initial specification created
