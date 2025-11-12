# Research: Visual Workflow Editor Design Decisions

**Feature**: Complete Visual Workflow Editor
**Date**: 2025-11-12
**Status**: Complete

## Overview

This document captures research findings and design decisions for the visual workflow editor implementation. Each decision is documented with the chosen approach, rationale, and alternatives considered.

## Decision 1: Canvas Rendering Strategy

**Chosen**: Unicode box drawing characters with simple text-based layout

**Rationale**:
- Unicode box drawing characters (─│┌┐└┘├┤┬┴┼) are widely supported in modern terminals (iTerm2, Terminal.app, Windows Terminal, GNOME Terminal)
- Provides clean visual appearance without complex graphics
- Maintains readability at different terminal sizes
- Goterm library already has utilities for box drawing
- Performance is excellent (simple character writes, no complex rendering)
- Degrades gracefully to ASCII fallback on older terminals

**Implementation approach**:
```
┌──────────────────┐
│   MCP Tool       │  ← Node box using Unicode
│  filesystem.read │
└──────────────────┘
         │
         ▼            ← Edge using box drawing chars
┌──────────────────┐
│   Transform      │
│  JSONPath        │
└──────────────────┘
```

**Alternatives considered**:
- **ASCII art only**: Rejected - Less visually appealing, harder to distinguish components
- **Complex TUI widgets**: Rejected - Overkill for workflow visualization, adds complexity
- **Graph visualization libraries**: Rejected - Too heavyweight, external dependencies

**References**:
- Unicode box drawing: U+2500 to U+257F
- Goterm box drawing utils: Existing in `pkg/tui/components`

---

## Decision 2: Edge Routing Algorithm

**Chosen**: Orthogonal (Manhattan-style) routing with simple heuristics

**Rationale**:
- Orthogonal edges (only horizontal and vertical segments) are easier to read than diagonal lines in terminal
- Simple to implement: connect nodes with up to 3 segments (horizontal → vertical → horizontal)
- Performance is O(1) per edge (no complex pathfinding required)
- Provides clean visual flow for workflow diagrams
- Works well with terminal character grid constraints

**Implementation approach**:
```
Node A ────┐
           │
           │  ← Vertical segment
           │
           └───► Node B
```

**Edge routing rules**:
1. If target is directly below source: straight vertical line
2. If target is to the right: horizontal → vertical → horizontal
3. If target is to the left or above: route around with 2-3 segments
4. Avoid node overlaps by routing around bounding boxes

**Alternatives considered**:
- **Straight lines (diagonal)**: Rejected - Looks messy in terminal, hard to follow
- **Bezier curves/splines**: Rejected - Not supported in terminal, requires approximation
- **Automatic pathfinding (A*)**: Rejected - Overkill for small workflows, performance cost

**References**:
- Orthogonal graph drawing: https://en.wikipedia.org/wiki/Orthogonal_graph_drawing

---

## Decision 3: Auto-layout Algorithm

**Chosen**: Topological hierarchical layout (Sugiyama framework simplified)

**Rationale**:
- Workflow graphs are DAGs (directed acyclic graphs) by design, perfect for hierarchical layout
- Topological sort provides natural left-to-right or top-to-bottom flow
- Algorithm is deterministic (same workflow always produces same layout)
- Performance is O(V + E) where V=nodes, E=edges - excellent for 100+ node workflows
- Produces readable layouts with clear execution flow

**Implementation approach**:
1. Perform topological sort on nodes
2. Assign layers (Y coordinate) based on longest path from start node
3. Within each layer, order nodes to minimize edge crossings (simple heuristic: median of connected nodes)
4. Assign X coordinates based on layer ordering
5. Add spacing between layers and nodes for readability

**Layout parameters**:
- Horizontal spacing: 4 characters between nodes
- Vertical spacing: 2 lines between layers
- Node width: Dynamic based on content (min 16, max 40 chars)
- Node height: 3-5 lines depending on content

