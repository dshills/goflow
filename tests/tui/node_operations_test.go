package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// MockTUIInterface simulates TUI keyboard and display interactions
// This will be replaced with actual goterm testing facilities when implemented
type MockTUIInterface struct {
	workflow       *workflow.Workflow
	selectedNode   workflow.Node
	paletteOpen    bool
	paletteIndex   int
	propertyEditor *PropertyEditor
	nodeCount      int
	keystrokes     []string
}

// PropertyEditor simulates the property panel for node configuration
type PropertyEditor struct {
	nodeType         string
	fields           map[string]string
	isOpen           bool
	cursorPos        int
	validationErrors []string
}

// NewMockTUI creates a new mock TUI for testing
func NewMockTUI(wf *workflow.Workflow) *MockTUIInterface {
	return &MockTUIInterface{
		workflow:   wf,
		keystrokes: make([]string, 0),
		propertyEditor: &PropertyEditor{
			fields: make(map[string]string),
		},
	}
}

// SendKey simulates keyboard input
func (m *MockTUIInterface) SendKey(key string) error {
	m.keystrokes = append(m.keystrokes, key)

	switch key {
	case "a":
		return m.openNodePalette()
	case "Esc":
		return m.cancelOperation()
	case "Enter":
		return m.confirmSelection()
	case "j", "Down":
		return m.navigateDown()
	case "k", "Up":
		return m.navigateUp()
	default:
		return m.handleTextInput(key)
	}
}

// openNodePalette opens the node type selection palette
func (m *MockTUIInterface) openNodePalette() error {
	m.paletteOpen = true
	m.paletteIndex = 0
	return nil
}

// cancelOperation cancels current operation (palette or property editor)
func (m *MockTUIInterface) cancelOperation() error {
	if m.propertyEditor.isOpen {
		m.propertyEditor.isOpen = false
		m.propertyEditor.fields = make(map[string]string)
		return nil
	}
	if m.paletteOpen {
		m.paletteOpen = false
		m.paletteIndex = 0
		return nil
	}
	return nil
}

// confirmSelection confirms palette selection or property editor field
func (m *MockTUIInterface) confirmSelection() error {
	if m.paletteOpen {
		// Select node type from palette
		nodeTypes := []string{"start", "end", "mcp_tool", "transform", "condition", "parallel", "loop"}
		if m.paletteIndex >= 0 && m.paletteIndex < len(nodeTypes) {
			m.paletteOpen = false
			return m.openPropertyEditor(nodeTypes[m.paletteIndex])
		}
	}
	if m.propertyEditor.isOpen {
		// Complete property editing and add node
		return m.addNodeFromProperties()
	}
	return nil
}

// navigateDown moves cursor down in palette or property editor
func (m *MockTUIInterface) navigateDown() error {
	if m.paletteOpen {
		nodeTypes := []string{"start", "end", "mcp_tool", "transform", "condition", "parallel", "loop"}
		if m.paletteIndex < len(nodeTypes)-1 {
			m.paletteIndex++
		}
	}
	if m.propertyEditor.isOpen {
		// Move to next field
		m.propertyEditor.cursorPos++
	}
	return nil
}

// navigateUp moves cursor up in palette or property editor
func (m *MockTUIInterface) navigateUp() error {
	if m.paletteOpen && m.paletteIndex > 0 {
		m.paletteIndex--
	}
	if m.propertyEditor.isOpen && m.propertyEditor.cursorPos > 0 {
		m.propertyEditor.cursorPos--
	}
	return nil
}

// handleTextInput handles text input for property editor
func (m *MockTUIInterface) handleTextInput(text string) error {
	if m.propertyEditor.isOpen {
		// Simplified: append to current field
		fieldName := m.getCurrentFieldName()
		m.propertyEditor.fields[fieldName] = text
	}
	return nil
}

