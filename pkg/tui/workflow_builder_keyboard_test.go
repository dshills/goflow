package tui

import (
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestHandleKey_GlobalShortcuts tests global shortcuts that work in all modes
func TestHandleKey_GlobalShortcuts(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Test '?' toggles help
	if builder.helpPanel.visible {
		t.Error("Help panel should be hidden initially")
	}
	if err := builder.HandleKey("?"); err != nil {
		t.Errorf("HandleKey('?') returned error: %v", err)
	}
	if !builder.helpPanel.visible {
		t.Error("Help panel should be visible after '?'")
	}
	if builder.mode != "help" {
		t.Errorf("Mode should be 'help', got %s", builder.mode)
	}

	// Test '?' again toggles off
	if err := builder.HandleKey("?"); err != nil {
		t.Errorf("HandleKey('?') returned error: %v", err)
	}
	if builder.helpPanel.visible {
		t.Error("Help panel should be hidden after second '?'")
	}
	if builder.mode != "normal" {
		t.Errorf("Mode should be 'normal', got %s", builder.mode)
	}

	// Test 'q' quits
	err = builder.HandleKey("q")
	if err == nil || !strings.Contains(err.Error(), "quit") {
		t.Errorf("Expected quit error, got %v", err)
	}
}

// TestHandleKey_EscapeReturnsToNormal tests Esc key returns to normal mode from all modes
func TestHandleKey_EscapeReturnsToNormal(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add a node for testing edit mode
	builder.AddNodeAtPosition("MCP Tool", Position{X: 5, Y: 5})
	builder.SelectNode("mcp-tool-0")

	// Test Esc from edit mode
	builder.mode = "edit"
	builder.ShowPropertyPanel("mcp-tool-0")
	if err := builder.HandleKey("Esc"); err != nil {
		t.Errorf("HandleKey('Esc') returned error: %v", err)
	}
	if builder.mode != "normal" {
		t.Errorf("Mode should be 'normal', got %s", builder.mode)
	}

	// Test Esc from palette mode
	builder.mode = "palette"
	builder.palette.Show()
	if err := builder.HandleKey("Esc"); err != nil {
		t.Errorf("HandleKey('Esc') returned error: %v", err)
	}
	if builder.mode != "normal" {
		t.Errorf("Mode should be 'normal', got %s", builder.mode)
	}
	if builder.palette.IsVisible() {
		t.Error("Palette should be hidden after Esc")
	}

	// Test Esc from help mode
	builder.mode = "help"
	builder.helpPanel.visible = true
	if err := builder.HandleKey("Esc"); err != nil {
		t.Errorf("HandleKey('Esc') returned error: %v", err)
	}
	if builder.mode != "normal" {
		t.Errorf("Mode should be 'normal', got %s", builder.mode)
	}
	if builder.helpPanel.visible {
		t.Error("Help panel should be hidden after Esc")
	}
}

// TestHandleKey_NormalMode tests normal mode keyboard shortcuts
func TestHandleKey_NormalMode(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add some nodes for testing
	builder.AddNodeAtPosition("MCP Tool", Position{X: 5, Y: 5})
	builder.AddNodeAtPosition("Transform", Position{X: 10, Y: 10})
	builder.SelectNode("mcp-tool-0")

	tests := []struct {
		name        string
		key         string
		wantMode    string
		wantError   bool
		errorSubstr string
		setup       func()
		verify      func(t *testing.T)
	}{
		{
			name:      "a opens palette",
			key:       "a",
			wantMode:  "palette",
			wantError: false,
			setup:     func() { builder.mode = "normal" },
			verify: func(t *testing.T) {
				if !builder.palette.IsVisible() {
					t.Error("Palette should be visible")
				}
			},
		},
		{
			name:      "d deletes node",
			key:       "d",
			wantMode:  "normal",
			wantError: false,
			setup: func() {
				builder.mode = "normal"
				builder.SelectNode("transform-1")
			},
			verify: func(t *testing.T) {
				// Verify node was deleted
				for _, node := range builder.workflow.Nodes {
					if node.GetID() == "transform-1" {
						t.Error("Node should be deleted")
					}
				}
			},
		},
		{
			name:        "d without selection fails",
			key:         "d",
			wantMode:    "normal",
			wantError:   true,
			errorSubstr: "no node selected",
			setup: func() {
				builder.mode = "normal"
				builder.selectedNodeID = ""
			},
			verify: nil,
		},
		{
			name:      "c enters edge creation mode",
			key:       "c",
			wantMode:  "normal",
			wantError: false,
			setup: func() {
				builder.mode = "normal"
				builder.SelectNode("mcp-tool-0")
			},
			verify: func(t *testing.T) {
				if !builder.edgeCreationMode {
					t.Error("Should be in edge creation mode")
				}
				if builder.edgeSourceID != "mcp-tool-0" {
					t.Errorf("Edge source should be mcp-tool-0, got %s", builder.edgeSourceID)
				}
			},
		},
		{
			name:      "v validates workflow",
			key:       "v",
			wantMode:  "normal",
			wantError: false,
			setup:     func() { builder.mode = "normal" },
			verify: func(t *testing.T) {
				if builder.validationStatus == nil {
					t.Error("Validation status should be set")
				}
			},
		},
		{
			name:      "u undoes change",
			key:       "u",
			wantMode:  "normal",
			wantError: true, // No undo history yet
			setup: func() {
				builder.mode = "normal"
				builder.undoStack = NewUndoStack(100)
			},
			verify: nil,
		},
		{
			name:      "Enter edits node properties",
			key:       "Enter",
			wantMode:  "edit",
			wantError: false,
			setup: func() {
				builder.mode = "normal"
				builder.SelectNode("mcp-tool-0")
			},
			verify: func(t *testing.T) {
				if !builder.propertyPanel.visible {
					t.Error("Property panel should be visible")
				}
			},
		},
		{
			name:      "0 resets view",
			key:       "0",
			wantMode:  "normal",
			wantError: false,
			setup:     func() { builder.mode = "normal" },
			verify: func(t *testing.T) {
				if builder.canvas.ZoomLevel != 1.0 {
					t.Errorf("Zoom should be 1.0, got %f", builder.canvas.ZoomLevel)
				}
			},
		},
		{
			name:      "f fits all nodes",
			key:       "f",
			wantMode:  "normal",
			wantError: false,
			setup:     func() { builder.mode = "normal" },
			verify:    nil, // FitAll adjusts viewport
		},
		{
			name:      "+ zooms in",
			key:       "+",
			wantMode:  "normal",
			wantError: false,
			setup:     func() { builder.mode = "normal"; builder.canvas.ZoomLevel = 1.0 },
			verify: func(t *testing.T) {
				if builder.canvas.ZoomLevel <= 1.0 {
					t.Errorf("Zoom should increase, got %f", builder.canvas.ZoomLevel)
				}
			},
		},
		{
			name:      "- zooms out",
			key:       "-",
			wantMode:  "normal",
			wantError: false,
			setup:     func() { builder.mode = "normal"; builder.canvas.ZoomLevel = 1.0 },
			verify: func(t *testing.T) {
				if builder.canvas.ZoomLevel >= 1.0 {
					t.Errorf("Zoom should decrease, got %f", builder.canvas.ZoomLevel)
				}
			},
		},
		{
			name:      "h moves node left",
			key:       "h",
			wantMode:  "normal",
			wantError: false,
			setup: func() {
				builder.mode = "normal"
				builder.SelectNode("mcp-tool-0")
			},
			verify: func(t *testing.T) {
				node := builder.canvas.nodes["mcp-tool-0"]
				if node.position.X != 4 { // Was at 5, should be at 4
					t.Errorf("Node X should be 4, got %d", node.position.X)
				}
			},
		},
		{
			name:      "j moves node down",
			key:       "j",
			wantMode:  "normal",
			wantError: false,
			setup: func() {
				builder.mode = "normal"
				builder.SelectNode("mcp-tool-0")
			},
			verify: func(t *testing.T) {
				node := builder.canvas.nodes["mcp-tool-0"]
				if node.position.Y != 6 { // Was at 5, should be at 6
					t.Errorf("Node Y should be 6, got %d", node.position.Y)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Execute
			err := builder.HandleKey(tt.key)

			// Verify error
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantError && tt.errorSubstr != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Error should contain '%s', got '%s'", tt.errorSubstr, err.Error())
				}
			}

			// Verify mode
			if builder.mode != tt.wantMode {
				t.Errorf("Mode should be '%s', got '%s'", tt.wantMode, builder.mode)
			}

			// Custom verification
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestHandleKey_EditMode tests edit mode keyboard shortcuts
func TestHandleKey_EditMode(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add node and enter edit mode
	builder.AddNodeAtPosition("Transform", Position{X: 5, Y: 5})
	builder.SelectNode("transform-0")
	builder.ShowPropertyPanel("transform-0")
	builder.mode = "edit"

	// Test Tab navigates fields
	initialIndex := builder.propertyPanel.editIndex
	if err := builder.HandleKey("Tab"); err != nil {
		t.Errorf("HandleKey('Tab') returned error: %v", err)
	}
	if builder.propertyPanel.editIndex == initialIndex {
		t.Error("Tab should move to next field")
	}

	// Test Shift+Tab navigates back
	prevIndex := builder.propertyPanel.editIndex
	if err := builder.HandleKey("Shift+Tab"); err != nil {
		t.Errorf("HandleKey('Shift+Tab') returned error: %v", err)
	}
	if builder.propertyPanel.editIndex == prevIndex {
		t.Error("Shift+Tab should move to previous field")
	}

	// Test Down navigates forward
	if err := builder.HandleKey("Down"); err != nil {
		t.Errorf("HandleKey('Down') returned error: %v", err)
	}

	// Test Up navigates back
	if err := builder.HandleKey("Up"); err != nil {
		t.Errorf("HandleKey('Up') returned error: %v", err)
	}
}

// TestHandleKey_PaletteMode tests palette mode keyboard shortcuts
func TestHandleKey_PaletteMode(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Enter palette mode
	builder.mode = "palette"
	builder.palette.Show()

	// Test Down/j navigates
	initialIndex := builder.palette.selectedIndex
	if err := builder.HandleKey("j"); err != nil {
		t.Errorf("HandleKey('j') returned error: %v", err)
	}
	if builder.palette.selectedIndex == initialIndex {
		t.Error("'j' should move selection")
	}

	// Test Up/k navigates
	prevIndex := builder.palette.selectedIndex
	if err := builder.HandleKey("k"); err != nil {
		t.Errorf("HandleKey('k') returned error: %v", err)
	}
	if builder.palette.selectedIndex == prevIndex {
		t.Error("'k' should move selection")
	}

	// Test filtering with single character
	builder.palette.Filter("") // Reset filter
	if err := builder.HandleKey("t"); err != nil {
		t.Errorf("HandleKey('t') returned error: %v", err)
	}
	if builder.palette.filterText != "t" {
		t.Errorf("Filter should be 't', got '%s'", builder.palette.filterText)
	}

	// Test Enter selects node type and creates node
	initialNodeCount := len(builder.workflow.Nodes)
	builder.palette.Filter("") // Reset filter
	builder.palette.selectedIndex = 0
	if err := builder.HandleKey("Enter"); err != nil {
		t.Errorf("HandleKey('Enter') returned error: %v", err)
	}
	if len(builder.workflow.Nodes) <= initialNodeCount {
		t.Error("Enter should create a new node")
	}
	if builder.mode != "normal" {
		t.Errorf("Mode should be 'normal' after selection, got '%s'", builder.mode)
	}
	if builder.palette.IsVisible() {
		t.Error("Palette should be hidden after selection")
	}
}

// TestHandleKey_ModeTransitions tests transitions between modes
func TestHandleKey_ModeTransitions(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add a node for testing
	builder.AddNodeAtPosition("MCP Tool", Position{X: 5, Y: 5})
	builder.SelectNode("mcp-tool-0")

	transitions := []struct {
		name      string
		startMode string
		key       string
		endMode   string
		setup     func()
		wantError bool
	}{
		{
			name:      "normal → edit via Enter",
			startMode: "normal",
			key:       "Enter",
			endMode:   "edit",
			setup:     func() { builder.SelectNode("mcp-tool-0") },
			wantError: false,
		},
		{
			name:      "edit → normal via Esc",
			startMode: "edit",
			key:       "Esc",
			endMode:   "normal",
			setup:     func() { builder.ShowPropertyPanel("mcp-tool-0") },
			wantError: false,
		},
		{
			name:      "normal → palette via a",
			startMode: "normal",
			key:       "a",
			endMode:   "palette",
			setup:     nil,
			wantError: false,
		},
		{
			name:      "palette → normal via Esc",
			startMode: "palette",
			key:       "Esc",
			endMode:   "normal",
			setup:     func() { builder.palette.Show() },
			wantError: false,
		},
		{
			name:      "normal → help via ?",
			startMode: "normal",
			key:       "?",
			endMode:   "help",
			setup:     nil,
			wantError: false,
		},
		{
			name:      "help → normal via Esc",
			startMode: "help",
			key:       "Esc",
			endMode:   "normal",
			setup:     func() { builder.helpPanel.visible = true },
			wantError: false,
		},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			builder.mode = tt.startMode
			if tt.setup != nil {
				tt.setup()
			}

			// Execute
			err := builder.HandleKey(tt.key)

			// Verify error
			if tt.wantError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify mode transition
			if builder.mode != tt.endMode {
				t.Errorf("Mode should be '%s', got '%s'", tt.endMode, builder.mode)
			}
		})
	}
}

// TestHandleKey_InvalidKeys tests that invalid keys show error messages
func TestHandleKey_InvalidKeys(t *testing.T) {
	wf, _ := workflow.NewWorkflow("test", "test workflow")
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	invalidKeys := []struct {
		mode string
		key  string
	}{
		{"normal", "x"},
		{"normal", "z"},
		{"edit", "a"},
		{"palette", "Ctrl+x"},
	}

	for _, tt := range invalidKeys {
		t.Run(tt.mode+"_"+tt.key, func(t *testing.T) {
			builder.mode = tt.mode
			err := builder.HandleKey(tt.key)
			if err == nil {
				t.Error("Expected error for invalid key, got nil")
			}
			if !strings.Contains(err.Error(), "unrecognized") {
				t.Errorf("Error should mention 'unrecognized', got: %v", err)
			}
		})
	}
}
