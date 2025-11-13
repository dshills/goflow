# Feature Specification: Complete Visual Workflow Editor

**Feature Branch**: `003-visual-editor`
**Created**: 2025-11-12
**Status**: Draft
**Input**: User description: "finish the visual editor with all features"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Visual Node Placement and Connection (Priority: P1)

A developer wants to visually construct a workflow by placing nodes on a canvas and connecting them with edges to define execution flow, using intuitive drag-and-drop or keyboard-based positioning.

**Why this priority**: This is the core value proposition of the visual editor - enabling visual workflow construction. Without this, users must manually edit YAML files, which defeats the purpose of having a TUI builder. This is the MVP that delivers immediate value.

**Independent Test**: Can be fully tested by creating a new workflow, adding 3-5 nodes of different types (MCP tool, transform, condition), positioning them on the canvas, connecting them with edges, and verifying the workflow structure is correctly saved to YAML format.

**Acceptance Scenarios**:

1. **Given** the workflow builder is open with an empty canvas, **When** the user presses 'a' to add a node and selects "MCP Tool" from the palette, **Then** a new MCP tool node appears on the canvas at the default position with a unique ID
2. **Given** two nodes exist on the canvas, **When** the user selects the first node and presses 'c' to create an edge, then selects the second node, **Then** an edge is drawn connecting the two nodes with an arrow indicating direction
3. **Given** a node is selected on the canvas, **When** the user presses 'h/j/k/l' keys (vim-style navigation), **Then** the selected node moves in the corresponding direction (left/down/up/right) by a fixed increment
4. **Given** a workflow contains 5 nodes with 3 edges, **When** the user presses 's' to save, **Then** the workflow is persisted to YAML format with all node configurations and edge connections preserved
5. **Given** a node is selected, **When** the user presses 'd' to delete, **Then** the node and all connected edges are removed from the canvas, and the workflow is marked as modified

---

### User Story 2 - Node Property Editing (Priority: P1)

A developer needs to configure node-specific properties (tool names, parameters, expressions, conditions) through an intuitive property panel without manually typing YAML, with real-time validation feedback.

**Why this priority**: Node placement is useless without configuration. Users must be able to set tool names, input parameters, transformation expressions, and conditional logic. This completes the MVP by making workflows functional. This is independently valuable as it enables full workflow authoring.

**Independent Test**: Can be fully tested by selecting a node, opening the property panel (Enter key), editing various field types (text, expression, condition, JSONPath), validating field contents (invalid JSONPath should show error), saving changes, and verifying the node configuration is updated in the workflow structure.

**Acceptance Scenarios**:

1. **Given** a node is selected on the canvas, **When** the user presses Enter, **Then** a property panel opens on the right side showing all configurable fields for that node type (name, server ID, tool name, input mapping, etc.)
2. **Given** the property panel is open for an MCP Tool node, **When** the user navigates to the "tool name" field and enters "filesystem.read", **Then** the field value is updated and marked as valid
3. **Given** the property panel shows a "JSONPath" field, **When** the user enters an invalid JSONPath expression "$.invalid[", **Then** the field is marked invalid with a red indicator and an error message appears below the field
4. **Given** multiple fields are edited in the property panel, **When** the user presses 'Ctrl+S', **Then** all valid changes are applied to the node, the property panel closes, and the workflow is marked as modified
5. **Given** a Transform node is selected, **When** the property panel opens, **Then** the panel displays fields for transformation type (JSONPath, template, jq-style) and the transformation expression with syntax highlighting hints

---

### User Story 3 - Workflow Validation and Error Highlighting (Priority: P2)

A developer needs real-time validation of workflow structure (circular dependencies, disconnected nodes, missing required fields) with visual error indicators so they can identify and fix issues before execution.

**Why this priority**: Validation prevents runtime failures and improves the developer experience by catching errors early. However, basic workflow creation (P1) should work without this - validation can be added as a quality-of-life enhancement. This is independently testable by intentionally creating invalid workflows.

**Independent Test**: Can be fully tested by creating workflows with various validation errors (circular edge creating a loop, disconnected node with no incoming/outgoing edges, node with missing required field like tool name), triggering validation (automatic or manual), and verifying that error indicators appear on problematic nodes and edges with descriptive error messages.

**Acceptance Scenarios**:

