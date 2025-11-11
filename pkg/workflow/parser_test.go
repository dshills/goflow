package workflow

import (
	"testing"
)

func TestParse_SimpleWorkflow(t *testing.T) {
	yaml := `version: "1.0"
name: "test"
servers:
  - id: "test-server"
    command: "echo"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if wf.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", wf.Name)
	}

	if len(wf.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(wf.Nodes))
	}

	if len(wf.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(wf.Edges))
	}
}

func TestParse_AllNodeTypes(t *testing.T) {
	yaml := `version: "1.0"
name: "test"
servers:
  - id: "test-server"
    command: "echo"
nodes:
  - id: "start"
    type: "start"
  - id: "tool1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    output: "result"
  - id: "transform1"
    type: "transform"
    input: "result"
    expression: "upper"
    output: "output"
  - id: "condition1"
    type: "condition"
    condition: "true"
  - id: "parallel1"
    type: "parallel"
    branches:
      - ["branch1"]
      - ["branch2"]
  - id: "loop1"
    type: "loop"
    collection: "items"
    item: "item"
    body: ["body1"]
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "tool1"
  - from: "tool1"
    to: "transform1"
  - from: "transform1"
    to: "condition1"
  - from: "condition1"
    to: "parallel1"
    condition: "true"
  - from: "condition1"
    to: "loop1"
    condition: "false"
  - from: "parallel1"
    to: "end"
  - from: "loop1"
    to: "end"
`
	wf, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(wf.Nodes) != 7 {
		t.Errorf("Expected 7 nodes, got %d", len(wf.Nodes))
	}

	// Check node types
	expectedTypes := map[string]string{
		"start":      "start",
		"tool1":      "mcp_tool",
		"transform1": "transform",
		"condition1": "condition",
		"parallel1":  "parallel",
		"loop1":      "loop",
		"end":        "end",
	}

	for _, node := range wf.Nodes {
		nodeID := node.GetID()
		expectedType, ok := expectedTypes[nodeID]
		if !ok {
			t.Errorf("Unexpected node ID: %s", nodeID)
			continue
		}
		if node.Type() != expectedType {
			t.Errorf("Node %s: expected type '%s', got '%s'", nodeID, expectedType, node.Type())
		}
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{
			name:        "empty_yaml",
			yaml:        "",
			expectError: true,
		},
		{
			name:        "missing_version",
			yaml:        "name: test\nnodes:\n  - id: start\n    type: start\n",
			expectError: true,
		},
		{
			name:        "missing_name",
			yaml:        "version: '1.0'\nnodes:\n  - id: start\n    type: start\n",
			expectError: true,
		},
		{
			name: "unknown_node_type",
			yaml: `version: "1.0"
name: "test"
servers:
  - id: "test"
    command: "echo"
nodes:
  - id: "invalid"
    type: "unknown_type"
edges: []
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestToYAML_RoundTrip(t *testing.T) {
	originalYAML := `version: "1.0"
name: "test"
servers:
  - id: "test-server"
    command: "echo"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	wf, err := Parse([]byte(originalYAML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	yamlBytes, err := ToYAML(wf)
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	wf2, err := Parse(yamlBytes)
	if err != nil {
		t.Fatalf("Re-parse failed: %v", err)
	}

	if wf.Name != wf2.Name {
		t.Errorf("Name mismatch: %s != %s", wf.Name, wf2.Name)
	}
	if len(wf.Nodes) != len(wf2.Nodes) {
		t.Errorf("Node count mismatch: %d != %d", len(wf.Nodes), len(wf2.Nodes))
	}
	if len(wf.Edges) != len(wf2.Edges) {
		t.Errorf("Edge count mismatch: %d != %d", len(wf.Edges), len(wf2.Edges))
	}
}

func TestTopologicalSort_Simple(t *testing.T) {
	wf, err := NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("NewWorkflow failed: %v", err)
	}

	// Add nodes
	start := &StartNode{ID: "start"}
	node1 := &MCPToolNode{ID: "node1", ServerID: "test", ToolName: "tool", OutputVariable: "out"}
	end := &EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node1)
	wf.AddNode(end)

	// Add edges
	wf.AddEdge(&Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node1"})
	wf.AddEdge(&Edge{ID: "e2", FromNodeID: "node1", ToNodeID: "end"})

	sorted, err := TopologicalSort(wf)
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(sorted))
	}

	// Check order
	expectedOrder := []string{"start", "node1", "end"}
	for i, nodeID := range sorted {
		if string(nodeID) != expectedOrder[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expectedOrder[i], nodeID)
		}
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	wf, err := NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("NewWorkflow failed: %v", err)
	}

	// Create cycle: node1 -> node2 -> node3 -> node1
	node1 := &MCPToolNode{ID: "node1", ServerID: "test", ToolName: "tool", OutputVariable: "out1"}
	node2 := &MCPToolNode{ID: "node2", ServerID: "test", ToolName: "tool", OutputVariable: "out2"}
	node3 := &MCPToolNode{ID: "node3", ServerID: "test", ToolName: "tool", OutputVariable: "out3"}

	wf.AddNode(node1)
	wf.AddNode(node2)
	wf.AddNode(node3)

	wf.AddEdge(&Edge{ID: "e1", FromNodeID: "node1", ToNodeID: "node2"})
	wf.AddEdge(&Edge{ID: "e2", FromNodeID: "node2", ToNodeID: "node3"})
	wf.AddEdge(&Edge{ID: "e3", FromNodeID: "node3", ToNodeID: "node1"})

	_, err = TopologicalSort(wf)
	if err == nil {
		t.Error("Expected cycle detection error, got nil")
	}
}
