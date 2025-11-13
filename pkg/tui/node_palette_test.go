package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

func TestNewNodePalette(t *testing.T) {
	palette := NewNodePalette()

	if palette == nil {
		t.Fatal("NewNodePalette() returned nil")
	}

	// Should have 6 node types (MCP Tool, Transform, Condition, Loop, Parallel, End)
	if len(palette.nodeTypes) != 6 {
		t.Errorf("expected 6 node types, got %d", len(palette.nodeTypes))
	}

	// Should start with index 0
	if palette.selectedIndex != 0 {
		t.Errorf("expected selectedIndex 0, got %d", palette.selectedIndex)
	}

	// Should start hidden
	if palette.visible {
		t.Error("expected palette to start hidden")
	}

	// Should have empty filter
	if palette.filterText != "" {
		t.Errorf("expected empty filterText, got %q", palette.filterText)
	}
}

func TestNodePaletteVisibility(t *testing.T) {
	palette := NewNodePalette()

	// Initially hidden
	if palette.IsVisible() {
		t.Error("palette should start hidden")
	}

	// Show palette
	palette.Show()
	if !palette.IsVisible() {
		t.Error("palette should be visible after Show()")
	}

	// Hide palette
	palette.Hide()
	if palette.IsVisible() {
		t.Error("palette should be hidden after Hide()")
	}
}

func TestNodePaletteShowResetsState(t *testing.T) {
	palette := NewNodePalette()

	// Set some state
	palette.selectedIndex = 3
	palette.filterText = "test"

	// Show should reset state
	palette.Show()

	if palette.selectedIndex != 0 {
		t.Errorf("Show() should reset selectedIndex to 0, got %d", palette.selectedIndex)
	}

	if palette.filterText != "" {
		t.Errorf("Show() should reset filterText to empty, got %q", palette.filterText)
	}
}

func TestNodePaletteFilter(t *testing.T) {
	tests := []struct {
		name          string
		filterText    string
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "empty filter shows all",
			filterText:    "",
			expectedCount: 6,
			expectedFirst: "MCP Tool",
		},
		{
			name:          "filter 'trans' matches Transform",
			filterText:    "trans",
			expectedCount: 1,
			expectedFirst: "Transform",
		},
		{
			name:          "filter 'mcp' matches MCP Tool",
			filterText:    "mcp",
			expectedCount: 1,
			expectedFirst: "MCP Tool",
		},
		{
			name:          "filter 'cond' matches Condition",
			filterText:    "cond",
			expectedCount: 1,
			expectedFirst: "Condition",
		},
		{
			name:          "filter 'loop' matches Loop",
			filterText:    "loop",
			expectedCount: 1,
			expectedFirst: "Loop",
		},
		{
			name:          "filter 'parallel' matches Parallel",
			filterText:    "parallel",
			expectedCount: 1,
			expectedFirst: "Parallel",
		},
		{
			name:          "filter 'end' matches End",
			filterText:    "end",
			expectedCount: 1,
			expectedFirst: "End",
		},
		{
			name:          "filter 'tool' matches MCP Tool",
			filterText:    "tool",
			expectedCount: 1,
			expectedFirst: "MCP Tool",
		},
		{
			name:          "case insensitive: 'TRANS' matches Transform",
			filterText:    "TRANS",
			expectedCount: 1,
			expectedFirst: "Transform",
		},
		{
			name:          "no match returns empty",
			filterText:    "nonexistent",
			expectedCount: 0,
			expectedFirst: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := NewNodePalette()
			filtered := palette.Filter(tt.filterText)

			if len(filtered) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(filtered))
			}

			if tt.expectedCount > 0 {
				if filtered[0].typeName != tt.expectedFirst {
					t.Errorf("expected first result %q, got %q", tt.expectedFirst, filtered[0].typeName)
				}
			}

			// Verify filterText was stored
			if palette.filterText != tt.filterText {
				t.Errorf("expected filterText %q, got %q", tt.filterText, palette.filterText)
			}
		})
	}
}

