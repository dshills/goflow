package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcp"
)

// TestHTTPClient_ConnectionLifecycle tests basic HTTP client connection lifecycle
func TestHTTPClient_ConnectionLifecycle(t *testing.T) {
	// Create a test HTTP server
	server := newTestHTTPServer()
	defer server.Close()

	ctx := context.Background()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
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

// TestHTTPClient_InvalidURL tests error handling for invalid URLs
func TestHTTPClient_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: "http://localhost:99999/nonexistent",
		Timeout: 1 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail with invalid URL, got nil error")
		_ = client.Close()
	}
}

// TestHTTPClient_EmptyBaseURL tests validation of empty base URL
func TestHTTPClient_EmptyBaseURL(t *testing.T) {
	config := mcp.HTTPConfig{
		BaseURL: "",
		Timeout: 10 * time.Second,
	}

	_, err := mcp.NewHTTPClient(config)
	if err == nil {
		t.Error("Expected error when creating client with empty BaseURL")
	}
}

// TestHTTPClient_CustomHeaders tests custom header support
func TestHTTPClient_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header

	// Create server that captures headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, _ := req["method"].(string)
		var resp map[string]interface{}

		if method == "initialize" {
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo": map[string]interface{}{
						"name":    "test-server",
						"version": "1.0.0",
					},
				},
			}
		} else {
			// For initialized notification, just return 200
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "test-value",
		},
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Verify custom headers were sent
	if receivedHeaders.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Expected Authorization header, got: %s", receivedHeaders.Get("Authorization"))
	}
	if receivedHeaders.Get("X-Custom") != "test-value" {
		t.Errorf("Expected X-Custom header, got: %s", receivedHeaders.Get("X-Custom"))
	}
}

// TestHTTPClient_Timeout tests timeout handling
func TestHTTPClient_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 500 * time.Millisecond,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to timeout, got nil error")
		_ = client.Close()
	}
}

// TestHTTPClient_DoubleConnect tests that double connect returns error
func TestHTTPClient_DoubleConnect(t *testing.T) {
	server := newTestHTTPServer()
	defer server.Close()

	ctx := context.Background()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// First connect should succeed
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Second connect should fail
	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected error on double connect, got nil")
	}
}

// newTestHTTPServer creates a test HTTP server that implements basic MCP protocol
func newTestHTTPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, _ := req["method"].(string)

		// If it's a notification (no ID), just return 200
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Otherwise, return appropriate response based on method
		var resp map[string]interface{}
		switch method {
		case "initialize":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo": map[string]interface{}{
						"name":    "test-server",
						"version": "1.0.0",
					},
				},
			}
		case "tools/list":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "echo",
							"description": "Echo a message",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"message": map[string]interface{}{
										"type": "string",
									},
								},
								"required": []string{"message"},
							},
						},
					},
				},
			}
		case "tools/call":
			params, _ := req["params"].(map[string]interface{})
			toolName, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]interface{})

			if toolName == "echo" {
				message, _ := args["message"].(string)
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"result": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": message,
							},
						},
					},
				}
			} else {
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"error": map[string]interface{}{
						"code":    -32601,
						"message": "Tool not found",
					},
				}
			}
		case "ping":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  map[string]interface{}{},
			}
		default:
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// TestHTTPClient_ToolDiscovery tests tool discovery via HTTP
func TestHTTPClient_ToolDiscovery(t *testing.T) {
	server := newTestHTTPServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
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
	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	foundEcho := false
	for _, tool := range tools {
		if tool.Name == "echo" {
			foundEcho = true
			if tool.InputSchema == nil {
				t.Error("Expected echo tool to have input schema")
			}
		}
	}

	if !foundEcho {
		t.Error("Expected to find 'echo' tool")
	}
}

// TestHTTPClient_ToolInvocation tests invoking tools via HTTP
func TestHTTPClient_ToolInvocation(t *testing.T) {
	server := newTestHTTPServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
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
		"message": "Hello, HTTP!",
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

	if text != "Hello, HTTP!" {
		t.Errorf("Expected echoed text 'Hello, HTTP!', got '%s'", text)
	}
}

// TestHTTPClient_Ping tests ping functionality
func TestHTTPClient_Ping(t *testing.T) {
	server := newTestHTTPServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test ping
	err = client.Ping(ctx)
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

// TestHTTPClient_ErrorHandling tests error responses from HTTP server
func TestHTTPClient_ErrorHandling(t *testing.T) {
	server := newTestHTTPServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
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
}

// TestHTTPClient_ServerError tests handling of HTTP server errors
func TestHTTPClient_ServerError(t *testing.T) {
	// Create server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected error when server returns 500, got nil")
		_ = client.Close()
	}
}

// TestHTTPClient_InvalidJSON tests handling of invalid JSON responses
func TestHTTPClient_InvalidJSON(t *testing.T) {
	// Create server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected error when server returns invalid JSON, got nil")
		_ = client.Close()
	}
}

// TestHTTPClient_ContentType tests that client sends correct Content-Type
func TestHTTPClient_ContentType(t *testing.T) {
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, _ := req["method"].(string)
		if method == "initialize" {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.HTTPConfig{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewHTTPClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if receivedContentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", receivedContentType)
	}
}
