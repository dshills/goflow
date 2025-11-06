package workflow

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestEdge_Validation tests basic Edge validation rules
func TestEdge_Validation(t *testing.T) {
	tests := []struct {
		name    string
		edge    *workflow.Edge
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid edge",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "end",
			},
			wantErr: false,
		},
		{
			name: "edge with label",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "tool-1",
				Label:      "main path",
			},
			wantErr: false,
		},
		{
			name: "edge with empty ID",
			edge: &workflow.Edge{
				ID:         "",
				FromNodeID: "start",
				ToNodeID:   "end",
			},
			wantErr: true,
			errMsg:  "empty edge ID",
		},
		{
			name: "edge with empty from node",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "",
				ToNodeID:   "end",
			},
			wantErr: true,
			errMsg:  "empty from node",
		},
		{
			name: "edge with empty to node",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "",
			},
			wantErr: true,
			errMsg:  "empty to node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Edge.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Edge.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEdge_NoSelfLoops tests that edges cannot create self-loops
func TestEdge_NoSelfLoops(t *testing.T) {
	tests := []struct {
		name    string
		edge    *workflow.Edge
		wantErr bool
	}{
		{
			name: "self-loop should fail",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "node-1",
				ToNodeID:   "node-1",
			},
			wantErr: true,
		},
		{
			name: "different nodes should pass",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "node-1",
				ToNodeID:   "node-2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()

			if tt.wantErr && err == nil {
				t.Error("Edge.Validate() expected error for self-loop but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Edge.Validate() unexpected error: %v", err)
			}
		})
	}
}

// TestEdge_NoDuplicates tests that duplicate edges are not allowed
func TestEdge_NoDuplicates(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	// Add first edge
	err := wf.AddEdge(&workflow.Edge{
		ID:         "edge-1",
		FromNodeID: "start",
		ToNodeID:   "tool-1",
	})
	if err != nil {
		t.Fatalf("AddEdge() first edge unexpected error: %v", err)
	}

	// Try to add duplicate edge (same from/to pair)
	err = wf.AddEdge(&workflow.Edge{
		ID:         "edge-2",
		FromNodeID: "start",
		ToNodeID:   "tool-1",
	})
	if err == nil {
		t.Error("AddEdge() should fail for duplicate edge")
	}
}

// TestEdge_WithCondition tests edges with conditions for branching
func TestEdge_WithCondition(t *testing.T) {
	tests := []struct {
		name    string
		edge    *workflow.Edge
		wantErr bool
		errMsg  string
	}{
		{
			name: "edge with valid condition",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "cond-1",
				ToNodeID:   "tool-1",
				Condition:  "${result == true}",
				Label:      "true branch",
			},
			wantErr: false,
		},
		{
			name: "edge with complex condition",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "cond-1",
				ToNodeID:   "tool-1",
				Condition:  "${count > 10 && status == 'active'}",
				Label:      "success path",
			},
			wantErr: false,
		},
		{
			name: "edge without condition from normal node",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "tool-1",
				ToNodeID:   "tool-2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Edge.Validate() expected error containing %q but got none", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Edge.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEdge_ConditionNodeRequirements tests that ConditionNodes require specific edge setup
func TestEdge_ConditionNodeRequirements(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "condition node with two branches should pass",
			setupGraph: func(wf *workflow.Workflow) {
				// Add variables used in conditions
				wf.AddVariable(&workflow.Variable{Name: "count", Type: "number"})
				wf.AddVariable(&workflow.Variable{Name: "result", Type: "boolean"})

				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.ConditionNode{ID: "cond-1", Condition: "count > 10"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "cond-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-1", Condition: "result == true", Label: "true"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-2", Condition: "result == false", Label: "false"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
			},
			wantErr: false,
		},
		{
			name: "condition node with one branch should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.ConditionNode{ID: "cond-1", Condition: "count > 10"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "cond-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-1", Condition: "result == true"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "condition node must have exactly 2 outgoing edges",
		},
		{
			name: "condition node with three branches should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.ConditionNode{ID: "cond-1", Condition: "count > 10"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "cond-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-1", Condition: "result == 1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-2", Condition: "result == 2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-3", Condition: "result == 3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "condition node must have exactly 2 outgoing edges",
		},
		{
			name: "condition node edges without conditions should fail",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.ConditionNode{ID: "cond-1", Condition: "count > 10"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "cond-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-1"}) // missing condition
				wf.AddEdge(&workflow.Edge{FromNodeID: "cond-1", ToNodeID: "tool-2"}) // missing condition
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "edges from condition node must have conditions",
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

// TestEdge_CircularDependencyDetection tests cycle detection algorithms
func TestEdge_CircularDependencyDetection(t *testing.T) {
	tests := []struct {
		name       string
		setupGraph func(*workflow.Workflow)
		wantErr    bool
		errMsg     string
	}{
		{
			name: "simple two-node cycle",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-1"}) // cycle
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "three-node cycle",
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
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "cycle not involving start node",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-4"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "tool-4"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-4", ToNodeID: "tool-2"}) // cycle in middle
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-4", ToNodeID: "end"})
			},
			wantErr: true,
			errMsg:  "circular dependency",
		},
		{
			name: "complex DAG without cycles should pass",
			setupGraph: func(wf *workflow.Workflow) {
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-4"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-5"})
				wf.AddNode(&workflow.EndNode{ID: "end"})

				// Diamond + merge pattern
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-3"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-4"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "tool-4"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-4", ToNodeID: "tool-5"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-5", ToNodeID: "end"})
			},
			wantErr: false,
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

