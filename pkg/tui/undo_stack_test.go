package tui

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/workflow"
)

func TestUndoStack_NewUndoStack(t *testing.T) {
	stack := NewUndoStack(100)

	if stack == nil {
		t.Fatal("NewUndoStack returned nil")
	}

	if stack.capacity != 100 {
		t.Errorf("expected capacity 100, got %d", stack.capacity)
	}

	if stack.cursor != -1 {
		t.Errorf("expected initial cursor -1, got %d", stack.cursor)
	}

	if len(stack.snapshots) != 0 {
		t.Errorf("expected empty snapshots, got %d", len(stack.snapshots))
	}
}

func TestUndoStack_Push(t *testing.T) {
	stack := NewUndoStack(5)

	// Create a simple workflow snapshot
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)

	positions := map[string]Position{
		start.GetID(): {X: 10, Y: 20},
	}

	// Push first snapshot
	err := stack.Push(wf, positions)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if len(stack.snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(stack.snapshots))
	}

	if stack.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", stack.cursor)
	}

	// Verify snapshot data
	snapshot := stack.snapshots[0]
	if len(snapshot.Nodes) != 1 {
		t.Errorf("expected 1 node in snapshot, got %d", len(snapshot.Nodes))
	}

	if len(snapshot.CanvasState) != 1 {
		t.Errorf("expected 1 position in snapshot, got %d", len(snapshot.CanvasState))
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("snapshot timestamp is zero")
	}
}

func TestUndoStack_PushMultiple(t *testing.T) {
	stack := NewUndoStack(5)

	// Push multiple snapshots
	for i := 0; i < 3; i++ {
		wf, _ := workflow.NewWorkflow("test", "test workflow")
		start := &workflow.StartNode{ID: "start-node-1"}
		wf.AddNode(start)
		positions := map[string]Position{start.GetID(): {X: i * 10, Y: i * 20}}

		err := stack.Push(wf, positions)
		if err != nil {
			t.Fatalf("Push %d failed: %v", i, err)
		}
	}

	if len(stack.snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(stack.snapshots))
	}

	if stack.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", stack.cursor)
	}
}

func TestUndoStack_CapacityOverflow(t *testing.T) {
	capacity := 3
	stack := NewUndoStack(capacity)

	// Push more snapshots than capacity
	for i := 0; i < 5; i++ {
		wf, _ := workflow.NewWorkflow("test", "test workflow")
		start := &workflow.StartNode{ID: "start-node-1"}
		wf.AddNode(start)
		positions := map[string]Position{start.GetID(): {X: i * 10, Y: i * 20}}

		err := stack.Push(wf, positions)
		if err != nil {
			t.Fatalf("Push %d failed: %v", i, err)
		}
	}

	// Should have exactly capacity snapshots (circular buffer)
	if len(stack.snapshots) != capacity {
		t.Errorf("expected %d snapshots after overflow, got %d", capacity, len(stack.snapshots))
	}

	// Cursor should be at last position
	if stack.cursor != capacity-1 {
		t.Errorf("expected cursor %d, got %d", capacity-1, stack.cursor)
	}

	// Oldest snapshots should be evicted
	// The remaining snapshots should be the last 3 pushed (indexes 2, 3, 4)
	firstSnapshot := stack.snapshots[0]
	firstPos := firstSnapshot.CanvasState[firstSnapshot.Nodes[0].GetID()]
	// After circular buffer wrapping, first snapshot should be from iteration 2
	// (iterations 0 and 1 were evicted)
	if firstPos.X != 20 { // iteration 2: X = 2 * 10 = 20
		t.Errorf("expected first snapshot X=20 (from iteration 2), got X=%d", firstPos.X)
	}
}

func TestUndoStack_Undo(t *testing.T) {
	stack := NewUndoStack(5)

	// Push three snapshots
	for i := 0; i < 3; i++ {
		wf, _ := workflow.NewWorkflow("test", "test workflow")
		start := &workflow.StartNode{ID: "start-node-1"}
		wf.AddNode(start)
		positions := map[string]Position{start.GetID(): {X: i * 10, Y: i * 20}}
		stack.Push(wf, positions)
	}

	// Undo once
	snapshot, err := stack.Undo()
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Undo returned nil snapshot")
	}

	// Should return snapshot at cursor-1 (index 1)
	pos := snapshot.CanvasState[snapshot.Nodes[0].GetID()]
	if pos.X != 10 { // iteration 1: X = 1 * 10 = 10
		t.Errorf("expected X=10, got X=%d", pos.X)
	}

	// Cursor should move back
	if stack.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", stack.cursor)
	}
}

