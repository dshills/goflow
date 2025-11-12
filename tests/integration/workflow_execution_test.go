package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowExecution_SimpleReadTransformWrite tests complete read-transform-write workflow
func TestWorkflowExecution_SimpleReadTransformWrite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup test files
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.txt")
	outputPath := filepath.Join(tmpDir, "output.txt")

	testContent := "hello world"
	err := os.WriteFile(inputPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test input file: %v", err)
	}

	// Load workflow
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// This should fail because workflow.ParseFile doesn't exist yet
	wf, err := workflow.ParseFile(absPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Create execution engine
	engine := runtimeexec.NewEngine()

	// Set input variables
	inputs := map[string]interface{}{
		"input_path":  inputPath,
		"output_path": outputPath,
	}

	// Execute workflow
	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Verify output file content (should be transformed)
	outputContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check that transformation was applied
	if string(outputContent) == testContent {
		t.Error("Expected content to be transformed, but it matches input")
	}
}

// TestWorkflowExecution_TopologicalSort tests that nodes execute in correct order
func TestWorkflowExecution_TopologicalSort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "topological-test"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "node_a"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "A"
    output: "result_a"
  - id: "node_b"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "${result_a} -> B"
    output: "result_b"
  - id: "node_c"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "${result_b} -> C"
    output: "result_c"
  - id: "end"
    type: "end"
    return: "${result_c}"
edges:
  - from: "start"
    to: "node_a"
  - from: "node_a"
    to: "node_b"
  - from: "node_b"
    to: "node_c"
  - from: "node_c"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify all nodes executed
	if len(result.NodeExecutions) != 5 { // start, node_a, node_b, node_c, end
		t.Errorf("Expected 5 node executions, got %d", len(result.NodeExecutions))
	}

	// Verify execution order follows dependencies
	executionOrder := make([]string, len(result.NodeExecutions))
	for i, ne := range result.NodeExecutions {
		executionOrder[i] = string(ne.NodeID)
	}

	// node_a must come before node_b, node_b before node_c
	aIndex, bIndex, cIndex := -1, -1, -1
	for i, nodeID := range executionOrder {
		switch nodeID {
		case "node_a":
			aIndex = i
		case "node_b":
			bIndex = i
		case "node_c":
			cIndex = i
		}
	}

	if aIndex == -1 || bIndex == -1 || cIndex == -1 {
		t.Fatal("Not all nodes were executed")
	}

	if aIndex >= bIndex {
		t.Errorf("node_a (index %d) should execute before node_b (index %d)", aIndex, bIndex)
	}

	if bIndex >= cIndex {
		t.Errorf("node_b (index %d) should execute before node_c (index %d)", bIndex, cIndex)
	}
}

// TestWorkflowExecution_VariablePassing tests variable passing between nodes
func TestWorkflowExecution_VariablePassing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "variable-passing-test"
variables:
  - name: "initial_value"
    type: "string"
    default: "Start"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "step1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "${initial_value}"
    output: "step1_result"
  - id: "step2"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "Step1 said: ${step1_result}"
    output: "step2_result"
  - id: "end"
    type: "end"
    return: "${step2_result}"
edges:
  - from: "start"
    to: "step1"
  - from: "step1"
    to: "step2"
  - from: "step2"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify variables were passed correctly
	if val, ok := result.Context.GetVariable("step1_result"); !ok || val == nil {
		t.Error("Expected step1_result to be set")
	}

	if val, ok := result.Context.GetVariable("step2_result"); !ok || val == nil {
		t.Error("Expected step2_result to be set")
	}

	// Verify final return value contains expected data
	if result.ReturnValue == nil {
		t.Error("Expected return value to be set")
	}
}

// TestWorkflowExecution_ErrorHandling tests error handling and propagation
func TestWorkflowExecution_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "error-handling-test"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "failing_node"
    type: "mcp_tool"
    server: "test-server"
    tool: "read_file"
    parameters:
      path: "/nonexistent/file/path.txt"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "failing_node"
  - from: "failing_node"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)

	// Expect execution to fail or result to have error status
	if err == nil && result.Status == execution.StatusCompleted {
		t.Error("Expected execution to fail or have error status")
	}

	// Verify error details are captured
	if result != nil {
		if result.Error == nil && result.Status == execution.StatusFailed {
			t.Error("Expected error details to be captured")
		}
	}
}

