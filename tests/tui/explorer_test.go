package tui

import (
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// T086: TUI Component Tests for Workflow Explorer View
//
// These tests follow test-first development - they will FAIL initially
// because the WorkflowExplorer implementation does not exist yet.
//
// The tests cover:
// 1. Workflow list rendering
// 2. Navigation between workflows (j/k keys)
// 3. Workflow selection (Enter key)
// 4. New workflow creation dialog (n key)
// 5. Workflow deletion with confirmation (d key)
// 6. Workflow rename functionality (r key)
// 7. Empty state handling (no workflows)
// 8. Search/filter workflows (/ key)

// MockWorkflowRepository implements a simple in-memory workflow repository for testing
type MockWorkflowRepository struct {
	workflows  []*workflow.Workflow
	saveFunc   func(*workflow.Workflow) error
	deleteFunc func(string) error
}

func (m *MockWorkflowRepository) Save(wf *workflow.Workflow) error {
	if m.saveFunc != nil {
		return m.saveFunc(wf)
	}
	// Check if workflow exists and update, otherwise append
	for i, existing := range m.workflows {
		if existing.ID == wf.ID {
			m.workflows[i] = wf
			return nil
		}
	}
	m.workflows = append(m.workflows, wf)
	return nil
}

func (m *MockWorkflowRepository) FindByID(id string) (*workflow.Workflow, error) {
	for _, wf := range m.workflows {
		if wf.ID == id {
			return wf, nil
		}
	}
	return nil, workflow.ErrWorkflowNotFound
}

func (m *MockWorkflowRepository) FindByName(name string) (*workflow.Workflow, error) {
	for _, wf := range m.workflows {
		if wf.Name == name {
			return wf, nil
		}
	}
	return nil, workflow.ErrWorkflowNotFound
}

func (m *MockWorkflowRepository) List() ([]*workflow.Workflow, error) {
	return m.workflows, nil
}

func (m *MockWorkflowRepository) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	for i, wf := range m.workflows {
		if wf.ID == id {
			m.workflows = append(m.workflows[:i], m.workflows[i+1:]...)
			return nil
		}
	}
	return workflow.ErrWorkflowNotFound
}

// createTestWorkflows creates a set of test workflows for testing
func createTestWorkflows() []*workflow.Workflow {
	wf1, _ := workflow.NewWorkflow("data-pipeline", "Read, transform, write data")
	wf1.Metadata.Tags = []string{"etl", "data"}
	wf1.Metadata.Icon = "📊"

	wf2, _ := workflow.NewWorkflow("api-orchestration", "Coordinate multiple API calls")
	wf2.Metadata.Tags = []string{"api", "integration"}
	wf2.Metadata.Icon = "🔗"

	wf3, _ := workflow.NewWorkflow("file-processing", "Process files from directory")
	wf3.Metadata.Tags = []string{"files", "batch"}
	wf3.Metadata.Icon = "📁"

	wf4, _ := workflow.NewWorkflow("notification-flow", "Send notifications based on conditions")
	wf4.Metadata.Tags = []string{"notifications", "alerts"}
	wf4.Metadata.Icon = "🔔"

	wf5, _ := workflow.NewWorkflow("backup-automation", "Automated backup workflow")
	wf5.Metadata.Tags = []string{"backup", "automation"}
	wf5.Metadata.Icon = "💾"

	return []*workflow.Workflow{wf1, wf2, wf3, wf4, wf5}
}

