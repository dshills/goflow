package workflow

import (
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestStartNode tests StartNode creation and validation
func TestStartNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.StartNode
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid start node",
			node:    &workflow.StartNode{ID: "start-1"},
			wantErr: false,
		},
		{
			name:    "start node with empty ID",
			node:    &workflow.StartNode{ID: ""},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name:    "start node with valid ID format",
			node:    &workflow.StartNode{ID: "start_main"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("StartNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("StartNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "start" {
				t.Errorf("StartNode.Type() = %v, want %v", tt.node.Type(), "start")
			}
		})
	}
}

// TestEndNode tests EndNode creation and validation
func TestEndNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.EndNode
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid end node",
			node:    &workflow.EndNode{ID: "end-1"},
			wantErr: false,
		},
		{
			name:    "end node with return value",
			node:    &workflow.EndNode{ID: "end-1", ReturnValue: "${result}"},
			wantErr: false,
		},
		{
			name:    "end node with empty ID",
			node:    &workflow.EndNode{ID: ""},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name:    "end node with complex return expression",
			node:    &workflow.EndNode{ID: "end-1", ReturnValue: "${count > 0 ? 'success' : 'failure'}"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("EndNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("EndNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "end" {
				t.Errorf("EndNode.Type() = %v, want %v", tt.node.Type(), "end")
			}
		})
	}
}

// TestMCPToolNode tests MCPToolNode creation and validation
func TestMCPToolNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.MCPToolNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid MCP tool node",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "fs-server",
				ToolName:       "read_file",
				Parameters:     map[string]string{"path": "/tmp/file.txt"},
				OutputVariable: "file_content",
			},
			wantErr: false,
		},
		{
			name: "MCP tool node with empty ID",
			node: &workflow.MCPToolNode{
				ID:             "",
				ServerID:       "fs-server",
				ToolName:       "read_file",
				OutputVariable: "result",
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name: "MCP tool node with empty server ID",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "",
				ToolName:       "read_file",
				OutputVariable: "result",
			},
			wantErr: true,
			errMsg:  "empty server ID",
		},
		{
			name: "MCP tool node with empty tool name",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "fs-server",
				ToolName:       "",
				OutputVariable: "result",
			},
			wantErr: true,
			errMsg:  "empty tool name",
		},
		{
			name: "MCP tool node with empty output variable",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "fs-server",
				ToolName:       "read_file",
				OutputVariable: "",
			},
			wantErr: true,
			errMsg:  "empty output variable",
		},
		{
			name: "MCP tool node with expression parameters",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "api-server",
				ToolName:       "call_api",
				Parameters:     map[string]string{"url": "${base_url}/users", "method": "GET"},
				OutputVariable: "api_response",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("MCPToolNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("MCPToolNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "mcp_tool" {
				t.Errorf("MCPToolNode.Type() = %v, want %v", tt.node.Type(), "mcp_tool")
			}
		})
	}
}

// TestTransformNode tests TransformNode creation and validation
func TestTransformNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.TransformNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid transform node with JSONPath",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "api_response",
				Expression:     "$.users[0].email",
				OutputVariable: "user_email",
			},
			wantErr: false,
		},
		{
			name: "valid transform node with template",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "user",
				Expression:     "Hello ${user.name}",
				OutputVariable: "greeting",
			},
			wantErr: false,
		},
		{
			name: "transform node with empty ID",
			node: &workflow.TransformNode{
				ID:             "",
				InputVariable:  "input",
				Expression:     "$.data",
				OutputVariable: "output",
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name: "transform node with empty input variable",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "",
				Expression:     "$.data",
				OutputVariable: "output",
			},
			wantErr: true,
			errMsg:  "empty input variable",
		},
		{
			name: "transform node with empty expression",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "",
				OutputVariable: "output",
			},
			wantErr: true,
			errMsg:  "empty expression",
		},
		{
			name: "transform node with empty output variable",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "input",
				Expression:     "$.data",
				OutputVariable: "",
			},
			wantErr: true,
			errMsg:  "empty output variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("TransformNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("TransformNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "transform" {
				t.Errorf("TransformNode.Type() = %v, want %v", tt.node.Type(), "transform")
			}
		})
	}
}

// TestConditionNode tests ConditionNode creation and validation
func TestConditionNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.ConditionNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid condition node",
			node: &workflow.ConditionNode{
				ID:        "cond-1",
				Condition: "${count > 10}",
			},
			wantErr: false,
		},
		{
			name: "condition node with complex expression",
			node: &workflow.ConditionNode{
				ID:        "cond-1",
				Condition: "${status == 'active' && count > 0}",
			},
			wantErr: false,
		},
		{
			name: "condition node with empty ID",
			node: &workflow.ConditionNode{
				ID:        "",
				Condition: "${count > 10}",
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name: "condition node with empty condition",
			node: &workflow.ConditionNode{
				ID:        "cond-1",
				Condition: "",
			},
			wantErr: true,
			errMsg:  "empty condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ConditionNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ConditionNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "condition" {
				t.Errorf("ConditionNode.Type() = %v, want %v", tt.node.Type(), "condition")
			}
		})
	}
}

