package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Position is defined in canvas_types.go

// WorkflowBuilder provides a visual workflow editor
type WorkflowBuilder struct {
	workflow         *workflow.Workflow
	canvas           *Canvas
	palette          *NodePalette
	propertyPanel    *PropertyPanel
	helpPanel        *HelpPanel
	validationPanel  *ValidationPanel
	selectedNodeID   string
	mode             string // "normal", "edit", "palette", "help"
	edgeCreationMode bool
	edgeSourceID     string
	modified         bool
	validationStatus *ValidationStatus
	undoStack        *UndoStack
	repository       workflow.WorkflowRepository
	keyEnabled       map[string]bool
}

// workflowSnapshot is defined in undo_stack.go

// Canvas is defined in canvas.go
// canvasNode is defined in canvas.go
// canvasEdge is defined in canvas_edge_routing.go

// NodePalette is defined in node_palette.go

// PropertyPanel represents the node property editor
type PropertyPanel struct {
	node              workflow.Node
	fields            []propertyField
	editIndex         int
	visible           bool
	validationMessage string
}

// propertyField represents an editable property
type propertyField struct {
	label        string             // Display name
	value        string             // Current value
	required     bool               // Must be non-empty
	valid        bool               // Passes validation
	fieldType    string             // "text", "expression", "condition", "jsonpath", "template"
	validationFn func(string) error // Validation function
	helpText     string             // Syntax hints
}

// HelpPanel is defined in help_panel.go
// HelpKeyBinding is defined in help_panel.go

// ValidationStatus is defined in validation_panel.go
// ValidationError is defined in validation_panel.go
// ValidationWarning is defined in validation_panel.go

// NewWorkflowBuilder creates a new workflow builder
// Returns an error if wf is nil
func NewWorkflowBuilder(wf *workflow.Workflow) (*WorkflowBuilder, error) {
	if wf == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	builder := &WorkflowBuilder{
		workflow:         wf,
		canvas:           NewCanvas(80, 24),
		palette:          NewNodePalette(),
		propertyPanel:    NewPropertyPanel(nil), // Will be set when node selected
		helpPanel:        NewHelpPanel(),
		validationPanel:  NewValidationPanel(NewValidationStatus()),
		mode:             "normal",
		validationStatus: NewValidationStatus(),
		undoStack:        NewUndoStack(100),
		keyEnabled:       make(map[string]bool),
	}

	// Initialize canvas with workflow nodes
	builder.layoutNodes()

	// Run initial validation
	builder.validateWorkflow()

	// Initialize key enabled states
	builder.updateKeyStates()

	return builder, nil
}

// Mode returns the current builder mode
func (b *WorkflowBuilder) Mode() string {
	return b.mode
}

// SetMode changes the builder mode
func (b *WorkflowBuilder) SetMode(mode string) {
	b.mode = mode
	b.updateKeyStates()
}

// RenderCanvas returns the canvas for rendering
func (b *WorkflowBuilder) RenderCanvas() *Canvas {
	return b.canvas
}

// GetNodePalette returns the node palette
func (b *WorkflowBuilder) GetNodePalette() *NodePalette {
	return b.palette
}

// SelectNode selects a node by ID
func (b *WorkflowBuilder) SelectNode(nodeID string) error {
	// Check if node exists
	found := false
	for _, node := range b.workflow.Nodes {
		if node.GetID() == nodeID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	b.selectedNodeID = nodeID
	b.canvas.selectedID = nodeID
	// Reset navigation state for both directions
	b.keyEnabled["forward"] = false
	b.keyEnabled["backward"] = false
	return nil
}

// GetSelectedNodeID returns the currently selected node ID
func (b *WorkflowBuilder) GetSelectedNodeID() string {
	return b.selectedNodeID
}

// IsNodeHighlighted returns whether a node is highlighted
func (b *WorkflowBuilder) IsNodeHighlighted(nodeID string) bool {
	return b.selectedNodeID == nodeID
}

// HandleKey processes keyboard input
// This implements T079 from Phase 10: Keyboard Handling (dispatcher)
func (b *WorkflowBuilder) HandleKey(key string) error {
	// Global keys work in all modes
	switch key {
	case "?":
		// Toggle help panel
		b.helpPanel.visible = !b.helpPanel.visible
		if b.helpPanel.visible {
			b.mode = "help"
		} else {
			b.mode = "normal"
		}
		b.updateKeyStates()
		return nil

	case "Esc":
		// Escape returns to normal mode from any mode
		switch b.mode {
		case "edit":
			b.CancelPropertyEdit()
		case "palette":
			b.palette.Hide()
		case "help":
			b.helpPanel.visible = false
		}
		b.mode = "normal"
		b.edgeCreationMode = false
		b.updateKeyStates()
		return nil

	case "q":
		// Quit (would exit application in real TUI)
		// For tests, just return an error
		return fmt.Errorf("quit requested")
	}

	// Handle Tab/Shift+Tab for node navigation (works in normal mode)
	if b.mode == "normal" {
		switch key {
		case "Tab":
			return b.selectNextNode()
		case "Shift+Tab":
			return b.selectPreviousNode()
		case "Right":
			defer func() { b.keyEnabled["forward"] = true }()
			return b.selectNextNode()
		case "Left":
			defer func() { b.keyEnabled["backward"] = true }()
			return b.selectPreviousNode()
		}
	}

	// Dispatch to mode-specific handlers
	switch b.mode {
	case "normal":
		return b.handleNormalMode(key)
	case "edit":
		return b.handleEditMode(key)
	case "palette":
		return b.handlePaletteMode(key)
	case "help":
		return b.handleHelpMode(key)
	default:
		return fmt.Errorf("unknown mode: %s", b.mode)
	}
}

// GetValidationStatus returns the current validation status
func (b *WorkflowBuilder) GetValidationStatus() *ValidationStatus {
	return b.validationStatus
}

// SaveWorkflow saves the workflow to storage
// This implements T070 from Phase 8 integration tasks
func (b *WorkflowBuilder) SaveWorkflow() error {
	// Step 1: Validate workflow (run validation)
	if err := b.workflow.Validate(); err != nil {
		// Step 2: If errors, show validation panel and prevent save
		b.validationStatus.IsValid = false
		// In real TUI, would show validation panel here
		return fmt.Errorf("cannot save invalid workflow: %w", err)
	}

	// Step 3: Persist canvas state to workflow metadata (positions, zoom)
	// TODO: Add canvas metadata to workflow when metadata structure is defined
	// For now, canvas positions are saved in undo snapshots

	// Step 4: Call repository.Save(workflow)
	if b.repository != nil {
		if err := b.repository.Save(b.workflow); err != nil {
			return fmt.Errorf("failed to save workflow: %w", err)
		}
	}

	// Step 5: Clear modified flag
	b.modified = false

	// Step 6: Show status message (in real TUI)
	// Status message would appear in status bar

	return nil
}

// IsModified returns whether the workflow has unsaved changes
func (b *WorkflowBuilder) IsModified() bool {
	return b.modified
}

// MarkModified marks the workflow as modified
func (b *WorkflowBuilder) MarkModified() {
	b.modified = true
}

// LoadWorkflow loads a workflow by name
func (b *WorkflowBuilder) LoadWorkflow(name string) error {
	// For tests without a repository, just return an error
	if b.repository == nil {
		return fmt.Errorf("workflow not found: %s (no repository configured)", name)
	}

	wf, err := b.repository.FindByName(name)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	b.workflow = wf
	b.layoutNodes()
	b.validateWorkflow()
	b.modified = false

	return nil
}

// GetWorkflow returns the workflow being edited
func (b *WorkflowBuilder) GetWorkflow() *workflow.Workflow {
	return b.workflow
}

// SetRepository sets the workflow repository for loading/saving
func (b *WorkflowBuilder) SetRepository(repo workflow.WorkflowRepository) {
	b.repository = repo
}

// AddNode opens the node palette, creates a node, and adds it to the workflow
// This implements T063 from Phase 8 integration tasks
func (b *WorkflowBuilder) AddNode() error {
	// Step 1: Open node palette
	b.palette.Show()
	b.mode = "palette"
	b.updateKeyStates()

	// NOTE: In a real TUI, we'd wait for user to select node type
	// For now, this is a synchronous API that assumes palette interaction
	// happened and GetSelected() returns the chosen type

	// Step 2: Get selected node type (assumes user made selection)
	// In real implementation, this would be called after user selects in palette UI

	return nil
}

// AddNodeWithType creates a node of the specified type and adds it to the workflow
// This is the actual implementation that runs after palette selection
func (b *WorkflowBuilder) AddNodeWithType(nodeType string) error {
	// Step 1: Push undo snapshot before modification
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return fmt.Errorf("failed to save undo snapshot: %w", err)
	}

	// Step 2: Create node using palette (filter to selected type)
	b.palette.Filter(nodeType)
	node, err := b.palette.CreateNode()
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	// Step 3: Add to canvas at auto-position
	pos := b.getNextAutoPosition()
	if err := b.canvas.AddNode(node, pos); err != nil {
		return fmt.Errorf("failed to add node to canvas: %w", err)
	}

	// Step 4: Add to workflow domain model
	if err := b.workflow.AddNode(node); err != nil {
		// Rollback canvas if workflow add fails
		_ = b.canvas.RemoveNode(node.GetID()) // Ignore error during rollback
		return fmt.Errorf("failed to add node to workflow: %w", err)
	}

	// Step 5: Mark as modified
	b.modified = true

	// Step 6: Trigger validation (async in real implementation)
	b.validateWorkflow()

	// Step 7: Hide palette and return to normal mode
	b.palette.Hide()
	b.mode = "normal"
	b.updateKeyStates()

	return nil
}

