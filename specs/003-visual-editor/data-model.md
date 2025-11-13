# Data Model: Visual Workflow Editor

**Feature**: Complete Visual Workflow Editor
**Date**: 2025-11-12
**Package**: `pkg/tui`

## Overview

This document defines the data structures and relationships for the visual workflow editor UI layer. These are **presentation entities** that wrap the domain model (`pkg/workflow`) with UI-specific state. They do not violate DDD principles as they operate as a pure view layer over the Workflow aggregate.

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      WorkflowBuilder                         │
│  ┌────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │   Canvas   │  │ NodePalette  │  │ PropertyPanel  │      │
│  │            │  │              │  │                │      │
│  └────────────┘  └──────────────┘  └────────────────┘      │
│  ┌────────────┐  ┌──────────────┐  ┌────────────────┐      │
│  │ UndoStack  │  │ HelpPanel    │  │ValidationStatus│      │
│  └────────────┘  └──────────────┘  └────────────────┘      │
└─────────────────────────────────────────────────────────────┘
                         │
                         │ wraps
                         ▼
              ┌─────────────────────┐
              │  workflow.Workflow  │  ← Domain aggregate
              │  workflow.Node      │
              │  workflow.Edge      │
              └─────────────────────┘
```

## Core Entities

### WorkflowBuilder

**Purpose**: Orchestrates all UI components and maintains builder state

**Attributes**:
```go
type WorkflowBuilder struct {
    workflow         *workflow.Workflow    // Domain model (existing)
    canvas           *Canvas               // Visual representation
    palette          *NodePalette          // Node type selector
    propertyPanel    *PropertyPanel        // Property editor
    helpPanel        *HelpPanel            // Help overlay
    selectedNodeID   string                // Currently selected node
    mode             string                // "normal", "edit", "palette", "help"
    edgeCreationMode bool                  // Creating edge in progress
    edgeSourceID     string                // Source node for edge creation
    modified         bool                  // Workflow has unsaved changes
    validationStatus *ValidationStatus     // Validation results
    undoStack        []workflowSnapshot    // Undo history
    redoStack        []workflowSnapshot    // Redo history
    repository       workflow.WorkflowRepository  // Persistence
    keyEnabled       map[string]bool       // Enabled keyboard shortcuts
}
```

**Responsibilities**:
- Coordinate state changes across all panels
- Handle mode transitions (normal → edit → normal)
- Dispatch keyboard events to active panel
- Maintain undo/redo history
- Trigger workflow validation
- Persist changes to repository

**Lifecycle**:
1. Created with `NewWorkflowBuilder(workflow, repository)`
2. Load existing workflow or create new
3. User interaction updates state
4. Save writes to repository
5. Closed when user exits builder

---

### Canvas

**Purpose**: Manages node positioning, viewport, and rendering

**Attributes**:
```go
type Canvas struct {
    Width          int                      // Terminal width (logical units)
    Height         int                      // Terminal height (logical units)
    ViewportX      int                      // Viewport offset X
    ViewportY      int                      // Viewport offset Y
    ZoomLevel      float64                  // Zoom factor (0.5 to 2.0)
    nodes          map[string]canvasNode    // Node positions
    edges          []canvasEdge             // Edge routing
    selectedID     string                   // Selected node ID
}
```

**Responsibilities**:
- Position nodes on canvas (auto-layout or manual)
- Render nodes and edges to terminal
- Handle viewport panning and zooming
- Translate terminal clicks/keys to logical coordinates
- Detect node selection from coordinates

**Coordinate conversion**:
```go
// Logical to terminal
terminalX := (logicalX - ViewportX) * ZoomLevel
terminalY := (logicalY - ViewportY) * ZoomLevel

// Terminal to logical
logicalX := (terminalX / ZoomLevel) + ViewportX
logicalY := (terminalY / ZoomLevel) + ViewportY
```

---

### canvasNode

**Purpose**: Wraps domain Node with rendering state

**Attributes**:
```go
type canvasNode struct {
    node     workflow.Node    // Domain node (immutable from TUI)
    position Position         // Logical coordinates (X, Y)
    width    int              // Rendered width in characters
    height   int              // Rendered height in lines
    selected bool             // Visual selection state
    highlighted bool          // Temporary highlight (hover, focus)
    validationStatus string   // "valid", "warning", "error"
}
```

**Relationships**:
- References `workflow.Node` (does not own or modify)
- Contained by `Canvas`

**Rendering**:
```
┌──────────────────┐
│   MCP Tool       │  ← Rendered based on node.Type
│  filesystem.read │  ← Displays node.Name and key properties
└──────────────────┘
```

**Visual states**:
- Normal: Gray border
- Selected: Blue border, highlighted background
- Error: Red border, error icon
- Warning: Yellow border, warning icon

---

### canvasEdge

**Purpose**: Wraps domain Edge with routing information

**Attributes**:
```go
type canvasEdge struct {
    edge          *workflow.Edge  // Domain edge
    fromPos       Position        // Source node position
    toPos         Position        // Target node position
    routingPoints []Position      // Intermediate waypoints
    selected      bool            // Visual selection state
}
```

**Routing algorithm** (simplified Orthogonal):
1. Start at source node center-bottom
2. If target directly below: straight line
3. Otherwise: horizontal segment → vertical segment → horizontal segment
4. Avoid overlapping other nodes (add waypoints)

**Rendering**:
```
Node A ────┐     ← Horizontal segment
           │
           ▼     ← Vertical segment with arrow
        Node B
