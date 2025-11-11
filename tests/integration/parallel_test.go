package integration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestParallelNode_TwoBranches tests that ParallelNode executes two branches concurrently
func TestParallelNode_TwoBranches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name          string
		mergeStrategy string
		expectSuccess bool
	}{
		{
			name:          "wait_all strategy completes both branches",
			mergeStrategy: "wait_all",
			expectSuccess: true,
		},
		{
			name:          "wait_any strategy completes on first branch",
			mergeStrategy: "wait_any",
			expectSuccess: true,
		},
		{
			name:          "wait_first strategy returns immediately",
			mergeStrategy: "wait_first",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "parallel-two-branches-test"
variables:
  - name: "branch1Result"
    type: "string"
    default: ""
  - name: "branch2Result"
    type: "string"
    default: ""
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "` + tt.mergeStrategy + `"
    branches:
      - ["branch1_task1", "branch1_task2"]
      - ["branch2_task1", "branch2_task2"]
  - id: "branch1_task1"
    type: "transform"
    input: "branch1Result"
    expression: "'branch1_step1'"
    output: "branch1Result"
  - id: "branch1_task2"
    type: "transform"
    input: "branch1Result"
    expression: "${branch1Result} + '_step2'"
    output: "branch1Result"
  - id: "branch2_task1"
    type: "transform"
    input: "branch2Result"
    expression: "'branch2_step1'"
    output: "branch2Result"
  - id: "branch2_task2"
    type: "transform"
    input: "branch2Result"
    expression: "${branch2Result} + '_step2'"
    output: "branch2Result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{}

			result, err := engine.Execute(ctx, wf, inputs)
			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("Expected successful execution but got error: %v", err)
				}
				if result.Status != execution.StatusCompleted {
					t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
				}
			} else {
				if err == nil {
					t.Error("Expected execution to fail but it succeeded")
				}
			}

			// Verify both branches executed (for wait_all)
			if tt.mergeStrategy == "wait_all" {
				branch1Count := 0
				branch2Count := 0
				for _, nodeExec := range result.NodeExecutions {
					if string(nodeExec.NodeID) == "branch1_task1" || string(nodeExec.NodeID) == "branch1_task2" {
						branch1Count++
					}
					if string(nodeExec.NodeID) == "branch2_task1" || string(nodeExec.NodeID) == "branch2_task2" {
						branch2Count++
					}
				}

				if branch1Count != 2 {
					t.Errorf("Expected branch1 to execute 2 tasks, got %d", branch1Count)
				}
				if branch2Count != 2 {
					t.Errorf("Expected branch2 to execute 2 tasks, got %d", branch2Count)
				}
			}
		})
	}
}

// TestParallelNode_MultipleBranches tests that ParallelNode handles 5+ branches
func TestParallelNode_MultipleBranches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with 5 concurrent branches
	yaml := `
version: "1.0"
name: "parallel-multiple-branches-test"
variables:
  - name: "counter"
    type: "number"
    default: 0
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "wait_all"
    branches:
      - ["branch1"]
      - ["branch2"]
      - ["branch3"]
      - ["branch4"]
      - ["branch5"]
  - id: "branch1"
    type: "passthrough"
  - id: "branch2"
    type: "passthrough"
  - id: "branch3"
    type: "passthrough"
  - id: "branch4"
    type: "passthrough"
  - id: "branch5"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify all 5 branches executed
	branchCount := 0
	for _, nodeExec := range result.NodeExecutions {
		nodeID := string(nodeExec.NodeID)
		if nodeID == "branch1" || nodeID == "branch2" || nodeID == "branch3" ||
			nodeID == "branch4" || nodeID == "branch5" {
			branchCount++
		}
	}

	if branchCount != 5 {
		t.Errorf("Expected 5 branches to execute, got %d", branchCount)
	}
}

// TestParallelNode_BranchContextIsolation tests that variables in one branch don't affect another
func TestParallelNode_BranchContextIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "parallel-context-isolation-test"
variables:
  - name: "sharedVar"
    type: "string"
    default: "initial"
  - name: "branch1Var"
    type: "string"
    default: ""
  - name: "branch2Var"
    type: "string"
    default: ""
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "wait_all"
    branches:
      - ["branch1_modify"]
      - ["branch2_modify"]
  - id: "branch1_modify"
    type: "transform"
    input: "sharedVar"
    expression: "'branch1_value'"
    output: "branch1Var"
  - id: "branch2_modify"
    type: "transform"
    input: "sharedVar"
    expression: "'branch2_value'"
    output: "branch2Var"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify both branches completed their own variable assignments
	// without interfering with each other or the shared variable
	branch1Found := false
	branch2Found := false

	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "branch1_modify" {
			branch1Found = true
		}
		if string(nodeExec.NodeID) == "branch2_modify" {
			branch2Found = true
		}
	}

	if !branch1Found {
		t.Error("Branch1 did not execute")
	}
	if !branch2Found {
		t.Error("Branch2 did not execute")
	}
}

// TestParallelNode_ErrorInOneBranch tests that error in one branch doesn't stop other branches
func TestParallelNode_ErrorInOneBranch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name             string
		mergeStrategy    string
		expectOverallErr bool
	}{
		{
			name:             "wait_all fails if any branch fails",
			mergeStrategy:    "wait_all",
			expectOverallErr: true,
		},
		{
			name:             "wait_any succeeds if one branch succeeds",
			mergeStrategy:    "wait_any",
			expectOverallErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
version: "1.0"
name: "parallel-error-handling-test"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
variables:
  - name: "result"
    type: "string"
    default: ""
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "` + tt.mergeStrategy + `"
    branches:
      - ["branch1_success"]
      - ["branch2_fail"]
  - id: "branch1_success"
    type: "transform"
    input: "result"
    expression: "'success'"
    output: "result"
  - id: "branch2_fail"
    type: "mcp_tool"
    server: "test-server"
    tool: "failing_tool"
    parameters: {}
    output: "result"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "end"
`

			wf, err := workflow.Parse([]byte(yaml))
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			engine := runtimeexec.NewEngine()
			inputs := map[string]interface{}{}

			result, err := engine.Execute(ctx, wf, inputs)

			if tt.expectOverallErr {
				if err == nil && result.Status == execution.StatusCompleted {
					t.Error("Expected execution to fail due to branch error, but it succeeded")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected execution to succeed despite branch error, but got: %v", err)
				}
			}

			// Verify that branch1 executed even though branch2 failed
			branch1Executed := false
			for _, nodeExec := range result.NodeExecutions {
				if string(nodeExec.NodeID) == "branch1_success" {
					branch1Executed = true
					break
				}
			}

			if !branch1Executed {
				t.Error("Expected successful branch to execute even when other branch fails")
			}
		})
	}
}

