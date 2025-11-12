package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestLoopNode_ArrayIteration tests that LoopNode iterates over array collections correctly
func TestLoopNode_ArrayIteration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name           string
		collection     []interface{}
		expectedCount  int
		collectionType string
	}{
		{
			name:           "iterate over string array",
			collection:     []interface{}{"apple", "banana", "cherry"},
			expectedCount:  3,
			collectionType: "strings",
		},
		{
			name:           "iterate over number array",
			collection:     []interface{}{10, 20, 30, 40, 50},
			expectedCount:  5,
			collectionType: "numbers",
		},
		{
			name: "iterate over object array",
			collection: []interface{}{
				map[string]interface{}{"id": 1, "name": "Alice"},
				map[string]interface{}{"id": 2, "name": "Bob"},
				map[string]interface{}{"id": 3, "name": "Charlie"},
			},
			expectedCount:  3,
			collectionType: "objects",
		},
		{
			name:           "iterate over single item",
			collection:     []interface{}{"single"},
			expectedCount:  1,
			collectionType: "single",
		},
		{
			name:           "iterate over empty array",
			collection:     []interface{}{},
			expectedCount:  0,
			collectionType: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "loop-array-test"
variables:
  - name: "items"
    type: "array"
    default: []
  - name: "processedCount"
    type: "number"
    default: 0
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "currentItem"
    body:
      - "process_item"
  - id: "process_item"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "${processedCount}"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{
				"items": tt.collection,
			}

			result, err := engine.Execute(ctx, wf, inputs)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if result.Status != execution.StatusCompleted {
				t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
			}

			// Verify loop body was executed correct number of times
			processItemCount := 0
			for _, nodeExec := range result.NodeExecutions {
				if string(nodeExec.NodeID) == "process_item" {
					processItemCount++
				}
			}

			if processItemCount != tt.expectedCount {
				t.Errorf("Expected loop body to execute %d times, got %d", tt.expectedCount, processItemCount)
			}
		})
	}
}

// TestLoopNode_ItemVariableScoping tests that item variable is correctly set per iteration
func TestLoopNode_ItemVariableScoping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-variable-scoping-test"
variables:
  - name: "items"
    type: "array"
    default: []
  - name: "results"
    type: "array"
    default: []
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "item"
    body:
      - "echo_item"
  - id: "echo_item"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "${item}"
    output: "itemResult"
  - id: "end"
    type: "end"
    return: "completed"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	testItems := []interface{}{"first", "second", "third"}
	inputs := map[string]interface{}{
		"items": testItems,
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify item variable was set for each iteration
	echoCount := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "echo_item" {
			echoCount++
			if nodeExec.Status != execution.NodeStatusCompleted {
				t.Errorf("Expected echo_item to complete, got status %s", nodeExec.Status)
			}
		}
	}

	if echoCount != len(testItems) {
		t.Errorf("Expected %d echo executions, got %d", len(testItems), echoCount)
	}
}

// TestLoopNode_BreakCondition tests early termination with break condition
func TestLoopNode_BreakCondition(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name               string
		items              []interface{}
		breakCondition     string
		expectedIterations int
	}{
		{
			name:               "break on specific value",
			items:              []interface{}{1, 2, 3, 4, 5},
			breakCondition:     "item > 3",
			expectedIterations: 3, // Should stop after processing 3
		},
		{
			name:               "break on first item",
			items:              []interface{}{"stop", "continue", "continue"},
			breakCondition:     "item == 'stop'",
			expectedIterations: 0, // Should break immediately
		},
		{
			name:               "no break - complete all",
			items:              []interface{}{1, 2, 3},
			breakCondition:     "item > 100",
			expectedIterations: 3, // Should process all items
		},
		{
			name:               "break on last item",
			items:              []interface{}{10, 20, 30},
			breakCondition:     "item == 30",
			expectedIterations: 2, // Should stop before processing last
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "loop-break-test"
variables:
  - name: "items"
    type: "array"
    default: []
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "item"
    break_condition: "` + tt.breakCondition + `"
    body:
      - "process"
  - id: "process"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{
				"items": tt.items,
			}

			result, err := engine.Execute(ctx, wf, inputs)
			if err != nil {
				t.Fatalf("Workflow execution failed: %v", err)
			}

			if result.Status != execution.StatusCompleted {
				t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
			}

			// Count how many times process node executed
			processCount := 0
			for _, nodeExec := range result.NodeExecutions {
				if string(nodeExec.NodeID) == "process" {
					processCount++
				}
			}

			if processCount != tt.expectedIterations {
				t.Errorf("Expected %d iterations before break, got %d", tt.expectedIterations, processCount)
			}
		})
	}
}

// TestLoopNode_ResultCollection tests that results from all iterations are collected
func TestLoopNode_ResultCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-result-collection-test"
variables:
  - name: "numbers"
    type: "array"
    default: []
  - name: "results"
    type: "array"
    default: []
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "loop_numbers"
    type: "loop"
    collection: "numbers"
    item: "num"
    body:
      - "process_number"
  - id: "process_number"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "Processed: ${num}"
    output: "processedResult"
  - id: "end"
    type: "end"
    return: "${results}"
edges:
  - from: "start"
    to: "loop_numbers"
  - from: "loop_numbers"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	testNumbers := []interface{}{1, 2, 3, 4, 5}
	inputs := map[string]interface{}{
		"numbers": testNumbers,
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify all numbers were processed
	processedCount := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "process_number" {
			processedCount++
		}
	}

	if processedCount != len(testNumbers) {
		t.Errorf("Expected %d processed numbers, got %d", len(testNumbers), processedCount)
	}
}

