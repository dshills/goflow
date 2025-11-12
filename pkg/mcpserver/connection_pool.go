package mcpserver

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PooledConnection represents a server connection with usage tracking
type PooledConnection struct {
	Server     *MCPServer
	UsageCount int64
	LastUsed   time.Time
	CreatedAt  time.Time
	PreWarmed  bool
	KeepAlive  bool
}

// PoolStats tracks connection pool performance metrics
type PoolStats struct {
	TotalConnections     int64
	ActiveConnections    int64
	PreWarmedConnections int64
	UsageHits            int64 // Connections reused from pool
	UsageMisses          int64 // New connections created
	ReuseRate            float64
	LastUpdated          time.Time
}

// ConnectionPool manages a pool of MCP server connections with pre-warming
// and usage tracking to optimize connection reuse
type ConnectionPool struct {
	connections map[string]*PooledConnection // key: server ID
	stats       *PoolStats
	mu          sync.RWMutex

	// Pre-warming configuration
	preWarmThreshold int           // Usage count to trigger pre-warming
	preWarmServers   []string      // Server IDs to pre-warm
	keepAliveTime    time.Duration // How long to keep connections alive

	// Background workers
	stopChan chan struct{}
	doneChan chan struct{}
}

// NewConnectionPool creates a new connection pool with default settings
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		stats: &PoolStats{
			LastUpdated: time.Now(),
		},
		preWarmThreshold: 3,               // Pre-warm after 3 uses
		keepAliveTime:    5 * time.Minute, // Keep connections alive for 5 minutes
		preWarmServers:   []string{},
		stopChan:         make(chan struct{}),
		doneChan:         make(chan struct{}),
	}

	// Start background cleanup worker
	go pool.cleanupWorker()

	return pool
}

// NewConnectionPoolWithConfig creates a pool with custom configuration
func NewConnectionPoolWithConfig(preWarmThreshold int, keepAliveTime time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		stats: &PoolStats{
			LastUpdated: time.Now(),
		},
		preWarmThreshold: preWarmThreshold,
		keepAliveTime:    keepAliveTime,
		preWarmServers:   []string{},
		stopChan:         make(chan struct{}),
		doneChan:         make(chan struct{}),
	}

	go pool.cleanupWorker()

	return pool
}

// Get retrieves or creates a connection to the specified server
func (p *ConnectionPool) Get(ctx context.Context, server *MCPServer) (*PooledConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if connection exists and is valid
	if conn, exists := p.connections[server.ID]; exists {
		// Verify connection is still active
		if server.Connection.State == StateConnected {
			// Update usage statistics
			conn.UsageCount++
			conn.LastUsed = time.Now()
			p.stats.UsageHits++
			p.stats.LastUpdated = time.Now()

			// Check if server should be pre-warmed
			if !conn.PreWarmed && conn.UsageCount >= int64(p.preWarmThreshold) {
				p.markForPreWarming(server.ID)
			}

			return conn, nil
		}

		// Connection exists but is not active, remove it
		delete(p.connections, server.ID)
		p.stats.ActiveConnections--
	}

	// Create new connection
	p.stats.UsageMisses++
	p.stats.LastUpdated = time.Now()

	conn := &PooledConnection{
		Server:     server,
		UsageCount: 1,
		LastUsed:   time.Now(),
		CreatedAt:  time.Now(),
		PreWarmed:  false,
		KeepAlive:  false,
	}

	p.connections[server.ID] = conn
	p.stats.TotalConnections++
	p.stats.ActiveConnections++

	return conn, nil
}

// Release marks a connection as available for reuse
func (p *ConnectionPool) Release(serverID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[serverID]; exists {
		conn.LastUsed = time.Now()

		// If this is a pre-warmed connection, keep it alive
		if conn.PreWarmed {
			conn.KeepAlive = true
		}
	}
}

// Remove removes a connection from the pool
func (p *ConnectionPool) Remove(serverID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.connections[serverID]; exists {
		delete(p.connections, serverID)
		p.stats.ActiveConnections--
		p.stats.LastUpdated = time.Now()
	}
}

// PreWarm pre-establishes connections to frequently used servers
func (p *ConnectionPool) PreWarm(ctx context.Context, registry *Registry, serverIDs []string) error {
	p.mu.Lock()
	p.preWarmServers = append(p.preWarmServers, serverIDs...)
	p.mu.Unlock()

	var errors []error

	for _, serverID := range serverIDs {
		// Get server from registry
		server, err := registry.Get(serverID)
		if err != nil {
			errors = append(errors, fmt.Errorf("server %s not found: %w", serverID, err))
			continue
		}

		// Check if already connected
		p.mu.RLock()
		conn, exists := p.connections[serverID]
		p.mu.RUnlock()

		if exists && server.Connection.State == StateConnected {
			// Already connected, just mark as pre-warmed
			p.mu.Lock()
			conn.PreWarmed = true
			conn.KeepAlive = true
			p.stats.PreWarmedConnections++
			p.mu.Unlock()
			continue
		}

		// Connect to server
		if err := server.Connect(); err != nil {
			errors = append(errors, fmt.Errorf("failed to connect to %s: %w", serverID, err))
			continue
		}

		if err := server.CompleteConnection(); err != nil {
			errors = append(errors, fmt.Errorf("failed to complete connection to %s: %w", serverID, err))
			continue
		}

		// Discover tools
		if err := server.DiscoverTools(); err != nil {
			// Tools discovery failure is not critical, log but continue
			errors = append(errors, fmt.Errorf("failed to discover tools on %s: %w", serverID, err))
		}

		// Add to pool
		p.mu.Lock()
		newConn := &PooledConnection{
			Server:     server,
			UsageCount: 0,
			LastUsed:   time.Now(),
			CreatedAt:  time.Now(),
			PreWarmed:  true,
			KeepAlive:  true,
		}
		p.connections[serverID] = newConn
		p.stats.TotalConnections++
		p.stats.ActiveConnections++
		p.stats.PreWarmedConnections++
		p.stats.LastUpdated = time.Now()
		p.mu.Unlock()
	}

	if len(errors) > 0 {
		return fmt.Errorf("pre-warming completed with errors: %v", errors)
	}

	return nil
}

