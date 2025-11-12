package mcp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// mockClient implements Client interface for testing
type mockClient struct {
	connected bool
	closed    bool
	mu        sync.Mutex
}

func (m *mockClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	m.connected = false
	return nil
}

func (m *mockClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected && !m.closed
}

func (m *mockClient) ListTools(ctx context.Context) ([]mcpserver.Tool, error) {
	return nil, nil
}

func (m *mockClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockClient) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected || m.closed {
		return errors.New("not connected")
	}
	return nil
}

func (m *mockClient) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// addMockConnection is a helper to add a mock connection to the pool
func addMockConnection(pool *ConnectionPool, serverID string) {
	mockConn := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   serverID,
		LastUsed:   time.Now(),
		AcquiredAt: time.Now(),
		InUse:      false,
		refCount:   0,
	}
	pool.mu.Lock()
	if _, exists := pool.connections[serverID]; !exists {
		pool.connections[serverID] = []*PooledConnection{}
	}
	pool.connections[serverID] = append(pool.connections[serverID], mockConn)
	pool.mu.Unlock()
}

// TestConnectionPoolGet_ValidServerID tests Get() with valid serverID
func TestConnectionPoolGet_ValidServerID(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	// Register a test server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}

	if err := pool.RegisterServer(config); err != nil {
		t.Fatalf("Failed to register server: %v", err)
	}

	// Pre-add a mock client to the pool to avoid actual connection
	addMockConnection(pool, "test-server")

	// Get connection
	ctx := context.Background()
	conn, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if conn == nil {
		t.Fatal("Get() returned nil connection")
	}

	// Release connection before Close() is called by defer
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}
}

// TestConnectionPoolGet_InvalidServerID tests Get() with invalid serverID
func TestConnectionPoolGet_InvalidServerID(t *testing.T) {
	pool := NewConnectionPool()
	defer pool.Close()

	ctx := context.Background()
	_, err := pool.Get(ctx, "nonexistent-server")
	if err == nil {
		t.Fatal("Get() should fail with invalid serverID")
	}

	expectedMsg := "server nonexistent-server not registered"
	if err.Error() != expectedMsg {
		t.Errorf("Get() error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestConnectionPoolRelease_ValidConnection tests Release() with valid connection
func TestConnectionPoolRelease_ValidConnection(t *testing.T) {
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

	// Pre-add a mock client to the pool
	addMockConnection(pool, "test-server")

	// Get connection
	ctx := context.Background()
	client, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Release connection
	if err := pool.Release("test-server"); err != nil {
		t.Errorf("Release() failed: %v", err)
	}

	// Verify connection can be reused
	stats := pool.Stats()
	if stats["test-server"].Idle != 1 {
		t.Errorf("Expected 1 idle connection after release, got %d", stats["test-server"].Idle)
	}

	// Verify we can get the same connection again
	client2, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Second Get() failed: %v", err)
	}

	if client2 != client {
		t.Error("Expected to get the same connection from pool")
	}

	// Release connection again before Close() is called by defer
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Second Release() failed: %v", err)
	}
}

// TestConnectionPoolGet_AfterClose tests Get() after Close()
func TestConnectionPoolGet_AfterClose(t *testing.T) {
	pool := NewConnectionPool()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Close pool
	if err := pool.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Try to get connection after close
	ctx := context.Background()
	_, err := pool.Get(ctx, "test-server")
	if err == nil {
		t.Fatal("Get() should fail after Close()")
	}

	expectedMsg := "connection pool is closed"
	if err.Error() != expectedMsg {
		t.Errorf("Get() after Close() error = %q, want %q", err.Error(), expectedMsg)
	}
}

// TestConnectionPoolConcurrent tests concurrent Get() and Release() operations
func TestConnectionPoolConcurrent(t *testing.T) {
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

	// Add multiple mock connections to the pool
	for i := 0; i < 10; i++ {
		addMockConnection(pool, "test-server")
	}

	// Run concurrent operations (fewer goroutines than connections to avoid exhaustion)
	const numGoroutines = 5
	const numIterations = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				ctx := context.Background()

				// Get connection
				_, err := pool.Get(ctx, "test-server")
				if err != nil {
					errors <- err
					continue
				}

				// Simulate some work
				time.Sleep(1 * time.Millisecond)

				// Release connection
				if err := pool.Release("test-server"); err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errCount++
	}

	if errCount > 0 {
		t.Fatalf("Found %d errors in concurrent operations", errCount)
	}
}

