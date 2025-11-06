package storage

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/require"
)

// BenchmarkLoadExecution_SmallWorkflow benchmarks loading execution with 10 nodes
func BenchmarkLoadExecution_SmallWorkflow(b *testing.B) {
	benchmarkLoadExecution(b, 10)
}

// BenchmarkLoadExecution_TypicalWorkflow benchmarks loading execution with 30 nodes
func BenchmarkLoadExecution_TypicalWorkflow(b *testing.B) {
	benchmarkLoadExecution(b, 30)
}

// BenchmarkLoadExecution_MediumWorkflow benchmarks loading execution with 50 nodes
func BenchmarkLoadExecution_MediumWorkflow(b *testing.B) {
	benchmarkLoadExecution(b, 50)
}

// BenchmarkLoadExecution_LargeWorkflow benchmarks loading execution with 100 nodes
func BenchmarkLoadExecution_LargeWorkflow(b *testing.B) {
	benchmarkLoadExecution(b, 100)
}

// BenchmarkLoadExecution_VeryLargeWorkflow benchmarks loading execution with 500 nodes
func BenchmarkLoadExecution_VeryLargeWorkflow(b *testing.B) {
	benchmarkLoadExecution(b, 500)
}

// benchmarkLoadExecution is the core benchmark helper
func benchmarkLoadExecution(b *testing.B, nodeCount int) {
	// Setup repository
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	repo, err := NewSQLiteExecutionRepositoryWithPath(dbPath)
	require.NoError(b, err)
	defer func() { _ = repo.Close() }()

	// Create test execution with specified number of nodes
	exec, err := execution.NewExecution(
		types.WorkflowID("benchmark-workflow"),
		"1.0.0",
		map[string]interface{}{
			"input_param": "test_value",
			"batch_size":  100,
		},
	)
	require.NoError(b, err)

	require.NoError(b, exec.Start())

	// Add node executions with realistic data
	for i := 0; i < nodeCount; i++ {
		nodeExec := &execution.NodeExecution{
			NodeID:      types.NodeID("node-" + string(rune('a'+i%26))),
			NodeType:    "mcp_tool",
			Status:      execution.NodeStatusCompleted,
			StartedAt:   time.Now().Add(time.Duration(i) * time.Millisecond),
			CompletedAt: time.Now().Add(time.Duration(i+1) * time.Millisecond),
			Inputs: map[string]interface{}{
				"operation":  "process",
				"item_id":    i,
				"batch":      i / 10,
				"parameters": map[string]interface{}{"retry": 3, "timeout": 30},
			},
			Outputs: map[string]interface{}{
				"status":    "success",
				"processed": i * 2,
				"metadata":  map[string]interface{}{"duration_ms": 45, "memory_mb": 128},
				"items":     generateItems(5), // Realistic nested data
			},
			RetryCount: i % 3,
		}
		require.NoError(b, exec.AddNodeExecution(nodeExec))
	}

	require.NoError(b, exec.Complete(map[string]interface{}{
		"total_processed":  nodeCount,
		"success_rate":     0.98,
		"duration_seconds": 123.45,
	}))

	// Save execution with all node executions
	require.NoError(b, repo.Save(exec))
	for _, ne := range exec.NodeExecutions {
		require.NoError(b, repo.SaveNodeExecution(ne))
	}

	execID := exec.ID

	// Reset timer to exclude setup
	b.ResetTimer()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		loaded, err := repo.Load(execID)
		if err != nil {
			b.Fatal(err)
		}
		if len(loaded.NodeExecutions) != nodeCount {
			b.Fatalf("expected %d node executions, got %d", nodeCount, len(loaded.NodeExecutions))
		}
	}

	b.StopTimer()

	// Report statistics
	b.ReportMetric(float64(nodeCount), "nodes")
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/1e6, "ms/op")
}

// BenchmarkLoadNodeExecutions_Sequential benchmarks sequential node execution loading
func BenchmarkLoadNodeExecutions_Sequential(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	repo, err := NewSQLiteExecutionRepositoryWithPath(dbPath)
	require.NoError(b, err)
	defer func() { _ = repo.Close() }()

	// Create execution
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(b, err)
	require.NoError(b, exec.Start())

	// Add 50 nodes
	for i := 0; i < 50; i++ {
		nodeExec := &execution.NodeExecution{
			NodeID:      types.NodeID("node-" + string(rune('a'+i%26))),
			NodeType:    "mcp_tool",
			Status:      execution.NodeStatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: time.Now().Add(100 * time.Millisecond),
			Inputs:      map[string]interface{}{"index": i},
			Outputs:     map[string]interface{}{"result": i * 2},
		}
		require.NoError(b, exec.AddNodeExecution(nodeExec))
		require.NoError(b, repo.SaveNodeExecution(nodeExec))
	}

	require.NoError(b, repo.Save(exec))
	execID := exec.ID

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nodeExecs, err := repo.loadNodeExecutions(execID)
		if err != nil {
			b.Fatal(err)
		}
		if len(nodeExecs) != 50 {
			b.Fatalf("expected 50 node executions, got %d", len(nodeExecs))
		}
	}
}

// BenchmarkJSONDeserialization benchmarks JSON unmarshaling performance
func BenchmarkJSONDeserialization(b *testing.B) {
	// Realistic node execution data
	data := map[string]interface{}{
		"operation":  "process_batch",
		"batch_id":   12345,
		"items":      generateItems(20),
		"parameters": map[string]interface{}{"retry": 3, "timeout": 30},
		"metadata":   map[string]interface{}{"source": "api", "version": "2.1.0"},
	}

	jsonData, err := json.Marshal(data)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := json.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(len(jsonData)), "bytes")
}

// BenchmarkSave benchmarks saving execution with node executions
func BenchmarkSave(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	repo, err := NewSQLiteExecutionRepositoryWithPath(dbPath)
	require.NoError(b, err)
	defer func() { _ = repo.Close() }()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		exec, err := execution.NewExecution(
			types.WorkflowID("bench-workflow"),
			"1.0.0",
			nil,
		)
		require.NoError(b, err)

		require.NoError(b, exec.Start())

		// Add 30 typical nodes
		for j := 0; j < 30; j++ {
			nodeExec := &execution.NodeExecution{
				NodeID:      types.NodeID("node-" + string(rune('a'+j%26))),
				NodeType:    "mcp_tool",
				Status:      execution.NodeStatusCompleted,
				StartedAt:   time.Now(),
				CompletedAt: time.Now().Add(50 * time.Millisecond),
				Inputs:      map[string]interface{}{"index": j},
				Outputs:     map[string]interface{}{"result": j * 2},
			}
			require.NoError(b, exec.AddNodeExecution(nodeExec))
		}

		require.NoError(b, exec.Complete(nil))
		require.NoError(b, repo.Save(exec))

		for _, ne := range exec.NodeExecutions {
			require.NoError(b, repo.SaveNodeExecution(ne))
		}
	}
}

// Helper function to generate realistic nested data
func generateItems(count int) []map[string]interface{} {
	items := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = map[string]interface{}{
			"id":     i,
			"name":   "item_" + string(rune('a'+i%26)),
			"value":  i * 10,
			"status": "processed",
		}
	}
	return items
}
