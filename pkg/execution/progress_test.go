package execution

import (
	"sync"
	"testing"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgressTracker(t *testing.T) {
	tests := []struct {
		name       string
		totalNodes int
	}{
		{
			name:       "zero nodes",
			totalNodes: 0,
		},
		{
			name:       "single node",
			totalNodes: 1,
		},
		{
			name:       "multiple nodes",
			totalNodes: 10,
		},
		{
			name:       "large workflow",
			totalNodes: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewProgressTracker(tt.totalNodes)

			assert.NotNil(t, tracker)
			assert.Equal(t, tt.totalNodes, tracker.totalNodes)

			progress := tracker.GetProgress()
			assert.Equal(t, tt.totalNodes, progress.TotalNodes)
			assert.Equal(t, 0, progress.CompletedNodes)
			assert.Equal(t, 0, progress.FailedNodes)
			assert.Equal(t, 0, progress.SkippedNodes)
			assert.Equal(t, types.NodeID(""), progress.CurrentNode)
			assert.Equal(t, 0.0, progress.PercentComplete)
		})
	}
}

func TestProgressTracker_OnNodeStarted(t *testing.T) {
	tracker := NewProgressTracker(5)

	nodeID := types.NodeID("node1")
	tracker.OnNodeStarted(nodeID)

	progress := tracker.GetProgress()
	assert.Equal(t, nodeID, progress.CurrentNode)
	assert.Equal(t, 0.0, progress.PercentComplete)
}

func TestProgressTracker_OnNodeCompleted(t *testing.T) {
	tracker := NewProgressTracker(4)

	// Complete first node
	tracker.OnNodeStarted("node1")
	tracker.OnNodeCompleted("node1")

	progress := tracker.GetProgress()
	assert.Equal(t, 1, progress.CompletedNodes)
	assert.Equal(t, 0, progress.FailedNodes)
	assert.Equal(t, 0, progress.SkippedNodes)
	assert.Equal(t, 25.0, progress.PercentComplete)
	assert.Equal(t, types.NodeID(""), progress.CurrentNode)

	// Complete second node
	tracker.OnNodeStarted("node2")
	tracker.OnNodeCompleted("node2")

	progress = tracker.GetProgress()
	assert.Equal(t, 2, progress.CompletedNodes)
	assert.Equal(t, 50.0, progress.PercentComplete)
}

func TestProgressTracker_OnNodeFailed(t *testing.T) {
	tracker := NewProgressTracker(4)

	tracker.OnNodeStarted("node1")
	tracker.OnNodeFailed("node1")

	progress := tracker.GetProgress()
	assert.Equal(t, 0, progress.CompletedNodes)
	assert.Equal(t, 1, progress.FailedNodes)
	assert.Equal(t, 0, progress.SkippedNodes)
	assert.Equal(t, 25.0, progress.PercentComplete)
	assert.Equal(t, types.NodeID(""), progress.CurrentNode)
}

func TestProgressTracker_OnNodeSkipped(t *testing.T) {
	tracker := NewProgressTracker(4)

	tracker.OnNodeStarted("node1")
	tracker.OnNodeSkipped("node1")

	progress := tracker.GetProgress()
	assert.Equal(t, 0, progress.CompletedNodes)
	assert.Equal(t, 0, progress.FailedNodes)
	assert.Equal(t, 1, progress.SkippedNodes)
	assert.Equal(t, 25.0, progress.PercentComplete)
	assert.Equal(t, types.NodeID(""), progress.CurrentNode)
}

func TestProgressTracker_MixedNodeOutcomes(t *testing.T) {
	tracker := NewProgressTracker(10)

	// Complete 5 nodes
	for i := 1; i <= 5; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeCompleted(nodeID)
	}

	// Fail 2 nodes
	for i := 6; i <= 7; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeFailed(nodeID)
	}

	// Skip 2 nodes
	for i := 8; i <= 9; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeSkipped(nodeID)
	}

	progress := tracker.GetProgress()
	assert.Equal(t, 10, progress.TotalNodes)
	assert.Equal(t, 5, progress.CompletedNodes)
	assert.Equal(t, 2, progress.FailedNodes)
	assert.Equal(t, 2, progress.SkippedNodes)
	assert.Equal(t, 90.0, progress.PercentComplete) // 9 out of 10 executed
}

