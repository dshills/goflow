package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// T090: TUI Component Tests for Execution Monitor View
//
// These tests follow test-first development - they will FAIL initially
// because the ExecutionMonitor implementation does not exist yet.
//
// The tests cover:
// 1. Execution monitor view rendering
// 2. Real-time node highlighting during execution
// 3. Variable inspector panel display
// 4. Error detail view
// 5. Execution log viewer
// 6. Performance metrics display
// 7. Keyboard navigation in monitor view
// 8. View refresh on execution events

// MockExecutionRepository implements a simple in-memory execution repository for testing
type MockExecutionRepository struct {
	executions map[types.ExecutionID]*execution.Execution
}

func NewMockExecutionRepository() *MockExecutionRepository {
	return &MockExecutionRepository{
		executions: make(map[types.ExecutionID]*execution.Execution),
	}
}

func (m *MockExecutionRepository) Save(exec *execution.Execution) error {
	m.executions[exec.ID] = exec
	return nil
}

func (m *MockExecutionRepository) FindByID(id types.ExecutionID) (*execution.Execution, error) {
	if exec, ok := m.executions[id]; ok {
		return exec, nil
	}
	return nil, fmt.Errorf("execution not found: %s", id)
}

func (m *MockExecutionRepository) List() ([]*execution.Execution, error) {
	result := make([]*execution.Execution, 0, len(m.executions))
	for _, exec := range m.executions {
		result = append(result, exec)
	}
	return result, nil
}

// createTestExecution creates a test execution for testing
func createTestExecution(wf *workflow.Workflow) *execution.Execution {
	exec, _ := execution.NewExecution(
		types.WorkflowID(wf.ID),
		wf.Version,
		map[string]interface{}{
			"input_file": "test.txt",
			"count":      42,
		},
	)
	return exec
}

// createTestWorkflowForExecution creates a test workflow for execution monitoring
func createTestWorkflowForExecution() *workflow.Workflow {
	wf, _ := workflow.NewWorkflow("test-execution-workflow", "Test workflow for execution monitoring")

	// Add nodes
	start := &workflow.StartNode{ID: "start"}
	tool1 := &workflow.MCPToolNode{
		ID:             "tool-1",
		ServerID:       "fs",
		ToolName:       "read_file",
		OutputVariable: "file_content",
	}
	tool2 := &workflow.MCPToolNode{
		ID:             "tool-2",
		ServerID:       "fs",
		ToolName:       "write_file",
		OutputVariable: "write_result",
	}
	end := &workflow.EndNode{ID: "end"}

	wf.AddNode(start)
	wf.AddNode(tool1)
	wf.AddNode(tool2)
	wf.AddNode(end)

	// Add edges
	wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
	wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})

	return wf
}

// TestExecutionMonitorRender tests basic rendering of execution monitor view
func TestExecutionMonitorRender(t *testing.T) {
	tests := []struct {
		name           string
		setupExecution func() *execution.Execution
		setupWorkflow  func() *workflow.Workflow
		wantStatus     string
		wantNodeCount  int
	}{
		{
			name: "render pending execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				return createTestExecution(wf)
			},
			setupWorkflow: createTestWorkflowForExecution,
			wantStatus:    "pending",
			wantNodeCount: 4,
		},
		{
			name: "render running execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			setupWorkflow: createTestWorkflowForExecution,
			wantStatus:    "running",
			wantNodeCount: 4,
		},
		{
			name: "render completed execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Complete(map[string]interface{}{"result": "success"})
				return exec
			},
			setupWorkflow: createTestWorkflowForExecution,
			wantStatus:    "completed",
			wantNodeCount: 4,
		},
		{
			name: "render failed execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "Tool execution failed",
					NodeID:  "tool-1",
				})
				return exec
			},
			setupWorkflow: createTestWorkflowForExecution,
			wantStatus:    "failed",
			wantNodeCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := tt.setupWorkflow()

			screen := goterm.NewScreen(120, 40)

			// Create execution monitor (this will fail - implementation doesn't exist)
			// Expected error: undefined: NewExecutionMonitor
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			// Render the monitor
			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Verify execution status is displayed
			if !screenContainsText(screen, tt.wantStatus) {
				t.Errorf("Expected status %q not found in screen buffer", tt.wantStatus)
			}

			// Verify execution ID is displayed
			if !screenContainsText(screen, exec.ID.String()) {
				t.Error("Expected execution ID not found in screen buffer")
			}

			// Verify workflow name is displayed
			if !screenContainsText(screen, wf.Name) {
				t.Error("Expected workflow name not found in screen buffer")
			}
		})
	}
}