// TestParallelNode tests ParallelNode creation and validation
func TestParallelNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.ParallelNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid parallel node with wait_all",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1", "node-2"}, {"node-3", "node-4"}},
				MergeStrategy: "wait_all",
			},
			wantErr: false,
		},
		{
			name: "valid parallel node with wait_any",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1"}, {"node-2"}, {"node-3"}},
				MergeStrategy: "wait_any",
			},
			wantErr: false,
		},
		{
			name: "valid parallel node with wait_first",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1"}, {"node-2"}},
				MergeStrategy: "wait_first",
			},
			wantErr: false,
		},
		{
			name: "parallel node with empty ID",
			node: &workflow.ParallelNode{
				ID:            "",
				Branches:      [][]string{{"node-1"}},
				MergeStrategy: "wait_all",
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name: "parallel node with empty branches",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{},
				MergeStrategy: "wait_all",
			},
			wantErr: true,
			errMsg:  "empty branches",
		},
		{
			name: "parallel node with single branch",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1"}},
				MergeStrategy: "wait_all",
			},
			wantErr: true,
			errMsg:  "must have at least 2 branches",
		},
		{
			name: "parallel node with invalid merge strategy",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1"}, {"node-2"}},
				MergeStrategy: "invalid_strategy",
			},
			wantErr: true,
			errMsg:  "invalid merge strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParallelNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParallelNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "parallel" {
				t.Errorf("ParallelNode.Type() = %v, want %v", tt.node.Type(), "parallel")
			}
		})
	}
}

// TestLoopNode tests LoopNode creation and validation
func TestLoopNode(t *testing.T) {
	tests := []struct {
		name    string
		node    *workflow.LoopNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid loop node",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "items",
				ItemVariable: "item",
				Body:         []string{"process-node"},
			},
			wantErr: false,
		},
		{
			name: "loop node with break condition",
			node: &workflow.LoopNode{
				ID:             "loop-1",
				Collection:     "users",
				ItemVariable:   "user",
				Body:           []string{"validate-user", "process-user"},
				BreakCondition: "${user.status == 'inactive'}",
			},
			wantErr: false,
		},
		{
			name: "loop node with empty ID",
			node: &workflow.LoopNode{
				ID:           "",
				Collection:   "items",
				ItemVariable: "item",
				Body:         []string{"process-node"},
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
		{
			name: "loop node with empty collection",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "",
				ItemVariable: "item",
				Body:         []string{"process-node"},
			},
			wantErr: true,
			errMsg:  "empty collection",
		},
		{
			name: "loop node with empty item variable",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "items",
				ItemVariable: "",
				Body:         []string{"process-node"},
			},
			wantErr: true,
			errMsg:  "empty item variable",
		},
		{
			name: "loop node with empty body",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "items",
				ItemVariable: "item",
				Body:         []string{},
			},
			wantErr: true,
			errMsg:  "empty body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoopNode.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("LoopNode.Validate() unexpected error: %v", err)
				}
			}

			// Test type identification
			if tt.node.Type() != "loop" {
				t.Errorf("LoopNode.Type() = %v, want %v", tt.node.Type(), "loop")
			}
		})
	}
}

// TestNode_Serialization tests node marshaling and unmarshaling
func TestNode_Serialization(t *testing.T) {
	tests := []struct {
		name string
		node workflow.Node
	}{
		{
			name: "start node serialization",
			node: &workflow.StartNode{ID: "start"},
		},
		{
			name: "end node serialization",
			node: &workflow.EndNode{ID: "end", ReturnValue: "${result}"},
		},
		{
			name: "MCP tool node serialization",
			node: &workflow.MCPToolNode{
				ID:             "tool-1",
				ServerID:       "fs-server",
				ToolName:       "read_file",
				Parameters:     map[string]string{"path": "/tmp/file.txt"},
				OutputVariable: "content",
			},
		},
		{
			name: "transform node serialization",
			node: &workflow.TransformNode{
				ID:             "transform-1",
				InputVariable:  "data",
				Expression:     "$.users[0]",
				OutputVariable: "first_user",
			},
		},
		{
			name: "condition node serialization",
			node: &workflow.ConditionNode{
				ID:        "cond-1",
				Condition: "${count > 10}",
			},
		},
		{
			name: "parallel node serialization",
			node: &workflow.ParallelNode{
				ID:            "parallel-1",
				Branches:      [][]string{{"node-1"}, {"node-2"}},
				MergeStrategy: "wait_all",
			},
		},
		{
			name: "loop node serialization",
			node: &workflow.LoopNode{
				ID:           "loop-1",
				Collection:   "items",
				ItemVariable: "item",
				Body:         []string{"process"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := tt.node.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error: %v", err)
			}

			// Unmarshal back
			newNode, err := workflow.UnmarshalNode(data)
			if err != nil {
				t.Fatalf("UnmarshalNode() error: %v", err)
			}

			// Verify type matches
			if newNode.Type() != tt.node.Type() {
				t.Errorf("Type mismatch after serialization: got %v, want %v", newNode.Type(), tt.node.Type())
			}

			// Verify ID matches
			if newNode.GetID() != tt.node.GetID() {
				t.Errorf("ID mismatch after serialization: got %v, want %v", newNode.GetID(), tt.node.GetID())
			}
		})
	}
}

// TestNode_UniqueIDs tests that node IDs are enforced to be unique
func TestNode_UniqueIDs(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")

	// Add first node
	err := wf.AddNode(&workflow.StartNode{ID: "node-1"})
	if err != nil {
		t.Fatalf("AddNode() first node unexpected error: %v", err)
	}

	// Add second node with same ID (should be allowed during construction)
	err = wf.AddNode(&workflow.EndNode{ID: "node-1"})
	if err != nil {
		t.Fatalf("AddNode() second node unexpected error: %v", err)
	}

	// Validate should catch the duplicate
	err = wf.Validate()
	if err == nil {
		t.Error("Validate() should fail for duplicate node IDs")
	}
	if err != nil && !strings.Contains(err.Error(), "duplicate node ID") {
		t.Errorf("Validate() error should mention duplicate node ID, got: %v", err)
	}
}
