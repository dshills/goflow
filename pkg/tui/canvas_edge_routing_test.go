package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestEdgeRoutingStraightVertical tests straight vertical edge routing
func TestEdgeRoutingStraightVertical(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add two nodes vertically aligned
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}

	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 10, Y: 15}) // Same X, different Y

	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	err := canvas.AddEdge(edge)
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	// Find the canvas edge
	var cEdge *canvasEdge
	for _, e := range canvas.edges {
		if e.edge.ID == "edge-1" {
			cEdge = e
			break
		}
	}

	if cEdge == nil {
		t.Fatalf("edge not found in canvas")
	}

	// Should have at least 2 routing points (start and end)
	if len(cEdge.routingPoints) < 2 {
		t.Errorf("expected at least 2 routing points, got %d", len(cEdge.routingPoints))
	}

	// First point should be at source node center-bottom
	firstPoint := cEdge.routingPoints[0]
	expectedSourceX := canvas.nodes["node-1"].position.X + canvas.nodes["node-1"].width/2
	expectedSourceY := canvas.nodes["node-1"].position.Y + canvas.nodes["node-1"].height

	if firstPoint.X != expectedSourceX || firstPoint.Y != expectedSourceY {
		t.Errorf("first routing point = (%d, %d), want (%d, %d)",
			firstPoint.X, firstPoint.Y, expectedSourceX, expectedSourceY)
	}

	// Last point should be at target node center-top
	lastPoint := cEdge.routingPoints[len(cEdge.routingPoints)-1]
	expectedTargetX := canvas.nodes["node-2"].position.X + canvas.nodes["node-2"].width/2
	expectedTargetY := canvas.nodes["node-2"].position.Y

	if lastPoint.X != expectedTargetX || lastPoint.Y != expectedTargetY {
		t.Errorf("last routing point = (%d, %d), want (%d, %d)",
			lastPoint.X, lastPoint.Y, expectedTargetX, expectedTargetY)
	}

	// For vertically aligned nodes, X coordinates should be same
	if firstPoint.X != lastPoint.X {
		t.Errorf("vertically aligned edge should have same X: start=%d, end=%d",
			firstPoint.X, lastPoint.X)
	}
}

// TestEdgeRoutingLShaped tests L-shaped edge routing (target to the right)
func TestEdgeRoutingLShaped(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add two nodes with target to the right and below
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}

	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15}) // To the right and below

	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	err := canvas.AddEdge(edge)
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	var cEdge *canvasEdge
	for _, e := range canvas.edges {
		if e.edge.ID == "edge-1" {
			cEdge = e
			break
		}
	}

	if cEdge == nil {
		t.Fatalf("edge not found in canvas")
	}

	// L-shaped route should have intermediate points
	if len(cEdge.routingPoints) < 3 {
		t.Errorf("L-shaped edge should have at least 3 routing points, got %d", len(cEdge.routingPoints))
	}

	// Verify routing goes through intermediate points
	// The route should be: source → intermediate → target
	// This forms horizontal and vertical segments
	hasHorizontalSegment := false
	hasVerticalSegment := false

	for i := 0; i < len(cEdge.routingPoints)-1; i++ {
		from := cEdge.routingPoints[i]
		to := cEdge.routingPoints[i+1]

		if from.Y == to.Y && from.X != to.X {
			hasHorizontalSegment = true
		}
		if from.X == to.X && from.Y != to.Y {
			hasVerticalSegment = true
		}
	}

	if !hasHorizontalSegment || !hasVerticalSegment {
		t.Errorf("L-shaped edge should have both horizontal and vertical segments")
	}
}

// TestEdgeRoutingBackwardEdge tests routing for backward edges (target above source)
func TestEdgeRoutingBackwardEdge(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add two nodes with target above source (unusual case)
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}

	canvas.AddNode(node1, Position{X: 10, Y: 15})
	canvas.AddNode(node2, Position{X: 30, Y: 5}) // Above and to the right

	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	err := canvas.AddEdge(edge)
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}

	var cEdge *canvasEdge
	for _, e := range canvas.edges {
		if e.edge.ID == "edge-1" {
			cEdge = e
			break
		}
	}

	if cEdge == nil {
		t.Fatalf("edge not found in canvas")
	}

	// Backward edge should have multiple routing points to route around
	if len(cEdge.routingPoints) < 3 {
		t.Errorf("backward edge should have at least 3 routing points, got %d", len(cEdge.routingPoints))
	}
}