// TestExecutionMonitorNodeHighlighting tests real-time node highlighting during execution
func TestExecutionMonitorNodeHighlighting(t *testing.T) {
	tests := []struct {
		name               string
		setupExecution     func() *execution.Execution
		activeNodeID       string
		wantHighlighted    bool
		wantHighlightStyle string
	}{
		{
			name: "highlight running node",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add node execution for tool-1 in running state
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				exec.AddNodeExecution(nodeExec)

				return exec
			},
			activeNodeID:       "tool-1",
			wantHighlighted:    true,
			wantHighlightStyle: "running",
		},
		{
			name: "highlight completed node",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add completed node execution
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				nodeExec.Complete(map[string]interface{}{"output": "data"})
				exec.AddNodeExecution(nodeExec)

				return exec
			},
			activeNodeID:       "tool-1",
			wantHighlighted:    true,
			wantHighlightStyle: "completed",
		},
		{
			name: "highlight failed node",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add failed node execution
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				nodeExec.Fail(&execution.NodeError{
					Type:    execution.ErrorTypeExecution,
					Message: "Node failed",
				})
				exec.AddNodeExecution(nodeExec)

				return exec
			},
			activeNodeID:       "tool-1",
			wantHighlighted:    true,
			wantHighlightStyle: "failed",
		},
		{
			name: "no highlight for pending node",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			activeNodeID:       "tool-2",
			wantHighlighted:    false,
			wantHighlightStyle: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Verify node highlighting status
			isHighlighted := monitor.IsNodeHighlighted(tt.activeNodeID)
			if isHighlighted != tt.wantHighlighted {
				t.Errorf("IsNodeHighlighted(%q) = %v, want %v", tt.activeNodeID, isHighlighted, tt.wantHighlighted)
			}

			// Verify highlight style
			if tt.wantHighlighted {
				style := monitor.GetNodeHighlightStyle(tt.activeNodeID)
				if style != tt.wantHighlightStyle {
					t.Errorf("GetNodeHighlightStyle(%q) = %q, want %q", tt.activeNodeID, style, tt.wantHighlightStyle)
				}
			}
		})
	}
}

// TestExecutionMonitorVariableInspector tests variable inspector panel display
func TestExecutionMonitorVariableInspector(t *testing.T) {
	tests := []struct {
		name           string
		setupExecution func() *execution.Execution
		wantVariables  map[string]interface{}
		wantVisible    bool
	}{
		{
			name: "display initial variables",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				return exec
			},
			wantVariables: map[string]interface{}{
				"input_file": "test.txt",
				"count":      42,
			},
			wantVisible: true,
		},
		{
			name: "display updated variables during execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Update context with new variables
				exec.Context.SetVariable("output_file", "result.txt")
				exec.Context.SetVariable("processed", true)

				return exec
			},
			wantVariables: map[string]interface{}{
				"input_file":  "test.txt",
				"count":       42,
				"output_file": "result.txt",
				"processed":   true,
			},
			wantVisible: true,
		},
		{
			name: "display variables with complex types",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Set complex variables
				exec.Context.SetVariable("items", []string{"a", "b", "c"})
				exec.Context.SetVariable("config", map[string]interface{}{
					"timeout": 30,
					"retries": 3,
				})

				return exec
			},
			wantVariables: map[string]interface{}{
				"input_file": "test.txt",
				"count":      42,
				"items":      []string{"a", "b", "c"},
				"config": map[string]interface{}{
					"timeout": 30,
					"retries": 3,
				},
			},
			wantVisible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Get variable inspector panel
			inspector := monitor.GetVariableInspector()
			if inspector == nil {
				t.Fatal("GetVariableInspector() returned nil")
			}

			// Verify visibility
			if inspector.IsVisible() != tt.wantVisible {
				t.Errorf("VariableInspector.IsVisible() = %v, want %v", inspector.IsVisible(), tt.wantVisible)
			}

			// Verify variables are displayed
			displayedVars := inspector.GetDisplayedVariables()
			for name, expectedValue := range tt.wantVariables {
				actualValue, exists := displayedVars[name]
				if !exists {
					t.Errorf("Variable %q not found in inspector", name)
					continue
				}

				// Basic type comparison (would need more sophisticated comparison for complex types)
				if !valuesEqual(actualValue, expectedValue) {
					t.Errorf("Variable %q = %v, want %v", name, actualValue, expectedValue)
				}
			}

			// Verify variable names are in screen buffer
			for name := range tt.wantVariables {
				if !screenContainsText(screen, name) {
					t.Errorf("Variable name %q not found in screen buffer", name)
				}
			}
		})
	}
}

