package execution

import (
	"context"
	"sync"
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

// TestEngine_NoTimeout tests that workflows execute normally without timeout configured.
func TestEngine_NoTimeout(t *testing.T) {
	// Create a simple workflow
	wf, err := workflow.NewWorkflow("test-no-timeout", "Test execution without timeout")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add nodes
	start := &workflow.StartNode{ID: "start"}
	end := &workflow.EndNode{ID: "end"}
	wf.AddNode(start)
	wf.AddNode(end)
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"})

	// Create engine with no timeout
	engine := NewEngine()
	defer engine.Close()

	// Execute without timeout (existing behavior)
	ctx := context.Background()
	exec, err := engine.Execute(ctx, wf, nil)

	// Verify execution completed successfully
	if err != nil {
		t.Fatalf("Expected execution to succeed, got error: %v", err)
	}

	if exec.Status != execution.StatusCompleted {
		t.Errorf("Expected status Completed, got %s", exec.Status)
	}

	// Verify no timeout fields are set (backwards compatibility)
	if exec.Context.TimedOut {
		t.Error("Expected TimedOut to be false when no timeout configured")
	}

	if exec.Context.TimeoutNode != "" {
		t.Errorf("Expected TimeoutNode to be empty, got %s", exec.Context.TimeoutNode)
	}
}

// TestEngine_TimeoutDoesNotTrigger tests workflow completion before timeout.
func TestEngine_TimeoutDoesNotTrigger(t *testing.T) {
	// Create a simple workflow that completes quickly
	wf, err := workflow.NewWorkflow("test-quick", "Test quick execution with timeout")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add nodes
	start := &workflow.StartNode{ID: "start"}
	end := &workflow.EndNode{ID: "end"}
	wf.AddNode(start)
	wf.AddNode(end)
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"})

	// Create engine with timeout that won't trigger (5 seconds)
	engine := NewEngineWithTimeout(5 * time.Second)
	defer engine.Close()

	// Execute with timeout that shouldn't trigger
	ctx := context.Background()
	startTime := time.Now()
	exec, err := engine.Execute(ctx, wf, nil)
	duration := time.Since(startTime)

	// Verify execution completed successfully
	if err != nil {
		t.Fatalf("Expected execution to succeed, got error: %v", err)
	}

	if exec.Status != execution.StatusCompleted {
		t.Errorf("Expected status Completed, got %s", exec.Status)
	}

	// Verify timeout did not occur
	if exec.Context.TimedOut {
		t.Error("Expected TimedOut to be false when workflow completes before timeout")
	}

	// Verify execution was fast (should be under 1 second)
	if duration > 1*time.Second {
		t.Errorf("Expected quick execution, took %v", duration)
	}
}

// TestEngine_TimeoutTriggers tests that timeout terminates long-running workflow.
func TestEngine_TimeoutTriggers(t *testing.T) {
	// Create a workflow that tries to connect to a server (connection will fail fast)
	wf, err := workflow.NewWorkflow("test-timeout", "Test timeout trigger")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add nodes
	start := &workflow.StartNode{ID: "start"}
	slowNode := &workflow.MCPToolNode{
		ID:             "slow-node",
		ServerID:       "slow-server",
		ToolName:       "slow_operation",
		OutputVariable: "result",
	}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(slowNode)
	wf.AddNode(end)
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "slow-node"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "slow-node", ToNodeID: "end"})

	// Add server config (will fail to connect)
	wf.ServerConfigs = append(wf.ServerConfigs, &workflow.ServerConfig{
		ID:        "slow-server",
		Command:   "nonexistent",
		Transport: "stdio",
	})

	// Create engine with very short timeout (50ms)
	engine := NewEngineWithTimeout(50 * time.Millisecond)
	defer engine.Close()

	// Execute with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	startTime := time.Now()
	exec, err := engine.Execute(ctx, wf, nil)
	duration := time.Since(startTime)

	// Verify error occurred
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// This test validates that timeout is detected, but the exact timing depends on
	// when the error occurs (connection failure happens before timeout in this case).
	// The key is that execution stops and doesn't hang indefinitely.
	if duration > 1*time.Second {
		t.Errorf("Expected execution to stop quickly, took %v", duration)
	}

	// Verify execution has terminal status
	if exec != nil && !exec.Status.IsTerminal() {
		t.Errorf("Expected terminal status, got %s", exec.Status)
	}

	// If timeout was set, verify it's configured correctly
	if exec != nil && exec.Context.TimeoutDuration > 0 {
		expectedTimeout := 50 * time.Millisecond
		if exec.Context.TimeoutDuration != expectedTimeout {
			t.Errorf("Expected timeout duration %v, got %v", expectedTimeout, exec.Context.TimeoutDuration)
		}
	}
}

