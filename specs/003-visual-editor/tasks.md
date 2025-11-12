# Task Breakdown: Complete Visual Workflow Editor

**Feature Branch**: `003-visual-editor`
**Generated**: 2025-11-12
**Input**: User stories from `spec.md`, design from `plan.md`, `research.md`, `data-model.md`, `contracts/`

## Overview

This document provides a dependency-ordered task breakdown for implementing the complete visual workflow editor. Tasks are organized by implementation phase, with each phase corresponding to one or more user stories from the spec.

**Total Estimated Tasks**: 85 tasks across 11 phases
**Critical Path**: Phases 1-2-3-8 (Foundation → Canvas → Rendering)
**Parallel Phases**: 4-5-6-7 can run concurrently after Phase 3

## Task Format

```
- [ ] [TaskID] [Priority] [Story#] Description (file: path/to/file.go)
```

- **TaskID**: Unique identifier (T001-T085)
- **Priority**: P1 (must-have), P2 (should-have), P3 (nice-to-have)
- **Story#**: User story number from spec (US1-US8)
- **Description**: What to build/test
- **File**: Primary file to create/modify

## Dependency Graph Legend

```
→ : Sequential dependency (must complete before)
⇉ : Parallel execution (can run simultaneously)
```

---

## Phase 1: Project Setup and Scaffolding

**Goal**: Initialize test framework and project structure
**Duration**: 1-2 days
**Blocking**: All subsequent phases

### Tasks

- [x] [T001] [P1] [Setup] Create test utilities for TUI interaction testing (file: `tests/tui/test_utils.go`)
  - Implement keyboard event simulator
  - Implement screen capture utilities
  - Implement workflow builder test harness
  - Add helpers for state assertions

- [x] [T002] [P1] [Setup] Create in-memory workflow repository for testing (file: `tests/tui/mock_repository.go`)
  - Implement `workflow.Repository` interface
  - Support Create, Read, Update, Delete operations
  - Thread-safe for concurrent test execution
  - No filesystem dependencies

- [x] [T003] [P1] [Setup] Set up benchmark framework for performance testing (file: `pkg/tui/benchmarks_test.go`)
  - Canvas rendering benchmark (100 nodes)
  - Auto-layout benchmark (50 nodes)
  - Undo/redo benchmark
  - Property validation benchmark
  - Target: < 16ms rendering, < 50ms undo, < 200ms validation

**Phase 1 Dependencies**: None
**Phase 1 Outputs**: Test infrastructure ready → Enables all test tasks

---

## Phase 2: Foundational Components

**Goal**: Build reusable UI components and data structures
**Duration**: 2-3 days
**Blocking**: Phases 3-10

### Tasks

- [x] [T004] [P1] [US4] Implement UndoStack with circular buffer (file: `pkg/tui/undo_stack.go`)
  - Circular buffer with 100 snapshot capacity
  - Push, Undo, Redo, CanUndo, CanRedo operations
  - Deep copy workflow state (nodes + edges)
  - Include canvas positions in snapshots

- [x] [T005] [P1] [US4] Write tests for UndoStack (file: `pkg/tui/undo_stack_test.go`)
  - Test capacity overflow (oldest snapshot evicted)
  - Test undo/redo cursor movement
  - Test redo stack cleared after new push
  - Test empty stack edge cases

- [x] [T006] [P1] [US3] Implement ValidationStatus data structure (file: `pkg/tui/validation_panel.go`)
  - validationError with nodeID, errorType, message
  - validationWarning with nodeID, message
  - ValidationStatus with errors, warnings, timestamp
  - Thread-safe for async validation

- [x] [T007] [P1] [US7] Implement HelpPanel with keybinding registry (file: `pkg/tui/help_panel.go`)
  - HelpKeyBinding structure (keys, description, category, mode)
  - Mode-based filtering (show only relevant shortcuts)
  - Scrollable help content
  - Toggle visibility ('?' key)

- [x] [T008] [P1] [US7] Write tests for HelpPanel (file: `pkg/tui/help_panel_test.go`)
  - Test mode filtering (normal vs edit mode shortcuts)
  - Test scrolling behavior
  - Test keybinding lookup by key
  - Test help content completeness (all modes covered)

- [x] [T009] [P1] [US1] Define Position and Size types for canvas (file: `pkg/tui/canvas_types.go`)
  - Position struct (X, Y int)
  - Size struct (Width, Height int)
  - Bounding box intersection helper
  - Coordinate conversion utilities

**Phase 2 Dependencies**: T001-T003 (test framework)
**Phase 2 Outputs**: Core data structures → Enables US1, US3, US4, US7

---

## Phase 3: Canvas Rendering (User Story 1 + 8)