// TestExecutionMonitorErrorDetailView tests error detail view display
func TestExecutionMonitorErrorDetailView(t *testing.T) {
	tests := []struct {
		name           string
		setupExecution func() *execution.Execution
		wantError      bool
		wantMessage    string
		wantNodeID     string
		wantType       string
	}{
		{
			name: "no error for successful execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Complete(nil)
				return exec
			},
			wantError: false,
		},
		{
			name: "display execution error details",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "MCP tool execution failed: file not found",
					NodeID:  "tool-1",
					Context: map[string]interface{}{
						"file_path": "/nonexistent/file.txt",
					},
					Recoverable: true,
				})
				return exec
			},
			wantError:   true,
			wantMessage: "file not found",
			wantNodeID:  "tool-1",
			wantType:    "execution",
		},
		{
			name: "display validation error details",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeValidation,
					Message: "Invalid parameter: count must be positive",
					NodeID:  "tool-2",
					Context: map[string]interface{}{
						"parameter": "count",
						"value":     -1,
					},
					Recoverable: false,
				})
				return exec
			},
			wantError:   true,
			wantMessage: "Invalid parameter",
			wantNodeID:  "tool-2",
			wantType:    "validation",
		},
		{
			name: "display connection error details",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeConnection,
					Message: "Failed to connect to MCP server 'fs'",
					Context: map[string]interface{}{
						"server_id": "fs",
						"timeout":   "5s",
					},
					Recoverable: true,
				})
				return exec
			},
			wantError:   true,
			wantMessage: "Failed to connect",
			wantNodeID:  "",
			wantType:    "connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Get error detail view
			errorView := monitor.GetErrorDetailView()
			if errorView == nil {
				t.Fatal("GetErrorDetailView() returned nil")
			}

			// Verify error visibility
			hasError := errorView.HasError()
			if hasError != tt.wantError {
				t.Errorf("ErrorDetailView.HasError() = %v, want %v", hasError, tt.wantError)
			}

			if !tt.wantError {
				return
			}

			// Verify error message is displayed
			if !screenContainsText(screen, tt.wantMessage) {
				t.Errorf("Error message %q not found in screen buffer", tt.wantMessage)
			}

			// Verify error type is displayed
			if !screenContainsText(screen, tt.wantType) {
				t.Errorf("Error type %q not found in screen buffer", tt.wantType)
			}

			// Verify node ID if applicable
			if tt.wantNodeID != "" {
				if !screenContainsText(screen, tt.wantNodeID) {
					t.Errorf("Node ID %q not found in screen buffer", tt.wantNodeID)
				}
			}

			// Verify error details are accessible
			errorDetails := errorView.GetErrorDetails()
			if errorDetails == nil {
				t.Fatal("GetErrorDetails() returned nil")
			}

			if !containsSubstring(errorDetails.Message, tt.wantMessage) {
				t.Errorf("Error details message = %q, want to contain %q", errorDetails.Message, tt.wantMessage)
			}
		})
	}
}

