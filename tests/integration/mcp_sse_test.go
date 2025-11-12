package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcp"
)

// TestSSEClient_ConnectionLifecycle tests basic SSE client connection lifecycle
func TestSSEClient_ConnectionLifecycle(t *testing.T) {
	// Create a test SSE server
	server := newTestSSEServer()
	defer server.Close()

	ctx := context.Background()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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

// TestSSEClient_InvalidURL tests error handling for invalid URLs
func TestSSEClient_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     "http://localhost:99999/nonexistent",
		Timeout: 1 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail with invalid URL, got nil error")
		_ = client.Close()
	}
}

// TestSSEClient_EmptyURL tests validation of empty URL
func TestSSEClient_EmptyURL(t *testing.T) {
	config := mcp.SSEConfig{
		URL:     "",
		Timeout: 10 * time.Second,
	}

	_, err := mcp.NewSSEClient(config)
	if err == nil {
		t.Error("Expected error when creating client with empty URL")
	}
}

// TestSSEClient_CustomHeaders tests custom header support
func TestSSEClient_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	responseChannel := make(chan map[string]interface{}, 10)

	// Create server that captures headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			receivedHeaders = r.Header.Clone()
			// SSE connection
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher := w.(http.Flusher)

			// Keep connection alive and send responses
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-r.Context().Done():
					return
				case resp := <-responseChannel:
					respJSON, _ := json.Marshal(resp)
					fmt.Fprintf(w, "data: %s\n\n", respJSON)
					flusher.Flush()
				case <-ticker.C:
					fmt.Fprintf(w, ": keepalive\n\n")
					flusher.Flush()
				}
			}
		} else {
			// POST request - decode and send response via SSE
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			// If it's a notification (no ID), just return 200
			if _, hasID := req["id"]; !hasID {
				w.WriteHeader(http.StatusOK)
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
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"result":  map[string]interface{}{},
				}
			}

			select {
			case responseChannel <- resp:
				w.WriteHeader(http.StatusAccepted)
			case <-time.After(100 * time.Millisecond):
				http.Error(w, "SSE channel full", http.StatusServiceUnavailable)
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "test-value",
		},
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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

// TestSSEClient_Timeout tests timeout handling
func TestSSEClient_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 500 * time.Millisecond,
	}

	client, err := mcp.NewSSEClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to timeout, got nil error")
		_ = client.Close()
	}
}

// TestSSEClient_DoubleConnect tests that double connect returns error
func TestSSEClient_DoubleConnect(t *testing.T) {
	server := newTestSSEServer()
	defer server.Close()

	ctx := context.Background()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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

// newTestSSEServer creates a test SSE server that implements basic MCP protocol
func newTestSSEServer() *httptest.Server {
	requestCounter := 0
	responseChannel := make(chan map[string]interface{}, 10)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// SSE connection
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming not supported", http.StatusInternalServerError)
				return
			}

			// Keep connection alive and send responses
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-r.Context().Done():
					return
				case resp := <-responseChannel:
					// Send SSE event with JSON-RPC response
					respJSON, _ := json.Marshal(resp)
					fmt.Fprintf(w, "data: %s\n\n", respJSON)
					flusher.Flush()
				case <-ticker.C:
					// Send keepalive comment
					fmt.Fprintf(w, ": keepalive\n\n")
					flusher.Flush()
				}
			}
		} else if r.Method == "POST" {
			// Handle POST requests (initialize, notifications, tool calls)
			requestCounter++

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

			// Otherwise, prepare response and send via SSE channel
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

			// Send response via SSE channel
			select {
			case responseChannel <- resp:
				w.WriteHeader(http.StatusAccepted)
			case <-time.After(100 * time.Millisecond):
				http.Error(w, "SSE channel full", http.StatusServiceUnavailable)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

// TestSSEClient_ToolDiscovery tests tool discovery via SSE
func TestSSEClient_ToolDiscovery(t *testing.T) {
	server := newTestSSEServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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

// TestSSEClient_ToolInvocation tests invoking tools via SSE
func TestSSEClient_ToolInvocation(t *testing.T) {
	server := newTestSSEServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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
		"message": "Hello, SSE!",
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
}

// TestSSEClient_Ping tests ping functionality
func TestSSEClient_Ping(t *testing.T) {
	server := newTestSSEServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
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

// TestSSEClient_SSEEventParsing tests parsing of SSE events
func TestSSEClient_SSEEventParsing(t *testing.T) {
	responseChannel := make(chan map[string]interface{}, 10)

	// Create server that sends multiline SSE data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher := w.(http.Flusher)

			// Keep connection alive and send responses
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-r.Context().Done():
					return
				case resp := <-responseChannel:
					// Send as multiline SSE data
					respJSON, _ := json.Marshal(resp)
					lines := strings.Split(string(respJSON), ",")
					for i, line := range lines {
						if i < len(lines)-1 {
							line += ","
						}
						fmt.Fprintf(w, "data: %s\n", line)
					}
					fmt.Fprintf(w, "\n")
					flusher.Flush()
				case <-ticker.C:
					fmt.Fprintf(w, ": keepalive\n\n")
					flusher.Flush()
				}
			}
		} else {
			// POST request
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			// If it's a notification (no ID), just return 200
			if _, hasID := req["id"]; !hasID {
				w.WriteHeader(http.StatusOK)
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
							"name":    "test",
							"version": "1.0",
						},
					},
				}
			} else {
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req["id"],
					"result":  map[string]interface{}{},
				}
			}

			select {
			case responseChannel <- resp:
				w.WriteHeader(http.StatusAccepted)
			case <-time.After(100 * time.Millisecond):
				http.Error(w, "SSE channel full", http.StatusServiceUnavailable)
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := mcp.SSEConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	client, err := mcp.NewSSEClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Connection should succeed even with multiline SSE data
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect with multiline SSE: %v", err)
	}
}
