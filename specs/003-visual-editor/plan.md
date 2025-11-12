# Implementation Plan: Complete Visual Workflow Editor

**Branch**: `003-visual-editor` | **Date**: 2025-11-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-visual-editor/spec.md`

## Summary

This feature completes the GoFlow visual workflow editor TUI, enabling developers to visually construct, edit, and validate workflows through an interactive terminal interface. The implementation focuses on completing 8 core capabilities: visual node placement and connection, node property editing, workflow validation with error highlighting, undo/redo support, canvas navigation and zoom, node type palette and templates, keyboard shortcuts and help overlay, and real-time workflow rendering. The technical approach uses the existing `pkg/tui` scaffolding with the goterm library for rendering, integrates with the workflow domain model for persistence, and implements vim-style keyboard navigation for all operations.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- `github.com/dshills/goterm` - Terminal UI framework (existing)
- `github.com/dshills/goflow/pkg/workflow` - Domain model (existing)
- `github.com/expr-lang/expr` - Expression validation (existing)
- `github.com/tidwall/gjson` - JSONPath validation (existing)

**Storage**: Filesystem (YAML workflow definitions via existing workflow.Repository interface)
**Testing**: Go standard testing (`go test`), table-driven tests for validation logic, TUI interaction tests using goterm test utilities
**Target Platform**: macOS, Linux, Windows terminals (cross-platform TUI)
**Project Type**: Single project (extension of existing `pkg/tui`)
**Performance Goals**:
- Canvas rendering < 100ms per frame (60 FPS target: 16ms)
- Undo/redo operations < 50ms
- Property validation < 200ms
- Support workflows up to 100 nodes without degradation

**Constraints**:
- Must work in standard terminals (no GUI dependencies)
- Vim-style keyboard navigation required (h/j/k/l)
- All operations keyboard-accessible (no mouse required)
- Terminal resize must be handled gracefully

**Scale/Scope**:
- Support workflows up to 200 nodes
- Undo stack depth: 100 operations
- Node types: 7 (MCP Tool, Transform, Condition, Loop, Parallel, Start, End)
- 30+ keyboard shortcuts
- 3 workflow templates (ETL Pipeline, API Integration, Batch Processing)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Domain-Driven Design (DDD)
**Status**: ✅ PASS

**Analysis**: The visual editor is a presentation layer concern, not a new aggregate. It operates on the existing Workflow aggregate (`pkg/workflow`) and respects aggregate boundaries by:
- Using workflow.Node, workflow.Edge, workflow.Workflow (not creating new domain entities)
- Interacting through workflow.Repository interface for persistence
- Maintaining canvas-specific view state (positions, zoom, selection) separate from domain model
- All domain mutations go through the Workflow aggregate's methods

The canvas entities (canvasNode, canvasEdge, Canvas) are pure UI state and do not violate aggregate boundaries.

### Test-First Development
**Status**: ✅ PASS (with plan)

**Analysis**: The feature spec includes 40 acceptance scenarios in Given-When-Then format, providing clear test cases. Implementation will follow Red-Green-Refactor:
1. Write tests for each functional requirement (FR-001 through FR-030)
2. Tests will fail initially (existing TODOs in implementation)
3. Implement features to make tests pass
4. Refactor with tests as safety net

Test coverage targets:
- Unit tests: All validation logic, undo/redo stack, node palette filtering
- Integration tests: Canvas rendering, property panel state management
- TUI interaction tests: Keyboard navigation, modal interactions

### Performance Consciousness
**Status**: ✅ PASS

**Analysis**: Feature spec defines explicit performance targets matching constitution requirements:
- SC-002: 100 nodes at 60 FPS (< 100ms per frame) ✓
- SC-004: Undo/redo < 50ms ✓
- SC-006: Canvas navigation < 16ms (60 FPS) ✓
- SC-008: Validation < 200ms ✓

Implementation will include:
- Canvas virtualization for large workflows (render only visible nodes)
- Performance mode for simplified rendering
- Benchmark tests for critical paths (rendering, undo/redo, validation)

### Security by Design
**Status**: ✅ PASS

**Analysis**: Visual editor has minimal security surface:
- No new credential storage (uses existing MCP server registry)
- Expression validation uses existing sandboxed evaluator (`expr-lang/expr`)
- No arbitrary code execution
- Workflow files remain shareable without secrets (no change to existing model)
- User input validated through existing workflow validation layer

Security considerations:
- Validate workflow names using existing `workflow.IsValidWorkflowName()`
- Sanitize display strings to prevent terminal escape sequence injection
- Limit undo stack size to prevent memory exhaustion (100 operations max)

### Observable and Debuggable
**Status**: ✅ PASS

**Analysis**: Visual editor enhances observability:
- Real-time validation with error highlighting (SC-005: 100% error detection)
- Status bar shows modification state, validation status, current mode
- Help overlay provides context-sensitive documentation
- Validation panel lists all errors with node IDs for navigation
- Undo/redo stack provides operation history

Debugging support:
- Canvas state can be inspected (node positions, selections, validation status)
- Property panel shows field-level validation errors
- Status messages for all operations
- TUI test harness for reproducing interaction bugs

## Project Structure

### Documentation (this feature)

```text
specs/003-visual-editor/
├── plan.md              # This file
├── research.md          # Phase 0: Design decisions and alternatives
├── data-model.md        # Phase 1: Canvas and UI entities
├── quickstart.md        # Phase 1: Developer guide for TUI components
├── contracts/           # Phase 1: Component interfaces
│   ├── canvas.md        # Canvas rendering contract
│   ├── property_panel.md # Property editing contract
│   ├── node_palette.md  # Node selection contract
│   └── validation.md    # Validation contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (existing)
└── tasks.md             # Phase 2: Created by /speckit.tasks
```

### Source Code (repository root)

```text
pkg/tui/
├── workflow_builder.go           # Main builder (existing - 1356 lines)
├── workflow_builder_test.go      # Builder tests (new)
├── workflow_builder_condition_test.go  # Condition node tests (existing)
├── canvas.go                     # Canvas rendering (new)
├── canvas_test.go                # Canvas tests (new)
├── canvas_layout.go              # Auto-layout algorithm (new)
├── canvas_zoom.go                # Zoom and pan (new)
├── node_palette.go               # Node type selector (new)
├── node_palette_test.go          # Palette tests (new)
├── property_panel.go             # Property editor (new)
├── property_panel_test.go        # Property tests (new)
├── property_fields.go            # Field validators (new)
├── help_panel.go                 # Help overlay (new)
├── help_panel_test.go            # Help tests (new)
├── validation_panel.go           # Validation details (new)
├── undo_stack.go                 # Undo/redo (new)
├── undo_stack_test.go            # Undo tests (new)
├── templates.go                  # Workflow templates (new)
├── templates_test.go             # Template tests (new)
└── components/                   # Existing UI components
    ├── list.go
    ├── modal.go
    ├── panel.go
    ├── statusbar.go
    └── button.go