// TestExecutionMonitorLogViewer tests execution log viewer display
func TestExecutionMonitorLogViewer(t *testing.T) {
	tests := []struct {
		name           string
		setupExecution func() *execution.Execution
		wantLogEntries int
		wantEvents     []string
	}{
		{
			name: "display execution start log",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			wantLogEntries: 1,
			wantEvents:     []string{"started"},
		},
		{
			name: "display node execution logs",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add node executions
				nodeExec1 := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec1.Start()
				nodeExec1.Complete(map[string]interface{}{"result": "ok"})
				exec.AddNodeExecution(nodeExec1)

				nodeExec2 := execution.NewNodeExecution(exec.ID, "tool-2", "mcp_tool")
				nodeExec2.Start()
				exec.AddNodeExecution(nodeExec2)

				return exec
			},
			wantLogEntries: 4, // exec start + node1 start + node1 complete + node2 start
			wantEvents:     []string{"started", "tool-1", "tool-2", "completed"},
		},
		{
			name: "display error logs",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add failed node execution
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				nodeExec.Fail(&execution.NodeError{
					Type:    execution.ErrorTypeExecution,
					Message: "Tool failed",
				})
				exec.AddNodeExecution(nodeExec)

				return exec
			},
			wantLogEntries: 3, // exec start + node start + node fail
			wantEvents:     []string{"started", "tool-1", "failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Get log viewer
			logViewer := monitor.GetLogViewer()
			if logViewer == nil {
				t.Fatal("GetLogViewer() returned nil")
			}

			// Verify log entry count
			entries := logViewer.GetLogEntries()
			if len(entries) < tt.wantLogEntries {
				t.Errorf("GetLogEntries() count = %d, want at least %d", len(entries), tt.wantLogEntries)
			}

			// Verify expected events are in logs
			for _, wantEvent := range tt.wantEvents {
				if !screenContainsText(screen, wantEvent) {
					t.Errorf("Expected log event %q not found in screen buffer", wantEvent)
				}
			}

			// Verify timestamps are displayed
			if !screenContainsText(screen, ":") { // Time format contains colons
				t.Error("Expected timestamp format not found in logs")
			}
		})
	}
}

// TestExecutionMonitorPerformanceMetrics tests performance metrics display
func TestExecutionMonitorPerformanceMetrics(t *testing.T) {
	tests := []struct {
		name           string
		setupExecution func() *execution.Execution
		wantMetrics    map[string]bool // metric name -> should be displayed
	}{
		{
			name: "display basic metrics for running execution",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Simulate some execution time
				time.Sleep(10 * time.Millisecond)

				return exec
			},
			wantMetrics: map[string]bool{
				"Duration":       true,
				"Nodes Executed": true,
				"Status":         true,
			},
		},
		{
			name: "display completed execution metrics",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add node executions
				for _, nodeID := range []string{"tool-1", "tool-2"} {
					nodeExec := execution.NewNodeExecution(exec.ID, types.NodeID(nodeID), "mcp_tool")
					nodeExec.Start()
					time.Sleep(5 * time.Millisecond)
					nodeExec.Complete(nil)
					exec.AddNodeExecution(nodeExec)
				}

				exec.Complete(nil)
				return exec
			},
			wantMetrics: map[string]bool{
				"Duration":       true,
				"Nodes Executed": true,
				"Status":         true,
				"Total Time":     true,
			},
		},
		{
			name: "display failure metrics",
			setupExecution: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()

				// Add failed node
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				time.Sleep(5 * time.Millisecond)
				nodeExec.Fail(&execution.NodeError{
					Type:    execution.ErrorTypeExecution,
					Message: "Failed",
				})
				exec.AddNodeExecution(nodeExec)

				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "Execution failed",
					NodeID:  "tool-1",
				})

				return exec
			},
			wantMetrics: map[string]bool{
				"Duration":       true,
				"Nodes Executed": true,
				"Status":         true,
				"Failed Node":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.setupExecution()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Get metrics panel
			metricsPanel := monitor.GetMetricsPanel()
			if metricsPanel == nil {
				t.Fatal("GetMetricsPanel() returned nil")
			}

			// Verify expected metrics are displayed
			for metric, shouldDisplay := range tt.wantMetrics {
				found := screenContainsText(screen, metric)
				if found != shouldDisplay {
					if shouldDisplay {
						t.Errorf("Expected metric %q not found in screen buffer", metric)
					} else {
						t.Errorf("Unexpected metric %q found in screen buffer", metric)
					}
				}
			}

			// Verify metrics have values
			metrics := metricsPanel.GetMetrics()
			if len(metrics) == 0 {
				t.Error("GetMetrics() returned empty map")
			}
		})
	}
}

