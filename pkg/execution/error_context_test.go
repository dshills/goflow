package execution_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/execution"
)

// TestNewOperationalError tests basic OperationalError creation.
func TestNewOperationalError(t *testing.T) {
	baseErr := errors.New("connection failed")

	tests := []struct {
		name        string
		operation   string
		workflowID  string
		nodeID      string
		cause       error
		wantNil     bool
		wantMessage string
	}{
		{
			name:        "valid error creation",
			operation:   "executing node",
			workflowID:  "wf-123",
			nodeID:      "node-456",
			cause:       baseErr,
			wantNil:     false,
			wantMessage: "connection failed",
		},
		{
			name:       "nil cause returns nil",
			operation:  "executing node",
			workflowID: "wf-123",
			nodeID:     "node-456",
			cause:      nil,
			wantNil:    true,
		},
		{
			name:        "empty operation is allowed",
			operation:   "",
			workflowID:  "wf-123",
			nodeID:      "node-456",
			cause:       baseErr,
			wantNil:     false,
			wantMessage: "connection failed",
		},
		{
			name:        "empty workflow ID",
			operation:   "executing node",
			workflowID:  "",
			nodeID:      "node-456",
			cause:       baseErr,
			wantNil:     false,
			wantMessage: "connection failed",
		},
		{
			name:        "empty node ID",
			operation:   "executing node",
			workflowID:  "wf-123",
			nodeID:      "",
			cause:       baseErr,
			wantNil:     false,
			wantMessage: "connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opErr := execution.NewOperationalError(tt.operation, tt.workflowID, tt.nodeID, tt.cause)

			if tt.wantNil {
				if opErr != nil {
					t.Errorf("NewOperationalError() = %v, want nil", opErr)
				}
				return
			}

			if opErr == nil {
				t.Fatal("NewOperationalError() returned nil, want non-nil")
			}

			// Verify error message contains cause
			if !strings.Contains(opErr.Error(), tt.wantMessage) {
				t.Errorf("Error() = %q, want to contain %q", opErr.Error(), tt.wantMessage)
			}

			// Verify timestamp is recent (within last second)
			if time.Since(opErr.Timestamp) > time.Second {
				t.Errorf("Timestamp is too old: %v", opErr.Timestamp)
			}
		})
	}
}

// TestNewOperationalErrorWithAttrs tests OperationalError creation with attributes.
func TestNewOperationalErrorWithAttrs(t *testing.T) {
	baseErr := errors.New("validation failed")

	tests := []struct {
		name       string
		operation  string
		workflowID string
		nodeID     string
		cause      error
		attrs      map[string]interface{}
		wantNil    bool
	}{
		{
			name:       "with attributes",
			operation:  "validating input",
			workflowID: "wf-123",
			nodeID:     "node-456",
			cause:      baseErr,
			attrs: map[string]interface{}{
				"inputSize": 1024,
				"schema":    "v2",
			},
			wantNil: false,
		},
		{
			name:       "with nil attributes",
			operation:  "validating input",
			workflowID: "wf-123",
			nodeID:     "node-456",
			cause:      baseErr,
			attrs:      nil,
			wantNil:    false,
		},
		{
			name:       "with empty attributes",
			operation:  "validating input",
			workflowID: "wf-123",
			nodeID:     "node-456",
			cause:      baseErr,
			attrs:      map[string]interface{}{},
			wantNil:    false,
		},
		{
			name:       "nil cause returns nil",
			operation:  "validating input",
			workflowID: "wf-123",
			nodeID:     "node-456",
			cause:      nil,
			attrs:      map[string]interface{}{"key": "value"},
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opErr := execution.NewOperationalErrorWithAttrs(tt.operation, tt.workflowID, tt.nodeID, tt.cause, tt.attrs)

			if tt.wantNil {
				if opErr != nil {
					t.Errorf("NewOperationalErrorWithAttrs() = %v, want nil", opErr)
				}
				return
			}

			if opErr == nil {
				t.Fatal("NewOperationalErrorWithAttrs() returned nil, want non-nil")
			}

			// Verify attributes are preserved
			if tt.attrs != nil && len(tt.attrs) > 0 {
				if opErr.Attributes == nil {
					t.Error("Attributes is nil, want non-nil")
				} else if len(opErr.Attributes) != len(tt.attrs) {
					t.Errorf("len(Attributes) = %d, want %d", len(opErr.Attributes), len(tt.attrs))
				}
			}
		})
	}
}

// TestOperationalError_Error tests the Error() method formatting.
func TestOperationalError_Error(t *testing.T) {
	baseErr := errors.New("network timeout")
	timestamp := time.Date(2025, 11, 12, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		opErr        *execution.OperationalError
		wantContains []string
	}{
		{
			name: "full error message",
			opErr: &execution.OperationalError{
				Operation:  "executing MCP tool",
				WorkflowID: "wf-123",
				NodeID:     "node-456",
				Timestamp:  timestamp,
				Cause:      baseErr,
			},
			wantContains: []string{
				"executing MCP tool",
				"workflow=wf-123",
				"node=node-456",
				"network timeout",
				"2025-11-12",
			},
		},
		{
			name: "empty node ID",
			opErr: &execution.OperationalError{
				Operation:  "connecting to server",
				WorkflowID: "wf-123",
				NodeID:     "",
				Timestamp:  timestamp,
				Cause:      baseErr,
			},
			wantContains: []string{
				"connecting to server",
				"workflow=wf-123",
				"network timeout",
			},
		},
		{
			name: "with attributes",
			opErr: &execution.OperationalError{
				Operation:  "calling tool",
				WorkflowID: "wf-123",
				NodeID:     "node-456",
				Timestamp:  timestamp,
				Attributes: map[string]interface{}{
					"serverID": "test-server",
					"toolName": "read_file",
				},
				Cause: baseErr,
			},
			wantContains: []string{
				"calling tool",
				"workflow=wf-123",
				"node=node-456",
				"network timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.opErr.Error()

			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("Error() = %q, want to contain %q", msg, want)
				}
			}
		})
	}
}

