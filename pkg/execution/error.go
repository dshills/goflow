package execution

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// MCPLogEntry represents a single log entry from an MCP server.
type MCPLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"` // debug, info, warn, error
	Message   string                 `json:"message"`
	ServerID  string                 `json:"server_id"`
	ToolName  string                 `json:"tool_name,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NodeExecutionStep represents a single step in the execution chain leading to an error.
type NodeExecutionStep struct {
	NodeID      types.NodeID           `json:"node_id"`
	NodeType    string                 `json:"node_type"`
	Status      execution.NodeStatus   `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
}

// EnhancedExecutionError extends ExecutionError with additional debugging context.
type EnhancedExecutionError struct {
	*execution.ExecutionError
	// MCPLogs captures MCP server logs leading up to the error
	MCPLogs []MCPLogEntry `json:"mcp_logs,omitempty"`
	// NodeExecutionChain shows the path taken to reach the error
	NodeExecutionChain []NodeExecutionStep `json:"node_execution_chain,omitempty"`
	// VariableSnapshot captures variable values at the time of error
	VariableSnapshot map[string]interface{} `json:"variable_snapshot,omitempty"`
	// DetailedStackTrace provides a parsed stack trace with file/line info
	DetailedStackTrace []StackFrame `json:"detailed_stack_trace,omitempty"`
	// ErrorClassification provides additional error categorization
	ErrorClassification ErrorClassification `json:"error_classification"`
}

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// ErrorClassification provides detailed categorization of errors.
type ErrorClassification struct {
	Category    execution.ErrorType `json:"category"`    // validation, connection, execution, data, timeout
	Severity    ErrorSeverity       `json:"severity"`    // critical, high, medium, low
	Recoverable bool                `json:"recoverable"` // can this be retried?
	RetryHint   string              `json:"retry_hint,omitempty"`
}

// ErrorSeverity indicates the impact level of an error.
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical" // System failure, cannot continue
	SeverityHigh     ErrorSeverity = "high"     // Major feature broken
	SeverityMedium   ErrorSeverity = "medium"   // Degraded functionality
	SeverityLow      ErrorSeverity = "low"      // Minor issue, workaround available
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

// CaptureRuntimeStack captures the current stack trace with detailed frame information.
func CaptureRuntimeStack(skip int) []StackFrame {
	const maxFrames = 32
	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(skip+2, pc) // +2 to skip this function and runtime.Callers
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pc[:n])
	var stackFrames []StackFrame

	for {
		frame, more := frames.Next()

		// Extract package name from function
		pkg := "unknown"
		if idx := strings.LastIndex(frame.Function, "/"); idx != -1 {
			pkg = frame.Function[:idx]
		}

		stackFrames = append(stackFrames, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
			Package:  pkg,
		})

		if !more {
			break
		}
	}

	return stackFrames
}

// ParseStackTrace parses a string stack trace into structured frames.
func ParseStackTrace(stackTrace string) []StackFrame {
	if stackTrace == "" {
		return nil
	}

	lines := strings.Split(stackTrace, "\n")
	var frames []StackFrame

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Look for function lines (not starting with tab or space after trimming)
		if !strings.HasPrefix(lines[i], "\t") && !strings.HasPrefix(lines[i], " ") {
			frame := StackFrame{
				Function: line,
			}

			// Next line should be file:line
			if i+1 < len(lines) {
				fileLine := strings.TrimSpace(lines[i+1])
				if parts := strings.Fields(fileLine); len(parts) >= 1 {
					fileInfo := parts[0]
					if idx := strings.LastIndex(fileInfo, ":"); idx != -1 {
						frame.File = fileInfo[:idx]
						// Parse line number (ignore error as frame.Line defaults to 0)
						_, _ = fmt.Sscanf(fileInfo[idx+1:], "%d", &frame.Line)
					}
				}
				i++ // Skip file:line
			}

			// Extract package
			if idx := strings.LastIndex(frame.Function, "/"); idx != -1 {
				frame.Package = frame.Function[:idx]
			}

			frames = append(frames, frame)
		}
	}

	return frames
}

// NewEnhancedError creates an EnhancedExecutionError with full debugging context.
func NewEnhancedError(
	baseErr *execution.ExecutionError,
	exec *execution.Execution,
	mcpLogs []MCPLogEntry,
) *EnhancedExecutionError {
	enhanced := &EnhancedExecutionError{
		ExecutionError: baseErr,
		MCPLogs:        mcpLogs,
	}

	// Capture variable snapshot
	if exec != nil && exec.Context != nil {
		enhanced.VariableSnapshot = exec.Context.CreateSnapshot()
	}

	// Build node execution chain
	if exec != nil && len(exec.NodeExecutions) > 0 {
		enhanced.NodeExecutionChain = buildNodeExecutionChain(exec.NodeExecutions)
	}

	// Parse stack trace if available
	if baseErr.StackTrace != "" {
		enhanced.DetailedStackTrace = ParseStackTrace(baseErr.StackTrace)
	}

	// Classify error
	enhanced.ErrorClassification = classifyError(baseErr)

	return enhanced
}

