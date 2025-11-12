package mcpserver

import (
	"context"
	"fmt"
	"time"
)

// MCPClient is an interface for MCP protocol communication
// This allows the MCPServer to be tested without a real MCP connection
type MCPClient interface {
	// Connect establishes a connection to the MCP server
	Connect(ctx context.Context) error

	// Close terminates the connection to the MCP server
	Close() error

	// IsConnected returns true if the client is connected
	IsConnected() bool

	// ListTools retrieves all available tools from the server
	ListTools(ctx context.Context) ([]Tool, error)

	// CallTool invokes a tool on the server with the given parameters
	CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)

	// Ping sends a ping request to verify server health
	Ping(ctx context.Context) error
}

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
	client          MCPClient // Optional MCP client for protocol communication
}

// ServerMetadata contains server capabilities and version information
type ServerMetadata struct {
	ProtocolVersion string
	ServerVersion   string
	Capabilities    []string
	Vendor          string
}

// SetClient sets the MCP client for protocol communication
// This is typically called after creating the server to inject the client dependency
func (s *MCPServer) SetClient(client MCPClient) {
	s.client = client
}

// GetClient returns the MCP client if set, nil otherwise
func (s *MCPServer) GetClient() MCPClient {
	return s.client
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
	state := s.Connection.GetState()
	if state != StateDisconnected && state != StateFailed {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot connect from %s", state))
	}

	// Update state
	s.Connection.SetState(StateConnecting)
	s.Connection.UpdateLastActivity()

	return nil
}

// CompleteConnection marks the connection as successfully established
func (s *MCPServer) CompleteConnection() error {
	// Validate state transition
	if s.Connection.GetState() != StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot complete connection from %s", s.Connection.GetState()))
	}

	// Update state (lock for multiple field updates)
	s.Connection.mu.Lock()
	s.Connection.State = StateConnected
	s.Connection.ConnectedAt = time.Now()
	s.Connection.LastActivity = time.Now()
	s.Connection.ErrorCount = 0
	s.Connection.RetryBackoff = 0
	s.Connection.mu.Unlock()

	return nil
}

// FailConnection marks the connection as failed
func (s *MCPServer) FailConnection(errorMsg string) error {
	// Validate state transition
	if s.Connection.GetState() != StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot fail connection from %s", s.Connection.GetState()))
	}

	// Update state (lock for multiple field updates)
	s.Connection.mu.Lock()
	s.Connection.State = StateFailed
	s.Connection.LastActivity = time.Now()
	s.Connection.ErrorCount++
	s.Connection.LastError = errorMsg
	errorCount := s.Connection.ErrorCount
	s.Connection.mu.Unlock()

	// Calculate exponential backoff with safe conversion
	if errorCount < 0 {
		errorCount = 0
	}
	// Cap error count to prevent overflow in bit shift (max 30 = ~17 minutes)
	if errorCount > 30 {
		errorCount = 30
	}
	backoff := time.Duration(1<<uint(errorCount)) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second // Cap at 60 seconds
	}

	// Update retry backoff (lock for field access)
	s.Connection.mu.Lock()
	s.Connection.RetryBackoff = backoff
	s.Connection.mu.Unlock()

	// Update health status
	s.HealthStatus = HealthUnhealthy
	s.LastHealthCheck = time.Now()

	return nil
}

// Disconnect closes the connection to the MCP server
func (s *MCPServer) Disconnect() error {
	// For state machine validation, only error if coming from connecting state
	if s.Connection.GetState() == StateConnecting {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot disconnect from %s", s.Connection.GetState()))
	}

	// Update state
	s.Connection.SetState(StateDisconnected)
	s.Connection.UpdateLastActivity()

	// Clear tools cache
	s.Tools = []Tool{}

	// Update health status
	s.HealthStatus = HealthDisconnected
	s.LastHealthCheck = time.Now()

	return nil
}

// Reconnect attempts to reconnect to the server
func (s *MCPServer) Reconnect() error {
	// THREAD-SAFETY: Use getter for state check
	currentState := s.Connection.GetState()
	if currentState != StateFailed && currentState != StateDisconnected {
		return NewConnectionError(fmt.Sprintf("invalid state transition: cannot reconnect from %s", currentState))
	}

	// THREAD-SAFETY: Use getter for error count
	errorCount := s.Connection.GetErrorCount()
	if errorCount > 0 {
		if errorCount < 0 {
			errorCount = 0
		}
		// Cap error count to prevent overflow in bit shift (max 30 = ~17 minutes)
		if errorCount > 30 {
			errorCount = 30
		}
		backoff := time.Duration(1<<uint(errorCount)) * time.Second
		if backoff > 60*time.Second {
			backoff = 60 * time.Second // Cap at 60 seconds
		}

		// THREAD-SAFETY: Lock for multiple field updates
		s.Connection.mu.Lock()
		s.Connection.RetryBackoff = backoff
		s.Connection.mu.Unlock()
	}

	// THREAD-SAFETY: Lock for multiple field updates
	s.Connection.mu.Lock()
	s.Connection.State = StateConnecting
	s.Connection.LastActivity = time.Now()
	s.Connection.mu.Unlock()

	s.HealthStatus = HealthUnknown

	return nil
}

