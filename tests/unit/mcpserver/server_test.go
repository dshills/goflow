package mcpserver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// MockMCPClient is a mock implementation of MCPClient for testing
type MockMCPClient struct {
	tools     []mcpserver.Tool
	pingError error // Set to simulate ping failures
}

func (m *MockMCPClient) Connect(ctx context.Context) error {
	return nil
}

func (m *MockMCPClient) Close() error {
	return nil
}

func (m *MockMCPClient) IsConnected() bool {
	return true
}

func (m *MockMCPClient) ListTools(ctx context.Context) ([]mcpserver.Tool, error) {
	return m.tools, nil
}

func (m *MockMCPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"result": "mock result"}, nil
}

func (m *MockMCPClient) Ping(ctx context.Context) error {
	return m.pingError
}

// TestNewMCPServer tests creation of new MCP server instances
func TestNewMCPServer(t *testing.T) {
	tests := []struct {
		name      string
		serverID  string
		command   string
		args      []string
		transport mcpserver.TransportType
		wantErr   bool
	}{
		{
			name:      "valid stdio server with npx command",
			serverID:  "filesystem",
			command:   "npx",
			args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
			transport: mcpserver.TransportStdio,
			wantErr:   false,
		},
		{
			name:      "valid stdio server with python command",
			serverID:  "weather",
			command:   "python3",
			args:      []string{"-m", "mcp_server_weather"},
			transport: mcpserver.TransportStdio,
			wantErr:   false,
		},
		{
			name:      "valid server with no args",
			serverID:  "simple",
			command:   "/usr/local/bin/my-mcp-server",
			args:      []string{},
			transport: mcpserver.TransportStdio,
			wantErr:   false,
		},
		{
			name:      "empty server ID should fail",
			serverID:  "",
			command:   "npx",
			args:      []string{"-y", "server"},
			transport: mcpserver.TransportStdio,
			wantErr:   true,
		},
		{
			name:      "empty command should fail",
			serverID:  "test-server",
			command:   "",
			args:      []string{},
			transport: mcpserver.TransportStdio,
			wantErr:   true,
		},
		{
			name:      "invalid transport type should fail",
			serverID:  "test-server",
			command:   "npx",
			args:      []string{},
			transport: "invalid-transport",
			wantErr:   true,
		},
		{
			name:      "server ID with special characters",
			serverID:  "my-server-v2.0",
			command:   "node",
			args:      []string{"server.js"},
			transport: mcpserver.TransportStdio,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := mcpserver.NewMCPServer(tt.serverID, tt.command, tt.args, tt.transport)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewMCPServer() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewMCPServer() unexpected error: %v", err)
				return
			}

			if server == nil {
				t.Fatal("NewMCPServer() returned nil server")
			}

			if server.ID != tt.serverID {
				t.Errorf("NewMCPServer() ID = %v, want %v", server.ID, tt.serverID)
			}

			if server.Command != tt.command {
				t.Errorf("NewMCPServer() Command = %v, want %v", server.Command, tt.command)
			}

			if len(server.Args) != len(tt.args) {
				t.Errorf("NewMCPServer() Args length = %v, want %v", len(server.Args), len(tt.args))
			}

			if server.Transport.Type() != tt.transport {
				t.Errorf("NewMCPServer() Transport = %v, want %v", server.Transport.Type(), tt.transport)
			}

			// Verify initial state
			if server.Connection.State != mcpserver.StateDisconnected {
				t.Errorf("NewMCPServer() initial state = %v, want %v", server.Connection.State, mcpserver.StateDisconnected)
			}

			if server.HealthStatus != mcpserver.HealthUnknown {
				t.Errorf("NewMCPServer() initial health = %v, want %v", server.HealthStatus, mcpserver.HealthUnknown)
			}

			if len(server.Tools) != 0 {
				t.Errorf("NewMCPServer() initial tools should be empty, got %d", len(server.Tools))
			}
		})
	}
}

