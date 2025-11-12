package tui

import (
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestNewPropertyPanel tests property panel creation
func TestNewPropertyPanel(t *testing.T) {
	node := &workflow.MCPToolNode{
		ID:             "test-node-1",
		ServerID:       "server-1",
		ToolName:       "test.tool",
		OutputVariable: "result",
	}

	panel := NewPropertyPanel(node)

	if panel == nil {
		t.Fatal("NewPropertyPanel returned nil")
	}

	if panel.node.GetID() != "test-node-1" {
		t.Errorf("node ID = %v, want test-node-1", panel.node.GetID())
	}

	if panel.visible {
		t.Error("panel should not be visible by default")
	}

	if panel.editIndex != 0 {
		t.Errorf("editIndex = %v, want 0", panel.editIndex)
	}

	if len(panel.fields) == 0 {
		t.Error("fields should not be empty for MCPToolNode")
	}
}

// TestPropertyPanelShowHide tests show/hide functionality
func TestPropertyPanelShowHide(t *testing.T) {
	node := &workflow.TransformNode{
		ID:             "transform-1",
		InputVariable:  "input",
		Expression:     "data + 1",
		OutputVariable: "output",
	}

	panel := NewPropertyPanel(node)

	// Initially hidden
	if panel.IsVisible() {
		t.Error("panel should be hidden initially")
	}

	// Show
	panel.Show()
	if !panel.IsVisible() {
		t.Error("panel should be visible after Show()")
	}

	// Hide
	panel.Hide()
	if panel.IsVisible() {
		t.Error("panel should be hidden after Hide()")
	}
}

// TestPropertyPanelFieldNavigation tests field navigation
func TestPropertyPanelFieldNavigation(t *testing.T) {
	node := &workflow.MCPToolNode{
		ID:             "test-node",
		ServerID:       "server",
		ToolName:       "tool",
		OutputVariable: "result",
	}

	panel := NewPropertyPanel(node)
	fieldCount := len(panel.fields)

	if fieldCount == 0 {
		t.Fatal("No fields to navigate")
	}

	// Test NextField
	initialIndex := panel.editIndex
	panel.NextField()

	if panel.editIndex != (initialIndex+1)%fieldCount {
		t.Errorf("NextField: editIndex = %v, want %v", panel.editIndex, (initialIndex+1)%fieldCount)
	}

	// Test PrevField
	panel.PrevField()
	if panel.editIndex != initialIndex {
		t.Errorf("PrevField: editIndex = %v, want %v", panel.editIndex, initialIndex)
	}

	// Test wrap-around (forward)
	for i := 0; i < fieldCount; i++ {
		panel.NextField()
	}
	if panel.editIndex != initialIndex {
		t.Errorf("NextField wrap-around: editIndex = %v, want %v", panel.editIndex, initialIndex)
	}

	// Test wrap-around (backward)
	panel.PrevField()
	if panel.editIndex != fieldCount-1 {
		t.Errorf("PrevField wrap-around: editIndex = %v, want %v", panel.editIndex, fieldCount-1)
	}
}

// TestPropertyPanelSetFieldValue tests field value updates
func TestPropertyPanelSetFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		node      workflow.Node
		fieldIdx  int
		newValue  string
		wantError bool
	}{
		{
			name: "valid text field update",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "server-1",
				ToolName:       "tool-1",
				OutputVariable: "result",
			},
			fieldIdx:  1, // Server ID field
			newValue:  "new-server",
			wantError: false,
		},
		{
			name: "valid expression field update",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "old_expr",
				OutputVariable: "output",
			},
			fieldIdx:  2, // Expression field
			newValue:  "new_expr + 1",
			wantError: false,
		},
		{
			name: "invalid expression syntax",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "valid",
				OutputVariable: "output",
			},
			fieldIdx:  2, // Expression field
			newValue:  "invalid +",
			wantError: true,
		},
		{
			name: "text field too long",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "server-1",
				ToolName:       "tool-1",
				OutputVariable: "result",
			},
			fieldIdx:  1, // Server ID field
			newValue:  strings.Repeat("a", 300),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewPropertyPanel(tt.node)
			panel.editIndex = tt.fieldIdx

			err := panel.SetFieldValue(tt.newValue)

			if (err != nil) != tt.wantError {
				t.Errorf("SetFieldValue() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && panel.fields[tt.fieldIdx].value != tt.newValue {
				t.Errorf("field value = %v, want %v", panel.fields[tt.fieldIdx].value, tt.newValue)
			}
		})
	}
}