// buildNodeExecutionChain converts node executions to execution steps.
func buildNodeExecutionChain(nodeExecutions []*execution.NodeExecution) []NodeExecutionStep {
	steps := make([]NodeExecutionStep, 0, len(nodeExecutions))

	for _, ne := range nodeExecutions {
		step := NodeExecutionStep{
			NodeID:      ne.NodeID,
			NodeType:    ne.NodeType,
			Status:      ne.Status,
			StartedAt:   ne.StartedAt,
			CompletedAt: ne.CompletedAt,
			Duration:    ne.Duration(),
			Inputs:      ne.Inputs,
			Outputs:     ne.Outputs,
		}
		steps = append(steps, step)
	}

	return steps
}

// classifyError determines error classification based on error type and context.
func classifyError(err *execution.ExecutionError) ErrorClassification {
	classification := ErrorClassification{
		Category:    err.Type,
		Recoverable: err.Recoverable,
	}

	// Determine severity and retry hints based on error type
	switch err.Type {
	case execution.ErrorTypeValidation:
		classification.Severity = SeverityHigh
		classification.RetryHint = "Fix validation errors in workflow definition"

	case execution.ErrorTypeConnection:
		classification.Severity = SeverityMedium
		classification.RetryHint = "Check MCP server connection and retry"

	case execution.ErrorTypeTimeout:
		classification.Severity = SeverityMedium
		classification.RetryHint = "Increase timeout or optimize operation"

	case execution.ErrorTypeData:
		classification.Severity = SeverityHigh
		classification.RetryHint = "Verify data transformation expressions and input data"

	case execution.ErrorTypeExecution:
		// Execution errors can vary in severity
		if err.Recoverable {
			classification.Severity = SeverityMedium
			classification.RetryHint = "Retry with same parameters"
		} else {
			classification.Severity = SeverityCritical
			classification.RetryHint = "Review error details and adjust workflow"
		}

	default:
		classification.Severity = SeverityHigh
		classification.RetryHint = "Review error details"
	}

	return classification
}

// FormatEnhancedError formats an enhanced error with rich context for debugging.
func FormatEnhancedError(enhanced *EnhancedExecutionError) string {
	var sb strings.Builder

	// Header
	sb.WriteString("=== EXECUTION ERROR ===\n")
	sb.WriteString(fmt.Sprintf("Type: %s\n", enhanced.Type))
	sb.WriteString(fmt.Sprintf("Severity: %s\n", enhanced.ErrorClassification.Severity))
	sb.WriteString(fmt.Sprintf("Recoverable: %v\n", enhanced.Recoverable))
	if enhanced.NodeID != "" {
		sb.WriteString(fmt.Sprintf("Node: %s\n", enhanced.NodeID))
	}
	sb.WriteString(fmt.Sprintf("Message: %s\n", enhanced.Message))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", enhanced.Timestamp.Format(time.RFC3339)))
	sb.WriteString("\n")

	// Retry hint
	if enhanced.ErrorClassification.RetryHint != "" {
		sb.WriteString(fmt.Sprintf("Recovery Hint: %s\n\n", enhanced.ErrorClassification.RetryHint))
	}

	// Node execution chain
	if len(enhanced.NodeExecutionChain) > 0 {
		sb.WriteString("=== EXECUTION PATH ===\n")
		for i, step := range enhanced.NodeExecutionChain {
			sb.WriteString(fmt.Sprintf("%d. %s (%s) - %s [%v]\n",
				i+1, step.NodeID, step.NodeType, step.Status, step.Duration))
		}
		sb.WriteString("\n")
	}

	// Variable snapshot
	if len(enhanced.VariableSnapshot) > 0 {
		sb.WriteString("=== VARIABLES AT ERROR ===\n")
		for name, value := range enhanced.VariableSnapshot {
			// Pretty print value
			valueStr := formatValue(value)
			sb.WriteString(fmt.Sprintf("  %s = %s\n", name, valueStr))
		}
		sb.WriteString("\n")
	}

	// Context
	if len(enhanced.Context) > 0 {
		sb.WriteString("=== ERROR CONTEXT ===\n")
		for key, value := range enhanced.Context {
			valueStr := formatValue(value)
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, valueStr))
		}
		sb.WriteString("\n")
	}

	// MCP logs
	if len(enhanced.MCPLogs) > 0 {
		sb.WriteString("=== MCP SERVER LOGS ===\n")
		for _, log := range enhanced.MCPLogs {
			sb.WriteString(fmt.Sprintf("[%s] [%s] %s: %s\n",
				log.Timestamp.Format("15:04:05.000"),
				strings.ToUpper(log.Level),
				log.ServerID,
				log.Message))
		}
		sb.WriteString("\n")
	}

	// Stack trace (limited to first 10 frames for readability)
	if len(enhanced.DetailedStackTrace) > 0 {
		sb.WriteString("=== STACK TRACE (Top 10 Frames) ===\n")
		maxFrames := 10
		if len(enhanced.DetailedStackTrace) < maxFrames {
			maxFrames = len(enhanced.DetailedStackTrace)
		}
		for i := 0; i < maxFrames; i++ {
			frame := enhanced.DetailedStackTrace[i]
			sb.WriteString(fmt.Sprintf("  %s\n", frame.Function))
			sb.WriteString(fmt.Sprintf("    %s:%d\n", frame.File, frame.Line))
		}
		if len(enhanced.DetailedStackTrace) > maxFrames {
			sb.WriteString(fmt.Sprintf("  ... and %d more frames\n", len(enhanced.DetailedStackTrace)-maxFrames))
		}
	}

	return sb.String()
}