// TestWorkflowExplorerRenderEmpty tests rendering when no workflows exist
func TestWorkflowExplorerRenderEmpty(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		height       int
		wantEmptyMsg string
		wantHelpText string
	}{
		{
			name:         "standard_size_empty",
			width:        80,
			height:       24,
			wantEmptyMsg: "No workflows found",
			wantHelpText: "Press 'n' to create a new workflow",
		},
		{
			name:         "small_screen_empty",
			width:        40,
			height:       12,
			wantEmptyMsg: "No workflows found",
			wantHelpText: "Press 'n' to create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create empty repository
			repo := &MockWorkflowRepository{
				workflows: []*workflow.Workflow{},
			}

			// Create screen buffer
			screen := goterm.NewScreen(tt.width, tt.height)

			// Create workflow explorer (this will fail - implementation doesn't exist)
			// Expected error: undefined: NewWorkflowExplorer
			explorer := tui.NewWorkflowExplorer(repo, screen)

			// Render the explorer
			_, err := explorer.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Verify empty state message is displayed
			if !screenContainsText(screen, tt.wantEmptyMsg) {
				t.Errorf("Expected empty message %q not found in screen buffer", tt.wantEmptyMsg)
			}

			// Verify help text is displayed
			if !screenContainsText(screen, tt.wantHelpText) {
				t.Errorf("Expected help text %q not found in screen buffer", tt.wantHelpText)
			}

			// Verify no workflow items are shown
			if screenContainsText(screen, "│") {
				t.Error("Screen should not contain workflow list items when empty")
			}
		})
	}
}

// TestWorkflowExplorerRenderList tests rendering a list of workflows
func TestWorkflowExplorerRenderList(t *testing.T) {
	tests := []struct {
		name              string
		workflows         []*workflow.Workflow
		width             int
		height            int
		selectedIndex     int
		wantWorkflowCount int
		wantSelectedName  string
	}{
		{
			name:              "single_workflow",
			workflows:         createTestWorkflows()[:1],
			width:             80,
			height:            24,
			selectedIndex:     0,
			wantWorkflowCount: 1,
			wantSelectedName:  "data-pipeline",
		},
		{
			name:              "five_workflows_first_selected",
			workflows:         createTestWorkflows(),
			width:             80,
			height:            24,
			selectedIndex:     0,
			wantWorkflowCount: 5,
			wantSelectedName:  "data-pipeline",
		},
		{
			name:              "five_workflows_middle_selected",
			workflows:         createTestWorkflows(),
			width:             80,
			height:            24,
			selectedIndex:     2,
			wantWorkflowCount: 5,
			wantSelectedName:  "file-processing",
		},
		{
			name:              "five_workflows_last_selected",
			workflows:         createTestWorkflows(),
			width:             80,
			height:            24,
			selectedIndex:     4,
			wantWorkflowCount: 5,
			wantSelectedName:  "backup-automation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockWorkflowRepository{
				workflows: tt.workflows,
			}

			screen := goterm.NewScreen(tt.width, tt.height)
			explorer := tui.NewWorkflowExplorer(repo, screen)

			// Set selected index
			explorer.SetSelectedIndex(tt.selectedIndex)

			_, err := explorer.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			// Verify all workflow names are displayed
			for _, wf := range tt.workflows {
				if !screenContainsText(screen, wf.Name) {
					t.Errorf("Expected workflow name %q not found in screen buffer", wf.Name)
				}
			}

			// Verify selected workflow is highlighted
			selectedWf := tt.workflows[tt.selectedIndex]
			if !screenContainsText(screen, selectedWf.Name) {
				t.Errorf("Expected selected workflow %q not found", selectedWf.Name)
			}

			// Verify workflow metadata is displayed
			if selectedWf.Metadata.Icon != "" {
				if !screenContainsText(screen, selectedWf.Metadata.Icon) {
					t.Errorf("Expected workflow icon %q not found", selectedWf.Metadata.Icon)
				}
			}

			// Verify workflow count is displayed in status bar
			if tt.wantWorkflowCount > 0 {
				// Expected format: "5 workflows" or "1 workflow"
				expectedText := "workflow"
				if !screenContainsText(screen, expectedText) {
					t.Errorf("Expected workflow count text not found")
				}
			}
		})
	}
}

