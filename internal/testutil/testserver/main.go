package testserver

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/dshills/goflow/pkg/validation"
)

// Server is a minimal MCP test server for integration testing.
// It implements the MCP protocol and provides basic tools for testing.
type Server struct {
	config    *ServerConfig
	validator *validation.PathValidator
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
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

// NewServer creates a new test server with the given configuration.
//
// Returns error if:
//   - config is invalid (fails Validate())
//   - Cannot create path validator
//
// Example:
//
//	config := DefaultConfig()
//	config.AllowedDirectory = "/var/app/data"
//	server, err := NewServer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewServer(config *ServerConfig) (*Server, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create path validator for secure file operations
	validator, err := validation.NewPathValidator(config.AllowedDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to create path validator: %w", err)
	}

	return &Server{
		config:    config,
		validator: validator,
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
	}, nil
}

// Start starts the test server and begins processing MCP requests.
//
// This method logs the server configuration at startup and blocks
// until an error occurs or the input stream is closed.
func (s *Server) Start() error {
	// Log configuration at startup
	if s.config.LogSecurityEvents {
		log.Printf("Test server started: allowed_dir=%s max_size=%dMB security_log=%v",
			s.config.AllowedDirectory,
			s.config.MaxFileSize/(1024*1024),
			s.config.LogSecurityEvents)
	}

	return s.run()
}

func (s *Server) run() error {
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

func (s *Server) handleRequest(req *JSONRPCRequest) {
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

func (s *Server) handleInitialize(req *JSONRPCRequest) {
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

func (s *Server) handleToolsList(req *JSONRPCRequest) {
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

func (s *Server) handleToolsCall(req *JSONRPCRequest) {
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

func (s *Server) handleEcho(id interface{}, args map[string]interface{}) {
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

func (s *Server) handleReadFile(id interface{}, args map[string]interface{}) {
	path, ok := args["path"].(string)
	if !ok {
		s.writeError(id, -32602, "Invalid params", "path must be a string")
		return
	}

	// Validate path using PathValidator (SECURITY: prevent directory traversal)
	validPath, err := s.validator.Validate(path)
	if err != nil {
		s.logSecurityViolation("read", path, err)
		s.writeError(id, -32602, "Invalid file path", err.Error())
		return
	}

	content, err := os.ReadFile(validPath)
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

func (s *Server) handleWriteFile(id interface{}, args map[string]interface{}) {
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

	// Validate path using PathValidator (SECURITY: prevent directory traversal)
	validPath, err := s.validator.Validate(path)
	if err != nil {
		s.logSecurityViolation("write", path, err)
		s.writeError(id, -32602, "Invalid file path", err.Error())
		return
	}

	// Check file size limit (SECURITY: prevent resource exhaustion)
	if int64(len(content)) > s.config.MaxFileSize {
		err := fmt.Errorf("file size exceeds limit: %d > %d", len(content), s.config.MaxFileSize)
		s.logSecurityViolation("write", path, err)
		s.writeError(id, -32602, "File size exceeds limit", err.Error())
		return
	}

	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
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

func (s *Server) handleFailingTool(id interface{}, args map[string]interface{}) {
	s.writeError(id, -32603, "Tool execution failed", "This tool always fails")
}

func (s *Server) handleDelayTask(id interface{}, args map[string]interface{}) {
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

func (s *Server) handlePing(req *JSONRPCRequest) {
	s.writeResponse(req.ID, map[string]interface{}{})
}

func (s *Server) writeResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.write(resp)
}

func (s *Server) writeError(id interface{}, code int, message string, data interface{}) {
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

func (s *Server) write(resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		_, _ = fmt.Fprintf(s.stderr, "Error marshaling response: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(s.stdout, "%s\n", data)
}

// logSecurityViolation logs a security policy violation.
//
// Format: "SECURITY [testserver] Rejected {operation}: input={path} error={err}"
//
// This method implements the security logging requirement from the contract.
// Logs are written to stderr (or LogFilePath if configured).
func (s *Server) logSecurityViolation(operation, path string, err error) {
	if !s.config.LogSecurityEvents {
		return
	}

	// Log to stderr by default (or configured log file)
	logOutput := s.stderr
	// TODO(Phase 4): Implement log file output when config.LogFilePath is set
	// For now, always use stderr for simplicity
	_ = s.config.LogFilePath // Explicitly acknowledge unused config field

	logMsg := fmt.Sprintf("SECURITY [testserver] Rejected %s: input=%s error=%v\n",
		operation, path, err)
	_, _ = fmt.Fprint(logOutput, logMsg)
}

// Helper methods for testing

// SetStdin sets the stdin reader for testing.
func (s *Server) SetStdin(r io.Reader) {
	s.stdin = r
}

// SetStdout sets the stdout writer for testing.
func (s *Server) SetStdout(w io.Writer) {
	s.stdout = w
}

// SetStderr sets the stderr writer for testing.
func (s *Server) SetStderr(w io.Writer) {
	s.stderr = w
}

// ProcessSingleRequest processes a single JSON-RPC request for testing.
// This is a simplified version that reads one line and processes it.
func (s *Server) ProcessSingleRequest() error {
	scanner := bufio.NewScanner(s.stdin)
	if !scanner.Scan() {
		return scanner.Err()
	}

	line := scanner.Text()
	if line == "" {
		return nil
	}

	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		s.writeError(nil, -32700, "Parse error", err.Error())
		return err
	}

	s.handleRequest(&req)
	return nil
}