func TestUndoStack_UndoEmpty(t *testing.T) {
	stack := NewUndoStack(5)

	// Try to undo on empty stack
	snapshot, err := stack.Undo()
	if err == nil {
		t.Error("expected error on empty stack undo, got nil")
	}

	if snapshot != nil {
		t.Error("expected nil snapshot on empty stack undo")
	}
}

func TestUndoStack_UndoAtBeginning(t *testing.T) {
	stack := NewUndoStack(5)

	// Push one snapshot
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)
	stack.Push(wf, map[string]Position{start.GetID(): {X: 10, Y: 20}})

	// Undo once (move to position -1, which returns nil snapshot)
	snapshot, err := stack.Undo()
	if err != nil {
		t.Fatalf("first undo failed: %v", err)
	}

	// Snapshot should be nil (before first snapshot)
	if snapshot != nil {
		t.Error("expected nil snapshot when undoing to before first snapshot")
	}

	// Try to undo again (already at beginning, cursor at -1)
	snapshot2, err := stack.Undo()
	if err == nil {
		t.Error("expected error when undoing past beginning")
	}

	if snapshot2 != nil {
		t.Error("expected nil snapshot when undoing past beginning")
	}
}

func TestUndoStack_Redo(t *testing.T) {
	stack := NewUndoStack(5)

	// Push three snapshots
	for i := 0; i < 3; i++ {
		wf, _ := workflow.NewWorkflow("test", "test")
		start := &workflow.StartNode{ID: "start-node-1"}
		wf.AddNode(start)
		positions := map[string]Position{start.GetID(): {X: i * 10, Y: i * 20}}
		stack.Push(wf, positions)
	}

	// Undo twice
	stack.Undo()
	stack.Undo()

	if stack.cursor != 0 {
		t.Errorf("expected cursor 0 after two undos, got %d", stack.cursor)
	}

	// Redo once
	snapshot, err := stack.Redo()
	if err != nil {
		t.Fatalf("Redo failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Redo returned nil snapshot")
	}

	// Should return snapshot at cursor+1 (index 1)
	pos := snapshot.CanvasState[snapshot.Nodes[0].GetID()]
	if pos.X != 10 { // iteration 1: X = 1 * 10 = 10
		t.Errorf("expected X=10, got X=%d", pos.X)
	}

	// Cursor should move forward
	if stack.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", stack.cursor)
	}
}

func TestUndoStack_RedoEmpty(t *testing.T) {
	stack := NewUndoStack(5)

	// Try to redo on empty stack
	snapshot, err := stack.Redo()
	if err == nil {
		t.Error("expected error on empty stack redo, got nil")
	}

	if snapshot != nil {
		t.Error("expected nil snapshot on empty stack redo")
	}
}

func TestUndoStack_RedoAtEnd(t *testing.T) {
	stack := NewUndoStack(5)

	// Push one snapshot
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)
	stack.Push(wf, map[string]Position{start.GetID(): {X: 10, Y: 20}})

	// Try to redo when already at end
	snapshot, err := stack.Redo()
	if err == nil {
		t.Error("expected error when redoing at end")
	}

	if snapshot != nil {
		t.Error("expected nil snapshot when redoing at end")
	}
}

func TestUndoStack_PushClearsRedoStack(t *testing.T) {
	stack := NewUndoStack(5)

	// Push three snapshots
	for i := 0; i < 3; i++ {
		wf, _ := workflow.NewWorkflow("test", "test")
		start := &workflow.StartNode{ID: "start-node-1"}
		wf.AddNode(start)
		positions := map[string]Position{start.GetID(): {X: i * 10, Y: i * 20}}
		stack.Push(wf, positions)
	}

	// Undo twice
	stack.Undo()
	stack.Undo()

	if stack.cursor != 0 {
		t.Errorf("expected cursor 0 after undos, got %d", stack.cursor)
	}

	// CanRedo should be true
	if !stack.CanRedo() {
		t.Error("expected CanRedo=true after undos")
	}

	// Push new snapshot (should clear redo stack)
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)
	stack.Push(wf, map[string]Position{start.GetID(): {X: 99, Y: 99}})

	// Redo should no longer be available
	if stack.CanRedo() {
		t.Error("expected CanRedo=false after push")
	}

	// Should only have snapshots up to cursor + new one
	if len(stack.snapshots) != 2 { // snapshots[0] and new one
		t.Errorf("expected 2 snapshots after push clears redo, got %d", len(stack.snapshots))
	}

	// Cursor should be at end
	if stack.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", stack.cursor)
	}
}

