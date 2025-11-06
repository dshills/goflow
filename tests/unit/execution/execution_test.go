package execution

import (
	"reflect"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// TestExecutionCreation tests creating new Execution entities
func TestExecutionCreation(t *testing.T) {
	tests := []struct {
		name           string
		workflowID     types.WorkflowID
		workflowVer    string
		inputs         map[string]interface{}
		wantErr        bool
		validateFields func(*testing.T, *execution.Execution)
	}{
		{
			name:        "valid execution creation",
			workflowID:  types.WorkflowID("wf-123"),
			workflowVer: "1.0.0",
			inputs: map[string]interface{}{
				"param1": "value1",
				"param2": 42,
			},
			wantErr: false,
			validateFields: func(t *testing.T, e *execution.Execution) {
				if e.ID == "" {
					t.Error("Execution ID should not be empty")
				}
				if e.WorkflowID != types.WorkflowID("wf-123") {
					t.Errorf("WorkflowID = %v, want wf-123", e.WorkflowID)
				}
				if e.WorkflowVersion != "1.0.0" {
					t.Errorf("WorkflowVersion = %v, want 1.0.0", e.WorkflowVersion)
				}
				if e.Status != execution.StatusPending {
					t.Errorf("Status = %v, want Pending", e.Status)
				}
				if e.StartedAt.IsZero() {
					t.Error("StartedAt should be set")
				}
			},
		},
		{
			name:        "empty workflow ID should fail",
			workflowID:  types.WorkflowID(""),
			workflowVer: "1.0.0",
			inputs:      nil,
			wantErr:     true,
		},
		{
			name:        "empty workflow version should fail",
			workflowID:  types.WorkflowID("wf-123"),
			workflowVer: "",
			inputs:      nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(tt.workflowID, tt.workflowVer, tt.inputs)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFields != nil {
				tt.validateFields(t, exec)
			}
		})
	}
}

// TestExecutionStates tests the execution state machine
func TestExecutionStates(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  execution.Status
		operation      func(*execution.Execution) error
		expectedStatus execution.Status
		wantErr        bool
	}{
		{
			name:          "pending to running via Start",
			initialStatus: execution.StatusPending,
			operation: func(e *execution.Execution) error {
				return e.Start()
			},
			expectedStatus: execution.StatusRunning,
			wantErr:        false,
		},
		{
			name:          "running to completed via Complete",
			initialStatus: execution.StatusRunning,
			operation: func(e *execution.Execution) error {
				return e.Complete(map[string]interface{}{"result": "success"})
			},
			expectedStatus: execution.StatusCompleted,
			wantErr:        false,
		},
		{
			name:          "running to failed via Fail",
			initialStatus: execution.StatusRunning,
			operation: func(e *execution.Execution) error {
				return e.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "test error",
				})
			},
			expectedStatus: execution.StatusFailed,
			wantErr:        false,
		},
		{
			name:          "running to cancelled via Cancel",
			initialStatus: execution.StatusRunning,
			operation: func(e *execution.Execution) error {
				return e.Cancel()
			},
			expectedStatus: execution.StatusCancelled,
			wantErr:        false,
		},
		{
			name:          "invalid transition: pending to completed",
			initialStatus: execution.StatusPending,
			operation: func(e *execution.Execution) error {
				return e.Complete(nil)
			},
			expectedStatus: execution.StatusPending, // Should remain unchanged
			wantErr:        true,
		},
		{
			name:          "invalid transition: completed to running",
			initialStatus: execution.StatusCompleted,
			operation: func(e *execution.Execution) error {
				return e.Start()
			},
			expectedStatus: execution.StatusCompleted, // Should remain unchanged
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(
				types.WorkflowID("wf-test"),
				"1.0.0",
				nil,
			)
			if err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}

			// Set initial status (bypassing state machine for test setup)
			exec.SetStatusForTest(tt.initialStatus)

			err = tt.operation(exec)

			if (err != nil) != tt.wantErr {
				t.Errorf("operation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if exec.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", exec.Status, tt.expectedStatus)
			}
		})
	}
}

// TestExecutionTracking tests timestamp and duration tracking
func TestExecutionTracking(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.Execution) error
		validate func(*testing.T, *execution.Execution)
	}{
		{
			name: "start sets StartedAt timestamp",
			setup: func(e *execution.Execution) error {
				return e.Start()
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if e.StartedAt.IsZero() {
					t.Error("StartedAt should be set after Start()")
				}
				if !e.CompletedAt.IsZero() {
					t.Error("CompletedAt should be zero when running")
				}
			},
		},
		{
			name: "complete sets CompletedAt timestamp",
			setup: func(e *execution.Execution) error {
				if err := e.Start(); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond) // Ensure duration > 0
				return e.Complete(nil)
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if e.CompletedAt.IsZero() {
					t.Error("CompletedAt should be set after Complete()")
				}
				if !e.CompletedAt.After(e.StartedAt) {
					t.Error("CompletedAt should be after StartedAt")
				}
				duration := e.Duration()
				if duration <= 0 {
					t.Errorf("Duration() = %v, want > 0", duration)
				}
			},
		},
		{
			name: "fail sets CompletedAt timestamp",
			setup: func(e *execution.Execution) error {
				if err := e.Start(); err != nil {
					return err
				}
				time.Sleep(10 * time.Millisecond)
				return e.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "test error",
				})
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if e.CompletedAt.IsZero() {
					t.Error("CompletedAt should be set after Fail()")
				}
				if e.Error == nil {
					t.Error("Error should be set after Fail()")
				}
				if e.Error.Message != "test error" {
					t.Errorf("Error.Message = %v, want 'test error'", e.Error.Message)
				}
			},
		},
		{
			name: "cancel sets CompletedAt timestamp",
			setup: func(e *execution.Execution) error {
				if err := e.Start(); err != nil {
					return err
				}
				return e.Cancel()
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if e.CompletedAt.IsZero() {
					t.Error("CompletedAt should be set after Cancel()")
				}
				if e.Status != execution.StatusCancelled {
					t.Errorf("Status = %v, want Cancelled", e.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(
				types.WorkflowID("wf-test"),
				"1.0.0",
				nil,
			)
			if err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}

			if err := tt.setup(exec); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, exec)
		})
	}
}