// TestEdgeRoutingMultipleEdges tests routing multiple edges between different nodes
func TestEdgeRoutingMultipleEdges(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add three nodes in a chain
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.MCPToolNode{ID: "node-2", ServerID: "s1", ToolName: "t1", OutputVariable: "v1"}
	node3 := &workflow.EndNode{ID: "node-3"}

	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})
	canvas.AddNode(node3, Position{X: 50, Y: 25})

	// Add edges
	edge1 := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	edge2 := &workflow.Edge{ID: "edge-2", FromNodeID: "node-2", ToNodeID: "node-3"}

	err := canvas.AddEdge(edge1)
	if err != nil {
		t.Fatalf("AddEdge(edge1) error = %v", err)
	}

	err = canvas.AddEdge(edge2)
	if err != nil {
		t.Fatalf("AddEdge(edge2) error = %v", err)
	}

	// Verify both edges are routed
	if len(canvas.edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(canvas.edges))
	}

	for _, e := range canvas.edges {
		if len(e.routingPoints) < 2 {
			t.Errorf("edge %s has insufficient routing points: %d", e.edge.ID, len(e.routingPoints))
		}
	}
}

// TestEdgeReroutingOnNodeMove tests that edges are re-routed when nodes move
func TestEdgeReroutingOnNodeMove(t *testing.T) {
	canvas := NewCanvas(80, 40)

	// Add nodes and edge
	node1 := &workflow.StartNode{ID: "node-1"}
	node2 := &workflow.EndNode{ID: "node-2"}

	canvas.AddNode(node1, Position{X: 10, Y: 5})
	canvas.AddNode(node2, Position{X: 30, Y: 15})

	edge := &workflow.Edge{ID: "edge-1", FromNodeID: "node-1", ToNodeID: "node-2"}
	canvas.AddEdge(edge)

	// Get initial routing
	var initialRouting []Position
	for _, e := range canvas.edges {
		if e.edge.ID == "edge-1" {
			initialRouting = make([]Position, len(e.routingPoints))
			copy(initialRouting, e.routingPoints)
			break
		}
	}

	// Move target node
	err := canvas.MoveNode("node-2", Position{X: 50, Y: 25})
	if err != nil {
		t.Fatalf("MoveNode() error = %v", err)
	}

	// Get new routing
	var newRouting []Position
	for _, e := range canvas.edges {
		if e.edge.ID == "edge-1" {
			newRouting = make([]Position, len(e.routingPoints))
			copy(newRouting, e.routingPoints)
			break
		}
	}

	// Routing should have changed
	if len(initialRouting) == len(newRouting) {
		same := true
		for i := range initialRouting {
			if initialRouting[i].X != newRouting[i].X || initialRouting[i].Y != newRouting[i].Y {
				same = false
				break
			}
		}
		if same {
			t.Errorf("edge routing should have changed after node move")
		}
	}

	// Verify new routing ends at new target position
	lastPoint := newRouting[len(newRouting)-1]
	expectedTargetX := canvas.nodes["node-2"].position.X + canvas.nodes["node-2"].width/2
	expectedTargetY := canvas.nodes["node-2"].position.Y

	if lastPoint.X != expectedTargetX || lastPoint.Y != expectedTargetY {
		t.Errorf("edge routing not updated to new target position: got (%d, %d), want (%d, %d)",
			lastPoint.X, lastPoint.Y, expectedTargetX, expectedTargetY)
	}
}

// TestGetEdgeDirection tests edge arrow direction calculation
func TestGetEdgeDirection(t *testing.T) {
	tests := []struct {
		name string
		from Position
		to   Position
		want string
	}{
		{
			name: "down",
			from: Position{X: 10, Y: 5},
			to:   Position{X: 10, Y: 10},
			want: "▼",
		},
		{
			name: "up",
			from: Position{X: 10, Y: 10},
			to:   Position{X: 10, Y: 5},
			want: "▲",
		},
		{
			name: "right",
			from: Position{X: 5, Y: 10},
			to:   Position{X: 15, Y: 10},
			want: "►",
		},
		{
			name: "left",
			from: Position{X: 15, Y: 10},
			to:   Position{X: 5, Y: 10},
			want: "◄",
		},
		{
			name: "same position",
			from: Position{X: 10, Y: 10},
			to:   Position{X: 10, Y: 10},
			want: "●",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEdgeDirection(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("getEdgeDirection() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetEdgeLineChar tests line character selection
func TestGetEdgeLineChar(t *testing.T) {
	tests := []struct {
		name       string
		from       Position
		to         Position
		isVertical bool
		want       string
	}{
		{
			name:       "vertical line",
			from:       Position{X: 10, Y: 5},
			to:         Position{X: 10, Y: 10},
			isVertical: true,
			want:       "│",
		},
		{
			name:       "horizontal line",
			from:       Position{X: 5, Y: 10},
			to:         Position{X: 15, Y: 10},
			isVertical: false,
			want:       "─",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEdgeLineChar(tt.from, tt.to, tt.isVertical)
			if got != tt.want {
				t.Errorf("getEdgeLineChar() = %q, want %q", got, tt.want)
			}
		})
	}
}