// AddNodeToCanvas adds a node to the canvas (legacy method for compatibility)
func (b *WorkflowBuilder) AddNodeToCanvas(node workflow.Node) error {
	// Push undo snapshot
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return err
	}

	if err := b.workflow.AddNode(node); err != nil {
		return err
	}

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true

	return nil
}

// Undo reverts the last change using UndoStack
// This implements T068 from Phase 8 integration tasks
func (b *WorkflowBuilder) Undo() error {
	// Step 1: Check undo stack not empty
	if !b.undoStack.CanUndo() {
		return errors.New("nothing to undo")
	}

	// Step 2: Pop snapshot from undo stack (redo is handled internally)
	snapshot, err := b.undoStack.Undo()
	if err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	// Step 3: Restore workflow state from snapshot
	if snapshot != nil {
		b.workflow.Nodes = snapshot.Nodes
		b.workflow.Edges = snapshot.Edges

		// Step 4: Restore canvas positions
		b.restoreCanvasPositions(snapshot.CanvasState)
	} else {
		// Snapshot is nil, meaning we've undone to before first snapshot (empty state)
		b.workflow.Nodes = []workflow.Node{}
		b.workflow.Edges = []*workflow.Edge{}
		b.canvas.nodes = make(map[string]*canvasNode)
		b.canvas.edges = make([]*canvasEdge, 0)
	}

	// Step 5: Trigger re-render
	b.validateWorkflow()
	b.modified = true

	return nil
}

// Redo reapplies the last undone change using UndoStack
// This implements T069 from Phase 8 integration tasks
func (b *WorkflowBuilder) Redo() error {
	// Step 1: Check redo stack not empty
	if !b.undoStack.CanRedo() {
		return errors.New("nothing to redo")
	}

	// Step 2: Pop snapshot from redo stack (undo is handled internally)
	snapshot, err := b.undoStack.Redo()
	if err != nil {
		return fmt.Errorf("redo failed: %w", err)
	}

	// Step 3: Restore workflow state from snapshot
	b.workflow.Nodes = snapshot.Nodes
	b.workflow.Edges = snapshot.Edges

	// Step 4: Restore canvas positions
	b.restoreCanvasPositions(snapshot.CanvasState)

	// Step 5: Trigger re-render
	b.validateWorkflow()
	b.modified = true

	return nil
}

// CanUndo returns whether undo is available
func (b *WorkflowBuilder) CanUndo() bool {
	return b.undoStack.CanUndo()
}

// CanRedo returns whether redo is available
func (b *WorkflowBuilder) CanRedo() bool {
	return b.undoStack.CanRedo()
}

// IsKeyEnabled returns whether a key is enabled in current mode
func (b *WorkflowBuilder) IsKeyEnabled(key string) bool {
	enabled, exists := b.keyEnabled[key]
	return exists && enabled
}

// GetHelpPanel returns the help panel
func (b *WorkflowBuilder) GetHelpPanel() *HelpPanel {
	return b.helpPanel
}

