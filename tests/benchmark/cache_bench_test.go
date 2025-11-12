package benchmark

import (
	"runtime"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	execpkg "github.com/dshills/goflow/pkg/execution"
)

// BenchmarkCacheSet benchmarks cache write performance
func BenchmarkCacheSet(b *testing.B) {
	cache := execpkg.NewExecutionCache()

	nodeID := types.NodeID("test-node")
	inputs := map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	}
	outputs := map[string]interface{}{
		"result": "success",
		"data":   []string{"a", "b", "c"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Set(nodeID, "mcp_tool", inputs, outputs)
	}
}

// BenchmarkCacheGet benchmarks cache read performance
func BenchmarkCacheGet(b *testing.B) {
	cache := execpkg.NewExecutionCache()

	nodeID := types.NodeID("test-node")
	inputs := map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	}
	outputs := map[string]interface{}{
		"result": "success",
	}

	// Pre-populate cache
	_ = cache.Set(nodeID, "mcp_tool", inputs, outputs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(nodeID, "mcp_tool", inputs)
	}
}

// BenchmarkCacheHitRate benchmarks cache hit rate with realistic usage
func BenchmarkCacheHitRate(b *testing.B) {
	cache := execpkg.NewExecutionCache()

	// Pre-populate with common entries
	commonInputs := []map[string]interface{}{
		{"param": "value1"},
		{"param": "value2"},
		{"param": "value3"},
	}

	for i, inputs := range commonInputs {
		nodeID := types.NodeID("node-" + string(rune('a'+i)))
		outputs := map[string]interface{}{"result": i}
		_ = cache.Set(nodeID, "transform", inputs, outputs)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate realistic access pattern (80% hits, 20% misses)
		idx := i % 5
		if idx < 3 {
			// Hit
			nodeID := types.NodeID("node-" + string(rune('a'+idx)))
			_, _ = cache.Get(nodeID, "transform", commonInputs[idx])
		} else {
			// Miss
			nodeID := types.NodeID("node-new")
			newInputs := map[string]interface{}{"param": "new-value"}
			_, _ = cache.Get(nodeID, "transform", newInputs)
		}
	}

	stats := cache.GetStats()
	b.ReportMetric(stats.HitRate*100, "hit_rate_%")
}

// BenchmarkCacheEviction benchmarks cache eviction performance
func BenchmarkCacheEviction(b *testing.B) {
	// Create cache with small size to trigger evictions
	cache := execpkg.NewExecutionCacheWithConfig(100, 30*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := types.NodeID("node-" + string(rune('a'+(i%26))))
		inputs := map[string]interface{}{"index": i}
		outputs := map[string]interface{}{"result": i}
		_ = cache.Set(nodeID, "transform", inputs, outputs)
	}

	stats := cache.GetStats()
	b.ReportMetric(float64(stats.Evictions), "evictions")
	b.ReportMetric(float64(stats.TotalSize), "final_size")
}

// BenchmarkCacheCleanExpired benchmarks expired entry cleanup performance
func BenchmarkCacheCleanExpired(b *testing.B) {
	benchmarks := []struct {
		name       string
		entryCount int
	}{
		{"100_entries", 100},
		{"500_entries", 500},
		{"1000_entries", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cache := execpkg.NewExecutionCacheWithConfig(2000, 1*time.Millisecond)

			// Pre-populate cache
			for i := 0; i < bm.entryCount; i++ {
				nodeID := types.NodeID("node-" + string(rune('a'+(i%26))))
				inputs := map[string]interface{}{"index": i}
				outputs := map[string]interface{}{"result": i}
				_ = cache.Set(nodeID, "transform", inputs, outputs)
			}

			// Wait for entries to expire
			time.Sleep(5 * time.Millisecond)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				removed := cache.CleanExpired()
				if i == 0 {
					b.ReportMetric(float64(removed), "removed")
				}
			}
		})
	}
}

// BenchmarkCacheShouldCache benchmarks cache eligibility checking
func BenchmarkCacheShouldCache(b *testing.B) {
	cache := execpkg.NewExecutionCache()

	nodeExec := &execution.NodeExecution{
		NodeType: "mcp_tool",
		Status:   execution.NodeStatusCompleted,
		Outputs:  map[string]interface{}{"result": "success"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.ShouldCache(nodeExec)
	}
}

// BenchmarkCacheParallel benchmarks concurrent cache access
func BenchmarkCacheParallel(b *testing.B) {
	cache := execpkg.NewExecutionCache()

	// Pre-populate with some entries
	for i := 0; i < 10; i++ {
		nodeID := types.NodeID("node-" + string(rune('a'+i)))
		inputs := map[string]interface{}{"param": i}
		outputs := map[string]interface{}{"result": i}
		_ = cache.Set(nodeID, "transform", inputs, outputs)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			idx := i % 15 // Mix of hits and misses
			nodeID := types.NodeID("node-" + string(rune('a'+(idx%10))))
			inputs := map[string]interface{}{"param": idx}

			if idx < 10 {
				// Get (hit)
				_, _ = cache.Get(nodeID, "transform", inputs)
			} else {
				// Set (new entry)
				outputs := map[string]interface{}{"result": idx}
				_ = cache.Set(nodeID, "transform", inputs, outputs)
			}
			i++
		}
	})

	stats := cache.GetStats()
	b.ReportMetric(stats.HitRate*100, "hit_rate_%")
}

// BenchmarkCacheInputHashing benchmarks input hash computation
func BenchmarkCacheInputHashing(b *testing.B) {
	benchmarks := []struct {
		name   string
		inputs map[string]interface{}
	}{
		{
			"simple",
			map[string]interface{}{
				"param1": "value1",
				"param2": 42,
			},
		},
		{
			"complex",
			map[string]interface{}{
				"nested": map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": []interface{}{1, 2, 3, 4, 5},
					},
				},
				"array": []string{"a", "b", "c", "d", "e"},
			},
		},
		{
			"large",
			map[string]interface{}{
				"data": make([]byte, 1024), // 1KB of data
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cache := execpkg.NewExecutionCache()
			nodeID := types.NodeID("test-node")
			outputs := map[string]interface{}{"result": "success"}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cache.Set(nodeID, "transform", bm.inputs, outputs)
			}
		})
	}
}

// BenchmarkCacheMemoryUsage benchmarks memory consumption
func BenchmarkCacheMemoryUsage(b *testing.B) {
	var m runtime.MemStats

	// Baseline
	runtime.GC()
	runtime.ReadMemStats(&m)
	baseline := m.Alloc

	cache := execpkg.NewExecutionCacheWithConfig(10000, 30*time.Minute)

	// Add 1000 entries
	for i := 0; i < 1000; i++ {
		nodeID := types.NodeID("node-" + string(rune('a'+(i%26))))
		inputs := map[string]interface{}{
			"param1": i,
			"param2": "value-" + string(rune('a'+(i%26))),
		}
		outputs := map[string]interface{}{
			"result": i,
			"data":   []string{"a", "b", "c"},
		}
		_ = cache.Set(nodeID, "transform", inputs, outputs)
	}

	// Measure memory
	runtime.GC()
	runtime.ReadMemStats(&m)
	memUsed := (m.Alloc - baseline) / 1024 // Convert to KB

	b.ReportMetric(float64(memUsed)/1000, "cache_1000_entries_KB")
}