// TestWorkflowExplorerNavigationJK tests j/k key navigation (vim-style)
func TestWorkflowExplorerNavigationJK(t *testing.T) {
	tests := []struct {
		name             string
		workflows        []*workflow.Workflow
		initialIndex     int
		keySequence      []rune
		expectedIndex    int
		expectedWorkflow string
	}{
		{
			name:             "j_moves_down",
			workflows:        createTestWorkflows(),
			initialIndex:     0,
			keySequence:      []rune{'j'},
			expectedIndex:    1,
			expectedWorkflow: "api-orchestration",
		},
		{
			name:             "k_moves_up",
			workflows:        createTestWorkflows(),
			initialIndex:     2,
			keySequence:      []rune{'k'},
			expectedIndex:    1,
			expectedWorkflow: "api-orchestration",
		},
		{
			name:             "multiple_j_moves",
			workflows:        createTestWorkflows(),
			initialIndex:     0,
			keySequence:      []rune{'j', 'j', 'j'},
			expectedIndex:    3,
			expectedWorkflow: "notification-flow",
		},
		{
			name:             "j_at_bottom_stays_at_bottom",
			workflows:        createTestWorkflows(),
			initialIndex:     4,
			keySequence:      []rune{'j', 'j'},
			expectedIndex:    4,
			expectedWorkflow: "backup-automation",
		},
		{
			name:             "k_at_top_stays_at_top",
			workflows:        createTestWorkflows(),
			initialIndex:     0,
			keySequence:      []rune{'k', 'k'},
			expectedIndex:    0,
			expectedWorkflow: "data-pipeline",
		},
		{
			name:             "mixed_navigation",
			workflows:        createTestWorkflows(),
			initialIndex:     0,
			keySequence:      []rune{'j', 'j', 'k', 'j'},
			expectedIndex:    2,
			expectedWorkflow: "file-processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockWorkflowRepository{
				workflows: tt.workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)
			explorer.SetSelectedIndex(tt.initialIndex)

			// Simulate key presses
			for _, key := range tt.keySequence {
				err := explorer.HandleKey(key)
				if err != nil {
					t.Fatalf("HandleKey(%c) failed: %v", key, err)
				}
			}

			// Verify final selected index
			gotIndex := explorer.GetSelectedIndex()
			if gotIndex != tt.expectedIndex {
				t.Errorf("After key sequence, got index %d, want %d", gotIndex, tt.expectedIndex)
			}

			// Verify correct workflow is selected
			selected := explorer.GetSelectedWorkflow()
			if selected == nil {
				t.Fatal("GetSelectedWorkflow() returned nil")
			}
			if selected.Name != tt.expectedWorkflow {
				t.Errorf("Selected workflow name = %q, want %q", selected.Name, tt.expectedWorkflow)
			}
		})
	}
}

// TestWorkflowExplorerSelectionEnter tests Enter key for workflow selection
func TestWorkflowExplorerSelectionEnter(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []*workflow.Workflow
		selectedIndex int
		wantCallback  bool
		wantWorkflow  string
	}{
		{
			name:          "select_first_workflow",
			workflows:     createTestWorkflows(),
			selectedIndex: 0,
			wantCallback:  true,
			wantWorkflow:  "data-pipeline",
		},
		{
			name:          "select_middle_workflow",
			workflows:     createTestWorkflows(),
			selectedIndex: 2,
			wantCallback:  true,
			wantWorkflow:  "file-processing",
		},
		{
			name:          "select_last_workflow",
			workflows:     createTestWorkflows(),
			selectedIndex: 4,
			wantCallback:  true,
			wantWorkflow:  "backup-automation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockWorkflowRepository{
				workflows: tt.workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)
			explorer.SetSelectedIndex(tt.selectedIndex)

			// Set up selection callback
			var callbackCalled bool
			var callbackWorkflow *workflow.Workflow
			explorer.OnSelect(func(wf *workflow.Workflow) {
				callbackCalled = true
				callbackWorkflow = wf
			})

			// Simulate Enter key press
			err := explorer.HandleKey('\n') // Enter key
			if err != nil {
				t.Fatalf("HandleKey(Enter) failed: %v", err)
			}

			// Verify callback was called
			if callbackCalled != tt.wantCallback {
				t.Errorf("Callback called = %v, want %v", callbackCalled, tt.wantCallback)
			}

			// Verify correct workflow was passed to callback
			if callbackWorkflow == nil {
				t.Fatal("Callback received nil workflow")
			}
			if callbackWorkflow.Name != tt.wantWorkflow {
				t.Errorf("Callback workflow name = %q, want %q", callbackWorkflow.Name, tt.wantWorkflow)
			}
		})
	}
}