// TestWorkflowExecution_CancellationHandling tests context cancellation
func TestWorkflowExecution_CancellationHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	yaml := `
version: "1.0"
name: "cancellation-test"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "node1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "Message 1"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "node1"
  - from: "node1"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	// Cancel context before execution completes
	cancel()

	result, err := engine.Execute(ctx, wf, nil)

	// Expect cancellation error or cancelled status
	if err == nil && result.Status != execution.StatusCancelled {
		t.Error("Expected execution to be cancelled")
	}

	if ctx.Err() != context.Canceled {
		t.Errorf("Expected context to be cancelled, got: %v", ctx.Err())
	}
}

// TestWorkflowExecution_InputValidation tests input variable validation
func TestWorkflowExecution_InputValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "input-validation-test"
variables:
  - name: "required_var"
    type: "string"
    required: true
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../cmd/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "node1"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "${required_var}"
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "node1"
  - from: "node1"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()

	// Execute without providing required variable
	_, err = engine.Execute(ctx, wf, nil)
	if err == nil {
		t.Error("Expected error for missing required variable")
	}

	// Execute with required variable
	inputs := map[string]interface{}{
		"required_var": "test value",
	}
	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Errorf("Expected successful execution with valid inputs, got error: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}
}

// TestWorkflowExecution_TransformNode tests transform node execution
func TestWorkflowExecution_TransformNode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "transform-test"
variables:
  - name: "input_data"
    type: "string"
    default: "hello"
nodes:
  - id: "start"
    type: "start"
  - id: "transform1"
    type: "transform"
    input: "input_data"
    expression: "'HELLO'"
    output: "transformed"
  - id: "end"
    type: "end"
    return: "${transformed}"
edges:
  - from: "start"
    to: "transform1"
  - from: "transform1"
    to: "end"
`

	// This should fail because workflow.Parse doesn't exist yet
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify transformation was applied
	transformed, ok := result.Context.GetVariable("transformed")
	if !ok {
		t.Fatal("Expected transformed variable to be set")
	}

	transformedStr, ok := transformed.(string)
	if !ok {
		t.Fatal("Expected transformed value to be a string")
	}

	if transformedStr != "HELLO" {
		t.Errorf("Expected transformed value 'HELLO', got '%s'", transformedStr)
	}
}

// TestWorkflowExecution_ExecutionTrace tests that execution trace is captured
func TestWorkflowExecution_ExecutionTrace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// This should fail because workflow.ParseFile doesn't exist yet
	wf, err := workflow.ParseFile(absPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	tmpDir := t.TempDir()
	inputs := map[string]interface{}{
		"input_path":  filepath.Join(tmpDir, "input.txt"),
		"output_path": filepath.Join(tmpDir, "output.txt"),
	}

	// Write input file
	err = os.WriteFile(inputs["input_path"].(string), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify execution trace contains all nodes
	if len(result.NodeExecutions) == 0 {
		t.Error("Expected node executions to be recorded")
	}

	// Verify each node execution has required fields
	for _, ne := range result.NodeExecutions {
		if ne.NodeID == "" {
			t.Error("Expected node execution to have NodeID")
		}

		if ne.StartedAt.IsZero() {
			t.Error("Expected node execution to have StartTime")
		}

		if ne.CompletedAt.IsZero() {
			t.Error("Expected node execution to have EndTime")
		}

		if ne.StartedAt.After(ne.CompletedAt) {
			t.Error("Expected StartTime to be before EndTime")
		}
	}

	// Verify execution has start and end times
	if result.StartedAt.IsZero() {
		t.Error("Expected execution to have StartTime")
	}

	if result.CompletedAt.IsZero() {
		t.Error("Expected execution to have EndTime")
	}

	if result.StartedAt.After(result.CompletedAt) {
		t.Error("Expected execution StartTime to be before EndTime")
	}
}
