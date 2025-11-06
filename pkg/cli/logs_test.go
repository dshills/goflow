package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	pkgexec "github.com/dshills/goflow/pkg/execution"
)

func TestParseEventTypeFilter(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		expected int // expected number of event types
	}{
		{
			name:     "empty filter",
			filter:   "",
			expected: 0,
		},
		{
			name:     "error shorthand",
			filter:   "error",
			expected: 3, // error, node_failed, execution_failed
		},
		{
			name:     "info shorthand",
			filter:   "info",
			expected: 4, // execution_started, execution_completed, node_started, node_completed
		},
		{
			name:     "warning shorthand",
			filter:   "warning",
			expected: 2, // node_retried, node_skipped
		},
		{
			name:     "multiple filters",
			filter:   "error,warning",
			expected: 5, // 3 + 2
		},
		{
			name:     "exact event type",
			filter:   "node_started",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEventTypeFilter(tt.filter)
			if len(result) != tt.expected {
				t.Errorf("parseEventTypeFilter(%q) returned %d event types, expected %d",
					tt.filter, len(result), tt.expected)
			}
		})
	}
}

func TestGetEventIcon(t *testing.T) {
	tests := []struct {
		eventType pkgexec.AuditEventType
		expected  string
	}{
		{pkgexec.AuditEventExecutionStarted, "▶"},
		{pkgexec.AuditEventExecutionCompleted, "✓"},
		{pkgexec.AuditEventExecutionFailed, "✗"},
		{pkgexec.AuditEventNodeStarted, "▶"},
		{pkgexec.AuditEventNodeCompleted, "✓"},
		{pkgexec.AuditEventNodeFailed, "✗"},
		{pkgexec.AuditEventNodeRetried, "↻"},
		{pkgexec.AuditEventVariableSet, "≔"},
		{pkgexec.AuditEventError, "⚠"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			result := getEventIcon(tt.eventType)
			if result != tt.expected {
				t.Errorf("getEventIcon(%s) = %s, expected %s",
					tt.eventType, result, tt.expected)
			}
		})
	}
}

func TestGetEventColor(t *testing.T) {
	tests := []struct {
		name      string
		eventType pkgexec.AuditEventType
		noColor   bool
		expected  string
	}{
		{
			name:      "success with color",
			eventType: pkgexec.AuditEventNodeCompleted,
			noColor:   false,
			expected:  colorGreen,
		},
		{
			name:      "error with color",
			eventType: pkgexec.AuditEventNodeFailed,
			noColor:   false,
			expected:  colorRed,
		},
		{
			name:      "info with color",
			eventType: pkgexec.AuditEventNodeStarted,
			noColor:   false,
			expected:  colorBlue,
		},
		{
			name:      "no color mode",
			eventType: pkgexec.AuditEventNodeCompleted,
			noColor:   true,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEventColor(tt.eventType, tt.noColor)
			if result != tt.expected {
				t.Errorf("getEventColor(%s, %t) = %q, expected %q",
					tt.eventType, tt.noColor, result, tt.expected)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   execution.Status
		noColor  bool
		contains string
	}{
		{
			name:     "completed with color",
			status:   execution.StatusCompleted,
			noColor:  false,
			contains: "completed",
		},
		{
			name:     "failed with color",
			status:   execution.StatusFailed,
			noColor:  false,
			contains: "failed",
		},
		{
			name:     "no color mode",
			status:   execution.StatusCompleted,
			noColor:  true,
			contains: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status, tt.noColor)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatStatus(%s, %t) = %q, should contain %q",
					tt.status, tt.noColor, result, tt.contains)
			}

			// Verify no color codes when noColor is true
			if tt.noColor && strings.Contains(result, "\033[") {
				t.Errorf("formatStatus(%s, true) should not contain color codes, got %q",
					tt.status, result)
			}
		})
	}
}