**Alternatives considered**:
- **Force-directed layout**: Rejected - Non-deterministic, slow for large graphs, poor terminal fit
- **Grid-based layout**: Rejected - Inflexible, wasted space, doesn't respect workflow structure
- **Random with collision avoidance**: Rejected - Unpredictable, no semantic meaning

**References**:
- Sugiyama framework: "Methods for Visual Understanding of Hierarchical System Structures" (1981)
- Simplified impl: https://blog.disy.net/sugiyama-method/

---

## Decision 4: Property Panel Field Types

**Chosen**: Five field types with specialized validation

**Field Types**:
1. **Text**: Simple string input (names, descriptions)
   - Validation: Required/optional, max length, regex patterns

2. **Expression**: Sandboxed expressions for data manipulation
   - Validation: Parse with `expr-lang/expr`, check for unsafe operations
   - Syntax hints: Show available variables and functions

3. **Condition**: Boolean expressions for conditional nodes
   - Validation: Must evaluate to boolean, parse with expr
   - Syntax hints: Comparison operators, logical operators

4. **JSONPath**: Query expressions for data extraction
   - Validation: Parse with `gjson` library, check syntax
   - Syntax hints: Show JSONPath operators ($, @, ., [], *)

5. **Template**: Template strings for string interpolation
   - Validation: Parse `${}` placeholders, check variable existence
   - Syntax hints: Show available variables

**Validation strategy**:
- Real-time validation on field blur (not on every keystroke to avoid annoyance)
- Show validation errors below field with red color
- Provide syntax hints in help text (shown when field focused)
- Use existing validation functions from `pkg/transform` and `pkg/expression`

**Alternatives considered**:
- **Single generic text field**: Rejected - No validation, poor UX
- **Separate editors for each type**: Rejected - Too complex, too many modal transitions
- **Inline validation on every key**: Rejected - Annoying, performance cost

---

## Decision 5: Undo/Redo Granularity

**Chosen**: Operation-level undo (coarse-grained)

**Undoable operations**:
- Add node
- Delete node
- Move node (single drag operation, not each step)
- Create edge
- Delete edge
- Edit node properties (entire property panel save, not each field)
- Paste nodes
- Apply template

**Not undoable** (redo from scratch if needed):
- Canvas pan/zoom (view state, not data state)
- Selection changes
- Panel open/close
- Help overlay toggle

**Implementation**:
- Snapshot workflow state (nodes + edges) after each operation
- Store in circular buffer (max 100 snapshots)
- Each snapshot stores only changed data (not full copy)
- Undo cursor moves backward, redo moves forward

**Memory estimate**:
- ~1KB per snapshot (delta encoding)
- 100 snapshots = ~100KB max memory overhead
- Acceptable for all workflows

**Alternatives considered**:
- **Fine-grained (every keystroke)**: Rejected - Excessive memory, annoying UX
- **Command pattern with reverse operations**: Rejected - Complex, error-prone
- **Git-like commit model**: Rejected - Overkill, steep learning curve

---

## Decision 6: Canvas Coordinate System

**Chosen**: Logical units (independent of terminal cells)

**Coordinate system**:
- Logical units: 1 unit = 1 "character width" at 100% zoom
- Node positions stored in logical coordinates
- At render time: Convert to terminal cells using zoom factor
- Zoom levels: 50%, 75%, 100%, 125%, 150%, 200%

**Zoom scaling**:
```go
terminalX := int(logicalX * zoomFactor)
terminalY := int(logicalY * zoomFactor)
```

**Benefits**:
- Zoom in/out without changing node positions
- Terminal resize doesn't affect logical layout
- Easy to persist positions in YAML (zoom-independent)
- Canvas can be arbitrarily large (not bound to terminal size)

**Viewport**:
- Viewport is a window into the logical canvas
- Pan moves viewport, not nodes
- Viewport stored as (offsetX, offsetY, width, height) in logical units

