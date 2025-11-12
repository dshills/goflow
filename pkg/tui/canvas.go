package tui

import (
	"fmt"

	"github.com/dshills/goflow/pkg/workflow"
)

// Canvas manages the visual workflow graph with node positioning,
// viewport, rendering, and user interaction.
type Canvas struct {
	// Width is the terminal width in logical units
	Width int
	// Height is the terminal height in logical units
	Height int
	// ViewportX is the viewport offset X in logical coordinates
	ViewportX int
	// ViewportY is the viewport offset Y in logical coordinates
	ViewportY int
	// ZoomLevel is the zoom factor (0.5 to 2.0)
	ZoomLevel float64
	// nodes maps node IDs to canvasNode instances
	nodes map[string]*canvasNode
	// edges contains all canvas edges
	edges []*canvasEdge
	// selectedID is the currently selected node ID
	selectedID string
}

// canvasNode wraps a domain Node with rendering state
type canvasNode struct {
	// node is the domain node (immutable from TUI perspective)
	node workflow.Node
	// position is the logical coordinates (X, Y)
	position Position
	// width is the rendered width in characters
	width int
	// height is the rendered height in lines
	height int
	// selected indicates visual selection state
	selected bool
	// highlighted indicates temporary highlight (hover, focus)
	highlighted bool
	// validationStatus is "valid", "warning", or "error"
	validationStatus string
}

// NewCanvas creates a canvas with the given dimensions in logical units
func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		Width:     width,
		Height:    height,
		ViewportX: 0,
		ViewportY: 0,
		ZoomLevel: 1.0,
		nodes:     make(map[string]*canvasNode),
		edges:     make([]*canvasEdge, 0),
	}
}

// AddNode adds a node to the canvas at the specified position.
// If position is nil, the node is auto-positioned.
// Returns error if node with same ID already exists.
func (c *Canvas) AddNode(node workflow.Node, pos Position) error {
	if node == nil {
		return fmt.Errorf("cannot add nil node")
	}

	nodeID := node.GetID()
	if nodeID == "" {
		return fmt.Errorf("cannot add node with empty ID")
	}

	if _, exists := c.nodes[nodeID]; exists {
		return fmt.Errorf("node already exists: %s", nodeID)
	}

	// Calculate node dimensions based on type and content
	width, height := c.calculateNodeSize(node)

	cNode := &canvasNode{
		node:             node,
		position:         pos,
		width:            width,
		height:           height,
		selected:         false,
		highlighted:      false,
		validationStatus: "valid",
	}

	c.nodes[nodeID] = cNode
	return nil
}

// RemoveNode removes a node from the canvas.
// Also removes all edges connected to the node.
// Returns error if node doesn't exist.
func (c *Canvas) RemoveNode(nodeID string) error {
	if _, exists := c.nodes[nodeID]; !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Remove the node
	delete(c.nodes, nodeID)

	// Remove all edges connected to this node
	newEdges := make([]*canvasEdge, 0, len(c.edges))
	for _, edge := range c.edges {
		if edge.edge.FromNodeID != nodeID && edge.edge.ToNodeID != nodeID {
			newEdges = append(newEdges, edge)
		}
	}
	c.edges = newEdges

	// Clear selection if removed node was selected
	if c.selectedID == nodeID {
		c.selectedID = ""
	}

	return nil
}

// MoveNode updates a node's position.
// Re-routes all connected edges.
// Returns error if node doesn't exist or position is invalid.
func (c *Canvas) MoveNode(nodeID string, newPos Position) error {
	cNode, exists := c.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Validate position (no negative coordinates)
	if newPos.X < 0 || newPos.Y < 0 {
		return fmt.Errorf("invalid position: coordinates cannot be negative")
	}

	// Update position
	cNode.position = newPos

	// Re-route all edges connected to this node
	for _, edge := range c.edges {
		if edge.edge.FromNodeID == nodeID || edge.edge.ToNodeID == nodeID {
			c.routeEdge(edge)
		}
	}

	return nil
}

