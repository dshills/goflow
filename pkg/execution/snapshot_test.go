package execution

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSnapshotManager(t *testing.T) {
	tests := []struct {
		name                string
		maxSnapshotsPerNode int
		maxAge              time.Duration
	}{
		{
			name:                "unlimited snapshots, infinite retention",
			maxSnapshotsPerNode: 0,
			maxAge:              0,
		},
		{
			name:                "limited to 10 snapshots per node",
			maxSnapshotsPerNode: 10,
			maxAge:              0,
		},
		{
			name:                "1 hour retention",
			maxSnapshotsPerNode: 0,
			maxAge:              1 * time.Hour,
		},
		{
			name:                "combined limits: 100 snapshots or 30 minutes",
			maxSnapshotsPerNode: 100,
			maxAge:              30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSnapshotManager(tt.maxSnapshotsPerNode, tt.maxAge)
			require.NotNil(t, sm)
			assert.Equal(t, tt.maxSnapshotsPerNode, sm.maxSnapshotsPerNode)
			assert.Equal(t, tt.maxAge, sm.maxAge)
			assert.NotNil(t, sm.snapshots)
			assert.Equal(t, 0, sm.GetTotalSnapshotCount())
		})
	}
}

func TestCaptureSnapshot(t *testing.T) {
	tests := []struct {
		name      string
		nodeID    types.NodeID
		variables map[string]interface{}
		wantErr   bool
	}{
		{
			name:   "simple variables",
			nodeID: "node1",
			variables: map[string]interface{}{
				"count":   float64(42), // JSON marshaling converts numbers to float64
				"name":    "test",
				"enabled": true,
			},
			wantErr: false,
		},
		{
			name:   "nested structures",
			nodeID: "node2",
			variables: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   float64(123), // JSON marshaling converts numbers to float64
					"name": "Alice",
					"tags": []interface{}{"admin", "user"},
				},
				"settings": map[string]interface{}{
					"theme": "dark",
					"lang":  "en",
				},
			},
			wantErr: false,
		},
		{
			name:   "arrays and slices",
			nodeID: "node3",
			variables: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2), float64(3), float64(4), float64(5)}, // JSON marshaling
				// Note: []string gets converted to []interface{} during JSON round-trip
				"strings": []interface{}{"a", "b", "c"},
				"mixed":   []interface{}{float64(1), "two", true, nil},
			},
			wantErr: false,
		},
		{
			name:      "nil variables",
			nodeID:    "node4",
			variables: nil,
			wantErr:   false,
		},
		{
			name:      "empty variables",
			nodeID:    "node5",
			variables: map[string]interface{}{},
			wantErr:   false,
		},
		{
			name:      "empty node ID should error",
			nodeID:    "",
			variables: map[string]interface{}{"test": "value"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSnapshotManager(0, 0)
			err := sm.CaptureSnapshot(tt.nodeID, tt.variables)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify snapshot was captured
			snapshot := sm.GetLatestSnapshot(tt.nodeID)
			require.NotNil(t, snapshot)
			assert.Equal(t, tt.nodeID, snapshot.NodeID)
			assert.False(t, snapshot.Timestamp.IsZero())

			// Verify deep copy (variables match)
			if tt.variables != nil {
				assert.Equal(t, len(tt.variables), len(snapshot.Variables))
				for key, expectedValue := range tt.variables {
					actualValue, exists := snapshot.Variables[key]
					assert.True(t, exists, "variable %s should exist", key)
					assert.Equal(t, expectedValue, actualValue, "variable %s value mismatch", key)
				}
			} else {
				assert.Nil(t, snapshot.Variables)
			}
		})
	}
}

