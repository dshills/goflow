package tui

import (
	"errors"
	"time"

	"github.com/dshills/goflow/pkg/workflow"
)

// workflowSnapshot represents a point-in-time state of the workflow
type workflowSnapshot struct {
	Nodes       []workflow.Node     // Deep copy of nodes
	Edges       []*workflow.Edge    // Deep copy of edges
	CanvasState map[string]Position // Node positions on canvas
	Timestamp   time.Time           // When snapshot was created
}

// UndoStack manages undo/redo history with a circular buffer
type UndoStack struct {
	snapshots []workflowSnapshot // Circular buffer of snapshots
	cursor    int                // Current position (-1 if empty)
	capacity  int                // Maximum number of snapshots
}

// NewUndoStack creates a new undo stack with the specified capacity
func NewUndoStack(capacity int) *UndoStack {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	return &UndoStack{
		snapshots: make([]workflowSnapshot, 0, capacity),
		cursor:    -1,
		capacity:  capacity,
	}
}

// Push adds a new snapshot to the stack
// This clears any redo history beyond the current cursor
func (u *UndoStack) Push(wf *workflow.Workflow, canvasPositions map[string]Position) error {
	if wf == nil {
		return errors.New("cannot push nil workflow")
	}

	// Create snapshot with deep copies
	snapshot := workflowSnapshot{
		Nodes:       u.deepCopyNodes(wf.Nodes),
		Edges:       u.deepCopyEdges(wf.Edges),
		CanvasState: u.deepCopyPositions(canvasPositions),
		Timestamp:   time.Now(),
	}

	// If cursor is not at the end, we need to clear the redo stack
	// (any snapshots beyond cursor position)
	if u.cursor < len(u.snapshots)-1 {
		// Truncate snapshots to cursor+1 length
		u.snapshots = u.snapshots[:u.cursor+1]
	}

	// Check if we're at capacity
	if len(u.snapshots) >= u.capacity {
		// Circular buffer: remove oldest snapshot (index 0)
		// Shift all snapshots left and replace last one
		copy(u.snapshots, u.snapshots[1:])
		u.snapshots[len(u.snapshots)-1] = snapshot
		u.cursor = len(u.snapshots) - 1
	} else {
		// Still have room, append normally
		u.snapshots = append(u.snapshots, snapshot)
		u.cursor = len(u.snapshots) - 1
	}

	return nil
}

// Undo moves back one snapshot and returns it
// When at cursor position 0, undoing moves to -1 (before first snapshot)
// and returns nil to indicate "no snapshot state"
func (u *UndoStack) Undo() (*workflowSnapshot, error) {
	if !u.CanUndo() {
		return nil, errors.New("nothing to undo")
	}

	// Move cursor back
	u.cursor--

	// If cursor is now -1, we've undone to before the first snapshot
	// Return nil snapshot to indicate empty state
	if u.cursor < 0 {
		return nil, nil
	}

	return &u.snapshots[u.cursor], nil
}

// Redo moves forward one snapshot and returns it
func (u *UndoStack) Redo() (*workflowSnapshot, error) {
	if !u.CanRedo() {
		return nil, errors.New("nothing to redo")
	}

	// Move cursor forward
	u.cursor++

	// Return snapshot at new cursor position
	return &u.snapshots[u.cursor], nil
}

// CanUndo returns true if undo is available
func (u *UndoStack) CanUndo() bool {
	// Can undo if cursor is at position 0 or greater
	// (meaning there's at least one snapshot and we can move back)
	// Note: cursor starts at 0 when we have 1 snapshot, so we can undo to position -1 (before first snapshot)
	return u.cursor >= 0 && len(u.snapshots) > 0
}

// CanRedo returns true if redo is available
func (u *UndoStack) CanRedo() bool {
	// Can redo if cursor is not at the end of the snapshots
	// cursor can be -1 (before first snapshot), and we can still redo to position 0
	return len(u.snapshots) > 0 && u.cursor < len(u.snapshots)-1
}

// Clear resets the undo stack
func (u *UndoStack) Clear() {
	u.snapshots = make([]workflowSnapshot, 0, u.capacity)
	u.cursor = -1
}

// Size returns the current number of snapshots
func (u *UndoStack) Size() int {
	return len(u.snapshots)
}

// deepCopyNodes creates a deep copy of the nodes slice
func (u *UndoStack) deepCopyNodes(nodes []workflow.Node) []workflow.Node {
	if nodes == nil {
		return nil
	}

	copied := make([]workflow.Node, len(nodes))
	for i, node := range nodes {
		// Each node type should implement deep copy
		// For now, we rely on the node being copied by value or having proper copy methods
		copied[i] = u.deepCopyNode(node)
	}

	return copied
}

