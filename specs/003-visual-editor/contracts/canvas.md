# Contract: Canvas Component

**Component**: Canvas
**Package**: `pkg/tui`
**Purpose**: Manage node positioning, viewport, rendering, and user interaction with the visual workflow graph

## Interface

```go
// Canvas manages the visual workflow graph
type Canvas struct {
    Width      int
    Height     int
    ViewportX  int
    ViewportY  int
    ZoomLevel  float64
    nodes      map[string]canvasNode
    edges      []canvasEdge
    selectedID string
}

// NewCanvas creates a canvas with the given dimensions
func NewCanvas(width, height int) *Canvas

// AddNode adds a node to the canvas at the specified position
// Returns error if node already exists
func (c *Canvas) AddNode(node workflow.Node, pos Position) error

// RemoveNode removes a node from the canvas
// Also removes all edges connected to the node
func (c *Canvas) RemoveNode(nodeID string) error

// MoveNode updates a node's position
// Returns error if node doesn't exist
func (c *Canvas) MoveNode(nodeID string, newPos Position) error

// AddEdge adds an edge between two nodes
// Calculates routing automatically
func (c *Canvas) AddEdge(edge *workflow.Edge) error

// RemoveEdge removes an edge from the canvas
func (c *Canvas) RemoveEdge(edgeID string) error

// SelectNode sets the selected node
func (c *Canvas) SelectNode(nodeID string) error

// GetSelectedNode returns the currently selected node
func (c *Canvas) GetSelectedNode() *canvasNode

// Pan moves the viewport by the given delta
func (c *Canvas) Pan(deltaX, deltaY int)

// Zoom sets the zoom level (0.5 to 2.0)
func (c *Canvas) Zoom(level float64) error

// FitAll adjusts zoom and viewport to show all nodes
func (c *Canvas) FitAll()

// ResetView resets to default zoom and centers on start node
func (c *Canvas) ResetView()

// Render draws the canvas to the terminal screen
func (c *Canvas) Render(screen *goterm.Screen) error

// AutoLayout positions nodes using hierarchical layout algorithm
func (c *Canvas) AutoLayout()

// NodeAtPosition returns the node ID at the given terminal coordinates
// Returns "" if no node at position
func (c *Canvas) NodeAtPosition(termX, termY int) string
```

## Behavior Contracts

### AddNode

**Preconditions**:
- Node ID must be unique (not already in canvas)
- Position coordinates must be valid logical units

**Postconditions**:
- Node added to `nodes` map
- Canvas marked as needing redraw
- Node positioned at specified coordinates

**Error conditions**:
- Returns error if node with same ID already exists

### RemoveNode

**Preconditions**:
- Node ID must exist in canvas

**Postconditions**:
- Node removed from `nodes` map
- All edges connected to node removed from `edges` list
- If removed node was selected, selection cleared
- Canvas marked as needing redraw

**Error conditions**:
- Returns error if node ID doesn't exist

### MoveNode

**Preconditions**:
- Node ID must exist in canvas
- New position must be valid logical coordinates

**Postconditions**:
- Node position updated
- All edges connected to node recalculated (routing updated)
- Canvas marked as needing redraw

**Error conditions**:
- Returns error if node ID doesn't exist
- Returns error if new position out of bounds (negative coordinates)

### AddEdge

**Preconditions**:
- Source and target nodes must exist in canvas
- Edge must not create circular dependency

**Postconditions**:
- Edge added to `edges` list
- Edge routing calculated using orthogonal algorithm
- Canvas marked as needing redraw

**Error conditions**:
- Returns error if source or target node doesn't exist
- Returns error if edge would create cycle (circular dependency check)

### AutoLayout

**Preconditions**:
- Canvas contains at least one node (start node)

**Postconditions**:
- All nodes repositioned using topological hierarchical layout
- Node positions assigned based on execution flow layers
- Spacing between nodes ensures readability
- Canvas marked as needing redraw

**Algorithm**:
1. Topological sort to determine layers
2. Assign Y coordinate based on layer (layer * verticalSpacing)
3. Within each layer, order nodes to minimize edge crossings
4. Assign X coordinate based on layer position (index * horizontalSpacing)
5. Recalculate all edge routing

**Parameters**:
- Horizontal spacing: 4 character widths between nodes
- Vertical spacing: 2 lines between layers
- Minimum node width: 16 characters
- Minimum node height: 3 lines

### Render

**Preconditions**:
- Screen must be initialized and ready for drawing
- Canvas dimensions must match screen dimensions (or be smaller)

**Postconditions**:
- All visible nodes rendered to screen with Unicode box drawing
- All visible edges rendered with orthogonal routing
- Selected node highlighted with blue border
- Error/warning nodes marked with red/yellow border
- Viewport indicators shown if canvas extends beyond screen

**Performance**:
- MUST complete in < 16ms for workflows with < 100 nodes (60 FPS)
- MUST use viewport culling (only render visible nodes/edges)
- MUST support performance mode for large workflows (simplified rendering)

**Rendering order**:
1. Clear canvas area
2. Draw edges (behind nodes)
3. Draw nodes
4. Draw selection highlights
5. Draw viewport indicators (scrollbars, minimap)

