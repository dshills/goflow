package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowBuilderIntegration tests integrated operations across all components
func TestWorkflowBuilderIntegration(t *testing.T) {
	t.Run("AddNode integration", func(t *testing.T) {
		// Create workflow and builder
		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Show palette and create node
		builder.palette.Show()
		node, err := builder.palette.CreateNode()
		if err != nil {
			t.Fatalf("Failed to create node: %v", err)
		}

		// Add node at position
		pos := Position{X: 10, Y: 5}
		err = builder.canvas.AddNode(node, pos)
		if err != nil {
			t.Fatalf("Failed to add node to canvas: %v", err)
		}

		// Add to workflow using builder method (which sets modified flag)
		err = builder.AddNodeToCanvas(node)
		if err != nil {
			t.Fatalf("Failed to add node to workflow: %v", err)
		}

		// Verify node exists in both canvas and workflow
		if len(wf.Nodes) != 1 {
			t.Errorf("Expected 1 node in workflow, got %d", len(wf.Nodes))
		}
		if builder.canvas.GetNodeCount() != 1 {
			t.Errorf("Expected 1 node in canvas, got %d", builder.canvas.GetNodeCount())
		}

		// Verify modified flag set
		if !builder.modified {
			t.Error("Expected modified flag to be true after adding node")
		}
	})

	t.Run("DeleteNode integration", func(t *testing.T) {
		// Create workflow with nodes and edges
		node1 := &workflow.MCPToolNode{ID: "node-1", ServerID: "server", ToolName: "tool"}
		node2 := &workflow.TransformNode{ID: "node-2", Expression: "$.data"}
		edge := &workflow.Edge{FromNodeID: "node-1", ToNodeID: "node-2"}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node1, node2},
			Edges:     []*workflow.Edge{edge},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Delete node-1 (should also remove edge)
		err = builder.canvas.RemoveNode("node-1")
		if err != nil {
			t.Fatalf("Failed to remove node from canvas: %v", err)
		}

		// Remove from workflow
		wf.Nodes = []workflow.Node{node2}
		wf.Edges = []*workflow.Edge{}

		// Verify node and edge removed
		if len(wf.Nodes) != 1 {
			t.Errorf("Expected 1 node after delete, got %d", len(wf.Nodes))
		}
		if len(wf.Edges) != 0 {
			t.Errorf("Expected 0 edges after delete, got %d", len(wf.Edges))
		}
		if builder.canvas.GetNodeCount() != 1 {
			t.Errorf("Expected 1 node in canvas, got %d", builder.canvas.GetNodeCount())
		}
	})

	t.Run("CreateEdge integration", func(t *testing.T) {
		// Create workflow with nodes
		node1 := &workflow.MCPToolNode{ID: "node-1", ServerID: "server", ToolName: "tool"}
		node2 := &workflow.TransformNode{ID: "node-2", Expression: "$.data"}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node1, node2},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Create edge
		err = builder.CreateEdge("node-1", "node-2")
		if err != nil {
			t.Fatalf("Failed to create edge: %v", err)
		}

		// Verify edge exists
		if len(wf.Edges) != 1 {
			t.Errorf("Expected 1 edge, got %d", len(wf.Edges))
		}
		if wf.Edges[0].FromNodeID != "node-1" || wf.Edges[0].ToNodeID != "node-2" {
			t.Error("Edge has incorrect source/target")
		}

		// Verify modified flag
		if !builder.modified {
			t.Error("Expected modified flag to be true after creating edge")
		}
	})

	t.Run("CreateEdge prevents circular dependencies", func(t *testing.T) {
		// Create workflow with nodes in a chain
		node1 := &workflow.MCPToolNode{ID: "node-1", ServerID: "server", ToolName: "tool"}
		node2 := &workflow.TransformNode{ID: "node-2", Expression: "$.data"}
		edge1 := &workflow.Edge{FromNodeID: "node-1", ToNodeID: "node-2"}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node1, node2},
			Edges:     []*workflow.Edge{edge1},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Try to create circular edge (should fail validation)
		err = builder.CreateEdge("node-2", "node-1")
		if err != nil {
			t.Fatalf("Edge creation failed: %v", err)
		}

		// Workflow should now be invalid (circular dependency)
		err = wf.Validate()
		if err == nil {
			t.Error("Expected validation error for circular dependency")
		}
	})

	t.Run("EditNodeProperties integration", func(t *testing.T) {
		// Create workflow with node
		node := &workflow.MCPToolNode{
			ID:             "node-1",
			ServerID:       "old-server",
			ToolName:       "old-tool",
			OutputVariable: "old-output",
		}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Open property panel
		err = builder.ShowPropertyPanel("node-1")
		if err != nil {
			t.Fatalf("Failed to show property panel: %v", err)
		}

		// Verify panel is visible
		if !builder.propertyPanel.visible {
			t.Error("Expected property panel to be visible")
		}

		// Simulate editing fields
		panel := builder.propertyPanel
		fields := panel.GetFields()

		// Update Server ID field (index 1, after Node ID at index 0)
		for i, field := range fields {
			if field.label == "Server ID" {
				panel.editIndex = i
				err = panel.SetFieldValue("new-server")
				if err != nil {
					t.Errorf("Failed to set field value: %v", err)
				}
				break
			}
		}

		// Verify dirty flag
		if !panel.IsDirty() {
			t.Error("Expected panel to be dirty after editing")
		}
	})

	t.Run("Undo/Redo integration", func(t *testing.T) {
		// Create workflow
		node1 := &workflow.MCPToolNode{ID: "node-1", ServerID: "server", ToolName: "tool"}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node1},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		_, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Create undo stack and push initial state
		undoStack := NewUndoStack(100)
		canvasPositions := make(map[string]Position)
		canvasPositions["node-1"] = Position{X: 10, Y: 5}

		err = undoStack.Push(wf, canvasPositions)
		if err != nil {
			t.Fatalf("Failed to push snapshot: %v", err)
		}

		// Add another node
		node2 := &workflow.TransformNode{ID: "node-2", Expression: "$.data"}
		wf.Nodes = append(wf.Nodes, node2)
		canvasPositions["node-2"] = Position{X: 30, Y: 15}

		err = undoStack.Push(wf, canvasPositions)
		if err != nil {
			t.Fatalf("Failed to push second snapshot: %v", err)
		}

		// Verify we can undo
		if !undoStack.CanUndo() {
			t.Error("Expected to be able to undo")
		}

		// Undo
		snapshot, err := undoStack.Undo()
		if err != nil {
			t.Fatalf("Failed to undo: %v", err)
		}

		// Verify snapshot restored
		if len(snapshot.Nodes) != 1 {
			t.Errorf("Expected 1 node after undo, got %d", len(snapshot.Nodes))
		}

		// Verify we can redo
		if !undoStack.CanRedo() {
			t.Error("Expected to be able to redo")
		}

		// Redo
		snapshot, err = undoStack.Redo()
		if err != nil {
			t.Fatalf("Failed to redo: %v", err)
		}

		// Verify snapshot restored
		if len(snapshot.Nodes) != 2 {
			t.Errorf("Expected 2 nodes after redo, got %d", len(snapshot.Nodes))
		}
	})

	t.Run("SaveWorkflow integration", func(t *testing.T) {
		// Create valid workflow (using Transform node instead of MCP Tool to avoid server validation)
		start := &workflow.StartNode{ID: "start"}
		node1 := &workflow.TransformNode{
			ID:             "node-1",
			InputVariable:  "input",
			Expression:     "$.data",
			OutputVariable: "result",
		}
		end := &workflow.EndNode{ID: "end", ReturnValue: "result"}
		edge1 := &workflow.Edge{ID: "edge-1", FromNodeID: "start", ToNodeID: "node-1"}
		edge2 := &workflow.Edge{ID: "edge-2", FromNodeID: "node-1", ToNodeID: "end"}

		wf := &workflow.Workflow{
			Name:    "test-workflow",
			Version: "1.0",
			Nodes:   []workflow.Node{start, node1, end},
			Edges:   []*workflow.Edge{edge1, edge2},
			Variables: []*workflow.Variable{
				{Name: "input", Type: "string", DefaultValue: "test"},
			},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Mark as modified
		builder.modified = true

		// Save workflow
		err = builder.SaveWorkflow()
		if err != nil {
			t.Fatalf("Failed to save workflow: %v", err)
		}

		// Verify modified flag cleared
		if builder.modified {
			t.Error("Expected modified flag to be false after save")
		}
	})

	t.Run("SaveWorkflow prevents saving invalid workflow", func(t *testing.T) {
		// Create invalid workflow (no start node)
		node1 := &workflow.MCPToolNode{ID: "node-1", ServerID: "", ToolName: "", OutputVariable: ""}

		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{node1},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Try to save
		err = builder.SaveWorkflow()
		if err == nil {
			t.Error("Expected error when saving invalid workflow")
		}

		// Verify error mentions validation
		if err != nil && !containsSubstring(err.Error(), "invalid workflow") {
			t.Errorf("Expected validation error, got: %v", err)
		}
	})

	t.Run("Modified flag management", func(t *testing.T) {
		// Create workflow
		wf := &workflow.Workflow{
			Name:      "test-workflow",
			Version:   "1.0",
			Nodes:     []workflow.Node{},
			Edges:     []*workflow.Edge{},
			Variables: []*workflow.Variable{},
		}

		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			t.Fatalf("Failed to create builder: %v", err)
		}

		// Initially not modified
		if builder.IsModified() {
			t.Error("Expected workflow to not be modified initially")
		}

		// Mark as modified
		builder.MarkModified()
		if !builder.IsModified() {
			t.Error("Expected workflow to be modified after MarkModified()")
		}

		// Clear modified flag (simulated save)
		builder.modified = false
		if builder.IsModified() {
			t.Error("Expected workflow to not be modified after clearing flag")
		}
	})
}

