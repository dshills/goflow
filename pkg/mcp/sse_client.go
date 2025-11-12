package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// SSEClient implements the Client interface using Server-Sent Events transport
// SSE is a unidirectional protocol where the server pushes events to the client.
// For MCP, we use SSE for server->client messages and POST requests for client->server.
type SSEClient struct {
	url             string
	headers         map[string]string
	httpClient      *http.Client
	sseConn         *http.Response
	mu              sync.Mutex
	closed          bool
	connected       bool
	pendingRequests map[interface{}]chan *JSONRPCResponse
	readerDone      chan error
}

// SSEConfig holds configuration for SSE transport
type SSEConfig struct {
	URL     string
	Headers map[string]string
	Timeout time.Duration
}

// NewSSEClient creates a new SSE-based MCP client
func NewSSEClient(config SSEConfig) (*SSEClient, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &SSEClient{
		url:     config.URL,
		headers: config.Headers,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		pendingRequests: make(map[interface{}]chan *JSONRPCResponse),
		readerDone:      make(chan error, 1),
	}, nil
}

// Connect establishes a connection to the MCP server via SSE
func (c *SSEClient) Connect(ctx context.Context) error {
	c.mu.Lock()

	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}

	// Create SSE connection request
	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Add custom headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Establish SSE connection
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		c.mu.Unlock()
		return fmt.Errorf("SSE connection failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	c.sseConn = resp
	c.connected = true

	// Start background reader for SSE events
	go c.readSSEEvents()

	// Release lock before initialize
	c.mu.Unlock()

	// Initialize the MCP connection
	if err := c.initialize(ctx); err != nil {
		_ = c.Close()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// initialize sends the initialize request to the MCP server
func (c *SSEClient) initialize(ctx context.Context) error {
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
	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
	}

	if err := c.sendNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// sendRequest sends a JSON-RPC request via POST and waits for the response via SSE
func (c *SSEClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	req, err := newRequest(method, params)
	if err != nil {
		return nil, err
	}

	// Create response channel for this request
	respChan := make(chan *JSONRPCResponse, 1)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.pendingRequests[req.ID] = respChan
	c.mu.Unlock()

	// Clean up pending request on exit
	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, req.ID)
		c.mu.Unlock()
	}()

	// Marshal request
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send via POST request - note we don't wait for response here, it comes via SSE
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, strings.NewReader(string(reqJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	// Use a goroutine to send the POST so we don't block waiting for SSE response
	sendDone := make(chan error, 1)
	go func() {
		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			sendDone <- fmt.Errorf("failed to send POST request: %w", err)
			return
		}
		defer func() {
			if err := httpResp.Body.Close(); err != nil {
				// Log error but don't fail - POST was already sent
				_ = err
			}
		}()

		if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted && httpResp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(httpResp.Body)
			sendDone <- fmt.Errorf("POST request failed with status %d: %s (body: %s)", httpResp.StatusCode, httpResp.Status, string(body))
			return
		}
		sendDone <- nil
	}()

	// Wait for either send to complete with error, or response via SSE
	select {
	case err := <-sendDone:
		if err != nil {
			return nil, err
		}
		// Send succeeded, now wait for SSE response
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case resp, ok := <-respChan:
			if !ok {
				return nil, fmt.Errorf("connection closed")
			}
			return resp, nil
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return resp, nil
	}
}

// sendNotification sends a JSON-RPC notification (no response expected)
func (c *SSEClient) sendNotification(ctx context.Context, notification interface{}) error {
	notifJSON, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, strings.NewReader(string(notifJSON)))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			// Log error but don't fail - notification was already sent
			_ = err
		}
	}()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("notification failed with status %d: %s", httpResp.StatusCode, httpResp.Status)
	}

	return nil
}

// readSSEEvents reads Server-Sent Events from the connection
func (c *SSEClient) readSSEEvents() {
	defer func() {
		c.mu.Lock()
		// Notify all pending requests of closure
		for _, ch := range c.pendingRequests {
			close(ch)
		}
		c.pendingRequests = make(map[interface{}]chan *JSONRPCResponse)
		c.mu.Unlock()
	}()

	reader := bufio.NewReader(c.sseConn.Body)
	var eventData strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				c.readerDone <- io.EOF
			} else {
				c.readerDone <- err
			}
			return
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line indicates end of event
		if line == "" {
			if eventData.Len() > 0 {
				c.processSSEEvent(eventData.String())
				eventData.Reset()
			}
			continue
		}

		// Parse SSE field
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if eventData.Len() > 0 {
				eventData.WriteString("\n")
			}
			eventData.WriteString(data)
		}
		// Ignore other SSE fields (event, id, retry, comment)
	}
}

// processSSEEvent processes a complete SSE event containing JSON-RPC response
func (c *SSEClient) processSSEEvent(data string) {
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		// Invalid JSON, skip
		return
	}

	// Route response to waiting request
	c.mu.Lock()
	if ch, ok := c.pendingRequests[resp.ID]; ok {
		select {
		case ch <- &resp:
		default:
			// Channel full, skip
		}
	}
	c.mu.Unlock()
}

// Close terminates the connection to the MCP server
func (c *SSEClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.connected = false

	// Close SSE connection
	if c.sseConn != nil {
		_ = c.sseConn.Body.Close()
	}

	return nil
}

// IsConnected returns true if the client is connected
func (c *SSEClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected && !c.closed
}

// ListTools retrieves all available tools from the server
func (c *SSEClient) ListTools(ctx context.Context) ([]mcpserver.Tool, error) {
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
func (c *SSEClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
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
func (c *SSEClient) Ping(ctx context.Context) error {
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
