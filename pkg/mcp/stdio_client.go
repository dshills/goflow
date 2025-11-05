package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// StdioClient implements the Client interface using stdio transport
type StdioClient struct {
	config          ServerConfig
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	stderr          io.ReadCloser
	scanner         *bufio.Scanner
	mu              sync.Mutex
	closed          bool
	pendingRequests map[interface{}]chan *JSONRPCResponse
	readerDone      chan error
}

// NewStdioClient creates a new stdio-based MCP client
func NewStdioClient(config ServerConfig) (*StdioClient, error) {
	if config.Command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	return &StdioClient{
		config:          config,
		pendingRequests: make(map[interface{}]chan *JSONRPCResponse),
		readerDone:      make(chan error, 1),
	}, nil
}

// Connect establishes a connection to the MCP server
func (c *StdioClient) Connect(ctx context.Context) error {
	c.mu.Lock()

	if c.cmd != nil {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}

	// Create command with context for timeout support
	c.cmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)

	// Set up environment variables if provided
	if len(c.config.Env) > 0 {
		env := c.cmd.Env
		for key, value := range c.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		c.cmd.Env = env
	}

	// Set up stdin pipe
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stdin = stdin

	// Set up stdout pipe
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	c.stdout = stdout
	c.scanner = bufio.NewScanner(stdout)

	// Set up stderr pipe for debugging
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	c.stderr = stderr

	// Start the process
	if err := c.cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Start background reader for responses
	go c.readResponses()

	// Release lock before initialize (which will call sendRequest)
	c.mu.Unlock()

	// Initialize the MCP connection
	if err := c.initialize(ctx); err != nil {
		_ = c.Close() // Use Close instead of closeWithoutLock since we don't hold the lock
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// initialize sends the initialize request to the MCP server
func (c *StdioClient) initialize(ctx context.Context) error {
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

	if _, err := fmt.Fprintf(c.stdin, "%s\n", notifJSON); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// sendRequest sends a JSON-RPC request and waits for the response
func (c *StdioClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
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

	// Marshal and send request
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send the request (don't hold lock during I/O)
	_, err = fmt.Fprintf(c.stdin, "%s\n", reqJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			return nil, fmt.Errorf("connection closed")
		}
		return resp, nil
	}
}

// readResponses reads JSON-RPC responses from stdout
func (c *StdioClient) readResponses() {
	defer func() {
		c.mu.Lock()
		// Notify all pending requests of closure
		for _, ch := range c.pendingRequests {
			close(ch)
		}
		c.pendingRequests = make(map[interface{}]chan *JSONRPCResponse)
		c.mu.Unlock()
	}()

	for c.scanner.Scan() {
		line := c.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Invalid JSON, skip
			continue
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

	if err := c.scanner.Err(); err != nil {
		c.readerDone <- err
	} else {
		c.readerDone <- io.EOF
	}
}

// Close terminates the connection to the MCP server
func (c *StdioClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeWithoutLock()
}

// closeWithoutLock closes the client without acquiring the lock
func (c *StdioClient) closeWithoutLock() error {
	if c.closed {
		return nil
	}

	c.closed = true

	// Close pipes
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.stderr != nil {
		_ = c.stderr.Close()
	}

	// Kill process if still running
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}

	return nil
}

// IsConnected returns true if the client is connected
func (c *StdioClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cmd != nil && !c.closed
}

// ListTools retrieves all available tools from the server
func (c *StdioClient) ListTools(ctx context.Context) ([]mcpserver.Tool, error) {
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
func (c *StdioClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
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
func (c *StdioClient) Ping(ctx context.Context) error {
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