// TestPropertyPanelSaveChanges tests saving changes
func TestPropertyPanelSaveChanges(t *testing.T) {
	tests := []struct {
		name          string
		node          workflow.Node
		modifications func(*PropertyPanel)
		wantError     bool
		validate      func(*testing.T, workflow.Node)
	}{
		{
			name: "save valid MCPToolNode changes",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "old-server",
				ToolName:       "old-tool",
				OutputVariable: "old_result",
			},
			modifications: func(p *PropertyPanel) {
				p.editIndex = 1 // Server ID
				_ = p.SetFieldValue("new-server")
				p.editIndex = 2 // Tool Name
				_ = p.SetFieldValue("new-tool")
			},
			wantError: false,
			validate: func(t *testing.T, n workflow.Node) {
				mcp := n.(*workflow.MCPToolNode)
				if mcp.ServerID != "new-server" {
					t.Errorf("ServerID = %v, want new-server", mcp.ServerID)
				}
				if mcp.ToolName != "new-tool" {
					t.Errorf("ToolName = %v, want new-tool", mcp.ToolName)
				}
			},
		},
		{
			name: "save valid TransformNode changes",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "old_input",
				Expression:     "old_expr",
				OutputVariable: "old_output",
			},
			modifications: func(p *PropertyPanel) {
				p.editIndex = 2 // Expression
				_ = p.SetFieldValue("new_expr + 1")
			},
			wantError: false,
			validate: func(t *testing.T, n workflow.Node) {
				transform := n.(*workflow.TransformNode)
				if transform.Expression != "new_expr + 1" {
					t.Errorf("Expression = %v, want new_expr + 1", transform.Expression)
				}
			},
		},
		{
			name: "save fails with invalid expression",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "valid",
				OutputVariable: "output",
			},
			modifications: func(p *PropertyPanel) {
				p.editIndex = 2                  // Expression
				_ = p.SetFieldValue("invalid +") // Intentionally set invalid
			},
			wantError: true,
		},
		{
			name: "save fails with empty required field",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "server-1",
				ToolName:       "tool-1",
				OutputVariable: "result",
			},
			modifications: func(p *PropertyPanel) {
				p.editIndex = 1 // Server ID (required)
				_ = p.SetFieldValue("")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewPropertyPanel(tt.node)

			if tt.modifications != nil {
				tt.modifications(panel)
			}

			updatedNode, err := panel.SaveChanges()

			if (err != nil) != tt.wantError {
				t.Errorf("SaveChanges() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && tt.validate != nil {
				tt.validate(t, *updatedNode)
			}
		})
	}
}

// TestPropertyPanelCancelChanges tests canceling changes
func TestPropertyPanelCancelChanges(t *testing.T) {
	node := &workflow.MCPToolNode{
		ID:             "node-1",
		ServerID:       "original-server",
		ToolName:       "original-tool",
		OutputVariable: "result",
	}

	panel := NewPropertyPanel(node)

	// Get original value
	originalServerID := panel.fields[1].value

	// Make a change
	panel.editIndex = 1 // Server ID
	err := panel.SetFieldValue("modified-server")
	if err != nil {
		t.Fatalf("SetFieldValue() error = %v", err)
	}

	// Verify change was made
	if panel.fields[1].value != "modified-server" {
		t.Error("Field value should be modified")
	}

	// Cancel changes
	panel.CancelChanges()

	// Verify value is restored
	if panel.fields[1].value != originalServerID {
		t.Errorf("Field value after cancel = %v, want %v", panel.fields[1].value, originalServerID)
	}

	// Verify edit index is reset
	if panel.editIndex != 0 {
		t.Errorf("editIndex after cancel = %v, want 0", panel.editIndex)
	}
}

// TestPropertyPanelIsDirty tests dirty flag tracking
func TestPropertyPanelIsDirty(t *testing.T) {
	node := &workflow.TransformNode{
		ID:             "transform-1",
		InputVariable:  "input",
		Expression:     "original",
		OutputVariable: "output",
	}

	panel := NewPropertyPanel(node)

	// Initially not dirty
	if panel.IsDirty() {
		t.Error("Panel should not be dirty initially")
	}

	// Modify a field
	panel.editIndex = 2 // Expression
	_ = panel.SetFieldValue("modified")

	// Should be dirty
	if !panel.IsDirty() {
		t.Error("Panel should be dirty after modification")
	}

	// Cancel changes
	panel.CancelChanges()

	// Should not be dirty anymore
	if panel.IsDirty() {
		t.Error("Panel should not be dirty after cancel")
	}

	// Modify again
	panel.editIndex = 2
	_ = panel.SetFieldValue("modified2")

	// Should be dirty
	if !panel.IsDirty() {
		t.Error("Panel should be dirty after second modification")
	}

	// Save changes
	_, _ = panel.SaveChanges() // Ignore error for this test

	// After save with same values, should not be dirty
	// (Note: SaveChanges updates the node, so we need to rebuild fields)
	panel.fields = buildFieldsForNode(panel.node)
	if panel.IsDirty() {
		t.Error("Panel should not be dirty after save")
	}
}

