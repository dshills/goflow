package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestAutoLayoutLinearWorkflow tests layout of a simple linear workflow
func TestAutoLayoutLinearWorkflow(t *testing.T) {
	canvas := NewCanvas(80, 40)
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow")
	if err != nil {
		t.Fatalf("NewWorkflow() error = %v", err)
	}

	// Create linear workflow: Start → Tool → End
	startNode := &workflow.StartNode{ID: "start"}
	toolNode := &workflow.MCPToolNode{ID: "tool-1", ServerID: "server-1", ToolName: "test", OutputVariable: "out"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(toolNode)
	wf.AddNode(endNode)

	// Add to canvas
	canvas.AddNode(startNode, Position{X: 0, Y: 0})
	canvas.AddNode(toolNode, Position{X: 0, Y: 0})
	canvas.AddNode(endNode, Position{X: 0, Y: 0})

	// Add edges
	edge1 := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "tool-1"}
	edge2 := &workflow.Edge{ID: "e2", FromNodeID: "tool-1", ToNodeID: "end"}
	wf.AddEdge(edge1)
	wf.AddEdge(edge2)

	canvas.AddEdge(edge1)
	canvas.AddEdge(edge2)

	// Run auto-layout
	canvas.AutoLayout(wf)

	// Verify nodes are positioned vertically in order
	startY := canvas.nodes["start"].position.Y
	toolY := canvas.nodes["tool-1"].position.Y
	endY := canvas.nodes["end"].position.Y

	if !(startY < toolY && toolY < endY) {
		t.Errorf("nodes not laid out vertically in order: start=%d, tool=%d, end=%d",
			startY, toolY, endY)
	}

	// Verify spacing between layers
	startBottom := startY + canvas.nodes["start"].height
	if toolY-startBottom < verticalSpacing {
		t.Errorf("insufficient vertical spacing between start and tool: %d", toolY-startBottom)
	}

	toolBottom := toolY + canvas.nodes["tool-1"].height
	if endY-toolBottom < verticalSpacing {
		t.Errorf("insufficient vertical spacing between tool and end: %d", endY-toolBottom)
	}
}

// TestAutoLayoutBranchingWorkflow tests layout with branching (condition node)
func TestAutoLayoutBranchingWorkflow(t *testing.T) {
	canvas := NewCanvas(80, 40)
	wf, err := workflow.NewWorkflow("branching-workflow", "Test branching")
	if err != nil {
		t.Fatalf("NewWorkflow() error = %v", err)
	}

	// Create branching workflow:
	//       Start
	//         |
	//     Condition
	//      /   \
	//   Tool1  Tool2
	//      \   /
	//       End

	startNode := &workflow.StartNode{ID: "start"}
	condNode := &workflow.ConditionNode{ID: "condition", Condition: "true"}
	tool1Node := &workflow.MCPToolNode{ID: "tool-1", ServerID: "s1", ToolName: "t1", OutputVariable: "v1"}
	tool2Node := &workflow.MCPToolNode{ID: "tool-2", ServerID: "s1", ToolName: "t2", OutputVariable: "v2"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(condNode)
	wf.AddNode(tool1Node)
	wf.AddNode(tool2Node)
	wf.AddNode(endNode)

	// Add to canvas
	canvas.AddNode(startNode, Position{X: 0, Y: 0})
	canvas.AddNode(condNode, Position{X: 0, Y: 0})
	canvas.AddNode(tool1Node, Position{X: 0, Y: 0})
	canvas.AddNode(tool2Node, Position{X: 0, Y: 0})
	canvas.AddNode(endNode, Position{X: 0, Y: 0})

	// Add edges
	e1 := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "condition"}
	e2 := &workflow.Edge{ID: "e2", FromNodeID: "condition", ToNodeID: "tool-1", Condition: "true"}
	e3 := &workflow.Edge{ID: "e3", FromNodeID: "condition", ToNodeID: "tool-2", Condition: "false"}
	e4 := &workflow.Edge{ID: "e4", FromNodeID: "tool-1", ToNodeID: "end"}
	e5 := &workflow.Edge{ID: "e5", FromNodeID: "tool-2", ToNodeID: "end"}

	wf.AddEdge(e1)
	wf.AddEdge(e2)
	wf.AddEdge(e3)
	wf.AddEdge(e4)
	wf.AddEdge(e5)

	canvas.AddEdge(e1)
	canvas.AddEdge(e2)
	canvas.AddEdge(e3)
	canvas.AddEdge(e4)
	canvas.AddEdge(e5)

	// Run auto-layout
	canvas.AutoLayout(wf)

	// Verify layer structure
	startY := canvas.nodes["start"].position.Y
	condY := canvas.nodes["condition"].position.Y
	tool1Y := canvas.nodes["tool-1"].position.Y
	tool2Y := canvas.nodes["tool-2"].position.Y
	endY := canvas.nodes["end"].position.Y

	// Start should be at top layer
	if startY >= condY {
		t.Errorf("start should be above condition: start=%d, cond=%d", startY, condY)
	}

	// Condition should be above both tools
	if condY >= tool1Y || condY >= tool2Y {
		t.Errorf("condition should be above tools: cond=%d, tool1=%d, tool2=%d",
			condY, tool1Y, tool2Y)
	}

	// Both tools should be at same Y level (same layer)
	if tool1Y != tool2Y {
		t.Errorf("parallel tools should be at same Y level: tool1=%d, tool2=%d", tool1Y, tool2Y)
	}

	// Tools should be at different X positions (side by side)
	tool1X := canvas.nodes["tool-1"].position.X
	tool2X := canvas.nodes["tool-2"].position.X
	if tool1X == tool2X {
		t.Errorf("parallel tools should have different X positions")
	}

	// End should be below tools
	if endY <= tool1Y {
		t.Errorf("end should be below tools: end=%d, tool1=%d", endY, tool1Y)
	}
}

