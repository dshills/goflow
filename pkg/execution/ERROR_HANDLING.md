# Enhanced Error Handling in GoFlow

This document describes the enhanced error handling system in GoFlow's execution engine, which provides comprehensive debugging context for workflow failures.

## Overview

GoFlow's enhanced error system captures rich debugging information when workflows fail, including:

- **Stack Traces**: Detailed runtime stack traces with file/line information
- **MCP Server Logs**: Recent logs from MCP servers leading up to errors
- **Node Execution Chain**: Complete path of node executions taken before failure
- **Variable Snapshots**: Values of all variables at the time of error
- **Error Classification**: Categorization with severity levels and recovery hints
- **Contextual Data**: Tool parameters, server IDs, and other relevant context

## Error Types

### Base Error Types

GoFlow categorizes errors into five types (defined in `pkg/domain/execution/types.go`):

1. **Validation Errors** (`ErrorTypeValidation`): Schema, parameter, or type validation failures
2. **Connection Errors** (`ErrorTypeConnection`): MCP server communication failures
3. **Execution Errors** (`ErrorTypeExecution`): Runtime failures (tool errors, resource issues)
4. **Data Errors** (`ErrorTypeData`): Transformation failures (JSONPath, type conversions)
5. **Timeout Errors** (`ErrorTypeTimeout`): Operations exceeding time limits

### Error Severity Levels

Each error is classified with a severity level:

- **Critical**: System failure, cannot continue (e.g., non-recoverable execution errors)
- **High**: Major feature broken (e.g., validation errors, data transformation failures)
- **Medium**: Degraded functionality (e.g., connection timeouts, recoverable errors)
- **Low**: Minor issue with workaround available

## Core Structures

### ExecutionError (Base)

The base error type from the domain layer:

```go
type ExecutionError struct {
    Type        ErrorType              // Error category
    Message     string                 // Human-readable description
    NodeID      types.NodeID           // Where error occurred
    StackTrace  string                 // Raw stack trace
    Context     map[string]interface{} // Additional context
    Recoverable bool                   // Can retry?
    Timestamp   time.Time              // When it occurred
}
```

### EnhancedExecutionError

Extended error with full debugging context:

```go
type EnhancedExecutionError struct {
    *ExecutionError
    MCPLogs             []MCPLogEntry          // MCP server logs
    NodeExecutionChain  []NodeExecutionStep    // Execution path
    VariableSnapshot    map[string]interface{} // Variable values
    DetailedStackTrace  []StackFrame           // Parsed stack trace
    ErrorClassification ErrorClassification    // Classification metadata
}
```

### MCPLogEntry

Represents a log entry from an MCP server:

```go
type MCPLogEntry struct {
    Timestamp time.Time              // When logged
    Level     string                 // debug, info, warn, error
    Message   string                 // Log message
    ServerID  string                 // Which server
    ToolName  string                 // Which tool (if applicable)
    Metadata  map[string]interface{} // Additional data
}
```

### NodeExecutionStep

A single step in the execution chain:

```go
type NodeExecutionStep struct {
    NodeID      types.NodeID           // Node identifier
    NodeType    string                 // Node type (start, transform, mcp_tool, etc.)
    Status      execution.NodeStatus   // Execution status
    StartedAt   time.Time              // Start time
    CompletedAt time.Time              // Completion time
    Duration    time.Duration          // How long it took
    Inputs      map[string]interface{} // Input values
    Outputs     map[string]interface{} // Output values
}
```

### StackFrame

A single frame in the stack trace:

```go
type StackFrame struct {
    Function string // Fully qualified function name
    File     string // Source file path
    Line     int    // Line number
    Package  string // Package name
}
```

## Usage Patterns

### Creating Enhanced Errors

#### Method 1: Using NewEnhancedError

```go
// Create base error
baseErr := &execution.ExecutionError{
    Type:        execution.ErrorTypeConnection,
    Message:     "MCP server timeout",
    NodeID:      "fetch_data",
    Recoverable: true,
    Timestamp:   time.Now(),
}

// Collect MCP logs
mcpLogs := []MCPLogEntry{
    {
        Timestamp: time.Now(),
        Level:     "error",
        Message:   "Connection timeout",
        ServerID:  "database-server",
    },
}

// Create enhanced error
enhanced := NewEnhancedError(baseErr, execution, mcpLogs)
```

#### Method 2: Using ErrorContextBuilder (Fluent API)

