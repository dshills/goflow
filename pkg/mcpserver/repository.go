package mcpserver

import (
	"fmt"
	"sync"
)

// ServerRepository defines the interface for managing MCP servers
type ServerRepository interface {
	Register(server *MCPServer) error
	Unregister(id string) error
	Get(id string) (*MCPServer, error)
	List() ([]*MCPServer, error)
}

// Registry is an in-memory implementation of ServerRepository
type Registry struct {
	servers map[string]*MCPServer
	mu      sync.RWMutex
}

// NewRegistry creates a new Registry
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]*MCPServer),
	}
}

// Register adds a new server to the registry
func (r *Registry) Register(server *MCPServer) error {
	if server == nil {
		return NewValidationError("cannot register nil server")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate server ID
	if _, exists := r.servers[server.ID]; exists {
		return NewValidationError(fmt.Sprintf("duplicate server ID: %s", server.ID))
	}

	r.servers[server.ID] = server
	return nil
}

// Unregister removes a server from the registry
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[id]; !exists {
		return NewValidationError(fmt.Sprintf("server not found: %s", id))
	}

	delete(r.servers, id)
	return nil
}

// Get retrieves a server by ID
func (r *Registry) Get(id string) (*MCPServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	server, exists := r.servers[id]
	if !exists {
		return nil, NewValidationError(fmt.Sprintf("server not found: %s", id))
	}

	return server, nil
}

// List returns all registered servers
func (r *Registry) List() ([]*MCPServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]*MCPServer, 0, len(r.servers))
	for _, server := range r.servers {
		servers = append(servers, server)
	}

	return servers, nil
}