func TestDisplayEvent(t *testing.T) {
	startTime := time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC)
	eventTime := startTime.Add(2 * time.Second)
	duration := 1500 * time.Millisecond

	tests := []struct {
		name     string
		event    pkgexec.AuditEvent
		noColor  bool
		contains []string
	}{
		{
			name: "execution started event",
			event: pkgexec.AuditEvent{
				Timestamp: eventTime,
				Type:      pkgexec.AuditEventExecutionStarted,
				Message:   "Execution started",
			},
			noColor:  true,
			contains: []string{"12:00:02.000", "Execution started", "▶"},
		},
		{
			name: "node completed with duration",
			event: pkgexec.AuditEvent{
				Timestamp: eventTime,
				Type:      pkgexec.AuditEventNodeCompleted,
				NodeID:    "process-data",
				NodeType:  "mcp_tool",
				Message:   "Node 'process-data' completed",
				Duration:  &duration,
			},
			noColor:  true,
			contains: []string{"12:00:02.000", "process-data", "mcp_tool", "1.500s", "✓"},
		},
		{
			name: "node failed with error",
			event: pkgexec.AuditEvent{
				Timestamp: eventTime,
				Type:      pkgexec.AuditEventNodeFailed,
				NodeID:    "validate",
				NodeType:  "transform",
				Message:   "Node 'validate' failed",
				Details: map[string]interface{}{
					"error_message": "Invalid JSON",
				},
			},
			noColor:  true,
			contains: []string{"12:00:02.000", "validate", "transform", "Invalid JSON", "✗"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			displayEvent(&buf, tt.event, startTime, tt.noColor)

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("displayEvent output should contain %q, got:\n%s",
						expected, output)
				}
			}

			// Verify no color codes when noColor is true
			if tt.noColor && strings.Contains(output, "\033[") {
				t.Errorf("displayEvent with noColor=true should not contain color codes, got:\n%s",
					output)
			}
		})
	}
}

func TestLogsCommandIntegration(t *testing.T) {
	// Test that the command is properly configured
	cmd := NewLogsCommand()

	if cmd.Use != "logs <execution-id>" {
		t.Errorf("command Use = %q, expected 'logs <execution-id>'", cmd.Use)
	}

	// Check flags are registered
	flags := []string{"follow", "type", "tail", "no-color", "show-variables"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag --%s is not registered", flagName)
		}
	}

	// Check that it requires exactly one argument
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("command should require at least one argument")
	}

	if err := cmd.Args(cmd, []string{"exec-123"}); err != nil {
		t.Errorf("command should accept one argument, got error: %v", err)
	}

	if err := cmd.Args(cmd, []string{"exec-123", "extra"}); err == nil {
		t.Error("command should not accept more than one argument")
	}
}

func TestDisplayHistoricalLogs(t *testing.T) {
	// Create a simple audit trail
	trail := &pkgexec.AuditTrail{
		ExecutionID:     types.ExecutionID("exec-test-123"),
		WorkflowID:      types.WorkflowID("test-workflow"),
		WorkflowVersion: "1.0.0",
		Status:          execution.StatusCompleted,
		StartedAt:       time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC),
		CompletedAt:     time.Date(2025, 11, 5, 12, 0, 5, 0, time.UTC),
		Duration:        5 * time.Second,
		Events: []pkgexec.AuditEvent{
			{
				Timestamp: time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC),
				Type:      pkgexec.AuditEventExecutionStarted,
				Message:   "Execution started",
			},
			{
				Timestamp: time.Date(2025, 11, 5, 12, 0, 5, 0, time.UTC),
				Type:      pkgexec.AuditEventExecutionCompleted,
				Message:   "Execution completed",
			},
		},
		NodeCount:  2,
		ErrorCount: 0,
	}

	var buf bytes.Buffer
	cmd := NewLogsCommand()
	cmd.SetOut(&buf)

	err := displayHistoricalLogs(cmd, trail, true)
	if err != nil {
		t.Fatalf("displayHistoricalLogs failed: %v", err)
	}

	output := buf.String()

	// Verify key components are present
	expectedStrings := []string{
		"exec-test-123",
		"test-workflow",
		"1.0.0",
		"completed",
		"Execution started",
		"Execution completed",
		"Completed in 5s",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output should contain %q, got:\n%s", expected, output)
		}
	}
}
