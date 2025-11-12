package mcp

import (
	"context"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// Client represents a client that can communicate with an MCP server
type Client interface {
	// Connect establishes a connection to the MCP server
	Connect(ctx context.Context) error

	// Close terminates the connection to the MCP server
	Close() error

	// IsConnected returns true if the client is connected
	IsConnected() bool

	// ListTools retrieves all available tools from the server
	ListTools(ctx context.Context) ([]mcpserver.Tool, error)

	// CallTool invokes a tool on the server with the given parameters
	CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)

	// Ping sends a ping request to verify server health
	Ping(ctx context.Context) error
}

// ServerConfig holds configuration for connecting to an MCP server
type ServerConfig struct {
	ID        string
	Command   string
	Args      []string
	Env       map[string]string
	Transport string            // "stdio", "sse", or "http"
	URL       string            // For SSE and HTTP transports
	Headers   map[string]string // For SSE and HTTP transports
}