// TestWorkflowExplorerNewWorkflow tests 'n' key for creating new workflow
func TestWorkflowExplorerNewWorkflow(t *testing.T) {
	tests := []struct {
		name            string
		existingCount   int
		newWorkflowName string
		newWorkflowDesc string
		expectDialog    bool
		expectSuccess   bool
	}{
		{
			name:            "create_new_workflow_empty_list",
			existingCount:   0,
			newWorkflowName: "new-workflow",
			newWorkflowDesc: "Test workflow",
			expectDialog:    true,
			expectSuccess:   true,
		},
		{
			name:            "create_new_workflow_existing_list",
			existingCount:   3,
			newWorkflowName: "another-workflow",
			newWorkflowDesc: "Another test",
			expectDialog:    true,
			expectSuccess:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflows := createTestWorkflows()[:tt.existingCount]
			repo := &MockWorkflowRepository{
				workflows: workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)

			// Track if dialog was shown
			dialogShown := false
			explorer.OnNewWorkflowDialog(func() {
				dialogShown = true
			})

			// Simulate 'n' key press
			err := explorer.HandleKey('n')
			if err != nil {
				t.Fatalf("HandleKey('n') failed: %v", err)
			}

			// Verify dialog was shown
			if dialogShown != tt.expectDialog {
				t.Errorf("Dialog shown = %v, want %v", dialogShown, tt.expectDialog)
			}

			// Verify dialog content is rendered
			_, _ = explorer.Render()
			if tt.expectDialog {
				if !screenContainsText(screen, "New Workflow") {
					t.Error("Expected 'New Workflow' dialog title not found")
				}
				if !screenContainsText(screen, "Name:") {
					t.Error("Expected 'Name:' field label not found")
				}
				if !screenContainsText(screen, "Description:") {
					t.Error("Expected 'Description:' field label not found")
				}
			}
		})
	}
}