// openPropertyEditor opens the property editor for a specific node type
func (m *MockTUIInterface) openPropertyEditor(nodeType string) error {
	m.propertyEditor.isOpen = true
	m.propertyEditor.nodeType = nodeType
	m.propertyEditor.cursorPos = 0
	m.propertyEditor.validationErrors = make([]string, 0)

	// Pre-populate with required fields
	switch nodeType {
	case "start":
		m.propertyEditor.fields["id"] = ""
	case "end":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["return_value"] = ""
	case "mcp_tool":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["server_id"] = ""
		m.propertyEditor.fields["tool_name"] = ""
		m.propertyEditor.fields["output_variable"] = ""
	case "transform":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["input_variable"] = ""
		m.propertyEditor.fields["expression"] = ""
		m.propertyEditor.fields["output_variable"] = ""
	case "condition":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["condition"] = ""
	case "parallel":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["merge_strategy"] = ""
	case "loop":
		m.propertyEditor.fields["id"] = ""
		m.propertyEditor.fields["collection"] = ""
		m.propertyEditor.fields["item_variable"] = ""
	}

	return nil
}

// getCurrentFieldName returns the name of the currently focused field
func (m *MockTUIInterface) getCurrentFieldName() string {
	fieldNames := []string{}
	for name := range m.propertyEditor.fields {
		fieldNames = append(fieldNames, name)
	}
	if m.propertyEditor.cursorPos < len(fieldNames) {
		return fieldNames[m.propertyEditor.cursorPos]
	}
	return ""
}

// addNodeFromProperties creates a node from property editor values and adds to workflow
func (m *MockTUIInterface) addNodeFromProperties() error {
	// Validate fields are populated
	for fieldName, value := range m.propertyEditor.fields {
		if value == "" && isRequiredField(m.propertyEditor.nodeType, fieldName) {
			m.propertyEditor.validationErrors = append(
				m.propertyEditor.validationErrors,
				"field "+fieldName+" is required",
			)
		}
	}

	if len(m.propertyEditor.validationErrors) > 0 {
		return nil // Keep editor open with errors
	}

	// Create appropriate node type
	var node workflow.Node
	var err error

	switch m.propertyEditor.nodeType {
	case "start":
		node = &workflow.StartNode{
			ID: m.propertyEditor.fields["id"],
		}
	case "end":
		node = &workflow.EndNode{
			ID:          m.propertyEditor.fields["id"],
			ReturnValue: m.propertyEditor.fields["return_value"],
		}
	case "mcp_tool":
		node = &workflow.MCPToolNode{
			ID:             m.propertyEditor.fields["id"],
			ServerID:       m.propertyEditor.fields["server_id"],
			ToolName:       m.propertyEditor.fields["tool_name"],
			OutputVariable: m.propertyEditor.fields["output_variable"],
			Parameters:     make(map[string]string),
		}
	case "transform":
		node = &workflow.TransformNode{
			ID:             m.propertyEditor.fields["id"],
			InputVariable:  m.propertyEditor.fields["input_variable"],
			Expression:     m.propertyEditor.fields["expression"],
			OutputVariable: m.propertyEditor.fields["output_variable"],
		}
	case "condition":
		node = &workflow.ConditionNode{
			ID:        m.propertyEditor.fields["id"],
			Condition: m.propertyEditor.fields["condition"],
		}
	case "parallel":
		node = &workflow.ParallelNode{
			ID:            m.propertyEditor.fields["id"],
			Branches:      [][]string{},
			MergeStrategy: m.propertyEditor.fields["merge_strategy"],
		}
	case "loop":
		node = &workflow.LoopNode{
			ID:           m.propertyEditor.fields["id"],
			Collection:   m.propertyEditor.fields["collection"],
			ItemVariable: m.propertyEditor.fields["item_variable"],
			Body:         []string{},
		}
	}

	// Add node to workflow
	err = m.workflow.AddNode(node)
	if err != nil {
		m.propertyEditor.validationErrors = append(m.propertyEditor.validationErrors, err.Error())
		return err
	}

	m.selectedNode = node
	m.nodeCount++

	// Close property editor
	m.propertyEditor.isOpen = false
	m.propertyEditor.fields = make(map[string]string)

	return nil
}

