package execution

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// NodeVariableSnapshot represents a complete snapshot of all variable values
// at the time a specific node was executed. Unlike VariableSnapshot which tracks
// individual variable changes, this captures the entire variable state for debugging.
type NodeVariableSnapshot struct {
	// NodeID identifies which node this snapshot was taken for
	NodeID types.NodeID
	// Timestamp records when the snapshot was captured
	Timestamp time.Time
	// Variables contains a deep copy of all variables at this point in time
	Variables map[string]interface{}
}

// SnapshotManager manages the storage and retrieval of variable snapshots
// with configurable retention policies to manage memory usage.
type SnapshotManager struct {
	// snapshots stores the captured snapshots indexed by NodeID
	snapshots map[types.NodeID][]NodeVariableSnapshot
	// maxSnapshotsPerNode limits how many snapshots to keep per node (0 = unlimited)
	maxSnapshotsPerNode int
	// maxAge defines how long to retain snapshots (0 = infinite)
	maxAge time.Duration
	// mu protects concurrent access to snapshots
	mu sync.RWMutex
}

// NewSnapshotManager creates a new snapshot manager with the specified retention policy.
// maxSnapshotsPerNode: number of snapshots to retain per node (0 = unlimited, recommended: 10-100)
// maxAge: duration to retain snapshots (0 = infinite, recommended: 1h for long workflows)
func NewSnapshotManager(maxSnapshotsPerNode int, maxAge time.Duration) *SnapshotManager {
	return &SnapshotManager{
		snapshots:           make(map[types.NodeID][]NodeVariableSnapshot),
		maxSnapshotsPerNode: maxSnapshotsPerNode,
		maxAge:              maxAge,
	}
}

// CaptureSnapshot creates a point-in-time snapshot of all variables for the given node.
// This performs a deep copy of all variable values to prevent mutation issues.
func (sm *SnapshotManager) CaptureSnapshot(nodeID types.NodeID, variables map[string]interface{}) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	// Deep copy the variables to ensure immutability
	variablesCopy, err := deepCopyVariables(variables)
	if err != nil {
		return fmt.Errorf("failed to deep copy variables: %w", err)
	}

	snapshot := NodeVariableSnapshot{
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Variables: variablesCopy,
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Add snapshot to the list for this node
	snapshots := sm.snapshots[nodeID]
	snapshots = append(snapshots, snapshot)

	// Apply retention policy
	snapshots = sm.applyRetentionPolicy(snapshots)

	sm.snapshots[nodeID] = snapshots

	return nil
}

// GetLatestSnapshot returns the most recent snapshot for a given node.
// Returns nil if no snapshot exists for the node.
// Returns a deep copy to prevent external modification.
func (sm *SnapshotManager) GetLatestSnapshot(nodeID types.NodeID) *NodeVariableSnapshot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshots := sm.snapshots[nodeID]
	if len(snapshots) == 0 {
		return nil
	}

	// Return a deep copy to prevent external modification of the variables map
	latest := snapshots[len(snapshots)-1]
	variablesCopy, _ := deepCopyVariables(latest.Variables)

	return &NodeVariableSnapshot{
		NodeID:    latest.NodeID,
		Timestamp: latest.Timestamp,
		Variables: variablesCopy,
	}
}

// GetAllSnapshots returns all snapshots for a given node.
// Returns a deep copy to prevent external modification.
func (sm *SnapshotManager) GetAllSnapshots(nodeID types.NodeID) []NodeVariableSnapshot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshots := sm.snapshots[nodeID]
	if len(snapshots) == 0 {
		return nil
	}

	// Return a deep copy to prevent external modification of the variables maps
	result := make([]NodeVariableSnapshot, len(snapshots))
	for i, snapshot := range snapshots {
		variablesCopy, _ := deepCopyVariables(snapshot.Variables)
		result[i] = NodeVariableSnapshot{
			NodeID:    snapshot.NodeID,
			Timestamp: snapshot.Timestamp,
			Variables: variablesCopy,
		}
	}
	return result
}

