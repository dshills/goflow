package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/dshills/goflow/pkg/mcp"
	"github.com/dshills/goflow/pkg/mcpserver"
)

// getTestServerPath returns the absolute path to the test server main.go file.
// It uses multiple strategies to locate the file for maximum robustness:
// 1. Uses runtime.Caller to find the path relative to this source file
// 2. Validates the file exists before returning
// 3. Provides clear error messages if the file cannot be found
func getTestServerPath() (string, error) {
	// Strategy 1: Use runtime.Caller to get path relative to this file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to determine source file location")
	}

	testServerPath := filepath.Join(filepath.Dir(filename), "testserver", "main.go")

	// Validate the path exists
	if _, err := os.Stat(testServerPath); err == nil {
		return testServerPath, nil
	}

	// If not found, provide helpful error message
	return "", fmt.Errorf("test server not found at %s (expected at internal/testutil/testserver/main.go)", testServerPath)
}

// TestServerConfig returns the configuration for the test MCP server.
// It automatically locates the test server executable using robust path resolution.
// Panics if the test server cannot be found (fail-fast for test setup issues).
func TestServerConfig(serverID string) mcp.ServerConfig {
	testServerPath, err := getTestServerPath()
	if err != nil {
		// Panic in test utility code is acceptable - we want tests to fail fast
		// if the test infrastructure is misconfigured
		panic(fmt.Sprintf("test setup error: %v", err))
	}

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
	testServerPath, err := getTestServerPath()
	if err != nil {
		return nil, nil, fmt.Errorf("test setup error: %w", err)
	}

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
