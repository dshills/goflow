package execution

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/workflow"
)

func TestEngine_TopologicalSort(t *testing.T) {
	// Create a simple workflow with dependencies
	wf, err := workflow.NewWorkflow("test-workflow", "Test topological sort")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add nodes
	start := &workflow.StartNode{ID: "start"}
	node1 := &workflow.MCPToolNode{ID: "node1", ServerID: "server1", ToolName: "tool1", OutputVariable: "out1"}
	node2 := &workflow.MCPToolNode{ID: "node2", ServerID: "server1", ToolName: "tool2", OutputVariable: "out2"}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node1)
	wf.AddNode(node2)
	wf.AddNode(end)

	// Add edges: start -> node1 -> node2 -> end
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "node1", ToNodeID: "node2"})
	wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "node2", ToNodeID: "end"})

	// Create engine
	engine := NewEngine()
	defer engine.Close()

	// Perform topological sort
	sorted, err := engine.topologicalSort(wf)
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	// Verify order
	if len(sorted) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(sorted))
	}

	// Verify node1 comes before node2
	node1Idx := -1
	node2Idx := -1
	for i, node := range sorted {
		switch node.GetID() {
		case "node1":
			node1Idx = i
		case "node2":
			node2Idx = i
		}
	}

	if node1Idx == -1 || node2Idx == -1 {
		t.Fatal("Not all nodes found in sorted result")
	}

	if node1Idx >= node2Idx {
		t.Errorf("node1 (idx %d) should come before node2 (idx %d)", node1Idx, node2Idx)
	}
}

func TestEngine_ValidateInputs(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test")

	// Add a variable (no Required field yet, so this is just a placeholder)
	wf.AddVariable(&workflow.Variable{
		Name: "test_var",
		Type: "string",
	})

	engine := NewEngine()
	defer engine.Close()

	// For now, validation always succeeds since we don't have Required field
	err := engine.validateInputs(wf, nil)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

func TestEngine_InitializeVariables(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test")

	// Add variables with defaults
	wf.AddVariable(&workflow.Variable{
		Name:         "var1",
		Type:         "string",
		DefaultValue: "default1",
	})
	wf.AddVariable(&workflow.Variable{
		Name:         "var2",
		Type:         "number",
		DefaultValue: 42,
	})

	// Create execution context with one variable set
	inputs := map[string]interface{}{
		"var1": "custom_value",
	}
	ctx, _ := execution.NewExecutionContext(inputs)

	engine := NewEngine()
	defer engine.Close()

	// Initialize variables
	err := engine.initializeVariables(ctx, wf)
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	// Verify var1 kept its input value
	val1, ok := ctx.GetVariable("var1")
	if !ok || val1 != "custom_value" {
		t.Errorf("Expected var1 to be 'custom_value', got %v", val1)
	}

	// Verify var2 got its default value
	val2, ok := ctx.GetVariable("var2")
	if !ok || val2 != 42 {
		t.Errorf("Expected var2 to be 42, got %v", val2)
	}
}

func TestEngine_SubstituteVariables(t *testing.T) {
	ctx, _ := execution.NewExecutionContext(map[string]interface{}{
		"name": "Alice",
		"age":  30,
		"city": "NYC",
	})

	engine := NewEngine()
	defer engine.Close()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello ${name}", "Hello Alice"},
		{"Age: ${age}", "Age: 30"},
		{"${name} lives in ${city}", "Alice lives in NYC"},
		{"No variables here", "No variables here"},
		{"${name} is ${age} years old", "Alice is 30 years old"},
	}

	for _, tt := range tests {
		result, err := engine.substituteVariables(tt.input, ctx)
		if err != nil {
			t.Errorf("Failed to substitute '%s': %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("substituteVariables('%s') = '%s', expected '%s'", tt.input, result, tt.expected)
		}
	}
}

func TestEngine_SubstituteVariables_MissingVariable(t *testing.T) {
	ctx, _ := execution.NewExecutionContext(map[string]interface{}{
		"name": "Alice",
	})

	engine := NewEngine()
	defer engine.Close()

	// Should return error for missing variable
	_, err := engine.substituteVariables("Hello ${missing}", ctx)
	if err == nil {
		t.Error("Expected error for missing variable")
	}
}

func TestExecutionError_Wrapping(t *testing.T) {
	nodeID := types.NodeID("test-node")

	// Test tool error wrapping
	toolErr := &MCPToolError{
		ServerID:    "server1",
		ToolName:    "tool1",
		Message:     "connection failed",
		Recoverable: true,
		Context: map[string]interface{}{
			"retry_count": 3,
		},
	}

	execErr := WrapToolError(nodeID, "server1", "tool1", toolErr, map[string]interface{}{
		"param1": "value1",
	})

	if execErr.Type != execution.ErrorTypeConnection {
		t.Errorf("Expected error type Connection, got %s", execErr.Type)
	}

	if !execErr.Recoverable {
		t.Error("Expected error to be recoverable")
	}

	if execErr.NodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, execErr.NodeID)
	}

	// Verify context was merged
	if execErr.Context["retry_count"] != 3 {
		t.Error("Expected retry_count in context")
	}
	if execErr.Context["server_id"] != "server1" {
		t.Error("Expected server_id in context")
	}
}