func TestProgressTracker_MonotonicProgress(t *testing.T) {
	tracker := NewProgressTracker(5)

	var progressValues []float64

	for i := 1; i <= 5; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeCompleted(nodeID)

		progress := tracker.GetProgress()
		progressValues = append(progressValues, progress.PercentComplete)
	}

	// Verify progress never decreases
	for i := 1; i < len(progressValues); i++ {
		assert.GreaterOrEqual(t, progressValues[i], progressValues[i-1],
			"Progress should be monotonically increasing")
	}

	// Final progress should be 100%
	assert.Equal(t, 100.0, progressValues[len(progressValues)-1])
}

func TestProgressTracker_ConditionalWorkflow(t *testing.T) {
	// Workflow with 6 total nodes, but only 4 executed due to conditional
	tracker := NewProgressTracker(6)

	// Start node
	tracker.OnNodeStarted("start")
	tracker.OnNodeCompleted("start")

	// Condition node
	tracker.OnNodeStarted("condition")
	tracker.OnNodeCompleted("condition")

	// True branch taken (1 node)
	tracker.OnNodeStarted("true_branch")
	tracker.OnNodeCompleted("true_branch")

	// False branch skipped (1 node)
	tracker.OnNodeStarted("false_branch")
	tracker.OnNodeSkipped("false_branch")

	progress := tracker.GetProgress()
	assert.Equal(t, 6, progress.TotalNodes)
	assert.Equal(t, 3, progress.CompletedNodes)
	assert.Equal(t, 0, progress.FailedNodes)
	assert.Equal(t, 1, progress.SkippedNodes)
	// 4 out of 6 nodes executed = 66.67%
	assert.InDelta(t, 66.67, progress.PercentComplete, 0.1)
}

func TestProgressTracker_ParallelExecution(t *testing.T) {
	tracker := NewProgressTracker(100)

	// Simulate parallel execution with goroutines
	var wg sync.WaitGroup
	nodeCount := 100

	for i := 0; i < nodeCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			nodeID := types.NodeID(string(rune('A' + index)))
			tracker.OnNodeStarted(nodeID)
			tracker.OnNodeCompleted(nodeID)
		}(i)
	}

	wg.Wait()

	progress := tracker.GetProgress()
	assert.Equal(t, 100, progress.TotalNodes)
	assert.Equal(t, 100, progress.CompletedNodes)
	assert.Equal(t, 0, progress.FailedNodes)
	assert.Equal(t, 0, progress.SkippedNodes)
	assert.Equal(t, 100.0, progress.PercentComplete)
}

func TestProgressTracker_ConcurrentReads(t *testing.T) {
	tracker := NewProgressTracker(50)

	var wg sync.WaitGroup

	// Start updating progress in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			nodeID := types.NodeID(string(rune('0' + i)))
			tracker.OnNodeStarted(nodeID)
			tracker.OnNodeCompleted(nodeID)
		}
	}()

	// Concurrent reads should never panic
	readCount := 100
	for i := 0; i < readCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = tracker.GetProgress()
		}()
	}

	wg.Wait()

	// Final state should be consistent
	progress := tracker.GetProgress()
	assert.Equal(t, 50, progress.CompletedNodes)
	assert.Equal(t, 100.0, progress.PercentComplete)
}

func TestProgressTracker_UpdateFromExecution(t *testing.T) {
	tracker := NewProgressTracker(5)

	// Create a mock execution with node executions
	exec, err := execution.NewExecution("workflow-1", "1.0", nil)
	require.NoError(t, err)

	// Add completed nodes
	for i := 1; i <= 3; i++ {
		nodeExec := execution.NewNodeExecution(
			exec.ID,
			types.NodeID(string(rune('0'+i))),
			"passthrough",
		)
		nodeExec.Start()
		nodeExec.Complete(nil)
		err := exec.AddNodeExecution(nodeExec)
		require.NoError(t, err)
	}

	// Add failed node
	failedExec := execution.NewNodeExecution(exec.ID, "node4", "transform")
	failedExec.Start()
	failedExec.Fail(&execution.NodeError{
		Type:    execution.ErrorTypeExecution,
		Message: "transform failed",
	})
	err = exec.AddNodeExecution(failedExec)
	require.NoError(t, err)

	// Update tracker from execution
	tracker.UpdateFromExecution(exec)

	progress := tracker.GetProgress()
	assert.Equal(t, 5, progress.TotalNodes)
	assert.Equal(t, 3, progress.CompletedNodes)
	assert.Equal(t, 1, progress.FailedNodes)
	assert.Equal(t, 0, progress.SkippedNodes)
	assert.Equal(t, 80.0, progress.PercentComplete) // 4 out of 5
}