// formatValue formats a value for display in error messages.
func formatValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		if len(v) > 100 {
			return fmt.Sprintf("%q... (truncated, %d chars)", v[:100], len(v))
		}
		return fmt.Sprintf("%q", v)
	case map[string]interface{}, []interface{}:
		// Pretty print JSON
		if jsonBytes, err := json.MarshalIndent(v, "    ", "  "); err == nil {
			jsonStr := string(jsonBytes)
			if len(jsonStr) > 200 {
				return jsonStr[:200] + "... (truncated)"
			}
			return jsonStr
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// MCPLogCollector provides an interface for collecting MCP server logs.
type MCPLogCollector interface {
	// CollectLogs retrieves recent logs from an MCP server
	CollectLogs(serverID string, since time.Time, limit int) ([]MCPLogEntry, error)
	// CollectLogsForExecution retrieves logs for a specific execution
	CollectLogsForExecution(execID types.ExecutionID, serverID string) ([]MCPLogEntry, error)
}

// ErrorContextBuilder helps build enhanced errors with fluent API.
type ErrorContextBuilder struct {
	baseErr          *execution.ExecutionError
	exec             *execution.Execution
	mcpLogs          []MCPLogEntry
	additionalFrames int
}

// NewErrorContextBuilder creates a new error context builder.
func NewErrorContextBuilder(baseErr *execution.ExecutionError) *ErrorContextBuilder {
	return &ErrorContextBuilder{
		baseErr: baseErr,
	}
}

// WithExecution adds execution context to the error.
func (b *ErrorContextBuilder) WithExecution(exec *execution.Execution) *ErrorContextBuilder {
	b.exec = exec
	return b
}

// WithMCPLogs adds MCP server logs to the error.
func (b *ErrorContextBuilder) WithMCPLogs(logs []MCPLogEntry) *ErrorContextBuilder {
	b.mcpLogs = logs
	return b
}

// WithAdditionalStackFrames captures additional stack frames.
func (b *ErrorContextBuilder) WithAdditionalStackFrames(count int) *ErrorContextBuilder {
	b.additionalFrames = count
	return b
}

// Build creates the enhanced error with all context.
func (b *ErrorContextBuilder) Build() *EnhancedExecutionError {
	enhanced := NewEnhancedError(b.baseErr, b.exec, b.mcpLogs)

	// Capture additional stack frames if requested
	if b.additionalFrames > 0 && len(enhanced.DetailedStackTrace) == 0 {
		enhanced.DetailedStackTrace = CaptureRuntimeStack(b.additionalFrames)
	}

	return enhanced
}

// WrapWithEnhancedContext wraps a standard error with enhanced context for a given execution.
func WrapWithEnhancedContext(
	err error,
	exec *execution.Execution,
	nodeID types.NodeID,
	mcpLogs []MCPLogEntry,
) *EnhancedExecutionError {
	// Convert to ExecutionError if not already
	var baseErr *execution.ExecutionError
	if execErr, ok := err.(*execution.ExecutionError); ok {
		baseErr = execErr
	} else {
		baseErr = &execution.ExecutionError{
			Type:        ExtractErrorType(err),
			Message:     err.Error(),
			NodeID:      nodeID,
			StackTrace:  string(debug.Stack()),
			Context:     make(map[string]interface{}),
			Recoverable: IsRecoverable(err),
			Timestamp:   time.Now(),
		}
	}

	return NewErrorContextBuilder(baseErr).
		WithExecution(exec).
		WithMCPLogs(mcpLogs).
		Build()
}
