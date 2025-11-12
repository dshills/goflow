package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestValidateWorkflow_CircularDependency tests cycle detection
func TestValidateWorkflow_CircularDependency(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *workflow.Workflow
		expectError bool
		errorType   string
	}{
		{
			name: "simple cycle A->B->A",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "a"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "a", ToNodeID: "b"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "b", ToNodeID: "a"}) // Cycle
				return wf
			}(),
			expectError: true,
			errorType:   "circular_dependency",
		},
		{
			name: "complex cycle A->B->C->A",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				nodeC := &workflow.PassthroughNode{ID: "c"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(nodeC)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "a"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "a", ToNodeID: "b"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "b", ToNodeID: "c"})
				wf.AddEdge(&workflow.Edge{ID: "e4", FromNodeID: "c", ToNodeID: "a"}) // Cycle
				return wf
			}(),
			expectError: true,
			errorType:   "circular_dependency",
		},
		{
			name: "no cycle - linear workflow",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "a"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "a", ToNodeID: "b"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "b", ToNodeID: "end"})
				return wf
			}(),
			expectError: false,
		},
		{
			name: "no cycle - branching workflow",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				cond := &workflow.ConditionNode{ID: "cond", Condition: "true"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(cond)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "cond"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "cond", ToNodeID: "a", Condition: "true"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "cond", ToNodeID: "b", Condition: "false"})
				wf.AddEdge(&workflow.Edge{ID: "e4", FromNodeID: "a", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{ID: "e5", FromNodeID: "b", ToNodeID: "end"})
				return wf
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ValidateWorkflow(tt.workflow)

			if tt.expectError {
				if !status.HasErrors() {
					t.Errorf("Expected error but got none")
				}

				// Check that the error type matches
				errors := status.GetErrors()
				found := false
				for _, err := range errors {
					if err.ErrorType == tt.errorType {
						found = true
						// Verify cycle path is included in message
						if !strings.Contains(err.Message, "→") {
							t.Errorf("Expected cycle path in error message, got: %s", err.Message)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s, but not found in errors: %v", tt.errorType, errors)
				}
			} else {
				if status.HasErrors() {
					errors := status.GetErrors()
					// Filter out errors that are not circular_dependency
					for _, err := range errors {
						if err.ErrorType == "circular_dependency" {
							t.Errorf("Expected no circular dependency error, but got: %s", err.Message)
						}
					}
				}
			}
		})
	}
}

// TestValidateWorkflow_Reachability tests reachability checking
func TestValidateWorkflow_Reachability(t *testing.T) {
	tests := []struct {
		name              string
		workflow          *workflow.Workflow
		expectWarnings    bool
		unreachableNodeID string
	}{
		{
			name: "all nodes reachable",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(nodeA)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "a"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "a", ToNodeID: "end"})
				return wf
			}(),
			expectWarnings: false,
		},
		{
			name: "disconnected node",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"} // Disconnected
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "a"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "a", ToNodeID: "end"})
				// nodeB is not connected
				return wf
			}(),
			expectWarnings:    true,
			unreachableNodeID: "b",
		},
		{
			name: "no start node",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				nodeA := &workflow.PassthroughNode{ID: "a"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(nodeA)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "a", ToNodeID: "end"})
				return wf
			}(),
			expectWarnings: false, // Should error, not warn
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ValidateWorkflow(tt.workflow)

			if tt.expectWarnings {
				if !status.HasWarnings() {
					t.Errorf("Expected warnings but got none")
				}

				warnings := status.GetWarnings()
				found := false
				for _, warn := range warnings {
					if warn.NodeID == tt.unreachableNodeID {
						found = true
						if !strings.Contains(warn.Message, "not reachable") {
							t.Errorf("Expected 'not reachable' in warning message, got: %s", warn.Message)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected warning for node %s, but not found in warnings: %v", tt.unreachableNodeID, warnings)
				}
			}
		})
	}
}

