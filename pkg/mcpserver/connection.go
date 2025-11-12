package mcpserver

import (
	"fmt"
	"sync"
	"time"
)

// Connection represents the connection state to an MCP server
type Connection struct {
	mu           sync.RWMutex // Protects concurrent access to connection state
	State        ConnectionState
	ConnectedAt  time.Time
	LastActivity time.Time
	ErrorCount   int
	RetryBackoff time.Duration
	LastError    string
}

// NewConnection creates a new Connection in disconnected state
func NewConnection() *Connection {
	return &Connection{
		State:        StateDisconnected,
		ErrorCount:   0,
		RetryBackoff: 0,
	}
}

// UpdateLastActivity sets the LastActivity timestamp to now (thread-safe)
func (c *Connection) UpdateLastActivity() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastActivity = time.Now()
}

// GetLastActivity returns the LastActivity timestamp (thread-safe)
func (c *Connection) GetLastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastActivity
}

// GetState returns the current connection state (thread-safe)
func (c *Connection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State
}

// SetState updates the connection state (thread-safe)
func (c *Connection) SetState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = state
}

// IncrementErrorCount increments the error counter (thread-safe)
func (c *Connection) IncrementErrorCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ErrorCount++
}

// ResetErrorCount resets the error counter to zero (thread-safe)
func (c *Connection) ResetErrorCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ErrorCount = 0
}

// GetErrorCount returns the current error count (thread-safe)
func (c *Connection) GetErrorCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ErrorCount
}

// SetLastError records the last error message (thread-safe)
func (c *Connection) SetLastError(err string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastError = err
}

// GetLastError returns the last error message (thread-safe)
func (c *Connection) GetLastError() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastError
}

// Transport represents the communication transport configuration for an MCP server
type Transport interface {
	Type() TransportType
	Validate() error
}

// StdioTransportConfig configures stdio transport
type StdioTransportConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// Type returns the transport type
func (c *StdioTransportConfig) Type() TransportType {
	return TransportStdio
}

// Validate checks if the configuration is valid
func (c *StdioTransportConfig) Validate() error {
	if c.Command == "" {
		return NewValidationError("stdio transport: command cannot be empty")
	}
	return nil
}

// SSETransportConfig configures Server-Sent Events transport
type SSETransportConfig struct {
	URL     string
	Headers map[string]string
}

// Type returns the transport type
func (c *SSETransportConfig) Type() TransportType {
	return TransportSSE
}

// Validate checks if the configuration is valid
func (c *SSETransportConfig) Validate() error {
	if c.URL == "" {
		return NewValidationError("sse transport: URL cannot be empty")
	}
	return nil
}

// HTTPTransportConfig configures HTTP JSON-RPC transport
type HTTPTransportConfig struct {
	BaseURL string
	Headers map[string]string
	Timeout time.Duration
}

// Type returns the transport type
func (c *HTTPTransportConfig) Type() TransportType {
	return TransportHTTP
}

// Validate checks if the configuration is valid
func (c *HTTPTransportConfig) Validate() error {
	if c.BaseURL == "" {
		return NewValidationError("http transport: BaseURL cannot be empty")
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second // default timeout
	}
	return nil
}

// NewTransport creates a new Transport based on the type and configuration
func NewTransport(transportType TransportType, config interface{}) (Transport, error) {
	if !transportType.IsValid() {
		return nil, NewValidationError(fmt.Sprintf("invalid transport type: %s", transportType))
	}

	switch transportType {
	case TransportStdio:
		cfg, ok := config.(*StdioTransportConfig)
		if !ok {
			return nil, NewValidationError("invalid config type for stdio transport")
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil

	case TransportSSE:
		cfg, ok := config.(*SSETransportConfig)
		if !ok {
			return nil, NewValidationError("invalid config type for sse transport")
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil

	case TransportHTTP:
		cfg, ok := config.(*HTTPTransportConfig)
		if !ok {
			return nil, NewValidationError("invalid config type for http transport")
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil

	default:
		return nil, NewValidationError(fmt.Sprintf("unsupported transport type: %s", transportType))
	}
}