// isRequiredField checks if a field is required for a node type
func isRequiredField(nodeType, fieldName string) bool {
	requiredFields := map[string][]string{
		"start":     {"id"},
		"end":       {"id"},
		"mcp_tool":  {"id", "server_id", "tool_name", "output_variable"},
		"transform": {"id", "input_variable", "expression", "output_variable"},
		"condition": {"id", "condition"},
		"parallel":  {"id"},
		"loop":      {"id", "collection", "item_variable"},
	}

	fields, exists := requiredFields[nodeType]
	if !exists {
		return false
	}

	for _, f := range fields {
		if f == fieldName {
			return true
		}
	}
	return false
}

// SetFieldValue sets a value for a specific property editor field
func (m *MockTUIInterface) SetFieldValue(fieldName, value string) {
	if m.propertyEditor.isOpen {
		m.propertyEditor.fields[fieldName] = value
	}
}

// TestNodePalette_OpenAndClose tests opening and closing the node palette
func TestNodePalette_OpenAndClose(t *testing.T) {
	tests := []struct {
		name            string
		keys            []string
		wantPaletteOpen bool
	}{
		{
			name:            "open palette with 'a' key",
			keys:            []string{"a"},
			wantPaletteOpen: true,
		},
		{
			name:            "open and close palette with Esc",
			keys:            []string{"a", "Esc"},
			wantPaletteOpen: false,
		},
		{
			name:            "multiple open/close cycles",
			keys:            []string{"a", "Esc", "a", "Esc"},
			wantPaletteOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			for _, key := range tt.keys {
				if err := tui.SendKey(key); err != nil {
					t.Errorf("SendKey(%s) error = %v", key, err)
				}
			}

			if tui.paletteOpen != tt.wantPaletteOpen {
				t.Errorf("paletteOpen = %v, want %v", tui.paletteOpen, tt.wantPaletteOpen)
			}
		})
	}
}

// TestNodePalette_Navigation tests navigating through node types in palette
func TestNodePalette_Navigation(t *testing.T) {
	tests := []struct {
		name      string
		keys      []string
		wantIndex int
	}{
		{
			name:      "navigate down once",
			keys:      []string{"a", "j"},
			wantIndex: 1,
		},
		{
			name:      "navigate down multiple times",
			keys:      []string{"a", "j", "j", "j"},
			wantIndex: 3,
		},
		{
			name:      "navigate down then up",
			keys:      []string{"a", "j", "j", "k"},
			wantIndex: 1,
		},
		{
			name:      "navigate up at start (should stay at 0)",
			keys:      []string{"a", "k"},
			wantIndex: 0,
		},
		{
			name:      "navigate to last item",
			keys:      []string{"a", "j", "j", "j", "j", "j", "j"},
			wantIndex: 6, // 7 node types (0-6)
		},
		{
			name:      "navigate beyond last item (should stay at 6)",
			keys:      []string{"a", "j", "j", "j", "j", "j", "j", "j"},
			wantIndex: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			for _, key := range tt.keys {
				if err := tui.SendKey(key); err != nil {
					t.Errorf("SendKey(%s) error = %v", key, err)
				}
			}

			if tui.paletteIndex != tt.wantIndex {
				t.Errorf("paletteIndex = %v, want %v", tui.paletteIndex, tt.wantIndex)
			}
		})
	}
}

