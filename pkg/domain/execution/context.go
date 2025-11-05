package execution

import (
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/domain/types"
)

// ExecutionContext holds the runtime state during workflow execution.
// All variable operations are thread-safe using RWMutex.
type ExecutionContext struct {
	// CurrentNodeID is the node currently being executed (nil if not running).
	CurrentNodeID *types.NodeID
	// Variables stores the current variable values (thread-safe).
	Variables map[string]interface{}
	// variableHistory is an append-only log of variable changes for audit trail.
	variableHistory []VariableSnapshot
	// executionTrace records the execution path through the workflow.
	executionTrace []TraceEntry
	// mu protects concurrent access to all fields.
	mu sync.RWMutex
}

// NewExecutionContext creates a new execution context with optional initial variables.
func NewExecutionContext(initialVars map[string]interface{}) (*ExecutionContext, error) {
	ctx := &ExecutionContext{
		Variables:       make(map[string]interface{}),
		variableHistory: []VariableSnapshot{},
		executionTrace:  []TraceEntry{},
	}

	// Copy initial variables if provided
	if initialVars != nil {
		for key, value := range initialVars {
			ctx.Variables[key] = value
		}
	}

	return ctx, nil
}

// GetVariable retrieves a variable value by name.
// Returns (value, true) if the variable exists, (nil, false) otherwise.
func (ctx *ExecutionContext) GetVariable(name string) (interface{}, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	val, exists := ctx.Variables[name]
	return val, exists
}

// SetVariable sets a variable value and records the change in the audit trail.
// This is a convenience wrapper for SetVariableWithNode with no node execution ID.
func (ctx *ExecutionContext) SetVariable(name string, value interface{}) error {
	return ctx.SetVariableWithNode(name, value, "")
}

// SetVariableWithNode sets a variable value and records which node made the change.
// Creates a snapshot in the variable history for audit trail.
func (ctx *ExecutionContext) SetVariableWithNode(name string, value interface{}, nodeExecID types.NodeExecutionID) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// Capture old value for snapshot
	oldValue, _ := ctx.Variables[name]

	// Update variable
	ctx.Variables[name] = value

	// Create snapshot for audit trail
	snapshot := VariableSnapshot{
		Timestamp:       time.Now(),
		NodeExecutionID: nodeExecID,
		VariableName:    name,
		OldValue:        oldValue,
		NewValue:        value,
	}

	ctx.variableHistory = append(ctx.variableHistory, snapshot)

	return nil
}

// GetVariableHistory returns the complete variable change history.
// Returns a copy to prevent external modification.
func (ctx *ExecutionContext) GetVariableHistory() []VariableSnapshot {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	// Return a copy to prevent external modification of the audit trail
	history := make([]VariableSnapshot, len(ctx.variableHistory))
	copy(history, ctx.variableHistory)
	return history
}

// RecordTrace adds an entry to the execution trace.
func (ctx *ExecutionContext) RecordTrace(nodeID types.NodeID, event string) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	entry := TraceEntry{
		NodeID:    nodeID,
		Event:     event,
		Timestamp: time.Now(),
	}

	ctx.executionTrace = append(ctx.executionTrace, entry)
	return nil
}

// GetExecutionTrace returns the complete execution trace.
// Returns a copy to prevent external modification.
func (ctx *ExecutionContext) GetExecutionTrace() []TraceEntry {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	// Return a copy to prevent external modification of the trace
	trace := make([]TraceEntry, len(ctx.executionTrace))
	copy(trace, ctx.executionTrace)
	return trace
}

// SetCurrentNode sets the currently executing node.
// Pass nil to clear the current node.
func (ctx *ExecutionContext) SetCurrentNode(nodeID *types.NodeID) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.CurrentNodeID = nodeID
}

// CreateSnapshot creates a point-in-time snapshot of all variables.
// This is useful for debugging or storing execution state.
func (ctx *ExecutionContext) CreateSnapshot() map[string]interface{} {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	// Create a deep copy of the variables map
	snapshot := make(map[string]interface{}, len(ctx.Variables))
	for key, value := range ctx.Variables {
		snapshot[key] = value
	}

	return snapshot
}
