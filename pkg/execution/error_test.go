package execution

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureRuntimeStack(t *testing.T) {
	frames := CaptureRuntimeStack(0)

	assert.NotEmpty(t, frames, "Should capture stack frames")
	assert.Greater(t, len(frames), 0, "Should have at least one frame")

	// First frame should be this test function
	firstFrame := frames[0]
	assert.Contains(t, firstFrame.Function, "TestCaptureRuntimeStack")
	assert.NotEmpty(t, firstFrame.File)
	assert.Greater(t, firstFrame.Line, 0)
}

func TestParseStackTrace(t *testing.T) {
	stackTrace := `github.com/dshills/goflow/pkg/execution.TestFunction(0x1234567)
	/path/to/file.go:42 +0x123
github.com/dshills/goflow/pkg/workflow.OtherFunction()
	/path/to/other.go:100 +0x456`

	frames := ParseStackTrace(stackTrace)

	require.Len(t, frames, 2, "Should parse 2 frames")

	// Check first frame
	assert.Contains(t, frames[0].Function, "TestFunction")
	assert.Contains(t, frames[0].File, "/path/to/file.go")
	assert.Equal(t, 42, frames[0].Line)

	// Check second frame
	assert.Contains(t, frames[1].Function, "OtherFunction")
	assert.Contains(t, frames[1].File, "/path/to/other.go")
	assert.Equal(t, 100, frames[1].Line)
}

func TestParseStackTrace_Empty(t *testing.T) {
	frames := ParseStackTrace("")
	assert.Nil(t, frames, "Empty stack trace should return nil")
}

func TestNewEnhancedError(t *testing.T) {
	// Create base error
	baseErr := &execution.ExecutionError{
		Type:        execution.ErrorTypeExecution,
		Message:     "test error",
		NodeID:      "node1",
		StackTrace:  "test stack trace",
		Context:     map[string]interface{}{"key": "value"},
		Recoverable: true,
		Timestamp:   time.Now(),
	}

	// Create execution context
	exec, err := execution.NewExecution("wf-123", "1.0", map[string]interface{}{
		"var1": "value1",
		"var2": 42,
	})
	require.NoError(t, err)

	// Add node executions
	nodeExec := &execution.NodeExecution{
		NodeID:    "node1",
		NodeType:  "transform",
		Status:    execution.NodeStatusCompleted,
		StartedAt: time.Now().Add(-1 * time.Second),
		Inputs:    map[string]interface{}{"input": "test"},
		Outputs:   map[string]interface{}{"output": "result"},
	}
	nodeExec.CompletedAt = time.Now()
	exec.AddNodeExecution(nodeExec)

	// Create MCP logs
	mcpLogs := []MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "connection failed",
			ServerID:  "server1",
			ToolName:  "tool1",
		},
	}

	// Create enhanced error
	enhanced := NewEnhancedError(baseErr, exec, mcpLogs)

	// Verify enhanced error
	assert.NotNil(t, enhanced)
	assert.Equal(t, baseErr, enhanced.ExecutionError)
	assert.Len(t, enhanced.MCPLogs, 1)
	assert.Equal(t, "connection failed", enhanced.MCPLogs[0].Message)
	assert.Len(t, enhanced.NodeExecutionChain, 1)
	assert.Equal(t, types.NodeID("node1"), enhanced.NodeExecutionChain[0].NodeID)
	assert.Contains(t, enhanced.VariableSnapshot, "var1")
	assert.Contains(t, enhanced.VariableSnapshot, "var2")
	assert.NotEmpty(t, enhanced.ErrorClassification)
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name            string
		errorType       execution.ErrorType
		recoverable     bool
		expectedSev     ErrorSeverity
		expectRetryHint bool
	}{
		{
			name:            "validation error",
			errorType:       execution.ErrorTypeValidation,
			recoverable:     false,
			expectedSev:     SeverityHigh,
			expectRetryHint: true,
		},
		{
			name:            "connection error",
			errorType:       execution.ErrorTypeConnection,
			recoverable:     true,
			expectedSev:     SeverityMedium,
			expectRetryHint: true,
		},
		{
			name:            "timeout error",
			errorType:       execution.ErrorTypeTimeout,
			recoverable:     true,
			expectedSev:     SeverityMedium,
			expectRetryHint: true,
		},
		{
			name:            "data error",
			errorType:       execution.ErrorTypeData,
			recoverable:     false,
			expectedSev:     SeverityHigh,
			expectRetryHint: true,
		},
		{
			name:            "recoverable execution error",
			errorType:       execution.ErrorTypeExecution,
			recoverable:     true,
			expectedSev:     SeverityMedium,
			expectRetryHint: true,
		},
		{
			name:            "critical execution error",
			errorType:       execution.ErrorTypeExecution,
			recoverable:     false,
			expectedSev:     SeverityCritical,
			expectRetryHint: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &execution.ExecutionError{
				Type:        tt.errorType,
				Message:     "test error",
				Recoverable: tt.recoverable,
			}

			classification := classifyError(err)

			assert.Equal(t, tt.errorType, classification.Category)
			assert.Equal(t, tt.expectedSev, classification.Severity)
			assert.Equal(t, tt.recoverable, classification.Recoverable)
			if tt.expectRetryHint {
				assert.NotEmpty(t, classification.RetryHint)
			}
		})
	}
}