1. **Given** a workflow contains a node with no incoming or outgoing edges (except start/end nodes), **When** validation runs, **Then** the disconnected node is highlighted in yellow with a warning icon, and the status bar shows "Warning: 1 disconnected node"
2. **Given** an MCP Tool node has an empty "tool name" field, **When** validation runs, **Then** the node is highlighted in red with an error icon, and hovering shows the message "Required field missing: tool name"
3. **Given** the user creates an edge that would form a circular dependency (A → B → C → A), **When** the edge is about to be created, **Then** the system prevents the edge creation and displays an error message "Cannot create circular dependency"
4. **Given** a workflow has 3 validation errors, **When** the user presses 'v' to view validation details, **Then** a validation panel opens listing all errors with node IDs and descriptions, allowing navigation to each problematic node
5. **Given** validation errors exist, **When** the user corrects all errors, **Then** the error indicators clear automatically, and the status bar shows "Valid workflow"

---

### User Story 4 - Undo/Redo Support (Priority: P2)

A developer accidentally deletes a node or creates an incorrect connection and wants to undo the last action (or redo an undone action) to recover from mistakes without reloading the workflow.

**Why this priority**: Undo/redo significantly improves the editing experience and reduces frustration from mistakes, but it's not essential for basic workflow creation. Users can manually recreate deleted nodes as a workaround. This is independently valuable for error recovery.

**Independent Test**: Can be fully tested by performing a sequence of operations (add node, move node, delete edge, edit property), pressing 'u' to undo each operation in reverse order, pressing 'Ctrl+R' to redo them forward, and verifying the workflow state matches the expected state at each step.

**Acceptance Scenarios**:

1. **Given** the user adds a new node to the canvas, **When** the user presses 'u' for undo, **Then** the newly added node is removed from the canvas, and the undo stack contains the previous state
2. **Given** the user has performed 5 operations and undone 2 of them, **When** the user presses 'Ctrl+R' for redo, **Then** the most recently undone operation is reapplied, and the workflow state advances forward
3. **Given** the user deletes an edge between two nodes, **When** the user presses 'u' for undo, **Then** the edge is restored with the correct source and target nodes
4. **Given** the user edits a node's property value from "old" to "new", **When** the user presses 'u' for undo, **Then** the property value reverts to "old", and the node property panel reflects the reverted value
5. **Given** the undo stack is empty (no prior operations), **When** the user presses 'u', **Then** nothing happens, and a status message shows "Nothing to undo"

---

### User Story 5 - Canvas Navigation and Zoom (Priority: P3)

A developer working with large workflows (50+ nodes) needs to pan the canvas to view different sections and zoom in/out to see the overall structure or focus on specific nodes.

**Why this priority**: Canvas navigation is important for large workflows but not essential for MVP. Small workflows (< 10 nodes) fit on screen without panning or zooming. This enhances usability for complex workflows but can be added later without blocking basic functionality.

**Independent Test**: Can be fully tested by creating a workflow with 20+ nodes spread across a large canvas area, using arrow keys or mouse to pan the viewport, using '+/-' keys to zoom in/out, and verifying that all nodes remain correctly positioned and visible at different zoom levels.

**Acceptance Scenarios**:

1. **Given** a workflow contains nodes positioned beyond the visible canvas area, **When** the user presses Shift+Arrow keys, **Then** the canvas viewport pans in the arrow direction, revealing previously hidden nodes
2. **Given** the canvas is at default zoom level (100%), **When** the user presses '+' to zoom in, **Then** the canvas zoom increases to 125%, nodes appear larger, and the viewport centers on the selected node
3. **Given** the canvas is zoomed in at 150%, **When** the user presses '-' to zoom out, **Then** the canvas zoom decreases to 125%, nodes appear smaller, and more of the workflow becomes visible
4. **Given** the user has panned and zoomed to focus on a specific section, **When** the user presses '0' (zero), **Then** the canvas resets to default zoom (100%) and centers on the start node
5. **Given** a large workflow is loaded, **When** the user presses 'f' to fit all nodes, **Then** the canvas automatically zooms and pans to show all nodes within the visible viewport

---

### User Story 6 - Node Type Palette and Templates (Priority: P3)

A developer wants to quickly add nodes of different types (MCP Tool, Transform, Condition, Loop, Parallel) from a searchable palette and optionally use pre-configured templates for common patterns.

**Why this priority**: The basic "add node" functionality (P1) can start with a simple list selection. A searchable palette and templates improve efficiency but aren't blocking for MVP. This is independently valuable for workflow authoring speed but can be added after core functionality works.

