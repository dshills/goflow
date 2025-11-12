package execution

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionCache(t *testing.T) {
	cache := NewExecutionCache()

	assert.NotNil(t, cache)
	assert.True(t, cache.IsEnabled())
	assert.Equal(t, 1000, cache.maxSize)
	assert.Equal(t, 30*time.Minute, cache.ttl)

	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.TotalSize)
}

func TestCacheGetSet(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")
	nodeType := "mcp_tool"
	inputs := map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	}
	outputs := map[string]interface{}{
		"result": "success",
		"data":   []string{"a", "b", "c"},
	}

	// First access should be a miss
	entry, found := cache.Get(nodeID, nodeType, inputs)
	assert.False(t, found)
	assert.Nil(t, entry)

	// Set the cache entry
	err := cache.Set(nodeID, nodeType, inputs, outputs)
	require.NoError(t, err)

	// Second access should be a hit
	entry, found = cache.Get(nodeID, nodeType, inputs)
	assert.True(t, found)
	require.NotNil(t, entry)
	assert.Equal(t, nodeID, entry.NodeID)
	assert.Equal(t, nodeType, entry.NodeType)
	assert.Equal(t, outputs["result"], entry.Outputs["result"])

	// Verify stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.TotalSize)
	assert.Equal(t, 0.5, stats.HitRate)
}

func TestCacheDifferentInputs(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")
	nodeType := "transform"

	inputs1 := map[string]interface{}{"param": "value1"}
	outputs1 := map[string]interface{}{"result": "output1"}

	inputs2 := map[string]interface{}{"param": "value2"}
	outputs2 := map[string]interface{}{"result": "output2"}

	// Cache both versions
	err := cache.Set(nodeID, nodeType, inputs1, outputs1)
	require.NoError(t, err)
	err = cache.Set(nodeID, nodeType, inputs2, outputs2)
	require.NoError(t, err)

	// Both should be retrievable
	entry1, found := cache.Get(nodeID, nodeType, inputs1)
	assert.True(t, found)
	assert.Equal(t, "output1", entry1.Outputs["result"])

	entry2, found := cache.Get(nodeID, nodeType, inputs2)
	assert.True(t, found)
	assert.Equal(t, "output2", entry2.Outputs["result"])

	// Stats should show 2 entries
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.TotalSize)
}

func TestCacheTTL(t *testing.T) {
	// Create cache with very short TTL
	cache := NewExecutionCacheWithConfig(100, 10*time.Millisecond)

	nodeID := types.NodeID("node-1")
	inputs := map[string]interface{}{"param": "value"}
	outputs := map[string]interface{}{"result": "output"}

	// Set cache entry
	err := cache.Set(nodeID, "mcp_tool", inputs, outputs)
	require.NoError(t, err)

	// Immediate access should succeed
	entry, found := cache.Get(nodeID, "mcp_tool", inputs)
	assert.True(t, found)
	assert.NotNil(t, entry)

	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)

	// Access after TTL should fail
	entry, found = cache.Get(nodeID, "mcp_tool", inputs)
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestCacheInvalidate(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")
	inputs := map[string]interface{}{"param": "value"}
	outputs := map[string]interface{}{"result": "output"}

	// Set cache entry
	err := cache.Set(nodeID, "transform", inputs, outputs)
	require.NoError(t, err)

	// Verify it's cached
	entry, found := cache.Get(nodeID, "transform", inputs)
	assert.True(t, found)
	assert.NotNil(t, entry)

	// Invalidate specific entry
	err = cache.Invalidate(nodeID, inputs)
	require.NoError(t, err)

	// Should no longer be cached
	entry, found = cache.Get(nodeID, "transform", inputs)
	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestCacheInvalidateNode(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")

	inputs1 := map[string]interface{}{"param": "value1"}
	inputs2 := map[string]interface{}{"param": "value2"}

	// Cache multiple entries for same node
	err := cache.Set(nodeID, "mcp_tool", inputs1, map[string]interface{}{"result": "output1"})
	require.NoError(t, err)
	err = cache.Set(nodeID, "mcp_tool", inputs2, map[string]interface{}{"result": "output2"})
	require.NoError(t, err)

	// Verify both are cached
	_, found := cache.Get(nodeID, "mcp_tool", inputs1)
	assert.True(t, found)
	_, found = cache.Get(nodeID, "mcp_tool", inputs2)
	assert.True(t, found)

	// Invalidate all entries for node
	cache.InvalidateNode(nodeID)

	// Both should be gone
	_, found = cache.Get(nodeID, "mcp_tool", inputs1)
	assert.False(t, found)
	_, found = cache.Get(nodeID, "mcp_tool", inputs2)
	assert.False(t, found)
}

func TestCacheClear(t *testing.T) {
	cache := NewExecutionCache()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		nodeID := types.NodeID("node-" + string(rune('1'+i)))
		inputs := map[string]interface{}{"index": i}
		outputs := map[string]interface{}{"result": i}
		err := cache.Set(nodeID, "transform", inputs, outputs)
		require.NoError(t, err)
	}

	// Verify cache has entries
	stats := cache.GetStats()
	assert.Equal(t, int64(5), stats.TotalSize)

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.TotalSize)
}