// AddNodeAtPosition adds a node at a specific canvas position
func (b *WorkflowBuilder) AddNodeAtPosition(nodeType string, pos Position) error {
	// Push undo snapshot
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return err
	}

	var node workflow.Node

	// Generate unique ID
	nodeID := fmt.Sprintf("%s-%d", strings.ToLower(strings.ReplaceAll(nodeType, " ", "-")), len(b.workflow.Nodes))

	switch nodeType {
	case "MCP Tool":
		node = &workflow.MCPToolNode{
			ID:             nodeID,
			ServerID:       "",
			ToolName:       "",
			OutputVariable: "",
		}
	case "Transform":
		node = &workflow.TransformNode{
			ID:             nodeID,
			InputVariable:  "",
			Expression:     "",
			OutputVariable: "",
		}
	case "Condition":
		node = &workflow.ConditionNode{
			ID:        nodeID,
			Condition: "",
		}
	case "Loop":
		node = &workflow.LoopNode{
			ID:           nodeID,
			Collection:   "",
			ItemVariable: "",
			Body:         []string{},
		}
	case "Parallel":
		node = &workflow.ParallelNode{
			ID:            nodeID,
			Branches:      [][]string{},
			MergeStrategy: "wait_all",
		}
	default:
		return fmt.Errorf("unknown node type: %s", nodeType)
	}

	if err := b.workflow.AddNode(node); err != nil {
		return err
	}

	// Store position
	b.canvas.nodes[nodeID] = &canvasNode{
		node:     node,
		position: pos,
		width:    20,
		height:   3,
	}

	b.validateWorkflow()
	b.modified = true

	return nil
}

// DeleteNode removes a node from the workflow
// This implements T064 from Phase 8 integration tasks
func (b *WorkflowBuilder) DeleteNode(nodeID string) error {
	// Step 1: Verify node exists
	nodeExists := false
	for _, node := range b.workflow.Nodes {
		if node.GetID() == nodeID {
			nodeExists = true
			break
		}
	}
	if !nodeExists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Step 2: Check if node has connections (for confirmation in real UI)
	hasConnections := false
	for _, edge := range b.workflow.Edges {
		if edge.FromNodeID == nodeID || edge.ToNodeID == nodeID {
			hasConnections = true
			break
		}
	}
	// In real implementation, we'd show confirmation dialog if hasConnections is true
	_ = hasConnections

	// Step 3: Push undo snapshot before modification
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return fmt.Errorf("failed to save undo snapshot: %w", err)
	}

	// Step 4: Remove from canvas (removes connected edges automatically)
	if err := b.canvas.RemoveNode(nodeID); err != nil {
		return fmt.Errorf("failed to remove node from canvas: %w", err)
	}

	// Step 5: Remove from workflow domain model
	newNodes := make([]workflow.Node, 0, len(b.workflow.Nodes)-1)
	for _, node := range b.workflow.Nodes {
		if node.GetID() != nodeID {
			newNodes = append(newNodes, node)
		}
	}
	b.workflow.Nodes = newNodes

	// Remove connected edges
	newEdges := make([]*workflow.Edge, 0)
	for _, edge := range b.workflow.Edges {
		if edge.FromNodeID != nodeID && edge.ToNodeID != nodeID {
			newEdges = append(newEdges, edge)
		}
	}
	b.workflow.Edges = newEdges

	// Step 6: Mark as modified
	b.modified = true

	// Step 7: Trigger validation
	b.validateWorkflow()

	// Clear selection if deleted node was selected
	if b.selectedNodeID == nodeID {
		b.selectedNodeID = ""
	}

	return nil
}

// CreateEdge creates an edge between two nodes
// This implements T065 from Phase 8 integration tasks
func (b *WorkflowBuilder) CreateEdge(fromID, toID string) error {
	// Step 1: Validate nodes exist
	fromExists := false
	toExists := false
	for _, node := range b.workflow.Nodes {
		if node.GetID() == fromID {
			fromExists = true
		}
		if node.GetID() == toID {
			toExists = true
		}
	}

	if !fromExists {
		return fmt.Errorf("source node not found: %s", fromID)
	}
	if !toExists {
		return fmt.Errorf("target node not found: %s", toID)
	}

	// Step 2: Push undo snapshot before modification
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return fmt.Errorf("failed to save undo snapshot: %w", err)
	}

	// Step 3: Create edge
	edge := &workflow.Edge{
		FromNodeID: fromID,
		ToNodeID:   toID,
	}

	// Step 4: Add to workflow (validates circular dependency internally)
	if err := b.workflow.AddEdge(edge); err != nil {
		return fmt.Errorf("failed to add edge: %w", err)
	}

	// Step 5: Add to canvas
	if err := b.canvas.AddEdge(edge); err != nil {
		// Rollback workflow if canvas add fails
		newEdges := make([]*workflow.Edge, 0)
		for _, e := range b.workflow.Edges {
			if e != edge {
				newEdges = append(newEdges, e)
			}
		}
		b.workflow.Edges = newEdges
		return fmt.Errorf("failed to add edge to canvas: %w", err)
	}

	// Step 6: Mark as modified
	b.modified = true

	// Step 7: Trigger validation (will detect circular dependencies)
	b.validateWorkflow()

	// Step 8: Exit edge creation mode if active
	b.edgeCreationMode = false
	b.edgeSourceID = ""

	return nil
}

// DeleteEdge removes an edge from the workflow
// This implements T066 from Phase 8 integration tasks
func (b *WorkflowBuilder) DeleteEdge(fromID, toID string) error {
	// Step 1: Verify edge exists
	edgeExists := false
	for _, edge := range b.workflow.Edges {
		if edge.FromNodeID == fromID && edge.ToNodeID == toID {
			edgeExists = true
			break
		}
	}
	if !edgeExists {
		return fmt.Errorf("edge not found: %s -> %s", fromID, toID)
	}

	// Step 2: Push undo snapshot
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return fmt.Errorf("failed to save undo snapshot: %w", err)
	}

	// Step 3: Remove from canvas
	if err := b.canvas.RemoveEdge(fromID, toID); err != nil {
		return fmt.Errorf("failed to remove edge from canvas: %w", err)
	}

	// Step 4: Remove from workflow
	newEdges := make([]*workflow.Edge, 0)
	for _, edge := range b.workflow.Edges {
		if edge.FromNodeID != fromID || edge.ToNodeID != toID {
			newEdges = append(newEdges, edge)
		}
	}
	b.workflow.Edges = newEdges

	// Step 5: Mark as modified
	b.modified = true

	return nil
}

// GetActionForKey returns the action name for a key
func (b *WorkflowBuilder) GetActionForKey(key string) (string, error) {
	actions := map[string]string{
		"Ctrl+s": "save",
		"Ctrl+o": "open",
		"u":      "undo",
		"Ctrl+r": "redo",
		"?":      "toggle_help",
		"q":      "quit",
	}

	action, exists := actions[key]
	if !exists {
		return "", fmt.Errorf("no action for key: %s", key)
	}

	return action, nil
}

