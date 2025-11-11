package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// TestMCPServer is a minimal MCP server for integration testing
// It implements the MCP protocol and provides basic tools for testing
type TestMCPServer struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

func main() {
	server := &TestMCPServer{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func (s *TestMCPServer) Run() error {
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

func (s *TestMCPServer) handleRequest(req *JSONRPCRequest) {
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

func (s *TestMCPServer) handleInitialize(req *JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "goflow-test-server",
			"version": "0.1.0",
		},
	}
	s.writeResponse(req.ID, result)
}

func (s *TestMCPServer) handleToolsList(req *JSONRPCRequest) {
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
		{
			"name":        "failing_tool",
			"description": "A tool that always fails (for error testing)",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "delay_task",
			"description": "Simulates a delayed task (for concurrency testing)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"duration_ms": map[string]interface{}{
						"type":        "number",
						"description": "Duration to delay in milliseconds",
					},
				},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}
	s.writeResponse(req.ID, result)
}

func (s *TestMCPServer) handleToolsCall(req *JSONRPCRequest) {
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
	case "failing_tool":
		s.handleFailingTool(req.ID, params.Arguments)
	case "delay_task":
		s.handleDelayTask(req.ID, params.Arguments)
	default:
		s.writeError(req.ID, -32602, "Unknown tool", params.Name)
	}
}

func (s *TestMCPServer) handleEcho(id interface{}, args map[string]interface{}) {
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

func (s *TestMCPServer) handleReadFile(id interface{}, args map[string]interface{}) {
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

func (s *TestMCPServer) handleWriteFile(id interface{}, args map[string]interface{}) {
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

func (s *TestMCPServer) handleFailingTool(id interface{}, args map[string]interface{}) {
	s.writeError(id, -32603, "Tool execution failed", "This tool always fails")
}

func (s *TestMCPServer) handleDelayTask(id interface{}, args map[string]interface{}) {
	// Handle optional delay for concurrency testing
	if durationMs, ok := args["duration_ms"].(float64); ok && durationMs > 0 {
		time.Sleep(time.Duration(durationMs) * time.Millisecond)
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Task completed",
			},
		},
	}
	s.writeResponse(id, result)
}

func (s *TestMCPServer) handlePing(req *JSONRPCRequest) {
	s.writeResponse(req.ID, map[string]interface{}{})
}

func (s *TestMCPServer) writeResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.write(resp)
}

func (s *TestMCPServer) writeError(id interface{}, code int, message string, data interface{}) {
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

func (s *TestMCPServer) write(resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		_, _ = fmt.Fprintf(s.stderr, "Error marshaling response: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(s.stdout, "%s\n", data)
}