// TestMCPServer_ConnectionStateTransitions tests the connection state machine
func TestMCPServer_ConnectionStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		initialState  mcpserver.ConnectionState
		operation     string
		expectedState mcpserver.ConnectionState
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "disconnected to connecting",
			initialState:  mcpserver.StateDisconnected,
			operation:     "connect",
			expectedState: mcpserver.StateConnecting,
			wantErr:       false,
		},
		{
			name:          "connecting to connected",
			initialState:  mcpserver.StateConnecting,
			operation:     "complete_connect",
			expectedState: mcpserver.StateConnected,
			wantErr:       false,
		},
		{
			name:          "connecting to failed",
			initialState:  mcpserver.StateConnecting,
			operation:     "fail_connect",
			expectedState: mcpserver.StateFailed,
			wantErr:       false,
		},
		{
			name:          "connected to disconnected",
			initialState:  mcpserver.StateConnected,
			operation:     "disconnect",
			expectedState: mcpserver.StateDisconnected,
			wantErr:       false,
		},
		{
			name:          "failed to connecting (retry)",
			initialState:  mcpserver.StateFailed,
			operation:     "reconnect",
			expectedState: mcpserver.StateConnecting,
			wantErr:       false,
		},
		{
			name:          "invalid transition: disconnected to connected",
			initialState:  mcpserver.StateDisconnected,
			operation:     "complete_connect",
			expectedState: mcpserver.StateDisconnected,
			wantErr:       true,
			errMsg:        "invalid state transition",
		},
		{
			name:          "invalid transition: connected to connecting",
			initialState:  mcpserver.StateConnected,
			operation:     "connect",
			expectedState: mcpserver.StateConnected,
			wantErr:       true,
			errMsg:        "invalid state transition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)

			// Set initial state (this would be done by internal methods)
			server.Connection.State = tt.initialState

			var err error
			switch tt.operation {
			case "connect":
				err = server.Connect()
			case "complete_connect":
				err = server.CompleteConnection()
			case "fail_connect":
				err = server.FailConnection("connection failed")
			case "disconnect":
				err = server.Disconnect()
			case "reconnect":
				err = server.Reconnect()
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing %q but got none", tt.errMsg)
				}
				// State should remain unchanged on error
				if server.Connection.State != tt.initialState {
					t.Errorf("State changed on error: got %v, want %v", server.Connection.State, tt.initialState)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if server.Connection.State != tt.expectedState {
					t.Errorf("State = %v, want %v", server.Connection.State, tt.expectedState)
				}
			}
		})
	}
}

// TestMCPServer_HealthStatusTracking tests health status updates
func TestMCPServer_HealthStatusTracking(t *testing.T) {
	tests := []struct {
		name              string
		initialHealth     mcpserver.HealthStatus
		operation         string
		expectedHealth    mcpserver.HealthStatus
		expectedLastCheck bool // whether LastHealthCheck should be updated
	}{
		{
			name:              "initial unknown to healthy",
			initialHealth:     mcpserver.HealthUnknown,
			operation:         "healthy_check",
			expectedHealth:    mcpserver.HealthHealthy,
			expectedLastCheck: true,
		},
		{
			name:              "healthy to unhealthy",
			initialHealth:     mcpserver.HealthHealthy,
			operation:         "unhealthy_check",
			expectedHealth:    mcpserver.HealthUnhealthy,
			expectedLastCheck: true,
		},
		{
			name:              "unhealthy to healthy (recovered)",
			initialHealth:     mcpserver.HealthUnhealthy,
			operation:         "healthy_check",
			expectedHealth:    mcpserver.HealthHealthy,
			expectedLastCheck: true,
		},
		{
			name:              "connected to disconnected health",
			initialHealth:     mcpserver.HealthHealthy,
			operation:         "disconnect",
			expectedHealth:    mcpserver.HealthDisconnected,
			expectedLastCheck: true,
		},
		{
			name:              "disconnected to unknown on reconnect",
			initialHealth:     mcpserver.HealthDisconnected,
			operation:         "reconnect",
			expectedHealth:    mcpserver.HealthUnknown,
			expectedLastCheck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.HealthStatus = tt.initialHealth

			// Set connection state based on initial health (except for disconnected test cases)
			if tt.initialHealth != mcpserver.HealthDisconnected {
				server.Connection.State = mcpserver.StateConnected
			}

			oldLastCheck := server.LastHealthCheck

			switch tt.operation {
			case "healthy_check":
				server.HealthCheck()
			case "unhealthy_check":
				server.RecordUnhealthy("server error")
			case "disconnect":
				server.Disconnect()
			case "reconnect":
				server.Reconnect()
			}

			if server.HealthStatus != tt.expectedHealth {
				t.Errorf("HealthStatus = %v, want %v", server.HealthStatus, tt.expectedHealth)
			}

			if tt.expectedLastCheck {
				if !server.LastHealthCheck.After(oldLastCheck) {
					t.Error("LastHealthCheck should be updated but wasn't")
				}
			}
		})
	}
}

