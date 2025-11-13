package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestKeyboardEventSimulator verifies the keyboard event simulator
func TestKeyboardEventSimulator(t *testing.T) {
	events := []string{"a", "b", "c"}
	sim := NewKeyboardEventSimulator(events)

	// Test HasMore and NextKey
	if !sim.HasMore() {
		t.Error("expected HasMore to be true")
	}

	if sim.RemainingCount() != 3 {
		t.Errorf("expected RemainingCount=3, got %d", sim.RemainingCount())
	}

	// Consume events
	if key := sim.NextKey(); key != "a" {
		t.Errorf("expected 'a', got '%s'", key)
	}

	if key := sim.NextKey(); key != "b" {
		t.Errorf("expected 'b', got '%s'", key)
	}

	if sim.RemainingCount() != 1 {
		t.Errorf("expected RemainingCount=1, got %d", sim.RemainingCount())
	}

	if key := sim.NextKey(); key != "c" {
		t.Errorf("expected 'c', got '%s'", key)
	}

	// Should be exhausted
	if sim.HasMore() {
		t.Error("expected HasMore to be false")
	}

	if key := sim.NextKey(); key != "" {
		t.Errorf("expected empty string, got '%s'", key)
	}

	// Test Reset
	sim.Reset()
	if !sim.HasMore() {
		t.Error("expected HasMore to be true after reset")
	}

	if key := sim.NextKey(); key != "a" {
		t.Errorf("expected 'a' after reset, got '%s'", key)
	}
}

// TestScreenCapture verifies the screen capture utility
func TestScreenCapture(t *testing.T) {
	capture := NewScreenCapture(80, 24)

	if len(capture.GetLines()) != 0 {
		t.Error("expected empty capture initially")
	}

	// Add some lines
	capture.lines = append(capture.lines, "Line 1")
	capture.lines = append(capture.lines, "Line 2 contains test")
	capture.lines = append(capture.lines, "Line 3")

	// Test GetLine
	if line := capture.GetLine(0); line != "Line 1" {
		t.Errorf("expected 'Line 1', got '%s'", line)
	}

	if line := capture.GetLine(5); line != "" {
		t.Errorf("expected empty string for out of bounds, got '%s'", line)
	}

	// Test Contains
	if !capture.Contains("test") {
		t.Error("expected capture to contain 'test'")
	}

	if capture.Contains("notfound") {
		t.Error("expected capture to not contain 'notfound'")
	}

	// Test GetLines
	lines := capture.GetLines()
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// Test Clear
	capture.Clear()
	if len(capture.GetLines()) != 0 {
		t.Error("expected empty capture after clear")
	}
}

// TestWorkflowBuilderTestHarness verifies the test harness functionality
func TestWorkflowBuilderTestHarness(t *testing.T) {
	// Create a simple workflow
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow")
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	// Add start node
	startNode := &workflow.StartNode{ID: "start"}
	if err := wf.AddNode(startNode); err != nil {
		t.Fatalf("failed to add start node: %v", err)
	}

	// Add end node
	endNode := &workflow.EndNode{ID: "end"}
	if err := wf.AddNode(endNode); err != nil {
		t.Fatalf("failed to add end node: %v", err)
	}

	// Create test harness
	harness := NewWorkflowBuilderTestHarness(t, wf)

	// Test assertion methods
	harness.AssertNodeCount(2)
	harness.AssertEdgeCount(0)
	harness.AssertMode("normal") // Initial mode is "normal"
	harness.AssertModified(false)
	harness.AssertCanUndo(false)
	harness.AssertCanRedo(false)

	// Add a node via harness
	newNode := &workflow.MCPToolNode{
		ID:             "test-node",
		ServerID:       "test-server",
		ToolName:       "test-tool",
		OutputVariable: "output",
	}

	if err := harness.AddNode(newNode); err != nil {
		t.Fatalf("failed to add node via harness: %v", err)
	}

	// Verify state changes
	harness.AssertNodeCount(3)
	harness.AssertModified(true)
	harness.AssertCanUndo(true)

	// Test keyboard simulation with valid keys
	// "?" toggles help, "Esc" returns to normal mode
	err = harness.SimulateKeySequence([]string{"?", "Esc"})
	if err != nil {
		t.Fatalf("failed to simulate key sequence: %v", err)
	}

	// Mode should be back to normal after "?" then "Esc"
	harness.AssertMode("normal")

	// Test GetBuilder, GetWorkflow, GetRepository
	if harness.GetBuilder() == nil {
		t.Error("expected builder to be non-nil")
	}

	if harness.GetWorkflow() == nil {
		t.Error("expected workflow to be non-nil")
	}

	if harness.GetRepository() == nil {
		t.Error("expected repository to be non-nil")
	}
}

// TestMockRepositoryIntegration verifies mock repository works with test harness
func TestMockRepositoryIntegration(t *testing.T) {
	wf, err := workflow.NewWorkflow("save-test", "Workflow to save")
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	// Add required nodes for valid workflow
	startNode := &workflow.StartNode{ID: "start"}
	if err := wf.AddNode(startNode); err != nil {
		t.Fatalf("failed to add start node: %v", err)
	}

	endNode := &workflow.EndNode{ID: "end"}
	if err := wf.AddNode(endNode); err != nil {
		t.Fatalf("failed to add end node: %v", err)
	}

	// Add edge to make it valid
	edge := &workflow.Edge{
		FromNodeID: "start",
		ToNodeID:   "end",
	}
	if err := wf.AddEdge(edge); err != nil {
		t.Fatalf("failed to add edge: %v", err)
	}

	// Create harness
	harness := NewWorkflowBuilderTestHarness(t, wf)

	// Save workflow via harness
	if err := harness.SaveWorkflow(); err != nil {
		t.Fatalf("failed to save workflow: %v", err)
	}

	// Verify it was saved to mock repository
	repo := harness.GetRepository()

	if !repo.HasName("save-test") {
		t.Error("expected workflow to be saved in repository")
	}

	if repo.Count() != 1 {
		t.Errorf("expected repository count=1, got %d", repo.Count())
	}

	// Retrieve and verify
	retrieved, err := repo.FindByName("save-test")
	if err != nil {
		t.Fatalf("failed to retrieve workflow: %v", err)
	}

	if retrieved.Name != "save-test" {
		t.Errorf("expected name='save-test', got '%s'", retrieved.Name)
	}

	if len(retrieved.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(retrieved.Nodes))
	}
}
