package mcp

import (
	"context"
	"sync"
	"time"
)

const (
	// HealthCheckInterval is how often to check server health
	HealthCheckInterval = 30 * time.Second
	// HealthCheckTimeout is the timeout for a single health check
	HealthCheckTimeout = 5 * time.Second
	// MaxFailedChecks before marking server as unhealthy
	MaxFailedChecks = 3
)

// HealthMonitor monitors the health of MCP servers
type HealthMonitor struct {
	pool         *ConnectionPool
	healthStatus map[string]*ServerHealth
	mu           sync.RWMutex
	stopChan     chan struct{}
	stopped      bool
}

// ServerHealth tracks health status for a server
type ServerHealth struct {
	ServerID         string
	IsHealthy        bool
	LastCheck        time.Time
	LastSuccess      time.Time
	FailedCheckCount int
	LastError        error
	mu               sync.RWMutex
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(pool *ConnectionPool) *HealthMonitor {
	hm := &HealthMonitor{
		pool:         pool,
		healthStatus: make(map[string]*ServerHealth),
		stopChan:     make(chan struct{}),
	}

	// Start background health checking
	go hm.startHealthChecks()

	return hm
}

// RegisterServer registers a server for health monitoring
func (hm *HealthMonitor) RegisterServer(serverID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.healthStatus[serverID]; !exists {
		hm.healthStatus[serverID] = &ServerHealth{
			ServerID:  serverID,
			IsHealthy: true, // Assume healthy initially
			LastCheck: time.Now(),
		}
	}
}

// GetHealth returns the current health status for a server
func (hm *HealthMonitor) GetHealth(serverID string) (*ServerHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.healthStatus[serverID]
	if !exists {
		return nil, false
	}

	health.mu.RLock()
	defer health.mu.RUnlock()

	// Create a copy to avoid race conditions
	healthCopy := &ServerHealth{
		ServerID:         health.ServerID,
		IsHealthy:        health.IsHealthy,
		LastCheck:        health.LastCheck,
		LastSuccess:      health.LastSuccess,
		FailedCheckCount: health.FailedCheckCount,
		LastError:        health.LastError,
	}

	return healthCopy, true
}

// GetAllHealth returns health status for all monitored servers
func (hm *HealthMonitor) GetAllHealth() map[string]*ServerHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]*ServerHealth)
	for serverID := range hm.healthStatus {
		if health, ok := hm.GetHealth(serverID); ok {
			result[serverID] = health
		}
	}

	return result
}

// CheckNow performs an immediate health check for a server
func (hm *HealthMonitor) CheckNow(ctx context.Context, serverID string) error {
	return hm.performHealthCheck(ctx, serverID)
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.stopped {
		close(hm.stopChan)
		hm.stopped = true
	}
}

// startHealthChecks runs periodic health checks in the background
func (hm *HealthMonitor) startHealthChecks() {
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.stopChan:
			return
		case <-ticker.C:
			hm.checkAllServers()
		}
	}
}

// checkAllServers performs health checks on all registered servers
func (hm *HealthMonitor) checkAllServers() {
	hm.mu.RLock()
	serverIDs := make([]string, 0, len(hm.healthStatus))
	for serverID := range hm.healthStatus {
		serverIDs = append(serverIDs, serverID)
	}
	hm.mu.RUnlock()

	// Check each server concurrently
	var wg sync.WaitGroup
	for _, serverID := range serverIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
			defer cancel()
			_ = hm.performHealthCheck(ctx, id)
		}(serverID)
	}

	wg.Wait()
}

// performHealthCheck performs a single health check on a server
func (hm *HealthMonitor) performHealthCheck(ctx context.Context, serverID string) error {
	hm.mu.RLock()
	health, exists := hm.healthStatus[serverID]
	hm.mu.RUnlock()

	if !exists {
		return nil // Server not registered
	}

	// Get a connection from the pool
	client, err := hm.pool.Get(ctx, serverID)
	if err != nil {
		hm.recordFailure(health, err)
		return err
	}
	defer func() { _ = hm.pool.Release(serverID, client) }()

	// Perform ping
	err = client.Ping(ctx)

	health.mu.Lock()
	health.LastCheck = time.Now()

	if err != nil {
		health.FailedCheckCount++
		health.LastError = err

		// Mark as unhealthy after max failed checks
		if health.FailedCheckCount >= MaxFailedChecks {
			health.IsHealthy = false
		}
	} else {
		// Successful check
		health.FailedCheckCount = 0
		health.LastError = nil
		health.IsHealthy = true
		health.LastSuccess = time.Now()
	}

	health.mu.Unlock()

	return err
}

// recordFailure records a health check failure
func (hm *HealthMonitor) recordFailure(health *ServerHealth, err error) {
	health.mu.Lock()
	defer health.mu.Unlock()

	health.LastCheck = time.Now()
	health.FailedCheckCount++
	health.LastError = err

	// Mark as unhealthy after max failed checks
	if health.FailedCheckCount >= MaxFailedChecks {
		health.IsHealthy = false
	}
}

// MarkHealthy manually marks a server as healthy (useful after successful operations)
func (hm *HealthMonitor) MarkHealthy(serverID string) {
	hm.mu.RLock()
	health, exists := hm.healthStatus[serverID]
	hm.mu.RUnlock()

	if !exists {
		return
	}

	health.mu.Lock()
	health.IsHealthy = true
	health.FailedCheckCount = 0
	health.LastError = nil
	health.LastSuccess = time.Now()
	health.mu.Unlock()
}

// MarkUnhealthy manually marks a server as unhealthy
func (hm *HealthMonitor) MarkUnhealthy(serverID string, err error) {
	hm.mu.RLock()
	health, exists := hm.healthStatus[serverID]
	hm.mu.RUnlock()

	if !exists {
		return
	}

	health.mu.Lock()
	health.IsHealthy = false
	health.FailedCheckCount = MaxFailedChecks
	health.LastError = err
	health.LastCheck = time.Now()
	health.mu.Unlock()
}
