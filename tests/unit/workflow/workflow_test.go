package workflow

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestNewWorkflow tests creation of new workflow instances
func TestNewWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		description  string
		wantErr      bool
	}{
		{
			name:         "valid workflow with name and description",
			workflowName: "data-pipeline",
			description:  "ETL workflow for data processing",
			wantErr:      false,
		},
		{
			name:         "valid workflow with only name",
			workflowName: "simple-workflow",
			description:  "",
			wantErr:      false,
		},
		{
			name:         "empty workflow name should fail",
			workflowName: "",
			description:  "Some description",
			wantErr:      true,
		},
		{
			name:         "workflow name with special characters",
			workflowName: "data-pipeline-v2.0",
			description:  "Version 2 pipeline",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := workflow.NewWorkflow(tt.workflowName, tt.description)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewWorkflow() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewWorkflow() unexpected error: %v", err)
				return
			}

			if wf == nil {
				t.Fatal("NewWorkflow() returned nil workflow")
			}

			if wf.Name != tt.workflowName {
				t.Errorf("NewWorkflow() name = %v, want %v", wf.Name, tt.workflowName)
			}

			if wf.Description != tt.description {
				t.Errorf("NewWorkflow() description = %v, want %v", wf.Description, tt.description)
			}

			if wf.ID == "" {
				t.Error("NewWorkflow() did not generate ID")
			}

			if wf.Version == "" {
				t.Error("NewWorkflow() did not set version")
			}
		})
	}
}

// TestWorkflow_Invariant_ExactlyOneStartNode tests that a workflow must have exactly one Start node
func TestWorkflow_Invariant_ExactlyOneStartNode(t *testing.T) {
	tests := []struct {
		name       string
		setupNodes func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "no start node should fail",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.EndNode{ID: "end-1"})
			},
			wantErr: true,
			errMsg:  "must have exactly one start node",
		},
		{
			name: "exactly one start node should pass",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start-1"})
				wf.AddNode(&workflow.EndNode{ID: "end-1"})
			},
			wantErr: false,
		},
		{
			name: "multiple start nodes should fail",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start-1"})
				wf.AddNode(&workflow.StartNode{ID: "start-2"})
				wf.AddNode(&workflow.EndNode{ID: "end-1"})
			},
			wantErr: true,
			errMsg:  "must have exactly one start node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupNodes(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_AtLeastOneEndNode tests that a workflow must have at least one End node
func TestWorkflow_Invariant_AtLeastOneEndNode(t *testing.T) {
	tests := []struct {
		name       string
		setupNodes func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "no end node should fail",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start-1"})
			},
			wantErr: true,
			errMsg:  "must have at least one end node",
		},
		{
			name: "one end node should pass",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start-1"})
				wf.AddNode(&workflow.EndNode{ID: "end-1"})
			},
			wantErr: false,
		},
		{
			name: "multiple end nodes should pass",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start-1"})
				wf.AddNode(&workflow.EndNode{ID: "end-1"})
				wf.AddNode(&workflow.EndNode{ID: "end-2"})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupNodes(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_NoCircularDependencies tests DAG property
func TestWorkflow_Invariant_NoCircularDependencies(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "linear graph should pass",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
			},
			wantErr: false,
		},
		{
			name: "simple cycle should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-1"}) // cycle
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "self-loop should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-1"}) // self-loop
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "diamond graph should pass",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "end"})
			},
			wantErr: false,
		},
		{
			name: "complex cycle should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "tool-1"}) // cycle back
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupGraph(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_UniqueNodeIDs tests that all node IDs must be unique
func TestWorkflow_Invariant_UniqueNodeIDs(t *testing.T) {
	tests := []struct {
		name       string
		setupNodes func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "unique node IDs should pass",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
			},
			wantErr: false,
		},
		{
			name: "duplicate node IDs should fail",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"}) // duplicate
				wf.AddNode(&workflow.EndNode{ID: "end"})
			},
			wantErr: true,
			errMsg:  "duplicate node ID",
		},
		{
			name: "empty node ID should fail",
			setupNodes: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: ""}) // empty ID
				wf.AddNode(&workflow.EndNode{ID: "end"})
			},
			wantErr: true,
			errMsg:  "empty node ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupNodes(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_UniqueVariableNames tests that all variable names must be unique