## Coordinate System

### Logical Coordinates

Canvas uses logical units independent of terminal character cells:
- 1 logical unit = 1 character width at 100% zoom
- Node positions stored in logical coordinates
- Viewport offset in logical coordinates

### Terminal Coordinates

At render time, logical coordinates converted to terminal cells:
```go
terminalX := int((logicalX - ViewportX) * ZoomLevel)
terminalY := int((logicalY - ViewportY) * ZoomLevel)
```

### Zoom Levels

Supported zoom levels: 50%, 75%, 100%, 125%, 150%, 200%
- Below 50%: Text becomes unreadable
- Above 200%: Too few nodes visible

## Edge Routing Algorithm

### Orthogonal Routing

Edges routed using Manhattan-style (horizontal and vertical segments only):

**Simple case** (target directly below source):
```
Node A
   │
   ▼
Node B
```

**Complex case** (target to right and below):
```
Node A ────┐
           │
           ▼
        Node B
```

**Routing steps**:
1. Start at source node center-bottom
2. End at target node center-top
3. If aligned vertically: straight line
4. Otherwise: H → V → H (up to 3 segments)
5. Avoid node bounding boxes (route around if needed)

**Performance**: O(1) per edge (no pathfinding)

## Visual States

### Node Visual States

| State | Border Color | Background | Icon |
|-------|-------------|------------|------|
| Normal | Gray | Default | Type icon |
| Selected | Blue | Light blue | Type icon |
| Error | Red | Light red | ❌ |
| Warning | Yellow | Light yellow | ⚠️ |
| Highlighted | Cyan | Default | Type icon |

### Edge Visual States

| State | Line Style | Color | Arrow |
|-------|-----------|-------|-------|
| Normal | Solid | Gray | ▼ |
| Selected | Thick | Blue | ▼ |
| Conditional true | Solid | Green | ▼ |
| Conditional false | Solid | Red | ▼ |

## Performance Guarantees

### Rendering Performance

- **< 16ms per frame** for workflows with < 100 nodes (60 FPS)
- **< 100ms per frame** for workflows with < 1000 nodes (10 FPS acceptable)
- **Viewport culling**: Only render nodes within visible viewport
- **Performance mode**: Simplified rendering (ASCII only, no gradients)

### Layout Performance

- **AutoLayout**: O(V + E) where V=nodes, E=edges
- **< 500ms** for workflows with < 100 nodes
- **< 2s** for workflows with < 1000 nodes

### Memory Usage

- **Node storage**: ~200 bytes per node (canvasNode struct + strings)
- **Edge storage**: ~100 bytes per edge (canvasEdge struct + routing points)
- **Total**: ~20KB for 100-node workflow

## Testing Requirements

### Unit Tests

- `TestCanvasAddNode`: Add single node, verify position
- `TestCanvasRemoveNode`: Remove node, verify edges cleaned up
- `TestCanvasMoveNode`: Move node, verify edge recalculation
- `TestCanvasAutoLayout`: Layout workflow, verify layer assignment
- `TestCanvasZoom`: Zoom in/out, verify coordinate conversion
- `TestCanvasPan`: Pan viewport, verify rendering bounds
- `TestCanvasFitAll`: Fit all nodes, verify viewport/zoom calculation

### Integration Tests

- `TestCanvasRenderingPerformance`: Benchmark rendering 100-node workflow
- `TestCanvasAutoLayoutPerformance`: Benchmark layout 100-node workflow
- `TestCanvasInteraction`: Simulate click → select → move sequence

### Edge Cases

- Empty canvas (no nodes)
- Single node (start only)
- Large workflow (200+ nodes)
- Deep hierarchy (20+ layers)
- Wide workflow (100+ nodes in single layer)
- Terminal resize during interaction
- Zoom to extreme levels (50%, 200%)

## Dependencies

**Internal**:
- `pkg/workflow`: workflow.Node, workflow.Edge, workflow.Workflow
- `pkg/tui/components`: Box, Line components for rendering

**External**:
- `github.com/dshills/goterm`: Terminal rendering library

## Example Usage

```go
// Create canvas
canvas := NewCanvas(80, 40)

// Add nodes
startNode := workflow.NewNode("start", workflow.NodeTypeStart)
toolNode := workflow.NewNode("tool-1", workflow.NodeTypeMCPTool)

canvas.AddNode(startNode, Position{X: 10, Y: 5})
canvas.AddNode(toolNode, Position{X: 10, Y: 15})

// Connect nodes
edge := workflow.NewEdge("start", "tool-1")
canvas.AddEdge(edge)

// Auto-layout
canvas.AutoLayout()

// Render
screen := goterm.NewScreen()
canvas.Render(screen)
screen.Refresh()
```

## Notes

- Canvas is stateful (maintains nodes, edges, viewport)
- Canvas does NOT modify workflow domain model directly
- Canvas operations are synchronous (no goroutines)
- Canvas is NOT thread-safe (single-threaded TUI)
- Terminal resize triggers canvas redraw (handled by WorkflowBuilder)