// TestExecutionWorkflowReference tests that execution maintains valid workflow reference
func TestExecutionWorkflowReference(t *testing.T) {
	tests := []struct {
		name        string
		workflowID  types.WorkflowID
		workflowVer string
		wantErr     bool
	}{
		{
			name:        "valid workflow reference",
			workflowID:  types.WorkflowID("wf-abc-123"),
			workflowVer: "2.1.0",
			wantErr:     false,
		},
		{
			name:        "empty workflow ID",
			workflowID:  types.WorkflowID(""),
			workflowVer: "1.0.0",
			wantErr:     true,
		},
		{
			name:        "empty workflow version",
			workflowID:  types.WorkflowID("wf-123"),
			workflowVer: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(tt.workflowID, tt.workflowVer, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if exec.WorkflowID != tt.workflowID {
					t.Errorf("WorkflowID = %v, want %v", exec.WorkflowID, tt.workflowID)
				}
				if exec.WorkflowVersion != tt.workflowVer {
					t.Errorf("WorkflowVersion = %v, want %v", exec.WorkflowVersion, tt.workflowVer)
				}
			}
		})
	}
}

// TestExecutionNodeExecutions tests node execution history tracking
func TestExecutionNodeExecutions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.Execution) error
		validate func(*testing.T, *execution.Execution)
	}{
		{
			name: "add node execution to history",
			setup: func(e *execution.Execution) error {
				nodeExec := &execution.NodeExecution{
					NodeID:      types.NodeID("node-1"),
					NodeType:    "mcp_tool",
					Status:      execution.NodeStatusCompleted,
					StartedAt:   time.Now(),
					CompletedAt: time.Now(),
				}
				return e.AddNodeExecution(nodeExec)
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if len(e.NodeExecutions) != 1 {
					t.Errorf("NodeExecutions length = %d, want 1", len(e.NodeExecutions))
				}
				if e.NodeExecutions[0].NodeID != types.NodeID("node-1") {
					t.Errorf("NodeID = %v, want node-1", e.NodeExecutions[0].NodeID)
				}
			},
		},
		{
			name: "node executions maintain order",
			setup: func(e *execution.Execution) error {
				for i := 1; i <= 3; i++ {
					nodeExec := &execution.NodeExecution{
						NodeID:   types.NodeID(string(rune('a' + i - 1))),
						NodeType: "test",
						Status:   execution.NodeStatusCompleted,
					}
					if err := e.AddNodeExecution(nodeExec); err != nil {
						return err
					}
				}
				return nil
			},
			validate: func(t *testing.T, e *execution.Execution) {
				if len(e.NodeExecutions) != 3 {
					t.Errorf("NodeExecutions length = %d, want 3", len(e.NodeExecutions))
				}
				expected := []types.NodeID{"a", "b", "c"}
				for i, nodeExec := range e.NodeExecutions {
					if nodeExec.NodeID != expected[i] {
						t.Errorf("NodeExecutions[%d].NodeID = %v, want %v", i, nodeExec.NodeID, expected[i])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(
				types.WorkflowID("wf-test"),
				"1.0.0",
				nil,
			)
			if err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}

			if err := tt.setup(exec); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, exec)
		})
	}
}

// TestExecutionReturnValue tests setting and retrieving return values
func TestExecutionReturnValue(t *testing.T) {
	tests := []struct {
		name        string
		returnValue interface{}
		wantErr     bool
	}{
		{
			name:        "string return value",
			returnValue: "success",
			wantErr:     false,
		},
		{
			name:        "map return value",
			returnValue: map[string]interface{}{"status": "ok", "count": 42},
			wantErr:     false,
		},
		{
			name:        "nil return value",
			returnValue: nil,
			wantErr:     false,
		},
		{
			name:        "array return value",
			returnValue: []interface{}{"a", "b", "c"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := execution.NewExecution(
				types.WorkflowID("wf-test"),
				"1.0.0",
				nil,
			)
			if err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}

			if err := exec.Start(); err != nil {
				t.Fatalf("Failed to start execution: %v", err)
			}

			err = exec.Complete(tt.returnValue)

			if (err != nil) != tt.wantErr {
				t.Errorf("Complete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(exec.ReturnValue, tt.returnValue) {
				t.Errorf("ReturnValue = %v, want %v", exec.ReturnValue, tt.returnValue)
			}
		})
	}
}
