package tui

import (
	"testing"

	"github.com/dshills/goterm"
)

// TestViewManager_RegisterView tests view registration
func TestViewManager_RegisterView(t *testing.T) {
	tests := []struct {
		name      string
		view      View
		wantError bool
		errMsg    string
	}{
		{
			name:      "register valid view",
			view:      NewWorkflowExplorerView(),
			wantError: false,
		},
		{
			name:      "register nil view",
			view:      nil,
			wantError: true,
			errMsg:    "cannot register nil view",
		},
		{
			name:      "register duplicate view",
			view:      NewWorkflowExplorerView(),
			wantError: true,
			errMsg:    "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewViewManager()

			// For duplicate test, register first
			if tt.name == "register duplicate view" {
				_ = vm.RegisterView(NewWorkflowExplorerView())
			}

			err := vm.RegisterView(tt.view)

			if tt.wantError && err == nil {
				t.Errorf("RegisterView() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("RegisterView() unexpected error: %v", err)
			}

			if tt.wantError && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("RegisterView() error = %q, want substring %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestViewManager_SwitchTo tests view switching
func TestViewManager_SwitchTo(t *testing.T) {
	tests := []struct {
		name       string
		viewName   string
		wantError  bool
		wantActive string
	}{
		{
			name:       "switch to registered view",
			viewName:   "explorer",
			wantError:  false,
			wantActive: "explorer",
		},
		{
			name:       "switch to unregistered view",
			viewName:   "nonexistent",
			wantError:  true,
			wantActive: "explorer",
		},
		{
			name:       "switch to same view (no-op)",
			viewName:   "explorer",
			wantError:  false,
			wantActive: "explorer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewViewManager()
			explorer := NewWorkflowExplorerView()
			builder := NewWorkflowBuilderView()

			_ = vm.RegisterView(explorer)
			_ = vm.RegisterView(builder)
			_ = vm.Initialize("explorer")

			err := vm.SwitchTo(tt.viewName)

			if tt.wantError && err == nil {
				t.Errorf("SwitchTo() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("SwitchTo() unexpected error: %v", err)
			}

			activeView := vm.GetCurrentView()
			if activeView == nil {
				t.Fatal("GetCurrentView() returned nil")
			}

			if activeView.Name() != tt.wantActive {
				t.Errorf("active view = %q, want %q", activeView.Name(), tt.wantActive)
			}
		})
	}
}

// TestViewManager_GoBack tests back navigation
func TestViewManager_GoBack(t *testing.T) {
	vm := NewViewManager()
	explorer := NewWorkflowExplorerView()
	builder := NewWorkflowBuilderView()
	monitor := NewExecutionMonitorView()

	_ = vm.RegisterView(explorer)
	_ = vm.RegisterView(builder)
	_ = vm.RegisterView(monitor)
	_ = vm.Initialize("explorer")

	// Switch to builder
	_ = vm.SwitchTo("builder")

	// Switch to monitor
	_ = vm.SwitchTo("monitor")

	// Current should be monitor
	if vm.GetCurrentView().Name() != "monitor" {
		t.Errorf("current view = %q, want %q", vm.GetCurrentView().Name(), "monitor")
	}

	// Go back to builder
	err := vm.GoBack()
	if err != nil {
		t.Errorf("GoBack() unexpected error: %v", err)
	}

	if vm.GetCurrentView().Name() != "builder" {
		t.Errorf("current view after back = %q, want %q", vm.GetCurrentView().Name(), "builder")
	}

	// Go back to explorer
	err = vm.GoBack()
	if err != nil {
		t.Errorf("GoBack() unexpected error: %v", err)
	}

	if vm.GetCurrentView().Name() != "explorer" {
		t.Errorf("current view after back = %q, want %q", vm.GetCurrentView().Name(), "explorer")
	}

	// No more history, should error
	err = vm.GoBack()
	if err == nil {
		t.Errorf("GoBack() expected error for empty history")
	}
}

// TestViewManager_NextView tests tab cycling
func TestViewManager_NextView(t *testing.T) {
	vm := NewViewManager()
	explorer := NewWorkflowExplorerView()
	builder := NewWorkflowBuilderView()
	monitor := NewExecutionMonitorView()
	registry := NewServerRegistryView()

	_ = vm.RegisterView(explorer)
	_ = vm.RegisterView(builder)
	_ = vm.RegisterView(monitor)
	_ = vm.RegisterView(registry)
	_ = vm.Initialize("explorer")

	// Expected order (alphabetical by name):
	// builder -> explorer -> monitor -> registry -> builder
	// Starting from explorer, next should be: monitor -> registry -> builder -> explorer

	currentView := vm.GetCurrentView().Name()
	expectedSequence := []string{"monitor", "registry", "builder", "explorer"}

	for i, expected := range expectedSequence {
		err := vm.NextView()
		if err != nil {
			t.Errorf("NextView() step %d unexpected error: %v", i, err)
		}

		currentView = vm.GetCurrentView().Name()
		if currentView != expected {
			t.Errorf("NextView() step %d = %q, want %q", i, currentView, expected)
		}
	}
}

// TestViewManager_ListViews tests view listing
func TestViewManager_ListViews(t *testing.T) {
	vm := NewViewManager()

	// Empty list
	views := vm.ListViews()
	if len(views) != 0 {
		t.Errorf("ListViews() empty = %d, want 0", len(views))
	}

	// Add views
	_ = vm.RegisterView(NewWorkflowExplorerView())
	_ = vm.RegisterView(NewWorkflowBuilderView())

	views = vm.ListViews()
	if len(views) != 2 {
		t.Errorf("ListViews() count = %d, want 2", len(views))
	}

	// Check all names are present
	hasExplorer := false
	hasBuilder := false
	for _, name := range views {
		if name == "explorer" {
			hasExplorer = true
		}
		if name == "builder" {
			hasBuilder = true
		}
	}

	if !hasExplorer || !hasBuilder {
		t.Errorf("ListViews() missing expected views, got: %v", views)
	}
}

// TestViewManager_Initialize tests initialization
func TestViewManager_Initialize(t *testing.T) {
	tests := []struct {
		name         string
		initialView  string
		wantError    bool
		shouldBeInit bool
	}{
		{
			name:         "initialize with valid view",
			initialView:  "explorer",
			wantError:    false,
			shouldBeInit: true,
		},
		{
			name:         "initialize with invalid view",
			initialView:  "nonexistent",
			wantError:    true,
			shouldBeInit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewViewManager()
			explorer := NewWorkflowExplorerView()
			_ = vm.RegisterView(explorer)

			err := vm.Initialize(tt.initialView)

			if tt.wantError && err == nil {
				t.Errorf("Initialize() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Initialize() unexpected error: %v", err)
			}

			if tt.shouldBeInit {
				if vm.GetCurrentView() == nil {
					t.Error("Initialize() succeeded but no active view")
				} else if !vm.GetCurrentView().IsActive() {
					t.Error("Initialize() succeeded but view not marked active")
				}
			}
		})
	}
}

// TestViewManager_Shutdown tests cleanup
func TestViewManager_Shutdown(t *testing.T) {
	vm := NewViewManager()
	explorer := NewWorkflowExplorerView()
	builder := NewWorkflowBuilderView()

	_ = vm.RegisterView(explorer)
	_ = vm.RegisterView(builder)
	_ = vm.Initialize("explorer")

	// Switch views to build history
	_ = vm.SwitchTo("builder")

	// Shutdown
	err := vm.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() unexpected error: %v", err)
	}

	// Verify cleanup
	if vm.GetCurrentView() != nil {
		t.Error("Shutdown() did not clear active view")
	}

	if len(vm.history) != 0 {
		t.Error("Shutdown() did not clear history")
	}
}

// TestView_Lifecycle tests view initialization and cleanup
func TestView_Lifecycle(t *testing.T) {
	view := NewWorkflowExplorerView()

	// Initially not active
	if view.IsActive() {
		t.Error("new view should not be active")
	}

	// Initialize
	err := view.Init()
	if err != nil {
		t.Errorf("Init() unexpected error: %v", err)
	}

	// Set active
	view.SetActive(true)
	if !view.IsActive() {
		t.Error("SetActive(true) failed")
	}

	// Cleanup
	err = view.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() unexpected error: %v", err)
	}

	// Deactivate
	view.SetActive(false)
	if view.IsActive() {
		t.Error("SetActive(false) failed")
	}
}

// TestView_HandleKey tests basic key handling
func TestView_HandleKey(t *testing.T) {
	tests := []struct {
		name      string
		view      View
		key       KeyEvent
		wantError bool
	}{
		{
			name:      "explorer handles j key",
			view:      NewWorkflowExplorerView(),
			key:       KeyEvent{Key: 'j'},
			wantError: false,
		},
		{
			name:      "builder handles k key",
			view:      NewWorkflowBuilderView(),
			key:       KeyEvent{Key: 'k'},
			wantError: false,
		},
		{
			name:      "monitor handles l key",
			view:      NewExecutionMonitorView(),
			key:       KeyEvent{Key: 'l'},
			wantError: false,
		},
		{
			name:      "registry handles Enter",
			view:      NewServerRegistryView(),
			key:       KeyEvent{IsSpecial: true, Special: "Enter"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.view.Init()
			err := tt.view.HandleKey(tt.key)

			if tt.wantError && err == nil {
				t.Errorf("HandleKey() expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("HandleKey() unexpected error: %v", err)
			}
		})
	}
}

// TestView_Render tests basic rendering
func TestView_Render(t *testing.T) {
	views := []View{
		NewWorkflowExplorerView(),
		NewWorkflowBuilderView(),
		NewExecutionMonitorView(),
		NewServerRegistryView(),
	}

	screen := goterm.NewScreen(80, 24)

	for _, view := range views {
		t.Run(view.Name(), func(t *testing.T) {
			_ = view.Init()

			err := view.Render(screen)
			if err != nil {
				t.Errorf("Render() unexpected error: %v", err)
			}
		})
	}
}

// TestViewManager_AddSwitchHook tests transition hooks
func TestViewManager_AddSwitchHook(t *testing.T) {
	vm := NewViewManager()
	explorer := NewWorkflowExplorerView()
	builder := NewWorkflowBuilderView()

	_ = vm.RegisterView(explorer)
	_ = vm.RegisterView(builder)
	_ = vm.Initialize("explorer")

	// Add hook that records transitions
	var transitions []string
	vm.AddSwitchHook(func(from, to View) error {
		fromName := "nil"
		if from != nil {
			fromName = from.Name()
		}
		transitions = append(transitions, fromName+" -> "+to.Name())
		return nil
	})

	// Switch views
	_ = vm.SwitchTo("builder")

	if len(transitions) != 1 {
		t.Errorf("hook called %d times, want 1", len(transitions))
	}

	expected := "explorer -> builder"
	if transitions[0] != expected {
		t.Errorf("transition = %q, want %q", transitions[0], expected)
	}
}

// TestViewManager_AddSwitchHook_Error tests hook error handling
func TestViewManager_AddSwitchHook_Error(t *testing.T) {
	vm := NewViewManager()
	explorer := NewWorkflowExplorerView()
	builder := NewWorkflowBuilderView()

	_ = vm.RegisterView(explorer)
	_ = vm.RegisterView(builder)
	_ = vm.Initialize("explorer")

	// Add hook that fails
	vm.AddSwitchHook(func(from, to View) error {
		return &viewSwitchError{"hook failed"}
	})

	// Switch should fail and rollback
	err := vm.SwitchTo("builder")
	if err == nil {
		t.Error("SwitchTo() expected error from hook")
	}

	// Should still be on explorer
	if vm.GetCurrentView().Name() != "explorer" {
		t.Errorf("current view = %q, want %q (rollback failed)", vm.GetCurrentView().Name(), "explorer")
	}

	// Explorer should still be active
	if !explorer.IsActive() {
		t.Error("explorer should still be active after rollback")
	}
}

// viewSwitchError is a test error type
type viewSwitchError struct {
	msg string
}

func (e *viewSwitchError) Error() string {
	return e.msg
}

// contains is a helper to check substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

// indexOf finds the index of substr in s
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
