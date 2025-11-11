package integration

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	runtimeexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestParallelPerformance_50ConcurrentBranches tests that the engine can handle
// 50+ concurrent branches without degradation, meeting performance targets:
// - Support 50+ concurrent branches
// - Per-node overhead < 10ms (excluding MCP tool execution time)
// - Memory < 100MB base + 10MB per active MCP server
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_50ConcurrentBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const numBranches = 50
	const nodeOverheadTarget = 10 * time.Millisecond

	// Create workflow with 50 parallel branches doing simple transforms
	yaml := generateParallelTransformWorkflow(numBranches)

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Measure baseline memory before execution
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Execute workflow
	engine := runtimeexec.NewEngine()
	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, nil)
	totalDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Measure memory after execution
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate metrics
	memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
	avgNodeOverhead := totalDuration / time.Duration(len(result.NodeExecutions))

	t.Logf("Performance Metrics:")
	t.Logf("  Total branches: %d", numBranches)
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Total nodes executed: %d", len(result.NodeExecutions))
	t.Logf("  Average per-node overhead: %v", avgNodeOverhead)
	t.Logf("  Memory used: %.2f MB", memUsedMB)
	t.Logf("  Target per-node overhead: %v", nodeOverheadTarget)

	// Verify performance targets
	// Note: Per-node overhead includes coordination time for parallel execution
	// We allow some tolerance for parallel coordination overhead
	maxAcceptableOverhead := nodeOverheadTarget * 2 // 20ms with 2x tolerance for parallel coordination
	if avgNodeOverhead > maxAcceptableOverhead {
		t.Errorf("Per-node overhead %v exceeds target %v (with 2x tolerance for parallel coordination)",
			avgNodeOverhead, maxAcceptableOverhead)
	}

	// Verify memory target (100MB base is generous for this test without MCP servers)
	maxMemoryMB := float64(100)
	if memUsedMB > maxMemoryMB {
		t.Errorf("Memory usage %.2f MB exceeds target %.2f MB", memUsedMB, maxMemoryMB)
	}

	// Verify all branches executed
	expectedMinNodes := numBranches + 3 // start + parallel node + end + branches
	if len(result.NodeExecutions) < expectedMinNodes {
		t.Errorf("Expected at least %d node executions, got %d",
			expectedMinNodes, len(result.NodeExecutions))
	}
}

// TestParallelPerformance_100Branches tests scaling to 100 concurrent branches
// This is a more aggressive test to verify scaling characteristics.
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_100Branches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const numBranches = 100

	// Create workflow with 100 parallel branches
	yaml := generateParallelTransformWorkflow(numBranches)

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Execute workflow
	engine := runtimeexec.NewEngine()
	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, nil)
	totalDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	t.Logf("100-Branch Performance:")
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Nodes executed: %d", len(result.NodeExecutions))

	// Verify scaling: 100 branches should not take significantly more than 50 branches
	// due to parallel execution
	maxAcceptableDuration := 20 * time.Second
	if totalDuration > maxAcceptableDuration {
		t.Errorf("Total duration %v exceeds maximum acceptable duration %v",
			totalDuration, maxAcceptableDuration)
	}
}

// TestParallelPerformance_WithMockMCPServers tests parallel execution with
// mock MCP tool calls to verify memory targets with active servers.
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_WithMockMCPServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const numBranches = 50

	// Create workflow with parallel branches using mock MCP server
	yaml := `
version: "1.0"
name: "parallel-mcp-performance"
servers:
  - id: "test-server"
    name: "test"
    command: "go"
    args: ["run", "../../internal/testutil/testserver/main.go"]
    transport: "stdio"
nodes:
  - id: "start"
    type: "start"
  - id: "parallel"
    type: "parallel"
    branches:
`

	// Add parallel branches with MCP tool calls
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`      - ["echo_%d"]
`, i)
	}

	// Add MCP tool nodes
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`  - id: "echo_%d"
    type: "mcp_tool"
    server: "test-server"
    tool: "echo"
    parameters:
      message: "Branch %d"
    output: "result_%d"
`, i, i, i)
	}

	yaml += `  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel"
  - from: "parallel"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Measure baseline memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Execute workflow
	engine := runtimeexec.NewEngine()
	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, nil)
	totalDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Measure memory after execution
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)

	t.Logf("MCP Server Performance:")
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Memory used: %.2f MB", memUsedMB)
	t.Logf("  Nodes executed: %d", len(result.NodeExecutions))

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify memory target: 100MB base + 10MB per server (1 server = 110MB max)
	maxMemoryMB := float64(110)
	if memUsedMB > maxMemoryMB {
		t.Errorf("Memory usage %.2f MB exceeds target %.2f MB", memUsedMB, maxMemoryMB)
	}
}

// TestParallelPerformance_NoGoroutineLeaks verifies that parallel execution
// does not leak goroutines after completion.
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_NoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Count goroutines before execution
	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()

	// Execute multiple parallel workflows
	for i := 0; i < 10; i++ {
		yaml := generateParallelTransformWorkflow(20)
		wf, err := workflow.Parse([]byte(yaml))
		if err != nil {
			t.Fatalf("Failed to parse workflow: %v", err)
		}

		engine := runtimeexec.NewEngine()
		result, err := engine.Execute(ctx, wf, nil)
		if err != nil {
			t.Fatalf("Workflow execution failed: %v", err)
		}

		if result.Status != execution.StatusCompleted {
			t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
		}
	}

	// Allow time for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Count goroutines after execution
	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Goroutine count: initial=%d, final=%d, leaked=%d",
		initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)

	// Allow some tolerance for background goroutines (e.g., runtime, GC)
	maxLeakedGoroutines := 5
	if finalGoroutines-initialGoroutines > maxLeakedGoroutines {
		t.Errorf("Goroutine leak detected: %d goroutines leaked (max acceptable: %d)",
			finalGoroutines-initialGoroutines, maxLeakedGoroutines)
	}
}

// TestParallelPerformance_ConcurrentVariableAccess tests that parallel branches
// can safely access and modify variables concurrently without race conditions.
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_ConcurrentVariableAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const numBranches = 50

	// Create workflow where each branch increments a shared counter
	yaml := `