// TestAddNode_AllNodeTypes tests adding each type of node
func TestAddNode_AllNodeTypes(t *testing.T) {
	tests := []struct {
		name         string
		nodeType     string
		paletteIndex int
		fields       map[string]string
		wantNodeType string
		wantErr      bool
	}{
		{
			name:         "add start node",
			nodeType:     "start",
			paletteIndex: 0,
			fields: map[string]string{
				"id": "start-1",
			},
			wantNodeType: "start",
			wantErr:      false,
		},
		{
			name:         "add end node",
			nodeType:     "end",
			paletteIndex: 1,
			fields: map[string]string{
				"id":           "end-1",
				"return_value": "result",
			},
			wantNodeType: "end",
			wantErr:      false,
		},
		{
			name:         "add MCP tool node",
			nodeType:     "mcp_tool",
			paletteIndex: 2,
			fields: map[string]string{
				"id":              "tool-1",
				"server_id":       "fs",
				"tool_name":       "read_file",
				"output_variable": "file_data",
			},
			wantNodeType: "mcp_tool",
			wantErr:      false,
		},
		{
			name:         "add transform node",
			nodeType:     "transform",
			paletteIndex: 3,
			fields: map[string]string{
				"id":              "transform-1",
				"input_variable":  "input",
				"expression":      "$.data",
				"output_variable": "output",
			},
			wantNodeType: "transform",
			wantErr:      false,
		},
		{
			name:         "add condition node",
			nodeType:     "condition",
			paletteIndex: 4,
			fields: map[string]string{
				"id":        "condition-1",
				"condition": "count > 10",
			},
			wantNodeType: "condition",
			wantErr:      false,
		},
		{
			name:         "add parallel node",
			nodeType:     "parallel",
			paletteIndex: 5,
			fields: map[string]string{
				"id":             "parallel-1",
				"merge_strategy": "wait_all",
			},
			wantNodeType: "parallel",
			wantErr:      false,
		},
		{
			name:         "add loop node",
			nodeType:     "loop",
			paletteIndex: 6,
			fields: map[string]string{
				"id":            "loop-1",
				"collection":    "items",
				"item_variable": "item",
			},
			wantNodeType: "loop",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			// Open palette
			tui.SendKey("a")

			// Navigate to correct index
			for i := 0; i < tt.paletteIndex; i++ {
				tui.SendKey("j")
			}

			// Select node type
			tui.SendKey("Enter")

			// Verify property editor opened
			if !tui.propertyEditor.isOpen {
				t.Fatal("property editor should be open after selecting node type")
			}

			// Fill in fields
			for fieldName, value := range tt.fields {
				tui.SetFieldValue(fieldName, value)
			}

			// Confirm addition
			err := tui.SendKey("Enter")

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify node added to workflow
			if len(wf.Nodes) != 1 {
				t.Errorf("expected 1 node in workflow, got %d", len(wf.Nodes))
				return
			}

			addedNode := wf.Nodes[0]
			if addedNode.Type() != tt.wantNodeType {
				t.Errorf("node type = %s, want %s", addedNode.Type(), tt.wantNodeType)
			}

			// Verify property editor closed
			if tui.propertyEditor.isOpen {
				t.Error("property editor should be closed after adding node")
			}
		})
	}
}