func TestNodePaletteFilterResetsSelection(t *testing.T) {
	palette := NewNodePalette()

	// Select last item
	palette.selectedIndex = 5

	// Filter to 1 item - should reset selection
	palette.Filter("trans")

	if palette.selectedIndex != 0 {
		t.Errorf("Filter() should reset selectedIndex when current selection filtered out, got %d", palette.selectedIndex)
	}
}

func TestNodePaletteNavigation(t *testing.T) {
	palette := NewNodePalette()

	// Test Next() with all items
	palette.selectedIndex = 0
	palette.Next()
	if palette.selectedIndex != 1 {
		t.Errorf("Next() should increment selectedIndex, got %d", palette.selectedIndex)
	}

	// Test wrap-around at end
	palette.selectedIndex = 5 // Last item
	palette.Next()
	if palette.selectedIndex != 0 {
		t.Errorf("Next() should wrap to 0 at end, got %d", palette.selectedIndex)
	}

	// Test Previous()
	palette.selectedIndex = 1
	palette.Previous()
	if palette.selectedIndex != 0 {
		t.Errorf("Previous() should decrement selectedIndex, got %d", palette.selectedIndex)
	}

	// Test wrap-around at start
	palette.selectedIndex = 0
	palette.Previous()
	if palette.selectedIndex != 5 {
		t.Errorf("Previous() should wrap to last item at start, got %d", palette.selectedIndex)
	}
}

func TestNodePaletteNavigationWithFilter(t *testing.T) {
	palette := NewNodePalette()

	// Filter to single item
	palette.Filter("trans")
	palette.selectedIndex = 0

	// Next should wrap to 0 (only 1 item)
	palette.Next()
	if palette.selectedIndex != 0 {
		t.Errorf("Next() with single filtered item should stay at 0, got %d", palette.selectedIndex)
	}

	// Previous should wrap to 0 (only 1 item)
	palette.Previous()
	if palette.selectedIndex != 0 {
		t.Errorf("Previous() with single filtered item should stay at 0, got %d", palette.selectedIndex)
	}

	// Filter to empty - navigation should be safe
	palette.Filter("nonexistent")
	palette.Next()     // Should not panic
	palette.Previous() // Should not panic
}

func TestNodePaletteGetSelected(t *testing.T) {
	palette := NewNodePalette()

	// Get first item
	selected := palette.GetSelected()
	if selected.typeName != "MCP Tool" {
		t.Errorf("expected first item 'MCP Tool', got %q", selected.typeName)
	}

	// Move to next and get
	palette.Next()
	selected = palette.GetSelected()
	if selected.typeName != "Transform" {
		t.Errorf("expected 'Transform', got %q", selected.typeName)
	}

	// Filter and get
	palette.Filter("cond")
	selected = palette.GetSelected()
	if selected.typeName != "Condition" {
		t.Errorf("expected 'Condition', got %q", selected.typeName)
	}
}

func TestNodePaletteGetSelectedEmpty(t *testing.T) {
	palette := NewNodePalette()

	// Filter to empty
	palette.Filter("nonexistent")
	selected := palette.GetSelected()

	// Should return empty nodeTypeInfo
	if selected.typeName != "" {
		t.Errorf("expected empty typeName for no match, got %q", selected.typeName)
	}
}

