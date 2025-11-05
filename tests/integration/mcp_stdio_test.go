package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcp"
)

// TestMCPStdio_ConnectionLifecycle tests basic MCP server connection lifecycle
func TestMCPStdio_ConnectionLifecycle(t *testing.T) {
	ctx := context.Background()

	// Get path to mock server
	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect to server
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Verify connection is active
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	// Close connection
	err = client.Close()
	if err != nil {
		t.Errorf("Failed to close connection: %v", err)
	}

	// Verify connection is closed
	if client.IsConnected() {
		t.Error("Expected client to be disconnected")
	}
}

// TestMCPStdio_ToolDiscovery tests MCP tool discovery via tools/list
func TestMCPStdio_ToolDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Discover tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Verify expected tools are present
	expectedTools := map[string]bool{
		"echo":       false,
		"read_file":  false,
		"write_file": false,
	}

	for _, tool := range tools {
		if _, exists := expectedTools[tool.Name]; exists {
			expectedTools[tool.Name] = true
		}
	}

	for toolName, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool '%s' not found in discovery", toolName)
		}
	}

	// Verify tool schemas
	for _, tool := range tools {
		if tool.Name == "echo" {
			if tool.InputSchema == nil {
				t.Error("Expected echo tool to have input schema")
			}
		}
	}
}

// TestMCPStdio_ToolInvocation tests invoking MCP tools
func TestMCPStdio_ToolInvocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test echo tool
	result, err := client.CallTool(ctx, "echo", map[string]interface{}{
		"message": "Hello, MCP!",
	})
	if err != nil {
		t.Fatalf("Failed to call echo tool: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Verify result structure
	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("Expected result to have content array")
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected content item to be a map")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		t.Fatal("Expected content to have text field")
	}

	if text != "Hello, MCP!" {
		t.Errorf("Expected echoed text 'Hello, MCP!', got '%s'", text)
	}
}

// TestMCPStdio_ToolInvocationWithFiles tests file operations via MCP
func TestMCPStdio_ToolInvocationWithFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create a temporary file for testing
	tmpPath := filepath.Join(t.TempDir(), "test.txt")
	testContent := "Test file content"

	// Write file via MCP
	_, err = client.CallTool(ctx, "write_file", map[string]interface{}{
		"path":    tmpPath,
		"content": testContent,
	})
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Read file via MCP
	result, err := client.CallTool(ctx, "read_file", map[string]interface{}{
		"path": tmpPath,
	})
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Verify content
	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("Expected result to have content array")
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected content item to be a map")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		t.Fatal("Expected content to have text field")
	}

	if text != testContent {
		t.Errorf("Expected file content '%s', got '%s'", testContent, text)
	}
}

// TestMCPStdio_ConnectionTimeout tests connection timeout handling
func TestMCPStdio_ConnectionTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "timeout-test",
		Command: "sleep",
		Args:    []string{"10"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to timeout, got nil error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got: %v", ctx.Err())
	}
}

// TestMCPStdio_InvalidCommand tests error handling for invalid commands
func TestMCPStdio_InvalidCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "invalid-test",
		Command: "nonexistent-command-12345",
		Args:    []string{},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail with invalid command, got nil error")
	}
}

// TestMCPStdio_MultipleClients tests multiple concurrent client connections
func TestMCPStdio_MultipleClients(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	clients := make([]*mcp.StdioClient, 3)
	for i := 0; i < 3; i++ {
		client, err := mcp.NewStdioClient(config)
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}
		defer client.Close()

		err = client.Connect(ctx)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}

		clients[i] = client
	}

	// Each client should be able to call tools independently
	for i, client := range clients {
		result, err := client.CallTool(ctx, "echo", map[string]interface{}{
			"message": "Client " + string(rune('0'+i)),
		})
		if err != nil {
			t.Errorf("Client %d failed to call tool: %v", i, err)
		}
		if result == nil {
			t.Errorf("Client %d got nil result", i)
		}
	}
}

// TestMCPStdio_ErrorHandling tests error responses from MCP server
func TestMCPStdio_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/mocks/mock_mcp_server.go")
	if err != nil {
		t.Fatalf("Failed to get mock server path: %v", err)
	}

	// This should fail because mcp.NewStdioClient doesn't exist yet
	config := mcp.ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath, "--mode=server"},
	}

	client, err := mcp.NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test calling non-existent tool
	_, err = client.CallTool(ctx, "nonexistent_tool", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error when calling non-existent tool, got nil")
	}

	// Test calling with invalid parameters
	_, err = client.CallTool(ctx, "echo", map[string]interface{}{
		"invalid_param": "value",
	})
	if err == nil {
		t.Error("Expected error with invalid parameters, got nil")
	}

	// Test reading non-existent file
	_, err = client.CallTool(ctx, "read_file", map[string]interface{}{
		"path": "/nonexistent/path/to/file.txt",
	})
	if err == nil {
		t.Error("Expected error when reading non-existent file, got nil")
	}
}