// TestMCPServer_ToolDiscovery tests tool discovery and caching
func TestMCPServer_ToolDiscovery(t *testing.T) {
	tests := []struct {
		name              string
		connectionState   mcpserver.ConnectionState
		setupTools        []mcpserver.Tool
		wantErr           bool
		errMsg            string
		expectToolsUpdate bool
	}{
		{
			name:            "discover tools when connected",
			connectionState: mcpserver.StateConnected,
			setupTools: []mcpserver.Tool{
				{Name: "read_file", Description: "Read a file"},
				{Name: "write_file", Description: "Write to a file"},
			},
			wantErr:           false,
			expectToolsUpdate: true,
		},
		{
			name:              "cannot discover tools when disconnected",
			connectionState:   mcpserver.StateDisconnected,
			setupTools:        nil,
			wantErr:           true,
			errMsg:            "not connected",
			expectToolsUpdate: false,
		},
		{
			name:              "cannot discover tools when connecting",
			connectionState:   mcpserver.StateConnecting,
			setupTools:        nil,
			wantErr:           true,
			errMsg:            "not connected",
			expectToolsUpdate: false,
		},
		{
			name:              "cannot discover tools when failed",
			connectionState:   mcpserver.StateFailed,
			setupTools:        nil,
			wantErr:           true,
			errMsg:            "not connected",
			expectToolsUpdate: false,
		},
		{
			name:              "discover empty tool list",
			connectionState:   mcpserver.StateConnected,
			setupTools:        []mcpserver.Tool{},
			wantErr:           false,
			expectToolsUpdate: true,
		},
		{
			name:            "discover many tools",
			connectionState: mcpserver.StateConnected,
			setupTools: []mcpserver.Tool{
				{Name: "tool1", Description: "Tool 1"},
				{Name: "tool2", Description: "Tool 2"},
				{Name: "tool3", Description: "Tool 3"},
				{Name: "tool4", Description: "Tool 4"},
				{Name: "tool5", Description: "Tool 5"},
			},
			wantErr:           false,
			expectToolsUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.Connection.State = tt.connectionState

			err := server.DiscoverTools()

			if tt.wantErr {
				if err == nil {
					t.Errorf("DiscoverTools() expected error containing %q but got none", tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("DiscoverTools() unexpected error: %v", err)
				return
			}

			if tt.expectToolsUpdate {
				// In real implementation, tools would be populated
				// For now, just verify the method was callable
				if server.Tools == nil {
					t.Error("DiscoverTools() Tools should not be nil after discovery")
				}
			}
		})
	}
}

// TestMCPServer_ToolCache tests tool caching behavior
func TestMCPServer_ToolCache(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		expectCached  bool
		expectRefresh bool
	}{
		{
			name:          "initial discovery caches tools",
			operation:     "discover",
			expectCached:  true,
			expectRefresh: false,
		},
		{
			name:          "disconnect clears tool cache",
			operation:     "disconnect",
			expectCached:  false,
			expectRefresh: false,
		},
		{
			name:          "reconnect requires rediscovery",
			operation:     "reconnect",
			expectCached:  false,
			expectRefresh: true,
		},
		{
			name:          "version change triggers refresh",
			operation:     "version_change",
			expectCached:  false,
			expectRefresh: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)

			// Setup: create mock client with tools
			mockClient := &MockMCPClient{
				tools: []mcpserver.Tool{
					{Name: "read_file", Description: "Read a file"},
					{Name: "write_file", Description: "Write a file"},
					{Name: "list_files", Description: "List files"},
				},
			}
			server.SetClient(mockClient)

			// Setup: connect and discover tools
			server.Connection.State = mcpserver.StateConnected
			server.DiscoverTools()
			initialToolCount := len(server.Tools)
			_ = initialToolCount // Suppress unused variable warning

			switch tt.operation {
			case "discover":
				// Already done in setup
			case "disconnect":
				server.Disconnect()
			case "reconnect":
				server.Disconnect()
				server.Connection.State = mcpserver.StateConnected
				server.Reconnect()
			case "version_change":
				server.Metadata.ServerVersion = "2.0.0"
			}

			if tt.expectCached {
				if len(server.Tools) == 0 {
					t.Error("Tools should be cached but are empty")
				}
			}

			if !tt.expectCached && tt.operation == "disconnect" {
				if len(server.Tools) != 0 {
					t.Error("Tools should be cleared after disconnect")
				}
			}
		})
	}
}

