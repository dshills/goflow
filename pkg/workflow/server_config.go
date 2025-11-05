package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ServerConfig represents configuration for connecting to an MCP server
type ServerConfig struct {
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name,omitempty" yaml:"name,omitempty"`
	Command       string            `json:"command" yaml:"command"`
	Args          []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Transport     string            `json:"transport,omitempty" yaml:"transport,omitempty"` // "stdio" | "sse" | "http"
	Env           map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty" yaml:"credential_ref,omitempty"`
}

// validTransportTypes are the allowed transport types
var validTransportTypes = map[string]bool{
	"stdio": true,
	"sse":   true,
	"http":  true,
}

// Validate checks if the server config is valid
func (s *ServerConfig) Validate() error {
	if s.ID == "" {
		return errors.New("server config: empty server ID")
	}
	if s.Command == "" {
		return errors.New("server config: empty command")
	}

	// Validate transport type if provided
	if s.Transport != "" && !validTransportTypes[s.Transport] {
		return fmt.Errorf("server config: invalid transport type: %s (must be one of: stdio, sse, http)", s.Transport)
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