func TestNodePaletteCreateNode(t *testing.T) {
	tests := []struct {
		name         string
		selectType   string
		expectedType string
		validate     func(*testing.T, workflow.Node)
	}{
		{
			name:         "create MCP Tool node",
			selectType:   "mcp",
			expectedType: "mcp_tool",
			validate: func(t *testing.T, node workflow.Node) {
				mcpNode, ok := node.(*workflow.MCPToolNode)
				if !ok {
					t.Fatal("expected MCPToolNode")
				}
				if mcpNode.ID == "" {
					t.Error("node ID should not be empty")
				}
				if mcpNode.OutputVariable != "result" {
					t.Errorf("expected OutputVariable 'result', got %q", mcpNode.OutputVariable)
				}
			},
		},
		{
			name:         "create Transform node",
			selectType:   "trans",
			expectedType: "transform",
			validate: func(t *testing.T, node workflow.Node) {
				transformNode, ok := node.(*workflow.TransformNode)
				if !ok {
					t.Fatal("expected TransformNode")
				}
				if transformNode.ID == "" {
					t.Error("node ID should not be empty")
				}
				if transformNode.InputVariable != "input" {
					t.Errorf("expected InputVariable 'input', got %q", transformNode.InputVariable)
				}
				if transformNode.OutputVariable != "output" {
					t.Errorf("expected OutputVariable 'output', got %q", transformNode.OutputVariable)
				}
			},
		},
		{
			name:         "create Condition node",
			selectType:   "cond",
			expectedType: "condition",
			validate: func(t *testing.T, node workflow.Node) {
				condNode, ok := node.(*workflow.ConditionNode)
				if !ok {
					t.Fatal("expected ConditionNode")
				}
				if condNode.ID == "" {
					t.Error("node ID should not be empty")
				}
				if condNode.Condition != "" {
					t.Errorf("expected empty Condition, got %q", condNode.Condition)
				}
			},
		},
		{
			name:         "create Loop node",
			selectType:   "loop",
			expectedType: "loop",
			validate: func(t *testing.T, node workflow.Node) {
				loopNode, ok := node.(*workflow.LoopNode)
				if !ok {
					t.Fatal("expected LoopNode")
				}
				if loopNode.ID == "" {
					t.Error("node ID should not be empty")
				}
				if loopNode.Body == nil {
					t.Error("Body should not be nil")
				}
			},
		},
		{
			name:         "create Parallel node",
			selectType:   "parallel",
			expectedType: "parallel",
			validate: func(t *testing.T, node workflow.Node) {
				parallelNode, ok := node.(*workflow.ParallelNode)
				if !ok {
					t.Fatal("expected ParallelNode")
				}
				if parallelNode.ID == "" {
					t.Error("node ID should not be empty")
				}
				if len(parallelNode.Branches) != 2 {
					t.Errorf("expected 2 branches, got %d", len(parallelNode.Branches))
				}
				if parallelNode.MergeStrategy != "wait_all" {
					t.Errorf("expected MergeStrategy 'wait_all', got %q", parallelNode.MergeStrategy)
				}
			},
		},
		{
			name:         "create End node",
			selectType:   "end",
			expectedType: "end",
			validate: func(t *testing.T, node workflow.Node) {
				endNode, ok := node.(*workflow.EndNode)
				if !ok {
					t.Fatal("expected EndNode")
				}
				if endNode.ID == "" {
					t.Error("node ID should not be empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := NewNodePalette()

			// Filter to select the type
			palette.Filter(tt.selectType)

			// Create node
			node, err := palette.CreateNode()
			if err != nil {
				t.Fatalf("CreateNode() error: %v", err)
			}

			if node == nil {
				t.Fatal("CreateNode() returned nil node")
			}

			// Verify node type
			if node.Type() != tt.expectedType {
				t.Errorf("expected node type %q, got %q", tt.expectedType, node.Type())
			}

			// Run type-specific validation
			if tt.validate != nil {
				tt.validate(t, node)
			}
		})
	}
}

func TestNodePaletteCreateNodeWithoutSelection(t *testing.T) {
	palette := NewNodePalette()

	// Filter to nothing
	palette.Filter("nonexistent")

	// Should return error
	_, err := palette.CreateNode()
	if err == nil {
		t.Error("CreateNode() should return error when no type selected")
	}
}

func TestNodePaletteNodeTypeDefinitions(t *testing.T) {
	palette := NewNodePalette()

	// Verify all expected node types are present
	expectedTypes := map[string]struct {
		icon        string
		description string
	}{
		"MCP Tool":  {icon: "🔧", description: "Execute MCP server tool"},
		"Transform": {icon: "🔄", description: "Transform data using JSONPath, template, or jq"},
		"Condition": {icon: "❓", description: "Conditional branching"},
		"Loop":      {icon: "🔁", description: "Iterate over collections"},
		"Parallel":  {icon: "⚡", description: "Concurrent execution"},
		"End":       {icon: "🏁", description: "Exit point with output"},
	}

	if len(palette.nodeTypes) != len(expectedTypes) {
		t.Fatalf("expected %d node types, got %d", len(expectedTypes), len(palette.nodeTypes))
	}

	for _, nodeType := range palette.nodeTypes {
		expected, ok := expectedTypes[nodeType.typeName]
		if !ok {
			t.Errorf("unexpected node type: %q", nodeType.typeName)
			continue
		}

		if nodeType.icon != expected.icon {
			t.Errorf("%s: expected icon %q, got %q", nodeType.typeName, expected.icon, nodeType.icon)
		}

		if nodeType.description != expected.description {
			t.Errorf("%s: expected description %q, got %q", nodeType.typeName, expected.description, nodeType.description)
		}

		if nodeType.defaultConfig == nil {
			t.Errorf("%s: defaultConfig should not be nil", nodeType.typeName)
		}
	}
}

func TestNodePaletteDefaultConfigs(t *testing.T) {
	palette := NewNodePalette()

	tests := []struct {
		typeName      string
		expectedKeys  []string
		expectedValue map[string]interface{}
	}{
		{
			typeName:     "MCP Tool",
			expectedKeys: []string{"name", "serverID", "toolName"},
			expectedValue: map[string]interface{}{
				"name": "tool",
			},
		},
		{
			typeName:     "Transform",
			expectedKeys: []string{"name", "type", "expression"},
			expectedValue: map[string]interface{}{
				"name": "transform",
				"type": "jsonpath",
			},
		},
		{
			typeName:     "Condition",
			expectedKeys: []string{"name", "expression"},
			expectedValue: map[string]interface{}{
				"name": "condition",
			},
		},
		{
			typeName:     "Loop",
			expectedKeys: []string{"name", "collection", "variable"},
			expectedValue: map[string]interface{}{
				"name": "loop",
			},
		},
		{
			typeName:     "Parallel",
			expectedKeys: []string{"name", "branches"},
			expectedValue: map[string]interface{}{
				"name":     "parallel",
				"branches": 2,
			},
		},
		{
			typeName:     "End",
			expectedKeys: []string{"name", "output"},
			expectedValue: map[string]interface{}{
				"name": "end",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			// Find node type
			var nodeType *nodeTypeInfo
			for i := range palette.nodeTypes {
				if palette.nodeTypes[i].typeName == tt.typeName {
					nodeType = &palette.nodeTypes[i]
					break
				}
			}

			if nodeType == nil {
				t.Fatalf("node type %q not found", tt.typeName)
			}

			// Verify all expected keys are present
			for _, key := range tt.expectedKeys {
				if _, ok := nodeType.defaultConfig[key]; !ok {
					t.Errorf("defaultConfig missing key %q", key)
				}
			}

			// Verify specific values
			for key, expectedValue := range tt.expectedValue {
				actualValue, ok := nodeType.defaultConfig[key]
				if !ok {
					t.Errorf("defaultConfig missing key %q", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("key %q: expected value %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestNodePaletteUniqueNodeIDs(t *testing.T) {
	palette := NewNodePalette()

	// Create multiple nodes and verify IDs are unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		node, err := palette.CreateNode()
		if err != nil {
			t.Fatalf("CreateNode() error: %v", err)
		}

		id := node.GetID()
		if ids[id] {
			t.Errorf("duplicate node ID: %s", id)
		}
		ids[id] = true
	}
}