// deepCopyNode creates a deep copy of a single node
func (u *UndoStack) deepCopyNode(node workflow.Node) workflow.Node {
	if node == nil {
		return nil
	}

	// Use type assertion to handle different node types
	switch n := node.(type) {
	case *workflow.StartNode:
		return u.copyStartNode(n)
	case *workflow.EndNode:
		return u.copyEndNode(n)
	case *workflow.MCPToolNode:
		return u.copyMCPToolNode(n)
	case *workflow.TransformNode:
		return u.copyTransformNode(n)
	case *workflow.ConditionNode:
		return u.copyConditionNode(n)
	case *workflow.LoopNode:
		return u.copyLoopNode(n)
	case *workflow.ParallelNode:
		return u.copyParallelNode(n)
	default:
		// Fallback: return the node as-is (may not be safe)
		return node
	}
}

// Node-specific copy functions
func (u *UndoStack) copyStartNode(n *workflow.StartNode) workflow.Node {
	if n == nil {
		return nil
	}
	copy := &workflow.StartNode{
		ID: n.ID,
	}
	return copy
}

func (u *UndoStack) copyEndNode(n *workflow.EndNode) workflow.Node {
	if n == nil {
		return nil
	}
	copy := &workflow.EndNode{
		ID:          n.ID,
		ReturnValue: n.ReturnValue,
	}
	return copy
}

func (u *UndoStack) copyMCPToolNode(n *workflow.MCPToolNode) workflow.Node {
	if n == nil {
		return nil
	}

	// Deep copy parameters map
	params := make(map[string]string)
	for k, v := range n.Parameters {
		params[k] = v
	}

	// Deep copy retry policy if present
	var retry *workflow.RetryPolicy
	if n.Retry != nil {
		retry = &workflow.RetryPolicy{
			MaxAttempts:       n.Retry.MaxAttempts,
			InitialDelay:      n.Retry.InitialDelay,
			MaxDelay:          n.Retry.MaxDelay,
			BackoffMultiplier: n.Retry.BackoffMultiplier,
		}
	}

	copy := &workflow.MCPToolNode{
		ID:             n.ID,
		ServerID:       n.ServerID,
		ToolName:       n.ToolName,
		Parameters:     params,
		OutputVariable: n.OutputVariable,
		Retry:          retry,
	}
	return copy
}

func (u *UndoStack) copyTransformNode(n *workflow.TransformNode) workflow.Node {
	if n == nil {
		return nil
	}

	// Deep copy retry policy if present
	var retry *workflow.RetryPolicy
	if n.Retry != nil {
		retry = &workflow.RetryPolicy{
			MaxAttempts:       n.Retry.MaxAttempts,
			InitialDelay:      n.Retry.InitialDelay,
			MaxDelay:          n.Retry.MaxDelay,
			BackoffMultiplier: n.Retry.BackoffMultiplier,
		}
	}

	copy := &workflow.TransformNode{
		ID:             n.ID,
		Expression:     n.Expression,
		InputVariable:  n.InputVariable,
		OutputVariable: n.OutputVariable,
		Retry:          retry,
	}
	return copy
}

func (u *UndoStack) copyConditionNode(n *workflow.ConditionNode) workflow.Node {
	if n == nil {
		return nil
	}
	copy := &workflow.ConditionNode{
		ID:        n.ID,
		Condition: n.Condition,
	}
	return copy
}

func (u *UndoStack) copyLoopNode(n *workflow.LoopNode) workflow.Node {
	if n == nil {
		return nil
	}

	// Deep copy body slice
	body := make([]string, len(n.Body))
	copy(body, n.Body)

	nodeCopy := &workflow.LoopNode{
		ID:             n.ID,
		Collection:     n.Collection,
		ItemVariable:   n.ItemVariable,
		Body:           body,
		BreakCondition: n.BreakCondition,
	}
	return nodeCopy
}

func (u *UndoStack) copyParallelNode(n *workflow.ParallelNode) workflow.Node {
	if n == nil {
		return nil
	}

	// Deep copy branches
	branches := make([][]string, len(n.Branches))
	for i, branch := range n.Branches {
		branches[i] = make([]string, len(branch))
		copy(branches[i], branch)
	}

	nodeCopy := &workflow.ParallelNode{
		ID:            n.ID,
		Branches:      branches,
		MergeStrategy: n.MergeStrategy,
	}
	return nodeCopy
}

// deepCopyEdges creates a deep copy of the edges slice
func (u *UndoStack) deepCopyEdges(edges []*workflow.Edge) []*workflow.Edge {
	if edges == nil {
		return nil
	}

	copied := make([]*workflow.Edge, len(edges))
	for i, edge := range edges {
		if edge != nil {
			copied[i] = &workflow.Edge{
				ID:         edge.ID,
				FromNodeID: edge.FromNodeID,
				ToNodeID:   edge.ToNodeID,
				Condition:  edge.Condition,
			}
		}
	}

	return copied
}

// deepCopyPositions creates a deep copy of the positions map
func (u *UndoStack) deepCopyPositions(positions map[string]Position) map[string]Position {
	if positions == nil {
		return nil
	}

	copied := make(map[string]Position, len(positions))
	for key, pos := range positions {
		copied[key] = Position{X: pos.X, Y: pos.Y}
	}

	return copied
}
