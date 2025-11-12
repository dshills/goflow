package tui

import (
	"testing"
)

// TestValidationPanel_ShowHide tests panel visibility
func TestValidationPanel_ShowHide(t *testing.T) {
	status := NewValidationStatus()
	panel := NewValidationPanel(status)

	if panel.IsVisible() {
		t.Error("Panel should be hidden by default")
	}

	panel.Show()
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()
	if panel.IsVisible() {
		t.Error("Panel should be hidden after Hide()")
	}
}

// TestValidationPanel_Navigation tests error/warning navigation
func TestValidationPanel_Navigation(t *testing.T) {
	status := NewValidationStatus()
	status.AddError("node1", "error1", "Error 1")
	status.AddError("node2", "error2", "Error 2")
	status.AddWarning("node3", "Warning 1")
	status.SetValidated()

	panel := NewValidationPanel(status)

	// Initial selection should be 0
	if panel.selectedIndex != 0 {
		t.Errorf("Initial selection should be 0, got %d", panel.selectedIndex)
	}

	// Test Next()
	panel.Next()
	if panel.selectedIndex != 1 {
		t.Errorf("After Next(), selection should be 1, got %d", panel.selectedIndex)
	}

	panel.Next()
	if panel.selectedIndex != 2 {
		t.Errorf("After Next(), selection should be 2, got %d", panel.selectedIndex)
	}

	// Should wrap around to 0
	panel.Next()
	if panel.selectedIndex != 0 {
		t.Errorf("After Next() at end, selection should wrap to 0, got %d", panel.selectedIndex)
	}

	// Test Previous()
	panel.Previous()
	if panel.selectedIndex != 2 {
		t.Errorf("After Previous() at start, selection should wrap to 2, got %d", panel.selectedIndex)
	}

	panel.Previous()
	if panel.selectedIndex != 1 {
		t.Errorf("After Previous(), selection should be 1, got %d", panel.selectedIndex)
	}

	panel.Previous()
	if panel.selectedIndex != 0 {
		t.Errorf("After Previous(), selection should be 0, got %d", panel.selectedIndex)
	}
}

// TestValidationPanel_GetSelectedNodeID tests node ID retrieval
func TestValidationPanel_GetSelectedNodeID(t *testing.T) {
	tests := []struct {
		name           string
		errors         []ValidationError
		warnings       []ValidationWarning
		selectedIndex  int
		expectedNodeID string
	}{
		{
			name: "select first error",
			errors: []ValidationError{
				{NodeID: "node1", ErrorType: "error1", Message: "Error 1"},
				{NodeID: "node2", ErrorType: "error2", Message: "Error 2"},
			},
			warnings:       []ValidationWarning{},
			selectedIndex:  0,
			expectedNodeID: "node1",
		},
		{
			name: "select second error",
			errors: []ValidationError{
				{NodeID: "node1", ErrorType: "error1", Message: "Error 1"},
				{NodeID: "node2", ErrorType: "error2", Message: "Error 2"},
			},
			warnings:       []ValidationWarning{},
			selectedIndex:  1,
			expectedNodeID: "node2",
		},
		{
			name: "select first warning",
			errors: []ValidationError{
				{NodeID: "node1", ErrorType: "error1", Message: "Error 1"},
			},
			warnings: []ValidationWarning{
				{NodeID: "node2", Message: "Warning 1"},
			},
			selectedIndex:  1, // After error at index 0
			expectedNodeID: "node2",
		},
		{
			name: "select global error",
			errors: []ValidationError{
				{NodeID: "", ErrorType: "global_error", Message: "Global error"},
			},
			warnings:       []ValidationWarning{},
			selectedIndex:  0,
			expectedNodeID: "", // Global errors have no node ID
		},
		{
			name:           "empty status",
			errors:         []ValidationError{},
			warnings:       []ValidationWarning{},
			selectedIndex:  0,
			expectedNodeID: "",
		},
		{
			name: "out of bounds selection",
			errors: []ValidationError{
				{NodeID: "node1", ErrorType: "error1", Message: "Error 1"},
			},
			warnings:       []ValidationWarning{},
			selectedIndex:  99,
			expectedNodeID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewValidationStatus()
			for _, err := range tt.errors {
				status.AddError(err.NodeID, err.ErrorType, err.Message)
			}
			for _, warn := range tt.warnings {
				status.AddWarning(warn.NodeID, warn.Message)
			}
			status.SetValidated()

			panel := NewValidationPanel(status)
			panel.selectedIndex = tt.selectedIndex

			nodeID := panel.GetSelectedNodeID()
			if nodeID != tt.expectedNodeID {
				t.Errorf("Expected node ID %q, got %q", tt.expectedNodeID, nodeID)
			}
		})
	}
}

