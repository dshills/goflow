package execution

import (
	"sync"
	"sync/atomic"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// ExecutionProgress tracks workflow execution progress.
type ExecutionProgress struct {
	// TotalNodes is the total number of nodes in the workflow.
	TotalNodes int
	// CompletedNodes is the number of completed nodes.
	CompletedNodes int
	// FailedNodes is the number of failed nodes.
	FailedNodes int
	// SkippedNodes is the number of skipped nodes.
	SkippedNodes int
	// CurrentNode is the node currently being executed (if any).
	CurrentNode types.NodeID
	// PercentComplete is the completion percentage (0-100).
	PercentComplete float64
}

// ProgressTracker maintains execution progress state with thread-safe updates.
// Uses atomic operations and minimal locking for high-performance progress tracking.
type ProgressTracker struct {
	// totalNodes is the immutable total count (set once during initialization)
	totalNodes int

	// Atomic counters for lock-free reads (int32 for atomic operations)
	completedNodes int32
	failedNodes    int32
	skippedNodes   int32

	// Current node tracking (requires mutex as it's not numeric)
	currentNodeMu sync.RWMutex
	currentNode   types.NodeID

	// Last computed progress percentage (cached for consistency)
	percentCompleteMu sync.RWMutex
	percentComplete   float64
}

// NewProgressTracker creates a new progress tracker for a workflow.
// totalNodes should be the count of all nodes in the workflow.
func NewProgressTracker(totalNodes int) *ProgressTracker {
	return &ProgressTracker{
		totalNodes:      totalNodes,
		completedNodes:  0,
		failedNodes:     0,
		skippedNodes:    0,
		percentComplete: 0.0,
	}
}

// OnNodeStarted should be called when a node begins execution.
// Updates the current node being tracked.
func (pt *ProgressTracker) OnNodeStarted(nodeID types.NodeID) {
	pt.currentNodeMu.Lock()
	pt.currentNode = nodeID
	pt.currentNodeMu.Unlock()
}

// OnNodeCompleted should be called when a node completes successfully.
// Increments completed count and recalculates progress.
func (pt *ProgressTracker) OnNodeCompleted(nodeID types.NodeID) {
	atomic.AddInt32(&pt.completedNodes, 1)
	pt.updateProgress()
	pt.clearCurrentNode(nodeID)
}

// OnNodeFailed should be called when a node fails during execution.
// Increments failed count and recalculates progress.
func (pt *ProgressTracker) OnNodeFailed(nodeID types.NodeID) {
	atomic.AddInt32(&pt.failedNodes, 1)
	pt.updateProgress()
	pt.clearCurrentNode(nodeID)
}

// OnNodeSkipped should be called when a node is skipped (e.g., conditional branch not taken).
// Increments skipped count and recalculates progress.
func (pt *ProgressTracker) OnNodeSkipped(nodeID types.NodeID) {
	atomic.AddInt32(&pt.skippedNodes, 1)
	pt.updateProgress()
	pt.clearCurrentNode(nodeID)
}

// GetProgress returns the current execution progress snapshot.
// This is an O(1) operation using atomic reads and cached percentage.
func (pt *ProgressTracker) GetProgress() ExecutionProgress {
	// Read atomic counters (no locks needed)
	completed := int(atomic.LoadInt32(&pt.completedNodes))
	failed := int(atomic.LoadInt32(&pt.failedNodes))
	skipped := int(atomic.LoadInt32(&pt.skippedNodes))

	// Read current node (requires lock)
	pt.currentNodeMu.RLock()
	currentNode := pt.currentNode
	pt.currentNodeMu.RUnlock()

	// Read cached percentage (requires lock)
	pt.percentCompleteMu.RLock()
	percent := pt.percentComplete
	pt.percentCompleteMu.RUnlock()

	return ExecutionProgress{
		TotalNodes:      pt.totalNodes,
		CompletedNodes:  completed,
		FailedNodes:     failed,
		SkippedNodes:    skipped,
		CurrentNode:     currentNode,
		PercentComplete: percent,
	}
}

// updateProgress recalculates and caches the progress percentage.
// Called after any node status change to maintain monotonic progress.
func (pt *ProgressTracker) updateProgress() {
	if pt.totalNodes == 0 {
		return
	}

	// Calculate executed nodes (completed + failed + skipped)
	// We count all executed nodes regardless of outcome
	completed := int(atomic.LoadInt32(&pt.completedNodes))
	failed := int(atomic.LoadInt32(&pt.failedNodes))
	skipped := int(atomic.LoadInt32(&pt.skippedNodes))

	executedNodes := completed + failed + skipped

	// Calculate percentage (ensure it doesn't exceed 100%)
	percent := float64(executedNodes) / float64(pt.totalNodes) * 100.0
	if percent > 100.0 {
		percent = 100.0
	}

	// Update cached percentage
	pt.percentCompleteMu.Lock()
	// Ensure monotonic increase (never decrease)
	if percent > pt.percentComplete {
		pt.percentComplete = percent
	}
	pt.percentCompleteMu.Unlock()
}

// clearCurrentNode clears the current node if it matches the given nodeID.
// This prevents race conditions where a new node starts before the old one finishes.
func (pt *ProgressTracker) clearCurrentNode(nodeID types.NodeID) {
	pt.currentNodeMu.Lock()
	if pt.currentNode == nodeID {
		pt.currentNode = ""
	}
	pt.currentNodeMu.Unlock()
}

// UpdateFromExecution updates progress based on an Execution's NodeExecutions.
// This is useful for reconstructing progress state from a persisted execution.
func (pt *ProgressTracker) UpdateFromExecution(exec *execution.Execution) {
	// Reset counters
	atomic.StoreInt32(&pt.completedNodes, 0)
	atomic.StoreInt32(&pt.failedNodes, 0)
	atomic.StoreInt32(&pt.skippedNodes, 0)

	// Count node executions by status
	var completed, failed, skipped int32
	for _, nodeExec := range exec.NodeExecutions {
		switch nodeExec.Status {
		case execution.NodeStatusCompleted:
			completed++
		case execution.NodeStatusFailed:
			failed++
		case execution.NodeStatusSkipped:
			skipped++
		}
	}

	// Update counters
	atomic.StoreInt32(&pt.completedNodes, completed)
	atomic.StoreInt32(&pt.failedNodes, failed)
	atomic.StoreInt32(&pt.skippedNodes, skipped)

	// Update current node from execution context
	pt.currentNodeMu.Lock()
	if exec.Context != nil && exec.Context.GetCurrentNode() != nil {
		pt.currentNode = *exec.Context.GetCurrentNode()
	} else {
		pt.currentNode = ""
	}
	pt.currentNodeMu.Unlock()

	// Recalculate progress
	pt.updateProgress()
}

// Reset resets all progress tracking to initial state.
// Useful for execution retry scenarios.
func (pt *ProgressTracker) Reset() {
	atomic.StoreInt32(&pt.completedNodes, 0)
	atomic.StoreInt32(&pt.failedNodes, 0)
	atomic.StoreInt32(&pt.skippedNodes, 0)

	pt.currentNodeMu.Lock()
	pt.currentNode = ""
	pt.currentNodeMu.Unlock()

	pt.percentCompleteMu.Lock()
	pt.percentComplete = 0.0
	pt.percentCompleteMu.Unlock()
}
