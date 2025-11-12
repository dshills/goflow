package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ServerConfig represents configuration for connecting to an MCP server
type ServerConfig struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name,omitempty" yaml:"name,omitempty"`
	Command       string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args          []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Transport     string            `json:"transport,omitempty" yaml:"transport,omitempty"` // "stdio" | "sse" | "http"
	Env           map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`

	// Transport-specific configuration
	URL     string            `json:"url,omitempty" yaml:"url,omitempty"`         // For SSE and HTTP transports
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"` // For SSE and HTTP transports
}

// validTransportTypes are the allowed transport types
var validTransportTypes = map[string]bool{
	"stdio": true,
	"sse":   true,
	"http":  true,
}

// GetTransport returns the transport type, defaulting to "stdio" for backward compatibility
func (s *ServerConfig) GetTransport() string {
	if s.Transport == "" {
		return "stdio"
	}
	return s.Transport
}

// Validate checks if the server config is valid
func (s *ServerConfig) Validate() error {
	if s.ID == "" {
		return errors.New("server config: empty server ID")
	}

	// Get transport type (defaults to stdio)
	transport := s.GetTransport()

	// Validate transport type
	if !validTransportTypes[transport] {
		return fmt.Errorf("server config: invalid transport type: %s (must be one of: stdio, sse, http)", transport)
	}

	// Validate transport-specific configuration
	switch transport {
	case "stdio":
		if s.Command == "" {
			return errors.New("server config: command is required for stdio transport")
		}
		if s.URL != "" {
			return errors.New("server config: URL should not be specified for stdio transport")
		}

	case "sse":
		if s.URL == "" {
			return errors.New("server config: URL is required for sse transport")
		}
		if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
			return errors.New("server config: URL must start with http:// or https:// for sse transport")
		}
		if s.Command != "" {
			return errors.New("server config: command should not be specified for sse transport")
		}
		if len(s.Args) > 0 {
			return errors.New("server config: args should not be specified for sse transport")
		}

	case "http":
		if s.URL == "" {
			return errors.New("server config: URL is required for http transport")
		}
		if !strings.HasPrefix(s.URL, "http://") && !strings.HasPrefix(s.URL, "https://") {
			return errors.New("server config: URL must start with http:// or https:// for http transport")
		}
		if s.Command != "" {
			return errors.New("server config: command should not be specified for http transport")
		}
		if len(s.Args) > 0 {
			return errors.New("server config: args should not be specified for http transport")
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for ServerConfig
func (s *ServerConfig) MarshalJSON() ([]byte, error) {
	type Alias ServerConfig
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for ServerConfig
func (s *ServerConfig) UnmarshalJSON(data []byte) error {
	type Alias ServerConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