**Independent Test**: Can be fully tested by pressing 'a' to open the node palette, typing to filter node types (e.g., "trans" filters to "Transform"), selecting a node type, verifying it's added to the canvas, and testing template selection for common patterns (e.g., "ETL Pipeline" template creates 3 pre-connected nodes).

**Acceptance Scenarios**:

1. **Given** the workflow builder is open, **When** the user presses 'a' to add a node, **Then** a node palette appears showing all available node types (MCP Tool, Transform, Condition, Loop, Parallel, End)
2. **Given** the node palette is open with 6 node types, **When** the user types "cond", **Then** the palette filters to show only "Condition" node type
3. **Given** the user selects "Loop" from the node palette, **When** the node is created, **Then** it appears on the canvas with default loop configuration (loop variable, collection source) pre-populated in the property panel
4. **Given** the user presses 't' to open the template selection, **When** the user selects the "ETL Pipeline" template, **Then** three nodes (Extract, Transform, Load) are created and connected with edges, forming a basic ETL workflow structure
5. **Given** the node palette is open, **When** the user presses '?' for help, **Then** a tooltip appears explaining the purpose and configuration requirements for the currently highlighted node type

---

### User Story 7 - Keyboard Shortcuts and Help Overlay (Priority: P3)

A developer working in the TUI wants to discover available keyboard shortcuts and access context-sensitive help without leaving the editor or consulting external documentation.

**Why this priority**: Help is important for learnability but not blocking for users who can learn through trial and error or external docs. Keyboard shortcuts improve efficiency but basic mouse-like selection and simple keys (Enter, 'd', 's') work for MVP. This enhances discoverability.

**Independent Test**: Can be fully tested by pressing '?' to open the help overlay, navigating through different help sections (general shortcuts, node operations, canvas navigation), pressing '?' again to close help, and verifying that all documented shortcuts work as described.

**Acceptance Scenarios**:

1. **Given** the workflow builder is open, **When** the user presses '?', **Then** a help overlay appears showing all available keyboard shortcuts organized by category (Node Operations, Canvas Navigation, Editing, File Operations)
2. **Given** the help overlay is open, **When** the user navigates through the help content, **Then** each shortcut is listed with its key combination and a brief description (e.g., "a - Add new node", "d - Delete selected node")
3. **Given** a specific panel is focused (e.g., property panel), **When** the user presses '?', **Then** the help overlay shows context-sensitive shortcuts relevant to the current panel
4. **Given** the help overlay is open, **When** the user presses '?' again or 'Esc', **Then** the help overlay closes and focus returns to the workflow builder
5. **Given** the user is editing a JSONPath field, **When** the help overlay opens, **Then** it includes a JSONPath syntax reference section with examples

---

### User Story 8 - Real-time Workflow Rendering (Priority: P1)

A developer editing a workflow needs to see the visual representation update immediately after any operation (add/delete/move node, create/delete edge) so they understand the current workflow structure without manually refreshing.

**Why this priority**: Real-time rendering is essential for the visual editor to be useful. Without it, the editor becomes a blind YAML editor with extra steps. This is part of the P1 MVP as it's the core visual feedback mechanism. However, it's listed separately to emphasize the rendering implementation requirements.

**Independent Test**: Can be fully tested by performing various operations (add node, delete node, move node, create edge, delete edge, edit node properties) and verifying that the canvas immediately updates to reflect changes, with proper node positioning, edge routing, and visual indicators (selection highlights, validation colors).

**Acceptance Scenarios**:

1. **Given** a workflow is displayed on the canvas, **When** the user adds a new node, **Then** the canvas immediately re-renders showing the new node without requiring a manual refresh
2. **Given** two nodes are connected by an edge, **When** the user moves the source node to a new position, **Then** the edge automatically re-routes to maintain the connection, updating in real-time as the node moves
3. **Given** a node is selected, **When** the selection changes to a different node, **Then** the previously selected node returns to normal appearance and the newly selected node is highlighted with a distinct border color
4. **Given** a node has validation errors, **When** the user corrects the error in the property panel, **Then** the node's error indicator (red highlight) immediately clears on the canvas
5. **Given** the workflow has 10 nodes, **When** the user deletes a node, **Then** the canvas re-renders with the remaining 9 nodes, all edges connected to the deleted node are removed, and the layout remains clean

---

### Edge Cases

- What happens when the user attempts to create an edge from a node to itself (self-loop)?
  - System should prevent self-loops and display error: "Cannot create edge from node to itself"
