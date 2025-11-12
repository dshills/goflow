package execution

import (
	"testing"

	"github.com/dshills/goflow/pkg/domain/types"
)

// TestExecutionError_Error_NilReceiver tests that calling Error() on a nil
// *ExecutionError doesn't panic. This is a regression test for Issue #90.
//
// FR-021: Check for nil before dereferencing pointers
// SC-008: Zero runtime panics from nil dereferences
func TestExecutionError_Error_NilReceiver(t *testing.T) {
	tests := []struct {
		name string
		err  *ExecutionError
		want string
	}{
		{
			name: "nil receiver should not panic",
			err:  nil,
			want: "<nil>",
		},
		{
			name: "valid error with NodeID",
			err: &ExecutionError{
				Type:    ErrorTypeValidation,
				Message: "test error",
				NodeID:  types.NodeID("node-1"),
			},
			want: "[validation] node node-1: test error",
		},
		{
			name: "valid error without NodeID",
			err: &ExecutionError{
				Type:    ErrorTypeConnection,
				Message: "connection failed",
			},
			want: "[connection] connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic even with nil receiver
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Error() panicked with nil receiver: %v", r)
				}
			}()

			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNodeError_Error_NilReceiver tests that calling Error() on a nil
// *NodeError doesn't panic.
//
// FR-021: Check for nil before dereferencing pointers
// SC-008: Zero runtime panics from nil dereferences
func TestNodeError_Error_NilReceiver(t *testing.T) {
	tests := []struct {
		name string
		err  *NodeError
		want string
	}{
		{
			name: "nil receiver should not panic",
			err:  nil,
			want: "<nil>",
		},
		{
			name: "valid error",
			err: &NodeError{
				Type:    ErrorTypeExecution,
				Message: "execution failed",
			},
			want: "[execution] execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Error() panicked with nil receiver: %v", r)
				}
			}()

			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExecutionError_NilFields tests that ExecutionError handles nil/empty fields gracefully.
//
// FR-021: Check for nil before dereferencing pointers
func TestExecutionError_NilFields(t *testing.T) {
	tests := []struct {
		name string
		err  *ExecutionError
	}{
		{
			name: "nil Context map",
			err: &ExecutionError{
				Type:    ErrorTypeData,
				Message: "data error",
				Context: nil, // Should not cause issues
			},
		},
		{
			name: "empty NodeID",
			err: &ExecutionError{
				Type:    ErrorTypeTimeout,
				Message: "timeout",
				NodeID:  "", // Should use alternate format
			},
		},
		{
			name: "all fields zero value",
			err:  &ExecutionError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Error handling panicked: %v", r)
				}
			}()

			// Should not panic
			_ = tt.err.Error()
			_ = tt.err.Type
			_ = tt.err.Message
			_ = tt.err.NodeID
			_ = tt.err.Context
		})
	}
}