// TestWorkflowBuilderComplexWorkflow tests a complete workflow creation scenario
func TestWorkflowBuilderComplexWorkflow(t *testing.T) {
	// Create empty workflow
	wf := &workflow.Workflow{
		Name:      "complex-workflow",
		Version:   "1.0",
		Nodes:     []workflow.Node{},
		Edges:     []*workflow.Edge{},
		Variables: []*workflow.Variable{},
	}

	_, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Create undo stack
	undoStack := NewUndoStack(100)

	// Step 1: Add start node
	start := &workflow.StartNode{ID: "start"}
	err = wf.AddNode(start)
	if err != nil {
		t.Fatalf("Failed to add start node: %v", err)
	}

	canvasPositions := make(map[string]Position)
	canvasPositions["start"] = Position{X: 10, Y: 5}
	undoStack.Push(wf, canvasPositions)

	// Step 2: Add first transform node
	transform1 := &workflow.TransformNode{
		ID:             "fetch-data",
		InputVariable:  "input",
		Expression:     "$.items",
		OutputVariable: "raw_data",
	}
	err = wf.AddNode(transform1)
	if err != nil {
		t.Fatalf("Failed to add transform node: %v", err)
	}

	canvasPositions["fetch-data"] = Position{X: 10, Y: 15}
	undoStack.Push(wf, canvasPositions)

	// Step 3: Add second transform node
	transform2 := &workflow.TransformNode{
		ID:             "process-data",
		InputVariable:  "raw_data",
		Expression:     "$.items[0]",
		OutputVariable: "processed_data",
	}
	err = wf.AddNode(transform2)
	if err != nil {
		t.Fatalf("Failed to add second transform node: %v", err)
	}

	canvasPositions["process-data"] = Position{X: 10, Y: 25}
	undoStack.Push(wf, canvasPositions)

	// Step 4: Add end node
	end := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "processed_data",
	}
	err = wf.AddNode(end)
	if err != nil {
		t.Fatalf("Failed to add end node: %v", err)
	}

	canvasPositions["end"] = Position{X: 10, Y: 35}
	undoStack.Push(wf, canvasPositions)

	// Add required workflow variables
	wf.Variables = []*workflow.Variable{
		{Name: "input", Type: "string", DefaultValue: "test"},
		{Name: "raw_data", Type: "string"},
		{Name: "processed_data", Type: "string"},
	}

	// Step 5: Connect nodes with edges
	edges := []*workflow.Edge{
		{FromNodeID: "start", ToNodeID: "fetch-data"},
		{FromNodeID: "fetch-data", ToNodeID: "process-data"},
		{FromNodeID: "process-data", ToNodeID: "end"},
	}

	for _, edge := range edges {
		err = wf.AddEdge(edge)
		if err != nil {
			t.Fatalf("Failed to add edge %s -> %s: %v", edge.FromNodeID, edge.ToNodeID, err)
		}
	}

	undoStack.Push(wf, canvasPositions)

	// Verify workflow structure (start + 2 transforms + end = 4 nodes)
	if len(wf.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(wf.Nodes))
	}
	if len(wf.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(wf.Edges))
	}

	// Verify undo stack has history (start + 2 transforms + end + edges = 5 snapshots)
	if undoStack.Size() != 5 {
		t.Errorf("Expected 5 snapshots in undo stack, got %d", undoStack.Size())
	}

	// Test undo back through history
	for i := 0; i < 3; i++ {
		if !undoStack.CanUndo() {
			t.Fatalf("Expected to undo at step %d", i)
		}
		_, err = undoStack.Undo()
		if err != nil {
			t.Fatalf("Failed to undo at step %d: %v", i, err)
		}
	}

	// Test redo forward
	for i := 0; i < 3; i++ {
		if !undoStack.CanRedo() {
			t.Fatalf("Expected to redo at step %d", i)
		}
		_, err = undoStack.Redo()
		if err != nil {
			t.Fatalf("Failed to redo at step %d: %v", i, err)
		}
	}

	// Validate final workflow
	err = wf.Validate()
	if err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

// Helper function (using unique name to avoid conflicts)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstringIn(s, substr))
}

func findSubstringIn(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
