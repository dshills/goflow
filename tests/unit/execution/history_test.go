package execution

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHistoryListWithPagination tests listing executions with pagination support
func TestHistoryListWithPagination(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create 25 test executions across different workflows
	executions := createTestExecutions(t, repo, 25)

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
		wantFirst types.ExecutionID // Expected first execution ID in results
		wantLast  types.ExecutionID // Expected last execution ID in results
	}{
		{
			name:      "first page (10 items)",
			limit:     10,
			offset:    0,
			wantCount: 10,
			wantFirst: executions[24].ID, // Most recent first (DESC order)
			wantLast:  executions[15].ID,
		},
		{
			name:      "second page (10 items)",
			limit:     10,
			offset:    10,
			wantCount: 10,
			wantFirst: executions[14].ID,
			wantLast:  executions[5].ID,
		},
		{
			name:      "third page (partial - 5 items)",
			limit:     10,
			offset:    20,
			wantCount: 5,
			wantFirst: executions[4].ID,
			wantLast:  executions[0].ID,
		},
		{
			name:      "offset beyond total count",
			limit:     10,
			offset:    30,
			wantCount: 0,
		},
		{
			name:      "large limit returns all",
			limit:     100,
			offset:    0,
			wantCount: 25,
			wantFirst: executions[24].ID,
			wantLast:  executions[0].ID,
		},
		{
			name:      "zero limit uses default",
			limit:     0,
			offset:    0,
			wantCount: 25, // Should return all when limit is 0
			wantFirst: executions[24].ID,
			wantLast:  executions[0].ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will initially FAIL - method doesn't exist yet
			result, err := repo.List(execution.ListOptions{
				Limit:  tt.limit,
				Offset: tt.offset,
			})

			require.NoError(t, err)
			assert.Len(t, result.Executions, tt.wantCount, "incorrect result count")
			assert.Equal(t, 25, result.TotalCount, "incorrect total count")

			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantFirst, result.Executions[0].ID, "incorrect first execution")
				assert.Equal(t, tt.wantLast, result.Executions[len(result.Executions)-1].ID, "incorrect last execution")
			}
		})
	}
}

// TestHistoryFilterByStatus tests filtering executions by status
func TestHistoryFilterByStatus(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create executions with different statuses
	// 10 completed, 5 failed, 3 running, 2 cancelled, 5 pending
	statusCounts := map[execution.Status]int{
		execution.StatusCompleted: 10,
		execution.StatusFailed:    5,
		execution.StatusRunning:   3,
		execution.StatusCancelled: 2,
		execution.StatusPending:   5,
	}

	createTestExecutionsWithStatuses(t, repo, statusCounts)

	tests := []struct {
		name      string
		status    execution.Status
		wantCount int
	}{
		{
			name:      "filter completed executions",
			status:    execution.StatusCompleted,
			wantCount: 10,
		},
		{
			name:      "filter failed executions",
			status:    execution.StatusFailed,
			wantCount: 5,
		},
		{
			name:      "filter running executions",
			status:    execution.StatusRunning,
			wantCount: 3,
		},
		{
			name:      "filter cancelled executions",
			status:    execution.StatusCancelled,
			wantCount: 2,
		},
		{
			name:      "filter pending executions",
			status:    execution.StatusPending,
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will initially FAIL - method doesn't exist yet
			result, err := repo.List(execution.ListOptions{
				Status: &tt.status,
			})

			require.NoError(t, err)
			assert.Len(t, result.Executions, tt.wantCount, "incorrect count for status %s", tt.status)

			// Verify all results have the expected status
			for _, exec := range result.Executions {
				assert.Equal(t, tt.status, exec.Status, "execution has wrong status")
			}
		})
	}
}

// TestHistoryFilterByWorkflowID tests filtering executions by workflow ID
func TestHistoryFilterByWorkflowID(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create executions for different workflows
	workflowCounts := map[types.WorkflowID]int{
		"workflow-auth":         7,
		"workflow-payment":      12,
		"workflow-notification": 5,
	}

	createTestExecutionsForWorkflows(t, repo, workflowCounts)

	tests := []struct {
		name       string
		workflowID types.WorkflowID
		wantCount  int
	}{
		{
			name:       "filter auth workflow executions",
			workflowID: "workflow-auth",
			wantCount:  7,
		},
		{
			name:       "filter payment workflow executions",
			workflowID: "workflow-payment",
			wantCount:  12,
		},
		{
			name:       "filter notification workflow executions",
			workflowID: "workflow-notification",
			wantCount:  5,
		},
		{
			name:       "filter non-existent workflow",
			workflowID: "workflow-nonexistent",
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will initially FAIL - method doesn't exist yet
			result, err := repo.List(execution.ListOptions{
				WorkflowID: &tt.workflowID,
			})

			require.NoError(t, err)
			assert.Len(t, result.Executions, tt.wantCount, "incorrect count for workflow %s", tt.workflowID)

			// Verify all results have the expected workflow ID
			for _, exec := range result.Executions {
				assert.Equal(t, tt.workflowID, exec.WorkflowID, "execution has wrong workflow ID")
			}
		})
	}
}