// TestAutoLayoutParallelPaths tests layout with parallel execution paths
func TestAutoLayoutParallelPaths(t *testing.T) {
	canvas := NewCanvas(80, 40)
	wf, err := workflow.NewWorkflow("parallel-workflow", "Test parallel")
	if err != nil {
		t.Fatalf("NewWorkflow() error = %v", err)
	}

	// Create workflow with parallel paths:
	//      Start
	//       / \
	//   Tool1 Tool2
	//       \ /
	//       End

	startNode := &workflow.StartNode{ID: "start"}
	tool1Node := &workflow.MCPToolNode{ID: "tool-1", ServerID: "s1", ToolName: "t1", OutputVariable: "v1"}
	tool2Node := &workflow.MCPToolNode{ID: "tool-2", ServerID: "s1", ToolName: "t2", OutputVariable: "v2"}
	endNode := &workflow.EndNode{ID: "end"}

	wf.AddNode(startNode)
	wf.AddNode(tool1Node)
	wf.AddNode(tool2Node)
	wf.AddNode(endNode)

	canvas.AddNode(startNode, Position{X: 0, Y: 0})
	canvas.AddNode(tool1Node, Position{X: 0, Y: 0})
	canvas.AddNode(tool2Node, Position{X: 0, Y: 0})
	canvas.AddNode(endNode, Position{X: 0, Y: 0})

	// Add edges (parallel branches)
	e1 := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "tool-1"}
	e2 := &workflow.Edge{ID: "e2", FromNodeID: "start", ToNodeID: "tool-2"}
	e3 := &workflow.Edge{ID: "e3", FromNodeID: "tool-1", ToNodeID: "end"}
	e4 := &workflow.Edge{ID: "e4", FromNodeID: "tool-2", ToNodeID: "end"}

	wf.AddEdge(e1)
	wf.AddEdge(e2)
	wf.AddEdge(e3)
	wf.AddEdge(e4)

	canvas.AddEdge(e1)
	canvas.AddEdge(e2)
	canvas.AddEdge(e3)
	canvas.AddEdge(e4)

	// Run auto-layout
	canvas.AutoLayout(wf)

	// Verify parallel tools are on same layer
	tool1Y := canvas.nodes["tool-1"].position.Y
	tool2Y := canvas.nodes["tool-2"].position.Y

	if tool1Y != tool2Y {
		t.Errorf("parallel tools should be at same Y level: tool1=%d, tool2=%d", tool1Y, tool2Y)
	}

	// Verify they're separated horizontally
	tool1X := canvas.nodes["tool-1"].position.X
	tool2X := canvas.nodes["tool-2"].position.X

	if abs(tool1X-tool2X) < horizontalSpacing {
		t.Errorf("parallel tools should have horizontal spacing: tool1=%d, tool2=%d, spacing=%d",
			tool1X, tool2X, abs(tool1X-tool2X))
	}
}

// TestAutoLayoutDeterminism tests that layout is deterministic
func TestAutoLayoutDeterminism(t *testing.T) {
	// Create same workflow twice and verify layout is identical
	createWorkflow := func() (*Canvas, *workflow.Workflow) {
		canvas := NewCanvas(80, 40)
		wf, _ := workflow.NewWorkflow("test", "Test")

		start := &workflow.StartNode{ID: "start"}
		tool := &workflow.MCPToolNode{ID: "tool", ServerID: "s", ToolName: "t", OutputVariable: "v"}
		end := &workflow.EndNode{ID: "end"}

		wf.AddNode(start)
		wf.AddNode(tool)
		wf.AddNode(end)

		canvas.AddNode(start, Position{X: 0, Y: 0})
		canvas.AddNode(tool, Position{X: 0, Y: 0})
		canvas.AddNode(end, Position{X: 0, Y: 0})

		e1 := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "tool"}
		e2 := &workflow.Edge{ID: "e2", FromNodeID: "tool", ToNodeID: "end"}
		wf.AddEdge(e1)
		wf.AddEdge(e2)
		canvas.AddEdge(e1)
		canvas.AddEdge(e2)

		return canvas, wf
	}

	canvas1, wf1 := createWorkflow()
	canvas2, wf2 := createWorkflow()

	canvas1.AutoLayout(wf1)
	canvas2.AutoLayout(wf2)

	// Compare positions
	for nodeID := range canvas1.nodes {
		pos1 := canvas1.nodes[nodeID].position
		pos2 := canvas2.nodes[nodeID].position

		if pos1.X != pos2.X || pos1.Y != pos2.Y {
			t.Errorf("layout not deterministic for node %s: pos1=(%d,%d), pos2=(%d,%d)",
				nodeID, pos1.X, pos1.Y, pos2.X, pos2.Y)
		}
	}
}