// TestExecutionMonitorKeyboardNavigation tests keyboard navigation in monitor view
func TestExecutionMonitorKeyboardNavigation(t *testing.T) {
	tests := []struct {
		name         string
		initialPanel string
		keys         []rune
		wantPanel    string
		wantAction   string
	}{
		{
			name:         "tab switches between panels",
			initialPanel: "workflow",
			keys:         []rune{'\t'},
			wantPanel:    "variables",
			wantAction:   "switch_panel",
		},
		{
			name:         "shift-tab switches backward",
			initialPanel: "variables",
			keys:         []rune{'\t'}, // Shift+Tab would be different key code
			wantPanel:    "logs",
			wantAction:   "switch_panel",
		},
		{
			name:         "j/k scrolls in log viewer",
			initialPanel: "logs",
			keys:         []rune{'j', 'j', 'k'},
			wantPanel:    "logs",
			wantAction:   "scroll",
		},
		{
			name:         "e expands variable details",
			initialPanel: "variables",
			keys:         []rune{'e'},
			wantPanel:    "variables",
			wantAction:   "expand",
		},
		{
			name:         "esc closes error detail",
			initialPanel: "error",
			keys:         []rune{27}, // ESC
			wantPanel:    "workflow",
			wantAction:   "close",
		},
		{
			name:         "question mark shows help",
			initialPanel: "workflow",
			keys:         []rune{'?'},
			wantPanel:    "help",
			wantAction:   "show_help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := createTestWorkflowForExecution()
			exec := createTestExecution(wf)
			exec.Start()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			// Set initial panel
			monitor.SetActivePanel(tt.initialPanel)

			// Simulate key presses
			for _, key := range tt.keys {
				err := monitor.HandleKey(key)
				if err != nil {
					t.Fatalf("HandleKey(%c) failed: %v", key, err)
				}
			}

			// Verify active panel changed correctly
			activePanel := monitor.GetActivePanel()
			if activePanel != tt.wantPanel {
				t.Errorf("After key sequence, active panel = %q, want %q", activePanel, tt.wantPanel)
			}

			// Verify last action
			lastAction := monitor.GetLastAction()
			if lastAction != tt.wantAction {
				t.Errorf("After key sequence, last action = %q, want %q", lastAction, tt.wantAction)
			}
		})
	}
}

// TestExecutionMonitorRefreshOnEvents tests view refresh on execution events
func TestExecutionMonitorRefreshOnEvents(t *testing.T) {
	tests := []struct {
		name         string
		initialState func() *execution.Execution
		event        func(*execution.Execution)
		wantRefresh  bool
		wantUpdated  []string // list of components that should be updated
	}{
		{
			name: "refresh on execution start",
			initialState: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				return createTestExecution(wf)
			},
			event: func(exec *execution.Execution) {
				exec.Start()
			},
			wantRefresh: true,
			wantUpdated: []string{"status", "metrics"},
		},
		{
			name: "refresh on node execution start",
			initialState: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			event: func(exec *execution.Execution) {
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				exec.AddNodeExecution(nodeExec)
			},
			wantRefresh: true,
			wantUpdated: []string{"workflow", "logs", "metrics"},
		},
		{
			name: "refresh on node execution complete",
			initialState: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				nodeExec := execution.NewNodeExecution(exec.ID, "tool-1", "mcp_tool")
				nodeExec.Start()
				exec.AddNodeExecution(nodeExec)
				return exec
			},
			event: func(exec *execution.Execution) {
				exec.NodeExecutions[0].Complete(map[string]interface{}{"result": "ok"})
			},
			wantRefresh: true,
			wantUpdated: []string{"workflow", "logs", "metrics"},
		},
		{
			name: "refresh on variable change",
			initialState: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			event: func(exec *execution.Execution) {
				exec.Context.SetVariable("new_var", "new_value")
			},
			wantRefresh: true,
			wantUpdated: []string{"variables"},
		},
		{
			name: "refresh on execution failure",
			initialState: func() *execution.Execution {
				wf := createTestWorkflowForExecution()
				exec := createTestExecution(wf)
				exec.Start()
				return exec
			},
			event: func(exec *execution.Execution) {
				exec.Fail(&execution.ExecutionError{
					Type:    execution.ErrorTypeExecution,
					Message: "Test failure",
				})
			},
			wantRefresh: true,
			wantUpdated: []string{"status", "error", "logs", "metrics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := tt.initialState()
			wf := createTestWorkflowForExecution()

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			// Render initial state
			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Initial Render() failed: %v", err)
			}

			// Apply event
			tt.event(exec)

			// Notify monitor of event (trigger refresh)
			refreshed := monitor.OnExecutionEvent(exec)

			// Verify refresh occurred
			if refreshed != tt.wantRefresh {
				t.Errorf("OnExecutionEvent() refreshed = %v, want %v", refreshed, tt.wantRefresh)
			}

			// Re-render after event
			_, err = monitor.Render()
			if err != nil {
				t.Fatalf("Post-event Render() failed: %v", err)
			}

			// Verify expected components were updated
			for _, component := range tt.wantUpdated {
				wasUpdated := monitor.WasComponentUpdated(component)
				if !wasUpdated {
					t.Errorf("Component %q was not updated after event", component)
				}
			}
		})
	}
}

