# Data Model: GoFlow Domain Entities

**Feature**: GoFlow - Visual MCP Workflow Orchestrator
**Date**: 2025-11-05
**Purpose**: Define domain model following Domain-Driven Design principles with three core aggregates

## Architecture Overview

GoFlow follows Domain-Driven Design (DDD) with clear aggregate boundaries. The system is organized around three core aggregates:

1. **Workflow Aggregate**: Manages workflow definitions, structure, and validation
2. **Execution Aggregate**: Handles workflow execution, runtime state, and audit trails
3. **MCP Server Registry Aggregate**: Manages MCP server connections, tool discovery, and health

Cross-aggregate communication happens **only** through references (IDs), never direct object access.

---

## Aggregate 1: Workflow

### Root Entity: Workflow

**Purpose**: Represents a directed acyclic graph (DAG) of nodes and edges defining an automation workflow.

**Fields**:
- `ID` (WorkflowID): Unique identifier (UUID)
- `Name` (string): Human-readable workflow name (unique per user)
- `Version` (string): Semantic version (e.g., "1.0.0")
- `Description` (string): Human-readable description
- `Metadata` (WorkflowMetadata): Author, timestamps, tags
- `Variables` ([]Variable): Workflow-scoped variable definitions
- `ServerConfigs` ([]ServerConfig): MCP server configurations (references only, credentials in keyring)
- `Nodes` ([]Node): Workflow steps
- `Edges` ([]Edge): Connections between nodes

**Invariants**:
1. Must have exactly one Start node
2. Must have at least one End node
3. No circular dependencies in the graph (DAG property)
4. All node IDs must be unique within workflow
5. All variable names must be unique within workflow
6. All edges must reference valid node IDs
7. No orphaned nodes (all nodes reachable from Start)

**Operations**:
- `NewWorkflow(name, description)`: Create new empty workflow
- `AddNode(node)`: Add node with validation
- `RemoveNode(nodeID)`: Remove node and dependent edges
- `AddEdge(edge)`: Add edge with validation
- `RemoveEdge(edgeID)`: Remove edge
- `Validate()`: Check all invariants
- `ToYAML()`: Serialize to YAML
- `FromYAML(bytes)`: Parse from YAML

---

### Value Object: Node

**Purpose**: Represents a single step in a workflow.

**Types** (discriminated union):

1. **StartNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "start"
   - One per workflow (invariant enforced by Workflow)

2. **EndNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "end"
   - `ReturnValue` (Expression): Optional value to return

3. **MCPToolNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "mcp_tool"
   - `ServerID` (string): Reference to MCP server (from ServerConfigs)
   - `ToolName` (string): MCP tool to invoke
   - `Parameters` (map[string]Expression): Tool parameters (can reference variables)
   - `OutputVariable` (string): Variable to store result

4. **TransformNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "transform"
   - `InputVariable` (string): Source variable
   - `Expression` (Expression): Transformation to apply (JSONPath, template, expression)
   - `OutputVariable` (string): Target variable

5. **ConditionNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "condition"
   - `Condition` (Expression): Boolean expression
   - Two outgoing edges required (true/false branches)

6. **ParallelNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "parallel"
   - `Branches` ([][]NodeID): Subgraphs to execute concurrently
   - `MergeStrategy`: "wait_all" | "wait_any" | "wait_first"

7. **LoopNode**:
   - `ID` (NodeID): Unique identifier
   - `Type`: "loop"
   - `Collection` (string): Variable containing array/collection
   - `ItemVariable` (string): Variable name for current item
   - `Body` ([]NodeID): Subgraph to execute per item
   - `BreakCondition` (Expression): Optional early termination

**Validation Rules**:
- Node IDs must be unique within workflow
- Server IDs must reference existing ServerConfigs
- Variable references must exist in workflow Variables
- Expressions must be valid (syntax check)

---

### Value Object: Edge

**Purpose**: Defines execution flow between nodes.

**Fields**:
- `ID` (EdgeID): Unique identifier
- `FromNodeID` (NodeID): Source node
- `ToNodeID` (NodeID): Target node
- `Condition` (Expression): Optional condition (for branching from ConditionNode)
- `Label` (string): Human-readable label (e.g., "true", "false", "on error")

**Validation Rules**:
- From and To nodes must exist in workflow
- No self-loops (FromNodeID != ToNodeID)
- No duplicate edges (same From/To pair)
- Condition required if FromNode is ConditionNode
- No circular paths in graph

---

### Value Object: Variable

**Purpose**: Workflow-scoped data storage for passing values between nodes.