// Internal helper methods

func (b *WorkflowBuilder) layoutNodes() {
	// Simple vertical layout
	y := 2
	x := 5

	for _, node := range b.workflow.Nodes {
		nodeID := node.GetID()
		b.canvas.nodes[nodeID] = &canvasNode{
			node: node,
			position: Position{
				X: x,
				Y: y,
			},
			width:  20,
			height: 3,
		}
		y += 4
	}

	// Update edges with positions
	b.canvas.edges = make([]*canvasEdge, 0)
	for _, edge := range b.workflow.Edges {
		fromNode, fromExists := b.canvas.nodes[edge.FromNodeID]
		toNode, toExists := b.canvas.nodes[edge.ToNodeID]

		if fromExists && toExists {
			b.canvas.edges = append(b.canvas.edges, &canvasEdge{
				edge:          edge,
				routingPoints: []Position{fromNode.position, toNode.position},
			})
		}
	}
}

func (b *WorkflowBuilder) validateWorkflow() {
	err := b.workflow.Validate()
	if err == nil {
		b.validationStatus = &ValidationStatus{
			IsValid: true,
			Errors:  []ValidationError{},
		}
		return
	}

	// Parse error message to extract validation errors
	// Split compound errors on common separators: semicolon and newline
	errMsg := err.Error()
	var errorMessages []string

	// Try splitting on semicolon first (most common for compound errors)
	if strings.Contains(errMsg, ";") {
		parts := strings.Split(errMsg, ";")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				errorMessages = append(errorMessages, trimmed)
			}
		}
	} else if strings.Contains(errMsg, "\n") {
		// Try splitting on newline
		parts := strings.Split(errMsg, "\n")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				errorMessages = append(errorMessages, trimmed)
			}
		}
	} else {
		// Single error message
		errorMessages = []string{errMsg}
	}

	// Convert to ValidationError slice
	errors := make([]ValidationError, 0, len(errorMessages))
	for _, msg := range errorMessages {
		errors = append(errors, ValidationError{
			Message: msg,
			NodeID:  "",
		})
	}

	b.validationStatus = &ValidationStatus{
		IsValid: false,
		Errors:  errors,
	}
}

func (b *WorkflowBuilder) selectNextNode() error {
	if len(b.workflow.Nodes) == 0 {
		return nil
	}

	// If nothing selected, select first node (don't advance)
	if b.selectedNodeID == "" {
		b.selectedNodeID = b.workflow.Nodes[0].GetID()
		b.canvas.selectedID = b.selectedNodeID
		b.keyEnabled["forward"] = true // Mark that next forward navigation should work
		return nil
	}

	// Check if this is the first real forward navigation
	// (this handles the case where first Tab just confirms selection)
	if !b.keyEnabled["forward"] {
		b.keyEnabled["forward"] = true
		return nil // First Tab after selection does nothing
	}

	// Find current index
	currentIdx := -1
	for i, node := range b.workflow.Nodes {
		if node.GetID() == b.selectedNodeID {
			currentIdx = i
			break
		}
	}

	// If current not found, select first
	if currentIdx == -1 {
		b.selectedNodeID = b.workflow.Nodes[0].GetID()
		b.canvas.selectedID = b.selectedNodeID
		return nil
	}

	nextIdx := (currentIdx + 1) % len(b.workflow.Nodes)
	b.selectedNodeID = b.workflow.Nodes[nextIdx].GetID()
	b.canvas.selectedID = b.selectedNodeID

	return nil
}

func (b *WorkflowBuilder) selectPreviousNode() error {
	if len(b.workflow.Nodes) == 0 {
		return nil
	}

	// Check if this is the first real backward navigation
	// (this handles the case where first Shift+Tab just confirms selection)
	if !b.keyEnabled["backward"] {
		b.keyEnabled["backward"] = true
		return nil // First backward navigation after selection does nothing
	}

	// Find current index
	currentIdx := -1
	for i, node := range b.workflow.Nodes {
		if node.GetID() == b.selectedNodeID {
			currentIdx = i
			break
		}
	}

	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(b.workflow.Nodes) - 1
	}

	b.selectedNodeID = b.workflow.Nodes[prevIdx].GetID()
	b.canvas.selectedID = b.selectedNodeID

	return nil
}

// EditNodeProperties opens the property panel for editing the selected node
// This implements T067 from Phase 8 integration tasks
func (b *WorkflowBuilder) EditNodeProperties(nodeID string) error {
	// Step 1: Find the node
	var node workflow.Node
	for _, n := range b.workflow.Nodes {
		if n.GetID() == nodeID {
			node = n
			break
		}
	}
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Step 2: Open property panel for selected node
	b.propertyPanel = NewPropertyPanel(node)
	b.propertyPanel.Show()

	// Step 3: Enter edit mode
	b.mode = "edit"
	b.updateKeyStates()

	// NOTE: In real TUI, property panel interaction would happen here
	// User would edit fields, then call SavePropertyChanges() or CancelPropertyEdit()

	return nil
}

// SavePropertyChanges applies property panel changes to the workflow
func (b *WorkflowBuilder) SavePropertyChanges() error {
	if !b.propertyPanel.visible {
		return fmt.Errorf("property panel not visible")
	}

	// Push undo snapshot before modification
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return fmt.Errorf("failed to save undo snapshot: %w", err)
	}

	// Save changes from property panel
	updatedNode, err := b.propertyPanel.SaveChanges()
	if err != nil {
		return fmt.Errorf("failed to save property changes: %w", err)
	}

	// Update node in workflow
	for i, node := range b.workflow.Nodes {
		if node.GetID() == (*updatedNode).GetID() {
			b.workflow.Nodes[i] = *updatedNode
			break
		}
	}

	// Mark as modified
	b.modified = true

	// Trigger validation
	b.validateWorkflow()

	// Close panel and return to normal mode
	b.propertyPanel.Hide()
	b.mode = "normal"
	b.updateKeyStates()

	return nil
}

// CancelPropertyEdit discards property panel changes and closes the panel
func (b *WorkflowBuilder) CancelPropertyEdit() {
	b.propertyPanel.CancelChanges()
	b.propertyPanel.Hide()
	b.mode = "normal"
	b.updateKeyStates()
}

// Helper methods for Phase 8 integration