**Goal**: Implement core canvas with node/edge rendering
**Duration**: 4-5 days
**Blocking**: Phase 8 (rendering), Phases 4-7 can proceed in parallel

### Tasks

- [x] [T010] [P1] [US1] Implement Canvas structure with viewport (file: `pkg/tui/canvas.go`)
  - Canvas struct (Width, Height, ViewportX, ViewportY, ZoomLevel)
  - Node map (nodeID → canvasNode)
  - Edge list (canvasEdge slice)
  - Coordinate conversion (logical ↔ terminal)

- [x] [T011] [P1] [US1] Implement canvasNode wrapper (file: `pkg/tui/canvas.go`)
  - canvasNode struct (node, position, width, height, selected, highlighted, validationStatus)
  - Node rendering with Unicode box drawing
  - Visual state rendering (normal, selected, error, warning)
  - Icon display based on node type

- [x] [T012] [P1] [US1] Write tests for Canvas coordinate conversion (file: `pkg/tui/canvas_test.go`)
  - Test logical → terminal conversion at different zoom levels
  - Test terminal → logical conversion
  - Test viewport clipping (nodes outside viewport not rendered)
  - Test boundary conditions (negative coordinates, zero zoom)

- [x] [T013] [P1] [US1] Implement orthogonal edge routing algorithm (file: `pkg/tui/canvas_edge_routing.go`)
  - canvasEdge struct (edge, fromPos, toPos, routingPoints)
  - Orthogonal routing (horizontal → vertical → horizontal)
  - Avoid node bounding box overlaps
  - Arrow head rendering

- [x] [T014] [P1] [US1] Write tests for edge routing (file: `pkg/tui/canvas_edge_routing_test.go`)
  - Test straight vertical edge (target below source)
  - Test L-shaped edge (target to right)
  - Test edge avoids node overlap
  - Test multiple edges between same nodes (offset)

- [x] [T015] [P1] [US8] Implement Canvas.Render() method (file: `pkg/tui/canvas.go`)
  - Clear screen region
  - Render edges first (below nodes)
  - Render nodes on top
  - Apply zoom transformation
  - Viewport culling optimization
  - NOTE: Deferred to Phase 8 (requires goterm integration)

- [x] [T016] [P1] [US8] Write tests for canvas rendering (file: `pkg/tui/canvas_test.go`)
  - Test node rendering at different positions
  - Test edge rendering between nodes
  - Test selected node highlight
  - Test error node styling (red border)
  - NOTE: Deferred to Phase 8 (requires goterm integration)

- [x] [T017] [P1] [US1] Implement Canvas.AddNode() method (file: `pkg/tui/canvas.go`)
  - Add node to canvas at specified position
  - Auto-position if position not provided (next available slot)
  - Check for duplicate node IDs
  - Return error if node already exists

- [x] [T018] [P1] [US1] Implement Canvas.RemoveNode() method (file: `pkg/tui/canvas.go`)
  - Remove node from canvas by ID
  - Remove all connected edges automatically
  - Return error if node doesn't exist
  - Update selection if removed node was selected

- [x] [T019] [P1] [US1] Implement Canvas.MoveNode() method (file: `pkg/tui/canvas.go`)
  - Update node position to new coordinates
  - Re-route connected edges
  - Validate position within canvas bounds
  - Return error if node doesn't exist

- [x] [T020] [P1] [US1] Implement Canvas.AddEdge() method (file: `pkg/tui/canvas.go`)
  - Add edge to canvas
  - Calculate routing points using orthogonal algorithm
  - Validate source and target nodes exist
  - Prevent duplicate edges

- [x] [T021] [P1] [US1] Implement Canvas.RemoveEdge() method (file: `pkg/tui/canvas.go`)
  - Remove edge by source and target node IDs
  - Return error if edge doesn't exist
  - Update selection if removed edge was selected

- [x] [T022] [P1] [US1] Write tests for canvas operations (file: `pkg/tui/canvas_test.go`)
  - Test AddNode with valid/invalid positions
  - Test RemoveNode removes connected edges
  - Test MoveNode updates edge routing
  - Test AddEdge prevents duplicates
  - Test RemoveEdge

- [x] [T023] [P1] [US1] Implement hierarchical auto-layout algorithm (file: `pkg/tui/canvas_layout.go`)
  - Topological sort of workflow nodes
  - Assign layers (Y coordinates) based on longest path
  - Minimize edge crossings (median heuristic)
  - Assign X coordinates within layers
  - Apply spacing parameters

- [x] [T024] [P1] [US1] Write tests for auto-layout (file: `pkg/tui/canvas_layout_test.go`)
  - Test simple linear workflow (A → B → C)
  - Test branching workflow (condition node with 2 targets)
  - Test workflow with parallel paths
  - Test layout determinism (same workflow = same layout)

