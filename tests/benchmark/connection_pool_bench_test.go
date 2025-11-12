package benchmark

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
)

// BenchmarkConnectionPoolGet benchmarks getting connections from the pool
func BenchmarkConnectionPoolGet(b *testing.B) {
	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	server, _ := mcpserver.NewMCPServer("test-server", "echo", []string{}, mcpserver.TransportStdio)
	server.Connection.State = mcpserver.StateConnected

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pool.Get(ctx, server)
	}

	stats := pool.GetStats()
	b.ReportMetric(stats.ReuseRate*100, "reuse_rate_%")
}

// BenchmarkConnectionPoolGetParallel benchmarks concurrent pool access
func BenchmarkConnectionPoolGetParallel(b *testing.B) {
	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	// Create multiple servers
	servers := make([]*mcpserver.MCPServer, 10)
	for i := 0; i < 10; i++ {
		server, _ := mcpserver.NewMCPServer(
			"server-"+string(rune('a'+i)),
			"echo",
			[]string{},
			mcpserver.TransportStdio,
		)
		server.Connection.State = mcpserver.StateConnected
		servers[i] = server
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			server := servers[i%len(servers)]
			_, _ = pool.Get(ctx, server)
			pool.Release(server.ID)
			i++
		}
	})

	stats := pool.GetStats()
	b.ReportMetric(stats.ReuseRate*100, "reuse_rate_%")
}

// BenchmarkConnectionPoolPreWarm benchmarks pre-warming connections
func BenchmarkConnectionPoolPreWarm(b *testing.B) {
	benchmarks := []struct {
		name        string
		serverCount int
	}{
		{"5_servers", 5},
		{"10_servers", 10},
		{"25_servers", 25},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			pool := mcpserver.NewConnectionPool()
			defer pool.Close()

			registry := mcpserver.NewRegistry()

			// Create servers
			serverIDs := make([]string, bm.serverCount)
			for i := 0; i < bm.serverCount; i++ {
				serverID := "server-" + string(rune('a'+(i%26))) + string(rune('0'+(i/26)))
				server, _ := mcpserver.NewMCPServer(serverID, "echo", []string{}, mcpserver.TransportStdio)
				_ = registry.Register(server)
				serverIDs[i] = serverID
			}

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool.PreWarm(ctx, registry, serverIDs)
			}

			stats := pool.GetStats()
			b.ReportMetric(float64(stats.PreWarmedConnections), "prewarmed")
		})
	}
}

// BenchmarkConnectionPoolCleanup benchmarks idle connection cleanup
func BenchmarkConnectionPoolCleanup(b *testing.B) {
	pool := mcpserver.NewConnectionPoolWithConfig(3, 10*time.Millisecond)
	defer pool.Close()

	ctx := context.Background()

	// Add multiple connections
	for i := 0; i < 20; i++ {
		server, _ := mcpserver.NewMCPServer(
			"server-"+string(rune('a'+i)),
			"echo",
			[]string{},
			mcpserver.TransportStdio,
		)
		server.Connection.State = mcpserver.StateConnected
		_, _ = pool.Get(ctx, server)
	}

	// Wait for cleanup to trigger
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cleanup happens in background worker
		time.Sleep(time.Millisecond)
	}

	b.ReportMetric(float64(pool.Size()), "active_connections")
}