**Fields**:
- `Name` (string): Unique variable name (identifier)
- `Type` (string): Data type hint ("string", "number", "boolean", "object", "array", "any")
- `DefaultValue` (interface{}): Optional default value
- `Description` (string): Human-readable description

**Validation Rules**:
- Name must be valid identifier (alphanumeric + underscore)
- Name must be unique within workflow
- DefaultValue must match Type if provided

---

### Value Object: ServerConfig

**Purpose**: Configuration for connecting to an MCP server (credentials stored separately in keyring).

**Fields**:
- `ID` (string): Unique server identifier within workflow
- `Name` (string): Human-readable server name
- `Command` (string): Executable command (e.g., "npx", "python")
- `Args` ([]string): Command arguments
- `Transport` (string): "stdio" | "sse" | "http"
- `Env` (map[string]string): Environment variables
- `CredentialRef` (string): Reference to keyring entry (if needed)

**Note**: Credentials never stored in workflow. CredentialRef points to keyring entry by ID.

---

### Value Object: WorkflowMetadata

**Purpose**: Descriptive information about workflow.

**Fields**:
- `Author` (string): Workflow creator
- `Created` (time.Time): Creation timestamp
- `LastModified` (time.Time): Last modification timestamp
- `Tags` ([]string): Categorization tags
- `Icon` (string): Emoji or icon for TUI display

---

## Aggregate 2: Execution

### Root Entity: Execution

**Purpose**: Represents a single run of a workflow with specific inputs.

**Fields**:
- `ID` (ExecutionID): Unique identifier (UUID)
- `WorkflowID` (WorkflowID): Reference to workflow being executed
- `WorkflowVersion` (string): Workflow version at execution time
- `Status` (ExecutionStatus): "pending" | "running" | "completed" | "failed" | "cancelled"
- `StartedAt` (time.Time): Execution start timestamp
- `CompletedAt` (time.Time): Execution completion timestamp (nil if running)
- `Error` (ExecutionError): Error details if failed
- `Context` (ExecutionContext): Current execution state
- `NodeExecutions` ([]NodeExecution): History of node executions
- `ReturnValue` (interface{}): Final output from End node

**Invariants**:
1. WorkflowID must reference valid workflow
2. NodeExecutions maintain topological order
3. Context mutations are append-only (for audit trail)
4. Status transitions follow state machine
5. CompletedAt set only when Status is terminal (completed/failed/cancelled)

**State Machine**:
```
pending → running → {completed, failed, cancelled}
```

**Operations**:
- `NewExecution(workflowID, inputs)`: Create new execution
- `Start()`: Begin execution (pending → running)
- `ExecuteNode(nodeID)`: Execute single node
- `Complete(returnValue)`: Mark as completed
- `Fail(error)`: Mark as failed
- `Cancel()`: Cancel execution
- `GetAuditTrail()`: Return complete execution history

---

### Value Object: ExecutionContext

**Purpose**: Runtime state during workflow execution.

**Fields**:
- `CurrentNodeID` (NodeID): Node currently executing (nil if not running)
- `Variables` (map[string]interface{}): Current variable values
- `VariableHistory` ([]VariableSnapshot): Append-only log of variable changes
- `ExecutionTrace` ([]TraceEntry): Execution path taken through workflow

**Operations**:
- `GetVariable(name)`: Retrieve current value
- `SetVariable(name, value)`: Update value (appends to history)
- `RecordTrace(nodeID, event)`: Log execution event

---

### Value Object: NodeExecution

**Purpose**: Record of a single node execution within workflow run.

**Fields**:
- `ID` (NodeExecutionID): Unique identifier
- `ExecutionID` (ExecutionID): Parent execution reference
- `NodeID` (NodeID): Node that was executed
- `NodeType` (string): Type of node (for quick filtering)
- `Status` (NodeStatus): "pending" | "running" | "completed" | "failed" | "skipped"
- `StartedAt` (time.Time): Node execution start
- `CompletedAt` (time.Time): Node execution completion
- `Inputs` (map[string]interface{}): Node input values
- `Outputs` (map[string]interface{}): Node output values
- `Error` (NodeError): Error details if failed
- `RetryCount` (int): Number of retries attempted
- `Duration` (time.Duration): Execution time

**Invariants**:
- CompletedAt > StartedAt
- Error set only if Status == "failed"
- RetryCount >= 0

---

### Value Object: VariableSnapshot

**Purpose**: Point-in-time capture of variable value (for audit trail).

**Fields**:
- `Timestamp` (time.Time): When value changed
- `NodeExecutionID` (NodeExecutionID): Which node made the change
- `VariableName` (string): Variable that changed
- `OldValue` (interface{}): Previous value (nil if first set)
- `NewValue` (interface{}): New value

