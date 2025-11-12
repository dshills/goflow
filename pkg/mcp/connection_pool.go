package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
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
	Client     Client
	ServerID   string
	LastUsed   time.Time
	AcquiredAt time.Time
	InUse      bool
	mu         sync.Mutex

	// Lifecycle tracking
	closed   bool  // Whether connection is closed
	closeErr error // Error from close operation
	refCount int32 // Reference count for leak detection
}

// IsClosed returns whether the connection has been closed.
func (c *PooledConnection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// ConnectionPool manages a pool of MCP client connections
type ConnectionPool struct {
	connections  map[string][]*PooledConnection // serverID -> connections
	configs      map[string]ServerConfig        // serverID -> config
	mu           sync.RWMutex
	maxPerServer int

	// Shutdown coordination
	closing chan struct{}  // Signals shutdown in progress
	wg      sync.WaitGroup // Tracks active operations
	closed  bool           // Whether pool is closed

	// Leak detection metrics
	leaksDetected atomic.Uint64 // Count of connection leaks detected
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	return newConnectionPoolWithInterval(1 * time.Minute)
}

// newConnectionPoolWithInterval creates a connection pool with custom cleanup interval (for testing)
func newConnectionPoolWithInterval(cleanupInterval time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		connections:  make(map[string][]*PooledConnection),
		configs:      make(map[string]ServerConfig),
		maxPerServer: MaxConnectionsPerServer,
		closing:      make(chan struct{}),
		closed:       false,
	}

	// Start background cleanup goroutine
	go pool.cleanupIdleConnectionsWithInterval(cleanupInterval)

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

	// Check if pool is closed
	if p.closed {
		return nil, fmt.Errorf("connection pool is closed")
	}

	// Track active operation
	p.wg.Add(1)

	// Check if server is registered
	config, exists := p.configs[serverID]
	if !exists {
		p.wg.Done()
		return nil, fmt.Errorf("server %s not registered", serverID)
	}

	// Try to find an available idle connection
	connections := p.connections[serverID]
	for _, conn := range connections {
		conn.mu.Lock()
		if !conn.InUse && !conn.closed && conn.Client.IsConnected() {
			conn.InUse = true
			conn.LastUsed = time.Now()
			conn.AcquiredAt = time.Now()
			conn.refCount++
			conn.mu.Unlock()
			return conn.Client, nil
		}
		conn.mu.Unlock()
	}

	// Check if we can create a new connection
	if len(connections) >= p.maxPerServer {
		p.wg.Done()
		return nil, fmt.Errorf("connection pool exhausted for server %s (max: %d)", serverID, p.maxPerServer)
	}

	// Create new connection based on transport type
	client, err := createClient(config)
	if err != nil {
		p.wg.Done()
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		p.wg.Done()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Add to pool
	pooledConn := &PooledConnection{
		Client:     client,
		ServerID:   serverID,
		LastUsed:   time.Now(),
		AcquiredAt: time.Now(),
		InUse:      true,
		refCount:   1,
	}

	p.connections[serverID] = append(p.connections[serverID], pooledConn)

	return client, nil
}

// Release marks a connection as no longer in use
func (p *ConnectionPool) Release(serverID string) error {
	p.mu.RLock()
	connections, exists := p.connections[serverID]
	p.mu.RUnlock()

	if !exists {
		p.wg.Done()
		return fmt.Errorf("server %s not found in pool", serverID)
	}

	// Find the in-use connection for this server
	var found bool
	for _, conn := range connections {
		conn.mu.Lock()
		if conn.InUse {
			conn.InUse = false
			conn.LastUsed = time.Now()
			conn.refCount--
			conn.mu.Unlock()
			found = true
			break
		}
		conn.mu.Unlock()
	}

	// Mark operation complete
	p.wg.Done()

	if !found {
		return fmt.Errorf("no active connection found for server %s", serverID)
	}

	return nil
}

// Close closes all connections in the pool with graceful shutdown
func (p *ConnectionPool) Close() error {
	p.mu.Lock()

	// Check if already closed
	if p.closed {
		p.mu.Unlock()
		return nil
	}

	// Mark as closing
	p.closed = true
	close(p.closing)
	p.mu.Unlock()

	// Wait for active operations with 30s grace period
	gracePeriodDone := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(gracePeriodDone)
	}()

	var forceCloseRequired bool
	select {
	case <-gracePeriodDone:
		// Graceful shutdown completed
	case <-time.After(30 * time.Second):
		// Grace period expired, force-close
		forceCloseRequired = true
	}

	// Close all connections
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for serverID, connections := range p.connections {
		for _, conn := range connections {
			conn.mu.Lock()
			conn.closed = true
			if err := conn.Client.Close(); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to close connection for server %s: %w", serverID, err)
				conn.closeErr = err
			}
			conn.mu.Unlock()
		}
		p.connections[serverID] = nil
	}

	p.connections = make(map[string][]*PooledConnection)

	if forceCloseRequired {
		if firstErr != nil {
			return fmt.Errorf("force-close required after grace period: %w", firstErr)
		}
		return fmt.Errorf("force-close required after grace period")
	}

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
		conn.mu.Lock()
		conn.closed = true
		if err := conn.Client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close connection: %w", err)
			conn.closeErr = err
		}
		conn.mu.Unlock()
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

// LeakStats returns leak detection metrics
func (p *ConnectionPool) LeakStats() uint64 {
	return p.leaksDetected.Load()
}

// cleanupIdleConnections periodically closes idle connections
func (p *ConnectionPool) cleanupIdleConnections() {
	p.cleanupIdleConnectionsWithInterval(1 * time.Minute)
}

// cleanupIdleConnectionsWithInterval periodically closes idle connections at the specified interval
func (p *ConnectionPool) cleanupIdleConnectionsWithInterval(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.closing:
			// Pool is closing, stop cleanup
			return
		case <-ticker.C:
			p.mu.Lock()

			for serverID, connections := range p.connections {
				newConnections := make([]*PooledConnection, 0, len(connections))

				for _, conn := range connections {
					conn.mu.Lock()
					shouldClose := !conn.InUse && time.Since(conn.LastUsed) > ConnectionIdleTimeout
					if shouldClose {
						// Check for connection leaks before closing
						if conn.refCount != 0 {
							log.Printf("WARNING: Connection leak detected for server %s: refCount=%d (expected 0)", serverID, conn.refCount)
							p.leaksDetected.Add(1)
						}

						conn.closed = true
						conn.closeErr = conn.Client.Close()
						conn.mu.Unlock()
					} else {
						conn.mu.Unlock()
						// Keep connection
						newConnections = append(newConnections, conn)
					}
				}

				p.connections[serverID] = newConnections
			}

			p.mu.Unlock()
		}
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
