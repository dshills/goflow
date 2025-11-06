package execution

// This file provides integration examples for using enhanced errors
// in the execution engine. These are examples of how to integrate
// the error handling system with node executors and MCP servers.

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// Example: Enhanced error handling in MCP tool node execution
func exampleMCPToolNodeWithEnhancedErrors(
	ctx context.Context,
	nodeID types.NodeID,
	serverID string,
	toolName string,
	params map[string]interface{},
	exec *execution.Execution,
	logCollector MCPLogCollector,
) error {
	// Execute the tool
	// result, err := server.InvokeTool(toolName, params)

	// Simulate an error for demonstration
	err := fmt.Errorf("connection timeout")

	if err != nil {
		// Collect MCP logs from the last 5 minutes
		mcpLogs, logErr := logCollector.CollectLogs(
			serverID,
			time.Now().Add(-5*time.Minute),
			50, // last 50 log entries
		)
		if logErr != nil {
			// If we can't get logs, use empty slice
			mcpLogs = []MCPLogEntry{}
		}

		// Wrap the error with enhanced context
		enhanced := WrapWithEnhancedContext(err, exec, nodeID, mcpLogs)

		// Log the formatted error for debugging
		fmt.Println("=== Error occurred during execution ===")
		fmt.Println(FormatEnhancedError(enhanced))

		// Check if error is recoverable
		if enhanced.ErrorClassification.Recoverable {
			fmt.Printf("Recovery hint: %s\n", enhanced.ErrorClassification.RetryHint)

			// Could implement retry logic here based on error type
			// For now, just return the base error
		}

		// Return the base ExecutionError (maintains domain boundaries)
		return enhanced.ExecutionError
	}

	return nil
}

// Example: Using ErrorContextBuilder for complex error construction
func exampleBuildComplexError(
	baseErr error,
	exec *execution.Execution,
	nodeID types.NodeID,
	mcpLogs []MCPLogEntry,
) *EnhancedExecutionError {
	// Convert base error to ExecutionError if needed
	var execErr *execution.ExecutionError
	if e, ok := baseErr.(*execution.ExecutionError); ok {
		execErr = e
	} else {
		execErr = &execution.ExecutionError{
			Type:        ExtractErrorType(baseErr),
			Message:     baseErr.Error(),
			NodeID:      nodeID,
			Context:     make(map[string]interface{}),
			Recoverable: IsRecoverable(baseErr),
			Timestamp:   time.Now(),
		}
	}

	// Build enhanced error with all context
	enhanced := NewErrorContextBuilder(execErr).
		WithExecution(exec).
		WithMCPLogs(mcpLogs).
		WithAdditionalStackFrames(10).
		Build()

	return enhanced
}

// Example: Error classification and recovery strategy
func exampleErrorRecoveryStrategy(enhanced *EnhancedExecutionError) error {
	switch enhanced.Type {
	case execution.ErrorTypeConnection:
		// Connection errors are usually recoverable with retry
		if enhanced.ErrorClassification.Recoverable {
			fmt.Printf("Retrying after connection error: %s\n",
				enhanced.ErrorClassification.RetryHint)
			// Implement exponential backoff retry
			return retryWithBackoff(enhanced)
		}

	case execution.ErrorTypeTimeout:
		// Timeout errors may succeed with increased timeout
		fmt.Printf("Increasing timeout: %s\n",
			enhanced.ErrorClassification.RetryHint)
		return retryWithIncreasedTimeout(enhanced)

	case execution.ErrorTypeValidation:
		// Validation errors are not recoverable - fail fast
		fmt.Println("Validation error - cannot retry")
		fmt.Println(FormatEnhancedError(enhanced))
		return enhanced.ExecutionError

	case execution.ErrorTypeData:
		// Data transformation errors - provide detailed context
		fmt.Println("Data transformation failed")
		if expr, ok := enhanced.Context["expression"]; ok {
			fmt.Printf("Expression: %v\n", expr)
		}
		if input, ok := enhanced.Context["input_value"]; ok {
			fmt.Printf("Input: %v\n", input)
		}
		return enhanced.ExecutionError

	default:
		// Unknown error type
		fmt.Println(FormatEnhancedError(enhanced))
		return enhanced.ExecutionError
	}

	return nil
}

// Example: Collecting and filtering MCP logs
type exampleMCPLogCollectorImpl struct {
	logs map[string][]MCPLogEntry
}

func newExampleLogCollector() *exampleMCPLogCollectorImpl {
	return &exampleMCPLogCollectorImpl{
		logs: make(map[string][]MCPLogEntry),
	}
}