func TestCacheEviction(t *testing.T) {
	// Create cache with small size
	cache := NewExecutionCacheWithConfig(3, 30*time.Minute)

	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		nodeID := types.NodeID("node-" + string(rune('1'+i)))
		inputs := map[string]interface{}{"index": i}
		outputs := map[string]interface{}{"result": i}
		err := cache.Set(nodeID, "transform", inputs, outputs)
		require.NoError(t, err)

		// Add small delay to ensure different access times
		time.Sleep(time.Millisecond)
	}

	// Access first entry to make it most recent
	firstNodeID := types.NodeID("node-1")
	firstInputs := map[string]interface{}{"index": 0}
	_, found := cache.Get(firstNodeID, "transform", firstInputs)
	assert.True(t, found)

	// Add one more entry, triggering eviction
	newNodeID := types.NodeID("node-4")
	newInputs := map[string]interface{}{"index": 4}
	newOutputs := map[string]interface{}{"result": 4}
	err := cache.Set(newNodeID, "transform", newInputs, newOutputs)
	require.NoError(t, err)

	// Cache should still be at max size
	stats := cache.GetStats()
	assert.Equal(t, int64(3), stats.TotalSize)
	assert.Equal(t, int64(1), stats.Evictions)

	// First entry should still be present (most recent)
	_, found = cache.Get(firstNodeID, "transform", firstInputs)
	assert.True(t, found)

	// Second entry should have been evicted (oldest)
	secondNodeID := types.NodeID("node-2")
	secondInputs := map[string]interface{}{"index": 1}
	_, found = cache.Get(secondNodeID, "transform", secondInputs)
	assert.False(t, found)
}

func TestCacheCleanExpired(t *testing.T) {
	// Create cache with short TTL
	cache := NewExecutionCacheWithConfig(100, 10*time.Millisecond)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		nodeID := types.NodeID("node-" + string(rune('1'+i)))
		inputs := map[string]interface{}{"index": i}
		outputs := map[string]interface{}{"result": i}
		err := cache.Set(nodeID, "transform", inputs, outputs)
		require.NoError(t, err)
	}

	// Verify cache has entries
	stats := cache.GetStats()
	assert.Equal(t, int64(5), stats.TotalSize)

	// Wait for TTL to expire
	time.Sleep(15 * time.Millisecond)

	// Clean expired entries
	removed := cache.CleanExpired()
	assert.Equal(t, 5, removed)

	// Cache should be empty
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.TotalSize)
	assert.Equal(t, int64(5), stats.Evictions)
}

