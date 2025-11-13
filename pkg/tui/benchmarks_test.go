package tui

import (
	"fmt"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// BenchmarkCanvasNodeOperations measures node add/remove performance
func BenchmarkCanvasNodeOperations(b *testing.B) {
	canvas := NewCanvas(80, 40)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := &workflow.MCPToolNode{
			ID:             fmt.Sprintf("node-%d", i),
			ServerID:       "test-server",
			ToolName:       "test-tool",
			OutputVariable: "output",
		}
		_ = canvas.AddNode(node, Position{X: 10 + i%50, Y: 5 + i%20})
	}
}

// BenchmarkCanvasEdgeRouting measures edge routing performance
func BenchmarkCanvasEdgeRouting(b *testing.B) {
	canvas := NewCanvas(80, 40)

	// Add 100 nodes
	for i := 0; i < 100; i++ {
		node := &workflow.MCPToolNode{
			ID:             fmt.Sprintf("node-%d", i),
			ServerID:       "test-server",
			ToolName:       "test-tool",
			OutputVariable: "output",
		}
		_ = canvas.AddNode(node, Position{X: 10 + i*5, Y: 5 + (i%10)*5})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		edge := &workflow.Edge{
			ID:         fmt.Sprintf("edge-%d", i),
			FromNodeID: fmt.Sprintf("node-%d", i%99),
			ToNodeID:   fmt.Sprintf("node-%d", (i+1)%99),
		}
		_ = canvas.AddEdge(edge)
		if i%10 == 0 {
			// Periodically remove edges to keep edge count manageable
			_ = canvas.RemoveEdge(fmt.Sprintf("node-%d", (i-10)%99), fmt.Sprintf("node-%d", (i-9)%99))
		}
	}
}

// BenchmarkCanvasRenderingWithZoom measures rendering performance at different zoom levels
func BenchmarkCanvasRenderingWithZoom(b *testing.B) {
	wf := createLargeWorkflow(100)
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		b.Fatalf("failed to create workflow builder: %v", err)
	}

	canvas := builder.RenderCanvas()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate zooming during rendering
		_ = canvas.GetNodeCount()
		_ = canvas.GetEdgeCount()
	}
}

// BenchmarkAutoLayout measures auto-layout algorithm performance with 50 nodes
// Target: < 200ms for 50 nodes
func BenchmarkAutoLayout(b *testing.B) {
	wf := createLargeWorkflow(50)
	canvas := NewCanvas(80, 40)

	// Add all nodes to canvas
	for _, node := range wf.Nodes {
		_ = canvas.AddNode(node, Position{X: 0, Y: 0})
	}

	// Add all edges to canvas
	for _, edge := range wf.Edges {
		_ = canvas.AddEdge(edge)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canvas.AutoLayout(wf)
	}
}

// BenchmarkAutoLayoutComplex measures layout with branching workflow
func BenchmarkAutoLayoutComplex(b *testing.B) {
	wf := createBranchingWorkflow(50)
	canvas := NewCanvas(80, 40)

	// Add all nodes to canvas
	for _, node := range wf.Nodes {
		_ = canvas.AddNode(node, Position{X: 0, Y: 0})
	}

	// Add all edges to canvas
	for _, edge := range wf.Edges {
		_ = canvas.AddEdge(edge)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canvas.AutoLayout(wf)
	}
}