pkg/workflow/                      # Existing domain model (no changes)
└── (no changes to domain aggregate)

tests/tui/                         # TUI integration tests (new)
├── builder_interaction_test.go
├── canvas_rendering_test.go
└── property_editing_test.go
```

**Structure Decision**: Single project extension. The visual editor is a natural extension of the existing `pkg/tui` package and operates on the existing workflow domain model. No new aggregates or services required. All new code goes into `pkg/tui` with clear separation of concerns: canvas rendering, property editing, undo management, and user interaction handled by focused modules.

## Complexity Tracking

> No constitutional violations identified. All checks pass.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |

## Phase 0: Research & Design Decisions

**Status**: Ready to execute

**Research Questions**:

1. **Canvas Rendering Strategy**: How should the canvas render nodes and edges efficiently?
   - Options: ASCII art, Unicode box drawing, simple text-based layout
   - Considerations: Terminal compatibility, rendering performance, visual clarity

2. **Edge Routing Algorithm**: How should edges be drawn between nodes?
   - Options: Straight lines, orthogonal (Manhattan), curved (splines), automatic routing
   - Considerations: Overlap avoidance, visual clarity, performance

3. **Auto-layout Algorithm**: How should nodes be positioned when loading workflows without position metadata?
   - Options: Topological hierarchical layout, force-directed, grid-based, random with spacing
   - Considerations: Readability, performance, determinism

4. **Property Panel Field Types**: What field types and validation strategies are needed?
   - JSONPath validation strategy
   - Expression syntax validation
   - Template string validation
   - Condition expression validation

5. **Undo/Redo Granularity**: What operations should be undoable?
   - Single field edits vs. complete property panel save
   - Individual node moves vs. batch moves
   - Memory vs. usability tradeoff

6. **Canvas Coordinate System**: What coordinate system for node positioning?
   - Options: Terminal character cells, virtual pixels, logical units
   - Considerations: Zoom scaling, terminal resize handling

7. **Keyboard Shortcut Conflicts**: How to handle mode-based shortcuts?
   - Normal mode vs. edit mode
   - Global shortcuts vs. panel-specific
   - Escape sequences for complex operations

**Output**: `research.md` with decisions and rationale

## Phase 1: Design & Contracts

**Prerequisites**: `research.md` complete

**Deliverables**:

1. **data-model.md**: Canvas and UI entity relationships
   - Canvas (viewport, zoom, node map, edge list)
   - canvasNode (position, size, visual state)
   - canvasEdge (routing points, visual state)
   - PropertyPanel (fields, validation state)
   - NodePalette (types, filter, selection)
   - UndoStack (snapshots, capacity, cursor)
   - ValidationStatus (errors, warnings)
   - HelpPanel (keybindings, context)

2. **contracts/**: Component interface contracts
   - `canvas.md`: Rendering interface, zoom/pan operations, selection management
   - `property_panel.md`: Field editing, validation, save/cancel
   - `node_palette.md`: Node type selection, filtering, template application
   - `validation.md`: Validation triggers, error display, navigation

3. **quickstart.md**: Developer guide
   - How to add new node types to palette
   - How to add new property field types
   - How to add new keyboard shortcuts
   - How to test TUI interactions
   - How to add workflow templates

**Output**: Design artifacts in `/specs/003-visual-editor/`

## Phase 2: Task Breakdown

**Note**: This phase is executed by `/speckit.tasks` command (NOT by `/speckit.plan`)

**Process**:
1. User runs `/speckit.tasks` after reviewing this plan
2. Tasks are generated from user stories and functional requirements
3. Tasks include test requirements, implementation steps, and acceptance criteria
4. Tasks are ordered by priority (P1 → P2 → P3) and dependencies

**Output**: `tasks.md` with dependency-ordered implementation tasks

## Post-Design Constitution Re-Check

**Status**: ✅ ALL CHECKS PASSED

After completing design artifacts (research.md, data-model.md, contracts/, quickstart.md), all constitutional principles remain satisfied:

1. **DDD**: ✅ Canvas entities remain pure UI state, no aggregate boundary violations
2. **Test-First**: ✅ Contracts define testable interfaces with clear preconditions/postconditions
3. **Performance**: ✅ Research decisions target all performance goals (< 16ms rendering, < 50ms undo)
4. **Security**: ✅ Uses existing validation layers, no new security surface
5. **Observability**: ✅ ValidationStatus and error display enhance debuggability

**No design changes required. Proceed to Phase 2 (Task Breakdown).**

## Next Steps

1. ✅ Review this plan for completeness and accuracy
2. ✅ Execute Phase 0: Run research to resolve design decisions
3. ✅ Execute Phase 1: Generate data model and contracts
4. → Run `/speckit.tasks` to generate task breakdown
5. → Run `/speckit.implement` to begin implementation

**Gate Status**: All constitution checks PASSED. Ready for Phase 2.