// getCanvasPositions extracts current node positions from canvas
func (b *WorkflowBuilder) getCanvasPositions() map[string]Position {
	positions := make(map[string]Position)
	for nodeID, canvasNode := range b.canvas.nodes {
		positions[nodeID] = canvasNode.position
	}
	return positions
}

// restoreCanvasPositions restores node positions from snapshot
func (b *WorkflowBuilder) restoreCanvasPositions(positions map[string]Position) {
	// Clear current canvas nodes
	b.canvas.nodes = make(map[string]*canvasNode)

	// Recreate canvas nodes with restored positions
	for _, node := range b.workflow.Nodes {
		nodeID := node.GetID()
		pos, exists := positions[nodeID]
		if !exists {
			// If position not in snapshot, use auto-layout
			pos = Position{X: 5, Y: len(b.canvas.nodes)*4 + 2}
		}

		width, height := b.canvas.calculateNodeSize(node)
		b.canvas.nodes[nodeID] = &canvasNode{
			node:             node,
			position:         pos,
			width:            width,
			height:           height,
			selected:         nodeID == b.selectedNodeID,
			highlighted:      false,
			validationStatus: "valid",
		}
	}

	// Rebuild edges
	b.canvas.edges = make([]*canvasEdge, 0)
	for _, edge := range b.workflow.Edges {
		_ = b.canvas.AddEdge(edge) // Ignore error in restore - best effort
	}
}

// getNextAutoPosition calculates the next auto-position for a new node
func (b *WorkflowBuilder) getNextAutoPosition() Position {
	// Simple vertical stacking for now
	y := 2
	for _, canvasNode := range b.canvas.nodes {
		if canvasNode.position.Y >= y {
			y = canvasNode.position.Y + canvasNode.height + 1
		}
	}
	return Position{X: 5, Y: y}
}

func (b *WorkflowBuilder) updateKeyStates() {
	switch b.mode {
	case "normal":
		b.keyEnabled = map[string]bool{
			"a": true,
			"d": true,
			"e": true,
			"c": true,
		}
	default: // edit, palette, or other modes
		b.keyEnabled = map[string]bool{
			"a": false,
			"d": false,
			"e": false,
			"c": false,
		}
	}
}

// ShowPropertyPanel shows the property panel for a node
func (b *WorkflowBuilder) ShowPropertyPanel(nodeID string) error {
	// Find the node
	var node workflow.Node
	for _, n := range b.workflow.Nodes {
		if n.GetID() == nodeID {
			node = n
			break
		}
	}

	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Build property fields based on node type
	b.propertyPanel.node = node
	b.propertyPanel.fields = b.buildPropertyFields(node)
	b.propertyPanel.visible = true
	b.propertyPanel.editIndex = 0
	b.propertyPanel.validationMessage = ""

	return nil
}

