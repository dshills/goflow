package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/mcpserver"
)

// This example demonstrates the performance optimizations in Phase 9:
// 1. Workflow caching to skip unchanged node executions
// 2. Connection pooling with pre-warming for frequently used servers

func main() {
	fmt.Println("GoFlow Performance Optimization Example")
	fmt.Println("========================================")

	// Example 1: Workflow Caching
	demonstrateWorkflowCaching()

	// Example 2: Connection Pooling
	demonstrateConnectionPooling()
}

func demonstrateWorkflowCaching() {
	fmt.Println("\n1. Workflow Caching")
	fmt.Println("-------------------")

	// Create execution cache
	cache := execution.NewExecutionCache()
	fmt.Printf("Created cache with TTL: 30 minutes, Max size: 1000\n")

	// Simulate node execution results
	nodeID := types.NodeID("transform-1")
	nodeType := "transform"
	inputs := map[string]interface{}{
		"data":   "example input",
		"format": "json",
	}
	outputs := map[string]interface{}{
		"result":       "processed data",
		"timestamp":    time.Now(),
		"rowsAffected": 42,
	}

	// First execution - cache miss
	_, found := cache.Get(nodeID, nodeType, inputs)
	fmt.Printf("First lookup: found=%v (expected cache miss)\n", found)

	// Store the result
	err := cache.Set(nodeID, nodeType, inputs, outputs)
	if err != nil {
		fmt.Printf("Error storing in cache: %v\n", err)
		return
	}
	fmt.Println("Stored execution result in cache")

	// Second execution - cache hit
	entry, found := cache.Get(nodeID, nodeType, inputs)
	fmt.Printf("Second lookup: found=%v (cache hit!)\n", found)
	if found {
		fmt.Printf("Cached result: %v\n", entry.Outputs["result"])
	}

	// Get cache statistics
	stats := cache.GetStats()
	fmt.Printf("\nCache Statistics:\n")
	fmt.Printf("  Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
	fmt.Printf("  Hit Rate: %.2f%%\n", stats.HitRate*100)
	fmt.Printf("  Total Size: %d entries\n", stats.TotalSize)

	// Demonstrate cache invalidation
	cache.InvalidateNode(nodeID)
	fmt.Println("\nInvalidated cache for node:", nodeID)

	_, found = cache.Get(nodeID, nodeType, inputs)
	fmt.Printf("After invalidation: found=%v\n", found)
}

func demonstrateConnectionPooling() {
	fmt.Println("\n2. Connection Pooling")
	fmt.Println("---------------------")

	// Create connection pool
	pool := mcpserver.NewConnectionPool()
	defer func() {
		if err := pool.Close(); err != nil {
			fmt.Printf("Failed to close connection pool: %v\n", err)
		}
	}()
	fmt.Println("Created connection pool")

	// Create registry and servers
	registry := mcpserver.NewRegistry()

	// Register some example servers
	servers := []struct {
		id      string
		command string
	}{
		{"filesystem", "mcp-server-filesystem"},
		{"github", "mcp-server-github"},
		{"database", "mcp-server-postgres"},
	}

	for _, s := range servers {
		server, err := mcpserver.NewMCPServer(s.id, s.command, []string{}, mcpserver.TransportStdio)
		if err != nil {
			fmt.Printf("Error creating server %s: %v\n", s.id, err)
			continue
		}
		err = registry.Register(server)
		if err != nil {
			fmt.Printf("Error registering server %s: %v\n", s.id, err)
			continue
		}
	}

	fmt.Printf("Registered %d servers\n", len(servers))

	// Pre-warm frequently used servers
	serverIDs := []string{"filesystem", "database"}
	ctx := context.Background()
	err := pool.PreWarm(ctx, registry, serverIDs)
	if err != nil {
		fmt.Printf("Pre-warming completed with errors: %v\n", err)
	} else {
		fmt.Printf("Pre-warmed %d servers\n", len(serverIDs))
	}

	// Check which servers are pre-warmed
	for _, id := range serverIDs {
		isPreWarmed := pool.IsPreWarmed(id)
		fmt.Printf("  %s: pre-warmed=%v\n", id, isPreWarmed)
	}

	// Simulate getting connections
	fmt.Println("\nSimulating connection usage:")
	server, _ := registry.Get("filesystem")
	for i := 0; i < 5; i++ {
		conn, err := pool.Get(ctx, server)
		if err != nil {
			fmt.Printf("Error getting connection: %v\n", err)
			continue
		}
		fmt.Printf("  Request %d: UsageCount=%d\n", i+1, conn.UsageCount)
		pool.Release(server.ID)
	}

	// Get pool statistics
	stats := pool.GetStats()
	fmt.Printf("\nPool Statistics:\n")
	fmt.Printf("  Active Connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Pre-warmed: %d\n", stats.PreWarmedConnections)
	fmt.Printf("  Hits: %d, Misses: %d\n", stats.UsageHits, stats.UsageMisses)
	fmt.Printf("  Reuse Rate: %.2f%%\n", stats.ReuseRate*100)

	// Get frequently used servers
	frequentServers := pool.GetFrequentServers(3)
	fmt.Println("\nMost frequently used servers:")
	for i, id := range frequentServers {
		fmt.Printf("  %d. %s\n", i+1, id)
	}
}

// Example output:
//
// GoFlow Performance Optimization Example
// ========================================
//
// 1. Workflow Caching
// -------------------
// Created cache with TTL: 30 minutes, Max size: 1000
// First lookup: found=false (expected cache miss)
// Stored execution result in cache
// Second lookup: found=true (cache hit!)
// Cached result: processed data
//
// Cache Statistics:
//   Hits: 1, Misses: 1
//   Hit Rate: 50.00%
//   Total Size: 1 entries
//
// Invalidated cache for node: transform-1
// After invalidation: found=false
//
// 2. Connection Pooling
// ---------------------
// Created connection pool
// Registered 3 servers
// Pre-warmed 2 servers
//   filesystem: pre-warmed=true
//   database: pre-warmed=true
//
// Simulating connection usage:
//   Request 1: UsageCount=1
//   Request 2: UsageCount=2
//   Request 3: UsageCount=3
//   Request 4: UsageCount=4
//   Request 5: UsageCount=5
//
// Pool Statistics:
//   Active Connections: 2
//   Pre-warmed: 2
//   Hits: 4, Misses: 1
//   Reuse Rate: 80.00%
//
// Most frequently used servers:
//   1. filesystem