- What happens when the canvas contains 100+ nodes and rendering becomes slow?
  - Implement virtualization: only render nodes visible in current viewport
  - Add performance mode that simplifies node rendering (no gradients, simpler shapes)
- What happens when the user resizes the terminal window while the editor is open?
  - Canvas should detect terminal resize events and adjust viewport dimensions accordingly
  - Re-center viewport on currently selected node if it's now out of bounds
- What happens when the user tries to save a workflow with an invalid name (special characters, too long)?
  - Validate workflow name using existing `workflow.IsValidWorkflowName()` function
  - Show error message and prevent save, keep the editor open for corrections
- What happens when two nodes are positioned at the exact same coordinates?
  - Automatically offset the second node by a small amount (e.g., +10 pixels in X and Y)
  - Highlight both nodes with a warning indicator suggesting manual repositioning
- What happens when the user undoes/redoes beyond the stack limits?
  - Undo on empty undo stack: show status message "Nothing to undo"
  - Redo on empty redo stack: show status message "Nothing to redo"
  - Do not modify workflow state
- What happens when the workflow contains node types not yet supported by the editor?
  - Display unknown nodes with a generic "Unknown" icon and type label
  - Show warning in status bar: "Workflow contains unsupported node types"
  - Allow viewing but disable editing of unsupported node properties
- What happens when the user presses conflicting keyboard shortcuts (e.g., Ctrl+S during property panel edit)?
  - Implement key binding priority: property panel edit mode overrides global shortcuts
  - Require explicit mode exit (Esc) before global shortcuts work
- What happens when the user loads a workflow with syntax errors or corrupted YAML?
  - Show error modal with YAML parsing error details
  - Offer option to "Edit YAML manually" or "Cancel and return to explorer"
  - Do not load corrupt workflow into editor to prevent data loss

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST render a canvas displaying all workflow nodes and edges with correct positioning and visual hierarchy (selected nodes highlighted, error nodes marked in red)
- **FR-002**: System MUST support adding new nodes to the canvas through a node palette accessible via keyboard shortcut ('a' key)
- **FR-003**: System MUST allow users to create edges between nodes by selecting a source node, pressing 'c', and selecting a target node
- **FR-004**: System MUST prevent creation of circular dependencies by detecting cycles before edge creation and displaying an error message
- **FR-005**: System MUST provide a property panel for editing node-specific configuration (name, tool name, parameters, expressions, conditions)
- **FR-006**: System MUST validate property field contents in real-time (JSONPath syntax, expression syntax, required fields) and display validation feedback
- **FR-007**: System MUST support keyboard-based node positioning using vim-style navigation keys (h/j/k/l) to move selected nodes
- **FR-008**: System MUST implement undo/redo functionality for all destructive operations (add, delete, move, edit) with keyboard shortcuts ('u' for undo, 'Ctrl+R' for redo)
- **FR-009**: System MUST save workflow changes to YAML format when user presses 's', preserving all node configurations, edges, and metadata
- **FR-010**: System MUST load existing workflows from YAML files into the canvas with correct node positions and edge connections
- **FR-011**: System MUST mark workflow as modified when any change occurs (add/delete/move node, create/delete edge, edit property) and display modification indicator in status bar
- **FR-012**: System MUST validate complete workflow structure (connected nodes, no circular dependencies, all required fields populated) and display validation status
- **FR-013**: System MUST support deleting nodes and edges with keyboard shortcuts ('d' for delete) and automatically remove orphaned edges when a node is deleted
- **FR-014**: System MUST provide a help overlay accessible via '?' key showing all keyboard shortcuts and context-sensitive help
- **FR-015**: System MUST update canvas rendering immediately (< 100ms) after any operation to reflect current workflow state
- **FR-016**: System MUST support canvas panning using Shift+Arrow keys to navigate workflows larger than the visible viewport
- **FR-017**: System MUST support canvas zoom in/out using '+'/'-' keys with zoom levels from 50% to 200%
- **FR-018**: System MUST provide node type filtering in the node palette through text input search
- **FR-019**: System MUST support workflow templates that create pre-configured node groups (ETL Pipeline, API Integration, Batch Processing)
- **FR-020**: System MUST handle terminal resize events and adjust canvas viewport dimensions accordingly
- **FR-021**: System MUST support all workflow node types: MCP Tool, Transform, Condition, Loop, Parallel, Start, End
- **FR-022**: System MUST provide syntax hints for expression fields (JSONPath, template strings, conditional expressions) in property panel
- **FR-023**: System MUST prevent saving workflows with validation errors and display a summary of blocking issues
- **FR-024**: System MUST auto-layout nodes when loading a workflow without explicit position metadata (using topological sort and hierarchical layout algorithm)
- **FR-025**: System MUST support multi-select of nodes using 'Ctrl+Click' pattern (keyboard-based multi-select) for batch operations
- **FR-026**: System MUST provide edge routing that avoids node overlaps using orthogonal or curved edge styles
- **FR-027**: System MUST display node execution status indicators (pending, running, completed, failed) when integrated with execution monitor
- **FR-028**: System MUST support copying and pasting nodes within the same workflow or between workflows using 'y' (yank) and 'p' (paste) vim-style shortcuts
- **FR-029**: System MUST validate edge creation rules (e.g., Condition nodes must have exactly 2 outgoing edges for true/false branches)
- **FR-030**: System MUST display minimap navigation for large workflows showing viewport position relative to entire canvas