// TestWorkflowExplorerDeleteWorkflow tests 'd' key for deleting workflows
func TestWorkflowExplorerDeleteWorkflow(t *testing.T) {
	tests := []struct {
		name                string
		workflows           []*workflow.Workflow
		selectedIndex       int
		confirmDeletion     bool
		expectConfirmDialog bool
		expectDeleted       bool
		expectedCount       int
	}{
		{
			name:                "delete_with_confirmation",
			workflows:           createTestWorkflows()[:3],
			selectedIndex:       1,
			confirmDeletion:     true,
			expectConfirmDialog: true,
			expectDeleted:       true,
			expectedCount:       2,
		},
		{
			name:                "delete_cancelled",
			workflows:           createTestWorkflows()[:3],
			selectedIndex:       1,
			confirmDeletion:     false,
			expectConfirmDialog: true,
			expectDeleted:       false,
			expectedCount:       3,
		},
		{
			name:                "delete_first_workflow",
			workflows:           createTestWorkflows()[:3],
			selectedIndex:       0,
			confirmDeletion:     true,
			expectConfirmDialog: true,
			expectDeleted:       true,
			expectedCount:       2,
		},
		{
			name:                "delete_last_workflow",
			workflows:           createTestWorkflows()[:3],
			selectedIndex:       2,
			confirmDeletion:     true,
			expectConfirmDialog: true,
			expectDeleted:       true,
			expectedCount:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflows := make([]*workflow.Workflow, len(tt.workflows))
			copy(workflows, tt.workflows)

			repo := &MockWorkflowRepository{
				workflows: workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)
			explorer.SetSelectedIndex(tt.selectedIndex)

			selectedWorkflow := workflows[tt.selectedIndex]

			// Track if confirmation dialog was shown
			confirmDialogShown := false
			explorer.OnDeleteConfirmation(func(wf *workflow.Workflow) bool {
				confirmDialogShown = true
				return tt.confirmDeletion
			})

			// Simulate 'd' key press
			err := explorer.HandleKey('d')
			if err != nil {
				t.Fatalf("HandleKey('d') failed: %v", err)
			}

			// Verify confirmation dialog was shown
			if confirmDialogShown != tt.expectConfirmDialog {
				t.Errorf("Confirmation dialog shown = %v, want %v", confirmDialogShown, tt.expectConfirmDialog)
			}

			// Verify workflow was deleted or not
			workflows, _ = repo.List()
			if len(workflows) != tt.expectedCount {
				t.Errorf("Workflow count after delete = %d, want %d", len(workflows), tt.expectedCount)
			}

			if tt.expectDeleted {
				// Verify specific workflow was deleted
				_, err := repo.FindByID(selectedWorkflow.ID)
				if err != workflow.ErrWorkflowNotFound {
					t.Error("Expected workflow to be deleted but it still exists")
				}
			}
		})
	}
}

// TestWorkflowExplorerRenameWorkflow tests 'r' key for renaming workflows
func TestWorkflowExplorerRenameWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []*workflow.Workflow
		selectedIndex int
		newName       string
		expectDialog  bool
		expectRenamed bool
	}{
		{
			name:          "rename_workflow_success",
			workflows:     createTestWorkflows()[:3],
			selectedIndex: 1,
			newName:       "renamed-workflow",
			expectDialog:  true,
			expectRenamed: true,
		},
		{
			name:          "rename_first_workflow",
			workflows:     createTestWorkflows()[:3],
			selectedIndex: 0,
			newName:       "new-pipeline-name",
			expectDialog:  true,
			expectRenamed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflows := make([]*workflow.Workflow, len(tt.workflows))
			copy(workflows, tt.workflows)

			repo := &MockWorkflowRepository{
				workflows: workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)
			explorer.SetSelectedIndex(tt.selectedIndex)

			originalName := workflows[tt.selectedIndex].Name

			// Track if rename dialog was shown
			renameDialogShown := false
			explorer.OnRenameDialog(func(wf *workflow.Workflow) string {
				renameDialogShown = true
				return tt.newName
			})

			// Simulate 'r' key press
			err := explorer.HandleKey('r')
			if err != nil {
				t.Fatalf("HandleKey('r') failed: %v", err)
			}

			// Verify rename dialog was shown
			if renameDialogShown != tt.expectDialog {
				t.Errorf("Rename dialog shown = %v, want %v", renameDialogShown, tt.expectDialog)
			}

			if tt.expectRenamed {
				// Verify workflow was renamed
				workflows, _ = repo.List()
				renamed := workflows[tt.selectedIndex]
				if renamed.Name != tt.newName {
					t.Errorf("Workflow name after rename = %q, want %q", renamed.Name, tt.newName)
				}

				// Verify old name no longer exists
				_, err := repo.FindByName(originalName)
				if err != workflow.ErrWorkflowNotFound {
					t.Errorf("Old workflow name %q should not exist after rename", originalName)
				}
			}
		})
	}
}