// TestOperationalError_Unwrap tests error unwrapping for errors.Is/As support.
func TestOperationalError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	opErr := execution.NewOperationalError("operation", "wf-123", "node-456", baseErr)

	if opErr == nil {
		t.Fatal("NewOperationalError() returned nil")
	}

	unwrapped := opErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}

	// Test errors.Is
	if !errors.Is(opErr, baseErr) {
		t.Error("errors.Is(opErr, baseErr) = false, want true")
	}

	// Test errors.As
	var target *execution.OperationalError
	if !errors.As(opErr, &target) {
		t.Error("errors.As(opErr, &target) = false, want true")
	}
}

// TestOperationalError_WithNilCause tests handling of nil cause errors.
func TestOperationalError_WithNilCause(t *testing.T) {
	opErr := execution.NewOperationalError("operation", "wf-123", "node-456", nil)

	if opErr != nil {
		t.Errorf("NewOperationalError(nil cause) = %v, want nil", opErr)
	}

	opErrWithAttrs := execution.NewOperationalErrorWithAttrs("operation", "wf-123", "node-456", nil, map[string]interface{}{"key": "value"})

	if opErrWithAttrs != nil {
		t.Errorf("NewOperationalErrorWithAttrs(nil cause) = %v, want nil", opErrWithAttrs)
	}
}

// TestOperationalError_ChainedWrapping tests multiple levels of error wrapping.
func TestOperationalError_ChainedWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	ctx1 := execution.NewOperationalError("operation1", "wf1", "node1", baseErr)
	ctx2 := execution.NewOperationalError("operation2", "wf1", "node2", ctx1)
	ctx3 := execution.NewOperationalError("operation3", "wf1", "node3", ctx2)

	if ctx1 == nil || ctx2 == nil || ctx3 == nil {
		t.Fatal("Error chain creation failed")
	}

	// Test unwrapping chain with errors.Is
	if !errors.Is(ctx3, baseErr) {
		t.Error("errors.Is(ctx3, baseErr) = false, want true")
	}
	if !errors.Is(ctx3, ctx2) {
		t.Error("errors.Is(ctx3, ctx2) = false, want true")
	}
	if !errors.Is(ctx3, ctx1) {
		t.Error("errors.Is(ctx3, ctx1) = false, want true")
	}

	// Test error message contains all operations
	msg := ctx3.Error()
	if !strings.Contains(msg, "operation3") {
		t.Errorf("Error() = %q, want to contain 'operation3'", msg)
	}

	// Test unwrapping manually
	unwrapped1 := ctx3.Unwrap()
	if unwrapped1 != ctx2 {
		t.Errorf("ctx3.Unwrap() = %v, want ctx2", unwrapped1)
	}

	unwrapped2 := ctx2.Unwrap()
	if unwrapped2 != ctx1 {
		t.Errorf("ctx2.Unwrap() = %v, want ctx1", unwrapped2)
	}

	unwrapped3 := ctx1.Unwrap()
	if unwrapped3 != baseErr {
		t.Errorf("ctx1.Unwrap() = %v, want baseErr", unwrapped3)
	}
}

// TestOperationalError_Attributes tests attribute handling.
func TestOperationalError_Attributes(t *testing.T) {
	baseErr := errors.New("tool execution failed")
	attrs := map[string]interface{}{
		"nodeType":  "mcp_tool",
		"serverID":  "test-server",
		"toolName":  "read_file",
		"inputSize": 1024,
	}

	opErr := execution.NewOperationalErrorWithAttrs("executing tool", "wf-123", "node-456", baseErr, attrs)

	if opErr == nil {
		t.Fatal("NewOperationalErrorWithAttrs() returned nil")
	}

	// Verify attributes are preserved
	if opErr.Attributes == nil {
		t.Fatal("Attributes is nil")
	}

	if len(opErr.Attributes) != len(attrs) {
		t.Errorf("len(Attributes) = %d, want %d", len(opErr.Attributes), len(attrs))
	}

	for key, expectedValue := range attrs {
		actualValue, ok := opErr.Attributes[key]
		if !ok {
			t.Errorf("Attributes missing key %q", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Attributes[%q] = %v, want %v", key, actualValue, expectedValue)
		}
	}

	// Verify attributes don't affect error unwrapping
	if !errors.Is(opErr, baseErr) {
		t.Error("errors.Is(opErr, baseErr) = false, want true")
	}
}

// TestOperationalError_EmptyFields tests behavior with empty/missing fields.
func TestOperationalError_EmptyFields(t *testing.T) {
	baseErr := errors.New("test error")

	tests := []struct {
		name       string
		operation  string
		workflowID string
		nodeID     string
	}{
		{"all empty", "", "", ""},
		{"only operation", "op", "", ""},
		{"only workflow", "", "wf", ""},
		{"only node", "", "", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opErr := execution.NewOperationalError(tt.operation, tt.workflowID, tt.nodeID, baseErr)

			if opErr == nil {
				t.Fatal("NewOperationalError() returned nil")
			}

			// Should still produce a valid error message
			msg := opErr.Error()
			if msg == "" {
				t.Error("Error() returned empty string")
			}

			// Should still contain cause message
			if !strings.Contains(msg, "test error") {
				t.Errorf("Error() = %q, want to contain 'test error'", msg)
			}
		})
	}
}