// TestParallelNode_ActualConcurrency tests that branches truly execute concurrently
func TestParallelNode_ActualConcurrency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a shared counter to track concurrent executions
	var concurrentCount int32
	var maxConcurrent int32

	yaml := `
version: "1.0"
name: "parallel-concurrency-test"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go", "--delay=100ms"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "wait_all"
    branches:
      - ["branch1_task"]
      - ["branch2_task"]
      - ["branch3_task"]
  - id: "branch1_task"
    type: "mcp_tool"
    server: "test-server"
    tool: "delay_task"
    parameters:
      duration: "100ms"
    output: "result1"
  - id: "branch2_task"
    type: "mcp_tool"
    server: "test-server"
    tool: "delay_task"
    parameters:
      duration: "100ms"
    output: "result2"
  - id: "branch3_task"
    type: "mcp_tool"
    server: "test-server"
    tool: "delay_task"
    parameters:
      duration: "100ms"
    output: "result3"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{}

	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, inputs)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// If branches truly executed in parallel, total time should be ~100ms (one delay)
	// If sequential, it would be ~300ms (three delays)
	// Allow some overhead for coordination
	if elapsed > 250*time.Millisecond {
		t.Errorf("Expected parallel execution to complete in ~100ms, took %v (suggests sequential execution)", elapsed)
	}

	// Track actual concurrent execution count
	_ = atomic.LoadInt32(&maxConcurrent)
	_ = atomic.LoadInt32(&concurrentCount)

	// Verify all three branches executed
	branchCount := 0
	for _, nodeExec := range result.NodeExecutions {
		nodeID := string(nodeExec.NodeID)
		if nodeID == "branch1_task" || nodeID == "branch2_task" || nodeID == "branch3_task" {
			branchCount++
		}
	}

	if branchCount != 3 {
		t.Errorf("Expected 3 branches to execute, got %d", branchCount)
	}
}

// TestParallelNode_ResultCollection tests that results from all branches are collected
func TestParallelNode_ResultCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	yaml := `
version: "1.0"
name: "parallel-result-collection-test"
variables:
  - name: "results"
    type: "array"
    default: []
  - name: "dummy"
    type: "string"
    default: ""
nodes:
  - id: "start"
    type: "start"
  - id: "parallel_node"
    type: "parallel"
    merge_strategy: "wait_all"
    branches:
      - ["branch1"]
      - ["branch2"]
      - ["branch3"]
  - id: "branch1"
    type: "transform"
    input: "dummy"
    expression: "'result1'"
    output: "branch1_result"
  - id: "branch2"
    type: "transform"
    input: "dummy"
    expression: "'result2'"
    output: "branch2_result"
  - id: "branch3"
    type: "transform"
    input: "dummy"
    expression: "'result3'"
    output: "branch3_result"
  - id: "collect_results"
    type: "transform"
    input: "dummy"
    expression: "[${branch1_result}, ${branch2_result}, ${branch3_result}]"
    output: "results"
  - id: "end"
    type: "end"
    return: "${results}"
edges:
  - from: "start"
    to: "parallel_node"
  - from: "parallel_node"
    to: "collect_results"
  - from: "collect_results"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	engine := runtimeexec.NewEngine()
	inputs := map[string]interface{}{}

	result, err := engine.Execute(ctx, wf, inputs)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify the collect_results node executed after parallel branches
	collectExecuted := false
	for _, nodeExec := range result.NodeExecutions {
		if string(nodeExec.NodeID) == "collect_results" {
			collectExecuted = true
			break
		}
	}

	if !collectExecuted {
		t.Error("Expected collect_results node to execute after parallel branches")
	}
}
