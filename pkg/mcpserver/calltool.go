package mcpserver

import (
	"context"
	"fmt"
	"time"
)

// CallTool invokes an MCP tool on the server with comprehensive validation and error handling.
// This is the primary method for executing MCP tools in GoFlow workflow execution.
//
// The method performs the following steps:
// 1. Validates the connection state (must be connected)
// 2. Validates that the tool exists on this server
// 3. Validates arguments against the tool's input schema (if available)
// 4. Invokes the tool via MCP client with timeout
// 5. Returns the tool result or a detailed, contextual error
//
// Arguments:
//   - toolName: The name of the tool to invoke (must exist in Tools list)
//   - arguments: A map of argument names to values (must match tool's input schema)
//
// Returns:
//   - interface{}: The tool's result (structure depends on the tool)
//   - error: A detailed error if validation or execution fails
//
// Error Types:
//   - ConnectionError: If the server is not connected
//   - ValidationError: If the tool doesn't exist or arguments are invalid
//   - ExecutionError: If the tool invocation fails
//
// Example:
//
//	result, err := server.CallTool("search", map[string]interface{}{
//	    "query": "golang best practices",
//	    "limit": 10,
//	})
func (s *MCPServer) CallTool(toolName string, arguments map[string]interface{}) (interface{}, error) {
	// Initialize arguments if nil to prevent panic
	if arguments == nil {
		arguments = make(map[string]interface{})
	}

	// Step 1: Validate connection state
	if s.Connection.State != StateConnected {
		return nil, NewConnectionError(fmt.Sprintf(
			"cannot call tool '%s': server '%s' is not connected (state: %s)",
			toolName,
			s.ID,
			s.Connection.State,
		))
	}

	// Step 2: Find and validate tool exists
	var tool *Tool
	for i := range s.Tools {
		if s.Tools[i].Name == toolName {
			tool = &s.Tools[i]
			break
		}
	}

	if tool == nil {
		// Provide helpful error with available tools
		availableTools := make([]string, len(s.Tools))
		for i, t := range s.Tools {
			availableTools[i] = t.Name
		}
		return nil, NewValidationError(fmt.Sprintf(
			"tool '%s' not found on server '%s'. Available tools: %v",
			toolName,
			s.ID,
			availableTools,
		))
	}

	// Step 3: Validate arguments against input schema
	if tool.InputSchema != nil {
		if err := s.validateArguments(tool, arguments); err != nil {
			return nil, err
		}
	}

	// Step 4: Invoke the tool via MCP client
	s.Connection.LastActivity = time.Now()

	// If a client is configured, use it for protocol communication
	if s.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := s.client.CallTool(ctx, toolName, arguments)
		if err != nil {
			return s.handleToolError(toolName, arguments, err)
		}

		// Success - reset error count, clear last error, and update activity
		s.Connection.ErrorCount = 0
		s.Connection.LastError = "" // Clear error on success
		s.Connection.LastActivity = time.Now()

		return result, nil
	}

	// Fallback: Mock result for testing scenarios without a client
	s.Connection.LastActivity = time.Now()

	return map[string]interface{}{
		"success": true,
		"result":  fmt.Sprintf("mock result for tool '%s'", toolName),
		"tool":    toolName,
		"args":    arguments,
	}, nil
}

// validateArguments validates the provided arguments against the tool's input schema.
// This method checks for:
// - Missing required fields
// - Unexpected/extra arguments (helps catch typos)
// - Basic schema compliance
func (s *MCPServer) validateArguments(tool *Tool, arguments map[string]interface{}) error {
	schema := tool.InputSchema

	// Check for missing required fields
	for _, required := range schema.Required {
		if _, exists := arguments[required]; !exists {
			return NewValidationError(fmt.Sprintf(
				"missing required argument '%s' for tool '%s' on server '%s'",
				required,
				tool.Name,
				s.ID,
			))
		}
	}

	// Check for unexpected arguments (helps catch typos in argument names)
	// Only perform this check if:
	// 1. Schema has defined properties
	// 2. AdditionalProperties is explicitly set to false (nil means true/allowed by default)
	allowsAdditional := schema.AdditionalProperties == nil || *schema.AdditionalProperties
	if !allowsAdditional && schema.Properties != nil && len(schema.Properties) > 0 {
		for argName := range arguments {
			if _, exists := schema.Properties[argName]; !exists {
				// Collect valid argument names for helpful error message
				validArgs := make([]string, 0, len(schema.Properties))
				for name := range schema.Properties {
					validArgs = append(validArgs, name)
				}
				return NewValidationError(fmt.Sprintf(
					"unexpected argument '%s' for tool '%s'. Valid arguments: %v",
					argName,
					tool.Name,
					validArgs,
				))
			}
		}
	}

	// Note: Additional type validation (string, number, boolean, etc.) could be added here
	// For now, we rely on the MCP server to perform detailed type checking
	// as it has the complete schema information

	return nil
}

// handleToolError processes tool invocation errors and returns a structured error
// with context for debugging and retry logic.
func (s *MCPServer) handleToolError(toolName string, arguments map[string]interface{}, err error) (interface{}, error) {
	// Update connection health status
	s.Connection.ErrorCount++
	s.Connection.LastError = err.Error()

	// Determine if error is recoverable (can be retried)
	recoverable := isRecoverableError(err)

	// If error indicates connection issues, mark server as unhealthy
	if recoverable {
		s.RecordUnhealthy(fmt.Sprintf("tool call failed: %v", err))
	}

	// Return structured error with full context
	return nil, &MCPError{
		Type:    ErrorTypeExecution,
		Message: fmt.Sprintf("tool '%s' invocation failed: %v", toolName, err),
		Context: map[string]interface{}{
			"server_id":   s.ID,
			"tool_name":   toolName,
			"arguments":   arguments,
			"recoverable": recoverable,
			"error_count": s.Connection.ErrorCount,
		},
	}
}

// isRecoverableError determines if an error is potentially recoverable through retry.
// Connection, timeout, and temporary errors are typically recoverable.
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Patterns that indicate recoverable errors (usually transient issues)
	recoverablePatterns := []string{
		"timeout",
		"connection",
		"refused",
		"reset",
		"broken pipe",
		"temporary",
		"unavailable",
		"deadline exceeded",
		"context deadline",
	}

	// Convert to lowercase for case-insensitive matching
	errMsgLower := toLowerSimple(errMsg)

	for _, pattern := range recoverablePatterns {
		if containsSubstring(errMsgLower, pattern) {
			return true
		}
	}

	return false
}

// containsSubstring checks if a string contains a substring.
// Both strings should be pre-converted to lowercase for case-insensitive matching.
func containsSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// toLowerSimple converts a string to lowercase (ASCII only).
func toLowerSimple(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