// TestEdge_RemoveNode_RemovesDependentEdges tests that removing a node removes its edges
func TestEdge_RemoveNode_RemovesDependentEdges(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test-workflow", "")
	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
	wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "tool-1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "tool-1", ToNodeID: "tool-2"})
	wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "tool-2", ToNodeID: "end"})

	// Verify initial edge count
	if len(wf.Edges) != 3 {
		t.Fatalf("Expected 3 edges initially, got %d", len(wf.Edges))
	}

	// Remove middle node
	err := wf.RemoveNode("tool-1")
	if err != nil {
		t.Fatalf("RemoveNode() unexpected error: %v", err)
	}

	// Verify edges involving tool-1 are removed
	if len(wf.Edges) != 1 {
		t.Errorf("Expected 1 edge after removing tool-1, got %d", len(wf.Edges))
	}

	// Verify remaining edge is the one not involving tool-1
	if wf.Edges[0].ID != "e3" {
		t.Errorf("Expected edge e3 to remain, got %v", wf.Edges[0].ID)
	}
}

// TestEdge_Serialization tests edge marshaling and unmarshaling
func TestEdge_Serialization(t *testing.T) {
	tests := []struct {
		name string
		edge *workflow.Edge
	}{
		{
			name: "simple edge",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "end",
			},
		},
		{
			name: "edge with condition",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "cond-1",
				ToNodeID:   "tool-1",
				Condition:  "${result == true}",
				Label:      "true branch",
			},
		},
		{
			name: "edge with label only",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "tool-1",
				Label:      "main path",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := tt.edge.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error: %v", err)
			}

			// Unmarshal back
			newEdge := &workflow.Edge{}
			err = newEdge.UnmarshalJSON(data)
			if err != nil {
				t.Fatalf("UnmarshalJSON() error: %v", err)
			}

			// Verify fields match
			if newEdge.ID != tt.edge.ID {
				t.Errorf("ID mismatch: got %v, want %v", newEdge.ID, tt.edge.ID)
			}
			if newEdge.FromNodeID != tt.edge.FromNodeID {
				t.Errorf("FromNodeID mismatch: got %v, want %v", newEdge.FromNodeID, tt.edge.FromNodeID)
			}
			if newEdge.ToNodeID != tt.edge.ToNodeID {
				t.Errorf("ToNodeID mismatch: got %v, want %v", newEdge.ToNodeID, tt.edge.ToNodeID)
			}
			if newEdge.Condition != tt.edge.Condition {
				t.Errorf("Condition mismatch: got %v, want %v", newEdge.Condition, tt.edge.Condition)
			}
			if newEdge.Label != tt.edge.Label {
				t.Errorf("Label mismatch: got %v, want %v", newEdge.Label, tt.edge.Label)
			}
		})
	}
}
