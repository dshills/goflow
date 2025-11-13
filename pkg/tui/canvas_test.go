package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestCanvasCreation tests NewCanvas constructor
func TestCanvasCreation(t *testing.T) {
	canvas := NewCanvas(80, 40)

	if canvas.Width != 80 {
		t.Errorf("expected width 80, got %d", canvas.Width)
	}
	if canvas.Height != 40 {
		t.Errorf("expected height 40, got %d", canvas.Height)
	}
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("expected zoom 1.0, got %f", canvas.ZoomLevel)
	}
	if canvas.ViewportX != 0 || canvas.ViewportY != 0 {
		t.Errorf("expected viewport at origin, got (%d, %d)", canvas.ViewportX, canvas.ViewportY)
	}
	if len(canvas.nodes) != 0 {
		t.Errorf("expected empty nodes map, got %d nodes", len(canvas.nodes))
	}
}

// TestCanvasAddNode tests adding nodes to canvas
func TestCanvasAddNode(t *testing.T) {
	canvas := NewCanvas(80, 40)

	tests := []struct {
		name    string
		node    workflow.Node
		pos     Position
		wantErr bool
		errMsg  string
	}{
		{
			name:    "add start node",
			node:    &workflow.StartNode{ID: "start-1"},
			pos:     Position{X: 10, Y: 5},
			wantErr: false,
		},
		{
			name:    "add tool node",
			node:    &workflow.MCPToolNode{ID: "tool-1", ServerID: "server-1", ToolName: "test", OutputVariable: "out"},
			pos:     Position{X: 30, Y: 15},
			wantErr: false,
		},
		{
			name:    "add duplicate node",
			node:    &workflow.StartNode{ID: "start-1"},
			pos:     Position{X: 50, Y: 25},
			wantErr: true,
			errMsg:  "node already exists",
		},
		{
			name:    "add nil node",
			node:    nil,
			pos:     Position{X: 10, Y: 10},
			wantErr: true,
			errMsg:  "cannot add nil node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.AddNode(tt.node, tt.pos)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				// Check error message contains expected string
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
			if !tt.wantErr && tt.node != nil {
				// Verify node was added
				if _, exists := canvas.nodes[tt.node.GetID()]; !exists {
					t.Errorf("node %s was not added to canvas", tt.node.GetID())
				}
			}
		})
	}
}

// TestCanvasRemoveNode tests removing nodes from canvas
func TestCanvasRemoveNode(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add nodes
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}
	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})

	// Add edge between them
	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	canvas.AddEdge(edge)

	tests := []struct {
		name    string
		nodeID  string
		wantErr bool
	}{
		{
			name:    "remove existing node",
			nodeID:  "node-1",
			wantErr: false,
		},
		{
			name:    "remove non-existent node",
			nodeID:  "node-3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.RemoveNode(tt.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify node was removed
				if _, exists := canvas.nodes[tt.nodeID]; exists {
					t.Errorf("node %s was not removed from canvas", tt.nodeID)
				}
				// Verify connected edges were removed
				for _, e := range canvas.edges {
					if e.edge.FromNodeID == tt.nodeID || e.edge.ToNodeID == tt.nodeID {
						t.Errorf("edge connected to removed node %s still exists", tt.nodeID)
					}
				}
			}
		})
	}
}