// TestConnectionPoolLifecycle_NormalAcquisitionRelease tests normal lifecycle
func TestConnectionPoolLifecycle_NormalAcquisitionRelease(t *testing.T) {
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

	// Acquire
	conn1, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("First Get() failed: %v", err)
	}

	stats := pool.Stats()
	if stats["test-server"].Active != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats["test-server"].Active)
	}

	// Release
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	stats = pool.Stats()
	if stats["test-server"].Idle != 1 {
		t.Errorf("Expected 1 idle connection after release, got %d", stats["test-server"].Idle)
	}

	// Acquire again (should reuse)
	conn2, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Second Get() failed: %v", err)
	}

	if conn1 != conn2 {
		t.Error("Expected to reuse the same connection")
	}

	// Release connection again before Close() is called by defer
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Second Release() failed: %v", err)
	}
}

// TestConnectionPoolGracefulShutdown tests graceful shutdown
func TestConnectionPoolGracefulShutdown(t *testing.T) {
	pool := NewConnectionPool()

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

	// Acquire connection
	_, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Start goroutine that will release after a delay
	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond)
		pool.Release("test-server")
		done <- true
	}()

	// Close should wait for release
	start := time.Now()
	if err := pool.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	elapsed := time.Since(start)

	// Wait for goroutine to complete
	<-done

	// Should have waited at least 100ms for graceful shutdown
	if elapsed < 100*time.Millisecond {
		t.Errorf("Close() returned too quickly: %v", elapsed)
	}

	// Should not have waited for full 30s timeout
	if elapsed > 5*time.Second {
		t.Errorf("Close() took too long: %v", elapsed)
	}
}

// TestConnectionPoolForceClose tests force-close after grace period
func TestConnectionPoolForceClose(t *testing.T) {
	// This test would take 30+ seconds to run, so we'll skip it in normal test runs
	// unless explicitly enabled
	if testing.Short() {
		t.Skip("Skipping force-close test in short mode")
	}

	pool := NewConnectionPool()

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

	// Acquire connection
	_, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Don't release connection (simulate stuck operation)

	// Close should force-close after grace period
	start := time.Now()
	err = pool.Close()
	elapsed := time.Since(start)

	// Should return error indicating force-close was required
	if err == nil {
		t.Error("Close() should return error when force-close is required")
	}

	// Should have waited approximately 30s
	if elapsed < 29*time.Second || elapsed > 31*time.Second {
		t.Errorf("Close() should wait ~30s for grace period, waited: %v", elapsed)
	}
}

// TestPooledConnectionIsClosed tests IsClosed() method
func TestPooledConnectionIsClosed(t *testing.T) {
	// Create a pooled connection with mock client
	client := &mockClient{connected: true}
	conn := &PooledConnection{
		Client:   client,
		ServerID: "test-server",
		LastUsed: time.Now(),
		InUse:    false,
		closed:   false,
	}

	// Should not be closed initially
	if conn.IsClosed() {
		t.Error("New connection should not be closed")
	}

	// Mark as closed
	conn.closed = true

	// Should now report as closed
	if !conn.IsClosed() {
		t.Error("Connection should report as closed")
	}
}

// TestConnectionPoolShutdownCoordination tests shutdown coordination with waitgroup
func TestConnectionPoolShutdownCoordination(t *testing.T) {
	pool := NewConnectionPool()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Add multiple mock connections to the pool
	for i := 0; i < 5; i++ {
		addMockConnection(pool, "test-server")
	}

	ctx := context.Background()

	// Start multiple operations
	const numOps = 5
	var wg sync.WaitGroup
	operationsDone := make(chan bool, numOps)

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

			// Release connection
			pool.Release("test-server")
			operationsDone <- true
		}()
	}

	// Give operations time to start
	time.Sleep(50 * time.Millisecond)

	// Start closing in background
	closeDone := make(chan bool)
	go func() {
		pool.Close()
		closeDone <- true
	}()

	// Wait for all operations to complete
	wg.Wait()
	close(operationsDone)

	// Count completed operations
	completedOps := 0
	for range operationsDone {
		completedOps++
	}

	// All operations should have completed
	if completedOps != numOps {
		t.Errorf("Expected %d operations to complete, got %d", numOps, completedOps)
	}

	// Close should now be done
	select {
	case <-closeDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Close() did not complete after all operations finished")
	}
}

// T061: Connection Cleanup Tests

// newTestPoolWithFastCleanup creates a pool with 1-second cleanup interval for testing
func newTestPoolWithFastCleanup() *ConnectionPool {
	return newConnectionPoolWithInterval(1 * time.Second)
}

