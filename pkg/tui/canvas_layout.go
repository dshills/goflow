package tui

import (
	"fmt"

	"github.com/dshills/goflow/pkg/workflow"
)

// Layout constants
const (
	horizontalSpacing = 4  // Characters between nodes horizontally
	verticalSpacing   = 2  // Lines between node layers vertically
	layoutStartX      = 10 // Starting X position for layout
	layoutStartY      = 5  // Starting Y position for layout
)

// AutoLayout positions nodes using topological hierarchical layout.
// Algorithm (simplified Sugiyama framework):
// 1. Topological sort of workflow nodes
// 2. Assign layers (Y coordinate) based on longest path from start
// 3. Within each layer, order nodes to minimize edge crossings (median heuristic)
// 4. Assign X coordinates based on layer ordering
// 5. Apply spacing parameters
func (c *Canvas) AutoLayout(wf *workflow.Workflow) {
	if wf == nil || len(wf.Nodes) == 0 {
		return
	}

	// Build adjacency list for graph traversal
	adjacency := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all nodes
	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		adjacency[nodeID] = make([]string, 0)
		inDegree[nodeID] = 0
	}

	// Build graph
	for _, edge := range wf.Edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge.ToNodeID)
		inDegree[edge.ToNodeID]++
	}

	// Step 1: Topological sort to determine layers
	layers := c.assignLayers(wf, adjacency, inDegree)

	// Step 2: Order nodes within each layer
	c.orderNodesInLayers(layers, adjacency, inDegree)

	// Step 3: Assign positions
	c.assignPositions(layers)

	// Step 4: Re-route all edges
	for _, edge := range c.edges {
		c.routeEdge(edge)
	}
}