**Immutable**: Once created, never modified (append-only audit log).

---

### Value Object: ExecutionError

**Purpose**: Detailed error information for failed executions.

**Fields**:
- `Type` (ErrorType): "validation" | "connection" | "execution" | "data" | "timeout"
- `Message` (string): Human-readable error message
- `NodeID` (NodeID): Node where error occurred
- `StackTrace` (string): Go stack trace
- `Context` (map[string]interface{}): Additional error context
- `Recoverable` (bool): Whether retry might succeed

---

## Aggregate 3: MCP Server Registry

### Root Entity: MCPServer

**Purpose**: Represents a registered MCP server with connection state and available tools.

**Fields**:
- `ID` (ServerID): Unique identifier
- `Name` (string): Human-readable server name
- `Command` (string): Executable command
- `Args` ([]string): Command arguments
- `Transport` (Transport): Connection transport
- `Connection` (Connection): Current connection state
- `Tools` ([]Tool): Available tools discovered from server
- `HealthStatus` (HealthStatus): "unknown" | "healthy" | "unhealthy" | "disconnected"
- `LastHealthCheck` (time.Time): Last health check timestamp
- `Metadata` (ServerMetadata): Version, capabilities, etc.

**Invariants**:
1. Server ID unique within registry
2. Tools list updated only when connected
3. HealthStatus updated by periodic health checks
4. Connection state machine followed

**Operations**:
- `NewMCPServer(id, command, args, transport)`: Create server registration
- `Connect()`: Establish connection
- `Disconnect()`: Close connection
- `DiscoverTools()`: Query available tools from server
- `InvokeTool(toolName, params)`: Execute tool
- `HealthCheck()`: Verify server responsiveness
- `Reconnect()`: Attempt reconnection

---

### Value Object: Transport

**Purpose**: Connection transport configuration.

**Types** (discriminated union):

1. **StdioTransport**:
   - `Type`: "stdio"
   - `Command`: Executable path
   - `Args`: Command arguments
   - `Env`: Environment variables

2. **SSETransport**:
   - `Type`: "sse"
   - `URL`: Server-sent events endpoint
   - `Headers`: HTTP headers

3. **HTTPTransport**:
   - `Type`: "http"
   - `BaseURL`: Base URL for JSON-RPC calls
   - `Headers`: HTTP headers
   - `Timeout`: Request timeout

---

### Value Object: Connection

**Purpose**: Current connection state to MCP server.

**Fields**:
- `State` (ConnectionState): "disconnected" | "connecting" | "connected" | "failed"
- `Process` (Process): OS process handle (for stdio transport)
- `Client` (Client): MCP protocol client
- `ConnectedAt` (time.Time): When connection established
- `LastActivity` (time.Time): Last communication timestamp
- `ErrorCount` (int): Consecutive error count
- `RetryBackoff` (time.Duration): Current retry delay

**State Machine**:
```
disconnected → connecting → {connected, failed}
connected → disconnected
failed → connecting (with backoff)
```

---

### Value Object: Tool

**Purpose**: MCP tool schema discovered from server.

**Fields**:
- `Name` (string): Tool identifier
- `Description` (string): Human-readable description
- `InputSchema` (JSONSchema): Input parameter schema
- `OutputSchema` (JSONSchema): Output result schema
- `Examples` ([]ToolExample): Usage examples

**Validation**:
- Name must be valid identifier
- InputSchema must be valid JSON Schema
- Tool invocations validated against InputSchema

---

### Value Object: ToolExample

**Purpose**: Example usage of a tool (for documentation and testing).

**Fields**:
- `Description` (string): What this example demonstrates
- `Inputs` (map[string]interface{}): Example input parameters
- `ExpectedOutput` (interface{}): Expected result

---

### Value Object: HealthStatus

**Purpose**: Server health state.

**Values**:
- `Unknown`: Health not yet checked
- `Healthy`: Server responding normally
- `Unhealthy`: Server responding with errors
- `Disconnected`: Connection lost

**Metadata**:
- `LastCheck` (time.Time): Last health check timestamp
- `ResponseTime` (time.Duration): Last ping response time
- `ErrorMessage` (string): Error details if unhealthy

---

### Value Object: ServerMetadata

**Purpose**: Server capabilities and version information.

**Fields**:
- `ProtocolVersion` (string): MCP protocol version supported
- `ServerVersion` (string): Server implementation version
- `Capabilities` ([]string): Supported features (e.g., "streaming", "batch")
- `Vendor` (string): Server provider/maintainer

---

## Cross-Aggregate References

### Workflow → MCP Server Registry

