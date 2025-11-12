package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowValidation_ValidWorkflow tests that a valid workflow passes validation
func TestWorkflowValidation_ValidWorkflow(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	// This should fail because workflow.Parse and Validate don't exist yet
	wf, err := workflow.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	err = wf.Validate()
	if err != nil {
		t.Errorf("Expected valid workflow to pass validation, got error: %v", err)
	}
}

// TestWorkflowValidation_CircularDependency tests detection of circular dependencies
func TestWorkflowValidation_CircularDependency(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/invalid-circular.yaml"
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
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	err = wf.Validate()
	if err == nil {
		t.Error("Expected circular dependency error, got nil")
	}

	// Check that the error message mentions circular dependency
	if err != nil && !containsString(err.Error(), "circular") && !containsString(err.Error(), "cycle") {
		t.Errorf("Expected error message to mention circular dependency or cycle, got: %v", err)
	}
}

// TestWorkflowValidation_OrphanedNode tests detection of orphaned nodes
func TestWorkflowValidation_OrphanedNode(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/invalid-orphaned.yaml"
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
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	err = wf.Validate()
	if err == nil {
		t.Error("Expected orphaned node error, got nil")
	}

	// Check that the error message mentions orphaned or unreachable
	if err != nil && !containsString(err.Error(), "orphan") && !containsString(err.Error(), "unreachable") {
		t.Errorf("Expected error message to mention orphaned or unreachable node, got: %v", err)
	}
}

// TestWorkflowValidation_InvalidEdgeReference tests detection of invalid edge references
func TestWorkflowValidation_InvalidEdgeReference(t *testing.T) {
	fixturePath := "../../internal/testutil/fixtures/invalid-missing-edge.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read fixture file: %v", err)
	}

	wf, err := workflow.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Validation should fail for invalid edge reference
	err = wf.Validate()
	if err == nil {
		t.Error("Expected validation to fail with invalid edge reference, got nil")
	}
}

// TestWorkflowValidation_StartNode tests validation of start node requirements
func TestWorkflowValidation_StartNode(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "no_start_node",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "node1"
    type: "mcp_tool"
    server: "test"
    tool: "echo"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "node1"
    to: "end"
`,
			expectError: true,
			errorMsg:    "must have exactly one start node",
		},
		{
			name: "multiple_start_nodes",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "start1"
    type: "start"
  - id: "start2"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start1"
    to: "end"
  - from: "start2"
    to: "end"
`,
			expectError: true,
			errorMsg:    "must have exactly one start node",
		},
		{
			name: "single_start_node",
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
    to: "end"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should fail because workflow.Parse doesn't exist yet
			wf, err := workflow.Parse([]byte(tt.yaml))
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			if err == nil {
				err = wf.Validate()

				if tt.expectError && err == nil {
					t.Errorf("Expected validation error containing '%s', got nil", tt.errorMsg)
				}

				if !tt.expectError && err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}

				if tt.expectError && err != nil && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// TestWorkflowValidation_DisconnectedGraph tests detection of disconnected graph components
func TestWorkflowValidation_DisconnectedGraph(t *testing.T) {
	yaml := `
version: "1.0"
name: "test"
servers:
  - id: "test"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "node1"
    type: "mcp_tool"
    server: "test"
    tool: "echo"
    output: "result1"
  - id: "node2"
    type: "mcp_tool"
    server: "test"
    tool: "echo"
    output: "result2"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "node1"
  # node2 is not connected to the main path
  - from: "node2"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	err = wf.Validate()
	if err == nil {
		t.Error("Expected disconnected graph error, got nil")
	}
}

// TestWorkflowValidation_VariableReferences tests validation of variable references
func TestWorkflowValidation_VariableReferences(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "undefined_variable_reference",
			yaml: `
version: "1.0"
name: "test"
servers:
  - id: "test"
    name: "test"
    command: "echo"
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "tool1"
    type: "mcp_tool"
    server: "test"
    tool: "echo"
    parameters:
      message: "${undefined_var}"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "tool1"
  - from: "tool1"
    to: "end"
`,
			expectError: true,
			errorMsg:    "undefined variable",
		},
		{
			name: "valid_variable_reference",
			yaml: `
version: "1.0"
name: "test"
variables:
  - name: "my_var"
    type: "string"
    default: "hello"
servers:
  - id: "test"
    name: "test"
    command: "echo"
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "tool1"
    type: "mcp_tool"
    server: "test"
    tool: "echo"
    parameters:
      message: "${my_var}"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "tool1"
  - from: "tool1"
    to: "end"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := workflow.Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Validation happens separately from parsing
			err = wf.Validate()

			if tt.expectError && err == nil {
				t.Errorf("Expected validation error containing '%s', got nil", tt.errorMsg)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}

			if tt.expectError && err != nil && !containsString(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error message containing '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

// TestWorkflowValidation_ServerReferences tests validation of server references
func TestWorkflowValidation_ServerReferences(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "undefined_server_reference",
			yaml: `
version: "1.0"
name: "test"
nodes:
  - id: "start"
    type: "start"
  - id: "tool1"
    type: "mcp_tool"
    server: "undefined_server"
    tool: "echo"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "tool1"
  - from: "tool1"
    to: "end"
`,
			expectError: true,
			errorMsg:    "undefined server",
		},
		{
			name: "valid_server_reference",
			yaml: `
version: "1.0"
name: "test"
servers:
  - id: "test-server"
    name: "test"
    command: "echo"
nodes:
  - id: "start"
    type: "start"
  - id: "tool1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "tool1"
  - from: "tool1"
    to: "end"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := workflow.Parse([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Validation happens separately from parsing
			err = wf.Validate()

			if tt.expectError && err == nil {
				t.Errorf("Expected validation error containing '%s', got nil", tt.errorMsg)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}

			if tt.expectError && err != nil && !containsString(err.Error(), tt.errorMsg) {
				t.Errorf("Expected error message containing '%s', got: %v", tt.errorMsg, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