func TestProgressTracker_Reset(t *testing.T) {
	tracker := NewProgressTracker(5)

	// Execute some nodes
	tracker.OnNodeStarted("node1")
	tracker.OnNodeCompleted("node1")
	tracker.OnNodeStarted("node2")
	tracker.OnNodeFailed("node2")

	// Verify progress was updated
	progress := tracker.GetProgress()
	assert.Equal(t, 1, progress.CompletedNodes)
	assert.Equal(t, 1, progress.FailedNodes)
	assert.Equal(t, 40.0, progress.PercentComplete)

	// Reset
	tracker.Reset()

	// Verify everything is reset
	progress = tracker.GetProgress()
	assert.Equal(t, 5, progress.TotalNodes) // Total nodes unchanged
	assert.Equal(t, 0, progress.CompletedNodes)
	assert.Equal(t, 0, progress.FailedNodes)
	assert.Equal(t, 0, progress.SkippedNodes)
	assert.Equal(t, types.NodeID(""), progress.CurrentNode)
	assert.Equal(t, 0.0, progress.PercentComplete)
}

func TestProgressTracker_ZeroNodes(t *testing.T) {
	tracker := NewProgressTracker(0)

	// Operations should not panic with zero nodes
	tracker.OnNodeStarted("node1")
	tracker.OnNodeCompleted("node1")

	progress := tracker.GetProgress()
	assert.Equal(t, 0, progress.TotalNodes)
	assert.Equal(t, 1, progress.CompletedNodes)
	// Percentage stays 0 for zero total nodes
	assert.Equal(t, 0.0, progress.PercentComplete)
}

func TestProgressTracker_CurrentNodeRaceCondition(t *testing.T) {
	tracker := NewProgressTracker(100)

	// Rapidly start and complete nodes in parallel
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			nodeID := types.NodeID(string(rune('A' + index)))
			tracker.OnNodeStarted(nodeID)
			tracker.OnNodeCompleted(nodeID)
		}(i)
	}

	wg.Wait()

	// Current node should be cleared (empty)
	progress := tracker.GetProgress()
	assert.Equal(t, types.NodeID(""), progress.CurrentNode)
}

func TestProgressTracker_PercentageNeverExceeds100(t *testing.T) {
	tracker := NewProgressTracker(5)

	// Execute more nodes than total (edge case)
	for i := 0; i < 10; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeCompleted(nodeID)

		progress := tracker.GetProgress()
		assert.LessOrEqual(t, progress.PercentComplete, 100.0,
			"Progress should never exceed 100%%")
	}
}

func BenchmarkProgressTracker_GetProgress(b *testing.B) {
	tracker := NewProgressTracker(100)

	// Setup some progress
	for i := 0; i < 50; i++ {
		nodeID := types.NodeID(string(rune('0' + i)))
		tracker.OnNodeStarted(nodeID)
		tracker.OnNodeCompleted(nodeID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.GetProgress()
	}
}

func BenchmarkProgressTracker_OnNodeCompleted(b *testing.B) {
	tracker := NewProgressTracker(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := types.NodeID(string(rune('0' + (i % 256))))
		tracker.OnNodeCompleted(nodeID)
	}
}

func BenchmarkProgressTracker_Concurrent(b *testing.B) {
	tracker := NewProgressTracker(b.N)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			nodeID := types.NodeID(string(rune('0' + (i % 256))))
			tracker.OnNodeCompleted(nodeID)
			_ = tracker.GetProgress()
			i++
		}
	})
}