// TestHistoryFilterByDateRange tests filtering executions by date range
func TestHistoryFilterByDateRange(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create executions spread across 30 days
	now := time.Now()
	baseTime := now.Add(-30 * 24 * time.Hour)

	executions := make([]*execution.Execution, 30)
	for i := 0; i < 30; i++ {
		exec, err := execution.NewExecution(
			types.WorkflowID("test-workflow"),
			"1.0.0",
			nil,
		)
		require.NoError(t, err)

		// Set StartedAt to specific date (baseTime + i days)
		exec.StartedAt = baseTime.Add(time.Duration(i) * 24 * time.Hour)

		require.NoError(t, repo.Save(exec))
		executions[i] = exec
	}

	tests := []struct {
		name      string
		startTime *time.Time
		endTime   *time.Time
		wantCount int
		wantFirst int // Index in executions array
		wantLast  int // Index in executions array
	}{
		{
			name:      "last 7 days",
			startTime: timePtr(now.Add(-7 * 24 * time.Hour)),
			endTime:   timePtr(now),
			wantCount: 7,
			wantFirst: 29, // Most recent
			wantLast:  23,
		},
		{
			name:      "days 10-20",
			startTime: timePtr(baseTime.Add(10 * 24 * time.Hour)),
			endTime:   timePtr(baseTime.Add(20 * 24 * time.Hour)),
			wantCount: 10,
			wantFirst: 19, // DESC order
			wantLast:  10,
		},
		{
			name:      "all time (no filters)",
			startTime: nil,
			endTime:   nil,
			wantCount: 30,
			wantFirst: 29,
			wantLast:  0,
		},
		{
			name:      "before first execution",
			startTime: timePtr(baseTime.Add(-10 * 24 * time.Hour)),
			endTime:   timePtr(baseTime.Add(-5 * 24 * time.Hour)),
			wantCount: 0,
		},
		{
			name:      "after last execution",
			startTime: timePtr(now.Add(5 * 24 * time.Hour)),
			endTime:   timePtr(now.Add(10 * 24 * time.Hour)),
			wantCount: 0,
		},
		{
			name:      "only start time (last 15 days)",
			startTime: timePtr(baseTime.Add(15 * 24 * time.Hour)),
			endTime:   nil,
			wantCount: 15,
			wantFirst: 29,
			wantLast:  15,
		},
		{
			name:      "only end time (first 10 days)",
			startTime: nil,
			endTime:   timePtr(baseTime.Add(10 * 24 * time.Hour)),
			wantCount: 10,
			wantFirst: 9,
			wantLast:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will initially FAIL - method doesn't exist yet
			result, err := repo.List(execution.ListOptions{
				StartedAfter:  tt.startTime,
				StartedBefore: tt.endTime,
			})

			require.NoError(t, err)
			assert.Len(t, result.Executions, tt.wantCount, "incorrect count for date range")

			if tt.wantCount > 0 {
				assert.Equal(t, executions[tt.wantFirst].ID, result.Executions[0].ID, "incorrect first execution")
				assert.Equal(t, executions[tt.wantLast].ID, result.Executions[len(result.Executions)-1].ID, "incorrect last execution")
			}
		})
	}
}