// TestValidationPanel_UpdateStatus tests status updates
func TestValidationPanel_UpdateStatus(t *testing.T) {
	// Create initial status with errors
	status1 := NewValidationStatus()
	status1.AddError("node1", "error1", "Error 1")
	status1.AddError("node2", "error2", "Error 2")
	status1.AddError("node3", "error3", "Error 3")
	status1.SetValidated()

	panel := NewValidationPanel(status1)
	panel.selectedIndex = 2 // Select third error

	// Update with new status that has fewer errors
	status2 := NewValidationStatus()
	status2.AddError("node1", "error1", "Error 1")
	status2.SetValidated()

	panel.UpdateStatus(status2)

	// Selection should be reset because index 2 is out of bounds
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selection to be reset to 0, got %d", panel.selectedIndex)
	}

	// Verify new status is set
	if panel.GetStatus() != status2 {
		t.Error("Expected new status to be set")
	}
}

// TestValidationPanel_EmptyStatus tests panel behavior with empty status
func TestValidationPanel_EmptyStatus(t *testing.T) {
	status := NewValidationStatus()
	panel := NewValidationPanel(status)

	// Navigation should not crash on empty status
	panel.Next()
	panel.Previous()

	// GetSelectedNodeID should return empty string
	nodeID := panel.GetSelectedNodeID()
	if nodeID != "" {
		t.Errorf("Expected empty node ID for empty status, got %q", nodeID)
	}

	// Selection should remain 0
	if panel.selectedIndex != 0 {
		t.Errorf("Expected selection to be 0, got %d", panel.selectedIndex)
	}
}

// TestValidationPanel_NilStatus tests panel behavior with nil status
func TestValidationPanel_NilStatus(t *testing.T) {
	panel := NewValidationPanel(nil)

	// Navigation should not crash on nil status
	panel.Next()
	panel.Previous()

	// GetSelectedNodeID should return empty string
	nodeID := panel.GetSelectedNodeID()
	if nodeID != "" {
		t.Errorf("Expected empty node ID for nil status, got %q", nodeID)
	}
}

// TestValidationPanel_MixedErrorsWarnings tests navigation with both errors and warnings
func TestValidationPanel_MixedErrorsWarnings(t *testing.T) {
	status := NewValidationStatus()
	status.AddError("node1", "error1", "Error 1")
	status.AddError("node2", "error2", "Error 2")
	status.AddWarning("node3", "Warning 1")
	status.AddWarning("node4", "Warning 2")
	status.SetValidated()

	panel := NewValidationPanel(status)

	// Test that errors come before warnings in navigation
	expectedNodeIDs := []string{"node1", "node2", "node3", "node4"}

	for i, expectedID := range expectedNodeIDs {
		nodeID := panel.GetSelectedNodeID()
		if nodeID != expectedID {
			t.Errorf("At index %d, expected node ID %q, got %q", i, expectedID, nodeID)
		}
		panel.Next()
	}

	// After cycling through all items, should be back at start
	nodeID := panel.GetSelectedNodeID()
	if nodeID != "node1" {
		t.Errorf("After full cycle, expected to be back at node1, got %q", nodeID)
	}
}

// TestValidationPanel_UpdateWithSameSize tests that selection is preserved when status has same size
func TestValidationPanel_UpdateWithSameSize(t *testing.T) {
	status1 := NewValidationStatus()
	status1.AddError("node1", "error1", "Error 1")
	status1.AddError("node2", "error2", "Error 2")
	status1.SetValidated()

	panel := NewValidationPanel(status1)
	panel.selectedIndex = 1 // Select second error

	// Update with new status of same size
	status2 := NewValidationStatus()
	status2.AddError("node3", "error3", "Error 3")
	status2.AddError("node4", "error4", "Error 4")
	status2.SetValidated()

	panel.UpdateStatus(status2)

	// Selection should be preserved
	if panel.selectedIndex != 1 {
		t.Errorf("Expected selection to be preserved at 1, got %d", panel.selectedIndex)
	}

	// But now pointing to different node
	nodeID := panel.GetSelectedNodeID()
	if nodeID != "node4" {
		t.Errorf("Expected node ID node4, got %q", nodeID)
	}
}