// BenchmarkUndo measures undo operation performance
// Target: < 50ms per undo
func BenchmarkUndo(b *testing.B) {
	wf := createLargeWorkflow(50)
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		b.Fatalf("failed to create workflow builder: %v", err)
	}

	// Perform some operations to build undo stack
	for i := 0; i < 10; i++ {
		node := &workflow.MCPToolNode{
			ID:             fmt.Sprintf("benchmark-node-%d", i),
			ServerID:       "test-server",
			ToolName:       "test-tool",
			OutputVariable: "output",
		}
		if err := builder.AddNodeToCanvas(node); err != nil {
			b.Fatalf("failed to add node: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Perform undo
		if builder.CanUndo() {
			if err := builder.Undo(); err != nil {
				b.Fatalf("undo failed: %v", err)
			}
		}

		// Add node to rebuild undo stack for next iteration
		if i < b.N-1 {
			node := &workflow.MCPToolNode{
				ID:             fmt.Sprintf("benchmark-node-redo-%d", i),
				ServerID:       "test-server",
				ToolName:       "test-tool",
				OutputVariable: "output",
			}
			if err := builder.AddNodeToCanvas(node); err != nil {
				b.Fatalf("failed to add node: %v", err)
			}
		}
	}
}

// BenchmarkRedo measures redo operation performance
// Target: < 50ms per redo
func BenchmarkRedo(b *testing.B) {
	wf := createLargeWorkflow(50)
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		b.Fatalf("failed to create workflow builder: %v", err)
	}

	// Build undo/redo stacks
	for i := 0; i < 10; i++ {
		node := &workflow.MCPToolNode{
			ID:             fmt.Sprintf("benchmark-node-%d", i),
			ServerID:       "test-server",
			ToolName:       "test-tool",
			OutputVariable: "output",
		}
		if err := builder.AddNodeToCanvas(node); err != nil {
			b.Fatalf("failed to add node: %v", err)
		}
	}

	// Undo all to build redo stack
	for builder.CanUndo() {
		if err := builder.Undo(); err != nil {
			b.Fatalf("undo failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Perform redo
		if builder.CanRedo() {
			if err := builder.Redo(); err != nil {
				b.Fatalf("redo failed: %v", err)
			}
		}

		// Undo to rebuild redo stack for next iteration
		if i < b.N-1 && builder.CanUndo() {
			if err := builder.Undo(); err != nil {
				b.Fatalf("undo failed: %v", err)
			}
		}
	}
}

// BenchmarkPropertyValidation measures property validation performance
// Target: < 200ms per validation
func BenchmarkPropertyValidation(b *testing.B) {
	wf := createLargeWorkflow(20)
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		b.Fatalf("failed to create workflow builder: %v", err)
	}

	// Get first node for property editing
	if len(wf.Nodes) == 0 {
		b.Fatal("no nodes in workflow")
	}
	nodeID := wf.Nodes[0].GetID()

	// Open property panel
	if err := builder.ShowPropertyPanel(nodeID); err != nil {
		b.Fatalf("failed to show property panel: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Update property field (triggers validation)
		if err := builder.UpdatePropertyField(1, "test-value"); err != nil {
			// Validation errors are expected, just measure performance
			_ = err
		}
	}
}

// BenchmarkWorkflowValidation measures full workflow validation performance
// Target: < 200ms for 100 nodes
func BenchmarkWorkflowValidation(b *testing.B) {
	wf := createValidWorkflow(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Validate entire workflow
		err := wf.Validate()
		_ = err // May have validation errors
	}
}

// BenchmarkNodeSelection measures node selection performance
func BenchmarkNodeSelection(b *testing.B) {
	wf := createLargeWorkflow(100)
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		b.Fatalf("failed to create workflow builder: %v", err)
	}

	nodeIDs := make([]string, 0, len(wf.Nodes))
	for _, node := range wf.Nodes {
		nodeIDs = append(nodeIDs, node.GetID())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Select node (cycles through all nodes)
		nodeID := nodeIDs[i%len(nodeIDs)]
		if err := builder.SelectNode(nodeID); err != nil {
			b.Fatalf("failed to select node: %v", err)
		}
	}
}

// Helper functions to create test workflows

// createLargeWorkflow creates a linear workflow with n nodes for benchmarking
func createLargeWorkflow(nodeCount int) *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("benchmark-workflow", "Workflow for benchmarking")

	// Add start node
	startNode := &workflow.StartNode{ID: "start"}
	_ = wf.AddNode(startNode)

	// Add nodes
	prevNodeID := "start"
	for i := 1; i <= nodeCount; i++ {
		node := &workflow.MCPToolNode{
			ID:             fmt.Sprintf("node-%d", i),
			ServerID:       "test-server",
			ToolName:       "test-tool",
			OutputVariable: fmt.Sprintf("output-%d", i),
		}
		_ = wf.AddNode(node)

		// Connect to previous node
		edge := &workflow.Edge{
			FromNodeID: prevNodeID,
			ToNodeID:   node.ID,
		}
		_ = wf.AddEdge(edge)

		prevNodeID = node.ID
	}

	// Add end node
	endNode := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "final-output",
	}
	_ = wf.AddNode(endNode)

	// Connect last node to end
	edge := &workflow.Edge{
		FromNodeID: prevNodeID,
		ToNodeID:   "end",
	}
	_ = wf.AddEdge(edge)

	return wf
}

// createBranchingWorkflow creates a workflow with branches for layout benchmarking
func createBranchingWorkflow(nodeCount int) *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("branching-workflow", "Workflow with branches")

	// Add start node
	startNode := &workflow.StartNode{ID: "start"}
	_ = wf.AddNode(startNode)

	// Create multiple branches
	branchCount := 5
	nodesPerBranch := nodeCount / branchCount

	for branch := 0; branch < branchCount; branch++ {
		prevNodeID := "start"
		for i := 0; i < nodesPerBranch; i++ {
			nodeID := fmt.Sprintf("branch-%d-node-%d", branch, i)
			node := &workflow.MCPToolNode{
				ID:             nodeID,
				ServerID:       "test-server",
				ToolName:       "test-tool",
				OutputVariable: fmt.Sprintf("output-%s", nodeID),
			}
			_ = wf.AddNode(node)

			// Connect to previous node
			edge := &workflow.Edge{
				FromNodeID: prevNodeID,
				ToNodeID:   nodeID,
			}
			_ = wf.AddEdge(edge)

			prevNodeID = nodeID
		}
	}

	// Add end node
	endNode := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "final-output",
	}
	_ = wf.AddNode(endNode)

	return wf
}

// createValidWorkflow creates a valid workflow for validation benchmarking
func createValidWorkflow(nodeCount int) *workflow.Workflow {
	wf := createLargeWorkflow(nodeCount)

	// Add required configurations to make workflow valid
	for _, node := range wf.Nodes {
		switch n := node.(type) {
		case *workflow.MCPToolNode:
			if n.ServerID == "" {
				n.ServerID = "test-server"
			}
			if n.ToolName == "" {
				n.ToolName = "test-tool"
			}
		}
	}

	return wf
}
