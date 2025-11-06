package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowParse_ValidSimpleWorkflow tests parsing a valid simple workflow
func TestWorkflowParse_ValidSimpleWorkflow(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse(data)
	if err != nil {
		t.Fatalf("Expected successful parse, got error: %v", err)
	}

	// Validate parsed workflow structure
	if wf.Name != "simple-read-transform-write" {
		t.Errorf("Expected name 'simple-read-transform-write', got '%s'", wf.Name)
	}

	if wf.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", wf.Version)
	}

	// Check nodes
	expectedNodeCount := 5 // start, read_file, transform, write_file, end
	if len(wf.Nodes) != expectedNodeCount {
		t.Errorf("Expected %d nodes, got %d", expectedNodeCount, len(wf.Nodes))
	}

	// Check edges
	expectedEdgeCount := 4
	if len(wf.Edges) != expectedEdgeCount {
		t.Errorf("Expected %d edges, got %d", expectedEdgeCount, len(wf.Edges))
	}

	// Check variables
	expectedVarCount := 4
	if len(wf.Variables) != expectedVarCount {
		t.Errorf("Expected %d variables, got %d", expectedVarCount, len(wf.Variables))
	}

	// Check servers
	expectedServerCount := 1
	if len(wf.ServerConfigs) != expectedServerCount {
		t.Errorf("Expected %d servers, got %d", expectedServerCount, len(wf.ServerConfigs))
	}
}

// TestWorkflowParse_AllNodeTypes tests that all node types parse correctly
func TestWorkflowParse_AllNodeTypes(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		expectedType string
		nodeID       string
	}{
		{
			name: "start_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "start"
    type: "start"
`,
			expectedType: "start",
			nodeID:       "start",
		},
		{
			name: "mcp_tool_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "tool1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "hello"
    output: "result"
`,
			expectedType: "mcp_tool",
			nodeID:       "tool1",
		},
		{
			name: "transform_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "transform1"
    type: "transform"
    input: "${data}"
    expression: "${input} | upper"
    output: "result"
`,
			expectedType: "transform",
			nodeID:       "transform1",
		},
		{
			name: "condition_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "condition1"
    type: "condition"
    condition: "${value} > 10"
`,
			expectedType: "condition",
			nodeID:       "condition1",
		},
		{
			name: "end_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "end"
    type: "end"
    return: "${result}"
`,
			expectedType: "end",
			nodeID:       "end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should fail because workflow.Parse doesn't exist yet
			wf, err := workflow.Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			if len(wf.Nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(wf.Nodes))
			}

			node := wf.Nodes[0]
			if node.GetID() != tt.nodeID {
				t.Errorf("Expected node ID '%s', got '%s'", tt.nodeID, node.GetID())
			}

			if node.Type() != tt.expectedType {
				t.Errorf("Expected node type '%s', got '%s'", tt.expectedType, node.Type())
			}
		})
	}
}

// TestWorkflowParse_InvalidYAML tests error handling for invalid YAML
func TestWorkflowParse_InvalidYAML(t *testing.T) {
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
			name:        "malformed_yaml",
			yaml:        "this is not: valid: yaml: syntax",
			expectError: true,
		},
		{
			name: "missing_version",
			yaml: `
name: "test"
nodes:
  - id: "start"
    type: "start"
`,
			expectError: true,
		},
		{
			name: "missing_name",
			yaml: `
version: "1.0"
nodes:
  - id: "start"
    type: "start"
`,
			expectError: true,
		},
		{
			name: "missing_nodes",
			yaml: `
version: "1.0"
name: "test"
`,
			expectError: true,
		},
		{
			name: "invalid_node_type",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "invalid"
    type: "unknown_type"
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should fail because workflow.Parse doesn't exist yet
			_, err := workflow.Parse([]byte(tt.yaml))

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestWorkflowParse_ValidationDuringParsing tests that basic validation happens during parsing
func TestWorkflowParse_ValidationDuringParsing(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "duplicate_node_ids",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "node1"
    type: "start"
  - id: "node1"
    type: "end"
`,
			expectError: true,
			errorMsg:    "duplicate node ID",
		},
		{
			name: "duplicate_variable_names",
			yaml: `
version: "1.0"
name: "test"
variables:
  - name: "var1"
    type: "string"
  - name: "var1"
    type: "integer"
nodes:
  - id: "start"
    type: "start"
`,
			expectError: true,
			errorMsg:    "duplicate variable name",
		},
		{
			name: "invalid_edge_from",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "nonexistent"
    to: "end"
`,
			expectError: true,
			errorMsg:    "edge references non-existent node",
		},
		{
			name: "invalid_edge_to",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "nonexistent"
`,
			expectError: true,
			errorMsg:    "edge references non-existent node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should fail because workflow.Parse doesn't exist yet
			_, err := workflow.Parse([]byte(tt.yaml))

			if tt.expectError && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestWorkflowParse_FromFile tests parsing workflow from a file
func TestWorkflowParse_FromFile(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// This should fail because workflow.ParseFile doesn't exist yet
	wf, err := workflow.ParseFile(absPath)
	if err != nil {
		t.Fatalf("Expected successful parse, got error: %v", err)
	}

	if wf.Name != "simple-read-transform-write" {
		t.Errorf("Expected name 'simple-read-transform-write', got '%s'", wf.Name)
	}
}

// TestWorkflowParse_NonExistentFile tests error handling for non-existent files
func TestWorkflowParse_NonExistentFile(t *testing.T) {
	// This should fail because workflow.ParseFile doesn't exist yet
	_, err := workflow.ParseFile("/nonexistent/path/workflow.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