// TestPropertyPanelValidate tests validation
func TestPropertyPanelValidate(t *testing.T) {
	tests := []struct {
		name      string
		node      workflow.Node
		mods      func(*PropertyPanel)
		wantError bool
		errorMsg  string
	}{
		{
			name: "all fields valid",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "server-1",
				ToolName:       "tool-1",
				OutputVariable: "result",
			},
			wantError: false,
		},
		{
			name: "required field empty",
			node: &workflow.MCPToolNode{
				ID:             "node-1",
				ServerID:       "server-1",
				ToolName:       "tool-1",
				OutputVariable: "result",
			},
			mods: func(p *PropertyPanel) {
				p.editIndex = 2 // Tool Name (required)
				_ = p.SetFieldValue("")
			},
			wantError: true,
			errorMsg:  "required",
		},
		{
			name: "invalid expression syntax",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "valid",
				OutputVariable: "output",
			},
			mods: func(p *PropertyPanel) {
				p.editIndex = 2 // Expression
				_ = p.SetFieldValue("invalid +")
			},
			wantError: true,
			errorMsg:  "Expression",
		},
		{
			name: "invalid JSONPath",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "$.items",
				ItemVariable: "item",
				Body:         []string{"node-1"},
			},
			mods: func(p *PropertyPanel) {
				p.editIndex = 1 // Collection (JSONPath)
				_ = p.SetFieldValue("invalid")
			},
			wantError: true,
			errorMsg:  "Collection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewPropertyPanel(tt.node)

			if tt.mods != nil {
				tt.mods(panel)
			}

			err := panel.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, should contain %v", err, tt.errorMsg)
				}
			}
		})
	}
}

// TestBuildFieldsForNode tests field generation for different node types
func TestBuildFieldsForNode(t *testing.T) {
	tests := []struct {
		name           string
		node           workflow.Node
		expectedFields int
		checkLabels    []string
	}{
		{
			name: "MCPToolNode",
			node: &workflow.MCPToolNode{
				ID:             "mcp-1",
				ServerID:       "server",
				ToolName:       "tool",
				OutputVariable: "result",
			},
			expectedFields: 4, // ID, ServerID, ToolName, OutputVariable
			checkLabels:    []string{"Node ID", "Server ID", "Tool Name", "Output Variable"},
		},
		{
			name: "TransformNode",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "expr",
				OutputVariable: "output",
			},
			expectedFields: 4, // ID, InputVariable, Expression, OutputVariable
			checkLabels:    []string{"Node ID", "Input Variable", "Expression", "Output Variable"},
		},
		{
			name: "ConditionNode",
			node: &workflow.ConditionNode{
				ID:        "condition-1",
				Condition: "x > 10",
			},
			expectedFields: 2, // ID, Condition
			checkLabels:    []string{"Node ID", "Condition"},
		},
		{
			name: "EndNode",
			node: &workflow.EndNode{
				ID:          "end-1",
				ReturnValue: "${result}",
			},
			expectedFields: 2, // ID, ReturnValue
			checkLabels:    []string{"Node ID", "Return Value"},
		},
		{
			name: "StartNode",
			node: &workflow.StartNode{
				ID: "start-1",
			},
			expectedFields: 1, // ID only
			checkLabels:    []string{"Node ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := buildFieldsForNode(tt.node)

			if len(fields) != tt.expectedFields {
				t.Errorf("field count = %v, want %v", len(fields), tt.expectedFields)
			}

			// Check that expected labels are present
			labelSet := make(map[string]bool)
			for _, field := range fields {
				labelSet[field.label] = true
			}

			for _, expectedLabel := range tt.checkLabels {
				if !labelSet[expectedLabel] {
					t.Errorf("missing expected field label: %v", expectedLabel)
				}
			}
		})
	}
}