// TestResetView tests viewport reset to start node
func TestResetView(t *testing.T) {
	canvas := NewCanvas(80, 40)
	wf, _ := workflow.NewWorkflow("test", "Test")

	start := &workflow.StartNode{ID: "start"}
	wf.AddNode(start)
	canvas.AddNode(start, Position{X: 100, Y: 50})

	// Set non-default viewport and zoom
	canvas.ViewportX = 20
	canvas.ViewportY = 15
	canvas.ZoomLevel = 1.5

	// Reset view
	canvas.ResetView()

	// Verify zoom is 100%
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("zoom not reset: got %f, want 1.0", canvas.ZoomLevel)
	}

	// Verify viewport is centered on start node center
	// Start node is 16 chars wide, 3 lines tall (from canvas.go calculateNodeSize)
	cNode := canvas.nodes["start"]
	startCenterX := cNode.position.X + cNode.width/2
	startCenterY := cNode.position.Y + cNode.height/2

	expectedViewportX := startCenterX - canvas.Width/2
	expectedViewportY := startCenterY - canvas.Height/2

	if abs(canvas.ViewportX-expectedViewportX) > 5 {
		t.Errorf("viewport X not centered on start: got %d, want ~%d",
			canvas.ViewportX, expectedViewportX)
	}
	if abs(canvas.ViewportY-expectedViewportY) > 5 {
		t.Errorf("viewport Y not centered on start: got %d, want ~%d",
			canvas.ViewportY, expectedViewportY)
	}
}

// TestFitAll tests fitting all nodes in viewport
func TestFitAll(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add nodes spread across large area
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}

	canvas.AddNode(node1, Position{X: 0, Y: 0})
	canvas.AddNode(node2, Position{X: 100, Y: 100})

	// Fit all nodes
	canvas.FitAll()

	// Verify zoom was adjusted (should be less than 1.0 to fit large content)
	if canvas.ZoomLevel >= 1.0 {
		t.Logf("Warning: zoom level might not be optimal: %f", canvas.ZoomLevel)
	}

	// Zoom should be in valid range
	if canvas.ZoomLevel < 0.5 || canvas.ZoomLevel > 2.0 {
		t.Errorf("zoom out of valid range: %f", canvas.ZoomLevel)
	}
}

// TestPan tests viewport panning
func TestPan(t *testing.T) {
	canvas := NewCanvas(80, 40)

	initialX := canvas.ViewportX
	initialY := canvas.ViewportY

	// Pan right and down
	canvas.Pan(10, 5)

	if canvas.ViewportX != initialX+10 {
		t.Errorf("ViewportX not updated: got %d, want %d", canvas.ViewportX, initialX+10)
	}

	if canvas.ViewportY != initialY+5 {
		t.Errorf("ViewportY not updated: got %d, want %d", canvas.ViewportY, initialY+5)
	}

	// Pan left and up (negative delta)
	canvas.Pan(-5, -3)

	if canvas.ViewportX != initialX+5 {
		t.Errorf("ViewportX after second pan: got %d, want %d", canvas.ViewportX, initialX+5)
	}

	if canvas.ViewportY != initialY+2 {
		t.Errorf("ViewportY after second pan: got %d, want %d", canvas.ViewportY, initialY+2)
	}
}

// TestZoom tests zoom level adjustment
func TestZoom(t *testing.T) {
	canvas := NewCanvas(80, 40)

	tests := []struct {
		name    string
		zoom    float64
		wantErr bool
	}{
		{"valid 100%", 1.0, false},
		{"valid 50%", 0.5, false},
		{"valid 200%", 2.0, false},
		{"valid 150%", 1.5, false},
		{"invalid too low", 0.3, true},
		{"invalid too high", 2.5, true},
		{"invalid negative", -1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.Zoom(tt.zoom)
			if (err != nil) != tt.wantErr {
				t.Errorf("Zoom(%f) error = %v, wantErr %v", tt.zoom, err, tt.wantErr)
			}
			if !tt.wantErr && canvas.ZoomLevel != tt.zoom {
				t.Errorf("zoom level not set: got %f, want %f", canvas.ZoomLevel, tt.zoom)
			}
		})
	}
}