// TestConnectionCleanup_StaleDetection tests that stale connections are detected
func TestConnectionCleanup_StaleDetection(t *testing.T) {
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

	// Add a stale connection (LastUsed beyond ConnectionIdleTimeout)
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
	pool.connections["test-server"] = []*PooledConnection{staleConn}
	pool.mu.Unlock()

	// Wait for cleanup cycle (1 second + buffer)
	time.Sleep(1500 * time.Millisecond)

	// Check that stale connection was removed
	pool.mu.RLock()
	connections := pool.connections["test-server"]
	pool.mu.RUnlock()

	if len(connections) != 0 {
		t.Errorf("Expected stale connection to be removed, found %d connections", len(connections))
	}

	// Verify connection was closed
	mock := staleConn.Client.(*mockClient)
	if !mock.IsClosed() {
		t.Error("Expected stale connection client to be closed")
	}
}

// TestConnectionCleanup_RemovalFromPool tests stale connection removal
func TestConnectionCleanup_RemovalFromPool(t *testing.T) {
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

	// Add multiple connections: 1 stale, 2 active
	staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	freshTime := time.Now()

	staleConn := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   "test-server",
		LastUsed:   staleTime,
		AcquiredAt: staleTime,
		InUse:      false,
		refCount:   0,
	}

	fresh1 := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   "test-server",
		LastUsed:   freshTime,
		AcquiredAt: freshTime,
		InUse:      false,
		refCount:   0,
	}

	fresh2 := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   "test-server",
		LastUsed:   freshTime,
		AcquiredAt: freshTime,
		InUse:      false,
		refCount:   0,
	}

	pool.mu.Lock()
	pool.connections["test-server"] = []*PooledConnection{staleConn, fresh1, fresh2}
	pool.mu.Unlock()

	// Wait for cleanup cycle
	time.Sleep(1500 * time.Millisecond)

	// Check that only fresh connections remain
	pool.mu.RLock()
	connections := pool.connections["test-server"]
	pool.mu.RUnlock()

	if len(connections) != 2 {
		t.Errorf("Expected 2 fresh connections to remain, found %d", len(connections))
	}

	// Verify stale was closed but fresh are still connected
	staleMock := staleConn.Client.(*mockClient)
	if !staleMock.IsClosed() {
		t.Error("Expected stale connection to be closed")
	}

	for i, conn := range []*PooledConnection{fresh1, fresh2} {
		mock := conn.Client.(*mockClient)
		if mock.IsClosed() {
			t.Errorf("Expected fresh connection %d to remain open", i+1)
		}
	}
}

// TestConnectionCleanup_ProperClosure tests that stale connections are closed properly
func TestConnectionCleanup_ProperClosure(t *testing.T) {
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

	// Add stale connection
	staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	mock := &mockClient{connected: true}
	staleConn := &PooledConnection{
		Client:     mock,
		ServerID:   "test-server",
		LastUsed:   staleTime,
		AcquiredAt: staleTime,
		InUse:      false,
		refCount:   0,
		closed:     false,
	}

	pool.mu.Lock()
	pool.connections["test-server"] = []*PooledConnection{staleConn}
	pool.mu.Unlock()

	// Wait for cleanup
	time.Sleep(1500 * time.Millisecond)

	// Verify PooledConnection.closed flag is set
	staleConn.mu.Lock()
	closed := staleConn.closed
	staleConn.mu.Unlock()

	if !closed {
		t.Error("Expected PooledConnection.closed to be true")
	}

	// Verify underlying client was closed
	if !mock.IsClosed() {
		t.Error("Expected underlying client to be closed")
	}
}

// TestConnectionCleanup_PeriodicExecution tests periodic cleanup runs
func TestConnectionCleanup_PeriodicExecution(t *testing.T) {
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

	// Add stale connections at intervals and verify they get cleaned up
	cleanupCount := 0

	for i := 0; i < 3; i++ {
		// Add stale connection
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
		currentCount := len(pool.connections["test-server"])
		pool.mu.Unlock()

		// Wait for cleanup cycle
		time.Sleep(1500 * time.Millisecond)

		// Verify connection was removed
		pool.mu.RLock()
		newCount := len(pool.connections["test-server"])
		pool.mu.RUnlock()

		if newCount >= currentCount {
			t.Errorf("Iteration %d: Expected connection count to decrease from %d, got %d", i+1, currentCount, newCount)
		} else {
			cleanupCount++
		}
	}

	if cleanupCount < 3 {
		t.Errorf("Expected 3 successful cleanup cycles, got %d", cleanupCount)
	}
}