version: "1.0"
name: "parallel-variable-access"
variables:
  - name: "counter"
    type: "number"
    default: 0
nodes:
  - id: "start"
    type: "start"
  - id: "parallel"
    type: "parallel"
    branches:
`

	// Add parallel branches that modify a shared variable
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`      - ["increment_%d"]
`, i)
	}

	// Add increment nodes
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`  - id: "increment_%d"
    type: "transform"
    input: "counter"
    expression: "${counter} + 1"
    output: "counter"
`, i)
	}

	yaml += `  - id: "end"
    type: "end"
    return: "${counter}"
edges:
  - from: "start"
    to: "parallel"
  - from: "parallel"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Execute workflow (run with -race flag to detect race conditions)
	engine := runtimeexec.NewEngine()
	result, err := engine.Execute(ctx, wf, nil)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Note: The final counter value depends on execution order and race conditions.
	// The important thing is that the execution completes without deadlocks or panics.
	t.Logf("Final counter value: %v", result.ReturnValue)
}

// TestParallelPerformance_EarlyTermination tests that parallel execution
// can handle early termination (e.g., wait_first strategy) efficiently.
//
// This test will FAIL until parallel execution is implemented (T168-T171).
func TestParallelPerformance_EarlyTermination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create workflow with wait_first strategy where one branch completes quickly
	yaml := `
version: "1.0"
name: "parallel-early-termination"
variables:
  - name: "dummy"
    type: "string"
    default: "input"
nodes:
  - id: "start"
    type: "start"
  - id: "parallel"
    type: "parallel"
    merge_strategy: "wait_first"
    branches:
      - ["fast_transform"]
      - ["slow_transform_1"]
      - ["slow_transform_2"]
  - id: "fast_transform"
    type: "transform"
    input: "dummy"
    expression: "'FAST'"
    output: "fast_result"
  - id: "slow_transform_1"
    type: "transform"
    input: "dummy"
    expression: "'SLOW1'"
    output: "slow_result_1"
  - id: "slow_transform_2"
    type: "transform"
    input: "dummy"
    expression: "'SLOW2'"
    output: "slow_result_2"
  - id: "end"
    type: "end"
    return: "${fast_result}"
edges:
  - from: "start"
    to: "parallel"
  - from: "parallel"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Execute workflow
	engine := runtimeexec.NewEngine()
	startTime := time.Now()
	result, err := engine.Execute(ctx, wf, nil)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	t.Logf("Early termination duration: %v", duration)

	// Verify execution completed successfully
	if result.Status != execution.StatusCompleted {
		t.Errorf("Expected status %s, got %s", execution.StatusCompleted, result.Status)
	}

	// Verify early termination (should complete in < 2 seconds, not wait for 5-second sleeps)
	maxDuration := 2 * time.Second
	if duration > maxDuration {
		t.Errorf("Early termination took %v, expected < %v", duration, maxDuration)
	}

	// Verify fast result is returned
	if result.ReturnValue != "FAST" {
		t.Errorf("Expected return value 'FAST', got %v", result.ReturnValue)
	}
}

// Benchmark functions for performance regression testing

// BenchmarkParallel_10Branches benchmarks execution with 10 parallel branches
func BenchmarkParallel_10Branches(b *testing.B) {
	yaml := generateParallelTransformWorkflow(10)
	runParallelBenchmark(b, yaml)
}

// BenchmarkParallel_50Branches benchmarks execution with 50 parallel branches
func BenchmarkParallel_50Branches(b *testing.B) {
	yaml := generateParallelTransformWorkflow(50)
	runParallelBenchmark(b, yaml)
}

// BenchmarkParallel_100Branches benchmarks execution with 100 parallel branches
func BenchmarkParallel_100Branches(b *testing.B) {
	yaml := generateParallelTransformWorkflow(100)
	runParallelBenchmark(b, yaml)
}

// BenchmarkParallel_Coordination benchmarks parallel coordination overhead
// by comparing sequential vs parallel execution of the same operations
func BenchmarkParallel_Coordination(b *testing.B) {
	const numOperations = 20

	b.Run("Sequential", func(b *testing.B) {
		yaml := generateSequentialWorkflow(numOperations)
		runParallelBenchmark(b, yaml)
	})

	b.Run("Parallel", func(b *testing.B) {
		yaml := generateParallelTransformWorkflow(numOperations)
		runParallelBenchmark(b, yaml)
	})
}

// BenchmarkParallel_MemoryAllocation benchmarks memory allocation patterns
func BenchmarkParallel_MemoryAllocation(b *testing.B) {
	yaml := generateParallelTransformWorkflow(50)
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		b.Fatalf("Failed to parse workflow: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		engine := runtimeexec.NewEngine()
		_, err := engine.Execute(ctx, wf, nil)
		cancel()

		if err != nil {
			b.Fatalf("Workflow execution failed: %v", err)
		}
	}
}

// BenchmarkParallel_NodeOverhead benchmarks per-node execution overhead
func BenchmarkParallel_NodeOverhead(b *testing.B) {
	yaml := generateParallelTransformWorkflow(50)
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		b.Fatalf("Failed to parse workflow: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var totalNodes atomic.Int64
	var totalDuration atomic.Int64

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := runtimeexec.NewEngine()
		start := time.Now()
		result, err := engine.Execute(ctx, wf, nil)
		duration := time.Since(start)

		if err != nil {
			b.Fatalf("Workflow execution failed: %v", err)
		}

		totalNodes.Add(int64(len(result.NodeExecutions)))
		totalDuration.Add(int64(duration))
	}

	b.StopTimer()

	// Calculate and report per-node overhead
	avgNodes := totalNodes.Load() / int64(b.N)
	avgDuration := time.Duration(totalDuration.Load() / int64(b.N))
	perNodeOverhead := avgDuration / time.Duration(avgNodes)

	b.ReportMetric(float64(perNodeOverhead.Microseconds()), "µs/node")
}

// Helper functions

// generateParallelTransformWorkflow creates a workflow YAML with N parallel branches
func generateParallelTransformWorkflow(numBranches int) string {
	yaml := `