func TestFormatEnhancedError(t *testing.T) {
	// Create base error
	baseErr := &execution.ExecutionError{
		Type:        execution.ErrorTypeConnection,
		Message:     "connection failed",
		NodeID:      "node1",
		StackTrace:  "test stack",
		Context:     map[string]interface{}{"server": "server1"},
		Recoverable: true,
		Timestamp:   time.Now(),
	}

	// Create execution
	exec, _ := execution.NewExecution("wf-123", "1.0", map[string]interface{}{
		"var1": "value1",
	})

	// Add node execution
	nodeExec := &execution.NodeExecution{
		NodeID:    "node1",
		NodeType:  "mcp_tool",
		Status:    execution.NodeStatusFailed,
		StartedAt: time.Now(),
	}
	nodeExec.CompletedAt = time.Now()
	exec.AddNodeExecution(nodeExec)

	// Create MCP logs
	mcpLogs := []MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "timeout",
			ServerID:  "server1",
		},
	}

	enhanced := NewEnhancedError(baseErr, exec, mcpLogs)
	formatted := FormatEnhancedError(enhanced)

	// Verify output contains expected sections
	assert.Contains(t, formatted, "=== EXECUTION ERROR ===")
	assert.Contains(t, formatted, "Type: connection")
	assert.Contains(t, formatted, "Severity: medium")
	assert.Contains(t, formatted, "Node: node1")
	assert.Contains(t, formatted, "Message: connection failed")
	assert.Contains(t, formatted, "=== EXECUTION PATH ===")
	assert.Contains(t, formatted, "=== VARIABLES AT ERROR ===")
	assert.Contains(t, formatted, "=== ERROR CONTEXT ===")
	assert.Contains(t, formatted, "=== MCP SERVER LOGS ===")
	assert.Contains(t, formatted, "Recovery Hint:")
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: "null",
		},
		{
			name:     "short string",
			value:    "test",
			expected: `"test"`,
		},
		{
			name:     "long string",
			value:    strings.Repeat("a", 150),
			expected: "... (truncated, 150 chars)",
		},
		{
			name:     "number",
			value:    42,
			expected: "42",
		},
		{
			name:     "boolean",
			value:    true,
			expected: "true",
		},
		{
			name:     "map",
			value:    map[string]interface{}{"key": "value"},
			expected: "key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.value)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestErrorContextBuilder(t *testing.T) {
	// Create base error
	baseErr := &execution.ExecutionError{
		Type:        execution.ErrorTypeExecution,
		Message:     "test error",
		NodeID:      "node1",
		Recoverable: true,
		Timestamp:   time.Now(),
	}

	// Create execution
	exec, _ := execution.NewExecution("wf-123", "1.0", nil)

	// Create MCP logs
	mcpLogs := []MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "test log",
			ServerID:  "server1",
		},
	}

	// Build enhanced error using builder
	enhanced := NewErrorContextBuilder(baseErr).
		WithExecution(exec).
		WithMCPLogs(mcpLogs).
		WithAdditionalStackFrames(5).
		Build()

	assert.NotNil(t, enhanced)
	assert.Equal(t, baseErr, enhanced.ExecutionError)
	assert.Len(t, enhanced.MCPLogs, 1)
	assert.NotNil(t, enhanced.VariableSnapshot)
}