```go
enhanced := NewErrorContextBuilder(baseErr).
    WithExecution(exec).
    WithMCPLogs(mcpLogs).
    WithAdditionalStackFrames(10).
    Build()
```

#### Method 3: Wrapping Any Error

```go
// Wrap any error with enhanced context
externalErr := fmt.Errorf("database error")
enhanced := WrapWithEnhancedContext(
    externalErr,
    execution,
    "node_id",
    mcpLogs,
)
```

### Capturing Stack Traces

Stack traces are automatically captured when using the wrapper functions, but you can also capture them manually:

```go
// Capture current runtime stack
frames := CaptureRuntimeStack(0) // skip=0 includes current function

// Parse existing stack trace string
frames := ParseStackTrace(stackTraceString)
```

### Formatting Errors for Display

```go
// Format enhanced error for human-readable output
formatted := FormatEnhancedError(enhanced)
fmt.Println(formatted)
```

Output format:
```
=== EXECUTION ERROR ===
Type: connection
Severity: medium
Recoverable: true
Node: fetch_data
Message: MCP server timeout
Timestamp: 2025-01-05T10:30:00Z

Recovery Hint: Check MCP server connection and retry

=== EXECUTION PATH ===
1. start (start) - completed [100ms]
2. validate (transform) - completed [50ms]
3. fetch_data (mcp_tool) - failed [30s]

=== VARIABLES AT ERROR ===
  user_id = "123"
  batch_size = 100
  max_retries = 3

=== ERROR CONTEXT ===
  server_id: "database-server"
  tool_name: "query_users"
  timeout: "30s"

=== MCP SERVER LOGS ===
[10:29:30.000] [INFO] database-server: Connecting to server
[10:29:35.000] [DEBUG] database-server: Sending query request
[10:30:00.000] [ERROR] database-server: Connection timeout

=== STACK TRACE (Top 10 Frames) ===
  github.com/dshills/goflow/pkg/execution.executeNode
    /path/to/execution.go:123
  github.com/dshills/goflow/pkg/execution.(*Engine).Execute
    /path/to/engine.go:456
  ...
```

## Integration with Execution Engine

### In Node Executors

Wrap errors with enhanced context when node execution fails:

```go
func (e *Engine) executeMCPToolNode(...) error {
    result, err := server.InvokeTool(toolName, params)
    if err != nil {
        // Collect MCP logs
        logs := e.mcpLogCollector.CollectLogs(serverID, time.Now().Add(-5*time.Minute), 50)

        // Create enhanced error
        enhanced := WrapWithEnhancedContext(err, exec, nodeID, logs)

        // Log formatted error
        fmt.Println(FormatEnhancedError(enhanced))

        return enhanced.ExecutionError // Return base for domain consistency
    }
    return nil
}
```

### With Error Recovery

Check error classification to determine recovery strategy:

```go
if err != nil {
    enhanced := WrapWithEnhancedContext(err, exec, nodeID, logs)

    if enhanced.ErrorClassification.Recoverable {
        // Retry logic
        fmt.Printf("Retrying: %s\n", enhanced.ErrorClassification.RetryHint)
        // ... retry implementation
    } else {
        // Fatal error, log and fail
        fmt.Println(FormatEnhancedError(enhanced))
        return enhanced.ExecutionError
    }
}
```

### MCP Log Collection

Implement the MCPLogCollector interface to provide logs:

```go
type MCPLogCollector interface {
    CollectLogs(serverID string, since time.Time, limit int) ([]MCPLogEntry, error)
    CollectLogsForExecution(execID types.ExecutionID, serverID string) ([]MCPLogEntry, error)
}
```

Example implementation:

```go
type ServerLogCollector struct {
    servers map[string]*MCPServer
}

func (c *ServerLogCollector) CollectLogs(serverID string, since time.Time, limit int) ([]MCPLogEntry, error) {
    server := c.servers[serverID]
    if server == nil {
        return nil, nil
    }

    logs := []MCPLogEntry{}
    for _, log := range server.GetRecentLogs() {
        if log.Timestamp.After(since) {
            logs = append(logs, MCPLogEntry{
                Timestamp: log.Timestamp,
                Level:     log.Level,
                Message:   log.Message,
                ServerID:  serverID,
                ToolName:  log.ToolName,
                Metadata:  log.Metadata,
            })
        }
        if len(logs) >= limit {
            break
        }
    }

    return logs, nil
}
```

## Error Recovery Strategies

Based on error classification, apply appropriate recovery:

### Connection Errors (Medium Severity, Usually Recoverable)