// TestAddNode_ValidationErrors tests validation of node parameters
func TestAddNode_ValidationErrors(t *testing.T) {
	tests := []struct {
		name              string
		nodeType          string
		paletteIndex      int
		fields            map[string]string
		wantValidationErr bool
	}{
		{
			name:         "start node with empty ID",
			nodeType:     "start",
			paletteIndex: 0,
			fields: map[string]string{
				"id": "",
			},
			wantValidationErr: true,
		},
		{
			name:         "MCP tool node missing server_id",
			nodeType:     "mcp_tool",
			paletteIndex: 2,
			fields: map[string]string{
				"id":              "tool-1",
				"server_id":       "", // missing
				"tool_name":       "read_file",
				"output_variable": "data",
			},
			wantValidationErr: true,
		},
		{
			name:         "MCP tool node missing tool_name",
			nodeType:     "mcp_tool",
			paletteIndex: 2,
			fields: map[string]string{
				"id":              "tool-1",
				"server_id":       "fs",
				"tool_name":       "", // missing
				"output_variable": "data",
			},
			wantValidationErr: true,
		},
		{
			name:         "transform node missing expression",
			nodeType:     "transform",
			paletteIndex: 3,
			fields: map[string]string{
				"id":              "transform-1",
				"input_variable":  "input",
				"expression":      "", // missing
				"output_variable": "output",
			},
			wantValidationErr: true,
		},
		{
			name:         "condition node missing condition",
			nodeType:     "condition",
			paletteIndex: 4,
			fields: map[string]string{
				"id":        "condition-1",
				"condition": "", // missing
			},
			wantValidationErr: true,
		},
		{
			name:         "loop node missing collection",
			nodeType:     "loop",
			paletteIndex: 6,
			fields: map[string]string{
				"id":            "loop-1",
				"collection":    "", // missing
				"item_variable": "item",
			},
			wantValidationErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			// Open palette and select node type
			tui.SendKey("a")
			for i := 0; i < tt.paletteIndex; i++ {
				tui.SendKey("j")
			}
			tui.SendKey("Enter")

			// Fill in fields
			for fieldName, value := range tt.fields {
				tui.SetFieldValue(fieldName, value)
			}

			// Attempt to confirm
			tui.SendKey("Enter")

			hasValidationErrors := len(tui.propertyEditor.validationErrors) > 0

			if tt.wantValidationErr && !hasValidationErrors {
				t.Error("expected validation errors but got none")
			}

			if !tt.wantValidationErr && hasValidationErrors {
				t.Errorf("unexpected validation errors: %v", tui.propertyEditor.validationErrors)
			}

			// If validation failed, property editor should still be open
			if tt.wantValidationErr && !tui.propertyEditor.isOpen {
				t.Error("property editor should remain open when validation fails")
			}

			// Node should not be added if validation failed
			if tt.wantValidationErr && len(wf.Nodes) > 0 {
				t.Errorf("node should not be added with validation errors, got %d nodes", len(wf.Nodes))
			}
		})
	}
}

// TestAddNode_CancelOperation tests canceling node addition
func TestAddNode_CancelOperation(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		wantNodeCount int
	}{
		{
			name:          "cancel palette with Esc",
			keys:          []string{"a", "Esc"},
			wantNodeCount: 0,
		},
		{
			name:          "cancel property editor with Esc",
			keys:          []string{"a", "Enter", "Esc"},
			wantNodeCount: 0,
		},
		{
			name:          "cancel after entering some properties",
			keys:          []string{"a", "j", "Enter", "Esc"},
			wantNodeCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			for _, key := range tt.keys {
				tui.SendKey(key)
			}

			if len(wf.Nodes) != tt.wantNodeCount {
				t.Errorf("node count = %d, want %d", len(wf.Nodes), tt.wantNodeCount)
			}

			if tui.paletteOpen {
				t.Error("palette should be closed after cancel")
			}

			if tui.propertyEditor.isOpen {
				t.Error("property editor should be closed after cancel")
			}
		})
	}
}