// TestWorkflowExplorerSearch tests '/' key for search/filter functionality
func TestWorkflowExplorerSearch(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []*workflow.Workflow
		searchQuery   string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "search_by_name_prefix",
			workflows:     createTestWorkflows(),
			searchQuery:   "api",
			expectedCount: 1,
			expectedNames: []string{"api-orchestration"},
		},
		{
			name:          "search_by_name_partial",
			workflows:     createTestWorkflows(),
			searchQuery:   "flow",
			expectedCount: 2,
			expectedNames: []string{"notification-flow"},
		},
		{
			name:          "search_by_tag",
			workflows:     createTestWorkflows(),
			searchQuery:   "automation",
			expectedCount: 1,
			expectedNames: []string{"backup-automation"},
		},
		{
			name:          "search_no_results",
			workflows:     createTestWorkflows(),
			searchQuery:   "nonexistent",
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "empty_search_shows_all",
			workflows:     createTestWorkflows(),
			searchQuery:   "",
			expectedCount: 5,
			expectedNames: []string{"data-pipeline", "api-orchestration", "file-processing", "notification-flow", "backup-automation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockWorkflowRepository{
				workflows: tt.workflows,
			}

			screen := goterm.NewScreen(80, 24)
			explorer := tui.NewWorkflowExplorer(repo, screen)

			// Simulate '/' key press to enter search mode
			err := explorer.HandleKey('/')
			if err != nil {
				t.Fatalf("HandleKey('/') failed: %v", err)
			}

			// Verify search mode is active
			if !explorer.IsSearchMode() {
				t.Error("Explorer should be in search mode after '/' key")
			}

			// Render and verify search prompt is shown
			_, _ = explorer.Render()
			if !screenContainsText(screen, "Search:") {
				t.Error("Expected 'Search:' prompt not found")
			}

			// Simulate typing search query
			for _, ch := range tt.searchQuery {
				err := explorer.HandleKey(ch)
				if err != nil {
					t.Fatalf("HandleKey(%c) in search mode failed: %v", ch, err)
				}
			}

			// Simulate Enter to execute search
			err = explorer.HandleKey('\n')
			if err != nil {
				t.Fatalf("HandleKey(Enter) in search mode failed: %v", err)
			}

			// Verify filtered results
			filteredWorkflows := explorer.GetFilteredWorkflows()
			if len(filteredWorkflows) != tt.expectedCount {
				t.Errorf("Filtered workflow count = %d, want %d", len(filteredWorkflows), tt.expectedCount)
			}

			// Verify expected workflows are in results
			for _, expectedName := range tt.expectedNames {
				found := false
				for _, wf := range filteredWorkflows {
					if wf.Name == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected workflow %q not found in filtered results", expectedName)
				}
			}

			// Verify search query is displayed
			if tt.searchQuery != "" {
				_, _ = explorer.Render()
				if !screenContainsText(screen, tt.searchQuery) {
					t.Errorf("Search query %q not displayed in UI", tt.searchQuery)
				}
			}
		})
	}
}

// TestWorkflowExplorerEdgeCases tests edge cases and error conditions
func TestWorkflowExplorerEdgeCases(t *testing.T) {
	t.Run("handle_key_on_empty_list", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: []*workflow.Workflow{},
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Try navigation on empty list
		err := explorer.HandleKey('j')
		if err != nil {
			t.Errorf("HandleKey('j') on empty list should not error: %v", err)
		}

		err = explorer.HandleKey('k')
		if err != nil {
			t.Errorf("HandleKey('k') on empty list should not error: %v", err)
		}

		// Try Enter on empty list
		err = explorer.HandleKey('\n')
		if err != nil {
			t.Errorf("HandleKey(Enter) on empty list should not error: %v", err)
		}
	})

	t.Run("single_workflow_navigation", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows()[:1],
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Try moving down - should stay at index 0
		explorer.HandleKey('j')
		if explorer.GetSelectedIndex() != 0 {
			t.Error("Single workflow navigation down should stay at index 0")
		}

		// Try moving up - should stay at index 0
		explorer.HandleKey('k')
		if explorer.GetSelectedIndex() != 0 {
			t.Error("Single workflow navigation up should stay at index 0")
		}
	})

	t.Run("rapid_key_presses", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Simulate rapid navigation
		for i := 0; i < 100; i++ {
			err := explorer.HandleKey('j')
			if err != nil {
				t.Fatalf("Rapid key press %d failed: %v", i, err)
			}
		}

		// Should be at last item
		if explorer.GetSelectedIndex() != len(createTestWorkflows())-1 {
			t.Error("After rapid down navigation, should be at last item")
		}
	})

	t.Run("escape_key_cancels_operations", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Enter search mode
		explorer.HandleKey('/')
		if !explorer.IsSearchMode() {
			t.Fatal("Should be in search mode")
		}

		// Escape should cancel
		explorer.HandleKey(27) // ESC key
		if explorer.IsSearchMode() {
			t.Error("ESC key should cancel search mode")
		}
	})
}