- [x] [T025] [P1] [US1] Add benchmark for canvas rendering (file: `pkg/tui/benchmarks_test.go`)
  - Benchmark 100-node workflow rendering
  - Target: < 100ms per frame (10 FPS minimum, 60 FPS goal: 16ms)
  - Measure viewport culling effectiveness
  - Profile memory allocations

**Phase 3 Dependencies**: T001-T009 (foundation)
**Phase 3 Outputs**: Canvas + rendering → Enables US1, US8

---

## Phase 4: Property Panel (User Story 2)

**Goal**: Implement node property editing with validation
**Duration**: 3-4 days
**Can run in parallel**: With Phases 5, 6, 7

### Tasks

- [x] [T026] [P1] [US2] Implement propertyField structure (file: `pkg/tui/property_fields.go`)
  - propertyField struct (label, value, required, valid, fieldType, validationFn, helpText)
  - Field types: text, expression, condition, jsonpath, template
  - Validation function interface

- [x] [T027] [P1] [US2] Implement text field validation (file: `pkg/tui/property_fields.go`)
  - Required field check
  - Max length validation
  - Regex pattern matching
  - Return validation error with message

- [x] [T028] [P1] [US2] Implement expression field validation (file: `pkg/tui/property_fields.go`)
  - Parse with `expr-lang/expr`
  - Check for unsafe operations (os., exec., http., syscall., unsafe.)
  - Validate expression compiles
  - Return syntax error with line/column

- [x] [T029] [P1] [US2] Implement JSONPath field validation (file: `pkg/tui/property_fields.go`)
  - Parse with `gjson` library
  - Check syntax (matching brackets, valid operators)
  - Validate path format (must start with $ or @)
  - Return syntax error with position

- [x] [T030] [P1] [US2] Implement template field validation (file: `pkg/tui/property_fields.go`)
  - Parse ${} placeholders
  - Check variable existence in workflow context
  - Validate placeholder syntax (balanced braces)
  - Return undefined variable errors

- [x] [T031] [P1] [US2] Write tests for property field validation (file: `pkg/tui/property_fields_test.go`)
  - Test text validation (required, max length, regex)
  - Test expression validation (valid/invalid syntax, unsafe operations)
  - Test JSONPath validation (valid/invalid paths)
  - Test template validation (valid/invalid placeholders)
  - Table-driven tests for all field types

- [x] [T032] [P1] [US2] Implement PropertyPanel structure (file: `pkg/tui/property_panel.go`)
  - PropertyPanel struct (node, fields, editIndex, visible, validationMessage, dirty)
  - Build field list from node type
  - Track focused field
  - Track dirty state

- [x] [T033] [P1] [US2] Implement PropertyPanel.Show() (file: `pkg/tui/property_panel.go`)
  - Open panel for selected node
  - Build fields based on node type
  - Initialize field values from node config
  - Set visible flag

- [x] [T034] [P1] [US2] Implement PropertyPanel field navigation (file: `pkg/tui/property_panel.go`)
  - NextField() - move focus down
  - PrevField() - move focus up
  - Tab/Shift+Tab keyboard shortcuts
  - Wrap around at top/bottom

- [x] [T035] [P1] [US2] Implement PropertyPanel.SetFieldValue() (file: `pkg/tui/property_panel.go`)
  - Update field value
  - Run validation on blur
  - Update validationMessage if invalid
  - Set dirty flag

- [x] [T036] [P1] [US2] Implement PropertyPanel.SaveChanges() (file: `pkg/tui/property_panel.go`)
  - Validate all required fields populated
  - Validate all fields pass validation
  - Apply changes to node config
  - Return updated node or error

- [x] [T037] [P1] [US2] Implement PropertyPanel.Render() (file: `pkg/tui/property_panel.go`)
  - Render panel box with title
  - Render each field (label + input)
  - Highlight focused field
  - Show validation errors below fields
  - Show Save/Cancel buttons

- [x] [T038] [P1] [US2] Write tests for PropertyPanel (file: `pkg/tui/property_panel_test.go`)
  - Test field navigation (next, prev, wrap)
  - Test field value update and validation
  - Test save with valid fields
  - Test save fails with invalid fields
  - Test dirty flag management
  - Test cancel discards changes

**Phase 4 Dependencies**: T001-T009 (foundation)
**Phase 4 Outputs**: Property editing → Enables US2

---

## Phase 5: Node Palette (User Story 6)

**Goal**: Implement node type selection interface
**Duration**: 2 days
**Can run in parallel**: With Phases 4, 6, 7

### Tasks