// TestHistorySearchByWorkflowName tests search functionality by workflow name
func TestHistorySearchByWorkflowName(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create executions with different workflow names
	workflows := map[types.WorkflowID]int{
		"user-authentication": 5,
		"user-registration":   4,
		"payment-processing":  3,
		"payment-refund":      2,
		"email-notification":  6,
		"sms-notification":    3,
	}

	createTestExecutionsForWorkflows(t, repo, workflows)

	tests := []struct {
		name          string
		searchTerm    string
		wantWorkflows []types.WorkflowID
		wantMinCount  int
	}{
		{
			name:       "search 'user' workflows",
			searchTerm: "user",
			wantWorkflows: []types.WorkflowID{
				"user-authentication",
				"user-registration",
			},
			wantMinCount: 9, // 5 + 4
		},
		{
			name:       "search 'payment' workflows",
			searchTerm: "payment",
			wantWorkflows: []types.WorkflowID{
				"payment-processing",
				"payment-refund",
			},
			wantMinCount: 5, // 3 + 2
		},
		{
			name:       "search 'notification' workflows",
			searchTerm: "notification",
			wantWorkflows: []types.WorkflowID{
				"email-notification",
				"sms-notification",
			},
			wantMinCount: 9, // 6 + 3
		},
		{
			name:       "search exact match 'email-notification'",
			searchTerm: "email-notification",
			wantWorkflows: []types.WorkflowID{
				"email-notification",
			},
			wantMinCount: 6,
		},
		{
			name:          "search non-existent term",
			searchTerm:    "nonexistent",
			wantWorkflows: []types.WorkflowID{},
			wantMinCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will initially FAIL - method doesn't exist yet
			result, err := repo.List(execution.ListOptions{
				WorkflowNameSearch: &tt.searchTerm,
			})

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result.Executions), tt.wantMinCount, "incorrect result count")

			// Verify all results contain the search term
			foundWorkflows := make(map[types.WorkflowID]bool)
			for _, exec := range result.Executions {
				foundWorkflows[exec.WorkflowID] = true
			}

			for _, expectedWorkflow := range tt.wantWorkflows {
				assert.True(t, foundWorkflows[expectedWorkflow], "expected workflow %s not found", expectedWorkflow)
			}
		})
	}
}

// TestHistoryExecutionDetail tests retrieving detailed execution information
func TestHistoryExecutionDetail(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create execution with full details
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		map[string]interface{}{"param": "value"},
	)
	require.NoError(t, err)

	// Start and complete the execution
	require.NoError(t, exec.Start())

	// Add node executions
	for i := 0; i < 5; i++ {
		nodeExec := &execution.NodeExecution{
			ID:          types.NewNodeExecutionID(),
			ExecutionID: exec.ID,
			NodeID:      types.NodeID("node-" + string(rune('a'+i))),
			NodeType:    "mcp_tool",
			Status:      execution.NodeStatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: time.Now().Add(100 * time.Millisecond),
			Inputs: map[string]interface{}{
				"input": i,
			},
			Outputs: map[string]interface{}{
				"output": i * 2,
			},
		}
		require.NoError(t, exec.AddNodeExecution(nodeExec))
		require.NoError(t, repo.SaveNodeExecution(nodeExec))
	}

	require.NoError(t, exec.Complete(map[string]interface{}{"result": "success"}))
	require.NoError(t, repo.Save(exec))

	// Load execution details
	loaded, err := repo.Load(exec.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all details are present
	assert.Equal(t, exec.ID, loaded.ID)
	assert.Equal(t, exec.WorkflowID, loaded.WorkflowID)
	assert.Equal(t, exec.Status, loaded.Status)
	assert.Len(t, loaded.NodeExecutions, 5, "should have all node executions")

	// Verify node execution details
	for i, nodeExec := range loaded.NodeExecutions {
		assert.Equal(t, types.NodeID("node-"+string(rune('a'+i))), nodeExec.NodeID)
		assert.NotNil(t, nodeExec.Inputs)
		assert.NotNil(t, nodeExec.Outputs)
	}

	// Verify return value
	assert.NotNil(t, loaded.ReturnValue)
}