// TestCanvasMoveNode tests moving nodes
func TestCanvasMoveNode(t *testing.T) {
	canvas := NewCanvas(80, 40)

	node := &workflow.StartNode{ID: "node-1"}
	canvas.AddNode(node, Position{X: 10, Y: 5})

	tests := []struct {
		name    string
		nodeID  string
		newPos  Position
		wantErr bool
	}{
		{
			name:    "move to valid position",
			nodeID:  "node-1",
			newPos:  Position{X: 20, Y: 10},
			wantErr: false,
		},
		{
			name:    "move to negative X",
			nodeID:  "node-1",
			newPos:  Position{X: -5, Y: 10},
			wantErr: true,
		},
		{
			name:    "move to negative Y",
			nodeID:  "node-1",
			newPos:  Position{X: 10, Y: -5},
			wantErr: true,
		},
		{
			name:    "move non-existent node",
			nodeID:  "node-999",
			newPos:  Position{X: 10, Y: 10},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.MoveNode(tt.nodeID, tt.newPos)
			if (err != nil) != tt.wantErr {
				t.Errorf("MoveNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify position was updated
				cNode := canvas.nodes[tt.nodeID]
				if cNode.position.X != tt.newPos.X || cNode.position.Y != tt.newPos.Y {
					t.Errorf("node position not updated: got (%d, %d), want (%d, %d)",
						cNode.position.X, cNode.position.Y, tt.newPos.X, tt.newPos.Y)
				}
			}
		})
	}
}

// TestCanvasAddEdge tests adding edges
func TestCanvasAddEdge(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add nodes
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}
	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})

	tests := []struct {
		name    string
		edge    *workflow.Edge
		wantErr bool
		errMsg  string
	}{
		{
			name:    "add valid edge",
			edge:    &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"},
			wantErr: false,
		},
		{
			name:    "add duplicate edge",
			edge:    &workflow.Edge{ID: "edge-2", FromNodeID: "node-1", ToNodeID: "node-2"},
			wantErr: true,
			errMsg:  "edge already exists",
		},
		{
			name:    "add edge with invalid source",
			edge:    &workflow.Edge{ID: "edge-3", FromNodeID: "node-999", ToNodeID: "node-2"},
			wantErr: true,
			errMsg:  "source node not found",
		},
		{
			name:    "add edge with invalid target",
			edge:    &workflow.Edge{ID: "edge-4", FromNodeID: "node-1", ToNodeID: "node-999"},
			wantErr: true,
			errMsg:  "target node not found",
		},
		{
			name:    "add nil edge",
			edge:    nil,
			wantErr: true,
			errMsg:  "cannot add nil edge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.AddEdge(tt.edge)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddEdge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
			if !tt.wantErr && tt.edge != nil {
				// Verify edge was added and routed
				found := false
				for _, e := range canvas.edges {
					if e.edge.ID == tt.edge.ID {
						found = true
						if len(e.routingPoints) < 2 {
							t.Errorf("edge routing points not calculated: got %d points", len(e.routingPoints))
						}
						break
					}
				}
				if !found {
					t.Errorf("edge %s was not added to canvas", tt.edge.ID)
				}
			}
		})
	}
}

// TestCanvasRemoveEdge tests removing edges
func TestCanvasRemoveEdge(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add nodes and edge
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}
	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})

	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	canvas.AddEdge(edge)

	tests := []struct {
		name    string
		fromID  string
		toID    string
		wantErr bool
	}{
		{
			name:    "remove existing edge",
			fromID:  "node-1",
			toID:    "node-2",
			wantErr: false,
		},
		{
			name:    "remove non-existent edge",
			fromID:  "node-1",
			toID:    "node-3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.RemoveEdge(tt.fromID, tt.toID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveEdge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify edge was removed
				for _, e := range canvas.edges {
					if e.edge.FromNodeID == tt.fromID && e.edge.ToNodeID == tt.toID {
						t.Errorf("edge from %s to %s was not removed", tt.fromID, tt.toID)
					}
				}
			}
		})
	}
}

// TestCanvasSelectNode tests node selection
func TestCanvasSelectNode(t *testing.T) {
	canvas := NewCanvas(80, 40)

	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}
	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})

	// Select first node
	err := canvas.SelectNode("node-1")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if canvas.selectedID != "node-1" {
		t.Errorf("expected selectedID = node-1, got %s", canvas.selectedID)
	}

	if !canvas.nodes["node-1"].selected {
		t.Errorf("node-1 should be marked as selected")
	}

	// Select second node (should deselect first)
	err = canvas.SelectNode("node-2")
	if err != nil {
		t.Fatalf("SelectNode() error = %v", err)
	}

	if canvas.nodes["node-1"].selected {
		t.Errorf("node-1 should be deselected")
	}

	if !canvas.nodes["node-2"].selected {
		t.Errorf("node-2 should be selected")
	}

	// Deselect all
	err = canvas.SelectNode("")
	if err != nil {
		t.Fatalf("SelectNode('') error = %v", err)
	}

	if canvas.selectedID != "" {
		t.Errorf("expected no selection, got %s", canvas.selectedID)
	}

	// Try to select non-existent node
	err = canvas.SelectNode("node-999")
	if err == nil {
		t.Errorf("expected error when selecting non-existent node")
	}
}