func TestIsRecoverable(t *testing.T) {
	// Test recoverable execution error
	execErr := &execution.ExecutionError{
		Type:        execution.ErrorTypeTimeout,
		Message:     "timeout",
		Recoverable: true,
	}

	if !IsRecoverable(execErr) {
		t.Error("Expected error to be recoverable")
	}

	// Test non-recoverable error
	execErr2 := &execution.ExecutionError{
		Type:        execution.ErrorTypeValidation,
		Message:     "invalid",
		Recoverable: false,
	}

	if IsRecoverable(execErr2) {
		t.Error("Expected error to not be recoverable")
	}

	// Test MCP tool error
	toolErr := &MCPToolError{
		ServerID:    "server1",
		ToolName:    "tool1",
		Message:     "failed",
		Recoverable: true,
	}

	if !IsRecoverable(toolErr) {
		t.Error("Expected tool error to be recoverable")
	}
}

func TestLogger_NoRepository(t *testing.T) {
	// Create logger without repository
	logger := NewLogger(nil)

	// These should not panic
	exec, _ := execution.NewExecution(types.WorkflowID("wf1"), "1.0", nil)
	logger.LogExecutionStart(exec)
	logger.LogExecutionComplete(exec)

	nodeExec := execution.NewNodeExecution(exec.ID, types.NodeID("node1"), "mcp_tool")
	logger.LogNodeExecution(nodeExec)

	// These should return errors
	_, err := logger.GetExecutionLogs(types.NewExecutionID())
	if err == nil {
		t.Error("Expected error when no repository configured")
	}
}

func BenchmarkTopologicalSort(b *testing.B) {
	// Create a workflow with 100 nodes in a linear chain
	wf, _ := workflow.NewWorkflow("bench", "benchmark")

	wf.AddNode(&workflow.StartNode{ID: "start"})
	for i := 0; i < 100; i++ {
		nodeID := types.NodeID(time.Now().Format("20060102150405.000000"))
		wf.AddNode(&workflow.MCPToolNode{
			ID:             string(nodeID),
			ServerID:       "server1",
			ToolName:       "tool",
			OutputVariable: "out",
		})
	}
	wf.AddNode(&workflow.EndNode{ID: "end"})

	// Add linear edges
	prevID := "start"
	for _, node := range wf.Nodes {
		if node.Type() != "start" && node.Type() != "end" {
			edgeID := types.NodeID(time.Now().Format("20060102150405.000000"))
			wf.AddEdge(&workflow.Edge{
				ID:         string(edgeID),
				FromNodeID: prevID,
				ToNodeID:   node.GetID(),
			})
			prevID = node.GetID()
		}
	}
	wf.AddEdge(&workflow.Edge{ID: "final", FromNodeID: prevID, ToNodeID: "end"})

	engine := NewEngine()
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.topologicalSort(wf)
		if err != nil {
			b.Fatalf("Sort failed: %v", err)
		}
	}
}

func BenchmarkVariableSubstitution(b *testing.B) {
	ctx, _ := execution.NewExecutionContext(map[string]interface{}{
		"var1": "value1",
		"var2": "value2",
		"var3": "value3",
	})

	engine := NewEngine()
	defer engine.Close()

	input := "Hello ${var1}, you are ${var2} and live in ${var3}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.substituteVariables(input, ctx)
		if err != nil {
			b.Fatalf("Substitution failed: %v", err)
		}
	}
}