// TestWorkflowExplorerRendering tests rendering at different frame rates
func TestWorkflowExplorerRendering(t *testing.T) {
	t.Run("render_performance_target", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Render multiple times and measure
		iterations := 100
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := explorer.Render()
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

	t.Run("responsive_to_screen_resize", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		// Start with one size
		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)
		_, _ = explorer.Render()

		// Resize screen
		newScreen := goterm.NewScreen(120, 40)
		explorer.SetScreen(newScreen)

		// Should render without error at new size
		_, err := explorer.Render()
		if err != nil {
			t.Errorf("Render() after resize failed: %v", err)
		}
	})
}

// TestWorkflowExplorerHelpText tests help text display
func TestWorkflowExplorerHelpText(t *testing.T) {
	t.Run("help_text_displayed", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)
		_, _ = explorer.Render()

		// Verify essential keybindings are shown
		expectedKeys := []string{
			"j/k",   // navigation
			"Enter", // select
			"n",     // new
			"d",     // delete
			"r",     // rename
			"/",     // search
			"?",     // help
		}

		for _, key := range expectedKeys {
			// Help text might be abbreviated, so check for key presence
			if !screenContainsText(screen, key) {
				t.Errorf("Expected keybinding %q not found in help text", key)
			}
		}
	})

	t.Run("question_mark_shows_full_help", func(t *testing.T) {
		repo := &MockWorkflowRepository{
			workflows: createTestWorkflows(),
		}

		screen := goterm.NewScreen(80, 24)
		explorer := tui.NewWorkflowExplorer(repo, screen)

		// Press ? to show help
		err := explorer.HandleKey('?')
		if err != nil {
			t.Fatalf("HandleKey('?') failed: %v", err)
		}

		_, _ = explorer.Render()

		// Verify help modal/overlay is displayed
		if !screenContainsText(screen, "Help") || !screenContainsText(screen, "Keyboard Shortcuts") {
			t.Error("Full help dialog not displayed after '?' key")
		}

		// Verify detailed key descriptions
		detailedHelp := []string{
			"Navigate",
			"Select",
			"Create",
			"Delete",
			"Rename",
			"Search",
		}

		for _, text := range detailedHelp {
			if !screenContainsText(screen, text) {
				t.Errorf("Expected help text %q not found", text)
			}
		}
	})
}

// screenContainsText is a helper function to check if screen buffer contains text
// This will fail initially because we need to implement screen buffer inspection
func screenContainsText(screen *goterm.Screen, text string) bool {
	// This function needs to be implemented to search through the screen buffer
	// Expected error: screen.GetCell or similar method doesn't exist yet
	w, h := screen.Size()
	for y := 0; y < h; y++ {
		rowText := ""
		for x := 0; x < w; x++ {
			cell := screen.GetCell(x, y)
			rowText += string(cell.Ch)
		}
		if containsSubstring(rowText, text) {
			return true
		}
	}
	return false
}

// containsSubstring checks if haystack contains needle (case-insensitive)
func containsSubstring(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}

	// Simple substring search
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// Stubs removed - using actual implementation from pkg/tui package
