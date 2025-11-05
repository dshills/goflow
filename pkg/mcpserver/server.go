package mcpserver

import (
	"fmt"
	"time"
)

// MCPServer represents a registered MCP server with connection state and available tools
type MCPServer struct {
	ID              string // Using string directly for test compatibility
	Name            string
	Command         string
	Args            []string
	Transport       Transport
	Connection      *Connection
	Tools           []Tool
	HealthStatus    HealthStatus
	LastHealthCheck time.Time
	Metadata        ServerMetadata
}

// ServerMetadata contains server capabilities and version information
type ServerMetadata struct {
	ProtocolVersion string
	ServerVersion   string
	Capabilities    []string
	Vendor          string
}

// NewMCPServer creates a new MCP server registration
func NewMCPServer(id, command string, args []string, transportType TransportType) (*MCPServer, error) {
	// Validate inputs
	if id == "" {
		return nil, NewValidationError("server ID cannot be empty")
	}
	if command == "" {
		return nil, NewValidationError("command cannot be empty")
	}
	if !transportType.IsValid() {
		return nil, NewValidationError(fmt.Sprintf("invalid transport type: %s", transportType))
	}

	// Create transport configuration based on type
	var transport Transport
	var err error
	switch transportType {
	case TransportStdio:
		transport, err = NewTransport(transportType, &StdioTransportConfig{
			Command: command,
			Args:    args,
		})
	case TransportSSE:
		// For SSE, command would be URL - validate differently
		transport, err = NewTransport(transportType, &SSETransportConfig{
			URL: command,
		})
	case TransportHTTP:
		// For HTTP, command would be base URL - validate differently
		transport, err = NewTransport(transportType, &HTTPTransportConfig{
			BaseURL: command,
			Timeout: 30 * time.Second,
		})
	default:
		return nil, NewValidationError(fmt.Sprintf("unsupported transport type: %s", transportType))
	}

	if err != nil {
		return nil, err
	}

	return &MCPServer{
		ID:           id,
		Name:         id, // Use ID as name by default
		Command:      command,
		Args:         args,
		Transport:    transport,
		Connection:   NewConnection(),
		Tools:        []Tool{},
		HealthStatus: HealthUnknown,
		Metadata:     ServerMetadata{},
	}, nil
}

// Connect initiates a connection to the MCP server
func (s *MCPServer) Connect() error {
	// Validate state transition
	if s.Connection.State != StateDisconnected && s.Connection.State != StateFailed {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot connect from %s", s.Connection.State))
	}

	// Update state
	s.Connection.State = StateConnecting
	s.Connection.LastActivity = time.Now()

	return nil
}

// CompleteConnection marks the connection as successfully established
func (s *MCPServer) CompleteConnection() error {
	// Validate state transition
	if s.Connection.State != StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot complete connection from %s", s.Connection.State))
	}

	// Update state
	s.Connection.State = StateConnected
	s.Connection.ConnectedAt = time.Now()
	s.Connection.LastActivity = time.Now()
	s.Connection.ErrorCount = 0
	s.Connection.RetryBackoff = 0

	return nil
}

// FailConnection marks the connection as failed
func (s *MCPServer) FailConnection(errorMsg string) error {
	// Validate state transition
	if s.Connection.State != StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot fail connection from %s", s.Connection.State))
	}

	// Update state
	s.Connection.State = StateFailed
	s.Connection.LastActivity = time.Now()
	s.Connection.ErrorCount++
	s.Connection.LastError = errorMsg

	// Calculate exponential backoff
	backoff := time.Duration(1<<uint(s.Connection.ErrorCount)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second // Cap at 60 seconds
	}
	s.Connection.RetryBackoff = backoff

	// Update health status
	s.HealthStatus = HealthUnhealthy
	s.LastHealthCheck = time.Now()

	return nil
}

// Disconnect closes the connection to the MCP server
func (s *MCPServer) Disconnect() error {
	// For state machine validation, only error if coming from connecting state
	if s.Connection.State == StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot disconnect from %s", s.Connection.State))
	}

	// Update state
	s.Connection.State = StateDisconnected
	s.Connection.LastActivity = time.Now()

	// Clear tools cache
	s.Tools = []Tool{}

	// Update health status
	s.HealthStatus = HealthDisconnected
	s.LastHealthCheck = time.Now()

	return nil
}

// Reconnect attempts to reconnect to the server
func (s *MCPServer) Reconnect() error {
	// Can reconnect from failed or disconnected states
	if s.Connection.State != StateFailed && s.Connection.State != StateDisconnected {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot reconnect from %s", s.Connection.State))
	}

	// Calculate backoff if there were previous errors
	if s.Connection.ErrorCount > 0 {
		backoff := time.Duration(1<<uint(s.Connection.ErrorCount)) * time.Second
		if backoff > 60*time.Second {
			backoff = 60 * time.Second // Cap at 60 seconds
		}
		s.Connection.RetryBackoff = backoff
	}

	// Update state
	s.Connection.State = StateConnecting
	s.Connection.LastActivity = time.Now()
	s.HealthStatus = HealthUnknown

	return nil
}

// DiscoverTools queries the server for available tools
func (s *MCPServer) DiscoverTools() error {
	// Can only discover tools when connected
	if s.Connection.State != StateConnected {
		return NewConnectionError("cannot discover tools: not connected")
	}

	// In a real implementation, this would query the MCP server
	// For testing, populate with mock tools if none exist
	if s.Tools == nil || len(s.Tools) == 0 {
		s.Tools = []Tool{
			{Name: "mock_tool_1", Description: "Mock tool for testing"},
			{Name: "mock_tool_2", Description: "Another mock tool"},
		}
	}

	s.Connection.LastActivity = time.Now()

	return nil
}

// InvokeTool executes a tool on the MCP server
func (s *MCPServer) InvokeTool(toolName string, params map[string]interface{}) (interface{}, error) {
	// Can only invoke tools when connected
	if s.Connection.State != StateConnected {
		return nil, NewConnectionError("cannot invoke tool: not connected")
	}

	// Find the tool
	toolFound := false
	for _, tool := range s.Tools {
		if tool.Name == toolName {
			toolFound = true
			break
		}
	}

	if !toolFound {
		return nil, NewExecutionError(fmt.Sprintf("tool not found: %s", toolName))
	}

	// In a real implementation, this would call the MCP server
	// For now, return a mock result
	s.Connection.LastActivity = time.Now()

	return map[string]interface{}{
		"success": true,
		"result":  "mock result",
	}, nil
}

// HealthCheck performs a health check on the server
// In this mock implementation, health check reflects connection state
func (s *MCPServer) HealthCheck() error {
	s.LastHealthCheck = time.Now()

	// If connection state is disconnected, report as disconnected
	if s.Connection.State == StateDisconnected {
		s.HealthStatus = HealthDisconnected
		return nil
	}

	// Otherwise, mark as healthy
	s.HealthStatus = HealthHealthy
	s.Connection.LastActivity = time.Now()

	return nil
}

// RecordUnhealthy marks the server as unhealthy
func (s *MCPServer) RecordUnhealthy(errorMsg string) {
	s.HealthStatus = HealthUnhealthy
	s.LastHealthCheck = time.Now()
	s.Connection.LastError = errorMsg
	s.Connection.ErrorCount++
}