func TestSnapshotImmutability(t *testing.T) {
	sm := NewSnapshotManager(0, 0)
	nodeID := types.NodeID("immutable-test")

	// Original variables
	originalVars := map[string]interface{}{
		"count": 1,
		"data": map[string]interface{}{
			"value": "original",
		},
	}

	// Capture snapshot
	err := sm.CaptureSnapshot(nodeID, originalVars)
	require.NoError(t, err)

	// Modify original variables
	originalVars["count"] = 999
	originalVars["data"].(map[string]interface{})["value"] = "modified"
	originalVars["new_key"] = "new_value"

	// Get snapshot and verify it wasn't affected
	snapshot := sm.GetLatestSnapshot(nodeID)
	require.NotNil(t, snapshot)

	assert.Equal(t, float64(1), snapshot.Variables["count"], "snapshot should not be affected by original mutation")

	dataMap, ok := snapshot.Variables["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "original", dataMap["value"], "nested values should not be affected")

	_, exists := snapshot.Variables["new_key"]
	assert.False(t, exists, "new keys should not appear in snapshot")
}

func TestGetLatestSnapshot(t *testing.T) {
	sm := NewSnapshotManager(0, 0)
	nodeID := types.NodeID("test-node")

	// No snapshot initially
	snapshot := sm.GetLatestSnapshot(nodeID)
	assert.Nil(t, snapshot)

	// Capture multiple snapshots
	for i := 1; i <= 5; i++ {
		vars := map[string]interface{}{
			"counter": i,
		}
		err := sm.CaptureSnapshot(nodeID, vars)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Get latest should return the last one
	latest := sm.GetLatestSnapshot(nodeID)
	require.NotNil(t, latest)
	assert.Equal(t, float64(5), latest.Variables["counter"])
}

func TestGetAllSnapshots(t *testing.T) {
	sm := NewSnapshotManager(0, 0)
	nodeID := types.NodeID("test-node")

	// No snapshots initially
	snapshots := sm.GetAllSnapshots(nodeID)
	assert.Nil(t, snapshots)

	// Capture 3 snapshots
	for i := 1; i <= 3; i++ {
		vars := map[string]interface{}{
			"value": i,
		}
		err := sm.CaptureSnapshot(nodeID, vars)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// Get all snapshots
	snapshots = sm.GetAllSnapshots(nodeID)
	require.Len(t, snapshots, 3)

	// Verify order (oldest to newest)
	for i := 0; i < 3; i++ {
		assert.Equal(t, float64(i+1), snapshots[i].Variables["value"])
	}

	// Verify returned copy is independent
	snapshots[0].Variables["value"] = 999
	originalSnapshots := sm.GetAllSnapshots(nodeID)
	assert.Equal(t, float64(1), originalSnapshots[0].Variables["value"], "original should not be affected")
}

func TestRetentionPolicy_MaxSnapshots(t *testing.T) {
	sm := NewSnapshotManager(3, 0) // Keep only 3 most recent snapshots
	nodeID := types.NodeID("test-node")

	// Capture 10 snapshots
	for i := 1; i <= 10; i++ {
		vars := map[string]interface{}{
			"iteration": i,
		}
		err := sm.CaptureSnapshot(nodeID, vars)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// Should only keep the last 3
	snapshots := sm.GetAllSnapshots(nodeID)
	require.Len(t, snapshots, 3)

	// Verify we kept snapshots 8, 9, 10
	assert.Equal(t, float64(8), snapshots[0].Variables["iteration"])
	assert.Equal(t, float64(9), snapshots[1].Variables["iteration"])
	assert.Equal(t, float64(10), snapshots[2].Variables["iteration"])
}

func TestRetentionPolicy_MaxAge(t *testing.T) {
	sm := NewSnapshotManager(0, 100*time.Millisecond) // Keep only snapshots from last 100ms
	nodeID := types.NodeID("test-node")

	// Capture first snapshot
	err := sm.CaptureSnapshot(nodeID, map[string]interface{}{"seq": 1})
	require.NoError(t, err)

	// Wait for it to age
	time.Sleep(150 * time.Millisecond)

	// Capture second snapshot (should remove first due to age)
	err = sm.CaptureSnapshot(nodeID, map[string]interface{}{"seq": 2})
	require.NoError(t, err)

	// Should only have the recent one
	snapshots := sm.GetAllSnapshots(nodeID)
	require.Len(t, snapshots, 1)
	assert.Equal(t, float64(2), snapshots[0].Variables["seq"])
}

func TestRetentionPolicy_Combined(t *testing.T) {
	sm := NewSnapshotManager(5, 200*time.Millisecond)
	nodeID := types.NodeID("test-node")

	// Capture 10 snapshots quickly
	for i := 1; i <= 10; i++ {
		err := sm.CaptureSnapshot(nodeID, map[string]interface{}{"seq": i})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Should be limited to 5 most recent due to count limit
	snapshots := sm.GetAllSnapshots(nodeID)
	assert.LessOrEqual(t, len(snapshots), 5)

	// Wait for all to age out
	time.Sleep(250 * time.Millisecond)

	// Add a new snapshot
	err := sm.CaptureSnapshot(nodeID, map[string]interface{}{"seq": 11})
	require.NoError(t, err)

	// Should only have the latest one (age limit removed old ones)
	snapshots = sm.GetAllSnapshots(nodeID)
	require.Len(t, snapshots, 1)
	assert.Equal(t, float64(11), snapshots[0].Variables["seq"])
}

func TestMultipleNodes(t *testing.T) {
	sm := NewSnapshotManager(0, 0)

	// Capture snapshots for different nodes
	nodes := []types.NodeID{"node1", "node2", "node3"}
	for _, nodeID := range nodes {
		for i := 1; i <= 3; i++ {
			vars := map[string]interface{}{
				"node":  string(nodeID),
				"count": i,
			}
			err := sm.CaptureSnapshot(nodeID, vars)
			require.NoError(t, err)
		}
	}

	// Verify each node has its own snapshots
	for _, nodeID := range nodes {
		snapshots := sm.GetAllSnapshots(nodeID)
		require.Len(t, snapshots, 3)
		assert.Equal(t, string(nodeID), snapshots[0].Variables["node"])
	}

	// Verify total count
	assert.Equal(t, 9, sm.GetTotalSnapshotCount())
}

func TestGetSnapshotCount(t *testing.T) {
	sm := NewSnapshotManager(0, 0)
	nodeID := types.NodeID("test-node")

	assert.Equal(t, 0, sm.GetSnapshotCount(nodeID))

	for i := 1; i <= 5; i++ {
		err := sm.CaptureSnapshot(nodeID, map[string]interface{}{"i": i})
		require.NoError(t, err)
		assert.Equal(t, i, sm.GetSnapshotCount(nodeID))
	}
}

func TestClear(t *testing.T) {
	sm := NewSnapshotManager(0, 0)

	// Add snapshots for multiple nodes
	for i := 1; i <= 3; i++ {
		nodeID := types.NodeID("node" + string(rune('0'+i)))
		err := sm.CaptureSnapshot(nodeID, map[string]interface{}{"value": i})
		require.NoError(t, err)
	}

	assert.Equal(t, 3, sm.GetTotalSnapshotCount())

	// Clear all
	sm.Clear()

	assert.Equal(t, 0, sm.GetTotalSnapshotCount())
	assert.Nil(t, sm.GetLatestSnapshot("node1"))
	assert.Nil(t, sm.GetLatestSnapshot("node2"))
	assert.Nil(t, sm.GetLatestSnapshot("node3"))
}

func TestClearNode(t *testing.T) {
	sm := NewSnapshotManager(0, 0)

	// Add snapshots for two nodes
	err := sm.CaptureSnapshot("node1", map[string]interface{}{"a": 1})
	require.NoError(t, err)
	err = sm.CaptureSnapshot("node2", map[string]interface{}{"b": 2})
	require.NoError(t, err)

	assert.Equal(t, 2, sm.GetTotalSnapshotCount())

	// Clear only node1
	sm.ClearNode("node1")

	assert.Equal(t, 1, sm.GetTotalSnapshotCount())
	assert.Nil(t, sm.GetLatestSnapshot("node1"))
	assert.NotNil(t, sm.GetLatestSnapshot("node2"))
}

func TestGetMemoryStats(t *testing.T) {
	sm := NewSnapshotManager(0, 0)

	// Initially empty
	stats := sm.GetMemoryStats()
	assert.Equal(t, 0, stats.TotalSnapshots)
	assert.Equal(t, 0, stats.NodeCount)
	assert.Equal(t, 0.0, stats.AverageSnapshotsPerNode)
	assert.Equal(t, int64(0), stats.EstimatedMemoryBytes)

	// Add snapshots for 2 nodes
	for i := 1; i <= 3; i++ {
		err := sm.CaptureSnapshot("node1", map[string]interface{}{
			"var1": i,
			"var2": i * 2,
		})
		require.NoError(t, err)
	}

	for i := 1; i <= 2; i++ {
		err := sm.CaptureSnapshot("node2", map[string]interface{}{
			"var1": i,
		})
		require.NoError(t, err)
	}

	stats = sm.GetMemoryStats()
	assert.Equal(t, 5, stats.TotalSnapshots)
	assert.Equal(t, 2, stats.NodeCount)
	assert.Equal(t, 2.5, stats.AverageSnapshotsPerNode)
	assert.Greater(t, stats.EstimatedMemoryBytes, int64(0))

	t.Logf("Memory stats: %+v", stats)
}

func TestDeepCopyVariables(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "nil map",
			variables: nil,
			wantErr:   false,
		},
		{
			name:      "empty map",
			variables: map[string]interface{}{},
			wantErr:   false,
		},
		{
			name: "simple types",
			variables: map[string]interface{}{
				"int":    float64(42), // JSON converts to float64
				"float":  3.14,
				"string": "test",
				"bool":   true,
				"nil":    nil,
			},
			wantErr: false,
		},
		{
			name: "nested structures",
			variables: map[string]interface{}{
				"nested": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "arrays and slices",
			variables: map[string]interface{}{
				"array": []interface{}{float64(1), float64(2), float64(3)}, // JSON converts to float64
				"nested_array": []interface{}{
					[]interface{}{float64(1), float64(2)},
					[]interface{}{float64(3), float64(4)},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copy, err := deepCopyVariables(tt.variables)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.variables == nil {
				assert.Nil(t, copy)
				return
			}

			// Verify copy matches original
			assert.Equal(t, tt.variables, copy)

			// Verify it's a real copy (modify original, check copy unchanged)
			if len(tt.variables) > 0 {
				tt.variables["new_key"] = "new_value"
				_, exists := copy["new_key"]
				assert.False(t, exists, "copy should not be affected by original modification")
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	sm := NewSnapshotManager(100, 0)
	nodeID := types.NodeID("concurrent-test")

	// Run concurrent captures
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(iteration int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				vars := map[string]interface{}{
					"goroutine": iteration,
					"iteration": j,
				}
				err := sm.CaptureSnapshot(nodeID, vars)
				assert.NoError(t, err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all snapshots were captured (should be 100 total)
	count := sm.GetSnapshotCount(nodeID)
	assert.Equal(t, 100, count, "all 100 snapshots should be captured without data races")
}

// Benchmark deep copy performance
func BenchmarkDeepCopyVariables_Simple(b *testing.B) {
	vars := map[string]interface{}{
		"count":   42,
		"name":    "test",
		"enabled": true,
		"value":   3.14,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepCopyVariables(vars)
	}
}

func BenchmarkDeepCopyVariables_Complex(b *testing.B) {
	vars := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "Alice",
			"profile": map[string]interface{}{
				"email": "alice@example.com",
				"tags":  []interface{}{"admin", "user", "developer"},
			},
		},
		"items": []interface{}{
			map[string]interface{}{"id": 1, "name": "item1"},
			map[string]interface{}{"id": 2, "name": "item2"},
			map[string]interface{}{"id": 3, "name": "item3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = deepCopyVariables(vars)
	}
}

func BenchmarkCaptureSnapshot(b *testing.B) {
	sm := NewSnapshotManager(1000, 0)
	vars := map[string]interface{}{
		"count": 42,
		"data": map[string]interface{}{
			"nested": "value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := types.NodeID("bench-node")
		_ = sm.CaptureSnapshot(nodeID, vars)
	}
}