// TestAddNode_MultipleNodesInSequence tests adding multiple nodes sequentially
func TestAddNode_MultipleNodesInSequence(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	tui := NewMockTUI(wf)

	// Add start node
	tui.SendKey("a")     // Open palette
	tui.SendKey("Enter") // Select start (index 0)
	tui.SetFieldValue("id", "start-1")
	tui.SendKey("Enter") // Confirm

	// Add MCP tool node
	tui.SendKey("a") // Open palette again
	tui.SendKey("j") // Navigate to mcp_tool
	tui.SendKey("j")
	tui.SendKey("Enter") // Select mcp_tool (index 2)
	tui.SetFieldValue("id", "tool-1")
	tui.SetFieldValue("server_id", "fs")
	tui.SetFieldValue("tool_name", "read_file")
	tui.SetFieldValue("output_variable", "data")
	tui.SendKey("Enter") // Confirm

	// Add end node
	tui.SendKey("a")     // Open palette again
	tui.SendKey("j")     // Navigate to end (index 1)
	tui.SendKey("Enter") // Select end
	tui.SetFieldValue("id", "end-1")
	tui.SendKey("Enter") // Confirm

	// Verify all nodes added
	if len(wf.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(wf.Nodes))
	}

	// Verify node types
	nodeTypes := []string{}
	for _, node := range wf.Nodes {
		nodeTypes = append(nodeTypes, node.Type())
	}

	expectedTypes := []string{"start", "mcp_tool", "end"}
	for i, expected := range expectedTypes {
		if i >= len(nodeTypes) || nodeTypes[i] != expected {
			t.Errorf("node %d: type = %s, want %s", i, nodeTypes[i], expected)
		}
	}
}

// TestAddNode_DuplicateIDs tests handling of duplicate node IDs
func TestAddNode_DuplicateIDs(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	tui := NewMockTUI(wf)

	// Add first node
	tui.SendKey("a")
	tui.SendKey("Enter")
	tui.SetFieldValue("id", "node-1")
	tui.SendKey("Enter")

	if len(wf.Nodes) != 1 {
		t.Fatalf("expected 1 node after first addition, got %d", len(wf.Nodes))
	}

	// Try to add second node with same ID
	tui.SendKey("a")
	tui.SendKey("j") // Navigate to end node
	tui.SendKey("Enter")
	tui.SetFieldValue("id", "node-1") // Duplicate ID
	tui.SendKey("Enter")

	// Note: Workflow.AddNode allows duplicates during construction
	// Validation catches this before execution
	// The TUI should eventually prevent this at the UI level
	// For now, just verify the node was added (workflow validation will catch it)
	if len(wf.Nodes) != 2 {
		t.Errorf("expected 2 nodes (validation catches duplicates later), got %d", len(wf.Nodes))
	}

	// Verify workflow validation fails
	err := wf.Validate()
	if err == nil {
		t.Error("expected workflow validation to fail with duplicate IDs")
	}
}

// TestAddNode_MaximumNodes tests behavior with many nodes
func TestAddNode_MaximumNodes(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	tui := NewMockTUI(wf)

	// Add 100 nodes
	maxNodes := 100
	for i := 0; i < maxNodes; i++ {
		tui.SendKey("a")
		tui.SendKey("Enter") // Start node
		tui.SetFieldValue("id", "node-"+string(rune(i)))
		tui.SendKey("Enter")
	}

	if len(wf.Nodes) != maxNodes {
		t.Errorf("expected %d nodes, got %d", maxNodes, len(wf.Nodes))
	}

	// Verify we can still add more nodes (no arbitrary limit)
	tui.SendKey("a")
	tui.SendKey("Enter")
	tui.SetFieldValue("id", "node-extra")
	tui.SendKey("Enter")

	if len(wf.Nodes) != maxNodes+1 {
		t.Errorf("expected %d nodes, got %d", maxNodes+1, len(wf.Nodes))
	}
}

// TestAddNode_PropertyEditorFieldNavigation tests navigating between property fields
func TestAddNode_PropertyEditorFieldNavigation(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	tui := NewMockTUI(wf)

	// Open property editor for MCP tool node (has multiple fields)
	tui.SendKey("a")
	tui.SendKey("j") // Navigate to mcp_tool
	tui.SendKey("j")
	tui.SendKey("Enter")

	initialCursor := tui.propertyEditor.cursorPos

	// Navigate down through fields
	tui.SendKey("j")
	if tui.propertyEditor.cursorPos <= initialCursor {
		t.Error("cursor should move down when pressing j in property editor")
	}

	cursorAfterDown := tui.propertyEditor.cursorPos

	// Navigate back up
	tui.SendKey("k")
	if tui.propertyEditor.cursorPos >= cursorAfterDown {
		t.Error("cursor should move up when pressing k in property editor")
	}
}