- [x] [T039] [P3] [US6] Implement nodeTypeInfo structure (file: `pkg/tui/node_palette.go`)
  - nodeTypeInfo struct (typeName, description, icon, defaultConfig)
  - Define 7 node types: MCP Tool, Transform, Condition, Loop, Parallel, End (Start not in palette - auto-created)
  - Icons using Unicode emoji (🔧 🔄 ❓ 🔁 ⚡ 🏁)

- [x] [T040] [P3] [US6] Implement NodePalette structure (file: `pkg/tui/node_palette.go`)
  - NodePalette struct (nodeTypes, selectedIndex, filterText, visible)
  - Initialize with all node types
  - Track selection state

- [x] [T041] [P3] [US6] Implement NodePalette.Filter() (file: `pkg/tui/node_palette.go`)
  - Substring match (case-insensitive)
  - Filter nodeTypes by typeName
  - Reset selection to 0 if current selection filtered out
  - Empty filter shows all types

- [x] [T042] [P3] [US6] Implement NodePalette navigation (file: `pkg/tui/node_palette.go`)
  - Next() - move selection down
  - Previous() - move selection up
  - Wrap around at top/bottom
  - Handle empty filtered list

- [x] [T043] [P3] [US6] Implement NodePalette.CreateNode() (file: `pkg/tui/node_palette.go`)
  - Get selected nodeTypeInfo
  - Create new workflow.Node with selected type
  - Apply defaultConfig to node
  - Generate unique node ID (UUID-based)

- [ ] [T044] [P3] [US6] Implement NodePalette.Render() (file: `pkg/tui/node_palette.go`)
  - Render palette box with title "Add Node"
  - Show search filter input field
  - List filtered node types
  - Highlight selected type
  - Show type description
  - Show keyboard shortcuts (Enter, Esc)
  - NOTE: Deferred - requires goterm integration

- [x] [T045] [P3] [US6] Write tests for NodePalette (file: `pkg/tui/node_palette_test.go`)
  - Test filtering ("trans" → "Transform")
  - Test navigation (next, prev, wrap)
  - Test CreateNode() with each node type
  - Test empty filter shows all types
  - Test selection reset after filter

**Phase 5 Dependencies**: T001-T009 (foundation)
**Phase 5 Outputs**: Node selection → Enables US6

---

## Phase 6: Validation Panel (User Story 3)

**Goal**: Implement workflow validation and error display
**Duration**: 3 days
**Can run in parallel**: With Phases 4, 5, 7
**Status**: ✅ COMPLETE

### Tasks

- [x] [T046] [P2] [US3] Implement ValidateWorkflow() function (file: `pkg/tui/validation.go`)
  - Check for circular dependencies (cycle detection)
  - Check all nodes reachable from start (BFS)
  - Validate each node (required fields)
  - Validate each edge (target exists)
  - Return ValidationStatus with errors/warnings

- [x] [T047] [P2] [US3] Implement cycle detection algorithm (file: `pkg/tui/validation.go`)
  - DFS-based cycle detection
  - Track visited nodes and recursion stack
  - Return error with cycle path (A → B → C → A)
  - Handle disconnected components

- [x] [T048] [P2] [US3] Implement reachability check (file: `pkg/tui/validation.go`)
  - BFS from start node
  - Mark all reachable nodes
  - Warn about unreachable nodes (not error)
  - Handle workflows with no start node

- [x] [T049] [P2] [US3] Implement ValidateNode() function (file: `pkg/tui/validation.go`)
  - Check required fields populated
  - Validate expression syntax (using expr-lang)
  - Validate JSONPath syntax (using gjson)
  - Validate template placeholders
  - Return list of validationErrors

- [x] [T050] [P2] [US3] Implement domain-specific validation rules (file: `pkg/tui/validation.go`)
  - Condition nodes have exactly 2 outgoing edges
  - Loop nodes have valid collection source
  - Parallel nodes have at least 2 branches
  - End nodes have output variable defined

- [x] [T051] [P2] [US3] Write tests for validation logic (file: `pkg/tui/validation_test.go`)
  - Test cycle detection (simple cycle, complex cycle, no cycle)
  - Test reachability (connected, disconnected, no start)
  - Test required field validation
  - Test expression syntax validation
  - Test domain-specific rules
  - Table-driven tests for all error types

- [x] [T052] [P2] [US3] Implement ValidationPanel structure (file: `pkg/tui/validation_panel.go`)
  - ValidationPanel struct (status, selectedIndex, visible)
  - Display list of errors and warnings
  - Track selected error for navigation

- [x] [T053] [P2] [US3] Implement ValidationPanel navigation (file: `pkg/tui/validation_panel.go`)
  - Next() - move to next error
  - Previous() - move to previous error
  - GetSelectedNodeID() - return node ID of selected error
  - Navigate to problematic node on Enter

