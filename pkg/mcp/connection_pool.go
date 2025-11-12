package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// createClient creates a new MCP client based on the transport type
func createClient(config ServerConfig) (Client, error) {
	// Default to stdio if transport not specified
	transport := config.Transport
	if transport == "" {
		transport = "stdio"
	}

	switch transport {
	case "stdio":
		return NewStdioClient(config)

	case "sse":
		return NewSSEClient(SSEConfig{
			URL:     config.URL,
			Headers: config.Headers,
		})

	case "http":
		return NewHTTPClient(HTTPConfig{
			BaseURL: config.URL,
			Headers: config.Headers,
		})

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transport)
	}
}

const (
	// MaxConnectionsPerServer limits connections per server
	MaxConnectionsPerServer = 10
	// ConnectionIdleTimeout is how long before idle connections are closed
	ConnectionIdleTimeout = 5 * time.Minute
)

// PooledConnection wraps a client with connection metadata
type PooledConnection struct {
	Client   Client
	ServerID string
	LastUsed time.Time
	InUse    bool
	mu       sync.Mutex
}

// ConnectionPool manages a pool of MCP client connections
type ConnectionPool struct {
	connections  map[string][]*PooledConnection // serverID -> connections
	configs      map[string]ServerConfig        // serverID -> config
	mu           sync.RWMutex
	maxPerServer int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections:  make(map[string][]*PooledConnection),
		configs:      make(map[string]ServerConfig),
		maxPerServer: MaxConnectionsPerServer,
	}

	// Start background cleanup goroutine
	go pool.cleanupIdleConnections()

	return pool
}

// RegisterServer registers a server configuration with the pool
func (p *ConnectionPool) RegisterServer(config ServerConfig) error {
	if config.ID == "" {
		return fmt.Errorf("server ID cannot be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.configs[config.ID] = config
	if _, exists := p.connections[config.ID]; !exists {
		p.connections[config.ID] = make([]*PooledConnection, 0)
	}

	return nil
}

// Get retrieves or creates a connection for the given server
func (p *ConnectionPool) Get(ctx context.Context, serverID string) (Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if server is registered
	config, exists := p.configs[serverID]
	if !exists {
		return nil, fmt.Errorf("server %s not registered", serverID)
	}

	// Try to find an available idle connection
	connections := p.connections[serverID]
	for _, conn := range connections {
		conn.mu.Lock()
		if !conn.InUse && conn.Client.IsConnected() {
			conn.InUse = true
			conn.LastUsed = time.Now()
			conn.mu.Unlock()
			return conn.Client, nil
		}
		conn.mu.Unlock()
	}

	// Check if we can create a new connection
	if len(connections) >= p.maxPerServer {
		return nil, fmt.Errorf("connection pool exhausted for server %s (max: %d)", serverID, p.maxPerServer)
	}

	// Create new connection based on transport type
	client, err := createClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Add to pool
	pooledConn := &PooledConnection{
		Client:   client,
		ServerID: serverID,
		LastUsed: time.Now(),
		InUse:    true,
	}

	p.connections[serverID] = append(p.connections[serverID], pooledConn)

	return client, nil
}

// Release marks a connection as no longer in use
func (p *ConnectionPool) Release(serverID string, client Client) error {
	p.mu.RLock()
	connections, exists := p.connections[serverID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	for _, conn := range connections {
		if conn.Client == client {
			conn.mu.Lock()
			conn.InUse = false
			conn.LastUsed = time.Now()
			conn.mu.Unlock()
			return nil
		}
	}

	return fmt.Errorf("connection not found in pool for server %s", serverID)
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for serverID, connections := range p.connections {
		for _, conn := range connections {
			if err := conn.Client.Close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to close connection for server %s: %w", serverID, err)
			}
		}
		p.connections[serverID] = nil
	}

	p.connections = make(map[string][]*PooledConnection)
	return firstErr
}

// CloseServer closes all connections for a specific server
func (p *ConnectionPool) CloseServer(serverID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	connections, exists := p.connections[serverID]
	if !exists {
		return nil // No connections to close
	}

	var firstErr error
	for _, conn := range connections {
		if err := conn.Client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close connection: %w", err)
		}
	}

	p.connections[serverID] = make([]*PooledConnection, 0)
	return firstErr
}

// Stats returns statistics about the connection pool
func (p *ConnectionPool) Stats() map[string]PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for serverID, connections := range p.connections {
		var active, idle int
		for _, conn := range connections {
			conn.mu.Lock()
			if conn.InUse {
				active++
			} else {
				idle++
			}
			conn.mu.Unlock()
		}

		stats[serverID] = PoolStats{
			Total:  len(connections),
			Active: active,
			Idle:   idle,
		}
	}

	return stats
}

// PoolStats holds statistics for a server's connection pool
type PoolStats struct {
	Total  int
	Active int
	Idle   int
}

// cleanupIdleConnections periodically closes idle connections
func (p *ConnectionPool) cleanupIdleConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()

		for serverID, connections := range p.connections {
			newConnections := make([]*PooledConnection, 0, len(connections))

			for _, conn := range connections {
				conn.mu.Lock()
				shouldClose := !conn.InUse && time.Since(conn.LastUsed) > ConnectionIdleTimeout
				conn.mu.Unlock()

				if shouldClose {
					// Close idle connection
					_ = conn.Client.Close()
				} else {
					// Keep connection
					newConnections = append(newConnections, conn)
				}
			}

			p.connections[serverID] = newConnections
		}

		p.mu.Unlock()
	}
}

// Reconnect attempts to reconnect a failed connection
func (p *ConnectionPool) Reconnect(ctx context.Context, serverID string, client Client) error {
	p.mu.RLock()
	config, exists := p.configs[serverID]
	connections := p.connections[serverID]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not registered", serverID)
	}

	// Find the connection in the pool
	for _, conn := range connections {
		if conn.Client == client {
			conn.mu.Lock()

			// Close old connection
			_ = conn.Client.Close()

			// Create new connection based on transport type
			newClient, err := createClient(config)
			if err != nil {
				conn.mu.Unlock()
				return fmt.Errorf("failed to create new client: %w", err)
			}

			// Connect
			if err := newClient.Connect(ctx); err != nil {
				conn.mu.Unlock()
				return fmt.Errorf("failed to reconnect: %w", err)
			}

			// Replace client
			conn.Client = newClient
			conn.LastUsed = time.Now()

			conn.mu.Unlock()
			return nil
		}
	}

	return fmt.Errorf("connection not found in pool for server %s", serverID)
}
