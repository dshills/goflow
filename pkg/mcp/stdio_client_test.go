package mcp

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStdioClient_BasicConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get path to test server
	mockServerPath, err := filepath.Abs("../../internal/testutil/testserver/main.go")
	if err != nil {
		t.Fatalf("Failed to get test server path: %v", err)
	}

	config := ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}

	client, err := NewStdioClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect to server
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Verify connection is active
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}

	// Test ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestStdioClient_ToolDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/testserver/main.go")
	if err != nil {
		t.Fatalf("Failed to get test server path: %v", err)
	}

	config := ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}

	client, err := NewStdioClient(config)
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
}

func TestStdioClient_ToolInvocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockServerPath, err := filepath.Abs("../../internal/testutil/testserver/main.go")
	if err != nil {
		t.Fatalf("Failed to get test server path: %v", err)
	}

	config := ServerConfig{
		ID:      "test-server",
		Command: "go",
		Args:    []string{"run", mockServerPath},
	}

	client, err := NewStdioClient(config)
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
		t.Fatalf("Expected result to have content array, got: %T", result["content"])
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