// TestCoordinateConversion tests logical <-> terminal coordinate conversion
func TestCoordinateConversion(t *testing.T) {
	tests := []struct {
		name      string
		logical   Position
		viewportX int
		viewportY int
		zoom      float64
		wantTerm  Position
	}{
		{
			name:      "100% zoom, no viewport offset",
			logical:   Position{X: 10, Y: 5},
			viewportX: 0,
			viewportY: 0,
			zoom:      1.0,
			wantTerm:  Position{X: 10, Y: 5},
		},
		{
			name:      "100% zoom, with viewport offset",
			logical:   Position{X: 20, Y: 10},
			viewportX: 5,
			viewportY: 3,
			zoom:      1.0,
			wantTerm:  Position{X: 15, Y: 7},
		},
		{
			name:      "200% zoom, no viewport offset",
			logical:   Position{X: 10, Y: 5},
			viewportX: 0,
			viewportY: 0,
			zoom:      2.0,
			wantTerm:  Position{X: 20, Y: 10},
		},
		{
			name:      "50% zoom, no viewport offset",
			logical:   Position{X: 20, Y: 10},
			viewportX: 0,
			viewportY: 0,
			zoom:      0.5,
			wantTerm:  Position{X: 10, Y: 5},
		},
		{
			name:      "200% zoom, with viewport offset",
			logical:   Position{X: 20, Y: 10},
			viewportX: 5,
			viewportY: 3,
			zoom:      2.0,
			wantTerm:  Position{X: 30, Y: 14},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test logical to terminal
			term := LogicalToTerminal(tt.logical, tt.viewportX, tt.viewportY, tt.zoom)
			if term.X != tt.wantTerm.X || term.Y != tt.wantTerm.Y {
				t.Errorf("LogicalToTerminal() = (%d, %d), want (%d, %d)",
					term.X, term.Y, tt.wantTerm.X, tt.wantTerm.Y)
			}

			// Test terminal to logical (round trip)
			logical := TerminalToLogical(term, tt.viewportX, tt.viewportY, tt.zoom)
			// Allow for rounding errors
			if abs(logical.X-tt.logical.X) > 1 || abs(logical.Y-tt.logical.Y) > 1 {
				t.Errorf("TerminalToLogical() = (%d, %d), want (%d, %d)",
					logical.X, logical.Y, tt.logical.X, tt.logical.Y)
			}
		})
	}
}

// TestNodeAtPosition tests node hit detection
func TestNodeAtPosition(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add a node at (10, 5) with default size
	node := &workflow.StartNode{ID: "node-1"}
	canvas.AddNode(node, Position{X: 10, Y: 5})

	tests := []struct {
		name     string
		termX    int
		termY    int
		wantNode string
	}{
		{
			name:     "click on node center",
			termX:    18, // 10 + width/2
			termY:    6,  // 5 + height/2
			wantNode: "node-1",
		},
		{
			name:     "click on node top-left",
			termX:    10,
			termY:    5,
			wantNode: "node-1",
		},
		{
			name:     "click outside node",
			termX:    50,
			termY:    50,
			wantNode: "",
		},
		{
			name:     "click before node",
			termX:    5,
			termY:    5,
			wantNode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeID := canvas.NodeAtPosition(tt.termX, tt.termY)
			if nodeID != tt.wantNode {
				t.Errorf("NodeAtPosition(%d, %d) = %q, want %q",
					tt.termX, tt.termY, nodeID, tt.wantNode)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function for absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
