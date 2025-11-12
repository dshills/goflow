package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// HTTPClient implements the Client interface using HTTP JSON-RPC transport
// This is a synchronous request-response protocol where each RPC call is a separate HTTP POST.
type HTTPClient struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client
	mu         sync.Mutex
	closed     bool
	connected  bool
}

// HTTPConfig holds configuration for HTTP transport
type HTTPConfig struct {
	BaseURL string
	Headers map[string]string
	Timeout time.Duration
}

// NewHTTPClient creates a new HTTP-based MCP client
func NewHTTPClient(config HTTPConfig) (*HTTPClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL cannot be empty")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPClient{
		baseURL: config.BaseURL,
		headers: config.Headers,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Connect establishes a connection to the MCP server
// For HTTP transport, this validates the endpoint and performs initialization
func (c *HTTPClient) Connect(ctx context.Context) error {
	c.mu.Lock()

	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}

	// Mark as connected before initialize
	c.connected = true
	c.mu.Unlock()

	// Initialize the MCP connection
	if err := c.initialize(ctx); err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// initialize sends the initialize request to the MCP server
func (c *HTTPClient) initialize(ctx context.Context) error {
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "goflow",
			"version": "0.1.0",
		},
	}

	resp, err := c.sendRequest(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %w", resp.Error)
	}

	// Send initialized notification (no response expected)
	notification := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	notifJSON, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal initialized notification: %w", err)
	}

	// Send notification as POST (we don't wait for response)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(notifJSON))
	if err != nil {
		return fmt.Errorf("failed to create notification request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			// Log error but don't fail - notification was already sent
			_ = err
		}
	}()

	// For notifications, we accept both 200 OK and 204 No Content
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("initialized notification failed with status %d: %s (body: %s)", httpResp.StatusCode, httpResp.Status, string(body))
	}

	return nil
}

// sendRequest sends a JSON-RPC request via HTTP POST and waits for the response
func (c *HTTPClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	req, err := newRequest(method, params)
	if err != nil {
		return nil, err
	}

	// Marshal request
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	// Send HTTP request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			// Log error but don't fail - response was already received
			_ = err
		}
	}()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d: %s (body: %s)", httpResp.StatusCode, httpResp.Status, string(body))
	}

	// Parse JSON-RPC response
	var resp JSONRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(body))
	}

	return &resp, nil
}

// Close terminates the connection to the MCP server
func (c *HTTPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.connected = false

	// For HTTP, we just close the http client's idle connections
	c.httpClient.CloseIdleConnections()

	return nil
}

// IsConnected returns true if the client is connected
func (c *HTTPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected && !c.closed
}

// ListTools retrieves all available tools from the server
func (c *HTTPClient) ListTools(ctx context.Context) ([]mcpserver.Tool, error) {
	resp, err := c.sendRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("tools/list request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %w", resp.Error)
	}

	// Parse response
	var result struct {
		Tools []struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description,omitempty"`
			InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools/list response: %w", err)
	}

	// Convert to domain tools
	tools := make([]mcpserver.Tool, len(result.Tools))
	for i, t := range result.Tools {
		tool := mcpserver.Tool{
			Name:        t.Name,
			Description: t.Description,
		}

		// Parse input schema if present
		if t.InputSchema != nil {
			schemaJSON, err := json.Marshal(t.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal input schema for tool %s: %w", t.Name, err)
			}

			var schema mcpserver.ToolSchema
			if err := json.Unmarshal(schemaJSON, &schema); err != nil {
				return nil, fmt.Errorf("failed to parse input schema for tool %s: %w", t.Name, err)
			}
			tool.InputSchema = &schema
		}

		tools[i] = tool
	}

	return tools, nil
}

// CallTool invokes a tool on the server with the given parameters
func (c *HTTPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	callParams := map[string]interface{}{
		"name":      toolName,
		"arguments": params,
	}

	resp, err := c.sendRequest(ctx, "tools/call", callParams)
	if err != nil {
		return nil, fmt.Errorf("tools/call request failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tools/call error: %w", resp.Error)
	}

	// Parse response as generic map
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools/call response: %w", err)
	}

	return result, nil
}

// Ping sends a ping request to verify server health
func (c *HTTPClient) Ping(ctx context.Context) error {
	// Set a shorter timeout for ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.sendRequest(pingCtx, "ping", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("ping error: %w", resp.Error)
	}

	return nil
}