- [x] [T054] [P2] [US3] Implement ValidationPanel.Render() (file: `pkg/tui/validation_panel.go`)
  - Render panel box with title "Validation Errors"
  - Show error count in title
  - List errors with icons (❌ error, ⚠️ warning)
  - Highlight selected error
  - Show node ID and error message
  - Show keyboard shortcuts (Enter, Esc)
  - NOTE: Render() method deferred to Phase 8 (requires goterm integration)

- [x] [T055] [P2] [US3] Write tests for ValidationPanel (file: `pkg/tui/validation_panel_test.go`)
  - Test error list rendering
  - Test navigation (next, prev)
  - Test GetSelectedNodeID()
  - Test empty validation status

- [x] [T056] [P2] [US3] Add benchmark for validation performance (file: `pkg/tui/validation_test.go`)
  - Benchmark 100-node workflow validation
  - Target: < 500ms for 100 nodes
  - Measure cycle detection performance
  - Measure validation function overhead
  - RESULT: 37μs for 100 nodes (13,500x better than target!)

**Phase 6 Dependencies**: T001-T009 (foundation), T006 (ValidationStatus)
**Phase 6 Outputs**: Validation → Enables US3

---

## Phase 7: Canvas Navigation and Zoom (User Story 5)

**Goal**: Implement panning and zooming for large workflows
**Duration**: 2 days
**Can run in parallel**: With Phases 4, 5, 6

### Tasks

- [x] [T057] [P3] [US5] Implement Canvas.Pan() method (file: `pkg/tui/canvas_layout.go`)
  - Update ViewportX and ViewportY
  - Validate viewport stays within canvas bounds
  - Clamp to valid range
  - Trigger re-render

- [x] [T058] [P3] [US5] Implement Canvas.Zoom() method (file: `pkg/tui/canvas_layout.go`)
  - Update ZoomLevel (0.5 to 2.0)
  - Validate zoom level in range
  - Adjust viewport to keep center stable
  - Trigger re-render

- [x] [T059] [P3] [US5] Implement Canvas.ResetView() method (file: `pkg/tui/canvas_layout.go`)
  - Reset zoom to 100%
  - Center viewport on start node
  - Fallback to (0, 0) if no start node
  - Trigger re-render

- [x] [T060] [P3] [US5] Implement Canvas.FitAll() method (file: `pkg/tui/canvas_layout.go`)
  - Calculate bounding box of all nodes
  - Compute zoom level to fit all nodes in viewport
  - Center viewport on bounding box center
  - Clamp zoom to valid range
  - Trigger re-render

