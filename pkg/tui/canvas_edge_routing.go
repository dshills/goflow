package tui

import "github.com/dshills/goflow/pkg/workflow"

// canvasEdge wraps a domain Edge with routing information
type canvasEdge struct {
	// edge is the domain edge
	edge *workflow.Edge
	// routingPoints are intermediate waypoints for the edge path
	routingPoints []Position
	// selected indicates visual selection state
	selected bool
}

// routeEdge calculates the routing points for an edge using orthogonal routing
// Algorithm:
// 1. Start at source node center-bottom
// 2. End at target node center-top
// 3. If aligned vertically: straight line
// 4. Otherwise: horizontal → vertical → horizontal (up to 3 segments)
func (c *Canvas) routeEdge(edge *canvasEdge) {
	fromNode, fromExists := c.nodes[edge.edge.FromNodeID]
	toNode, toExists := c.nodes[edge.edge.ToNodeID]

	if !fromExists || !toExists {
		// Nodes don't exist yet, skip routing
		edge.routingPoints = make([]Position, 0)
		return
	}

	// Calculate source and target points
	// Source: center-bottom of from node
	sourceX := fromNode.position.X + fromNode.width/2
	sourceY := fromNode.position.Y + fromNode.height

	// Target: center-top of to node
	targetX := toNode.position.X + toNode.width/2
	targetY := toNode.position.Y

	// Build routing points based on relative positions
	routingPoints := make([]Position, 0, 4)

	// Start point
	routingPoints = append(routingPoints, Position{X: sourceX, Y: sourceY})

	// Case 1: Target is directly below source (vertical alignment)
	// Just draw a straight line
	if sourceX == targetX {
		// Straight vertical line - no intermediate points needed
	} else if targetY > sourceY {
		// Case 2: Target is below and to the side
		// Route: horizontal → vertical → horizontal
		midY := (sourceY + targetY) / 2

		// Horizontal segment from source
		routingPoints = append(routingPoints, Position{X: sourceX, Y: midY})
		// Vertical segment to target level
		routingPoints = append(routingPoints, Position{X: targetX, Y: midY})
	} else {
		// Case 3: Target is above source (backward edge)
		// Route around: down → horizontal → down → horizontal
		gapY := 2 // Gap below source node
		routingPoints = append(routingPoints, Position{X: sourceX, Y: sourceY + gapY})

		// Horizontal segment
		if targetX > sourceX {
			// Target is to the right
			routingPoints = append(routingPoints, Position{X: targetX, Y: sourceY + gapY})
		} else {
			// Target is to the left
			routingPoints = append(routingPoints, Position{X: targetX, Y: sourceY + gapY})
		}

		// Vertical segment down to target
		routingPoints = append(routingPoints, Position{X: targetX, Y: targetY})
	}

	// End point
	routingPoints = append(routingPoints, Position{X: targetX, Y: targetY})

	edge.routingPoints = routingPoints
}

// getEdgeDirection returns the arrow direction character for an edge segment
// based on the direction from 'from' to 'to'
func getEdgeDirection(from, to Position) string {
	if to.Y > from.Y {
		return "▼" // Down
	} else if to.Y < from.Y {
		return "▲" // Up
	} else if to.X > from.X {
		return "►" // Right
	} else if to.X < from.X {
		return "◄" // Left
	}
	return "●" // Point (same position)
}

// getEdgeLineChar returns the Unicode box drawing character for a line segment
func getEdgeLineChar(from, to Position, isVertical bool) string {
	if isVertical {
		return "│"
	}
	return "─"
}

// getEdgeCornerChar returns the Unicode box drawing character for a corner
func getEdgeCornerChar(from, mid, to Position) string {
	// Determine the corner type based on directions
	fromToMid := Position{X: mid.X - from.X, Y: mid.Y - from.Y}
	midToTo := Position{X: to.X - mid.X, Y: to.Y - mid.Y}

	// Horizontal to vertical down
	if fromToMid.Y == 0 && midToTo.Y > 0 {
		if fromToMid.X > 0 {
			return "┐" // Right then down
		}
		return "┌" // Left then down
	}

	// Horizontal to vertical up
	if fromToMid.Y == 0 && midToTo.Y < 0 {
		if fromToMid.X > 0 {
			return "┘" // Right then up
		}
		return "└" // Left then up
	}

	// Vertical to horizontal
	if fromToMid.X == 0 && midToTo.X != 0 {
		if fromToMid.Y > 0 {
			if midToTo.X > 0 {
				return "└" // Down then right
			}
			return "┘" // Down then left
		} else {
			if midToTo.X > 0 {
				return "┌" // Up then right
			}
			return "┐" // Up then left
		}
	}

	return "┼" // Default cross
}