- `Workflow.ServerConfigs[].ID` references `MCPServer.ID`
- Workflow stores **only** server ID, not full server details
- Actual connection/tools fetched from registry at execution time

### Execution → Workflow

- `Execution.WorkflowID` references `Workflow.ID`
- `Execution.WorkflowVersion` captures version at execution time
- Execution stores workflow snapshot (for historical accuracy)

### Execution → MCP Server Registry

- Node executions invoke tools via server registry
- Execution logs server responses but doesn't own server state

---

## Repository Interfaces

### WorkflowRepository

```go
type WorkflowRepository interface {
    Save(workflow *Workflow) error
    FindByID(id WorkflowID) (*Workflow, error)
    FindByName(name string) (*Workflow, error)
    List() ([]*Workflow, error)
    Delete(id WorkflowID) error
}
```

**Implementation**: Filesystem (YAML files)

---

### ExecutionRepository

```go
type ExecutionRepository interface {
    Save(execution *Execution) error
    FindByID(id ExecutionID) (*Execution, error)
    FindByWorkflowID(workflowID WorkflowID) ([]*Execution, error)
    List(filters ExecutionFilters) ([]*Execution, error)
    Delete(id ExecutionID) error
    SaveNodeExecution(nodeExec *NodeExecution) error
    SaveVariableSnapshot(snapshot *VariableSnapshot) error
}
```

**Implementation**: SQLite database

---

### ServerRepository

```go
type ServerRepository interface {
    Save(server *MCPServer) error
    FindByID(id ServerID) (*MCPServer, error)
    List() ([]*MCPServer, error)
    Delete(id ServerID) error
    UpdateHealthStatus(id ServerID, status HealthStatus) error
}
```

**Implementation**: SQLite database + in-memory cache

---

## Data Flow Example

**User Story 1**: Create and Execute Simple Workflow

1. **Workflow Creation** (Workflow Aggregate):
   ```
   workflow := NewWorkflow("data-pipeline", "ETL workflow")
   workflow.AddNode(StartNode{})
   workflow.AddNode(MCPToolNode{ServerID: "fs", ToolName: "read_file"})
   workflow.AddNode(TransformNode{Expression: "jq(.data)"})
   workflow.AddNode(MCPToolNode{ServerID: "fs", ToolName: "write_file"})
   workflow.AddNode(EndNode{})
   workflow.AddEdge(Edge{From: "start", To: "read"})
   // ... add more edges
   workflow.Validate()  // Check invariants
   workflowRepo.Save(workflow)
   ```

2. **Execution** (Execution Aggregate):
   ```
   execution := NewExecution(workflow.ID, inputs)
   execution.Start()

   // Execute each node in topological order
   for node in topologicalSort(workflow) {
       nodeExec := execution.ExecuteNode(node.ID)

       if node is MCPToolNode {
           // Cross-aggregate reference
           server := serverRegistry.FindByID(node.ServerID)
           result := server.InvokeTool(node.ToolName, node.Parameters)
           nodeExec.Outputs = result
       }

       execution.Context.SetVariable(node.OutputVariable, nodeExec.Outputs)
       executionRepo.SaveNodeExecution(nodeExec)
   }

   execution.Complete()
   executionRepo.Save(execution)
   ```

3. **Audit Trail** (Execution Aggregate):
   ```
   trail := execution.GetAuditTrail()
   // Returns: all NodeExecutions, VariableSnapshots, complete trace
   ```

---

## Validation Rules Summary

### Workflow Aggregate

- ✓ Exactly one Start node
- ✓ At least one End node
- ✓ No circular dependencies (DAG)
- ✓ Unique node IDs
- ✓ Unique variable names
- ✓ Valid edge references
- ✓ No orphaned nodes

### Execution Aggregate

- ✓ Valid workflow reference
- ✓ Topological execution order
- ✓ Append-only audit trail
- ✓ Valid state transitions
- ✓ Timestamps consistent

### MCP Server Registry Aggregate

- ✓ Unique server IDs
- ✓ Valid connection state machine
- ✓ MCP-compliant tool schemas
- ✓ Tools updated only when connected

---

## Performance Considerations

### Workflow Aggregate

- **Validation**: Graph traversal O(N+E) where N=nodes, E=edges
- **Optimization**: Cache validation results, invalidate on changes

### Execution Aggregate

- **Audit Trail**: Append-only log grows linearly with execution length
- **Optimization**: Page from SQLite, don't load entire history in memory

### MCP Server Registry Aggregate

- **Tool Discovery**: Expensive (requires server round-trip)
- **Optimization**: Cache tool schemas, refresh on server version change

---

This data model provides clear aggregate boundaries, maintains invariants, and supports all user stories while adhering to constitutional DDD principles.