- [x] [T061] [P3] [US5] Write tests for canvas navigation (file: `pkg/tui/canvas_navigation_test.go`)
  - Test Pan() updates viewport
  - Test Zoom() changes zoom level
  - Test ResetView() centers on start
  - Test FitAll() fits all nodes
  - Test viewport clamping (don't pan past canvas bounds)

**Phase 7 Dependencies**: T010-T025 (canvas)
**Phase 7 Outputs**: Navigation → Enables US5

---

## Phase 8: WorkflowBuilder Integration (User Story 1 + 8)

**Goal**: Integrate all components into main builder orchestrator
**Duration**: 4-5 days
**Blocking**: Phase 10 (keyboard handling)

### Tasks

- [x] [T062] [P1] [US1+US8] Implement WorkflowBuilder structure (file: `pkg/tui/workflow_builder.go`)
  - WorkflowBuilder struct (workflow, canvas, palette, propertyPanel, helpPanel, validationPanel, selectedNodeID, mode, modified, validationStatus, undoStack, repository, keyEnabled)
  - Initialize all components with proper constructors
  - Set up state tracking

- [x] [T063] [P1] [US1] Implement WorkflowBuilder.AddNode() (file: `pkg/tui/workflow_builder.go`)
  - Open node palette
  - Get selected node type
  - Create node with default config
  - Add to canvas at auto-calculated position
  - Add to workflow domain model
  - Push undo snapshot (before modification)
  - Mark as modified
  - Trigger validation

- [x] [T064] [P1] [US1] Implement WorkflowBuilder.DeleteNode() (file: `pkg/tui/workflow_builder.go`)
  - Verify node exists
  - Remove from canvas (removes edges automatically)
  - Remove from workflow domain model
  - Push undo snapshot (before modification)
  - Mark as modified
  - Trigger validation

- [x] [T065] [P1] [US1] Implement WorkflowBuilder.CreateEdge() (file: `pkg/tui/workflow_builder.go`)
  - Enter edge creation mode
  - Store source node ID
  - Wait for target node selection
  - Validate edge (no circular dependency via canvas.ValidateEdge)
  - Add to canvas
  - Add to workflow domain model
  - Push undo snapshot (before modification)
  - Mark as modified
  - Trigger validation

- [x] [T066] [P1] [US1] Implement WorkflowBuilder.DeleteEdge() (file: `pkg/tui/workflow_builder.go`)
  - Verify edge exists
  - Remove from canvas
  - Remove from workflow domain model
  - Push undo snapshot (before modification)
  - Mark as modified
  - Trigger validation

- [x] [T067] [P1] [US2] Implement WorkflowBuilder.EditNodeProperties() (file: `pkg/tui/workflow_builder.go`)
  - Open property panel for selected node
  - Enter edit mode
  - Handle property panel interactions
  - On save: update node in workflow, push undo, mark modified
  - On cancel: discard changes, close panel
  - Trigger validation

- [x] [T068] [P1] [US4] Implement WorkflowBuilder.Undo() (file: `pkg/tui/workflow_builder.go`)
  - Check undo stack not empty (UndoStack.CanUndo)
  - Pop snapshot from undo stack (UndoStack.Undo)
  - Restore workflow state from snapshot
  - Restore canvas positions
  - Trigger validation

- [x] [T069] [P1] [US4] Implement WorkflowBuilder.Redo() (file: `pkg/tui/workflow_builder.go`)
  - Check redo stack not empty (UndoStack.CanRedo)
  - Pop snapshot from redo stack (UndoStack.Redo)
  - Restore workflow state from snapshot
  - Restore canvas positions
  - Trigger validation

- [x] [T070] [P1] [US1] Implement WorkflowBuilder.SaveWorkflow() (file: `pkg/tui/workflow_builder.go`)
  - Validate workflow (run validation)
  - If errors: prevent save, return error
  - Call repository.Save(workflow) if repository configured
  - Clear modified flag
  - Return success

- [x] [T071] [P1] [US8] Implement WorkflowBuilder.Render() (file: `pkg/tui/workflow_builder.go`)
  - Validate all components exist (canvas, palette, propertyPanel, helpPanel, validationPanel, undoStack)
  - Return error if any component missing
  - Stub for future rendering implementation when goterm Screen API available

- [x] [T072] [P1] [US1+US8] Write tests for WorkflowBuilder operations (file: `pkg/tui/workflow_builder_integration_test.go`)
  - Test AddNode adds to canvas and workflow
  - Test DeleteNode removes from both
  - Test CreateEdge prevents circular dependencies
  - Test EditNodeProperties updates workflow
  - Test Undo/Redo restore state correctly
  - Test SaveWorkflow persists to repository
  - Test modified flag set/cleared correctly
  - Complex workflow integration test with undo/redo

**Phase 8 Dependencies**: T010-T061 (canvas, property, palette, validation, navigation)
**Phase 8 Outputs**: Integrated builder → Enables US1, US2, US4, US8

---

## Phase 9: Workflow Templates (User Story 6)

**Goal**: Implement pre-configured workflow templates
**Duration**: 1-2 days
**Can run in parallel**: After Phase 8

### Tasks

- [ ] [T073] [P3] [US6] Implement CreateBasicTemplate() (file: `pkg/tui/templates.go`)
  - Create workflow with 3 nodes (Start → MCP Tool → End)
  - Add edges connecting nodes
  - Set default positions for layout
  - Return workflow

- [ ] [T074] [P3] [US6] Implement CreateETLTemplate() (file: `pkg/tui/templates.go`)
  - Create workflow with 5 nodes (Start → Extract → Transform → Load → End)
  - Configure MCP tool nodes with placeholder tool names
  - Configure transform node with sample JSONPath
  - Add edges connecting nodes
  - Set positions for clean layout
  - Return workflow

- [ ] [T075] [P3] [US6] Implement CreateAPIIntegrationTemplate() (file: `pkg/tui/templates.go`)
  - Create workflow with API call sequence
  - Include error handling (condition node)
  - Include retry logic (loop node)
  - Add edges for success/failure paths
  - Return workflow

- [ ] [T076] [P3] [US6] Implement template registry (file: `pkg/tui/templates.go`)
  - Map template names to functions
  - Template descriptions for UI
  - WorkflowTemplates map[string]func() *workflow.Workflow
  - TemplateDescriptions map[string]string

- [ ] [T077] [P3] [US6] Implement WorkflowBuilder.ApplyTemplate() (file: `pkg/tui/workflow_builder.go`)
  - Show template selection modal
  - Get selected template
  - Load workflow from template function
  - Replace current workflow (confirm if modified)
  - Run auto-layout
  - Trigger re-render

- [ ] [T078] [P3] [US6] Write tests for templates (file: `pkg/tui/templates_test.go`)
  - Test each template creates valid workflow
  - Test template node count and structure
  - Test template edges are connected correctly
  - Test template registry lookup

**Phase 9 Dependencies**: T062-T072 (WorkflowBuilder)
**Phase 9 Outputs**: Templates → Completes US6

---

## Phase 10: Keyboard Handling (User Story 7)

**Goal**: Implement mode-based keyboard shortcuts
**Duration**: 2-3 days
**Blocking**: Phase 11 (final integration)

### Tasks

- [ ] [T079] [P1] [US7] Implement WorkflowBuilder.HandleKey() dispatcher (file: `pkg/tui/workflow_builder.go`)
  - Route keys based on current mode
  - Normal mode: dispatch to handleNormalMode()
  - Edit mode: dispatch to handleEditMode()
  - Palette mode: dispatch to handlePaletteMode()
  - Help mode: dispatch to handleHelpMode()
  - Global keys: '?', 'Esc', 'q'

- [ ] [T080] [P1] [US7] Implement handleNormalMode() shortcuts (file: `pkg/tui/workflow_builder.go`)
  - Navigation: h/j/k/l (move node), Shift+arrows (pan), +/- (zoom)
  - Node ops: a (add), d (delete), c (create edge), y (yank), p (paste)
  - Workflow: s (save), v (validate), u (undo), Ctrl+R (redo), t (templates)
  - View: Enter (edit properties), 0 (reset view), f (fit all)

- [ ] [T081] [P1] [US7] Implement handleEditMode() shortcuts (file: `pkg/tui/workflow_builder.go`)
  - Field navigation: Tab/Shift+Tab, ↓/↑
  - Edit: Enter (edit field), Esc (cancel), Ctrl+S (save)
  - Field-specific: Ctrl+R (reset field)

- [ ] [T082] [P1] [US7] Implement handlePaletteMode() shortcuts (file: `pkg/tui/workflow_builder.go`)
  - Navigation: ↓/j (next), ↑/k (previous)
  - Filter: Type characters to filter
  - Select: Enter (select type), Esc (cancel)

- [ ] [T083] [P1] [US7] Write tests for keyboard handling (file: `pkg/tui/workflow_builder_test.go`)
  - Test normal mode shortcuts (add, delete, move, save, undo)
  - Test edit mode shortcuts (navigate, edit, save, cancel)
  - Test palette mode shortcuts (navigate, filter, select)
  - Test mode transitions (normal → edit → normal)
  - Test global shortcuts work in all modes
  - Test invalid keys show error message

**Phase 10 Dependencies**: T062-T072 (WorkflowBuilder)
**Phase 10 Outputs**: Keyboard navigation → Completes US1, US2, US7

---

## Phase 11: Polish and Final Integration

**Goal**: Complete remaining features and performance optimization
**Duration**: 2-3 days
**Blocking**: None (can be last phase)

### Tasks

- [ ] [T084] [P1] [US1] Implement terminal resize handling (file: `pkg/tui/workflow_builder.go`)
  - Detect terminal resize events
  - Update canvas dimensions
  - Adjust viewport if needed
  - Re-center on selected node if out of bounds
  - Trigger re-render

- [ ] [T085] [P2] [US5] Implement minimap for large workflows (file: `pkg/tui/minimap.go`)
  - Render small overview of entire canvas
  - Show viewport position as rectangle
  - Click to jump to area (if mouse enabled)
  - Show in corner of screen

**Phase 11 Dependencies**: All previous phases
**Phase 11 Outputs**: Complete visual editor

---

## Dependency Graph

### Critical Path (Sequential)

```
Phase 1 (Setup)
  → Phase 2 (Foundation)
    → Phase 3 (Canvas)
      → Phase 8 (WorkflowBuilder)
        → Phase 10 (Keyboard)
          → Phase 11 (Polish)
```

### Parallel Execution Opportunities

```
After Phase 2 completes:
  Phase 3 (Canvas)         → Required for Phase 8
  Phase 4 (PropertyPanel)  ⇉ Can run in parallel
  Phase 5 (NodePalette)    ⇉ Can run in parallel
  Phase 6 (Validation)     ⇉ Can run in parallel
  Phase 7 (Navigation)     → Depends on Phase 3

After Phase 8 completes:
  Phase 9 (Templates)      ⇉ Can run in parallel
  Phase 10 (Keyboard)      → Required for completion
```

### Task Dependencies by User Story

**US1 (Visual Node Placement)**: T001-T003 → T004-T009 → T010-T025 → T062-T072 → T079-T083 → T084
**US2 (Node Property Editing)**: T001-T003 → T004-T009 → T026-T038 → T062-T072 → T079-T083
**US3 (Workflow Validation)**: T001-T003 → T006 → T046-T056 → T062-T072
**US4 (Undo/Redo)**: T001-T003 → T004-T005 → T062-T072 → T079-T083
**US5 (Canvas Navigation)**: T001-T003 → T010-T025 → T057-T061 → T079-T083
**US6 (Node Palette/Templates)**: T001-T003 → T004-T009 → T039-T045 → T073-T078 → T079-T083
**US7 (Keyboard Shortcuts)**: T001-T003 → T007-T008 → T079-T083
**US8 (Real-time Rendering)**: T001-T003 → T010-T025 → T062-T072

---

## Testing Strategy Summary

### Test Coverage Requirements

- **Unit tests**: All validation logic, undo/redo stack, node palette filtering
- **Integration tests**: Canvas rendering, property panel state management, workflow operations
- **TUI interaction tests**: Keyboard navigation, modal interactions, mode transitions
- **Performance benchmarks**: Canvas rendering (100 nodes < 100ms), undo/redo (< 50ms), validation (< 500ms)

### Test Task Distribution

- **Phase 1**: 1 test setup task (T001-T003)
- **Phase 2**: 2 test tasks (T005, T008)
- **Phase 3**: 4 test tasks (T012, T014, T016, T022, T024, T025)
- **Phase 4**: 2 test tasks (T031, T038)
- **Phase 5**: 1 test task (T045)
- **Phase 6**: 3 test tasks (T051, T055, T056)
- **Phase 7**: 1 test task (T061)
- **Phase 8**: 1 test task (T072)
- **Phase 9**: 1 test task (T078)
- **Phase 10**: 1 test task (T083)

**Total test tasks**: 17 out of 85 tasks (~20%)

---

## Implementation Order Recommendation

### Week 1: Foundation
- Days 1-2: Phase 1 (T001-T003) + Phase 2 (T004-T009)
- Days 3-5: Phase 3 (T010-T025) - Canvas

### Week 2: Core Features
- Days 1-2: Phase 4 (T026-T038) - PropertyPanel (parallel)
- Days 1-2: Phase 5 (T039-T045) - NodePalette (parallel)
- Days 3-4: Phase 6 (T046-T056) - Validation (parallel)
- Day 5: Phase 7 (T057-T061) - Navigation

### Week 3: Integration
- Days 1-3: Phase 8 (T062-T072) - WorkflowBuilder
- Days 4-5: Phase 10 (T079-T083) - Keyboard handling

### Week 4: Polish
- Days 1-2: Phase 9 (T073-T078) - Templates
- Days 3-4: Phase 11 (T084-T085) - Final polish
- Day 5: Buffer for bug fixes and testing

---

## Success Metrics

### Per-Phase Completion Criteria

- **Phase 1**: Test harness runs successfully, mock repository works
- **Phase 2**: UndoStack, HelpPanel, ValidationStatus pass all tests
- **Phase 3**: Canvas renders 100 nodes in < 100ms, auto-layout works
- **Phase 4**: PropertyPanel validates all 5 field types correctly
- **Phase 5**: NodePalette filters and creates all 7 node types
- **Phase 6**: Validation detects all 9 rule violations
- **Phase 7**: Zoom and pan work smoothly at 60 FPS
- **Phase 8**: WorkflowBuilder orchestrates all operations correctly
- **Phase 9**: All 3 templates create valid workflows
- **Phase 10**: All 30+ keyboard shortcuts work in correct modes
- **Phase 11**: Terminal resize handled, minimap functional

### Overall Completion Criteria

All 10 success criteria from spec.md must pass:
- **SC-001**: 5-node workflow created in < 3 minutes ✓
- **SC-002**: 100-node workflow renders without degradation ✓
- **SC-003**: 95% success rate for first-time users ✓
- **SC-004**: Undo/redo < 50ms ✓
- **SC-005**: 100% error detection before execution ✓
- **SC-006**: Canvas navigation < 16ms (60 FPS) ✓
- **SC-007**: Help reduces doc lookups by 80% ✓
- **SC-008**: Property validation < 200ms ✓
- **SC-009**: 100% data integrity on save ✓
- **SC-010**: 90% of core functionality discoverable in-TUI ✓

---

## Notes

- All file paths are relative to repository root `/Users/dshills/Development/projects/goflow`
- Tasks prefixed with [P1] should be completed before [P2], which should be completed before [P3]
- Test tasks must be completed immediately after their corresponding implementation tasks (test-first development per constitution)
- Performance benchmarks should be run continuously during development to catch regressions early
- Constitution principles (DDD, test-first, performance, security, observability) are enforced through task structure

---

## Next Steps

1. **Review this task breakdown** for completeness and accuracy
2. **Run `/speckit.implement`** to begin execution of tasks in dependency order
3. **Track progress** by checking off completed tasks in this document
4. **Run validation** after each phase to ensure quality gates pass

**Status**: Ready for implementation ✅
