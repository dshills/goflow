package execution

import (
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// ExecutionEventType categorizes different event types during workflow execution.
type ExecutionEventType string

const (
	// EventExecutionStarted is emitted when a workflow execution begins.
	EventExecutionStarted ExecutionEventType = "execution.started"
	// EventExecutionCompleted is emitted when a workflow execution finishes successfully.
	EventExecutionCompleted ExecutionEventType = "execution.completed"
	// EventExecutionFailed is emitted when a workflow execution fails with an error.
	EventExecutionFailed ExecutionEventType = "execution.failed"
	// EventExecutionCancelled is emitted when a workflow execution is cancelled.
	EventExecutionCancelled ExecutionEventType = "execution.cancelled"

	// EventNodeStarted is emitted when a node begins execution.
	EventNodeStarted ExecutionEventType = "node.started"
	// EventNodeCompleted is emitted when a node completes successfully.
	EventNodeCompleted ExecutionEventType = "node.completed"
	// EventNodeFailed is emitted when a node fails with an error.
	EventNodeFailed ExecutionEventType = "node.failed"
	// EventNodeSkipped is emitted when a node is skipped (e.g., conditional branch).
	EventNodeSkipped ExecutionEventType = "node.skipped"

	// EventVariableChanged is emitted when a workflow variable is modified.
	EventVariableChanged ExecutionEventType = "variable.changed"

	// EventConditionEvaluated is emitted when a condition node evaluates its expression.
	EventConditionEvaluated ExecutionEventType = "condition.evaluated"

	// EventLoopStarted is emitted when a loop node begins iteration.
	EventLoopStarted ExecutionEventType = "loop.started"
	// EventLoopIteration is emitted for each loop iteration.
	EventLoopIteration ExecutionEventType = "loop.iteration"
	// EventLoopCompleted is emitted when a loop completes all iterations.
	EventLoopCompleted ExecutionEventType = "loop.completed"

	// EventProgressUpdate is emitted periodically to report execution progress.
	EventProgressUpdate ExecutionEventType = "progress.update"
)

// ExecutionEvent represents a real-time event during workflow execution.
type ExecutionEvent struct {
	// Type categorizes the event.
	Type ExecutionEventType
	// Timestamp records when this event occurred.
	Timestamp time.Time
	// ExecutionID identifies which execution this event belongs to.
	ExecutionID types.ExecutionID
	// NodeID identifies which node this event is about (if applicable).
	NodeID types.NodeID
	// Status is the new status for execution/node.
	Status interface{} // execution.Status or execution.NodeStatus
	// Variables is a snapshot of variables at this point.
	Variables map[string]interface{}
	// Error contains error details if applicable.
	Error error
	// Metadata contains additional event-specific data.
	Metadata map[string]interface{}
}

// EventFilter defines criteria for filtering events.
type EventFilter struct {
	// EventTypes specifies which event types to include (nil/empty means all types).
	EventTypes []ExecutionEventType
	// NodeIDs specifies which node IDs to include (nil/empty means all nodes).
	NodeIDs []types.NodeID
}

// Matches returns true if the event matches the filter criteria.
func (f *EventFilter) Matches(event ExecutionEvent) bool {
	// Check event type filter
	if len(f.EventTypes) > 0 {
		matched := false
		for _, eventType := range f.EventTypes {
			if event.Type == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check node ID filter
	if len(f.NodeIDs) > 0 {
		if event.NodeID == "" {
			// Event has no node ID, doesn't match node filter
			return false
		}
		matched := false
		for _, nodeID := range f.NodeIDs {
			if event.NodeID == nodeID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// ExecutionMonitor provides real-time monitoring of workflow execution.
// Note: ExecutionProgress is defined in progress.go
type ExecutionMonitor interface {
	// Subscribe returns a channel that receives all execution events in real-time.
	Subscribe() <-chan ExecutionEvent
	// Unsubscribe closes and removes a subscription.
	Unsubscribe(ch <-chan ExecutionEvent)
	// SubscribeFiltered returns a channel that only receives events matching the filter.
	SubscribeFiltered(filter EventFilter) <-chan ExecutionEvent
	// GetProgress returns current execution progress (percentage and node counts).
	GetProgress() ExecutionProgress
	// GetVariableSnapshot returns current values of all variables.
	GetVariableSnapshot() map[string]interface{}
	// GetExecutionState returns the current execution state.
	GetExecutionState() *execution.Execution
}

// subscription represents a single event subscriber.
type subscription struct {
	ch     chan ExecutionEvent
	filter *EventFilter // nil means no filtering
}

// monitor implements ExecutionMonitor with thread-safe event broadcasting.
type monitor struct {
	mu sync.RWMutex

	// exec is the execution being monitored
	exec *execution.Execution

	// totalNodes tracks the total number of nodes for progress calculation
	totalNodes int

	// subscribers holds all active event subscribers
	subscribers []*subscription

	// closed indicates if the monitor has been closed
	closed bool
}

// NewMonitor creates a new execution monitor for the given execution.
// totalNodes should be the total number of nodes in the workflow for progress tracking.
func NewMonitor(exec *execution.Execution, totalNodes int) ExecutionMonitor {
	return &monitor{
		exec:        exec,
		totalNodes:  totalNodes,
		subscribers: make([]*subscription, 0),
		closed:      false,
	}
}

// Subscribe returns a channel that receives all execution events.
func (m *monitor) Subscribe() <-chan ExecutionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		// Return a closed channel if monitor is closed
		ch := make(chan ExecutionEvent)
		close(ch)
		return ch
	}

	// Create buffered channel to prevent blocking event emission
	// Buffer size of 200 should handle bursts of events in high-throughput scenarios
	ch := make(chan ExecutionEvent, 200)
	sub := &subscription{
		ch:     ch,
		filter: nil,
	}
	m.subscribers = append(m.subscribers, sub)

	return ch
}

// SubscribeFiltered returns a channel that receives only filtered events.
func (m *monitor) SubscribeFiltered(filter EventFilter) <-chan ExecutionEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		// Return a closed channel if monitor is closed
		ch := make(chan ExecutionEvent)
		close(ch)
		return ch
	}

	// Create buffered channel to prevent blocking event emission
	ch := make(chan ExecutionEvent, 200)
	sub := &subscription{
		ch:     ch,
		filter: &filter,
	}
	m.subscribers = append(m.subscribers, sub)

	return ch
}

// Unsubscribe closes and removes a subscription.
func (m *monitor) Unsubscribe(ch <-chan ExecutionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the subscription
	for i, sub := range m.subscribers {
		if sub.ch == ch {
			// Close the channel
			close(sub.ch)
			// Remove from subscribers list
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			break
		}
	}
}

// GetProgress returns current execution progress.
func (m *monitor) GetProgress() ExecutionProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.exec == nil {
		return ExecutionProgress{}
	}

	var completedNodes, failedNodes, skippedNodes int
	var currentNode types.NodeID

	// Count node statuses from execution history
	for _, nodeExec := range m.exec.NodeExecutions {
		switch nodeExec.Status {
		case execution.NodeStatusCompleted:
			completedNodes++
		case execution.NodeStatusFailed:
			failedNodes++
		case execution.NodeStatusSkipped:
			skippedNodes++
		case execution.NodeStatusRunning:
			currentNode = nodeExec.NodeID
		}
	}

	// If no current node from running nodes, check context
	if currentNode == "" && m.exec.Context != nil {
		if ctxNode := m.exec.Context.GetCurrentNode(); ctxNode != nil {
			currentNode = *ctxNode
		}
	}

	// Calculate percentage
	var percentComplete float64
	if m.totalNodes > 0 {
		percentComplete = float64(completedNodes+failedNodes+skippedNodes) / float64(m.totalNodes) * 100.0
	}

	return ExecutionProgress{
		TotalNodes:      m.totalNodes,
		CompletedNodes:  completedNodes,
		FailedNodes:     failedNodes,
		SkippedNodes:    skippedNodes,
		CurrentNode:     currentNode,
		PercentComplete: percentComplete,
	}
}

// GetVariableSnapshot returns a copy of current variable values.
func (m *monitor) GetVariableSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.exec == nil || m.exec.Context == nil {
		return make(map[string]interface{})
	}

	// Get variable snapshot from context (already returns a copy)
	return m.exec.Context.GetVariableSnapshot()
}

// GetExecutionState returns the current execution state.
func (m *monitor) GetExecutionState() *execution.Execution {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.exec
}

// Emit sends an event to all subscribers (non-blocking).
// This is called by the execution engine to publish events.
func (m *monitor) Emit(event ExecutionEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Broadcast to all subscribers
	for _, sub := range m.subscribers {
		// Apply filter if present
		if sub.filter != nil && !sub.filter.Matches(event) {
			continue
		}

		// Non-blocking send to prevent slow subscribers from blocking execution
		select {
		case sub.ch <- event:
			// Event sent successfully
		default:
			// Channel buffer full, drop event to prevent blocking
			// In production, this could be logged or counted as a metric
		}
	}
}

// Close closes the monitor and all subscriber channels.
func (m *monitor) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.closed = true

	// Close all subscriber channels
	for _, sub := range m.subscribers {
		close(sub.ch)
	}

	// Clear subscribers
	m.subscribers = nil
}
