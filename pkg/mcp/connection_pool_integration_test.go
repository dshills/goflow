package mcp

import (
	"context"
	"sync"
	"testing"
	"time"
)

// T065: Connection Pool Integration Tests

// TestConnectionPoolIntegration_FullLifecycle tests the complete connection lifecycle
func TestConnectionPoolIntegration_FullLifecycle(t *testing.T) {
	pool := newTestPoolWithFastCleanup()
	defer pool.Close()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	if err := pool.RegisterServer(config); err != nil {
		t.Fatalf("RegisterServer() failed: %v", err)
	}

	// Add mock connection
	addMockConnection(pool, "test-server")

	ctx := context.Background()

	// 1. Acquire connection
	client, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if client == nil {
		t.Fatal("Get() returned nil client")
	}

	// Verify stats show active connection
	stats := pool.Stats()
	if stats["test-server"].Active != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats["test-server"].Active)
	}

	// 2. Use connection (simulate work)
	time.Sleep(100 * time.Millisecond)

	// 3. Release connection
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	// Verify stats show idle connection
	stats = pool.Stats()
	if stats["test-server"].Idle != 1 {
		t.Errorf("Expected 1 idle connection, got %d", stats["test-server"].Idle)
	}

	// 4. Connection becomes stale (simulate by setting LastUsed)
	pool.mu.Lock()
	conn := pool.connections["test-server"][0]
	conn.mu.Lock()
	conn.LastUsed = time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	conn.mu.Unlock()
	pool.mu.Unlock()

	// Wait for cleanup cycle
	time.Sleep(1500 * time.Millisecond)

	// 5. Verify cleanup removed stale connection
	pool.mu.RLock()
	connections := pool.connections["test-server"]
	pool.mu.RUnlock()

	if len(connections) != 0 {
		t.Errorf("Expected stale connection to be cleaned up, found %d connections", len(connections))
	}

	// Verify no leaks
	leaks := pool.LeakStats()
	if leaks != 0 {
		t.Errorf("Expected 0 leaks, got %d", leaks)
	}
}

// TestConnectionPoolIntegration_MultipleServers tests multiple servers in same pool
func TestConnectionPoolIntegration_MultipleServers(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	// Register multiple servers
	servers := []string{"server-1", "server-2", "server-3"}
	for _, serverID := range servers {
		config := ServerConfig{
			ID:        serverID,
			Transport: "stdio",
			Command:   "echo",
			Args:      []string{"test"},
		}
		if err := pool.RegisterServer(config); err != nil {
			t.Fatalf("RegisterServer(%s) failed: %v", serverID, err)
		}

		// Add mock connections
		addMockConnection(pool, serverID)
		addMockConnection(pool, serverID)
	}

	ctx := context.Background()

	// Acquire connections from different servers
	for _, serverID := range servers {
		client, err := pool.Get(ctx, serverID)
		if err != nil {
			t.Errorf("Get(%s) failed: %v", serverID, err)
			continue
		}
		if client == nil {
			t.Errorf("Get(%s) returned nil client", serverID)
		}
	}

	// Verify stats for all servers
	stats := pool.Stats()
	for _, serverID := range servers {
		if stats[serverID].Active != 1 {
			t.Errorf("Server %s: expected 1 active connection, got %d", serverID, stats[serverID].Active)
		}
		if stats[serverID].Total != 2 {
			t.Errorf("Server %s: expected 2 total connections, got %d", serverID, stats[serverID].Total)
		}
	}

	// Release all connections
	for _, serverID := range servers {
		if err := pool.Release(serverID); err != nil {
			t.Errorf("Release(%s) failed: %v", serverID, err)
		}
	}

	// Verify all are now idle
	stats = pool.Stats()
	for _, serverID := range servers {
		if stats[serverID].Idle != 2 {
			t.Errorf("Server %s: expected 2 idle connections, got %d", serverID, stats[serverID].Idle)
		}
	}
}

