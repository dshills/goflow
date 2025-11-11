package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/dshills/goflow/pkg/mcp"
	"github.com/dshills/goflow/pkg/mcpserver"
)

// TestServerConfig returns the configuration for the test MCP server
func TestServerConfig(serverID string) mcp.ServerConfig {
	_, filename, _, _ := runtime.Caller(0)
	testServerPath := filepath.Join(filepath.Dir(filename), "testserver", "main.go")

	return mcp.ServerConfig{
		ID:      serverID,
		Command: "go",
		Args:    []string{"run", testServerPath},
	}
}

// StartTestServer creates and connects to a test MCP server
// Returns the client and a cleanup function
func StartTestServer(ctx context.Context, serverID string) (*mcp.StdioClient, func(), error) {
	config := TestServerConfig(serverID)

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	cleanup := func() {
		_ = client.Close()
	}

	return client, cleanup, nil
}

// CreateTestMCPServer creates an MCPServer instance configured with the test server
func CreateTestMCPServer(ctx context.Context, serverID string) (*mcpserver.MCPServer, func(), error) {
	_, filename, _, _ := runtime.Caller(0)
	testServerPath := filepath.Join(filepath.Dir(filename), "testserver", "main.go")

	// Create MCP server instance
	server, err := mcpserver.NewMCPServer(serverID, "go", []string{"run", testServerPath}, mcpserver.TransportStdio)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Create and configure MCP client
	config := TestServerConfig(serverID)
	client, err := mcp.NewStdioClient(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Connect the client
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Create adapter and set it on the server
	adapter := mcpserver.NewClientAdapter(client)
	server.SetClient(adapter)

	// Update server state
	if err := server.Connect(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("failed to update server state: %w", err)
	}

	if err := server.CompleteConnection(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("failed to complete connection: %w", err)
	}

	// Discover tools
	if err := server.DiscoverTools(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("failed to discover tools: %w", err)
	}

	cleanup := func() {
		_ = server.Disconnect()
		_ = client.Close()
	}

	return server, cleanup, nil
}