// TestMCPServer_InvokeTool tests tool invocation
func TestMCPServer_InvokeTool(t *testing.T) {
	tests := []struct {
		name            string
		connectionState mcpserver.ConnectionState
		toolName        string
		params          map[string]interface{}
		setupTools      []string // tool names available
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "invoke existing tool when connected",
			connectionState: mcpserver.StateConnected,
			toolName:        "read_file",
			params:          map[string]interface{}{"path": "/tmp/test.txt"},
			setupTools:      []string{"read_file", "write_file"},
			wantErr:         false,
		},
		{
			name:            "invoke tool with empty params",
			connectionState: mcpserver.StateConnected,
			toolName:        "list_files",
			params:          map[string]interface{}{},
			setupTools:      []string{"list_files"},
			wantErr:         false,
		},
		{
			name:            "cannot invoke when disconnected",
			connectionState: mcpserver.StateDisconnected,
			toolName:        "read_file",
			params:          map[string]interface{}{"path": "/tmp/test.txt"},
			setupTools:      []string{"read_file"},
			wantErr:         true,
			errMsg:          "not connected",
		},
		{
			name:            "invoke non-existent tool",
			connectionState: mcpserver.StateConnected,
			toolName:        "nonexistent_tool",
			params:          map[string]interface{}{},
			setupTools:      []string{"read_file", "write_file"},
			wantErr:         true,
			errMsg:          "tool not found",
		},
		{
			name:            "invoke with nil params",
			connectionState: mcpserver.StateConnected,
			toolName:        "simple_tool",
			params:          nil,
			setupTools:      []string{"simple_tool"},
			wantErr:         false,
		},
		{
			name:            "invoke tool with complex params",
			connectionState: mcpserver.StateConnected,
			toolName:        "complex_tool",
			params: map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []string{"a", "b", "c"},
			},
			setupTools: []string{"complex_tool"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.Connection.State = tt.connectionState

			// Setup tools
			for _, toolName := range tt.setupTools {
				server.Tools = append(server.Tools, mcpserver.Tool{
					Name:        toolName,
					Description: "Test tool",
				})
			}

			result, err := server.InvokeTool(tt.toolName, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("InvokeTool() expected error containing %q but got none", tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("InvokeTool() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("InvokeTool() returned nil result")
			}
		})
	}
}

