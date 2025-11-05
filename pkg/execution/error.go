package execution

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// WrapToolError wraps an MCP tool error with rich context for debugging.
func WrapToolError(nodeID types.NodeID, serverID, toolName string, err error, params map[string]interface{}) *execution.ExecutionError {
	context := map[string]interface{}{
		"server_id":  serverID,
		"tool_name":  toolName,
		"parameters": params,
	}

	// Check if it's already an MCP tool error with additional context
	if toolErr, ok := err.(*MCPToolError); ok {
		// Merge contexts
		for k, v := range toolErr.Context {
			context[k] = v
		}

		return &execution.ExecutionError{
			Type:        execution.ErrorTypeConnection,
			Message:     toolErr.Message,
			NodeID:      nodeID,
			StackTrace:  string(debug.Stack()),
			Context:     context,
			Recoverable: toolErr.Recoverable,
			Timestamp:   time.Now(),
		}
	}

	// Generic tool error
	return &execution.ExecutionError{
		Type:        execution.ErrorTypeExecution,
		Message:     fmt.Sprintf("tool %s/%s failed: %v", serverID, toolName, err),
		NodeID:      nodeID,
		StackTrace:  string(debug.Stack()),
		Context:     context,
		Recoverable: false,
		Timestamp:   time.Now(),
	}
}

// WrapTransformError wraps a transformation error with rich context.
func WrapTransformError(nodeID types.NodeID, inputVar, expression string, err error, inputValue interface{}) *execution.ExecutionError {
	context := map[string]interface{}{
		"input_variable": inputVar,
		"expression":     expression,
		"input_value":    inputValue,
	}

	// Check if it's already a transform error with additional context
	if transformErr, ok := err.(*TransformError); ok {
		// Merge contexts
		for k, v := range transformErr.Context {
			context[k] = v
		}

		return &execution.ExecutionError{
			Type:        execution.ErrorTypeData,
			Message:     transformErr.Message,
			NodeID:      nodeID,
			StackTrace:  string(debug.Stack()),
			Context:     context,
			Recoverable: false,
			Timestamp:   time.Now(),
		}
	}

	// Generic transformation error
	return &execution.ExecutionError{
		Type:        execution.ErrorTypeData,
		Message:     fmt.Sprintf("transformation failed: %v", err),
		NodeID:      nodeID,
		StackTrace:  string(debug.Stack()),
		Context:     context,
		Recoverable: false,
		Timestamp:   time.Now(),
	}
}

// WrapValidationError wraps a validation error with context.
func WrapValidationError(message string, context map[string]interface{}) *execution.ExecutionError {
	return &execution.ExecutionError{
		Type:        execution.ErrorTypeValidation,
		Message:     message,
		StackTrace:  string(debug.Stack()),
		Context:     context,
		Recoverable: false,
		Timestamp:   time.Now(),
	}
}

// WrapConnectionError wraps a connection error with server context.
func WrapConnectionError(nodeID types.NodeID, serverID string, err error) *execution.ExecutionError {
	context := map[string]interface{}{
		"server_id": serverID,
	}

	return &execution.ExecutionError{
		Type:        execution.ErrorTypeConnection,
		Message:     fmt.Sprintf("connection to server %s failed: %v", serverID, err),
		NodeID:      nodeID,
		StackTrace:  string(debug.Stack()),
		Context:     context,
		Recoverable: true,
		Timestamp:   time.Now(),
	}
}

// WrapTimeoutError wraps a timeout error with execution context.
func WrapTimeoutError(nodeID types.NodeID, operation string, timeout time.Duration) *execution.ExecutionError {
	context := map[string]interface{}{
		"operation": operation,
		"timeout":   timeout.String(),
	}

	return &execution.ExecutionError{
		Type:        execution.ErrorTypeTimeout,
		Message:     fmt.Sprintf("%s timed out after %s", operation, timeout),
		NodeID:      nodeID,
		StackTrace:  string(debug.Stack()),
		Context:     context,
		Recoverable: true,
		Timestamp:   time.Now(),
	}
}

// NewNodeError creates a node-level error from an execution error.
func NewNodeError(execErr *execution.ExecutionError) *execution.NodeError {
	return &execution.NodeError{
		Type:       execErr.Type,
		Message:    execErr.Message,
		StackTrace: execErr.StackTrace,
		Context:    execErr.Context,
	}
}

// ErrorContext provides helper methods for building error context.
type ErrorContext struct {
	data map[string]interface{}
}

// NewErrorContext creates a new error context builder.
func NewErrorContext() *ErrorContext {
	return &ErrorContext{
		data: make(map[string]interface{}),
	}
}

// Add adds a key-value pair to the error context.
func (ec *ErrorContext) Add(key string, value interface{}) *ErrorContext {
	ec.data[key] = value
	return ec
}

// AddIf conditionally adds a key-value pair if the condition is true.
func (ec *ErrorContext) AddIf(condition bool, key string, value interface{}) *ErrorContext {
	if condition {
		ec.data[key] = value
	}
	return ec
}

// Build returns the constructed context map.
func (ec *ErrorContext) Build() map[string]interface{} {
	return ec.data
}

// FormatErrorChain formats an error and its chain for logging.
func FormatErrorChain(err error) string {
	if err == nil {
		return ""
	}

	// If it's an execution error, format with details
	if execErr, ok := err.(*execution.ExecutionError); ok {
		msg := fmt.Sprintf("[%s] %s", execErr.Type, execErr.Message)
		if execErr.NodeID != "" {
			msg = fmt.Sprintf("Node %s: %s", execErr.NodeID, msg)
		}
		if len(execErr.Context) > 0 {
			msg = fmt.Sprintf("%s\nContext: %v", msg, execErr.Context)
		}
		return msg
	}

	// Otherwise, just return the error message
	return err.Error()
}

// IsRecoverable checks if an error is recoverable (can be retried).
func IsRecoverable(err error) bool {
	if execErr, ok := err.(*execution.ExecutionError); ok {
		return execErr.Recoverable
	}
	if toolErr, ok := err.(*MCPToolError); ok {
		return toolErr.Recoverable
	}
	return false
}

// ExtractErrorType returns the error type from any error.
func ExtractErrorType(err error) execution.ErrorType {
	if execErr, ok := err.(*execution.ExecutionError); ok {
		return execErr.Type
	}
	if _, ok := err.(*MCPToolError); ok {
		return execution.ErrorTypeConnection
	}
	if _, ok := err.(*TransformError); ok {
		return execution.ErrorTypeData
	}
	return execution.ErrorTypeExecution
}