**Alternatives considered**:
- **Terminal cells directly**: Rejected - Zoom requires recalculating all positions
- **Virtual pixels**: Rejected - No benefit over logical units, adds conversion complexity
- **Percentage-based**: Rejected - Doesn't scale well, fractional positions

---

## Decision 7: Keyboard Shortcut Conflicts

**Chosen**: Mode-based keyboard handling with clear precedence

**Modes**:
1. **Normal mode** (default): Canvas navigation, node selection, workflow operations
2. **Edit mode**: Property panel field editing
3. **Palette mode**: Node type selection
4. **Help mode**: Help overlay (read-only)

**Precedence rules**:
1. Help mode shortcuts (? to toggle) override all other modes
2. Edit mode shortcuts override normal mode (Esc to exit edit mode)
3. Palette mode shortcuts override normal mode (Esc to cancel)
4. Normal mode shortcuts only active when no panels open

**Conflict resolution**:
- Global shortcuts: '?', 'Esc', 'q' (quit) - work in all modes with consistent behavior
- Mode-specific shortcuts: 'a' (add node in normal, 'a' character in edit) - context-dependent
- Modifier keys: Ctrl/Alt for operations that might conflict (Ctrl+S for save)

**Shortcut categories**:
- **Navigation**: h/j/k/l (move), Shift+arrows (pan), +/- (zoom)
- **Node operations**: a (add), d (delete), c (connect), y (yank), p (paste)
- **Editing**: Enter (edit properties), Esc (cancel), Ctrl+S (save)
- **Workflow**: s (save workflow), v (validate), u (undo), Ctrl+R (redo)
- **View**: ? (help), t (templates), 0 (reset view), f (fit all)

**Visual feedback**:
- Mode indicator in status bar: "NORMAL", "EDIT", "PALETTE", "HELP"
- Available shortcuts shown in context-sensitive help
- Invalid shortcuts show brief message in status bar

**Alternatives considered**:
- **Single mode with modifiers**: Rejected - Too many modifier combinations, hard to remember
- **Emacs-style key chords**: Rejected - Steep learning curve, not intuitive
- **Mouse-only**: Rejected - Violates TUI keyboard-first philosophy

---

## Summary Table

| Decision | Choice | Key Benefit | Performance Impact |
|----------|--------|-------------|-------------------|
| Canvas Rendering | Unicode box drawing | Visual clarity | Minimal (~1ms per node) |
| Edge Routing | Orthogonal | Readability | O(1) per edge |
| Auto-layout | Hierarchical | Deterministic flow | O(V + E) |
| Property Fields | 5 specialized types | Type-safe validation | Validation ~10ms |
| Undo/Redo | Operation-level | Usability/memory balance | ~50ms per operation |
| Coordinates | Logical units | Zoom-independent | Minimal (conversion) |
| Shortcuts | Mode-based | Clear context | Zero |

## Implementation Notes

**Testing strategy**:
- Unit tests for each algorithm (layout, routing, undo stack)
- Table-driven tests for property validation
- Integration tests for keyboard navigation
- Performance benchmarks for rendering and layout

**Goterm integration**:
- Use existing `goterm.Screen` for rendering
- Leverage `goterm.Component` interface for panels
- Use `goterm.Event` for keyboard handling
- Build on existing `pkg/tui/components` (List, Modal, Panel)

**Accessibility**:
- Terminal colors chosen for colorblind accessibility (avoid red/green alone)
- ASCII fallback for terminals without Unicode support
- Keyboard-only navigation (no mouse required)
- Clear visual and textual feedback for all operations

**Extensibility**:
- New node types: Add to `NodePalette.nodeTypes` list
- New field types: Implement `propertyField` interface
- New shortcuts: Add to keyboard binding registry
- New templates: Add to `templates.go` predefined list

## Next Phase

All design decisions resolved. Ready to proceed to Phase 1: Data Model and Contracts.