func TestWrapWithEnhancedContext(t *testing.T) {
	// Create execution
	exec, _ := execution.NewExecution("wf-123", "1.0", map[string]interface{}{
		"var1": "value1",
	})

	// Create MCP logs
	mcpLogs := []MCPLogEntry{
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "test log",
			ServerID:  "server1",
		},
	}

	t.Run("wrap standard error", func(t *testing.T) {
		err := errors.New("standard error")
		enhanced := WrapWithEnhancedContext(err, exec, "node1", mcpLogs)

		assert.NotNil(t, enhanced)
		assert.Contains(t, enhanced.Message, "standard error")
		assert.Equal(t, types.NodeID("node1"), enhanced.NodeID)
		assert.Len(t, enhanced.MCPLogs, 1)
		assert.NotEmpty(t, enhanced.StackTrace)
	})

	t.Run("wrap execution error", func(t *testing.T) {
		baseErr := &execution.ExecutionError{
			Type:        execution.ErrorTypeConnection,
			Message:     "connection error",
			NodeID:      "node2",
			Recoverable: true,
			Timestamp:   time.Now(),
		}

		enhanced := WrapWithEnhancedContext(baseErr, exec, "node2", mcpLogs)

		assert.NotNil(t, enhanced)
		assert.Equal(t, baseErr, enhanced.ExecutionError)
		assert.Equal(t, execution.ErrorTypeConnection, enhanced.Type)
		assert.Len(t, enhanced.MCPLogs, 1)
	})
}

func TestBuildNodeExecutionChain(t *testing.T) {
	nodeExecutions := []*execution.NodeExecution{
		{
			NodeID:      "start",
			NodeType:    "start",
			Status:      execution.NodeStatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: time.Now().Add(100 * time.Millisecond),
			Inputs:      map[string]interface{}{},
			Outputs:     map[string]interface{}{"started": true},
		},
		{
			NodeID:      "node1",
			NodeType:    "transform",
			Status:      execution.NodeStatusCompleted,
			StartedAt:   time.Now().Add(100 * time.Millisecond),
			CompletedAt: time.Now().Add(200 * time.Millisecond),
			Inputs:      map[string]interface{}{"input": "test"},
			Outputs:     map[string]interface{}{"output": "result"},
		},
	}

	chain := buildNodeExecutionChain(nodeExecutions)

	assert.Len(t, chain, 2)
	assert.Equal(t, types.NodeID("start"), chain[0].NodeID)
	assert.Equal(t, "start", chain[0].NodeType)
	assert.Equal(t, types.NodeID("node1"), chain[1].NodeID)
	assert.Equal(t, "transform", chain[1].NodeType)
}

func TestMCPLogEntry_JSON(t *testing.T) {
	log := MCPLogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Message:   "test message",
		ServerID:  "server1",
		ToolName:  "tool1",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	// Test that it can be JSON marshaled
	_, err := log.Timestamp.MarshalJSON()
	assert.NoError(t, err)
}

func TestNodeExecutionStep_JSON(t *testing.T) {
	step := NodeExecutionStep{
		NodeID:      "node1",
		NodeType:    "transform",
		Status:      execution.NodeStatusCompleted,
		StartedAt:   time.Now(),
		CompletedAt: time.Now().Add(1 * time.Second),
		Duration:    1 * time.Second,
		Inputs:      map[string]interface{}{"input": "test"},
		Outputs:     map[string]interface{}{"output": "result"},
	}

	// Test that it can be JSON marshaled
	_, err := step.StartedAt.MarshalJSON()
	assert.NoError(t, err)
}

func TestErrorClassification_Severity(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{SeverityCritical, "critical"},
		{SeverityHigh, "high"},
		{SeverityMedium, "medium"},
		{SeverityLow, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.severity))
		})
	}
}

func TestWrapToolError_WithEnhancedContext(t *testing.T) {
	// Test that WrapToolError creates proper error
	toolErr := &MCPToolError{
		ServerID:    "server1",
		ToolName:    "tool1",
		Message:     "tool failed",
		Recoverable: true,
		Context: map[string]interface{}{
			"detail": "timeout",
		},
	}

	execErr := WrapToolError("node1", "server1", "tool1", toolErr, map[string]interface{}{
		"param1": "value1",
	})

	assert.Equal(t, execution.ErrorTypeConnection, execErr.Type)
	assert.Contains(t, execErr.Message, "tool failed")
	assert.Equal(t, types.NodeID("node1"), execErr.NodeID)
	assert.True(t, execErr.Recoverable)
	assert.NotEmpty(t, execErr.StackTrace)
	assert.Contains(t, execErr.Context, "server_id")
	assert.Contains(t, execErr.Context, "tool_name")
	assert.Contains(t, execErr.Context, "detail")
}

func TestStackFrame_Structure(t *testing.T) {
	frame := StackFrame{
		Function: "github.com/dshills/goflow/pkg/execution.TestFunction",
		File:     "/path/to/file.go",
		Line:     42,
		Package:  "github.com/dshills/goflow/pkg/execution",
	}

	assert.NotEmpty(t, frame.Function)
	assert.NotEmpty(t, frame.File)
	assert.Greater(t, frame.Line, 0)
	assert.NotEmpty(t, frame.Package)
}