// BenchmarkConnectionPoolUsageTracking benchmarks usage frequency tracking
func BenchmarkConnectionPoolUsageTracking(b *testing.B) {
	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	ctx := context.Background()

	// Create servers with varying usage patterns
	servers := make([]*mcpserver.MCPServer, 10)
	for i := 0; i < 10; i++ {
		server, _ := mcpserver.NewMCPServer(
			"server-"+string(rune('a'+i)),
			"echo",
			[]string{},
			mcpserver.TransportStdio,
		)
		server.Connection.State = mcpserver.StateConnected
		servers[i] = server

		// Simulate different usage frequencies
		for j := 0; j < (i+1)*5; j++ {
			_, _ = pool.Get(ctx, server)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.GetFrequentServers(5)
	}
}

// BenchmarkConnectionPoolStats benchmarks statistics calculation
func BenchmarkConnectionPoolStats(b *testing.B) {
	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	ctx := context.Background()

	// Pre-populate pool
	for i := 0; i < 100; i++ {
		server, _ := mcpserver.NewMCPServer(
			"server-"+string(rune('a'+(i%26))),
			"echo",
			[]string{},
			mcpserver.TransportStdio,
		)
		server.Connection.State = mcpserver.StateConnected
		_, _ = pool.Get(ctx, server)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.GetStats()
	}
}

// BenchmarkConnectionPoolIsPreWarmed benchmarks pre-warm status checking
func BenchmarkConnectionPoolIsPreWarmed(b *testing.B) {
	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	registry := mcpserver.NewRegistry()
	server, _ := mcpserver.NewMCPServer("test-server", "echo", []string{}, mcpserver.TransportStdio)
	_ = registry.Register(server)

	ctx := context.Background()
	_ = pool.PreWarm(ctx, registry, []string{"test-server"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.IsPreWarmed("test-server")
	}
}

// BenchmarkConnectionPoolMemoryUsage benchmarks memory consumption
func BenchmarkConnectionPoolMemoryUsage(b *testing.B) {
	var m runtime.MemStats

	// Baseline
	runtime.GC()
	runtime.ReadMemStats(&m)
	baseline := m.Alloc

	pool := mcpserver.NewConnectionPool()
	defer pool.Close()

	ctx := context.Background()

	// Add 100 connections
	for i := 0; i < 100; i++ {
		server, _ := mcpserver.NewMCPServer(
			"server-"+string(rune('a'+(i%26)))+string(rune('0'+(i/26))),
			"echo",
			[]string{},
			mcpserver.TransportStdio,
		)
		server.Connection.State = mcpserver.StateConnected
		_, _ = pool.Get(ctx, server)
	}

	// Measure memory
	runtime.GC()
	runtime.ReadMemStats(&m)
	memUsed := (m.Alloc - baseline) / 1024 // Convert to KB

	b.ReportMetric(float64(memUsed)/100, "KB_per_connection")
}

// BenchmarkConnectionPoolReuse benchmarks connection reuse scenarios
func BenchmarkConnectionPoolReuse(b *testing.B) {
	benchmarks := []struct {
		name       string
		reuseRatio float64 // Ratio of hits to total accesses
	}{
		{"low_reuse_20%", 0.2},
		{"medium_reuse_50%", 0.5},
		{"high_reuse_80%", 0.8},
		{"very_high_reuse_95%", 0.95},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			pool := mcpserver.NewConnectionPool()
			defer pool.Close()

			ctx := context.Background()

			// Create pool of servers
			commonServers := make([]*mcpserver.MCPServer, 10)
			for i := 0; i < 10; i++ {
				server, _ := mcpserver.NewMCPServer(
					"common-"+string(rune('a'+i)),
					"echo",
					[]string{},
					mcpserver.TransportStdio,
				)
				server.Connection.State = mcpserver.StateConnected
				commonServers[i] = server
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if float64(i%100)/100.0 < bm.reuseRatio {
					// Reuse existing connection
					server := commonServers[i%len(commonServers)]
					_, _ = pool.Get(ctx, server)
				} else {
					// Create new connection
					server, _ := mcpserver.NewMCPServer(
						"new-"+string(rune('a'+(i%26))),
						"echo",
						[]string{},
						mcpserver.TransportStdio,
					)
					server.Connection.State = mcpserver.StateConnected
					_, _ = pool.Get(ctx, server)
				}
			}

			stats := pool.GetStats()
			b.ReportMetric(stats.ReuseRate*100, "actual_reuse_%")
		})
	}
}