// GetSnapshotCount returns the number of snapshots stored for a given node.
func (sm *SnapshotManager) GetSnapshotCount(nodeID types.NodeID) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.snapshots[nodeID])
}

// GetTotalSnapshotCount returns the total number of snapshots across all nodes.
func (sm *SnapshotManager) GetTotalSnapshotCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	total := 0
	for _, snapshots := range sm.snapshots {
		total += len(snapshots)
	}
	return total
}

// Clear removes all snapshots (useful for cleanup or testing).
func (sm *SnapshotManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.snapshots = make(map[types.NodeID][]NodeVariableSnapshot)
}

// ClearNode removes all snapshots for a specific node.
func (sm *SnapshotManager) ClearNode(nodeID types.NodeID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.snapshots, nodeID)
}

// applyRetentionPolicy enforces the retention limits on a snapshot list.
// This method assumes the caller holds the write lock.
func (sm *SnapshotManager) applyRetentionPolicy(snapshots []NodeVariableSnapshot) []NodeVariableSnapshot {
	if len(snapshots) == 0 {
		return snapshots
	}

	// Apply time-based retention first
	if sm.maxAge > 0 {
		cutoff := time.Now().Add(-sm.maxAge)
		filtered := make([]NodeVariableSnapshot, 0, len(snapshots))
		for _, snapshot := range snapshots {
			if snapshot.Timestamp.After(cutoff) {
				filtered = append(filtered, snapshot)
			}
		}
		snapshots = filtered
	}

	// Apply count-based retention (keep most recent N)
	if sm.maxSnapshotsPerNode > 0 && len(snapshots) > sm.maxSnapshotsPerNode {
		// Keep only the most recent maxSnapshotsPerNode snapshots
		snapshots = snapshots[len(snapshots)-sm.maxSnapshotsPerNode:]
	}

	return snapshots
}

// deepCopyVariables creates a deep copy of a variable map to ensure immutability.
// This handles complex types (arrays, maps, nested structures) by using JSON serialization.
// For most workflow use cases, this provides a good balance between correctness and performance.
func deepCopyVariables(variables map[string]interface{}) (map[string]interface{}, error) {
	if variables == nil {
		return nil, nil
	}

	// Use JSON marshal/unmarshal for deep copying
	// This works for all JSON-serializable types and is simple/reliable
	// Performance: ~1-2μs per variable for simple types, ~10-50μs for complex nested structures
	data, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}

	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
	}

	return copy, nil
}

// GetMemoryStats returns statistics about memory usage of the snapshot manager.
// This is useful for monitoring and debugging memory consumption.
type SnapshotMemoryStats struct {
	// TotalSnapshots is the total number of snapshots stored
	TotalSnapshots int
	// NodeCount is the number of unique nodes with snapshots
	NodeCount int
	// AverageSnapshotsPerNode is the average number of snapshots per node
	AverageSnapshotsPerNode float64
	// EstimatedMemoryBytes is a rough estimate of memory usage in bytes
	EstimatedMemoryBytes int64
}

// GetMemoryStats computes memory usage statistics for the snapshot manager.
func (sm *SnapshotManager) GetMemoryStats() SnapshotMemoryStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := SnapshotMemoryStats{
		TotalSnapshots: 0,
		NodeCount:      len(sm.snapshots),
	}

	var totalVariableCount int64
	for _, snapshots := range sm.snapshots {
		stats.TotalSnapshots += len(snapshots)
		for _, snapshot := range snapshots {
			totalVariableCount += int64(len(snapshot.Variables))
		}
	}

	if stats.NodeCount > 0 {
		stats.AverageSnapshotsPerNode = float64(stats.TotalSnapshots) / float64(stats.NodeCount)
	}

	// Rough memory estimate:
	// - 100 bytes per snapshot struct overhead
	// - 100 bytes per variable entry (average for key + value)
	// - 50 bytes per NodeID in map
	stats.EstimatedMemoryBytes = int64(stats.TotalSnapshots)*100 +
		totalVariableCount*100 +
		int64(stats.NodeCount)*50

	return stats
}