// TestValidateNode_RequiredFields tests required field validation
func TestValidateNode_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		node        workflow.Node
		expectError bool
		errorType   string
		fieldName   string
	}{
		{
			name:        "MCP tool node missing server_id",
			node:        &workflow.MCPToolNode{ID: "tool1", ToolName: "test", OutputVariable: "out"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "server_id",
		},
		{
			name:        "MCP tool node missing tool_name",
			node:        &workflow.MCPToolNode{ID: "tool1", ServerID: "srv1", OutputVariable: "out"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "tool_name",
		},
		{
			name:        "MCP tool node missing output_variable",
			node:        &workflow.MCPToolNode{ID: "tool1", ServerID: "srv1", ToolName: "test"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "output_variable",
		},
		{
			name:        "MCP tool node valid",
			node:        &workflow.MCPToolNode{ID: "tool1", ServerID: "srv1", ToolName: "test", OutputVariable: "out"},
			expectError: false,
		},
		{
			name:        "Transform node missing input_variable",
			node:        &workflow.TransformNode{ID: "t1", Expression: "$.test", OutputVariable: "out"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "input_variable",
		},
		{
			name:        "Transform node missing expression",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", OutputVariable: "out"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "expression",
		},
		{
			name:        "Transform node missing output_variable",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "$.test"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "output_variable",
		},
		{
			name:        "Transform node valid",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "$.test", OutputVariable: "out"},
			expectError: false,
		},
		{
			name:        "Condition node missing condition",
			node:        &workflow.ConditionNode{ID: "c1"},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "condition",
		},
		{
			name:        "Condition node valid",
			node:        &workflow.ConditionNode{ID: "c1", Condition: "true == true"},
			expectError: false,
		},
		{
			name:        "Loop node missing collection",
			node:        &workflow.LoopNode{ID: "l1", ItemVariable: "item", Body: []string{"n1"}},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "collection",
		},
		{
			name:        "Loop node missing item_variable",
			node:        &workflow.LoopNode{ID: "l1", Collection: "items", Body: []string{"n1"}},
			expectError: true,
			errorType:   "missing_required_field",
			fieldName:   "item_variable",
		},
		{
			name:        "Loop node empty body",
			node:        &workflow.LoopNode{ID: "l1", Collection: "items", ItemVariable: "item"},
			expectError: true,
			errorType:   "missing_required_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test", "test")
			errors := ValidateNode(tt.node, wf)

			if tt.expectError {
				if len(errors) == 0 {
					t.Errorf("Expected error but got none")
				}

				found := false
				for _, err := range errors {
					if err.ErrorType == tt.errorType {
						if tt.fieldName != "" && !strings.Contains(err.Message, tt.fieldName) {
							continue
						}
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s with field %s, but not found in errors: %v", tt.errorType, tt.fieldName, errors)
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("Expected no errors, but got: %v", errors)
				}
			}
		})
	}
}

// TestValidateNode_ExpressionSyntax tests expression syntax validation
func TestValidateNode_ExpressionSyntax(t *testing.T) {
	tests := []struct {
		name        string
		node        workflow.Node
		expectError bool
		errorType   string
	}{
		{
			name:        "valid condition expression",
			node:        &workflow.ConditionNode{ID: "c1", Condition: "count > 10"},
			expectError: false,
		},
		{
			name:        "valid JSONPath in transform",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "$.users[0].name", OutputVariable: "out"},
			expectError: false,
		},
		{
			name:        "invalid JSONPath unclosed bracket",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "$.users[0", OutputVariable: "out"},
			expectError: true,
			errorType:   "invalid_jsonpath",
		},
		{
			name:        "valid template in transform",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "Hello ${user.name}", OutputVariable: "out"},
			expectError: false,
		},
		{
			name:        "invalid template unclosed brace",
			node:        &workflow.TransformNode{ID: "t1", InputVariable: "in", Expression: "Hello ${user.name", OutputVariable: "out"},
			expectError: true,
			errorType:   "invalid_template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test", "test")
			errors := ValidateNode(tt.node, wf)

			if tt.expectError {
				if len(errors) == 0 {
					t.Errorf("Expected error but got none")
				}

				found := false
				for _, err := range errors {
					if err.ErrorType == tt.errorType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s, but not found in errors: %v", tt.errorType, errors)
				}
			} else {
				// Filter out only syntax errors
				for _, err := range errors {
					if err.ErrorType == "invalid_expression" || err.ErrorType == "invalid_jsonpath" || err.ErrorType == "invalid_template" {
						t.Errorf("Expected no syntax errors, but got: %v", err)
					}
				}
			}
		})
	}
}

