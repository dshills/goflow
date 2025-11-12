package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
)

// Position represents a 2D coordinate on the canvas
type Position struct {
	X int
	Y int
}

// WorkflowBuilder provides a visual workflow editor
type WorkflowBuilder struct {
	workflow         *workflow.Workflow
	canvas           *Canvas
	palette          *NodePalette
	propertyPanel    *PropertyPanel
	helpPanel        *HelpPanel
	selectedNodeID   string
	mode             string // "view" or "edit"
	edgeCreationMode bool
	edgeSourceID     string
	modified         bool
	validationStatus *ValidationStatus
	undoStack        []workflowSnapshot
	redoStack        []workflowSnapshot
	repository       workflow.WorkflowRepository
	keyEnabled       map[string]bool
}

// workflowSnapshot stores workflow state for undo/redo
type workflowSnapshot struct {
	nodes []workflow.Node
	edges []*workflow.Edge
}

// Canvas represents the drawing surface for the workflow graph
type Canvas struct {
	Width      int
	Height     int
	nodes      map[string]canvasNode
	edges      []canvasEdge
	selectedID string
}

// canvasNode represents a node positioned on the canvas
type canvasNode struct {
	node     workflow.Node
	position Position
	width    int
	height   int
}

// canvasEdge represents an edge drawn on the canvas
type canvasEdge struct {
	edge    *workflow.Edge
	fromPos Position
	toPos   Position
}

// NodePalette represents the node selection palette
type NodePalette struct {
	nodeTypes     []string
	selectedIndex int
	visible       bool
}

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
	label        string
	value        string
	required     bool
	valid        bool
	fieldType    string // "text", "expression", "condition", "jsonpath", "template"
	validationFn func(string) error
}

// HelpPanel represents the help display
type HelpPanel struct {
	visible bool
}

// HelpKeyBinding represents a keyboard shortcut for help display
type HelpKeyBinding struct {
	Key         string
	Description string
}

// ValidationStatus represents validation state
type ValidationStatus struct {
	IsValid bool
	Errors  []ValidationError
}

// ValidationError represents a single validation error
type ValidationError struct {
	Message string
	NodeID  string
}

