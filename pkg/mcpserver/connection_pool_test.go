package mcpserver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectionPool(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.Equal(t, 0, pool.Size())

	stats := pool.GetStats()
	assert.Equal(t, int64(0), stats.ActiveConnections)
	assert.Equal(t, int64(0), stats.UsageHits)
	assert.Equal(t, int64(0), stats.UsageMisses)
}

func TestPoolGetAndRelease(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	// Create test server
	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)

	// Mark as connected for testing
	server.Connection.State = StateConnected

	ctx := context.Background()

	// First get should create connection
	conn1, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.NotNil(t, conn1)
	assert.Equal(t, int64(1), conn1.UsageCount)

	stats := pool.GetStats()
	assert.Equal(t, int64(1), stats.ActiveConnections)
	assert.Equal(t, int64(0), stats.UsageHits)
	assert.Equal(t, int64(1), stats.UsageMisses)

	// Release connection
	pool.Release(server.ID)

	// Second get should reuse connection
	conn2, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.NotNil(t, conn2)
	assert.Equal(t, int64(2), conn2.UsageCount)
	assert.Equal(t, conn1, conn2) // Should be same connection

	stats = pool.GetStats()
	assert.Equal(t, int64(1), stats.ActiveConnections)
	assert.Equal(t, int64(1), stats.UsageHits)
	assert.Equal(t, int64(1), stats.UsageMisses)
	assert.Equal(t, 0.5, stats.ReuseRate)
}