### Key Entities

- **Canvas**: Represents the drawing surface for the workflow graph
  - Attributes: Width, Height, ViewportX, ViewportY, ZoomLevel, nodes (map of node ID to canvasNode), edges (list of canvasEdges)
  - Relationships: Contains canvasNodes and canvasEdges

- **canvasNode**: Represents a workflow node positioned on the canvas
  - Attributes: node (workflow.Node), position (X, Y coordinates), width, height, selected (boolean), highlighted (boolean), validationStatus (valid/warning/error)
  - Relationships: References workflow.Node from domain model

- **canvasEdge**: Represents an edge drawn on the canvas between two nodes
  - Attributes: edge (workflow.Edge), fromPos (source position), toPos (target position), routingPoints (list of intermediate coordinates for curved edges), selected (boolean)
  - Relationships: References workflow.Edge from domain model

- **NodePalette**: Represents the node selection interface
  - Attributes: nodeTypes (list of available node type names), selectedIndex (current selection), visible (boolean), filterText (search query)
  - Relationships: Contains node type metadata for creation

- **PropertyPanel**: Represents the node property editor
  - Attributes: node (workflow.Node being edited), fields (list of propertyField), editIndex (currently focused field), visible (boolean), validationMessage (current error/warning)
  - Relationships: Edits properties of selected workflow.Node

- **propertyField**: Represents an editable node property
  - Attributes: label (field name), value (current value), required (boolean), valid (boolean), fieldType (text/expression/condition/jsonpath/template), validationFn (validation function), helpText (syntax hints)
  - Relationships: Part of PropertyPanel

- **workflowSnapshot**: Stores workflow state for undo/redo
  - Attributes: nodes (copy of workflow nodes), edges (copy of workflow edges), timestamp
  - Relationships: Stored in undoStack and redoStack of WorkflowBuilder

- **ValidationStatus**: Represents workflow validation state
  - Attributes: valid (boolean), errors (list of validation error messages with node IDs), warnings (list of warning messages), lastValidated (timestamp)
  - Relationships: Associated with WorkflowBuilder

- **HelpPanel**: Represents the help overlay
  - Attributes: visible (boolean), currentSection (category of help being displayed), keyBindings (list of HelpKeyBinding entries)
  - Relationships: Contains help content organized by context

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can create a functional 5-node workflow with edge connections in under 3 minutes using only keyboard shortcuts
- **SC-002**: Visual editor supports workflows with up to 100 nodes without rendering performance degradation (< 100ms per render frame at 60 FPS)
- **SC-003**: 95% of users successfully add a node, connect edges, edit properties, and save workflow on their first attempt without external help
- **SC-004**: Undo/redo operations complete in under 50ms and accurately restore workflow state for 100% of tested operation types
- **SC-005**: Real-time validation catches 100% of structural errors (circular dependencies, disconnected nodes, missing required fields) before workflow execution
- **SC-006**: Canvas navigation (pan, zoom) responds within 16ms to provide smooth 60 FPS interaction for workflows up to 200 nodes
- **SC-007**: Help overlay provides keyboard shortcuts and syntax reference, reducing external documentation lookups by 80%
- **SC-008**: Property panel field validation provides immediate feedback (< 200ms) for JSONPath, expression syntax, and type errors
- **SC-009**: Workflow modifications (add/delete/move operations) persist correctly to YAML format with 100% data integrity
- **SC-010**: Users can discover and use 90% of core functionality (add, connect, delete, save, undo) through in-TUI help without external tutorials