// TestConnectionCleanup_ShutdownWaitsForCleanup tests shutdown coordination with cleanup
func TestConnectionCleanup_ShutdownWaitsForCleanup(t *testing.T) {
	pool := NewConnectionPool()

	// Register server
	config := ServerConfig{
		ID:        "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"test"},
	}
	pool.RegisterServer(config)

	// Add connection
	addMockConnection(pool, "test-server")

	// Give cleanup goroutine time to start
	time.Sleep(100 * time.Millisecond)

	// Close should signal cleanup to stop and wait for it
	start := time.Now()
	err := pool.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Close should be fast (not wait for next cleanup cycle)
	if elapsed > 2*time.Second {
		t.Errorf("Close() took too long: %v (expected < 2s)", elapsed)
	}

	// Verify pool is closed
	pool.mu.RLock()
	closed := pool.closed
	pool.mu.RUnlock()

	if !closed {
		t.Error("Expected pool to be marked as closed")
	}
}

// T064: Connection Leak Detection Tests

// TestConnectionLeakDetection_RefCountTracking tests refCount is tracked correctly
func TestConnectionLeakDetection_RefCountTracking(t *testing.T) {
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

	// Get connection - should increment refCount
	_, err := pool.Get(ctx, "test-server")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Check refCount is 1
	pool.mu.RLock()
	conn := pool.connections["test-server"][0]
	pool.mu.RUnlock()

	conn.mu.Lock()
	refCount := conn.refCount
	conn.mu.Unlock()

	if refCount != 1 {
		t.Errorf("Expected refCount=1 after Get(), got %d", refCount)
	}

	// Release connection - should decrement refCount
	if err := pool.Release("test-server"); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	conn.mu.Lock()
	refCount = conn.refCount
	conn.mu.Unlock()

	if refCount != 0 {
		t.Errorf("Expected refCount=0 after Release(), got %d", refCount)
	}
}

// TestConnectionLeakDetection_LeakWarning tests leak warning is logged
func TestConnectionLeakDetection_LeakWarning(t *testing.T) {
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

	// Add stale connection with non-zero refCount (simulating a leak)
	staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	leakedConn := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   "test-server",
		LastUsed:   staleTime,
		AcquiredAt: staleTime,
		InUse:      false,
		refCount:   1, // Leak: refCount should be 0 when not in use
	}

	pool.mu.Lock()
	pool.connections["test-server"] = []*PooledConnection{leakedConn}
	pool.mu.Unlock()

	// Wait for cleanup cycle
	time.Sleep(1500 * time.Millisecond)

	// Check leak was detected
	leaks := pool.LeakStats()
	if leaks != 1 {
		t.Errorf("Expected 1 leak detected, got %d", leaks)
	}

	// Verify connection was still closed despite leak
	leakedConn.mu.Lock()
	closed := leakedConn.closed
	leakedConn.mu.Unlock()

	if !closed {
		t.Error("Expected leaked connection to be closed")
	}
}

// TestConnectionLeakDetection_MultipleLeaks tests multiple leaks are counted
func TestConnectionLeakDetection_MultipleLeaks(t *testing.T) {
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

	// Add multiple leaked connections
	staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	for i := 0; i < 3; i++ {
		leakedConn := &PooledConnection{
			Client:     &mockClient{connected: true},
			ServerID:   "test-server",
			LastUsed:   staleTime,
			AcquiredAt: staleTime,
			InUse:      false,
			refCount:   2, // Simulated leak
		}

		pool.mu.Lock()
		pool.connections["test-server"] = append(pool.connections["test-server"], leakedConn)
		pool.mu.Unlock()

		// Wait for cleanup
		time.Sleep(1500 * time.Millisecond)
	}

	// Check all leaks were detected
	leaks := pool.LeakStats()
	if leaks != 3 {
		t.Errorf("Expected 3 leaks detected, got %d", leaks)
	}
}

// TestConnectionLeakDetection_NoLeakForValidUsage tests no leak for proper usage
func TestConnectionLeakDetection_NoLeakForValidUsage(t *testing.T) {
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

	// Add properly released connection
	staleTime := time.Now().Add(-ConnectionIdleTimeout - 1*time.Minute)
	validConn := &PooledConnection{
		Client:     &mockClient{connected: true},
		ServerID:   "test-server",
		LastUsed:   staleTime,
		AcquiredAt: staleTime,
		InUse:      false,
		refCount:   0, // Proper: refCount is 0
	}

	pool.mu.Lock()
	pool.connections["test-server"] = []*PooledConnection{validConn}
	pool.mu.Unlock()

	// Wait for cleanup
	time.Sleep(1500 * time.Millisecond)

	// Check no leak was detected
	leaks := pool.LeakStats()
	if leaks != 0 {
		t.Errorf("Expected 0 leaks detected for valid usage, got %d", leaks)
	}
}