// TestExecutionMonitorAutoScroll tests auto-scrolling behavior in log viewer
func TestExecutionMonitorAutoScroll(t *testing.T) {
	tests := []struct {
		name                 string
		logEntryCount        int
		viewportHeight       int
		autoScrollEnabled    bool
		wantScrolledToBottom bool
	}{
		{
			name:                 "auto-scroll enabled shows latest logs",
			logEntryCount:        100,
			viewportHeight:       10,
			autoScrollEnabled:    true,
			wantScrolledToBottom: true,
		},
		{
			name:                 "auto-scroll disabled maintains position",
			logEntryCount:        100,
			viewportHeight:       10,
			autoScrollEnabled:    false,
			wantScrolledToBottom: false,
		},
		{
			name:                 "few logs no scroll needed",
			logEntryCount:        5,
			viewportHeight:       10,
			autoScrollEnabled:    true,
			wantScrolledToBottom: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := createTestWorkflowForExecution()
			exec := createTestExecution(wf)
			exec.Start()

			// Add many node executions to generate logs
			for i := 0; i < tt.logEntryCount; i++ {
				nodeExec := execution.NewNodeExecution(exec.ID, types.NodeID("node-"+string(rune(i))), "mcp_tool")
				nodeExec.Start()
				nodeExec.Complete(nil)
				exec.AddNodeExecution(nodeExec)
			}

			screen := goterm.NewScreen(120, 40)
			monitor := tui.NewExecutionMonitor(exec, wf, screen)

			// Configure auto-scroll
			logViewer := monitor.GetLogViewer()
			logViewer.SetAutoScroll(tt.autoScrollEnabled)

			// Render
			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Check scroll position
			isAtBottom := logViewer.IsScrolledToBottom()
			if isAtBottom != tt.wantScrolledToBottom {
				t.Errorf("IsScrolledToBottom() = %v, want %v", isAtBottom, tt.wantScrolledToBottom)
			}
		})
	}
}

// TestExecutionMonitorRenderPerformance tests rendering performance targets
func TestExecutionMonitorRenderPerformance(t *testing.T) {
	t.Run("render_within_16ms_target", func(t *testing.T) {
		wf := createTestWorkflowForExecution()
		exec := createTestExecution(wf)
		exec.Start()

		// Add significant execution history
		for i := 0; i < 50; i++ {
			nodeExec := execution.NewNodeExecution(exec.ID, types.NodeID("node-"+string(rune(i))), "mcp_tool")
			nodeExec.Start()
			nodeExec.Complete(map[string]interface{}{"result": i})
			exec.AddNodeExecution(nodeExec)

			exec.Context.SetVariable("var_"+string(rune(i)), i)
		}

		screen := goterm.NewScreen(120, 40)
		monitor := tui.NewExecutionMonitor(exec, wf, screen)

		// Measure render time over multiple iterations
		iterations := 100
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := monitor.Render()
			if err != nil {
				t.Fatalf("Render() iteration %d failed: %v", i, err)
			}
		}
		duration := time.Since(start)

		avgRenderTime := duration / time.Duration(iterations)

		// Performance target: < 16ms per frame (60 FPS)
		targetFrameTime := 16 * time.Millisecond
		if avgRenderTime > targetFrameTime {
			t.Errorf("Average render time %v exceeds target %v", avgRenderTime, targetFrameTime)
		}
	})
}

// Helper function to compare values (basic implementation)
func valuesEqual(a, b interface{}) bool {
	// This is a simplified comparison - would need more sophisticated logic for deep equality
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