func TestUndoStack_CanUndo(t *testing.T) {
	stack := NewUndoStack(5)

	// Empty stack
	if stack.CanUndo() {
		t.Error("expected CanUndo=false on empty stack")
	}

	// Push one snapshot
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)
	stack.Push(wf, map[string]Position{start.GetID(): {X: 10, Y: 20}})

	// Should be able to undo
	if !stack.CanUndo() {
		t.Error("expected CanUndo=true after push")
	}

	// Undo once
	stack.Undo()

	// Should not be able to undo anymore
	if stack.CanUndo() {
		t.Error("expected CanUndo=false after undoing to beginning")
	}
}

func TestUndoStack_CanRedo(t *testing.T) {
	stack := NewUndoStack(5)

	// Empty stack
	if stack.CanRedo() {
		t.Error("expected CanRedo=false on empty stack")
	}

	// Push one snapshot
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)
	stack.Push(wf, map[string]Position{start.GetID(): {X: 10, Y: 20}})

	// Should not be able to redo (at end)
	if stack.CanRedo() {
		t.Error("expected CanRedo=false when at end")
	}

	// Undo once
	stack.Undo()

	// Should be able to redo
	if !stack.CanRedo() {
		t.Error("expected CanRedo=true after undo")
	}

	// Redo
	stack.Redo()

	// Should not be able to redo anymore
	if stack.CanRedo() {
		t.Error("expected CanRedo=false after redoing to end")
	}
}

func TestUndoStack_DeepCopy(t *testing.T) {
	stack := NewUndoStack(5)

	// Create workflow with nodes and edges
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	end := &workflow.EndNode{ID: "end-node-1"}
	wf.AddNode(start)
	wf.AddNode(end)

	edge := &workflow.Edge{
		FromNodeID: start.GetID(),
		ToNodeID:   end.GetID(),
	}
	wf.AddEdge(edge)

	positions := map[string]Position{
		start.GetID(): {X: 10, Y: 20},
		end.GetID():   {X: 30, Y: 40},
	}

	// Push snapshot
	err := stack.Push(wf, positions)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Modify original workflow
	newNode := &workflow.StartNode{ID: "start-node-new"}
	wf.AddNode(newNode)
	positions[newNode.GetID()] = Position{X: 50, Y: 60}

	// Snapshot should not be affected
	snapshot := stack.snapshots[0]
	if len(snapshot.Nodes) != 2 {
		t.Errorf("snapshot nodes affected by mutation: expected 2, got %d", len(snapshot.Nodes))
	}

	if len(snapshot.CanvasState) != 2 {
		t.Errorf("snapshot positions affected by mutation: expected 2, got %d", len(snapshot.CanvasState))
	}
}

func TestUndoStack_TimestampOrder(t *testing.T) {
	stack := NewUndoStack(5)

	// Push snapshots with delays
	wf, _ := workflow.NewWorkflow("test", "test")
	start := &workflow.StartNode{ID: "start-node-1"}
	wf.AddNode(start)

	stack.Push(wf, map[string]Position{start.GetID(): {X: 10, Y: 20}})
	time.Sleep(10 * time.Millisecond)

	stack.Push(wf, map[string]Position{start.GetID(): {X: 20, Y: 30}})
	time.Sleep(10 * time.Millisecond)

	stack.Push(wf, map[string]Position{start.GetID(): {X: 30, Y: 40}})

	// Timestamps should be in order
	for i := 0; i < len(stack.snapshots)-1; i++ {
		if !stack.snapshots[i].Timestamp.Before(stack.snapshots[i+1].Timestamp) {
			t.Errorf("timestamps not in order: %v >= %v",
				stack.snapshots[i].Timestamp,
				stack.snapshots[i+1].Timestamp)
		}
	}
}