// AddEdge adds an edge between two nodes.
// Calculates routing automatically using orthogonal algorithm.
// Returns error if source or target node doesn't exist.
func (c *Canvas) AddEdge(edge *workflow.Edge) error {
	if edge == nil {
		return fmt.Errorf("cannot add nil edge")
	}

	// Validate that both nodes exist
	if _, exists := c.nodes[edge.FromNodeID]; !exists {
		return fmt.Errorf("source node not found: %s", edge.FromNodeID)
	}
	if _, exists := c.nodes[edge.ToNodeID]; !exists {
		return fmt.Errorf("target node not found: %s", edge.ToNodeID)
	}

	// Check for duplicate edges
	for _, existing := range c.edges {
		if existing.edge.FromNodeID == edge.FromNodeID &&
			existing.edge.ToNodeID == edge.ToNodeID {
			return fmt.Errorf("edge already exists from %s to %s", edge.FromNodeID, edge.ToNodeID)
		}
	}

	// Create canvas edge
	cEdge := &canvasEdge{
		edge:          edge,
		routingPoints: make([]Position, 0),
		selected:      false,
	}

	// Calculate routing
	c.routeEdge(cEdge)

	c.edges = append(c.edges, cEdge)
	return nil
}

// RemoveEdge removes an edge from the canvas by edge ID.
// Returns error if edge doesn't exist.
func (c *Canvas) RemoveEdge(fromID, toID string) error {
	found := false
	newEdges := make([]*canvasEdge, 0, len(c.edges))
	for _, edge := range c.edges {
		if edge.edge.FromNodeID == fromID && edge.edge.ToNodeID == toID {
			found = true
		} else {
			newEdges = append(newEdges, edge)
		}
	}

	if !found {
		return fmt.Errorf("edge not found from %s to %s", fromID, toID)
	}

	c.edges = newEdges
	return nil
}

// SelectNode sets the selected node.
// Returns error if node doesn't exist.
func (c *Canvas) SelectNode(nodeID string) error {
	if nodeID != "" {
		if _, exists := c.nodes[nodeID]; !exists {
			return fmt.Errorf("node not found: %s", nodeID)
		}
	}

	// Clear previous selection
	if c.selectedID != "" {
		if prevNode, exists := c.nodes[c.selectedID]; exists {
			prevNode.selected = false
		}
	}

	// Set new selection
	c.selectedID = nodeID
	if nodeID != "" {
		c.nodes[nodeID].selected = true
	}

	return nil
}

// GetSelectedNode returns the currently selected node, or nil if none selected
func (c *Canvas) GetSelectedNode() *canvasNode {
	if c.selectedID == "" {
		return nil
	}
	return c.nodes[c.selectedID]
}

// NodeAtPosition returns the node ID at the given terminal coordinates.
// Returns "" if no node at position.
func (c *Canvas) NodeAtPosition(termX, termY int) string {
	// Convert terminal coordinates to logical coordinates
	logical := TerminalToLogical(
		Position{X: termX, Y: termY},
		c.ViewportX,
		c.ViewportY,
		c.ZoomLevel,
	)

	// Check each node's bounding box
	for nodeID, cNode := range c.nodes {
		bbox := BoundingBox{
			TopLeft: cNode.position,
			Size:    Size{Width: cNode.width, Height: cNode.height},
		}
		if bbox.Contains(logical) {
			return nodeID
		}
	}

	return ""
}

// calculateNodeSize computes the rendered dimensions for a node
// based on its type and content
func (c *Canvas) calculateNodeSize(node workflow.Node) (width int, height int) {
	// Default minimum size
	width = 20
	height = 5

	// Adjust based on node type
	switch node.Type() {
	case "start":
		width = 16
		height = 3
	case "end":
		width = 16
		height = 3
	case "mcp_tool":
		// Size based on tool name length
		width = 20
		height = 5
	case "transform":
		width = 20
		height = 5
	case "condition":
		width = 18
		height = 4
	case "loop":
		width = 18
		height = 4
	case "parallel":
		width = 20
		height = 4
	}

	return width, height
}