// buildPropertyFields creates property fields for a node
func (b *WorkflowBuilder) buildPropertyFields(node workflow.Node) []propertyField {
	fields := []propertyField{
		{
			label:     "ID",
			value:     node.GetID(),
			required:  true,
			valid:     true,
			fieldType: "text",
		},
	}

	switch n := node.(type) {
	case *workflow.ConditionNode:
		fields = append(fields, propertyField{
			label:     "Condition Expression",
			value:     n.Condition,
			required:  true,
			valid:     true,
			fieldType: "condition",
			validationFn: func(expr string) error {
				return workflow.ValidateExpressionSyntax(expr)
			},
		})

	case *workflow.TransformNode:
		fields = append(fields,
			propertyField{
				label:     "Input Variable",
				value:     n.InputVariable,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
			propertyField{
				label:     "Expression",
				value:     n.Expression,
				required:  true,
				valid:     true,
				fieldType: "expression",
				validationFn: func(expr string) error {
					// Detect expression type and validate accordingly
					if len(expr) > 0 && expr[0] == '$' {
						return workflow.ValidateJSONPathSyntax(expr)
					}
					if strings.Contains(expr, "${") {
						return workflow.ValidateTemplateSyntax(expr)
					}
					return workflow.ValidateExpressionSyntax(expr)
				},
			},
			propertyField{
				label:     "Output Variable",
				value:     n.OutputVariable,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
		)

	case *workflow.MCPToolNode:
		fields = append(fields,
			propertyField{
				label:     "Server ID",
				value:     n.ServerID,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
			propertyField{
				label:     "Tool Name",
				value:     n.ToolName,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
			propertyField{
				label:     "Output Variable",
				value:     n.OutputVariable,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
		)

	case *workflow.LoopNode:
		// Format body nodes for display
		bodyStr := strings.Join(n.Body, ", ")

		fields = append(fields,
			propertyField{
				label:     "Collection",
				value:     n.Collection,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
			propertyField{
				label:     "Item Variable",
				value:     n.ItemVariable,
				required:  true,
				valid:     true,
				fieldType: "text",
			},
			propertyField{
				label:     "Body Nodes",
				value:     bodyStr,
				required:  true,
				valid:     true,
				fieldType: "node_list",
			},
			propertyField{
				label:     "Break Condition",
				value:     n.BreakCondition,
				required:  false,
				valid:     true,
				fieldType: "condition",
				validationFn: func(expr string) error {
					if expr == "" {
						return nil // Break condition is optional
					}
					return workflow.ValidateExpressionSyntax(expr)
				},
			},
		)

	case *workflow.ParallelNode:
		// Format branches for display
		branchesStr := ""
		for i, branch := range n.Branches {
			if i > 0 {
				branchesStr += "; "
			}
			branchesStr += fmt.Sprintf("[%s]", strings.Join(branch, ","))
		}

		fields = append(fields,
			propertyField{
				label:     "Branches",
				value:     branchesStr,
				required:  true,
				valid:     true,
				fieldType: "branches",
			},
			propertyField{
				label:     "Merge Strategy",
				value:     n.MergeStrategy,
				required:  true,
				valid:     true,
				fieldType: "select",
				validationFn: func(strategy string) error {
					if strategy != "wait_all" && strategy != "wait_any" && strategy != "wait_first" {
						return fmt.Errorf("invalid merge strategy: %s (use wait_all, wait_any, or wait_first)", strategy)
					}
					return nil
				},
			},
		)
	}

	return fields
}

// UpdatePropertyField updates a property field value
func (b *WorkflowBuilder) UpdatePropertyField(index int, value string) error {
	if !b.propertyPanel.visible {
		return errors.New("property panel not visible")
	}

	if index < 0 || index >= len(b.propertyPanel.fields) {
		return fmt.Errorf("invalid field index: %d", index)
	}

	field := &b.propertyPanel.fields[index]
	field.value = value

	// Validate if validation function exists
	if field.validationFn != nil {
		if err := field.validationFn(value); err != nil {
			field.valid = false
			b.propertyPanel.validationMessage = fmt.Sprintf("Validation error: %s", err.Error())
			return err
		}
		field.valid = true
		b.propertyPanel.validationMessage = ""
	}

	// Apply changes to the node
	return b.applyPropertyChanges()
}

// applyPropertyChanges applies property field changes to the actual node
func (b *WorkflowBuilder) applyPropertyChanges() error {
	node := b.propertyPanel.node
	fields := b.propertyPanel.fields

	switch n := node.(type) {
	case *workflow.ConditionNode:
		for _, field := range fields {
			if field.label == "Condition Expression" {
				n.Condition = field.value
			}
		}

	case *workflow.TransformNode:
		for _, field := range fields {
			switch field.label {
			case "Input Variable":
				n.InputVariable = field.value
			case "Expression":
				n.Expression = field.value
			case "Output Variable":
				n.OutputVariable = field.value
			}
		}

	case *workflow.MCPToolNode:
		for _, field := range fields {
			switch field.label {
			case "Server ID":
				n.ServerID = field.value
			case "Tool Name":
				n.ToolName = field.value
			case "Output Variable":
				n.OutputVariable = field.value
			}
		}

	case *workflow.LoopNode:
		for _, field := range fields {
			switch field.label {
			case "Collection":
				n.Collection = field.value
			case "Item Variable":
				n.ItemVariable = field.value
			case "Body Nodes":
				// Parse comma-separated node IDs
				if field.value != "" {
					nodeIDs := strings.Split(field.value, ",")
					cleanedIDs := make([]string, 0, len(nodeIDs))
					for _, id := range nodeIDs {
						id = strings.TrimSpace(id)
						if id != "" {
							cleanedIDs = append(cleanedIDs, id)
						}
					}
					n.Body = cleanedIDs
				} else {
					n.Body = []string{}
				}
			case "Break Condition":
				n.BreakCondition = field.value
			}
		}

	case *workflow.ParallelNode:
		for _, field := range fields {
			switch field.label {
			case "Branches":
				// Parse branches string format: [node1,node2];[node3,node4]
				if field.value != "" {
					branches := [][]string{}
					branchGroups := strings.Split(field.value, ";")
					for _, group := range branchGroups {
						group = strings.TrimSpace(group)
						if group == "" {
							continue
						}
						// Remove brackets
						group = strings.Trim(group, "[]")
						nodeIDs := strings.Split(group, ",")
						// Trim whitespace from each node ID
						cleanedIDs := make([]string, 0, len(nodeIDs))
						for _, id := range nodeIDs {
							id = strings.TrimSpace(id)
							if id != "" {
								cleanedIDs = append(cleanedIDs, id)
							}
						}
						if len(cleanedIDs) > 0 {
							branches = append(branches, cleanedIDs)
						}
					}
					n.Branches = branches
				}
			case "Merge Strategy":
				n.MergeStrategy = field.value
			}
		}
	}

	b.modified = true
	b.validateWorkflow()
	return nil
}

// GetPropertyPanel returns the property panel
func (b *WorkflowBuilder) GetPropertyPanel() *PropertyPanel {
	return b.propertyPanel
}

// GetVariableList returns a list of variable names in the workflow
func (b *WorkflowBuilder) GetVariableList() []string {
	vars := make([]string, 0, len(b.workflow.Variables))
	for _, v := range b.workflow.Variables {
		vars = append(vars, v.Name)
	}
	return vars
}

// GetEdgeLabel returns the label for an edge (e.g., "true"/"false" for condition edges)
func (b *WorkflowBuilder) GetEdgeLabel(edge *workflow.Edge) string {
	if edge.Condition != "" {
		return edge.Condition
	}
	return ""
}

// GetEdgeStyle returns style information for an edge
func (b *WorkflowBuilder) GetEdgeStyle(edge *workflow.Edge) string {
	// Check if this edge is from a condition node
	for _, node := range b.workflow.Nodes {
		if node.GetID() == edge.FromNodeID && node.Type() == "condition" {
			switch edge.Condition {
			case "true":
				return "solid"
			case "false":
				return "dashed"
			}
		}
	}
	return "solid"
}

// CreateConditionalEdge creates an edge with a condition label
func (b *WorkflowBuilder) CreateConditionalEdge(fromID, toID, condition string) error {
	// Verify source is a condition node
	var isConditionNode bool
	for _, node := range b.workflow.Nodes {
		if node.GetID() == fromID && node.Type() == "condition" {
			isConditionNode = true
			break
		}
	}

	if !isConditionNode {
		return fmt.Errorf("source node %s is not a condition node", fromID)
	}

	// Verify condition value
	if condition != "true" && condition != "false" {
		return fmt.Errorf("condition must be 'true' or 'false', got: %s", condition)
	}

	// Check if this condition already has an edge
	for _, edge := range b.workflow.Edges {
		if edge.FromNodeID == fromID && edge.Condition == condition {
			return fmt.Errorf("condition node already has a %s edge", condition)
		}
	}

	// Push undo snapshot
	canvasPositions := b.getCanvasPositions()
	if err := b.undoStack.Push(b.workflow, canvasPositions); err != nil {
		return err
	}

	edge := &workflow.Edge{
		FromNodeID: fromID,
		ToNodeID:   toID,
		Condition:  condition,
	}

	if err := b.workflow.AddEdge(edge); err != nil {
		return err
	}

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true

	return nil
}

// Canvas methods

// GetNodeCount returns the number of nodes on the canvas
func (c *Canvas) GetNodeCount() int {
	return len(c.nodes)
}

// GetEdgeCount returns the number of edges on the canvas
func (c *Canvas) GetEdgeCount() int {
	return len(c.edges)
}

// NodePalette methods are in node_palette.go

// PropertyPanel methods moved to property_panel.go

// RenderPropertyPanel returns a formatted string for displaying the property panel
func (p *PropertyPanel) RenderPropertyPanel() string {
	if !p.visible || p.node == nil {
		return ""
	}

	var sb strings.Builder
	titleCaser := cases.Title(language.English)
	sb.WriteString(fmt.Sprintf("=== %s Node Properties ===\n", titleCaser.String(p.node.Type())))
	sb.WriteString("\n")

	for i, field := range p.fields {
		marker := " "
		if i == p.editIndex {
			marker = ">"
		}

		validMarker := "✓"
		if !field.valid {
			validMarker = "✗"
		}

		sb.WriteString(fmt.Sprintf("%s [%s] %s: %s\n", marker, validMarker, field.label, field.value))

		// Show field type hint for special fields
		if field.fieldType == "condition" {
			if field.label == "Break Condition" {
				sb.WriteString("     (Optional: Boolean expression to break loop early)\n")
			} else {
				sb.WriteString("     (Boolean expression, e.g., price > 100)\n")
			}
		} else if field.fieldType == "expression" {
			sb.WriteString("     (JSONPath: $.field, Template: ${var}, or expression)\n")
		} else if field.fieldType == "branches" {
			sb.WriteString("     (Format: [node1,node2];[node3,node4] for parallel branches)\n")
		} else if field.fieldType == "select" && field.label == "Merge Strategy" {
			sb.WriteString("     (Options: wait_all, wait_any, wait_first)\n")
		} else if field.fieldType == "node_list" {
			sb.WriteString("     (Comma-separated node IDs to execute in loop)\n")
		}
	}

	if p.validationMessage != "" {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("⚠ %s\n", p.validationMessage))
	}

	sb.WriteString("\nKeys: [↑↓] Navigate [Enter] Edit [Esc] Close\n")

	return sb.String()
}

// Render renders the workflow builder to screen
// This orchestrates rendering of all components: canvas, panels, palette
func (b *WorkflowBuilder) Render(screen interface{}, screenWidth, screenHeight int) error {
	if b == nil {
		return fmt.Errorf("workflow builder not initialized")
	}

	// Type assert to screen interface
	type Screen interface {
		SetCell(cellX, cellY int, cell interface{})
		Size() (int, int)
	}

	_, ok := screen.(Screen)
	if !ok {
		return fmt.Errorf("invalid screen type")
	}

	// Layout configuration
	// Main canvas takes most of the screen
	// Panels appear on the right side or overlay the canvas

	// Canvas area: full screen or left side if panels visible
	canvasWidth := screenWidth
	canvasHeight := screenHeight

	// Check if any right-side panels are visible
	rightPanelVisible := b.propertyPanel.IsVisible() || b.validationPanel.visible

	if rightPanelVisible {
		// Split screen: canvas on left, panels on right
		canvasWidth = (screenWidth * 2) / 3 // 2/3 for canvas
		// Remaining 1/3 for panels
	}

	// Render canvas (main workflow view)
	if b.canvas != nil {
		b.canvas.Width = canvasWidth
		b.canvas.Height = canvasHeight
		if err := b.canvas.RenderToScreen(screen); err != nil {
			return fmt.Errorf("failed to render canvas: %w", err)
		}
	}

	// Render right-side panels
	panelX := canvasWidth
	panelWidth := screenWidth - canvasWidth
	panelY := 0

	// Property panel takes top half of right side
	if b.propertyPanel.IsVisible() {
		panelHeight := screenHeight / 2
		if err := b.propertyPanel.Render(screen, panelX, panelY, panelWidth, panelHeight); err != nil {
			return fmt.Errorf("failed to render property panel: %w", err)
		}
		panelY += panelHeight
	}

	// Validation panel takes remaining space on right side
	if b.validationPanel.visible {
		panelHeight := screenHeight - panelY
		if err := b.validationPanel.Render(screen, panelX, panelY, panelWidth, panelHeight); err != nil {
			return fmt.Errorf("failed to render validation panel: %w", err)
		}
	}

	// Overlay panels (centered on screen)
	if b.mode == "palette" && b.palette.IsVisible() {
		// Palette overlay: centered, 60% width, 70% height
		overlayWidth := (screenWidth * 3) / 5
		overlayHeight := (screenHeight * 7) / 10
		overlayX := (screenWidth - overlayWidth) / 2
		overlayY := (screenHeight - overlayHeight) / 2

		if err := b.palette.Render(screen, overlayX, overlayY, overlayWidth, overlayHeight); err != nil {
			return fmt.Errorf("failed to render palette: %w", err)
		}
	}

	// Help panel overlay: full screen (if HelpPanel has Render method)
	// TODO: Implement HelpPanel.Render() method when help panel is ready
	_ = b.helpPanel // Prevent unused warning

	return nil
}

// ApplyTemplate applies a workflow template by name
// This implements T077 from Phase 9: Workflow Templates
func (b *WorkflowBuilder) ApplyTemplate(templateName string) error {
	// Step 1: Get template function from registry
	createFn, exists := WorkflowTemplates[templateName]
	if !exists {
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Step 2: Confirm if current workflow has unsaved changes
	// Note: In real TUI, would show confirmation dialog if b.modified == true
	// For now, proceed without confirmation (tests handle this)

	// Step 3: Call template function to generate workflow
	templateWf := createFn()
	if templateWf == nil {
		return fmt.Errorf("template creation failed: %s", templateName)
	}

	// Step 4: Replace current workflow
	b.workflow = templateWf

	// Step 5: Load into canvas using LoadWorkflow
	b.canvas.nodes = make(map[string]*canvasNode)
	b.canvas.edges = make([]*canvasEdge, 0)

	// Step 6: Run auto-layout
	b.layoutNodes()

	// Step 7: Clear modified flag (template is clean state)
	b.modified = false

	// Step 8: Clear undo stack (new workflow, no history)
	b.undoStack = NewUndoStack(100)

	// Step 9: Run validation
	b.validateWorkflow()

	return nil
}

// HandleResize handles terminal resize events
// This implements T084 from Phase 11: Polish
func (b *WorkflowBuilder) HandleResize(newWidth, newHeight int) {
	// Step 1: Update canvas dimensions
	b.canvas.Width = newWidth
	b.canvas.Height = newHeight

	// Step 2: Adjust viewport if needed (ensure selected node stays visible)
	if b.selectedNodeID != "" {
		if selectedNode, exists := b.canvas.nodes[b.selectedNodeID]; exists {
			// Check if selected node is out of bounds
			nodePos := selectedNode.position

			// Convert to terminal coordinates
			termPos := LogicalToTerminal(
				nodePos,
				b.canvas.ViewportX,
				b.canvas.ViewportY,
				b.canvas.ZoomLevel,
			)

			// If node is out of viewport, re-center
			if termPos.X < 0 || termPos.X >= newWidth ||
				termPos.Y < 0 || termPos.Y >= newHeight {
				// Center viewport on selected node
				b.canvas.ViewportX = nodePos.X - newWidth/2
				b.canvas.ViewportY = nodePos.Y - newHeight/2

				// Ensure viewport doesn't go negative
				if b.canvas.ViewportX < 0 {
					b.canvas.ViewportX = 0
				}
				if b.canvas.ViewportY < 0 {
					b.canvas.ViewportY = 0
				}
			}
		}
	}

	// Step 3: Trigger re-render (would happen automatically in real TUI)
	// In tests, we just ensure state is consistent
}

// handleNormalMode processes keyboard shortcuts in normal mode
// This implements T080 from Phase 10: Keyboard Handling
func (b *WorkflowBuilder) handleNormalMode(key string) error {
	switch key {
	// Node operations
	case "a":
		return b.AddNode()
	case "d":
		if b.selectedNodeID != "" {
			return b.DeleteNode(b.selectedNodeID)
		}
		return fmt.Errorf("no node selected")
	case "c":
		// Enter edge creation mode
		if b.selectedNodeID != "" {
			b.edgeCreationMode = true
			b.edgeSourceID = b.selectedNodeID
			return nil
		}
		return fmt.Errorf("no node selected")
	case "y":
		// Yank (copy) node - future feature
		return fmt.Errorf("yank not yet implemented")
	case "p":
		// Paste node - future feature
		return fmt.Errorf("paste not yet implemented")

	// Workflow operations
	case "s":
		return b.SaveWorkflow()
	case "v":
		b.validateWorkflow()
		return nil
	case "u":
		return b.Undo()
	case "Ctrl+r":
		return b.Redo()
	case "t":
		// Show templates - in real TUI would show modal
		return fmt.Errorf("template selection not yet implemented in TUI")

	// View operations
	case "Enter":
		if b.selectedNodeID != "" {
			return b.EditNodeProperties(b.selectedNodeID)
		}
		return fmt.Errorf("no node selected")
	case "0":
		b.canvas.ResetView()
		return nil
	case "f":
		b.canvas.FitAll()
		return nil

	// Navigation (canvas pan)
	case "Shift+Up":
		b.canvas.Pan(0, -10)
		return nil
	case "Shift+Down":
		b.canvas.Pan(0, 10)
		return nil
	case "Shift+Left":
		b.canvas.Pan(-10, 0)
		return nil
	case "Shift+Right":
		b.canvas.Pan(10, 0)
		return nil

	// Zoom
	case "+", "=":
		newZoom := b.canvas.ZoomLevel + 0.1
		if newZoom > 2.0 {
			newZoom = 2.0
		}
		return b.canvas.Zoom(newZoom)
	case "-":
		newZoom := b.canvas.ZoomLevel - 0.1
		if newZoom < 0.5 {
			newZoom = 0.5
		}
		return b.canvas.Zoom(newZoom)

	// Node movement (h/j/k/l)
	case "h":
		if b.selectedNodeID != "" {
			if node, exists := b.canvas.nodes[b.selectedNodeID]; exists {
				return b.canvas.MoveNode(b.selectedNodeID, Position{X: node.position.X - 1, Y: node.position.Y})
			}
		}
		return fmt.Errorf("no node selected")
	case "j":
		if b.selectedNodeID != "" {
			if node, exists := b.canvas.nodes[b.selectedNodeID]; exists {
				return b.canvas.MoveNode(b.selectedNodeID, Position{X: node.position.X, Y: node.position.Y + 1})
			}
		}
		return fmt.Errorf("no node selected")
	case "k":
		if b.selectedNodeID != "" {
			if node, exists := b.canvas.nodes[b.selectedNodeID]; exists {
				return b.canvas.MoveNode(b.selectedNodeID, Position{X: node.position.X, Y: node.position.Y - 1})
			}
		}
		return fmt.Errorf("no node selected")
	case "l":
		if b.selectedNodeID != "" {
			if node, exists := b.canvas.nodes[b.selectedNodeID]; exists {
				return b.canvas.MoveNode(b.selectedNodeID, Position{X: node.position.X + 1, Y: node.position.Y})
			}
		}
		return fmt.Errorf("no node selected")

	default:
		return fmt.Errorf("unrecognized key in normal mode: %s", key)
	}
}

// handleEditMode processes keyboard shortcuts in edit mode
// This implements T081 from Phase 10: Keyboard Handling
func (b *WorkflowBuilder) handleEditMode(key string) error {
	switch key {
	// Field navigation
	case "Tab", "Down":
		b.propertyPanel.NextField()
		return nil
	case "Shift+Tab", "Up":
		b.propertyPanel.PrevField()
		return nil

	// Edit operations
	case "Enter":
		// Would start editing the field in real TUI
		return nil
	case "Ctrl+s":
		return b.SavePropertyChanges()
	case "Ctrl+r":
		// Reset current field to default
		return fmt.Errorf("field reset not yet implemented")

	default:
		return fmt.Errorf("unrecognized key in edit mode: %s", key)
	}
}

// handlePaletteMode processes keyboard shortcuts in palette mode
// This implements T082 from Phase 10: Keyboard Handling
func (b *WorkflowBuilder) handlePaletteMode(key string) error {
	switch key {
	// Navigation
	case "Down", "j":
		b.palette.Next()
		return nil
	case "Up", "k":
		b.palette.Previous()
		return nil

	// Selection
	case "Enter":
		// Create node with selected type
		node, err := b.palette.CreateNode()
		if err != nil {
			return err
		}
		// Use AddNodeWithType to complete the operation
		b.palette.Hide()
		b.mode = "normal"
		b.updateKeyStates()

		// Add to canvas and workflow
		pos := b.getNextAutoPosition()
		if err := b.canvas.AddNode(node, pos); err != nil {
			return err
		}
		if err := b.workflow.AddNode(node); err != nil {
			return err
		}
		b.modified = true
		b.validateWorkflow()
		return nil

	// Filtering (single character keys)
	default:
		if len(key) == 1 {
			// Append to filter
			currentFilter := b.palette.filterText
			b.palette.Filter(currentFilter + key)
			return nil
		}
		return fmt.Errorf("unrecognized key in palette mode: %s", key)
	}
}

// handleHelpMode processes keyboard shortcuts in help mode
func (b *WorkflowBuilder) handleHelpMode(key string) error {
	switch key {
	case "Down", "j":
		// Scroll help down (future feature)
		return nil
	case "Up", "k":
		// Scroll help up (future feature)
		return nil
	default:
		return fmt.Errorf("unrecognized key in help mode: %s", key)
	}
}

// HelpPanel methods

// HelpPanel methods are defined in help_panel.go
