package mocks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// MockMCPServer is a minimal MCP server implementation for testing
// It supports stdio transport and provides echo, read_file, write_file tools
type MockMCPServer struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// NewMockMCPServer creates a new mock MCP server
func NewMockMCPServer() *MockMCPServer {
	return &MockMCPServer{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// JSONRPCRequest represents an MCP JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an MCP JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolCallParams represents parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Run starts the mock MCP server and processes requests
func (s *MockMCPServer) Run() error {
	scanner := bufio.NewScanner(s.stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.writeError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}

	return scanner.Err()
}

func (s *MockMCPServer) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "ping":
		s.handlePing(req)
	default:
		s.writeError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *MockMCPServer) handleInitialize(req *JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mock-mcp-server",
			"version": "0.1.0",
		},
	}
	s.writeResponse(req.ID, result)
}

func (s *MockMCPServer) handleToolsList(req *JSONRPCRequest) {
	tools := []map[string]interface{}{
		{
			"name":        "echo",
			"description": "Echoes back the input message",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message to echo",
					},
				},
				"required": []string{"message"},
			},
		},
		{
			"name":        "read_file",
			"description": "Reads content from a file",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "write_file",
			"description": "Writes content to a file",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}
	s.writeResponse(req.ID, result)
}

func (s *MockMCPServer) handleToolsCall(req *JSONRPCRequest) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	switch params.Name {
	case "echo":
		s.handleEcho(req.ID, params.Arguments)
	case "read_file":
		s.handleReadFile(req.ID, params.Arguments)
	case "write_file":
		s.handleWriteFile(req.ID, params.Arguments)
	default:
		s.writeError(req.ID, -32602, "Unknown tool", params.Name)
	}
}

func (s *MockMCPServer) handleEcho(id interface{}, args map[string]interface{}) {
	message, ok := args["message"].(string)
	if !ok {
		s.writeError(id, -32602, "Invalid params", "message must be a string")
		return
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": message,
			},
		},
	}
	s.writeResponse(id, result)
}

func (s *MockMCPServer) handleReadFile(id interface{}, args map[string]interface{}) {
	path, ok := args["path"].(string)
	if !ok {
		s.writeError(id, -32602, "Invalid params", "path must be a string")
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		s.writeError(id, -32603, "Internal error", err.Error())
		return
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(content),
			},
		},
	}
	s.writeResponse(id, result)
}

func (s *MockMCPServer) handleWriteFile(id interface{}, args map[string]interface{}) {
	path, ok := args["path"].(string)
	if !ok {
		s.writeError(id, -32602, "Invalid params", "path must be a string")
		return
	}

	content, ok := args["content"].(string)
	if !ok {
		s.writeError(id, -32602, "Invalid params", "content must be a string")
		return
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		s.writeError(id, -32603, "Internal error", err.Error())
		return
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
			},
		},
	}
	s.writeResponse(id, result)
}

func (s *MockMCPServer) handlePing(req *JSONRPCRequest) {
	s.writeResponse(req.ID, map[string]interface{}{})
}

func (s *MockMCPServer) writeResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.write(resp)
}

func (s *MockMCPServer) writeError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.write(resp)
}

func (s *MockMCPServer) write(resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		_, _ = fmt.Fprintf(s.stderr, "Error marshaling response: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(s.stdout, "%s\n", data)
}

// GetToolExecutable returns the path to a mock MCP server executable
// This should be called with "go run" to start the mock server
func GetToolExecutable() string {
	// Return the path to this mock server so it can be executed
	return "go run internal/testutil/mocks/mock_mcp_server.go"
}

// StartMockServer starts a mock MCP server for testing
// This is typically called via exec.Command
func StartMockServer(args []string) error {
	if len(args) > 0 && args[0] == "--mode=server" {
		server := NewMockMCPServer()
		return server.Run()
	}
	return fmt.Errorf("invalid arguments: expected --mode=server")
}

// IsServerMode checks if the current process should run as a mock server
func IsServerMode() bool {
	return len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "--mode=server")
}
