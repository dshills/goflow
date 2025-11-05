package mcpserver

import "fmt"

// ServerID uniquely identifies an MCP server
type ServerID string

// String returns the string representation of ServerID
func (s ServerID) String() string {
	return string(s)
}

// ToolID uniquely identifies a tool within a server
type ToolID string

// String returns the string representation of ToolID
func (t ToolID) String() string {
	return string(t)
}

// ConnectionState represents the current connection state of an MCP server
type ConnectionState string

const (
	// StateDisconnected indicates the server is not connected
	StateDisconnected ConnectionState = "disconnected"
	// StateConnecting indicates a connection is being established
	StateConnecting ConnectionState = "connecting"
	// StateConnected indicates the server is connected and ready
	StateConnected ConnectionState = "connected"
	// StateFailed indicates the connection attempt failed
	StateFailed ConnectionState = "failed"
)

// String returns the string representation of a ConnectionState
func (cs ConnectionState) String() string {
	return string(cs)
}

// IsValid checks if the ConnectionState is valid
func (cs ConnectionState) IsValid() bool {
	switch cs {
	case StateDisconnected, StateConnecting, StateConnected, StateFailed:
		return true
	default:
		return false
	}
}

// HealthStatus represents the health state of an MCP server
type HealthStatus string

const (
	// HealthUnknown indicates health status has not been checked
	HealthUnknown HealthStatus = "unknown"
	// HealthHealthy indicates the server is responding normally
	HealthHealthy HealthStatus = "healthy"
	// HealthUnhealthy indicates the server is experiencing errors
	HealthUnhealthy HealthStatus = "unhealthy"
	// HealthDisconnected indicates the connection is lost
	HealthDisconnected HealthStatus = "disconnected"
)

// String returns the string representation of a HealthStatus
func (hs HealthStatus) String() string {
	return string(hs)
}

// IsValid checks if the HealthStatus is valid
func (hs HealthStatus) IsValid() bool {
	switch hs {
	case HealthUnknown, HealthHealthy, HealthUnhealthy, HealthDisconnected:
		return true
	default:
		return false
	}
}

// TransportType represents the communication transport type for an MCP server
type TransportType string

const (
	// TransportStdio uses standard input/output for communication
	TransportStdio TransportType = "stdio"
	// TransportSSE uses Server-Sent Events for communication
	TransportSSE TransportType = "sse"
	// TransportHTTP uses HTTP JSON-RPC for communication
	TransportHTTP TransportType = "http"
)

// String returns the string representation of a TransportType
func (tt TransportType) String() string {
	return string(tt)
}

// IsValid checks if the TransportType is valid
func (tt TransportType) IsValid() bool {
	switch tt {
	case TransportStdio, TransportSSE, TransportHTTP:
		return true
	default:
		return false
	}
}

// ErrorType categorizes different types of errors
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeConnection indicates a connection error
	ErrorTypeConnection ErrorType = "connection"
	// ErrorTypeExecution indicates an execution error
	ErrorTypeExecution ErrorType = "execution"
	// ErrorTypeData indicates a data transformation error
	ErrorTypeData ErrorType = "data"
	// ErrorTypeTimeout indicates a timeout error
	ErrorTypeTimeout ErrorType = "timeout"
)

// MCPError represents a structured error with context
type MCPError struct {
	Type    ErrorType
	Message string
	Context map[string]interface{}
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("%s: %s (context: %v)", e.Type, e.Message, e.Context)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return &MCPError{
		Type:    ErrorTypeValidation,
		Message: message,
	}
}

// NewConnectionError creates a new connection error
func NewConnectionError(message string) error {
	return &MCPError{
		Type:    ErrorTypeConnection,
		Message: message,
	}
}

// NewExecutionError creates a new execution error
func NewExecutionError(message string) error {
	return &MCPError{
		Type:    ErrorTypeExecution,
		Message: message,
	}
}