```

---

### NodePalette

**Purpose**: Node type selection interface

**Attributes**:
```go
type NodePalette struct {
    nodeTypes     []nodeTypeInfo   // Available node types
    selectedIndex int               // Currently highlighted type
    filterText    string            // Search filter
    visible       bool              // Palette open/closed
}

type nodeTypeInfo struct {
    typeName    string   // "MCP Tool", "Transform", "Condition", etc.
    description string   // Short help text
    icon        string   // Unicode icon (optional)
    defaultConfig map[string]interface{}  // Default node properties
}
```

**Node types** (from workflow domain):
1. MCP Tool - Execute MCP server tool
2. Transform - Data transformation (JSONPath, template, jq)
3. Condition - Conditional branching
4. Loop - Iteration over collections
5. Parallel - Concurrent execution
6. Start - Entry point (system-generated)
7. End - Exit point with output

**Filtering**:
- User types text → filter nodeTypes by substring match (case-insensitive)
- Example: "trans" matches "Transform"

---

### PropertyPanel

**Purpose**: Node property editor with validation

**Attributes**:
```go
type PropertyPanel struct {
    node              workflow.Node        // Node being edited (copy)
    fields            []propertyField      // Editable fields
    editIndex         int                  // Currently focused field
    visible           bool                 // Panel open/closed
    validationMessage string               // Current error/warning
    dirty             bool                 // Unsaved changes
}

type propertyField struct {
    label        string                   // Display name
    value        string                   // Current value
    required     bool                     // Must be non-empty
    valid        bool                     // Passes validation
    fieldType    string                   // "text", "expression", "condition", "jsonpath", "template"
    validationFn func(string) error       // Validation function
    helpText     string                   // Syntax hints
}
```

**Field types** (from research.md):
1. **text**: Simple string (workflow name, node name)
2. **expression**: Sandboxed expr evaluation
3. **condition**: Boolean expression
4. **jsonpath**: Data query ($.path.to.field)
5. **template**: String with ${} interpolation

**Validation timing**:
- Real-time on field blur (not every keystroke)
- Show errors below field with red color
- Prevent save if any required field invalid

**Layout**:
```
Property Panel
──────────────
Name: [workflow-step-1]

Server ID: [local-server]

Tool Name: [filesystem.read]

Input Mapping: [$.data]
  ✓ Valid JSONPath

Output Variable: [file_contents]

[Save (Ctrl+S)]  [Cancel (Esc)]
```

---

### UndoStack

**Purpose**: Undo/redo functionality

**Attributes**:
```go
type UndoStack struct {
    snapshots []workflowSnapshot  // Circular buffer
    cursor    int                  // Current position
    capacity  int                  // Max snapshots (100)
}

type workflowSnapshot struct {
    nodes       []workflow.Node   // Deep copy of nodes
    edges       []*workflow.Edge  // Deep copy of edges
    canvasState map[string]Position  // Node positions
    timestamp   time.Time         // When snapshot created
}
```

**Operations**:
- `Push(snapshot)`: Add new snapshot, clear redo stack
- `Undo()`: Move cursor back, return previous snapshot
- `Redo()`: Move cursor forward, return next snapshot
- `CanUndo()`: Check if undo available
- `CanRedo()`: Check if redo available

**Memory management**:
- Circular buffer: oldest snapshots overwritten when capacity reached
- Delta encoding considered but not implemented (too complex for initial version)
- Max 100 snapshots × ~1KB = ~100KB overhead

---

### ValidationStatus

**Purpose**: Workflow validation results

**Attributes**:
```go
type ValidationStatus struct {
    valid         bool                    // Overall valid/invalid
    errors        []validationError       // Blocking errors
    warnings      []validationWarning     // Non-blocking warnings
    lastValidated time.Time               // Validation timestamp
}

type validationError struct {
    nodeID      string   // Node with error (or "" for global)
    errorType   string   // "missing_field", "circular_dependency", "disconnected"
    message     string   // Human-readable message
}