// TestHistoryConcurrentQueries tests concurrent query operations
func TestHistoryConcurrentQueries(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create 100 test executions
	executions := createTestExecutions(t, repo, 100)

	// Run multiple concurrent queries
	concurrency := 10
	queriesPerGoroutine := 20

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*queriesPerGoroutine)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < queriesPerGoroutine; j++ {
				// Alternate between different query types
				switch j % 4 {
				case 0:
					// List with pagination
					_, err := repo.List(execution.ListOptions{
						Limit:  10,
						Offset: j % 50,
					})
					if err != nil {
						errors <- err
					}

				case 1:
					// Load specific execution
					idx := (workerID + j) % len(executions)
					_, err := repo.Load(executions[idx].ID)
					if err != nil {
						errors <- err
					}

				case 2:
					// Filter by status
					status := execution.StatusCompleted
					_, err := repo.List(execution.ListOptions{
						Status: &status,
					})
					if err != nil {
						errors <- err
					}

				case 3:
					// Filter by workflow
					workflowID := types.WorkflowID("test-workflow")
					_, err := repo.List(execution.ListOptions{
						WorkflowID: &workflowID,
					})
					if err != nil {
						errors <- err
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorList := make([]error, 0)
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "concurrent queries should not produce errors")
}

// TestHistoryQueryPerformance tests query performance with large datasets
func TestHistoryQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create 1000 test executions
	t.Log("Creating 1000 test executions...")
	startCreate := time.Now()
	executions := createTestExecutions(t, repo, 1000)
	createDuration := time.Since(startCreate)
	t.Logf("Created 1000 executions in %v (%.2f exec/sec)", createDuration, 1000.0/createDuration.Seconds())

	tests := []struct {
		name        string
		query       func() error
		maxDuration time.Duration
		description string
	}{
		{
			name: "list first page (100 items)",
			query: func() error {
				_, err := repo.List(execution.ListOptions{
					Limit:  100,
					Offset: 0,
				})
				return err
			},
			maxDuration: 50 * time.Millisecond,
			description: "listing first 100 executions",
		},
		{
			name: "list deep pagination (offset 900)",
			query: func() error {
				_, err := repo.List(execution.ListOptions{
					Limit:  100,
					Offset: 900,
				})
				return err
			},
			maxDuration: 100 * time.Millisecond,
			description: "listing with deep offset",
		},
		{
			name: "filter by status",
			query: func() error {
				status := execution.StatusCompleted
				_, err := repo.List(execution.ListOptions{
					Status: &status,
				})
				return err
			},
			maxDuration: 100 * time.Millisecond,
			description: "filtering by status",
		},
		{
			name: "filter by workflow ID",
			query: func() error {
				workflowID := types.WorkflowID("test-workflow")
				_, err := repo.List(execution.ListOptions{
					WorkflowID: &workflowID,
				})
				return err
			},
			maxDuration: 100 * time.Millisecond,
			description: "filtering by workflow ID",
		},
		{
			name: "load single execution with node executions",
			query: func() error {
				_, err := repo.Load(executions[500].ID)
				return err
			},
			maxDuration: 50 * time.Millisecond,
			description: "loading single execution details",
		},
		{
			name: "date range query (last 7 days)",
			query: func() error {
				startTime := time.Now().Add(-7 * 24 * time.Hour)
				_, err := repo.List(execution.ListOptions{
					StartedAfter: &startTime,
				})
				return err
			},
			maxDuration: 100 * time.Millisecond,
			description: "filtering by date range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Warm up
			_ = tt.query()

			// Measure performance over 10 runs
			var totalDuration time.Duration
			runs := 10

			for i := 0; i < runs; i++ {
				start := time.Now()
				err := tt.query()
				duration := time.Since(start)
				totalDuration += duration

				require.NoError(t, err)
			}

			avgDuration := totalDuration / time.Duration(runs)
			t.Logf("%s: avg %v over %d runs", tt.description, avgDuration, runs)

			assert.Less(t, avgDuration.Milliseconds(), tt.maxDuration.Milliseconds(),
				"query too slow: %v (max %v)", avgDuration, tt.maxDuration)
		})
	}
}

// TestHistoryCombinedFilters tests combining multiple filters
func TestHistoryCombinedFilters(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// Create diverse set of executions
	now := time.Now()
	baseTime := now.Add(-10 * 24 * time.Hour)

	statusCounts := map[execution.Status]int{
		execution.StatusCompleted: 10,
		execution.StatusFailed:    5,
		execution.StatusRunning:   3,
	}

	workflowIDs := []types.WorkflowID{
		"workflow-a",
		"workflow-b",
	}

	for _, workflowID := range workflowIDs {
		for status, count := range statusCounts {
			for i := 0; i < count; i++ {
				exec, err := execution.NewExecution(workflowID, "1.0.0", nil)
				require.NoError(t, err)

				exec.StartedAt = baseTime.Add(time.Duration(i) * 24 * time.Hour)
				exec.SetStatusForTest(status)

				if status.IsTerminal() {
					exec.CompletedAt = exec.StartedAt.Add(1 * time.Hour)
				}

				require.NoError(t, repo.Save(exec))
			}
		}
	}

	tests := []struct {
		name      string
		options   execution.ListOptions
		wantCount int
	}{
		{
			name: "workflow + status",
			options: execution.ListOptions{
				WorkflowID: ptrWorkflowID("workflow-a"),
				Status:     ptrStatus(execution.StatusCompleted),
			},
			wantCount: 10,
		},
		{
			name: "workflow + status + date range",
			options: execution.ListOptions{
				WorkflowID:    ptrWorkflowID("workflow-a"),
				Status:        ptrStatus(execution.StatusCompleted),
				StartedAfter:  timePtr(baseTime.Add(5 * 24 * time.Hour)),
				StartedBefore: timePtr(now),
			},
			wantCount: 5, // Completed executions from day 5-10
		},
		{
			name: "status + pagination",
			options: execution.ListOptions{
				Status: ptrStatus(execution.StatusCompleted),
				Limit:  15,
				Offset: 0,
			},
			wantCount: 15, // 10 per workflow * 2 = 20 total, limited to 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.List(tt.options)
			require.NoError(t, err)
			assert.Len(t, result.Executions, tt.wantCount)
		})
	}
}