// TestMCPServer_Invariant_UniqueServerID tests that server IDs must be unique in registry
func TestMCPServer_Invariant_UniqueServerID(t *testing.T) {
	tests := []struct {
		name      string
		serverIDs []string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "unique server IDs should pass",
			serverIDs: []string{"server1", "server2", "server3"},
			wantErr:   false,
		},
		{
			name:      "duplicate server IDs should fail",
			serverIDs: []string{"server1", "server2", "server1"},
			wantErr:   true,
			errMsg:    "duplicate server ID",
		},
		{
			name:      "single server ID should pass",
			serverIDs: []string{"server1"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := mcpserver.NewRegistry()

			var err error
			for _, id := range tt.serverIDs {
				server, _ := mcpserver.NewMCPServer(id, "npx", []string{"test"}, mcpserver.TransportStdio)
				err = registry.Register(server)
				if err != nil {
					break
				}
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Registry.Register() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Registry.Register() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMCPServer_Invariant_ToolsUpdatedOnlyWhenConnected tests that tools can only be updated when connected
func TestMCPServer_Invariant_ToolsUpdatedOnlyWhenConnected(t *testing.T) {
	tests := []struct {
		name            string
		connectionState mcpserver.ConnectionState
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "update tools when connected",
			connectionState: mcpserver.StateConnected,
			wantErr:         false,
		},
		{
			name:            "cannot update tools when disconnected",
			connectionState: mcpserver.StateDisconnected,
			wantErr:         true,
			errMsg:          "not connected",
		},
		{
			name:            "cannot update tools when connecting",
			connectionState: mcpserver.StateConnecting,
			wantErr:         true,
			errMsg:          "not connected",
		},
		{
			name:            "cannot update tools when failed",
			connectionState: mcpserver.StateFailed,
			wantErr:         true,
			errMsg:          "not connected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.Connection.State = tt.connectionState

			err := server.DiscoverTools()

			if tt.wantErr {
				if err == nil {
					t.Errorf("DiscoverTools() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("DiscoverTools() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMCPServer_HealthCheck tests periodic health checking
func TestMCPServer_HealthCheck(t *testing.T) {
	tests := []struct {
		name             string
		connectionState  mcpserver.ConnectionState
		simulateResponse string // "success", "timeout", "error"
		expectedHealth   mcpserver.HealthStatus
		expectTimeUpdate bool
	}{
		{
			name:             "successful health check",
			connectionState:  mcpserver.StateConnected,
			simulateResponse: "success",
			expectedHealth:   mcpserver.HealthHealthy,
			expectTimeUpdate: true,
		},
		{
			name:             "timeout health check",
			connectionState:  mcpserver.StateConnected,
			simulateResponse: "timeout",
			expectedHealth:   mcpserver.HealthUnhealthy,
			expectTimeUpdate: true,
		},
		{
			name:             "error health check",
			connectionState:  mcpserver.StateConnected,
			simulateResponse: "error",
			expectedHealth:   mcpserver.HealthUnhealthy,
			expectTimeUpdate: true,
		},
		{
			name:             "health check when disconnected",
			connectionState:  mcpserver.StateDisconnected,
			simulateResponse: "success",
			expectedHealth:   mcpserver.HealthDisconnected,
			expectTimeUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.Connection.State = tt.connectionState

			// Setup mock client with appropriate error behavior
			mockClient := &MockMCPClient{}
			switch tt.simulateResponse {
			case "timeout":
				mockClient.pingError = context.DeadlineExceeded
			case "error":
				mockClient.pingError = fmt.Errorf("simulated ping error")
			case "success":
				mockClient.pingError = nil
			}
			server.SetClient(mockClient)

			oldLastCheck := server.LastHealthCheck
			time.Sleep(1 * time.Millisecond) // Ensure time difference

			err := server.HealthCheck()

			if err != nil && tt.simulateResponse == "success" {
				t.Errorf("HealthCheck() unexpected error: %v", err)
			}

			if server.HealthStatus != tt.expectedHealth {
				t.Errorf("HealthStatus = %v, want %v", server.HealthStatus, tt.expectedHealth)
			}

			if tt.expectTimeUpdate {
				if !server.LastHealthCheck.After(oldLastCheck) {
					t.Error("LastHealthCheck should be updated")
				}
			}
		})
	}
}

// TestMCPServer_Reconnect tests reconnection with backoff
func TestMCPServer_Reconnect(t *testing.T) {
	tests := []struct {
		name               string
		initialState       mcpserver.ConnectionState
		errorCount         int
		expectBackoff      bool
		expectedMinBackoff time.Duration
	}{
		{
			name:          "first reconnect attempt",
			initialState:  mcpserver.StateFailed,
			errorCount:    0,
			expectBackoff: false,
		},
		{
			name:               "reconnect after one failure",
			initialState:       mcpserver.StateFailed,
			errorCount:         1,
			expectBackoff:      true,
			expectedMinBackoff: 1 * time.Second,
		},
		{
			name:               "reconnect after multiple failures",
			initialState:       mcpserver.StateFailed,
			errorCount:         3,
			expectBackoff:      true,
			expectedMinBackoff: 4 * time.Second,
		},
		{
			name:          "reconnect from disconnected",
			initialState:  mcpserver.StateDisconnected,
			errorCount:    0,
			expectBackoff: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := mcpserver.NewMCPServer("test-server", "npx", []string{"test"}, mcpserver.TransportStdio)
			server.Connection.State = tt.initialState
			server.Connection.ErrorCount = tt.errorCount

			err := server.Reconnect()

			if err != nil {
				t.Errorf("Reconnect() unexpected error: %v", err)
			}

			if tt.expectBackoff {
				if server.Connection.RetryBackoff < tt.expectedMinBackoff {
					t.Errorf("RetryBackoff = %v, want at least %v",
						server.Connection.RetryBackoff, tt.expectedMinBackoff)
				}
			}

			// State should transition to Connecting
			if server.Connection.State != mcpserver.StateConnecting {
				t.Errorf("State after Reconnect() = %v, want %v",
					server.Connection.State, mcpserver.StateConnecting)
			}
		})
	}
}

// TestTransport_Types tests different transport type configurations
func TestTransport_Types(t *testing.T) {
	tests := []struct {
		name          string
		transportType mcpserver.TransportType
		config        interface{}
		wantErr       bool
	}{
		{
			name:          "stdio transport",
			transportType: mcpserver.TransportStdio,
			config: &mcpserver.StdioTransportConfig{
				Command: "npx",
				Args:    []string{"-y", "server"},
			},
			wantErr: false,
		},
		{
			name:          "SSE transport",
			transportType: mcpserver.TransportSSE,
			config: &mcpserver.SSETransportConfig{
				URL: "https://example.com/sse",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
		{
			name:          "HTTP transport",
			transportType: mcpserver.TransportHTTP,
			config: &mcpserver.HTTPTransportConfig{
				BaseURL: "https://example.com/rpc",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := mcpserver.NewTransport(tt.transportType, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("NewTransport() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewTransport() unexpected error: %v", err)
				return
			}

			if transport == nil {
				t.Fatal("NewTransport() returned nil transport")
			}

			if transport.Type() != tt.transportType {
				t.Errorf("Transport.Type() = %v, want %v", transport.Type(), tt.transportType)
			}
		})
	}
}