func TestWorkflow_Invariant_UniqueVariableNames(t *testing.T) {
	tests := []struct {
		name           string
		setupVariables func(*workflow.Workflow)
		wantErr        bool
		errMsg         string
	}{
		{
			name: "unique variable names should pass",
			setupVariables: func(wf *workflow.Workflow) {
				wf.AddVariable(&workflow.Variable{Name: "input_file"})
				wf.AddVariable(&workflow.Variable{Name: "output_file"})
				wf.AddVariable(&workflow.Variable{Name: "temp_data"})
			},
			wantErr: false,
		},
		{
			name: "duplicate variable names should fail",
			setupVariables: func(wf *workflow.Workflow) {
				wf.AddVariable(&workflow.Variable{Name: "data"})
				wf.AddVariable(&workflow.Variable{Name: "result"})
				wf.AddVariable(&workflow.Variable{Name: "data"}) // duplicate
			},
			wantErr: true,
			errMsg:  "duplicate variable name",
		},
		{
			name: "empty variable name should fail",
			setupVariables: func(wf *workflow.Workflow) {
				wf.AddVariable(&workflow.Variable{Name: "valid_name"})
				wf.AddVariable(&workflow.Variable{Name: ""}) // empty
			},
			wantErr: true,
			errMsg:  "empty variable name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			wf.AddNode(&workflow.StartNode{ID: "start"})
			wf.AddNode(&workflow.EndNode{ID: "end"})
			tt.setupVariables(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_ValidEdgeReferences tests that edges reference valid nodes
func TestWorkflow_Invariant_ValidEdgeReferences(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid edge references should pass",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
			},
			wantErr: false,
		},
		{
			name: "edge with invalid from node should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "nonexistent", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "invalid node reference",
		},
		{
			name: "edge with invalid to node should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "nonexistent"})
			},
			wantErr: true,
			errMsg:  "invalid node reference",
		},
		{
			name: "edge with both invalid nodes should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "invalid-from", ToNodeID: "invalid-to"})
			},
			wantErr: true,
			errMsg:  "invalid node reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupGraph(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_Invariant_NoOrphanedNodes tests that all nodes are reachable from Start
func TestWorkflow_Invariant_NoOrphanedNodes(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "all nodes connected should pass",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
			},
			wantErr: false,
		},
		{
			name: "orphaned node should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-orphan"}) // not connected
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "orphaned node",
		},
		{
			name: "disconnected subgraph should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				// Main graph
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				// Disconnected subgraph
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-3"})
			},
			wantErr: true,
			errMsg:  "orphaned node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			tt.setupGraph(wf)

			err := wf.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestWorkflow_AddNode tests adding nodes to workflow
func TestWorkflow_AddNode(t *testing.T) {
	tests := []struct {
		name    string
		node    workflow.Node
		wantErr bool
	}{
		{
			name:    "add start node",
			node:    &workflow.StartNode{ID: "start"},
			wantErr: false,
		},
		{
			name:    "add end node",
			node:    &workflow.EndNode{ID: "end"},
			wantErr: false,
		},
		{
			name:    "add MCP tool node",
			node:    &workflow.MCPToolNode{ID: "tool-1", ServerID: "fs", ToolName: "read_file"},
			wantErr: false,
		},
		{
			name:    "add transform node",
			node:    &workflow.TransformNode{ID: "transform-1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test-workflow", "")
			err := wf.AddNode(tt.node)

			if tt.wantErr && err == nil {
				t.Error("AddNode() expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("AddNode() unexpected error: %v", err)
			}
		})
	}
}

// TestWorkflow_RemoveNode tests removing nodes from workflow
func TestWorkflow_RemoveNode(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
	wf.AddNode(&workflow.EndNode{ID: "end"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})

	err := wf.RemoveNode("tool-1")
	if err != nil {
		t.Errorf("RemoveNode() unexpected error: %v", err)
	}

	// Verify dependent edges also removed
	if len(wf.Edges) != 0 {
		t.Errorf("RemoveNode() should remove dependent edges, got %d edges remaining", len(wf.Edges))
	}
}