// TestAddNode_NodeAppearanceInWorkflow tests that added nodes appear in workflow model
func TestAddNode_NodeAppearanceInWorkflow(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	tui := NewMockTUI(wf)

	// Add a start node
	tui.SendKey("a")
	tui.SendKey("Enter")
	tui.SetFieldValue("id", "start-1")
	tui.SendKey("Enter")

	// Verify node exists in workflow
	if len(wf.Nodes) == 0 {
		t.Fatal("node should exist in workflow after addition")
	}

	node := wf.Nodes[0]

	// Verify node ID
	if node.GetID() != "start-1" {
		t.Errorf("node ID = %s, want start-1", node.GetID())
	}

	// Verify node type
	if node.Type() != "start" {
		t.Errorf("node type = %s, want start", node.Type())
	}

	// Verify node validates
	if err := node.Validate(); err != nil {
		t.Errorf("node validation failed: %v", err)
	}
}

// TestAddNode_ComplexNodeConfiguration tests configuring nodes with complex parameters
func TestAddNode_ComplexNodeConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		nodeType     string
		paletteIndex int
		fields       map[string]string
		validate     func(*testing.T, workflow.Node)
	}{
		{
			name:         "MCP tool with parameters",
			nodeType:     "mcp_tool",
			paletteIndex: 2,
			fields: map[string]string{
				"id":              "tool-read",
				"server_id":       "filesystem",
				"tool_name":       "read_file",
				"output_variable": "file_contents",
			},
			validate: func(t *testing.T, node workflow.Node) {
				mcpNode, ok := node.(*workflow.MCPToolNode)
				if !ok {
					t.Fatal("node should be MCPToolNode")
				}
				if mcpNode.ServerID != "filesystem" {
					t.Errorf("ServerID = %s, want filesystem", mcpNode.ServerID)
				}
				if mcpNode.ToolName != "read_file" {
					t.Errorf("ToolName = %s, want read_file", mcpNode.ToolName)
				}
			},
		},
		{
			name:         "transform with JSONPath expression",
			nodeType:     "transform",
			paletteIndex: 3,
			fields: map[string]string{
				"id":              "transform-extract",
				"input_variable":  "file_contents",
				"expression":      "$.data.users[0].email",
				"output_variable": "user_email",
			},
			validate: func(t *testing.T, node workflow.Node) {
				transformNode, ok := node.(*workflow.TransformNode)
				if !ok {
					t.Fatal("node should be TransformNode")
				}
				if transformNode.Expression != "$.data.users[0].email" {
					t.Errorf("Expression = %s, want $.data.users[0].email", transformNode.Expression)
				}
			},
		},
		{
			name:         "parallel with merge strategy",
			nodeType:     "parallel",
			paletteIndex: 5,
			fields: map[string]string{
				"id":             "parallel-branches",
				"merge_strategy": "wait_all",
			},
			validate: func(t *testing.T, node workflow.Node) {
				parallelNode, ok := node.(*workflow.ParallelNode)
				if !ok {
					t.Fatal("node should be ParallelNode")
				}
				if parallelNode.MergeStrategy != "wait_all" {
					t.Errorf("MergeStrategy = %s, want wait_all", parallelNode.MergeStrategy)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tui := NewMockTUI(wf)

			// Navigate to node type and open property editor
			tui.SendKey("a")
			for i := 0; i < tt.paletteIndex; i++ {
				tui.SendKey("j")
			}
			tui.SendKey("Enter")

			// Fill in fields
			for fieldName, value := range tt.fields {
				tui.SetFieldValue(fieldName, value)
			}

			// Confirm addition
			tui.SendKey("Enter")

			// Validate node was added
			if len(wf.Nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(wf.Nodes))
			}

			// Run custom validation
			tt.validate(t, wf.Nodes[0])
		})
	}
}