```go
if enhanced.Type == execution.ErrorTypeConnection {
    // Exponential backoff retry
    for attempt := 1; attempt <= maxRetries; attempt++ {
        time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
        if err := retry(); err == nil {
            return nil
        }
    }
}
```

### Timeout Errors (Medium Severity, Recoverable)

```go
if enhanced.Type == execution.ErrorTypeTimeout {
    // Increase timeout and retry
    newTimeout := currentTimeout * 2
    return retryWithTimeout(newTimeout)
}
```

### Validation Errors (High Severity, Not Recoverable)

```go
if enhanced.Type == execution.ErrorTypeValidation {
    // Log detailed error and fail fast
    fmt.Println(FormatEnhancedError(enhanced))
    return enhanced.ExecutionError
}
```

### Data Errors (High Severity, Not Recoverable)

```go
if enhanced.Type == execution.ErrorTypeData {
    // Provide detailed transformation context
    fmt.Printf("Data transformation failed:\n")
    fmt.Printf("Expression: %s\n", enhanced.Context["expression"])
    fmt.Printf("Input: %v\n", enhanced.Context["input_value"])
    return enhanced.ExecutionError
}
```

## Best Practices

### 1. Always Capture Context

When creating errors, include relevant context:

```go
context := map[string]interface{}{
    "server_id":  serverID,
    "tool_name":  toolName,
    "parameters": params,
    "timeout":    timeout.String(),
}
```

### 2. Collect Recent MCP Logs

Capture logs from before the error occurred (e.g., last 5 minutes):

```go
logs := collector.CollectLogs(serverID, time.Now().Add(-5*time.Minute), 50)
```

### 3. Use Enhanced Errors for Debugging

Enhanced errors are primarily for debugging and logging:

```go
// Log enhanced error for debugging
enhanced := WrapWithEnhancedContext(err, exec, nodeID, logs)
fmt.Println(FormatEnhancedError(enhanced))

// Return base error to maintain domain boundaries
return enhanced.ExecutionError
```

### 4. Classify Errors Consistently

The classification system provides consistent severity and recovery hints:

```go
// Error classification is automatic
enhanced := NewEnhancedError(baseErr, exec, logs)
fmt.Printf("Severity: %s\n", enhanced.ErrorClassification.Severity)
fmt.Printf("Hint: %s\n", enhanced.ErrorClassification.RetryHint)
```

### 5. Preserve Stack Traces

Stack traces are captured automatically, but parse them for structured access:

```go
// Raw stack trace
fmt.Println(enhanced.StackTrace)

// Parsed stack frames
for _, frame := range enhanced.DetailedStackTrace {
    fmt.Printf("%s:%d - %s\n", frame.File, frame.Line, frame.Function)
}
```

## Testing Enhanced Errors

### Unit Tests

```go
func TestEnhancedError(t *testing.T) {
    baseErr := &execution.ExecutionError{
        Type:    execution.ErrorTypeConnection,
        Message: "test error",
    }

    exec, _ := execution.NewExecution("wf-1", "1.0", nil)
    logs := []MCPLogEntry{{Message: "test log"}}

    enhanced := NewEnhancedError(baseErr, exec, logs)

    assert.NotNil(t, enhanced)
    assert.Len(t, enhanced.MCPLogs, 1)
    assert.NotEmpty(t, enhanced.ErrorClassification.RetryHint)
}
```

### Integration Tests

```go
func TestErrorRecovery(t *testing.T) {
    // Simulate error during execution
    err := simulateConnectionError()
    enhanced := WrapWithEnhancedContext(err, exec, nodeID, logs)

    // Verify error details
    assert.Equal(t, execution.ErrorTypeConnection, enhanced.Type)
    assert.True(t, enhanced.Recoverable)

    // Test recovery
    if enhanced.ErrorClassification.Recoverable {
        err := retry()
        assert.NoError(t, err)
    }
}
```

## Future Enhancements

- **Error Aggregation**: Collect multiple errors during parallel execution
- **Error Metrics**: Track error rates and patterns for monitoring
- **Error Serialization**: JSON/YAML serialization for storage and transmission
- **Error Replay**: Reconstruct execution state from error context for debugging
- **Error Annotations**: User-defined annotations for business context
- **Error Webhooks**: Notify external systems of critical errors

## See Also

- `pkg/domain/execution/types.go` - Base error types
- `pkg/execution/error.go` - Enhanced error implementation
- `pkg/execution/error_test.go` - Comprehensive test suite
- `pkg/execution/error_example_test.go` - Usage examples
