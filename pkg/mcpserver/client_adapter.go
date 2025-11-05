package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// clientAdapter adapts a generic MCP protocol client to the MCPClient interface
// This allows the pkg/mcp.StdioClient (and future HTTP/SSE clients) to be used
// with the MCPServer entity without creating a circular dependency.
//
// The adapter handles the differences in type signatures between the protocol
// client interface and the domain model's MCPClient interface.
type clientAdapter struct {
	// Underlying protocol client - stored as interface{} to avoid import cycles
	protocolClient interface{}
}

// NewClientAdapter creates a new adapter that wraps a protocol client
// The protocolClient should implement Connect, Close, IsConnected, ListTools,
// CallTool, and Ping methods. This is typically a *mcp.StdioClient.
//
// Example usage:
//
//	import "github.com/dshills/goflow/pkg/mcp"
//
//	config := mcp.ServerConfig{
//	    ID:      "my-server",
//	    Command: "python",
//	    Args:    []string{"-m", "my_mcp_server"},
//	}
//	stdioClient, err := mcp.NewStdioClient(config)
//	if err != nil {
//	    return nil, err
//	}
//
//	// Wrap the stdio client in an adapter
//	mcpClient := mcpserver.NewClientAdapter(stdioClient)
//
//	// Create server with adapted client
//	server, err := mcpserver.NewMCPServerWithClient("my-server", "python",
//	    []string{"-m", "my_mcp_server"}, mcpserver.TransportStdio, mcpClient)
func NewClientAdapter(protocolClient interface{}) MCPClient {
	return &clientAdapter{
		protocolClient: protocolClient,
	}
}

// Connect establishes a connection to the MCP server
func (a *clientAdapter) Connect(ctx context.Context) error {
	type connector interface {
		Connect(ctx context.Context) error
	}
	if c, ok := a.protocolClient.(connector); ok {
		return c.Connect(ctx)
	}
	return fmt.Errorf("protocol client does not support Connect")
}

// Close terminates the connection to the MCP server
func (a *clientAdapter) Close() error {
	type closer interface {
		Close() error
	}
	if c, ok := a.protocolClient.(closer); ok {
		return c.Close()
	}
	return fmt.Errorf("protocol client does not support Close")
}

// IsConnected returns true if the client is connected
func (a *clientAdapter) IsConnected() bool {
	type connectionChecker interface {
		IsConnected() bool
	}
	if c, ok := a.protocolClient.(connectionChecker); ok {
		return c.IsConnected()
	}
	return false
}

// ListTools retrieves all available tools from the server
// This method adapts between different tool representations:
// - The protocol client returns raw tool data (typically map[string]interface{})
// - The domain model expects []Tool with strongly-typed fields
func (a *clientAdapter) ListTools(ctx context.Context) ([]Tool, error) {
	// Try to call ListTools that returns []Tool directly
	type directToolLister interface {
		ListTools(ctx context.Context) ([]Tool, error)
	}
	if c, ok := a.protocolClient.(directToolLister); ok {
		return c.ListTools(ctx)
	}

	// Try to call ListTools that returns []interface{} (generic)
	type genericToolLister interface {
		ListTools(ctx context.Context) ([]interface{}, error)
	}
	if c, ok := a.protocolClient.(genericToolLister); ok {
		rawTools, err := c.ListTools(ctx)
		if err != nil {
			return nil, err
		}

		// Convert []interface{} to []Tool
		tools := make([]Tool, len(rawTools))
		for i, raw := range rawTools {
			// Marshal and unmarshal to convert types
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool %d: %w", i, err)
			}

			var tool Tool
			if err := json.Unmarshal(data, &tool); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool %d: %w", i, err)
			}

			tools[i] = tool
		}

		return tools, nil
	}

	return nil, fmt.Errorf("protocol client does not support ListTools")
}

// CallTool invokes a tool on the server with the given parameters
func (a *clientAdapter) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	type toolCaller interface {
		CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error)
	}
	if c, ok := a.protocolClient.(toolCaller); ok {
		return c.CallTool(ctx, toolName, params)
	}
	return nil, fmt.Errorf("protocol client does not support CallTool")
}

// Ping sends a ping request to verify server health
func (a *clientAdapter) Ping(ctx context.Context) error {
	type pinger interface {
		Ping(ctx context.Context) error
	}
	if c, ok := a.protocolClient.(pinger); ok {
		return c.Ping(ctx)
	}
	return fmt.Errorf("protocol client does not support Ping")
}