// TestValidateWorkflow_DomainRules tests domain-specific validation rules
func TestValidateWorkflow_DomainRules(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *workflow.Workflow
		expectError bool
		errorType   string
		nodeID      string
	}{
		{
			name: "condition node with 2 edges - valid",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				cond := &workflow.ConditionNode{ID: "cond", Condition: "true"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(cond)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "cond"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "cond", ToNodeID: "a", Condition: "true"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "cond", ToNodeID: "b", Condition: "false"})
				wf.AddEdge(&workflow.Edge{ID: "e4", FromNodeID: "a", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{ID: "e5", FromNodeID: "b", ToNodeID: "end"})
				return wf
			}(),
			expectError: false,
		},
		{
			name: "condition node with 1 edge - invalid",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				cond := &workflow.ConditionNode{ID: "cond", Condition: "true"}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(cond)
				wf.AddNode(nodeA)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "cond"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "cond", ToNodeID: "a", Condition: "true"})
				wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "a", ToNodeID: "end"})
				return wf
			}(),
			expectError: true,
			errorType:   "invalid_condition_edges",
			nodeID:      "cond",
		},
		{
			name: "parallel node with 2 branches - valid",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				par := &workflow.ParallelNode{
					ID:            "par",
					Branches:      [][]string{{"a"}, {"b"}},
					MergeStrategy: "wait_all",
				}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				nodeB := &workflow.PassthroughNode{ID: "b"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(par)
				wf.AddNode(nodeA)
				wf.AddNode(nodeB)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "par"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "par", ToNodeID: "end"})
				return wf
			}(),
			expectError: false,
		},
		{
			name: "parallel node with 1 branch - invalid",
			workflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "test workflow")
				start := &workflow.StartNode{ID: "start"}
				par := &workflow.ParallelNode{
					ID:            "par",
					Branches:      [][]string{{"a"}},
					MergeStrategy: "wait_all",
				}
				nodeA := &workflow.PassthroughNode{ID: "a"}
				end := &workflow.EndNode{ID: "end"}

				wf.AddNode(start)
				wf.AddNode(par)
				wf.AddNode(nodeA)
				wf.AddNode(end)

				wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "par"})
				wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "par", ToNodeID: "end"})
				return wf
			}(),
			expectError: true,
			errorType:   "invalid_parallel_branches",
			nodeID:      "par",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := ValidateWorkflow(tt.workflow)

			if tt.expectError {
				if !status.HasErrors() {
					t.Errorf("Expected error but got none")
				}

				errors := status.GetErrors()
				found := false
				for _, err := range errors {
					if err.ErrorType == tt.errorType && err.NodeID == tt.nodeID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error type %s for node %s, but not found in errors: %v", tt.errorType, tt.nodeID, errors)
				}
			} else {
				// Check that specific domain rule errors don't exist
				errors := status.GetErrors()
				for _, err := range errors {
					if err.ErrorType == "invalid_condition_edges" || err.ErrorType == "invalid_parallel_branches" {
						t.Errorf("Expected no domain rule errors, but got: %v", err)
					}
				}
			}
		})
	}
}

// BenchmarkValidateWorkflow benchmarks workflow validation performance
func BenchmarkValidateWorkflow(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes=%d", size), func(b *testing.B) {
			wf := createBenchmarkWorkflow(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				status := ValidateWorkflow(wf)
				if status == nil {
					b.Fatal("validation returned nil")
				}
			}
		})
	}
}

// createBenchmarkWorkflow creates a workflow with n nodes for benchmarking
func createBenchmarkWorkflow(n int) *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("benchmark", "benchmark workflow")

	// Add start node
	start := &workflow.StartNode{ID: "start"}
	wf.AddNode(start)

	// Add n passthrough nodes in a chain
	prevID := "start"
	for i := 0; i < n; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		node := &workflow.PassthroughNode{ID: nodeID}
		wf.AddNode(node)

		edge := &workflow.Edge{
			ID:         fmt.Sprintf("edge-%d", i),
			FromNodeID: prevID,
			ToNodeID:   nodeID,
		}
		wf.AddEdge(edge)

		prevID = nodeID
	}

	// Add end node
	end := &workflow.EndNode{ID: "end"}
	wf.AddNode(end)
	wf.AddEdge(&workflow.Edge{
		ID:         "edge-end",
		FromNodeID: prevID,
		ToNodeID:   "end",
	})

	return wf
}

// TestValidateWorkflow_Performance ensures validation meets performance targets
func TestValidateWorkflow_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Create 100-node workflow
	wf := createBenchmarkWorkflow(100)

	// Run validation 10 times and measure average time
	var totalDuration int64
	iterations := 10

	for i := 0; i < iterations; i++ {
		result := testing.Benchmark(func(b *testing.B) {
			ValidateWorkflow(wf)
		})
		totalDuration += result.NsPerOp()
	}

	avgDuration := totalDuration / int64(iterations)
	avgMs := float64(avgDuration) / 1e6

	// Target: < 500ms for 100 nodes
	if avgMs > 500 {
		t.Errorf("Validation too slow: %.2fms (target: < 500ms for 100 nodes)", avgMs)
	} else {
		t.Logf("Validation performance: %.2fms for 100 nodes (target: < 500ms)", avgMs)
	}
}