// markForPreWarming marks a server for pre-warming
func (p *ConnectionPool) markForPreWarming(serverID string) {
	// Add to pre-warm list if not already there
	alreadyMarked := false
	for _, id := range p.preWarmServers {
		if id == serverID {
			alreadyMarked = true
			break
		}
	}

	if !alreadyMarked {
		p.preWarmServers = append(p.preWarmServers, serverID)
	}

	// Mark connection as pre-warmed
	if conn, exists := p.connections[serverID]; exists {
		conn.PreWarmed = true
		conn.KeepAlive = true
		p.stats.PreWarmedConnections++
	}
}

// GetFrequentServers returns server IDs sorted by usage frequency
func (p *ConnectionPool) GetFrequentServers(limit int) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create slice of connections for sorting
	type serverUsage struct {
		serverID   string
		usageCount int64
	}

	usages := make([]serverUsage, 0, len(p.connections))
	for serverID, conn := range p.connections {
		usages = append(usages, serverUsage{
			serverID:   serverID,
			usageCount: conn.UsageCount,
		})
	}

	// Simple bubble sort by usage count (descending)
	for i := 0; i < len(usages); i++ {
		for j := i + 1; j < len(usages); j++ {
			if usages[j].usageCount > usages[i].usageCount {
				usages[i], usages[j] = usages[j], usages[i]
			}
		}
	}

	// Return top N server IDs
	result := make([]string, 0, limit)
	for i := 0; i < len(usages) && i < limit; i++ {
		result = append(result, usages[i].serverID)
	}

	return result
}

// GetStats returns current pool statistics
func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := *p.stats
	stats.ActiveConnections = int64(len(p.connections))

	// Calculate reuse rate
	total := stats.UsageHits + stats.UsageMisses
	if total > 0 {
		stats.ReuseRate = float64(stats.UsageHits) / float64(total)
	}

	return stats
}

// cleanupWorker runs periodically to clean up idle connections
func (p *ConnectionPool) cleanupWorker() {
	defer close(p.doneChan)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.cleanupIdle()
		}
	}
}

// cleanupIdle removes connections that have been idle for too long
func (p *ConnectionPool) cleanupIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for serverID, conn := range p.connections {
		// Skip keep-alive connections (pre-warmed)
		if conn.KeepAlive {
			continue
		}

		// Remove if idle for longer than keep-alive time
		if now.Sub(conn.LastUsed) > p.keepAliveTime {
			toRemove = append(toRemove, serverID)
		}
	}

	// Remove idle connections
	for _, serverID := range toRemove {
		if conn, exists := p.connections[serverID]; exists {
			// Disconnect server
			if conn.Server != nil {
				_ = conn.Server.Disconnect()
			}
			delete(p.connections, serverID)
			p.stats.ActiveConnections--
		}
	}

	if len(toRemove) > 0 {
		p.stats.LastUpdated = time.Now()
	}
}

// Close shuts down the connection pool and all connections
func (p *ConnectionPool) Close() error {
	// Stop background worker
	close(p.stopChan)
	<-p.doneChan

	p.mu.Lock()
	defer p.mu.Unlock()

	// Disconnect all servers
	for _, conn := range p.connections {
		if conn.Server != nil {
			_ = conn.Server.Disconnect()
		}
	}

	// Clear connections
	p.connections = make(map[string]*PooledConnection)
	p.stats.ActiveConnections = 0
	p.stats.LastUpdated = time.Now()

	return nil
}

// IsPreWarmed returns whether a server has been pre-warmed
func (p *ConnectionPool) IsPreWarmed(serverID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if conn, exists := p.connections[serverID]; exists {
		return conn.PreWarmed
	}

	return false
}

// SetKeepAlive enables or disables keep-alive for a connection
func (p *ConnectionPool) SetKeepAlive(serverID string, keepAlive bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[serverID]; exists {
		conn.KeepAlive = keepAlive
	}
}

// GetConnection retrieves connection information for a server
func (p *ConnectionPool) GetConnection(serverID string) (*PooledConnection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conn, exists := p.connections[serverID]
	return conn, exists
}

// Size returns the number of active connections in the pool
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.connections)
}
