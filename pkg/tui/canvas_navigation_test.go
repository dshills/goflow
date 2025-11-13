package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestPan_UpdatesViewport verifies that Pan() correctly updates viewport coordinates
func TestPan_UpdatesViewport(t *testing.T) {
	canvas := NewCanvas(100, 50)
	canvas.ViewportX = 10
	canvas.ViewportY = 5

	// Pan right and down
	canvas.Pan(20, 15)

	if canvas.ViewportX != 30 {
		t.Errorf("Expected ViewportX=30, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 20 {
		t.Errorf("Expected ViewportY=20, got %d", canvas.ViewportY)
	}
}

// TestPan_NegativeDelta verifies panning with negative delta (left/up)
func TestPan_NegativeDelta(t *testing.T) {
	canvas := NewCanvas(100, 50)
	canvas.ViewportX = 30
	canvas.ViewportY = 20

	// Pan left and up
	canvas.Pan(-10, -5)

	if canvas.ViewportX != 20 {
		t.Errorf("Expected ViewportX=20, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 15 {
		t.Errorf("Expected ViewportY=15, got %d", canvas.ViewportY)
	}
}

// TestPan_ClampsToBounds verifies that Pan() clamps viewport to valid bounds
func TestPan_ClampsToBounds(t *testing.T) {
	canvas := NewCanvas(100, 50)
	canvas.ViewportX = 10
	canvas.ViewportY = 5

	// Pan far left and up (should clamp to 0)
	canvas.Pan(-50, -20)

	if canvas.ViewportX != 0 {
		t.Errorf("Expected ViewportX=0 (clamped), got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 0 {
		t.Errorf("Expected ViewportY=0 (clamped), got %d", canvas.ViewportY)
	}
}

// TestPan_AllowsPositiveBounds verifies that Pan() allows panning to positive coordinates
func TestPan_AllowsPositiveBounds(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Pan to large positive values (should be allowed)
	canvas.Pan(500, 300)

	if canvas.ViewportX != 500 {
		t.Errorf("Expected ViewportX=500, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 300 {
		t.Errorf("Expected ViewportY=300, got %d", canvas.ViewportY)
	}
}

// TestZoom_ChangesZoomLevel verifies that Zoom() updates zoom level correctly
func TestZoom_ChangesZoomLevel(t *testing.T) {
	canvas := NewCanvas(100, 50)
	canvas.ZoomLevel = 1.0
	canvas.ViewportX = 100
	canvas.ViewportY = 50

	// Zoom in to 150%
	err := canvas.Zoom(1.5)
	if err != nil {
		t.Fatalf("Zoom(1.5) returned error: %v", err)
	}

	if canvas.ZoomLevel != 1.5 {
		t.Errorf("Expected ZoomLevel=1.5, got %.2f", canvas.ZoomLevel)
	}
}

// TestZoom_RejectsInvalidLevels verifies that Zoom() rejects out-of-range levels
func TestZoom_RejectsInvalidLevels(t *testing.T) {
	tests := []struct {
		name      string
		zoomLevel float64
		wantError bool
	}{
		{"Valid 0.5", 0.5, false},
		{"Valid 1.0", 1.0, false},
		{"Valid 2.0", 2.0, false},
		{"Invalid too low", 0.3, true},
		{"Invalid too high", 2.5, true},
		{"Invalid negative", -1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewCanvas(100, 50)
			canvas.ZoomLevel = 1.0

			err := canvas.Zoom(tt.zoomLevel)

			if tt.wantError && err == nil {
				t.Errorf("Expected error for zoom level %.2f, got nil", tt.zoomLevel)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error for zoom level %.2f, got: %v", tt.zoomLevel, err)
			}

			// Verify zoom level unchanged on error
			if tt.wantError && canvas.ZoomLevel != 1.0 {
				t.Errorf("Expected ZoomLevel unchanged (1.0), got %.2f", canvas.ZoomLevel)
			}
		})
	}
}

// TestZoom_MaintainsCenter verifies that Zoom() adjusts viewport to keep center stable
func TestZoom_MaintainsCenter(t *testing.T) {
	canvas := NewCanvas(100, 50)
	canvas.ZoomLevel = 1.0
	canvas.ViewportX = 100
	canvas.ViewportY = 50

	// Calculate center before zoom
	oldCenterX := canvas.ViewportX + int(float64(canvas.Width)/(2.0*canvas.ZoomLevel))
	oldCenterY := canvas.ViewportY + int(float64(canvas.Height)/(2.0*canvas.ZoomLevel))

	// Zoom in
	err := canvas.Zoom(2.0)
	if err != nil {
		t.Fatalf("Zoom(2.0) returned error: %v", err)
	}

	// Calculate center after zoom
	newCenterX := canvas.ViewportX + int(float64(canvas.Width)/(2.0*canvas.ZoomLevel))
	newCenterY := canvas.ViewportY + int(float64(canvas.Height)/(2.0*canvas.ZoomLevel))

	// Centers should be approximately the same (within rounding tolerance)
	tolerance := 2
	if absInt(oldCenterX-newCenterX) > tolerance {
		t.Errorf("Center X shifted too much: old=%d, new=%d", oldCenterX, newCenterX)
	}
	if absInt(oldCenterY-newCenterY) > tolerance {
		t.Errorf("Center Y shifted too much: old=%d, new=%d", oldCenterY, newCenterY)
	}
}

// TestResetView_CentersOnStartNode verifies that ResetView() centers on start node
func TestResetView_CentersOnStartNode(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Add start node at position (200, 100)
	startNode := &workflow.StartNode{ID: "start"}
	err := canvas.AddNode(startNode, Position{X: 200, Y: 100})
	if err != nil {
		t.Fatalf("AddNode() failed: %v", err)
	}

	// Set non-default zoom
	canvas.ZoomLevel = 1.5

	// Reset view
	canvas.ResetView()

	// Verify zoom is 100%
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("Expected ZoomLevel=1.0, got %.2f", canvas.ZoomLevel)
	}

	// Verify viewport centers on start node
	// Start node is 16 chars wide, 3 lines tall (from canvas.go calculateNodeSize)
	cNode := canvas.nodes["start"]
	startCenterX := cNode.position.X + cNode.width/2
	startCenterY := cNode.position.Y + cNode.height/2

	expectedViewportX := startCenterX - canvas.Width/2
	expectedViewportY := startCenterY - canvas.Height/2

	// Allow for clamping to zero
	if expectedViewportX < 0 {
		expectedViewportX = 0
	}
	if expectedViewportY < 0 {
		expectedViewportY = 0
	}

	if canvas.ViewportX != expectedViewportX {
		t.Errorf("Expected ViewportX=%d, got %d", expectedViewportX, canvas.ViewportX)
	}
	if canvas.ViewportY != expectedViewportY {
		t.Errorf("Expected ViewportY=%d, got %d", expectedViewportY, canvas.ViewportY)
	}
}

// TestResetView_FallbackWhenNoStartNode verifies ResetView() fallback behavior
func TestResetView_FallbackWhenNoStartNode(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Add a non-start node
	toolNode := &workflow.MCPToolNode{
		ID:       "tool-1",
		ServerID: "test-server",
		ToolName: "test-tool",
	}
	err := canvas.AddNode(toolNode, Position{X: 50, Y: 30})
	if err != nil {
		t.Fatalf("AddNode() failed: %v", err)
	}

	// Reset view
	canvas.ResetView()

	// Verify zoom is 100%
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("Expected ZoomLevel=1.0, got %.2f", canvas.ZoomLevel)
	}

	// Verify viewport at origin
	if canvas.ViewportX != 0 {
		t.Errorf("Expected ViewportX=0, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 0 {
		t.Errorf("Expected ViewportY=0, got %d", canvas.ViewportY)
	}
}

// TestResetView_EmptyCanvas verifies ResetView() with no nodes
func TestResetView_EmptyCanvas(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Reset view on empty canvas
	canvas.ResetView()

	// Verify zoom is 100%
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("Expected ZoomLevel=1.0, got %.2f", canvas.ZoomLevel)
	}

	// Verify viewport at origin
	if canvas.ViewportX != 0 {
		t.Errorf("Expected ViewportX=0, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 0 {
		t.Errorf("Expected ViewportY=0, got %d", canvas.ViewportY)
	}
}

// TestFitAll_FitsAllNodesInViewport verifies that FitAll() shows all nodes
func TestFitAll_FitsAllNodesInViewport(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Add three nodes spread out
	nodes := []workflow.Node{
		&workflow.StartNode{ID: "node-1"},
		&workflow.MCPToolNode{
			ID:       "node-2",
			ServerID: "test-server",
			ToolName: "test-tool",
		},
		&workflow.TransformNode{
			ID:         "node-3",
			Expression: "test",
		},
	}
	positions := []Position{
		{X: 10, Y: 10},
		{X: 200, Y: 50},
		{X: 100, Y: 150},
	}

	for i, node := range nodes {
		err := canvas.AddNode(node, positions[i])
		if err != nil {
			t.Fatalf("AddNode() failed: %v", err)
		}
	}

	// Fit all
	canvas.FitAll()

	// Verify zoom is in valid range
	if canvas.ZoomLevel < 0.5 || canvas.ZoomLevel > 2.0 {
		t.Errorf("Expected ZoomLevel in [0.5, 2.0], got %.2f", canvas.ZoomLevel)
	}

	// Verify all nodes would be visible in the viewport
	// (This is a heuristic check - in practice we'd need to verify rendering)
	for _, cNode := range canvas.nodes {
		// Convert to terminal coordinates
		termPos := LogicalToTerminal(cNode.position, canvas.ViewportX, canvas.ViewportY, canvas.ZoomLevel)

		// Check if at least part of the node is in viewport
		// (We allow some margin for padding)
		margin := 30 // Increased margin to account for zoom padding
		if termPos.X < -margin || termPos.X > canvas.Width+margin {
			t.Errorf("Node %s X position %d outside viewport width %d (zoom=%.2f)",
				cNode.node.GetID(), termPos.X, canvas.Width, canvas.ZoomLevel)
		}
		if termPos.Y < -margin || termPos.Y > canvas.Height+margin {
			t.Errorf("Node %s Y position %d outside viewport height %d (zoom=%.2f)",
				cNode.node.GetID(), termPos.Y, canvas.Height, canvas.ZoomLevel)
		}
	}
}

// TestFitAll_EmptyCanvas verifies FitAll() with no nodes
func TestFitAll_EmptyCanvas(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Fit all on empty canvas
	canvas.FitAll()

	// Verify defaults
	if canvas.ZoomLevel != 1.0 {
		t.Errorf("Expected ZoomLevel=1.0 for empty canvas, got %.2f", canvas.ZoomLevel)
	}
	if canvas.ViewportX != 0 {
		t.Errorf("Expected ViewportX=0 for empty canvas, got %d", canvas.ViewportX)
	}
	if canvas.ViewportY != 0 {
		t.Errorf("Expected ViewportY=0 for empty canvas, got %d", canvas.ViewportY)
	}
}

// TestFitAll_SingleNode verifies FitAll() with one node
func TestFitAll_SingleNode(t *testing.T) {
	canvas := NewCanvas(100, 50)

	// Add single node
	node := &workflow.MCPToolNode{
		ID:       "node-1",
		ServerID: "test-server",
		ToolName: "test-tool",
	}
	err := canvas.AddNode(node, Position{X: 50, Y: 30})
	if err != nil {
		t.Fatalf("AddNode() failed: %v", err)
	}

	// Fit all
	canvas.FitAll()

	// Verify zoom is in valid range
	if canvas.ZoomLevel < 0.5 || canvas.ZoomLevel > 2.0 {
		t.Errorf("Expected ZoomLevel in [0.5, 2.0], got %.2f", canvas.ZoomLevel)
	}

	// Verify node is visible
	cNode := canvas.nodes["node-1"]
	termPos := LogicalToTerminal(cNode.position, canvas.ViewportX, canvas.ViewportY, canvas.ZoomLevel)

	margin := 20
	if termPos.X < -margin || termPos.X > canvas.Width+margin {
		t.Errorf("Node X position %d outside viewport width %d", termPos.X, canvas.Width)
	}
	if termPos.Y < -margin || termPos.Y > canvas.Height+margin {
		t.Errorf("Node Y position %d outside viewport height %d", termPos.Y, canvas.Height)
	}
}

// TestFitAll_ClampsZoom verifies that FitAll() clamps zoom to valid range
func TestFitAll_ClampsZoom(t *testing.T) {
	// Create a very small canvas
	canvas := NewCanvas(10, 5)

	// Add nodes spread very far apart (would require zoom < 0.5)
	nodes := []workflow.Node{
		&workflow.StartNode{ID: "node-1"},
		&workflow.MCPToolNode{
			ID:       "node-2",
			ServerID: "test-server",
			ToolName: "test-tool",
		},
	}
	positions := []Position{
		{X: 10, Y: 10},
		{X: 1000, Y: 500}, // Very far away
	}

	for i, node := range nodes {
		err := canvas.AddNode(node, positions[i])
		if err != nil {
			t.Fatalf("AddNode() failed: %v", err)
		}
	}

	// Fit all
	canvas.FitAll()

	// Verify zoom is clamped to minimum
	if canvas.ZoomLevel < 0.5 {
		t.Errorf("Expected ZoomLevel >= 0.5 (clamped), got %.2f", canvas.ZoomLevel)
	}
	if canvas.ZoomLevel > 2.0 {
		t.Errorf("Expected ZoomLevel <= 2.0 (clamped), got %.2f", canvas.ZoomLevel)
	}
}