func TestCacheShouldCache(t *testing.T) {
	cache := NewExecutionCache()

	tests := []struct {
		name     string
		nodeExec *execution.NodeExecution
		want     bool
	}{
		{
			name: "cache mcp_tool success",
			nodeExec: &execution.NodeExecution{
				NodeType: "mcp_tool",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"result": "success"},
			},
			want: true,
		},
		{
			name: "cache transform success",
			nodeExec: &execution.NodeExecution{
				NodeType: "transform",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"result": "transformed"},
			},
			want: true,
		},
		{
			name: "cache condition success",
			nodeExec: &execution.NodeExecution{
				NodeType: "condition",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"result": true},
			},
			want: true,
		},
		{
			name: "don't cache failed execution",
			nodeExec: &execution.NodeExecution{
				NodeType: "mcp_tool",
				Status:   execution.NodeStatusFailed,
				Outputs:  map[string]interface{}{"error": "failed"},
			},
			want: false,
		},
		{
			name: "don't cache start node",
			nodeExec: &execution.NodeExecution{
				NodeType: "start",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"started": true},
			},
			want: false,
		},
		{
			name: "don't cache end node",
			nodeExec: &execution.NodeExecution{
				NodeType: "end",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"completed": true},
			},
			want: false,
		},
		{
			name: "don't cache parallel node",
			nodeExec: &execution.NodeExecution{
				NodeType: "parallel",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"branches": []interface{}{}},
			},
			want: false,
		},
		{
			name: "don't cache loop node",
			nodeExec: &execution.NodeExecution{
				NodeType: "loop",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{"iterations": []interface{}{}},
			},
			want: false,
		},
		{
			name: "don't cache empty outputs",
			nodeExec: &execution.NodeExecution{
				NodeType: "mcp_tool",
				Status:   execution.NodeStatusCompleted,
				Outputs:  map[string]interface{}{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.ShouldCache(tt.nodeExec)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCacheDisable(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")
	inputs := map[string]interface{}{"param": "value"}
	outputs := map[string]interface{}{"result": "output"}

	// Cache should be enabled initially
	assert.True(t, cache.IsEnabled())

	// Set entry when enabled
	err := cache.Set(nodeID, "mcp_tool", inputs, outputs)
	require.NoError(t, err)

	// Disable cache
	cache.Disable()
	assert.False(t, cache.IsEnabled())

	// Get should return false when disabled
	entry, found := cache.Get(nodeID, "mcp_tool", inputs)
	assert.False(t, found)
	assert.Nil(t, entry)

	// Set should be no-op when disabled
	err = cache.Set(nodeID, "mcp_tool", inputs, outputs)
	require.NoError(t, err)

	// Re-enable cache
	cache.Enable()
	assert.True(t, cache.IsEnabled())

	// Should be able to get previously cached entry
	entry, found = cache.Get(nodeID, "mcp_tool", inputs)
	assert.True(t, found)
	assert.NotNil(t, entry)
}

func TestCacheDeepCopy(t *testing.T) {
	cache := NewExecutionCache()

	nodeID := types.NodeID("node-1")
	inputs := map[string]interface{}{"param": "value"}
	outputs := map[string]interface{}{
		"result": "output",
		"slice":  []string{"a", "b", "c"},
	}

	// Set entry
	err := cache.Set(nodeID, "transform", inputs, outputs)
	require.NoError(t, err)

	// Get entry
	entry, found := cache.Get(nodeID, "transform", inputs)
	assert.True(t, found)

	// Modify returned outputs
	if slice, ok := entry.Outputs["slice"].([]interface{}); ok {
		slice[0] = "modified"
	}

	// Get entry again and verify original is unchanged
	entry2, found := cache.Get(nodeID, "transform", inputs)
	assert.True(t, found)
	if slice, ok := entry2.Outputs["slice"].([]interface{}); ok {
		assert.Equal(t, "a", slice[0])
	}
}