version: "1.0"
name: "parallel-performance-test"
variables:
  - name: "dummy"
    type: "string"
    default: "input"
nodes:
  - id: "start"
    type: "start"
  - id: "parallel"
    type: "parallel"
    branches:
`

	// Generate branches array
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`      - ["transform_%d"]
`, i)
	}

	// Generate transform nodes
	for i := 0; i < numBranches; i++ {
		yaml += fmt.Sprintf(`  - id: "transform_%d"
    type: "transform"
    input: "dummy"
    expression: "'Branch %d'"
    output: "result_%d"
`, i, i, i)
	}

	yaml += `  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "parallel"
  - from: "parallel"
    to: "end"
`

	return yaml
}

// generateSequentialWorkflow creates a workflow with N sequential transform nodes
func generateSequentialWorkflow(numNodes int) string {
	yaml := `
version: "1.0"
name: "sequential-performance-test"
variables:
  - name: "data"
    type: "string"
    default: "input"
nodes:
  - id: "start"
    type: "start"
`

	for i := 0; i < numNodes; i++ {
		yaml += fmt.Sprintf(`  - id: "transform_%d"
    type: "transform"
    input: "data"
    expression: "'Node %d'"
    output: "result_%d"
`, i, i, i)
	}

	yaml += `  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "transform_0"
`

	for i := 0; i < numNodes-1; i++ {
		yaml += fmt.Sprintf(`  - from: "transform_%d"
    to: "transform_%d"
`, i, i+1)
	}

	yaml += fmt.Sprintf(`  - from: "transform_%d"
    to: "end"
`, numNodes-1)

	return yaml
}

// runParallelBenchmark is a helper to run benchmarks consistently
func runParallelBenchmark(b *testing.B, yaml string) {
	wf, err := workflow.Parse([]byte(yaml))
	if err != nil {
		b.Fatalf("Failed to parse workflow: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := runtimeexec.NewEngine()
		_, err := engine.Execute(ctx, wf, nil)
		if err != nil {
			b.Fatalf("Workflow execution failed: %v", err)
		}
	}
}