// TestConnectionPoolIntegration_ConnectionReuse tests connection reuse behavior
func TestConnectionPoolIntegration_ConnectionReuse(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Add mock connection
	addMockConnection(pool, "test-server")

	ctx := context.Background()

	// Acquire, release, and reacquire multiple times
	const numCycles = 5
	var firstClient Client

	for i := 0; i < numCycles; i++ {
		client, err := pool.Get(ctx, "test-server")
		if err != nil {
			t.Fatalf("Cycle %d: Get() failed: %v", i+1, err)
		}

		if i == 0 {
			firstClient = client
		} else {
			// Should reuse the same connection
			if client != firstClient {
				t.Errorf("Cycle %d: Expected to reuse same connection", i+1)
			}
		}

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		if err := pool.Release("test-server"); err != nil {
			t.Fatalf("Cycle %d: Release() failed: %v", i+1, err)
		}
	}

	// Verify only one connection was created
	stats := pool.Stats()
	if stats["test-server"].Total != 1 {
		t.Errorf("Expected 1 total connection (reused), got %d", stats["test-server"].Total)
	}
}

// TestConnectionPoolIntegration_CleanupUnderLoad tests cleanup works under concurrent load
func TestConnectionPoolIntegration_CleanupUnderLoad(t *testing.T) {
	pool := newTestPoolWithFastCleanup()
	defer pool.Close()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Add multiple mock connections
	for i := 0; i < 10; i++ {
		addMockConnection(pool, "test-server")
	}

	ctx := context.Background()

	// Run concurrent operations while cleanup is happening
	const numWorkers = 5
	const numOperations = 10
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers*numOperations)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Acquire
				_, err := pool.Get(ctx, "test-server")
				if err != nil {
					errors <- err
					continue
				}

				// Hold connection briefly
				time.Sleep(50 * time.Millisecond)

				// Release
				if err := pool.Release("test-server"); err != nil {
					errors <- err
				}

				// Allow cleanup cycles to run
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	// Add stale connections periodically
	staleMaker := func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			time.Sleep(500 * time.Millisecond)

			staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
			staleConn := &PooledConnection{
				Client:     &mockClient{connected: true},
				ServerID:   "test-server",
				LastUsed:   staleTime,
				AcquiredAt: staleTime,
				InUse:      false,
				refCount:   0,
			}

			pool.mu.Lock()
			pool.connections["test-server"] = append(pool.connections["test-server"], staleConn)
			pool.mu.Unlock()
		}
	}
	wg.Add(1)
	go staleMaker()

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var errCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errCount++
	}

	if errCount > 0 {
		t.Fatalf("Found %d errors during concurrent operations", errCount)
	}

	// Wait for final cleanup
	time.Sleep(2 * time.Second)

	// Verify cleanup worked correctly
	stats := pool.Stats()
	if stats["test-server"].Total > 10 {
		t.Errorf("Expected cleanup to keep connection count reasonable, got %d connections", stats["test-server"].Total)
	}

	// Verify no leaks were detected
	leaks := pool.LeakStats()
	if leaks != 0 {
		t.Errorf("Expected 0 leaks under load, got %d", leaks)
	}
}

// TestConnectionPoolIntegration_GracefulShutdownWithActiveConnections tests shutdown with active operations
func TestConnectionPoolIntegration_GracefulShutdownWithActiveConnections(t *testing.T) {
	pool := NewConnectionPool()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Add mock connections
	for i := 0; i < 5; i++ {
		addMockConnection(pool, "test-server")
	}

	ctx := context.Background()

	// Start multiple long-running operations
	const numOps = 5
	var wg sync.WaitGroup
	completedOps := make(chan bool, numOps)

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire connection
			_, err := pool.Get(ctx, "test-server")
			if err != nil {
				return
			}

			// Simulate work
			time.Sleep(200 * time.Millisecond)

			// Release
			pool.Release("test-server")
			completedOps <- true
		}()
	}

	// Give operations time to start
	time.Sleep(50 * time.Millisecond)

	// Close pool (should wait for operations to complete)
	closeDone := make(chan bool)
	go func() {
		pool.Close()
		closeDone <- true
	}()

	// Wait for all operations
	wg.Wait()
	close(completedOps)

	// Count completed operations
	var completed int
	for range completedOps {
		completed++
	}

	if completed != numOps {
		t.Errorf("Expected %d operations to complete before shutdown, got %d", numOps, completed)
	}

	// Verify close completed
	select {
	case <-closeDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Close() did not complete after all operations finished")
	}
}
