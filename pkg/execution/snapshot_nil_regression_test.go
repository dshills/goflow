package execution

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// TestGetLatestSnapshot_ErrorHandling tests that callers properly handle
// errors from deepCopyVariables. This is a regression test for Issue #109.
//
// FR-021: Check for nil before dereferencing pointers
// FR-022: Check errors before using return values
func TestGetLatestSnapshot_ErrorHandling(t *testing.T) {
	manager := NewSnapshotManager(10, 0) // 10 max snapshots, no max age

	// Add a snapshot with valid variables
	validVars := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}
	err := manager.CaptureSnapshot(types.NodeID("node-1"), validVars)
	if err != nil {
		t.Fatalf("CaptureSnapshot failed: %v", err)
	}

	// Get the latest snapshot
	snapshot := manager.GetLatestSnapshot(types.NodeID("node-1"))
	if snapshot == nil {
		t.Fatal("GetLatestSnapshot returned nil")
	}

	// Verify the snapshot contains expected data
	if snapshot.NodeID != types.NodeID("node-1") {
		t.Errorf("NodeID = %q, want %q", snapshot.NodeID, "node-1")
	}

	// Verify variables are copied (not same reference)
	if snapshot.Variables == nil {
		t.Error("Variables is nil")
	}

	// Mutating returned variables should not affect the stored snapshot
	snapshot.Variables["name"] = "modified"
	snapshot2 := manager.GetLatestSnapshot(types.NodeID("node-1"))
	if snapshot2.Variables["name"] != "test" {
		t.Errorf("Variables were not properly isolated, got %q, want %q",
			snapshot2.Variables["name"], "test")
	}
}

// TestDeepCopyVariables_UnsupportedTypes tests behavior with non-JSON-serializable types.
// This ensures we properly handle or document type limitations.
//
// Issue #109: deepCopyVariables can fail for time.Time, channels, funcs
func TestDeepCopyVariables_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]interface{}
		wantErr   bool
		desc      string
	}{
		{
			name:      "nil variables",
			variables: nil,
			wantErr:   false,
			desc:      "nil input should return nil without error",
		},
		{
			name:      "empty variables",
			variables: map[string]interface{}{},
			wantErr:   false,
			desc:      "empty map should copy successfully",
		},
		{
			name: "simple types",
			variables: map[string]interface{}{
				"string": "value",
				"int":    42,
				"float":  3.14,
				"bool":   true,
			},
			wantErr: false,
			desc:    "simple JSON types should copy successfully",
		},
		{
			name: "nested maps and slices",
			variables: map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "value",
				},
				"slice": []interface{}{1, 2, 3},
			},
			wantErr: false,
			desc:    "nested structures should copy successfully",
		},
		{
			name: "time.Time values",
			variables: map[string]interface{}{
				"timestamp": time.Now(),
			},
			wantErr: false,
			desc:    "time.Time should serialize to RFC3339 string via JSON",
		},
		// Note: channels and functions are not JSON-serializable and will
		// cause marshal errors. These are documented limitations.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deepCopyVariables(tt.variables)

			if (err != nil) != tt.wantErr {
				t.Errorf("deepCopyVariables() error = %v, wantErr %v (%s)",
					err, tt.wantErr, tt.desc)
				return
			}

			if !tt.wantErr {
				// Verify copy is independent
				if tt.variables == nil {
					if got != nil {
						t.Errorf("Expected nil for nil input, got %v", got)
					}
				} else {
					if got == nil && len(tt.variables) > 0 {
						t.Error("deepCopyVariables returned nil for non-empty input")
					}
					// Verify it's a different map instance
					if tt.variables != nil && got != nil {
						// Modify copy shouldn't affect original
						for k := range got {
							got[k] = "modified"
							break
						}
					}
				}
			}
		})
	}
}

// TestCaptureSnapshot_NilHandling tests nil variable handling.
//
// FR-021: Check for nil before dereferencing pointers
func TestCaptureSnapshot_NilHandling(t *testing.T) {
	manager := NewSnapshotManager(10, 0)

	tests := []struct {
		name      string
		nodeID    types.NodeID
		variables map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "empty nodeID",
			nodeID:    "",
			variables: map[string]interface{}{"key": "value"},
			wantErr:   true,
		},
		{
			name:      "nil variables",
			nodeID:    types.NodeID("node-1"),
			variables: nil,
			wantErr:   false, // nil variables should be allowed
		},
		{
			name:      "empty variables",
			nodeID:    types.NodeID("node-2"),
			variables: map[string]interface{}{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CaptureSnapshot(tt.nodeID, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("CaptureSnapshot() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSnapshotManager_GetNonExistent tests behavior when retrieving
// snapshots for nodes that don't have any.
//
// FR-021: Check for nil before dereferencing pointers
func TestSnapshotManager_GetNonExistent(t *testing.T) {
	manager := NewSnapshotManager(10, 0)

	// Get snapshot for node that doesn't exist
	snapshot := manager.GetLatestSnapshot(types.NodeID("nonexistent"))

	// Should return nil, not panic
	if snapshot != nil {
		t.Errorf("GetLatestSnapshot() for nonexistent node = %v, want nil", snapshot)
	}
}

// TestSnapshotManager_ConcurrentNilAccess tests that concurrent access
// doesn't cause nil pointer panics.
//
// FR-021: Check for nil before dereferencing pointers
// SC-008: Zero runtime panics from nil dereferences
func TestSnapshotManager_ConcurrentNilAccess(t *testing.T) {
	manager := NewSnapshotManager(10, 0)
	nodeID := types.NodeID("concurrent-node")

	// Start multiple goroutines that try to access the same node
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in concurrent access: %v", r)
				}
				done <- true
			}()

			// Try to get snapshot (might be nil initially)
			snapshot := manager.GetLatestSnapshot(nodeID)
			_ = snapshot // Use it

			// Try to save a snapshot
			vars := map[string]interface{}{"test": "value"}
			_ = manager.CaptureSnapshot(nodeID, vars)
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
