package mcpserver

import (
	"context"
	"fmt"
)

// NewMCPServerWithClient creates a new MCP server with a configured client
// This is a convenience function that creates an MCPServer and sets up the
// MCP protocol client for stdio transport.
//
// The client must be provided as a parameter to allow for dependency injection
// and testing. In production code, you would typically use mcp.NewStdioClient.
//
// Example usage:
//
//	import "github.com/dshills/goflow/pkg/mcp"
//
//	config := mcp.ServerConfig{
//	    ID:      serverID,
//	    Command: command,
//	    Args:    args,
//	    Env:     env,
//	}
//	client, err := mcp.NewStdioClient(config)
//	if err != nil {
//	    return nil, err
//	}
//
//	server, err := mcpserver.NewMCPServerWithClient(serverID, command, args,
//	    mcpserver.TransportStdio, client)
//	if err != nil {
//	    return nil, err
//	}
//
//	// Connect and discover tools
//	ctx := context.Background()
//	if err := server.Connect(); err != nil {
//	    return nil, err
//	}
//	if err := client.Connect(ctx); err != nil {
//	    server.FailConnection(err.Error())
//	    return nil, err
//	}
//	server.CompleteConnection()
//
//	if err := server.DiscoverTools(); err != nil {
//	    return nil, err
//	}
func NewMCPServerWithClient(id, command string, args []string, transportType TransportType, client MCPClient) (*MCPServer, error) {
	// Create the basic MCPServer
	server, err := NewMCPServer(id, command, args, transportType)
	if err != nil {
		return nil, err
	}

	// Set the MCP client
	server.SetClient(client)

	return server, nil
}

// ConnectWithClient establishes a connection using the configured MCP client
// This is a convenience method that handles the connection lifecycle:
// 1. Transitions the server to connecting state
// 2. Calls Connect on the MCP client
// 3. On success, completes the connection
// 4. On failure, marks the connection as failed
//
// Returns an error if no client is configured or if the connection fails.
func (s *MCPServer) ConnectWithClient(ctx context.Context) error {
	if s.client == nil {
		return NewConnectionError("no MCP client configured")
	}

	// Start the connection process
	if err := s.Connect(); err != nil {
		return err
	}

	// Attempt to connect via the client
	if err := s.client.Connect(ctx); err != nil {
		// Connection failed, update state
		_ = s.FailConnection(fmt.Sprintf("client connection failed: %v", err))
		return fmt.Errorf("failed to connect client: %w", err)
	}

	// Connection successful
	if err := s.CompleteConnection(); err != nil {
		return err
	}

	return nil
}

// DisconnectWithClient closes the connection using the configured MCP client
// This is a convenience method that handles the disconnection lifecycle:
// 1. Closes the MCP client connection
// 2. Transitions the server to disconnected state
//
// Returns an error if the client close fails, but still disconnects the server.
func (s *MCPServer) DisconnectWithClient() error {
	var clientErr error
	if s.client != nil {
		clientErr = s.client.Close()
	}

	// Always disconnect the server state, even if client close failed
	if err := s.Disconnect(); err != nil {
		return err
	}

	return clientErr
}