// TestLoopNode_NestedLoops tests loops within loops
func TestLoopNode_NestedLoops(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "nested-loop-test"
variables:
  - name: "outerItems"
    type: "array"
    default: []
  - name: "innerItems"
    type: "array"
    default: []
nodes:
  - id: "start"
    type: "start"
  - id: "outer_loop"
    type: "loop"
    collection: "outerItems"
    item: "outerItem"
    body:
      - "inner_loop"
  - id: "inner_loop"
    type: "loop"
    collection: "innerItems"
    item: "innerItem"
    body:
      - "process_pair"
  - id: "process_pair"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "outer_loop"
  - from: "outer_loop"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{
		"outerItems": []interface{}{"A", "B", "C"},
		"innerItems": []interface{}{1, 2},
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify nested execution count (3 outer * 2 inner = 6)
	pairCount := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "process_pair" {
			pairCount++
		}
	}

	expectedPairs := 3 * 2 // outer items * inner items
	if pairCount != expectedPairs {
		t.Errorf("Expected %d pair processes, got %d", expectedPairs, pairCount)
	}
}

// TestLoopNode_ErrorHandling tests error handling within loop iterations
func TestLoopNode_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-error-test"
variables:
  - name: "items"
    type: "array"
    default: []
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "item"
    body:
      - "risky_operation"
  - id: "risky_operation"
    type: "mcp_tool"
    server: "test-server"
    tool: "read_file"
    parameters:
      path: "${item}"
    output: "fileContent"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{
		"items": []interface{}{
			"/nonexistent/file1.txt",
			"/nonexistent/file2.txt",
			"/nonexistent/file3.txt",
		},
	}

	result, err := engine.Execute(ctx, wf, inputs)

	// Loop should handle errors appropriately
	// Either stop on first error or continue depending on error handling strategy
	if err == nil && result.Status == execution.StatusCompleted {
		// If error handling allows continuation, verify partial execution
		t.Log("Loop continued despite errors - checking partial execution")
	} else {
		// If loop stops on first error, verify error was captured
		if result != nil && result.Status == execution.StatusFailed {
			if result.Error == nil {
				t.Error("Expected error details to be captured")
			}
		}
	}
}

// TestLoopNode_ComplexObjectIteration tests iterating over complex objects
func TestLoopNode_ComplexObjectIteration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-complex-objects-test"
variables:
  - name: "users"
    type: "array"
    default: []
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "loop_users"
    type: "loop"
    collection: "users"
    item: "user"
    body:
      - "process_user"
  - id: "process_user"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "Processing user: ${user.name}"
    output: "userResult"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "loop_users"
  - from: "loop_users"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{"id": 1, "name": "Alice", "active": true},
			map[string]interface{}{"id": 2, "name": "Bob", "active": false},
			map[string]interface{}{"id": 3, "name": "Charlie", "active": true},
		},
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify all users were processed
	processedUsers := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "process_user" {
			processedUsers++
		}
	}

	if processedUsers != 3 {
		t.Errorf("Expected 3 users processed, got %d", processedUsers)
	}
}

// TestLoopNode_VariableIsolation tests that loop variables don't leak between iterations
func TestLoopNode_VariableIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-isolation-test"
variables:
  - name: "items"
    type: "array"
    default: []
  - name: "lastItem"
    type: "string"
    default: ""
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "currentItem"
    body:
      - "store_item"
  - id: "store_item"
    type: "passthrough"
  - id: "check_final"
    type: "passthrough"
  - id: "end"
    type: "end"
    return: "${lastItem}"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "check_final"
  - from: "check_final"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	testItems := []interface{}{"first", "second", "third"}
	inputs := map[string]interface{}{
		"items": testItems,
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// After loop completes, currentItem should not be accessible outside loop scope
	// This tests proper variable scoping
	_, exists := result.Context.GetVariable("currentItem")
	if exists {
		t.Error("Loop item variable 'currentItem' should not leak outside loop scope")
	}
}

// TestLoopNode_LargeCollection tests performance with larger collections
func TestLoopNode_LargeCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large collection test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-large-collection-test"
variables:
  - name: "items"
    type: "array"
    default: []
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "item"
    body:
      - "process"
  - id: "process"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Generate large collection
	largeCollection := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		largeCollection[i] = fmt.Sprintf("item-%d", i)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{
		"items": largeCollection,
	}

	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, inputs)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify all items were processed
	processCount := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "process" {
			processCount++
		}
	}

	if processCount != 100 {
		t.Errorf("Expected 100 iterations, got %d", processCount)
	}

	// Performance check: should complete within reasonable time
	t.Logf("Processed 100 items in %v", duration)
	if duration > 30*time.Second {
		t.Errorf("Large collection processing took too long: %v", duration)
	}
}

// TestLoopNode_ConditionalBreakWithTransform tests break condition with data transformation
func TestLoopNode_ConditionalBreakWithTransform(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "loop-conditional-break-transform-test"
variables:
  - name: "items"
    type: "array"
    default: []
  - name: "threshold"
    type: "number"
    default: 50
nodes:
  - id: "start"
    type: "start"
  - id: "loop_items"
    type: "loop"
    collection: "items"
    item: "item"
    break_condition: "item.value > threshold"
    body:
      - "process_item"
  - id: "process_item"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "loop_items"
  - from: "loop_items"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"value": 10},
			map[string]interface{}{"value": 25},
			map[string]interface{}{"value": 40},
			map[string]interface{}{"value": 60}, // Should break before this
			map[string]interface{}{"value": 80},
		},
		"threshold": 50,
	}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Should process items with values 10, 25, 40 and break before 60
	processCount := 0
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "process_item" {
			processCount++
		}
	}

	if processCount != 3 {
		t.Errorf("Expected 3 iterations before break (values <= 50), got %d", processCount)
	}
}
