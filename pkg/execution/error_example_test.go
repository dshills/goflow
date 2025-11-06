package execution_test

import (
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
)

// ExampleEnhancedExecutionError demonstrates creating and formatting an enhanced error
// with full debugging context including MCP logs, execution chain, and variable snapshots.
func ExampleEnhancedExecutionError() {
	// Create base execution error
	baseErr := &execution.ExecutionError{
		Type:       execution.ErrorTypeConnection,
		Message:    "MCP server connection timeout",
		NodeID:     "fetch_data",
		StackTrace: "goroutine 1 [running]:\ngithub.com/dshills/goflow/pkg/execution.executeNode()\n\t/path/to/file.go:123",
		Context: map[string]interface{}{
			"server_id": "database-server",
			"tool_name": "query_users",
			"timeout":   "30s",
		},
		Recoverable: true,
		Timestamp:   time.Now(),
	}

	// Create execution context
	exec, _ := execution.NewExecution("user-workflow", "1.0", map[string]interface{}{
		"user_id":     "123",
		"batch_size":  100,
		"max_retries": 3,
	})

	// Add node executions to show the path taken
	node1 := &execution.NodeExecution{
		NodeID:      "start",
		NodeType:    "start",
		Status:      execution.NodeStatusCompleted,
		StartedAt:   time.Now().Add(-5 * time.Second),
		CompletedAt: time.Now().Add(-4 * time.Second),
	}
	exec.AddNodeExecution(node1)

	node2 := &execution.NodeExecution{
		NodeID:      "validate_input",
		NodeType:    "transform",
		Status:      execution.NodeStatusCompleted,
		StartedAt:   time.Now().Add(-4 * time.Second),
		CompletedAt: time.Now().Add(-3 * time.Second),
		Inputs:      map[string]interface{}{"user_id": "123"},
		Outputs:     map[string]interface{}{"valid": true},
	}
	exec.AddNodeExecution(node2)

	node3 := &execution.NodeExecution{
		NodeID:      "fetch_data",
		NodeType:    "mcp_tool",
		Status:      execution.NodeStatusFailed,
		StartedAt:   time.Now().Add(-3 * time.Second),
		CompletedAt: time.Now(),
		Inputs: map[string]interface{}{
			"query":      "SELECT * FROM users WHERE id = ?",
			"parameters": []interface{}{"123"},
		},
	}
	exec.AddNodeExecution(node3)

	// Collect MCP server logs
	mcpLogs := []runtimeexec.MCPLogEntry{
		{
			Timestamp: time.Now().Add(-3 * time.Second),
			Level:     "info",
			Message:   "Connecting to database-server",
			ServerID:  "database-server",
		},
		{
			Timestamp: time.Now().Add(-2 * time.Second),
			Level:     "debug",
			Message:   "Sending query_users tool request",
			ServerID:  "database-server",
			ToolName:  "query_users",
			Metadata: map[string]interface{}{
				"query_length": 45,
			},
		},
		{
			Timestamp: time.Now().Add(-1 * time.Second),
			Level:     "warn",
			Message:   "Server response delayed, waiting...",
			ServerID:  "database-server",
		},
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Connection timeout after 30s",
			ServerID:  "database-server",
			ToolName:  "query_users",
			Metadata: map[string]interface{}{
				"elapsed": "30.2s",
			},
		},
	}

	// Create enhanced error with all context
	enhanced := runtimeexec.NewEnhancedError(baseErr, exec, mcpLogs)

	// Format and display the error
	formatted := runtimeexec.FormatEnhancedError(enhanced)
	fmt.Println(formatted)

	// Output contains rich debugging information:
	// - Error classification and severity
	// - Node execution chain showing the path
	// - Variable values at time of error
	// - MCP server logs leading to failure
	// - Stack trace for debugging
	// - Recovery hints
}

// ExampleErrorContextBuilder demonstrates using the fluent API to build enhanced errors.
func ExampleErrorContextBuilder() {
	// Create base error
	baseErr := &execution.ExecutionError{
		Type:        execution.ErrorTypeData,
		Message:     "JSONPath expression failed",
		NodeID:      "transform_response",
		Recoverable: false,
		Timestamp:   time.Now(),
	}

	// Create execution
	exec, _ := execution.NewExecution("api-workflow", "1.0", map[string]interface{}{
		"api_url":  "https://api.example.com/users",
		"response": map[string]interface{}{"data": []interface{}{}},
	})

	// Build enhanced error using fluent API
	enhanced := runtimeexec.NewErrorContextBuilder(baseErr).
		WithExecution(exec).
		WithAdditionalStackFrames(10).
		Build()

	// Access error details
	fmt.Printf("Error Type: %s\n", enhanced.Type)
	fmt.Printf("Severity: %s\n", enhanced.ErrorClassification.Severity)
	fmt.Printf("Recoverable: %v\n", enhanced.Recoverable)
	fmt.Printf("Variables: %d captured\n", len(enhanced.VariableSnapshot))
	fmt.Printf("Retry Hint: %s\n", enhanced.ErrorClassification.RetryHint)

	// Output:
	// Error Type: data
	// Severity: high
	// Recoverable: false
	// Variables: 2 captured
	// Retry Hint: Verify data transformation expressions and input data
}