// DiscoverTools queries the server for available tools via MCP protocol
// This method calls the MCP server's tools/list endpoint and populates
// the MCPServer's Tools slice with the discovered tool schemas.
//
// The discovery process:
// 1. Validates the connection state (must be connected)
// 2. If an MCP client is configured, calls the tools/list JSON-RPC method
// 3. Parses the response and stores tools with their input/output schemas
// 4. Updates the last activity timestamp
//
// Returns an error if:
// - The server is not connected
// - The MCP client call fails
// - The response cannot be parsed
func (s *MCPServer) DiscoverTools() error {
	// THREAD-SAFETY: Use getter for state check
	if s.Connection.GetState() != StateConnected {
		return NewConnectionError("cannot discover tools: not connected")
	}

	// If a client is configured, use it to discover tools via MCP protocol
	if s.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tools, err := s.client.ListTools(ctx)
		if err != nil {
			errorMsg := fmt.Sprintf("tool discovery failed: %v", err)
			s.RecordUnhealthy(errorMsg)
			return NewConnectionError(fmt.Sprintf("failed to discover tools: %v", err))
		}

		// Store the discovered tools
		s.Tools = tools
		// THREAD-SAFETY: Use UpdateLastActivity method
		s.Connection.UpdateLastActivity()

		return nil
	}

	// For mock/testing scenarios without a client, initialize empty tools slice
	if s.Tools == nil {
		s.Tools = []Tool{}
	}

	// THREAD-SAFETY: Use UpdateLastActivity method
	s.Connection.UpdateLastActivity()

	return nil
}

// InvokeTool executes a tool on the MCP server via MCP protocol
// This method calls the MCP server's tools/call endpoint with the specified
// tool name and parameters.
//
// The invocation process:
// 1. Validates the connection state (must be connected)
// 2. Verifies the tool exists in the discovered tools list
// 3. If an MCP client is configured, calls the tools/call JSON-RPC method
// 4. Returns the tool execution result
//
// Returns an error if:
// - The server is not connected
// - The tool is not found in the tools list
// - The MCP client call fails
func (s *MCPServer) InvokeTool(toolName string, params map[string]interface{}) (interface{}, error) {
	// THREAD-SAFETY: Use getter for state check
	if s.Connection.GetState() != StateConnected {
		return nil, NewConnectionError("cannot invoke tool: not connected")
	}

	// Find the tool in the discovered tools list
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

	// If a client is configured, use it to invoke the tool via MCP protocol
	if s.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := s.client.CallTool(ctx, toolName, params)
		if err != nil {
			errorMsg := fmt.Sprintf("tool invocation failed: %v", err)
			s.RecordUnhealthy(errorMsg)
			return nil, NewExecutionError(fmt.Sprintf("failed to invoke tool %s: %v", toolName, err))
		}

		// THREAD-SAFETY: UpdateLastActivity already uses locking
		s.Connection.UpdateLastActivity()
		return result, nil
	}

	// For mock/testing scenarios without a client, return a mock result
	// THREAD-SAFETY: UpdateLastActivity already uses locking
	s.Connection.UpdateLastActivity()

	return map[string]interface{}{
		"success": true,
		"result":  "mock result",
	}, nil
}

// HealthCheck performs a health check on the server via MCP protocol
// If a client is configured, this sends a ping request to verify the server
// is responsive. Otherwise, it checks the connection state.
//
// The health check process:
// 1. Updates the LastHealthCheck timestamp
// 2. If disconnected, marks status as HealthDisconnected
// 3. If client is configured, sends a ping request
// 4. Updates health status based on ping result or connection state
//
// Returns an error if the ping fails
func (s *MCPServer) HealthCheck() error {
	s.LastHealthCheck = time.Now()

	// THREAD-SAFETY: Use getter for state check
	currentState := s.Connection.GetState()

	// If connection state is disconnected, report as disconnected
	if currentState == StateDisconnected {
		s.HealthStatus = HealthDisconnected
		return nil
	}

	// If a client is configured, use it to ping the server
	if s.client != nil && currentState == StateConnected {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := s.client.Ping(ctx)
		if err != nil {
			s.HealthStatus = HealthUnhealthy
			// THREAD-SAFETY: Use setters for error tracking
			s.Connection.SetLastError(fmt.Sprintf("ping failed: %v", err))
			s.Connection.IncrementErrorCount()
			return fmt.Errorf("health check failed: %w", err)
		}

		s.HealthStatus = HealthHealthy
		// THREAD-SAFETY: Use UpdateLastActivity method
		s.Connection.UpdateLastActivity()
		return nil
	}

	// For mock/testing scenarios without a client, mark as healthy if connected
	s.HealthStatus = HealthHealthy
	// THREAD-SAFETY: Use UpdateLastActivity method
	s.Connection.UpdateLastActivity()

	return nil
}

// RecordUnhealthy marks the server as unhealthy
func (s *MCPServer) RecordUnhealthy(errorMsg string) {
	s.HealthStatus = HealthUnhealthy
	s.LastHealthCheck = time.Now()
	// THREAD-SAFETY: Use setters for error tracking
	s.Connection.SetLastError(errorMsg)
	s.Connection.IncrementErrorCount()
}