type validationWarning struct {
    nodeID   string
    message  string
}
```

**Validation rules** (from workflow domain):
1. No circular dependencies (detect cycles)
2. All nodes reachable from start node
3. All required fields populated
4. Valid expressions/JSONPath/templates
5. Condition nodes have exactly 2 outgoing edges (true/false)
6. Edge targets exist

**Validation triggers**:
- After any node/edge add/delete/modify
- On explicit validate command ('v' key)
- Before save
- Async (non-blocking) for large workflows

---

### HelpPanel

**Purpose**: Context-sensitive help overlay

**Attributes**:
```go
type HelpPanel struct {
    visible        bool                // Panel open/closed
    currentSection string              // "general", "node", "edit", "palette"
    keyBindings    []HelpKeyBinding    // Available shortcuts
}

type HelpKeyBinding struct {
    keys        []string   // ["h", "j", "k", "l"]
    description string     // "Navigate canvas"
    category    string     // "Navigation", "Editing", "Workflow"
    mode        string     // "normal", "edit", "*" (all modes)
}
```

**Content sections**:
1. **General**: '?', 'Esc', 'q' - Always available
2. **Navigation**: h/j/k/l, Shift+arrows, +/-, 0, f
3. **Node operations**: a, d, c, y, p
4. **Editing**: Enter, Ctrl+S
5. **Workflow**: s, v, u, Ctrl+R, t

**Context-sensitive**:
- Normal mode: Show all shortcuts
- Edit mode: Show field editing shortcuts + syntax hints
- Palette mode: Show node type descriptions

**Layout**:
```
────── Help (Press ? to close) ──────

Navigation
  h/j/k/l    Move selection
  Shift+←↑↓→ Pan canvas
  +/-        Zoom in/out

Node Operations
  a          Add node
  d          Delete node
  c          Create edge

... (scrollable)
```

---

## State Transitions

### Mode Transitions

```
┌────────┐  'a'   ┌─────────┐
│ NORMAL ├───────►│ PALETTE │
│        │◄───────┤         │
└───┬────┘  Esc   └─────────┘
    │
    │ Enter (on node)
    ▼
┌────────┐
│  EDIT  │
└───┬────┘
    │
    │ '?'
    ▼
┌────────┐
│  HELP  │
└───┬────┘
    │
    │ '?' (toggle)
    │
    └───► NORMAL
```

### Workflow State

```
┌─────────┐  Add/Edit/Delete   ┌──────────┐
│  Clean  ├──────────────────►│ Modified │
└─────────┘                    └────┬─────┘
     ▲                              │
     │            Save              │
     └──────────────────────────────┘
```

## Persistence Mapping

### Canvas to YAML

Canvas state (node positions, zoom) is persisted as workflow metadata:

```yaml
version: "1.0"
name: "my-workflow"
metadata:
  canvas:
    zoom: 1.0
    viewport:
      x: 0
      y: 0
    positions:
      node-1: {x: 10, y: 5}
      node-2: {x: 30, y: 15}
      node-3: {x: 50, y: 25}
nodes:
  - id: node-1
    type: mcp_tool
    # ... node config
edges:
  - from: node-1
    to: node-2
```

### Loading Workflow

1. Read YAML → `workflow.Workflow` (domain model)
2. Extract canvas metadata → `Canvas`
3. If no positions: run auto-layout algorithm
4. Create `canvasNode` for each `workflow.Node`
5. Create `canvasEdge` for each `workflow.Edge`
6. Render initial state

## Performance Considerations

### Rendering Optimization

**Viewport culling**:
```go
// Only render nodes visible in viewport
for _, node := range canvas.nodes {
    if nodeInViewport(node, canvas) {
        renderNode(node, screen)
    }
}
```

**Dirty flag**:
- Only re-render when state changes
- Track which components need redraw
- Batch multiple changes before render

**Target**: < 16ms per frame (60 FPS)

### Memory Optimization

**Undo stack**:
- Circular buffer prevents unbounded growth
- Cap at 100 snapshots (~100KB)

**Large workflows**:
- Lazy load node content (don't render all properties always)
- Virtualized rendering for 100+ nodes
- Simple rendering mode (no gradients, fewer Unicode chars)

## Testing Strategy

### Unit Tests

```go
func TestCanvasAutoLayout(t *testing.T) {
    // Test topological hierarchical layout
}

func TestUndoStack(t *testing.T) {
    // Test push, undo, redo, capacity
}

func TestPropertyPanelValidation(t *testing.T) {
    // Test each field type validation
}

func TestNodePaletteFiltering(t *testing.T) {
    // Test substring filtering
}
```

### Integration Tests

```go
func TestWorkflowBuilder(t *testing.T) {
    // Test complete workflow: add node, edit, connect, save
}

func TestModeTransitions(t *testing.T) {
    // Test normal → edit → normal flow
}
```

### TUI Interaction Tests

```go
func TestKeyboardNavigation(t *testing.T) {
    // Simulate key events, verify state changes
}
```

## Next Phase

Data model complete. Ready for Phase 1: Contracts.