// ExampleWrapWithEnhancedContext demonstrates wrapping any error with enhanced context.
func ExampleWrapWithEnhancedContext() {
	// Create execution
	exec, _ := execution.NewExecution("data-pipeline", "1.0", map[string]interface{}{
		"source": "database",
		"target": "cache",
	})

	// Simulate an error from external code
	externalErr := fmt.Errorf("database connection refused")

	// Collect relevant MCP logs
	mcpLogs := []runtimeexec.MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Connection refused on port 5432",
			ServerID:  "postgres-server",
		},
	}

	// Wrap with enhanced context
	enhanced := runtimeexec.WrapWithEnhancedContext(
		externalErr,
		exec,
		"connect_db",
		mcpLogs,
	)

	// Now we have full debugging context
	fmt.Printf("Error: %s\n", enhanced.Message)
	fmt.Printf("Node: %s\n", enhanced.NodeID)
	fmt.Printf("MCP Logs: %d entries\n", len(enhanced.MCPLogs))
	fmt.Printf("Has Stack Trace: %v\n", enhanced.StackTrace != "")
	fmt.Printf("Recovery: %s\n", enhanced.ErrorClassification.RetryHint)

	// Output:
	// Error: database connection refused
	// Node: connect_db
	// MCP Logs: 1 entries
	// Has Stack Trace: true
	// Recovery: Review error details and adjust workflow
}

// ExampleMCPLogEntry demonstrates creating MCP log entries for error context.
func ExampleMCPLogEntry() {
	// Create log entries from MCP server activity
	logs := []runtimeexec.MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "debug",
			Message:   "Tool invocation started",
			ServerID:  "api-server",
			ToolName:  "fetch_data",
			Metadata: map[string]interface{}{
				"endpoint": "/api/v1/data",
				"method":   "GET",
			},
		},
		{
			Timestamp: time.Now().Add(1 * time.Second),
			Level:     "error",
			Message:   "HTTP 500 Internal Server Error",
			ServerID:  "api-server",
			ToolName:  "fetch_data",
			Metadata: map[string]interface{}{
				"status_code":   500,
				"response_time": "1.2s",
			},
		},
	}

	// These logs can be attached to enhanced errors for debugging
	for _, log := range logs {
		fmt.Printf("[%s] %s: %s\n", log.Level, log.ServerID, log.Message)
	}

	// Output:
	// [debug] api-server: Tool invocation started
	// [error] api-server: HTTP 500 Internal Server Error
}

// ExampleErrorClassification demonstrates error classification for recovery decisions.
func ExampleErrorClassification() {
	tests := []execution.ErrorType{
		execution.ErrorTypeValidation,
		execution.ErrorTypeConnection,
		execution.ErrorTypeTimeout,
		execution.ErrorTypeData,
		execution.ErrorTypeExecution,
	}

	for _, errorType := range tests {
		err := &execution.ExecutionError{
			Type:        errorType,
			Message:     "test error",
			Recoverable: errorType == execution.ErrorTypeConnection || errorType == execution.ErrorTypeTimeout,
		}

		enhanced := runtimeexec.NewEnhancedError(err, nil, nil)
		classification := enhanced.ErrorClassification

		fmt.Printf("%s: severity=%s, recoverable=%v\n",
			classification.Category,
			classification.Severity,
			classification.Recoverable)
	}

	// Output:
	// validation: severity=high, recoverable=false
	// connection: severity=medium, recoverable=true
	// timeout: severity=medium, recoverable=true
	// data: severity=high, recoverable=false
	// execution: severity=critical, recoverable=false
}

// ExampleCaptureRuntimeStack demonstrates capturing stack traces for debugging.
func ExampleCaptureRuntimeStack() {
	// Capture current stack (skip=0 means include this function)
	frames := runtimeexec.CaptureRuntimeStack(0)

	// Display top 3 frames
	for i := 0; i < 3 && i < len(frames); i++ {
		frame := frames[i]
		fmt.Printf("Frame %d: %s at %s:%d\n", i+1, frame.Function, frame.File, frame.Line)
	}

	// Stack traces are automatically captured in ExecutionError.StackTrace
	// and parsed into DetailedStackTrace in EnhancedExecutionError
}
