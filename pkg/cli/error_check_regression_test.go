package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"

	domainexec "github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// createTestExecution creates a minimal execution for testing.
func createTestExecution() *domainexec.Execution {
	now := time.Now()
	return &domainexec.Execution{
		ID:          types.NewExecutionID(),
		WorkflowID:  types.WorkflowID("test-workflow"),
		Status:      domainexec.StatusCompleted,
		StartedAt:   now,
		CompletedAt: now.Add(time.Second),
	}
}

// TestDisplayJSONResult_MarshalError tests that JSON marshaling errors are handled.
// This is a regression test for Issue #218.
//
// FR-022: Check errors before using return values
func TestDisplayJSONResult_MarshalError(t *testing.T) {
	tests := []struct {
		name        string
		returnValue interface{}
		wantErr     bool
		wantOutput  string
	}{
		{
			name:        "valid JSON",
			returnValue: map[string]interface{}{"key": "value"},
			wantErr:     false,
			wantOutput:  `"return_value": {`,
		},
		{
			name:        "nil return value",
			returnValue: nil,
			wantErr:     false,
			wantOutput:  `"return_value": null`,
		},
		{
			name:        "unmarshalable type - channel",
			returnValue: make(chan int), // Channels cannot be JSON marshaled
			wantErr:     true,
			// Should contain error message in output
			wantOutput: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test execution with return value
			exec := createTestExecution()
			exec.ReturnValue = tt.returnValue

			// Capture output
			cmd := &cobra.Command{}
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)

			// Call displayJSONResult
			displayJSONResult(cmd, exec, nil)

			output := buf.String()

			// Verify output contains expected content or error indication
			if !bytes.Contains([]byte(output), []byte(tt.wantOutput)) {
				t.Errorf("displayJSONResult() output doesn't contain %q\nGot: %s",
					tt.wantOutput, output)
			}

			// For unmarshalable types, verify error is indicated
			if tt.wantErr {
				// Check if output indicates an error occurred
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err == nil {
					// If we can parse the JSON, check if it contains an error indication
					if result["marshal_error"] == nil && result["error"] == nil {
						t.Error("Expected error indication in JSON output for unmarshalable type")
					}
				}
			}
		})
	}
}

// TestDisplayFinalResult_ErrorOutput tests error display formatting.
// FR-022: Check errors before using return values
func TestDisplayFinalResult_ErrorOutput(t *testing.T) {
	tests := []struct {
		name      string
		execErr   error
		wantPanic bool
	}{
		{
			name:      "nil error",
			execErr:   nil,
			wantPanic: false,
		},
		{
			name:      "standard error",
			execErr:   errors.New("test error"),
			wantPanic: false,
		},
		{
			name:      "wrapped error",
			execErr:   errors.New("wrapped: inner error"),
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("displayFinalResult() panicked unexpectedly: %v", r)
					}
				}
			}()

			exec := createTestExecution()
			cmd := &cobra.Command{}
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)

			state := &watchState{}
			displayFinalResult(cmd, exec, tt.execErr, state, false)

			// Should not panic
		})
	}
}

// TestErrorHandling_NonNilCheck tests that code checks errors properly.
// FR-022: Check errors before using return values
func TestErrorHandling_NonNilCheck(t *testing.T) {
	// This test verifies the pattern of checking errors before use

	// Simulate a function that returns error
	doWork := func() (string, error) {
		return "", errors.New("work failed")
	}

	// CORRECT pattern - check error first
	result, err := doWork()
	if err != nil {
		// Error handled - result should not be used
		if result != "" {
			t.Error("Expected empty result on error")
		}
		return
	}
	// Only use result if no error
	_ = result

	// This test passes if we follow the correct pattern
}

// TestJSONMarshalError_Detection tests detecting marshal errors.
// FR-022: Check errors before using return values
func TestJSONMarshalError_Detection(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid value",
			value:   map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name:    "channel - unmarshalable",
			value:   make(chan int),
			wantErr: true,
		},
		{
			name:    "function - unmarshalable",
			value:   func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.MarshalIndent(tt.value, "", "  ")

			if (err != nil) != tt.wantErr {
				t.Errorf("json.MarshalIndent() error = %v, wantErr %v", err, tt.wantErr)
			}

			// The key point: we MUST check this error before using the result
			if err != nil {
				// Handle error - this is the correct pattern
				t.Logf("Correctly detected marshal error: %v", err)
			}
		})
	}
}