func (c *exampleMCPLogCollectorImpl) CollectLogs(
	serverID string,
	since time.Time,
	limit int,
) ([]MCPLogEntry, error) {
	serverLogs, ok := c.logs[serverID]
	if !ok {
		return []MCPLogEntry{}, nil
	}

	// Filter logs by timestamp and limit
	var result []MCPLogEntry
	for _, log := range serverLogs {
		if log.Timestamp.After(since) {
			result = append(result, log)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

func (c *exampleMCPLogCollectorImpl) CollectLogsForExecution(
	execID types.ExecutionID,
	serverID string,
) ([]MCPLogEntry, error) {
	// Implementation would filter logs by execution ID
	// For now, just return recent logs
	return c.CollectLogs(serverID, time.Now().Add(-10*time.Minute), 100)
}

// Add log entry (for testing/simulation)
func (c *exampleMCPLogCollectorImpl) addLog(log MCPLogEntry) {
	c.logs[log.ServerID] = append(c.logs[log.ServerID], log)
}

// Example: Error monitoring and metrics
func exampleErrorMonitoring(enhanced *EnhancedExecutionError) {
	// Record error metrics
	recordErrorMetric(
		enhanced.Type,
		enhanced.ErrorClassification.Severity,
		enhanced.NodeID,
	)

	// Send alert for critical errors
	if enhanced.ErrorClassification.Severity == SeverityCritical {
		sendCriticalAlert(enhanced)
	}

	// Store error for debugging
	storeErrorForDebugging(enhanced)

	// Log structured error data
	logStructuredError(enhanced)
}

// Mock functions for the example
func recordErrorMetric(errType execution.ErrorType, severity ErrorSeverity, nodeID types.NodeID) {
	fmt.Printf("Metric: type=%s severity=%s node=%s\n", errType, severity, nodeID)
}

func sendCriticalAlert(enhanced *EnhancedExecutionError) {
	fmt.Printf("CRITICAL ALERT: %s\n", enhanced.Message)
}

func storeErrorForDebugging(enhanced *EnhancedExecutionError) {
	fmt.Printf("Stored error %s for debugging\n", enhanced.NodeID)
}

func logStructuredError(enhanced *EnhancedExecutionError) {
	fmt.Printf("Structured log: [%s] %s at %s\n",
		enhanced.Type, enhanced.Message, enhanced.NodeID)
}

func retryWithBackoff(enhanced *EnhancedExecutionError) error {
	fmt.Println("Implementing exponential backoff retry...")
	return nil
}

func retryWithIncreasedTimeout(enhanced *EnhancedExecutionError) error {
	fmt.Println("Retrying with increased timeout...")
	return nil
}

// Example: Complete error handling flow in node executor
func exampleCompleteNodeExecutionFlow(
	ctx context.Context,
	nodeID types.NodeID,
	exec *execution.Execution,
	logCollector MCPLogCollector,
) error {
	// 1. Execute the node (simulated)
	err := executeNodeSimulated(ctx, nodeID)

	if err != nil {
		// 2. Collect relevant MCP logs
		mcpLogs, _ := logCollector.CollectLogs(
			"example-server",
			time.Now().Add(-5*time.Minute),
			50,
		)

		// 3. Create enhanced error with full context
		enhanced := WrapWithEnhancedContext(err, exec, nodeID, mcpLogs)

		// 4. Log formatted error for debugging
		fmt.Println(FormatEnhancedError(enhanced))

		// 5. Record metrics and monitoring
		exampleErrorMonitoring(enhanced)

		// 6. Apply recovery strategy
		if recoveryErr := exampleErrorRecoveryStrategy(enhanced); recoveryErr != nil {
			// Recovery failed, return original error
			return enhanced.ExecutionError
		}

		// Recovery succeeded
		return nil
	}

	return nil
}

func executeNodeSimulated(ctx context.Context, nodeID types.NodeID) error {
	// Simulate node execution
	return fmt.Errorf("simulated error for demonstration")
}

// Example: Error reporting to external systems
func exampleErrorReporting(enhanced *EnhancedExecutionError) {
	// Format error for external reporting
	report := map[string]interface{}{
		"error_type":     enhanced.Type,
		"severity":       enhanced.ErrorClassification.Severity,
		"message":        enhanced.Message,
		"node_id":        enhanced.NodeID,
		"timestamp":      enhanced.Timestamp,
		"recoverable":    enhanced.Recoverable,
		"execution_path": formatExecutionPath(enhanced.NodeExecutionChain),
		"variable_count": len(enhanced.VariableSnapshot),
		"log_count":      len(enhanced.MCPLogs),
	}

	// Send to reporting system (mock)
	fmt.Printf("Error report: %+v\n", report)
}

func formatExecutionPath(chain []NodeExecutionStep) []string {
	var path []string
	for _, step := range chain {
		path = append(path, fmt.Sprintf("%s(%s)", step.NodeID, step.NodeType))
	}
	return path
}
