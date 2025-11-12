package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestCreateBasicTemplate tests the basic workflow template
func TestCreateBasicTemplate(t *testing.T) {
	wf := CreateBasicTemplate()

	// Verify workflow created
	if wf == nil {
		t.Fatal("CreateBasicTemplate() returned nil")
	}

	// Verify node count (3 nodes: Start, MCP Tool, End)
	if len(wf.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(wf.Nodes))
	}

	// Verify node types
	nodeTypes := make(map[string]bool)
	for _, node := range wf.Nodes {
		nodeTypes[node.Type()] = true
	}

	expectedTypes := []string{"start", "mcp_tool", "end"}
	for _, expectedType := range expectedTypes {
		if !nodeTypes[expectedType] {
			t.Errorf("Missing node type: %s", expectedType)
		}
	}

	// Verify edge count (2 edges: Start→Tool, Tool→End)
	if len(wf.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(wf.Edges))
	}

	// Verify edges are connected correctly
	edges := make(map[string]string)
	for _, edge := range wf.Edges {
		edges[edge.FromNodeID] = edge.ToNodeID
	}

	if edges["start"] != "mcp-tool-1" {
		t.Errorf("Expected edge from start to mcp-tool-1, got %s", edges["start"])
	}
	if edges["mcp-tool-1"] != "end" {
		t.Errorf("Expected edge from mcp-tool-1 to end, got %s", edges["mcp-tool-1"])
	}

	// Verify workflow is valid
	if err := wf.Validate(); err != nil {
		t.Errorf("Template workflow failed validation: %v", err)
	}
}

// TestCreateETLTemplate tests the ETL workflow template
func TestCreateETLTemplate(t *testing.T) {
	wf := CreateETLTemplate()

	// Verify workflow created
	if wf == nil {
		t.Fatal("CreateETLTemplate() returned nil")
	}

	// Verify node count (5 nodes: Start, Extract, Transform, Load, End)
	if len(wf.Nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(wf.Nodes))
	}

	// Verify node types
	nodeTypes := make(map[string]int)
	for _, node := range wf.Nodes {
		nodeTypes[node.Type()]++
	}

	if nodeTypes["start"] != 1 {
		t.Errorf("Expected 1 start node, got %d", nodeTypes["start"])
	}
	if nodeTypes["mcp_tool"] != 2 {
		t.Errorf("Expected 2 mcp_tool nodes, got %d", nodeTypes["mcp_tool"])
	}
	if nodeTypes["transform"] != 1 {
		t.Errorf("Expected 1 transform node, got %d", nodeTypes["transform"])
	}
	if nodeTypes["end"] != 1 {
		t.Errorf("Expected 1 end node, got %d", nodeTypes["end"])
	}

	// Verify edge count (4 edges: linear flow)
	if len(wf.Edges) != 4 {
		t.Errorf("Expected 4 edges, got %d", len(wf.Edges))
	}

	// Verify edges form correct sequence
	edges := make(map[string]string)
	for _, edge := range wf.Edges {
		edges[edge.FromNodeID] = edge.ToNodeID
	}

	expectedSequence := map[string]string{
		"start":     "extract",
		"extract":   "transform",
		"transform": "load",
		"load":      "end",
	}

	for from, expectedTo := range expectedSequence {
		if edges[from] != expectedTo {
			t.Errorf("Expected edge from %s to %s, got %s", from, expectedTo, edges[from])
		}
	}

	// Verify MCP tool nodes are configured
	for _, node := range wf.Nodes {
		if mcpNode, ok := node.(*workflow.MCPToolNode); ok {
			if mcpNode.ToolName == "" {
				t.Errorf("MCP tool node %s has empty tool name", mcpNode.ID)
			}
			if mcpNode.ServerID == "" {
				t.Errorf("MCP tool node %s has empty server ID", mcpNode.ID)
			}
		}
	}

	// Verify transform node is configured
	var transformNode *workflow.TransformNode
	for _, node := range wf.Nodes {
		if tn, ok := node.(*workflow.TransformNode); ok {
			transformNode = tn
			break
		}
	}

	if transformNode == nil {
		t.Fatal("Transform node not found")
	}
	if transformNode.Expression == "" {
		t.Error("Transform node has empty expression")
	}
	if transformNode.InputVariable == "" {
		t.Error("Transform node has empty input variable")
	}
	if transformNode.OutputVariable == "" {
		t.Error("Transform node has empty output variable")
	}

	// Verify workflow is valid
	if err := wf.Validate(); err != nil {
		t.Errorf("Template workflow failed validation: %v", err)
	}
}