func TestPoolPreWarming(t *testing.T) {
	pool := NewConnectionPoolWithConfig(2, 5*time.Minute)
	defer pool.Close()

	// Create registry with test servers
	registry := NewRegistry()

	server1, err := NewMCPServer("server-1", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	err = registry.Register(server1)
	require.NoError(t, err)

	server2, err := NewMCPServer("server-2", "cat", []string{}, TransportStdio)
	require.NoError(t, err)
	err = registry.Register(server2)
	require.NoError(t, err)

	ctx := context.Background()

	// Pre-warm connections
	serverIDs := []string{"server-1", "server-2"}
	err = pool.PreWarm(ctx, registry, serverIDs)
	require.NoError(t, err)

	// Verify connections are pre-warmed
	assert.True(t, pool.IsPreWarmed("server-1"))
	assert.True(t, pool.IsPreWarmed("server-2"))

	stats := pool.GetStats()
	assert.Equal(t, int64(2), stats.ActiveConnections)
	assert.Equal(t, int64(2), stats.PreWarmedConnections)

	// Verify connections have keep-alive enabled
	conn1, exists := pool.GetConnection("server-1")
	assert.True(t, exists)
	assert.True(t, conn1.KeepAlive)

	conn2, exists := pool.GetConnection("server-2")
	assert.True(t, exists)
	assert.True(t, conn2.KeepAlive)
}

func TestPoolAutoPreWarming(t *testing.T) {
	// Create pool with threshold of 3
	pool := NewConnectionPoolWithConfig(3, 5*time.Minute)
	defer pool.Close()

	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	server.Connection.State = StateConnected

	ctx := context.Background()

	// Initially not pre-warmed
	assert.False(t, pool.IsPreWarmed(server.ID))

	// Use connection multiple times
	for i := 0; i < 3; i++ {
		conn, err := pool.Get(ctx, server)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		pool.Release(server.ID)
	}

	// After threshold uses, should be pre-warmed
	assert.True(t, pool.IsPreWarmed(server.ID))

	conn, exists := pool.GetConnection(server.ID)
	assert.True(t, exists)
	assert.True(t, conn.PreWarmed)
	assert.True(t, conn.KeepAlive)
}

func TestPoolGetFrequentServers(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	ctx := context.Background()

	// Create servers with different usage patterns
	servers := []*MCPServer{}
	usageCounts := []int{5, 10, 3, 8, 1}

	for i, count := range usageCounts {
		server, err := NewMCPServer(
			fmt.Sprintf("server-%d", i),
			"echo",
			[]string{},
			TransportStdio,
		)
		require.NoError(t, err)
		server.Connection.State = StateConnected
		servers = append(servers, server)

		// Simulate usage
		for j := 0; j < count; j++ {
			_, err := pool.Get(ctx, server)
			require.NoError(t, err)
		}
	}

	// Get top 3 frequent servers
	frequent := pool.GetFrequentServers(3)
	assert.Equal(t, 3, len(frequent))

	// Should be ordered by usage: server-1 (10), server-3 (8), server-0 (5)
	assert.Equal(t, "server-1", frequent[0])
	assert.Equal(t, "server-3", frequent[1])
	assert.Equal(t, "server-0", frequent[2])
}

func TestPoolRemove(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	server.Connection.State = StateConnected

	ctx := context.Background()

	// Add connection
	conn, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	// Verify exists
	assert.Equal(t, 1, pool.Size())

	// Remove connection
	pool.Remove(server.ID)

	// Verify removed
	assert.Equal(t, 0, pool.Size())
	_, exists := pool.GetConnection(server.ID)
	assert.False(t, exists)
}

func TestPoolSetKeepAlive(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	server.Connection.State = StateConnected

	ctx := context.Background()

	// Create connection
	conn, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.False(t, conn.KeepAlive) // Initially false

	// Enable keep-alive
	pool.SetKeepAlive(server.ID, true)

	conn, exists := pool.GetConnection(server.ID)
	assert.True(t, exists)
	assert.True(t, conn.KeepAlive)

	// Disable keep-alive
	pool.SetKeepAlive(server.ID, false)

	conn, exists = pool.GetConnection(server.ID)
	assert.True(t, exists)
	assert.False(t, conn.KeepAlive)
}

func TestPoolCleanupIdle(t *testing.T) {
	// Create pool with very short keep-alive time
	pool := NewConnectionPoolWithConfig(3, 50*time.Millisecond)
	defer pool.Close()

	ctx := context.Background()

	// Create regular connection (not pre-warmed)
	server1, err := NewMCPServer("server-1", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	server1.Connection.State = StateConnected

	conn1, err := pool.Get(ctx, server1)
	require.NoError(t, err)
	assert.NotNil(t, conn1)
	pool.Release(server1.ID)

	// Create pre-warmed connection
	server2, err := NewMCPServer("server-2", "cat", []string{}, TransportStdio)
	require.NoError(t, err)
	server2.Connection.State = StateConnected

	conn2, err := pool.Get(ctx, server2)
	require.NoError(t, err)
	assert.NotNil(t, conn2)
	pool.SetKeepAlive(server2.ID, true) // Pre-warmed connections have keep-alive
	pool.Release(server2.ID)

	// Verify both exist
	assert.Equal(t, 2, pool.Size())

	// Wait for keep-alive to expire
	time.Sleep(100 * time.Millisecond)

	// Trigger cleanup
	pool.cleanupIdle()

	// Regular connection should be removed, pre-warmed should remain
	assert.Equal(t, 1, pool.Size())
	_, exists := pool.GetConnection(server1.ID)
	assert.False(t, exists)
	_, exists = pool.GetConnection(server2.ID)
	assert.True(t, exists)
}

func TestPoolStatsCalculation(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)
	server.Connection.State = StateConnected

	ctx := context.Background()

	// First access - miss
	_, err = pool.Get(ctx, server)
	require.NoError(t, err)

	stats := pool.GetStats()
	assert.Equal(t, int64(0), stats.UsageHits)
	assert.Equal(t, int64(1), stats.UsageMisses)
	assert.Equal(t, 0.0, stats.ReuseRate)

	// Second access - hit
	_, err = pool.Get(ctx, server)
	require.NoError(t, err)

	stats = pool.GetStats()
	assert.Equal(t, int64(1), stats.UsageHits)
	assert.Equal(t, int64(1), stats.UsageMisses)
	assert.Equal(t, 0.5, stats.ReuseRate)

	// Third access - hit
	_, err = pool.Get(ctx, server)
	require.NoError(t, err)

	stats = pool.GetStats()
	assert.Equal(t, int64(2), stats.UsageHits)
	assert.Equal(t, int64(1), stats.UsageMisses)
	assert.InDelta(t, 0.666, stats.ReuseRate, 0.01)
}

func TestPoolClose(t *testing.T) {
	pool := NewConnectionPool()

	// Add some connections
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		server, err := NewMCPServer(
			fmt.Sprintf("server-%d", i),
			"echo",
			[]string{},
			TransportStdio,
		)
		require.NoError(t, err)
		server.Connection.State = StateConnected

		_, err = pool.Get(ctx, server)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, pool.Size())

	// Close pool
	err := pool.Close()
	require.NoError(t, err)

	// Verify all connections removed
	assert.Equal(t, 0, pool.Size())

	stats := pool.GetStats()
	assert.Equal(t, int64(0), stats.ActiveConnections)
}

func TestPoolDisconnectedServer(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	server, err := NewMCPServer("test-server", "echo", []string{}, TransportStdio)
	require.NoError(t, err)

	ctx := context.Background()

	// First connect and get
	server.Connection.State = StateConnected
	conn1, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.NotNil(t, conn1)

	// Disconnect server
	server.Connection.State = StateDisconnected

	// Next get should create new connection (old one is invalid)
	conn2, err := pool.Get(ctx, server)
	require.NoError(t, err)
	assert.NotNil(t, conn2)

	// Should have triggered a miss (connection recreated)
	stats := pool.GetStats()
	assert.Equal(t, int64(2), stats.UsageMisses)
}