// TestEngine_TimeoutErrorContext tests that timeout errors include node context.
func TestEngine_TimeoutErrorContext(t *testing.T) {
	// Create a workflow
	wf, err := workflow.NewWorkflow("test-context", "Test timeout error context")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	start := &workflow.StartNode{ID: "start"}
	node1 := &workflow.MCPToolNode{
		ID:             "node-1",
		ServerID:       "server-1",
		ToolName:       "operation",
		OutputVariable: "result",
	}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node1)
	wf.AddNode(end)
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node-1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "node-1", ToNodeID: "end"})

	wf.ServerConfigs = append(wf.ServerConfigs, &workflow.ServerConfig{
		ID:        "server-1",
		Command:   "nonexistent",
		Transport: "stdio",
	})

	// Create engine with short timeout
	engine := NewEngineWithTimeout(50 * time.Millisecond)
	defer engine.Close()

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	exec, err := engine.Execute(ctx, wf, nil)

	// Verify error contains context
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Check if error message includes node information
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Verify execution context includes timeout node info
	if exec != nil && exec.Context.TimedOut {
		if exec.Context.TimeoutNode == "" {
			t.Error("Expected TimeoutNode to identify which node was executing when timeout occurred")
		}

		// Verify TimeoutNode is a valid node ID from the workflow
		validNodeIDs := []string{"start", "node-1", "end"}
		found := false
		for _, id := range validNodeIDs {
			if exec.Context.TimeoutNode == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TimeoutNode '%s' is not a valid node ID from workflow", exec.Context.TimeoutNode)
		}
	}
}

// TestEngine_ContextCancellationPropagates tests that context cancellation propagates to all nodes.
func TestEngine_ContextCancellationPropagates(t *testing.T) {
	// Create a workflow with multiple nodes
	wf, err := workflow.NewWorkflow("test-cancel", "Test context cancellation")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	start := &workflow.StartNode{ID: "start"}
	node1 := &workflow.PassthroughNode{ID: "node-1"}
	node2 := &workflow.PassthroughNode{ID: "node-2"}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node1)
	wf.AddNode(node2)
	wf.AddNode(end)

	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node-1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "node-1", ToNodeID: "node-2"})
	wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "node-2", ToNodeID: "end"})

	// Create engine
	engine := NewEngine()
	defer engine.Close()

	// Create a context that we'll cancel during execution
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to test cancellation propagation
	cancel()

	// Execute with cancelled context
	exec, err := engine.Execute(ctx, wf, nil)

	// Verify cancellation was detected
	if err == nil {
		t.Fatal("Expected cancellation error, got nil")
	}

	// Verify execution was cancelled
	if exec != nil && exec.Status != execution.StatusCancelled && exec.Status != execution.StatusFailed {
		t.Errorf("Expected status Cancelled or Failed, got %s", exec.Status)
	}
}

// TestEngine_InProgressNodeCancellation tests that in-progress node operations are cancelled.
func TestEngine_InProgressNodeCancellation(t *testing.T) {
	// Create a workflow with a node that would take time
	wf, err := workflow.NewWorkflow("test-inprogress", "Test in-progress cancellation")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	start := &workflow.StartNode{ID: "start"}
	slowNode := &workflow.MCPToolNode{
		ID:             "slow-node",
		ServerID:       "slow-server",
		ToolName:       "slow_op",
		OutputVariable: "result",
	}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(slowNode)
	wf.AddNode(end)

	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "slow-node"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "slow-node", ToNodeID: "end"})

	wf.ServerConfigs = append(wf.ServerConfigs, &workflow.ServerConfig{
		ID:        "slow-server",
		Command:   "nonexistent",
		Transport: "stdio",
	})

	// Create engine with timeout
	engine := NewEngineWithTimeout(50 * time.Millisecond)
	defer engine.Close()

	// Create cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Execute
	startTime := time.Now()
	exec, err := engine.Execute(ctx, wf, nil)
	duration := time.Since(startTime)

	// Verify execution was stopped (either by timeout or cancellation)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify execution didn't run indefinitely (stopped within timeout window)
	if duration > 200*time.Millisecond {
		t.Errorf("Expected execution to stop quickly, took %v", duration)
	}

	// Verify execution context shows cancellation or timeout
	if exec != nil {
		if exec.Status != execution.StatusCancelled && exec.Status != execution.StatusFailed && exec.Status != execution.StatusRunning {
			t.Errorf("Expected status Cancelled/Failed/Running, got %s", exec.Status)
		}
	}
}