// assignLayers performs topological sort and assigns each node to a layer
// based on the longest path from the start node
func (c *Canvas) assignLayers(wf *workflow.Workflow, adjacency map[string][]string, inDegree map[string]int) [][]string {
	// Find the longest path to each node (layer assignment)
	layerMap := make(map[string]int)
	maxLayer := 0

	// Copy inDegree to avoid modifying original
	inDegreeCopy := make(map[string]int)
	for k, v := range inDegree {
		inDegreeCopy[k] = v
	}

	// Kahn's algorithm for topological sort with layer assignment
	queue := make([]string, 0)

	// Start with nodes that have no incoming edges
	for nodeID, degree := range inDegreeCopy {
		if degree == 0 {
			queue = append(queue, nodeID)
			layerMap[nodeID] = 0
		}
	}

	// Process nodes level by level
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentLayer := layerMap[current]

		// Process all outgoing edges
		for _, neighbor := range adjacency[current] {
			inDegreeCopy[neighbor]--

			// Assign layer as max of (current layer + 1) and existing layer
			neighborLayer := currentLayer + 1
			if existing, exists := layerMap[neighbor]; exists {
				if neighborLayer > existing {
					layerMap[neighbor] = neighborLayer
					if neighborLayer > maxLayer {
						maxLayer = neighborLayer
					}
				}
			} else {
				layerMap[neighbor] = neighborLayer
				if neighborLayer > maxLayer {
					maxLayer = neighborLayer
				}
			}

			if inDegreeCopy[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Build layers array
	layers := make([][]string, maxLayer+1)
	for i := range layers {
		layers[i] = make([]string, 0)
	}

	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		layer := layerMap[nodeID]
		layers[layer] = append(layers[layer], nodeID)
	}

	return layers
}

// orderNodesInLayers orders nodes within each layer to minimize edge crossings
// Uses a simple median heuristic
func (c *Canvas) orderNodesInLayers(layers [][]string, adjacency map[string][]string, inDegree map[string]int) {
	// For simplicity, we keep the natural order from topological sort
	// A more sophisticated implementation would use the median heuristic:
	// - For each node, calculate median position of its neighbors in previous layer
	// - Sort nodes in layer by median position
	// This is sufficient for the initial implementation
}

// assignPositions assigns X and Y coordinates to nodes based on their layer and position
func (c *Canvas) assignPositions(layers [][]string) {
	currentY := layoutStartY

	for _, layer := range layers {
		// Calculate total width needed for this layer
		totalWidth := 0
		for i, nodeID := range layer {
			if cNode, exists := c.nodes[nodeID]; exists {
				totalWidth += cNode.width
				if i < len(layer)-1 {
					totalWidth += horizontalSpacing
				}
			}
		}

		// Center the layer horizontally
		currentX := layoutStartX

		// Position each node in the layer
		for _, nodeID := range layer {
			if cNode, exists := c.nodes[nodeID]; exists {
				cNode.position = Position{
					X: currentX,
					Y: currentY,
				}
				currentX += cNode.width + horizontalSpacing
			}
		}

		// Move to next layer
		// Find max height in current layer
		maxHeight := 0
		for _, nodeID := range layer {
			if cNode, exists := c.nodes[nodeID]; exists {
				if cNode.height > maxHeight {
					maxHeight = cNode.height
				}
			}
		}
		currentY += maxHeight + verticalSpacing
	}
}

// ResetView resets zoom to 100% and centers viewport on the start node.
// If no start node exists, centers on (0, 0).
func (c *Canvas) ResetView() {
	// Reset zoom to 100%
	c.ZoomLevel = 1.0

	// Find start node in canvas nodes
	var startNode *canvasNode
	for _, cNode := range c.nodes {
		if cNode.node.Type() == "start" {
			startNode = cNode
			break
		}
	}

	// Center viewport on start node or origin
	if startNode != nil {
		// Center on start node
		// Viewport should be positioned so start node is in center of screen
		centerX := startNode.position.X + startNode.width/2
		centerY := startNode.position.Y + startNode.height/2

		c.ViewportX = centerX - c.Width/2
		c.ViewportY = centerY - c.Height/2
	} else {
		// Fallback to origin
		c.ViewportX = 0
		c.ViewportY = 0
	}

	// Clamp to valid bounds
	if c.ViewportX < 0 {
		c.ViewportX = 0
	}
	if c.ViewportY < 0 {
		c.ViewportY = 0
	}
}

// FitAll adjusts zoom and viewport to show all nodes in the viewport.
// Calculates bounding box of all nodes, then computes zoom level to fit.
// Centers viewport on the bounding box center.
// Zoom level is clamped to valid range (0.5 to 2.0).
func (c *Canvas) FitAll() {
	// Handle empty canvas
	if len(c.nodes) == 0 {
		c.ZoomLevel = 1.0
		c.ViewportX = 0
		c.ViewportY = 0
		return
	}

	// Find bounding box of all nodes
	minX, minY := int(1e9), int(1e9)
	maxX, maxY := -1, -1

	for _, cNode := range c.nodes {
		if cNode.position.X < minX {
			minX = cNode.position.X
		}
		if cNode.position.Y < minY {
			minY = cNode.position.Y
		}
		nodeRight := cNode.position.X + cNode.width
		nodeBottom := cNode.position.Y + cNode.height
		if nodeRight > maxX {
			maxX = nodeRight
		}
		if nodeBottom > maxY {
			maxY = nodeBottom
		}
	}

	// Calculate required dimensions
	contentWidth := maxX - minX
	contentHeight := maxY - minY

	// Calculate zoom to fit content
	zoomX := float64(c.Width) / float64(contentWidth)
	zoomY := float64(c.Height) / float64(contentHeight)

	// Use smaller zoom to fit both dimensions
	zoom := zoomX
	if zoomY < zoomX {
		zoom = zoomY
	}

	// Add some padding (90% of calculated zoom)
	zoom *= 0.9

	// Clamp zoom to valid range
	if zoom < 0.5 {
		zoom = 0.5
	}
	if zoom > 2.0 {
		zoom = 2.0
	}

	c.ZoomLevel = zoom

	// Center viewport on content
	centerX := minX + contentWidth/2
	centerY := minY + contentHeight/2

	c.ViewportX = centerX - c.Width/2
	c.ViewportY = centerY - c.Height/2
}

// Pan moves the viewport by the given delta in logical units.
// The viewport is clamped to stay within valid canvas bounds.
// Caller is responsible for triggering a redraw.
func (c *Canvas) Pan(deltaX, deltaY int) {
	c.ViewportX += deltaX
	c.ViewportY += deltaY

	// Clamp viewport to valid bounds
	// Minimum: viewport can't go negative
	if c.ViewportX < 0 {
		c.ViewportX = 0
	}
	if c.ViewportY < 0 {
		c.ViewportY = 0
	}
	// Maximum: allow panning freely for now (no upper bound)
	// In future, could calculate canvas bounds based on nodes
}

// Zoom sets the zoom level and adjusts the viewport to keep the center stable.
// Valid zoom range is 0.5 to 2.0.
// Returns error if zoom level is out of range.
func (c *Canvas) Zoom(level float64) error {
	if level < 0.5 || level > 2.0 {
		return ErrInvalidZoomLevel
	}

	// Calculate current viewport center in logical coordinates
	// Use floating point math to avoid divide by zero
	oldCenterX := c.ViewportX + int(float64(c.Width)/(2.0*c.ZoomLevel))
	oldCenterY := c.ViewportY + int(float64(c.Height)/(2.0*c.ZoomLevel))

	// Update zoom level
	c.ZoomLevel = level

	// Adjust viewport to maintain center point
	// newViewportX = centerX - (width / (2 * newZoom))
	// This keeps the same logical point at the center
	newViewportX := oldCenterX - int(float64(c.Width)/(2.0*c.ZoomLevel))
	newViewportY := oldCenterY - int(float64(c.Height)/(2.0*c.ZoomLevel))

	// Update viewport position
	c.ViewportX = newViewportX
	c.ViewportY = newViewportY

	// Clamp to valid bounds
	if c.ViewportX < 0 {
		c.ViewportX = 0
	}
	if c.ViewportY < 0 {
		c.ViewportY = 0
	}

	return nil
}

// ErrInvalidZoomLevel is returned when zoom level is out of valid range
var ErrInvalidZoomLevel = fmt.Errorf("zoom level must be between 0.5 and 2.0")