// TestCreateAPIIntegrationTemplate tests the API integration workflow template
func TestCreateAPIIntegrationTemplate(t *testing.T) {
	wf := CreateAPIIntegrationTemplate()

	// Verify workflow created
	if wf == nil {
		t.Fatal("CreateAPIIntegrationTemplate() returned nil")
	}

	// Verify node count (6 nodes: Start, API Call, Check, Retry, Success, Failure)
	if len(wf.Nodes) != 6 {
		t.Errorf("Expected 6 nodes, got %d", len(wf.Nodes))
	}

	// Verify node types
	nodeTypes := make(map[string]int)
	for _, node := range wf.Nodes {
		nodeTypes[node.Type()]++
	}

	if nodeTypes["start"] != 1 {
		t.Errorf("Expected 1 start node, got %d", nodeTypes["start"])
	}
	if nodeTypes["mcp_tool"] != 1 {
		t.Errorf("Expected 1 mcp_tool node, got %d", nodeTypes["mcp_tool"])
	}
	if nodeTypes["condition"] != 1 {
		t.Errorf("Expected 1 condition node, got %d", nodeTypes["condition"])
	}
	if nodeTypes["loop"] != 1 {
		t.Errorf("Expected 1 loop node, got %d", nodeTypes["loop"])
	}
	if nodeTypes["end"] != 2 {
		t.Errorf("Expected 2 end nodes, got %d", nodeTypes["end"])
	}

	// Verify edge count (5 edges: includes conditional branches)
	if len(wf.Edges) != 5 {
		t.Errorf("Expected 5 edges, got %d", len(wf.Edges))
	}

	// Verify condition node has 2 outgoing edges with conditions
	conditionEdges := 0
	for _, edge := range wf.Edges {
		if edge.FromNodeID == "check-status" {
			conditionEdges++
			if edge.Condition == "" {
				t.Error("Condition edge missing condition label")
			}
		}
	}
	if conditionEdges != 2 {
		t.Errorf("Expected 2 edges from condition node, got %d", conditionEdges)
	}

	// Verify loop node is configured
	var loopNode *workflow.LoopNode
	for _, node := range wf.Nodes {
		if ln, ok := node.(*workflow.LoopNode); ok {
			loopNode = ln
			break
		}
	}

	if loopNode == nil {
		t.Fatal("Loop node not found")
	}
	if loopNode.Collection == "" {
		t.Error("Loop node has empty collection")
	}
	if loopNode.ItemVariable == "" {
		t.Error("Loop node has empty item variable")
	}
	if len(loopNode.Body) == 0 {
		t.Error("Loop node has empty body")
	}

	// Verify workflow is valid
	if err := wf.Validate(); err != nil {
		t.Errorf("Template workflow failed validation: %v", err)
	}
}

// TestTemplateRegistry tests the template registry maps
func TestTemplateRegistry(t *testing.T) {
	// Verify all templates are registered
	expectedTemplates := []string{"basic", "etl", "api-integration"}
	for _, name := range expectedTemplates {
		if _, exists := WorkflowTemplates[name]; !exists {
			t.Errorf("Template not registered: %s", name)
		}
		if _, exists := TemplateDescriptions[name]; !exists {
			t.Errorf("Template description missing: %s", name)
		}
	}

	// Verify all registered templates can be created
	for name, createFn := range WorkflowTemplates {
		wf := createFn()
		if wf == nil {
			t.Errorf("Template %s creation returned nil", name)
		}
		if err := wf.Validate(); err != nil {
			t.Errorf("Template %s failed validation: %v", name, err)
		}
	}

	// Verify descriptions are non-empty
	for name, desc := range TemplateDescriptions {
		if desc == "" {
			t.Errorf("Template %s has empty description", name)
		}
	}
}

// TestTemplateStructure verifies that each template has the expected structure
func TestTemplateStructure(t *testing.T) {
	tests := []struct {
		name           string
		template       func() *workflow.Workflow
		expectedNodes  int
		expectedEdges  int
		requiredTypes  []string
		shouldValidate bool
	}{
		{
			name:           "basic",
			template:       CreateBasicTemplate,
			expectedNodes:  3,
			expectedEdges:  2,
			requiredTypes:  []string{"start", "mcp_tool", "end"},
			shouldValidate: true,
		},
		{
			name:           "etl",
			template:       CreateETLTemplate,
			expectedNodes:  5,
			expectedEdges:  4,
			requiredTypes:  []string{"start", "mcp_tool", "transform", "end"},
			shouldValidate: true,
		},
		{
			name:           "api-integration",
			template:       CreateAPIIntegrationTemplate,
			expectedNodes:  6,
			expectedEdges:  5,
			requiredTypes:  []string{"start", "mcp_tool", "condition", "loop", "end"},
			shouldValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.template()

			// Check node count
			if len(wf.Nodes) != tt.expectedNodes {
				t.Errorf("Expected %d nodes, got %d", tt.expectedNodes, len(wf.Nodes))
			}

			// Check edge count
			if len(wf.Edges) != tt.expectedEdges {
				t.Errorf("Expected %d edges, got %d", tt.expectedEdges, len(wf.Edges))
			}

			// Check required node types are present
			nodeTypes := make(map[string]bool)
			for _, node := range wf.Nodes {
				nodeTypes[node.Type()] = true
			}

			for _, requiredType := range tt.requiredTypes {
				if !nodeTypes[requiredType] {
					t.Errorf("Missing required node type: %s", requiredType)
				}
			}

			// Check validation
			err := wf.Validate()
			if tt.shouldValidate && err != nil {
				t.Errorf("Expected valid workflow, got error: %v", err)
			}
			if !tt.shouldValidate && err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}