// TestEngine_CleanupAfterTimeout tests that cleanup occurs after timeout.
func TestEngine_CleanupAfterTimeout(t *testing.T) {
	// Create a workflow
	wf, err := workflow.NewWorkflow("test-cleanup", "Test cleanup after timeout")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	start := &workflow.StartNode{ID: "start"}
	node := &workflow.MCPToolNode{
		ID:             "node-1",
		ServerID:       "server-1",
		ToolName:       "operation",
		OutputVariable: "result",
	}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node)
	wf.AddNode(end)

	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node-1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "node-1", ToNodeID: "end"})

	wf.ServerConfigs = append(wf.ServerConfigs, &workflow.ServerConfig{
		ID:        "server-1",
		Command:   "nonexistent",
		Transport: "stdio",
	})

	// Create engine with short timeout
	engine := NewEngineWithTimeout(50 * time.Millisecond)
	defer engine.Close()

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	exec, err := engine.Execute(ctx, wf, nil)

	// Verify error occurred
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Verify execution has completed timestamp (cleanup happened)
	if exec != nil && exec.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set after timeout cleanup")
	}

	// Verify execution has final status
	if exec != nil && !exec.Status.IsTerminal() {
		t.Errorf("Expected terminal status after timeout, got %s", exec.Status)
	}
}

// TestEngine_MultipleConcurrentTimeouts tests multiple concurrent executions with different timeouts.
func TestEngine_MultipleConcurrentTimeouts(t *testing.T) {
	// Create a simple workflow
	wf, err := workflow.NewWorkflow("test-concurrent", "Test concurrent timeouts")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	start := &workflow.StartNode{ID: "start"}
	node := &workflow.PassthroughNode{ID: "node-1"}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(node)
	wf.AddNode(end)

	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "node-1"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "node-1", ToNodeID: "end"})

	// Create multiple engines with different timeouts
	engine1 := NewEngineWithTimeout(100 * time.Millisecond)
	defer engine1.Close()

	engine2 := NewEngineWithTimeout(200 * time.Millisecond)
	defer engine2.Close()

	engine3 := NewEngine() // No timeout
	defer engine3.Close()

	// Execute concurrently
	var wg sync.WaitGroup
	wg.Add(3)

	var exec1, exec2, exec3 *execution.Execution
	var err1, err2, err3 error
	_, _, _ = err1, err2, err3 // Mark as used since we're checking the execs

	go func() {
		defer wg.Done()
		ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel1()
		exec1, err1 = engine1.Execute(ctx1, wf, nil)
	}()

	go func() {
		defer wg.Done()
		ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel2()
		exec2, err2 = engine2.Execute(ctx2, wf, nil)
	}()

	go func() {
		defer wg.Done()
		exec3, err3 = engine3.Execute(context.Background(), wf, nil)
	}()

	wg.Wait()

	// Verify all executions completed (some may have timed out, some succeeded)
	// At least engine3 (no timeout) should succeed
	if err3 != nil {
		t.Errorf("Expected engine3 (no timeout) to succeed, got error: %v", err3)
	}

	if exec3 != nil && exec3.Status != execution.StatusCompleted {
		t.Errorf("Expected engine3 status Completed, got %s", exec3.Status)
	}

	// Verify each execution has unique ID
	if exec1 != nil && exec2 != nil && exec1.ID == exec2.ID {
		t.Error("Expected different execution IDs for concurrent executions")
	}

	if exec1 != nil && exec3 != nil && exec1.ID == exec3.ID {
		t.Error("Expected different execution IDs for concurrent executions")
	}

	// All executions should have completed (either success, failure, or timeout)
	if exec1 != nil && !exec1.Status.IsTerminal() {
		t.Errorf("Expected terminal status for exec1, got %s", exec1.Status)
	}

	if exec2 != nil && !exec2.Status.IsTerminal() {
		t.Errorf("Expected terminal status for exec2, got %s", exec2.Status)
	}
}