// TestHistoryEmptyResults tests handling of empty result sets
func TestHistoryEmptyResults(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	// No executions created

	tests := []struct {
		name    string
		options execution.ListOptions
	}{
		{
			name:    "list all (empty)",
			options: execution.ListOptions{},
		},
		{
			name: "filter by non-existent workflow",
			options: execution.ListOptions{
				WorkflowID: ptrWorkflowID("nonexistent"),
			},
		},
		{
			name: "filter by status (empty)",
			options: execution.ListOptions{
				Status: ptrStatus(execution.StatusCompleted),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.List(tt.options)
			require.NoError(t, err)
			assert.Empty(t, result.Executions)
			assert.Equal(t, 0, result.TotalCount)
		})
	}
}

// TestHistoryInvalidFilters tests handling of invalid filter values
func TestHistoryInvalidFilters(t *testing.T) {
	repo := setupTestRepository(t)
	defer cleanupTestRepository(t, repo)

	tests := []struct {
		name    string
		options execution.ListOptions
		wantErr bool
	}{
		{
			name: "negative limit",
			options: execution.ListOptions{
				Limit: -10,
			},
			wantErr: true,
		},
		{
			name: "negative offset",
			options: execution.ListOptions{
				Offset: -5,
			},
			wantErr: true,
		},
		{
			name: "end time before start time",
			options: execution.ListOptions{
				StartedAfter:  timePtr(time.Now()),
				StartedBefore: timePtr(time.Now().Add(-24 * time.Hour)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.List(tt.options)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions

func setupTestRepository(t *testing.T) *storage.SQLiteExecutionRepository {
	t.Helper()

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := storage.NewSQLiteExecutionRepositoryWithPath(dbPath)
	require.NoError(t, err, "failed to create test repository")

	return repo
}

func cleanupTestRepository(t *testing.T, repo *storage.SQLiteExecutionRepository) {
	t.Helper()
	if err := repo.Close(); err != nil {
		t.Logf("warning: failed to close repository: %v", err)
	}
}

func createTestExecutions(t *testing.T, repo *storage.SQLiteExecutionRepository, count int) []*execution.Execution {
	t.Helper()

	executions := make([]*execution.Execution, count)
	for i := 0; i < count; i++ {
		exec, err := execution.NewExecution(
			types.WorkflowID("test-workflow"),
			"1.0.0",
			nil,
		)
		require.NoError(t, err)

		// Stagger creation times
		exec.StartedAt = time.Now().Add(time.Duration(-count+i) * time.Second)

		// Complete some executions
		if i%2 == 0 {
			exec.SetStatusForTest(execution.StatusCompleted)
			exec.CompletedAt = exec.StartedAt.Add(1 * time.Second)
		}

		require.NoError(t, repo.Save(exec))
		executions[i] = exec
	}

	return executions
}

func createTestExecutionsWithStatuses(t *testing.T, repo *storage.SQLiteExecutionRepository, counts map[execution.Status]int) {
	t.Helper()

	for status, count := range counts {
		for i := 0; i < count; i++ {
			exec, err := execution.NewExecution(
				types.WorkflowID("test-workflow"),
				"1.0.0",
				nil,
			)
			require.NoError(t, err)

			exec.SetStatusForTest(status)
			if status.IsTerminal() {
				exec.CompletedAt = time.Now()
			}

			require.NoError(t, repo.Save(exec))
		}
	}
}

func createTestExecutionsForWorkflows(t *testing.T, repo *storage.SQLiteExecutionRepository, counts map[types.WorkflowID]int) {
	t.Helper()

	for workflowID, count := range counts {
		for i := 0; i < count; i++ {
			exec, err := execution.NewExecution(
				workflowID,
				"1.0.0",
				nil,
			)
			require.NoError(t, err)

			// Vary statuses
			if i%2 == 0 {
				exec.SetStatusForTest(execution.StatusCompleted)
				exec.CompletedAt = time.Now()
			}

			require.NoError(t, repo.Save(exec))
		}
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func ptrWorkflowID(id types.WorkflowID) *types.WorkflowID {
	return &id
}

func ptrStatus(s execution.Status) *execution.Status {
	return &s
}
