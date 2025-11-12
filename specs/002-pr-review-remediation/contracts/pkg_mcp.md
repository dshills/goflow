// Package contracts defines API contracts for the 002-pr-review-remediation feature.
//
// This file documents contract changes (API fixes) for the mcp package.
// Implementation will be in pkg/mcp/
package contracts

import (
	"context"
	"sync"
	"time"
)

// ConnectionPool manages a pool of MCP server connections for reuse.
//
// FIXED: API signatures corrected for consistency
// ENHANCED: Graceful shutdown support
//
// Thread-safe for concurrent use.
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*PooledConnection
	servers     map[string]*MCPServer
	maxIdle     time.Duration
	cleanupTick *time.Ticker

	// NEW: Shutdown coordination
	closing chan struct{}  // Signals shutdown in progress
	wg      sync.WaitGroup // Tracks active operations
}

// NewConnectionPool creates a new connection pool.
//
// ENHANCED: Initializes new shutdown coordination fields
func NewConnectionPool(maxIdle time.Duration) *ConnectionPool

// Get acquires a connection from the pool.
//
// FIXED: Signature changed from Get(ctx, server) to Get(ctx, serverID)
//
// Returns:
//   - *PooledConnection if successful
//   - error if serverID not found, pool is closing, or connection failed
//
// The connection must be released back to the pool via Release().
//
// Example:
//
//	conn, err := pool.Get(ctx, "my-server")
//	if err != nil {
//	    return err
//	}
//	defer pool.Release("my-server")
//	// Use conn.Client
func (p *ConnectionPool) Get(ctx context.Context, serverID string) (*PooledConnection, error)

// Release returns a connection to the pool.
//
// FIXED: Signature changed from Release(serverID, client) to Release(serverID)
//
// The pool internally tracks which connection belongs to which serverID,
// so only the serverID is needed to release.
//
// If the connection is stale or the pool is closing, it will be closed
// instead of returned to the pool.
//
// Example:
//
//	conn, _ := pool.Get(ctx, "my-server")
//	defer pool.Release("my-server")
func (p *ConnectionPool) Release(serverID string) error

// Close shuts down the connection pool gracefully.
//
// ENHANCED: Now implements graceful shutdown with timeout
//
// Behavior:
//  1. Mark pool as closing (no new Get() calls accepted)
//  2. Wait up to 30 seconds for active operations to complete
//  3. Force-close any remaining connections
//  4. Release all resources
//
// Returns error if force-close was required (graceful shutdown timeout exceeded).
//
// Example:
//
//	defer pool.Close()
func (p *ConnectionPool) Close() error

// PooledConnection represents a connection managed by the pool.
//
// ENHANCED: Lifecycle tracking added
type PooledConnection struct {
	ServerID   string
	Client     interface{} // protocol.Client (avoiding import for contract definition)
	AcquiredAt time.Time
	LastUsedAt time.Time

	// NEW: Lifecycle tracking
	closed   bool  // Whether connection is closed
	closeErr error // Error from close operation
	refCount int32 // Reference count for leak detection
}

// IsClosed returns whether the connection has been closed.
//
// NEW METHOD: Check if connection is still usable
func (c *PooledConnection) IsClosed() bool

// MCPServer represents configuration for an MCP server.
//
// UNCHANGED: No modifications to server configuration
type MCPServer struct {
	ID      string
	Command string
	Args    []string
	// ... other fields unchanged ...
}

// PerformHealthCheck checks if an MCP server is healthy.
//
// FIXED: Now uses corrected Get(ctx, serverID) and Release(serverID) signatures
//
// Returns error if:
//   - Server not found in pool
//   - Cannot acquire connection
//   - Connection is unhealthy
//
// Example:
//
//	if err := PerformHealthCheck(ctx, pool, "my-server"); err != nil {
//	    log.Printf("Server unhealthy: %v", err)
//	}
func PerformHealthCheck(ctx context.Context, pool *ConnectionPool, serverID string) error