// NewWorkflowBuilder creates a new workflow builder
// Returns an error if wf is nil
func NewWorkflowBuilder(wf *workflow.Workflow) (*WorkflowBuilder, error) {
	if wf == nil {
		return nil, errors.New("workflow cannot be nil")
	}

	builder := &WorkflowBuilder{
		workflow: wf,
		canvas: &Canvas{
			Width:  80,
			Height: 24,
			nodes:  make(map[string]canvasNode),
			edges:  make([]canvasEdge, 0),
		},
		palette: &NodePalette{
			nodeTypes: []string{
				"MCP Tool",
				"Transform",
				"Condition",
				"Loop",
				"Parallel",
			},
			selectedIndex: 0,
			visible:       false,
		},
		propertyPanel: &PropertyPanel{
			visible: false,
		},
		helpPanel: &HelpPanel{
			visible: false,
		},
		mode:             "view",
		validationStatus: &ValidationStatus{IsValid: true, Errors: []ValidationError{}},
		undoStack:        make([]workflowSnapshot, 0),
		redoStack:        make([]workflowSnapshot, 0),
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
func (b *WorkflowBuilder) HandleKey(key string) error {
	// Handle mode switching
	if key == "i" && b.mode == "view" {
		b.SetMode("edit")
		return nil
	}
	if key == "Esc" && b.mode == "edit" {
		b.SetMode("view")
		b.palette.visible = false
		b.edgeCreationMode = false
		return nil
	}

	// Handle palette toggle
	if key == "a" && b.mode == "edit" {
		b.palette.visible = !b.palette.visible
		return nil
	}

	// Handle edge creation
	if key == "e" && b.mode == "edit" {
		if !b.edgeCreationMode {
			b.edgeCreationMode = true
			b.edgeSourceID = b.selectedNodeID
		} else {
			// Create edge
			if b.selectedNodeID != "" && b.edgeSourceID != "" {
				err := b.CreateEdge(b.edgeSourceID, b.selectedNodeID)
				b.edgeCreationMode = false
				b.edgeSourceID = ""
				return err
			}
		}
		return nil
	}

	// Handle node navigation
	if key == "Tab" {
		return b.selectNextNode()
	}
	if key == "Shift+Tab" {
		return b.selectPreviousNode()
	}

	// Handle arrow keys for spatial navigation
	// Arrow keys also use the same navigation state as Tab
	if key == "Right" {
		// Arrow key movement enables subsequent forward navigation
		defer func() { b.keyEnabled["forward"] = true }()
		return b.selectNextNode()
	}
	if key == "Left" {
		// Arrow key movement enables subsequent backward navigation
		defer func() { b.keyEnabled["backward"] = true }()
		return b.selectPreviousNode()
	}

	// Handle undo/redo
	if key == "u" && b.mode == "edit" {
		return b.Undo()
	}
	if key == "Ctrl+r" && b.mode == "edit" {
		return b.Redo()
	}

	// Handle save
	if key == "Ctrl+s" {
		return b.SaveWorkflow()
	}

	// Handle help
	if key == "?" {
		b.helpPanel.visible = !b.helpPanel.visible
		return nil
	}

	// Return error for unrecognized keys
	return fmt.Errorf("unrecognized key: %s", key)
}

// GetValidationStatus returns the current validation status
func (b *WorkflowBuilder) GetValidationStatus() *ValidationStatus {
	return b.validationStatus
}

// SaveWorkflow saves the workflow to storage
func (b *WorkflowBuilder) SaveWorkflow() error {
	// Validate before saving
	if err := b.workflow.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid workflow: %w", err)
	}

	// Save using repository if available
	if b.repository != nil {
		if err := b.repository.Save(b.workflow); err != nil {
			return fmt.Errorf("failed to save workflow: %w", err)
		}
	}

	b.modified = false
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

// AddNodeToCanvas adds a node to the canvas
func (b *WorkflowBuilder) AddNodeToCanvas(node workflow.Node) error {
	b.pushUndo()

	if err := b.workflow.AddNode(node); err != nil {
		return err
	}

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true
	b.redoStack = make([]workflowSnapshot, 0) // Clear redo stack

	return nil
}

// Undo reverts the last change
func (b *WorkflowBuilder) Undo() error {
	if len(b.undoStack) == 0 {
		return errors.New("nothing to undo")
	}

	// Save current state to redo stack
	b.pushRedo()

	// Pop from undo stack
	snapshot := b.undoStack[len(b.undoStack)-1]
	b.undoStack = b.undoStack[:len(b.undoStack)-1]

	// Restore state
	b.workflow.Nodes = snapshot.nodes
	b.workflow.Edges = snapshot.edges

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true

	return nil
}

// Redo reapplies the last undone change
func (b *WorkflowBuilder) Redo() error {
	if len(b.redoStack) == 0 {
		return errors.New("nothing to redo")
	}

	// Save current state to undo stack
	b.pushUndo()

	// Pop from redo stack
	snapshot := b.redoStack[len(b.redoStack)-1]
	b.redoStack = b.redoStack[:len(b.redoStack)-1]

	// Restore state
	b.workflow.Nodes = snapshot.nodes
	b.workflow.Edges = snapshot.edges

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true

	return nil
}

// CanUndo returns whether undo is available
func (b *WorkflowBuilder) CanUndo() bool {
	return len(b.undoStack) > 0
}

// CanRedo returns whether redo is available
func (b *WorkflowBuilder) CanRedo() bool {
	return len(b.redoStack) > 0
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
	b.pushUndo()

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
	b.canvas.nodes[nodeID] = canvasNode{
		node:     node,
		position: pos,
		width:    20,
		height:   3,
	}

	b.validateWorkflow()
	b.modified = true
	b.redoStack = make([]workflowSnapshot, 0)

	return nil
}

// CreateEdge creates an edge between two nodes
func (b *WorkflowBuilder) CreateEdge(fromID, toID string) error {
	// Check if nodes exist
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

	b.pushUndo()

	edge := &workflow.Edge{
		FromNodeID: fromID,
		ToNodeID:   toID,
	}

	if err := b.workflow.AddEdge(edge); err != nil {
		return err
	}

	b.layoutNodes()
	b.validateWorkflow()
	b.modified = true
	b.redoStack = make([]workflowSnapshot, 0)

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
		b.canvas.nodes[nodeID] = canvasNode{
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
	b.canvas.edges = make([]canvasEdge, 0)
	for _, edge := range b.workflow.Edges {
		fromNode, fromExists := b.canvas.nodes[edge.FromNodeID]
		toNode, toExists := b.canvas.nodes[edge.ToNodeID]

		if fromExists && toExists {
			b.canvas.edges = append(b.canvas.edges, canvasEdge{
				edge:    edge,
				fromPos: fromNode.position,
				toPos:   toNode.position,
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

func (b *WorkflowBuilder) pushUndo() {
	// Copy current state
	nodes := make([]workflow.Node, len(b.workflow.Nodes))
	copy(nodes, b.workflow.Nodes)

	edges := make([]*workflow.Edge, len(b.workflow.Edges))
	copy(edges, b.workflow.Edges)

	b.undoStack = append(b.undoStack, workflowSnapshot{
		nodes: nodes,
		edges: edges,
	})

	// Limit stack size
	if len(b.undoStack) > 50 {
		b.undoStack = b.undoStack[1:]
	}
}

func (b *WorkflowBuilder) pushRedo() {
	// Copy current state
	nodes := make([]workflow.Node, len(b.workflow.Nodes))
	copy(nodes, b.workflow.Nodes)

	edges := make([]*workflow.Edge, len(b.workflow.Edges))
	copy(edges, b.workflow.Edges)

	b.redoStack = append(b.redoStack, workflowSnapshot{
		nodes: nodes,
		edges: edges,
	})

	// Limit stack size
	if len(b.redoStack) > 50 {
		b.redoStack = b.redoStack[1:]
	}
}

func (b *WorkflowBuilder) updateKeyStates() {
	if b.mode == "view" {
		b.keyEnabled = map[string]bool{
			"a": false,
			"d": false,
			"e": true,
			"c": false,
		}
	} else {
		b.keyEnabled = map[string]bool{
			"a": true,
			"d": true,
			"e": true,
			"c": true,
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
			if edge.Condition == "true" {
				return "solid"
			} else if edge.Condition == "false" {
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

	b.pushUndo()

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
	b.redoStack = make([]workflowSnapshot, 0)

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

// NodePalette methods

// GetNodeTypes returns the list of available node types
func (p *NodePalette) GetNodeTypes() []string {
	return p.nodeTypes
}

// GetSelectedIndex returns the currently selected index
func (p *NodePalette) GetSelectedIndex() int {
	return p.selectedIndex
}

// HandleKey processes keyboard input for the palette
func (p *NodePalette) HandleKey(key string) error {
	switch key {
	case "j":
		p.selectedIndex = (p.selectedIndex + 1) % len(p.nodeTypes)
	case "k":
		p.selectedIndex = p.selectedIndex - 1
		if p.selectedIndex < 0 {
			p.selectedIndex = len(p.nodeTypes) - 1
		}
	case "g":
		p.selectedIndex = 0
	case "G":
		p.selectedIndex = len(p.nodeTypes) - 1
	}
	return nil
}

// GetSelectedNodeType returns the currently selected node type
func (p *NodePalette) GetSelectedNodeType() string {
	if p.selectedIndex >= 0 && p.selectedIndex < len(p.nodeTypes) {
		return p.nodeTypes[p.selectedIndex]
	}
	return ""
}

// PropertyPanel methods

// IsVisible returns whether the property panel is visible
func (p *PropertyPanel) IsVisible() bool {
	return p.visible
}

// GetFields returns the property fields
func (p *PropertyPanel) GetFields() []propertyField {
	return p.fields
}

// GetEditIndex returns the currently edited field index
func (p *PropertyPanel) GetEditIndex() int {
	return p.editIndex
}

// GetValidationMessage returns the current validation message
func (p *PropertyPanel) GetValidationMessage() string {
	return p.validationMessage
}

// GetNodeType returns the type of node being edited
func (p *PropertyPanel) GetNodeType() string {
	if p.node == nil {
		return ""
	}
	return p.node.Type()
}

// RenderPropertyPanel returns a formatted string for displaying the property panel
func (p *PropertyPanel) RenderPropertyPanel() string {
	if !p.visible || p.node == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s Node Properties ===\n", strings.Title(p.node.Type())))
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

// HelpPanel methods

// IsVisible returns whether the help panel is visible
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

// GetKeyBindings returns the key bindings to display
func (h *HelpPanel) GetKeyBindings() []HelpKeyBinding {
	return []HelpKeyBinding{
		{Key: "i", Description: "Enter edit mode"},
		{Key: "Tab", Description: "Next node"},
		{Key: "?", Description: "Toggle help"},
		{Key: "q", Description: "Quit"},
		{Key: "a", Description: "Add node"},
		{Key: "d", Description: "Delete node"},
		{Key: "e", Description: "Edit node"},
		{Key: "c", Description: "Connect nodes"},
		{Key: "u", Description: "Undo"},
		{Key: "Ctrl+r", Description: "Redo"},
		{Key: "Esc", Description: "Exit edit mode"},
	}
}
