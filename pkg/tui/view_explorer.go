package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/goterm"
)

// WorkflowExplorerView displays a list of available workflows
// Users can browse, search, and select workflows to edit or execute
type WorkflowExplorerView struct {
	name         string
	active       bool
	workflows    []string // List of workflow names
	selectedIdx  int      // Currently selected workflow index
	statusMsg    string   // Status message to display
	initialized  bool
	width        int          // View width
	height       int          // View height
	viewSwitcher ViewSwitcher // For switching to other views
	workflowsDir string       // Directory containing workflows
}

// NewWorkflowExplorerView creates a new workflow explorer view
func NewWorkflowExplorerView() *WorkflowExplorerView {
	// Get workflows directory from environment or use default
	workflowsDir := os.Getenv("GOFLOW_WORKFLOWS_DIR")
	if workflowsDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			workflowsDir = "."
		} else {
			workflowsDir = filepath.Join(homeDir, ".goflow", "workflows")
		}
	}

	return &WorkflowExplorerView{
		name:         "explorer",
		active:       false,
		workflows:    make([]string, 0),
		selectedIdx:  0,
		workflowsDir: workflowsDir,
	}
}

// SetViewSwitcher stores the ViewSwitcher for requesting view changes
func (v *WorkflowExplorerView) SetViewSwitcher(switcher ViewSwitcher) {
	v.viewSwitcher = switcher
}

// Name returns the unique identifier for this view
func (v *WorkflowExplorerView) Name() string {
	return v.name
}

// Init initializes the workflow explorer view
func (v *WorkflowExplorerView) Init() error {
	if v.initialized {
		return nil // already initialized, preserve state
	}

	// Load workflows from filesystem
	v.workflows = make([]string, 0)

	// Check if workflows directory exists
	if _, err := os.Stat(v.workflowsDir); os.IsNotExist(err) {
		// Directory doesn't exist yet - show empty list
		v.statusMsg = "No workflows found. Create one with 'goflow init <name>'"
		v.initialized = true
		return nil
	}

	// Walk the workflows directory
	err := filepath.WalkDir(v.workflowsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip files we can't access
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only include .yaml and .yml files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Get relative path from workflows directory
		relPath, err := filepath.Rel(v.workflowsDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		v.workflows = append(v.workflows, relPath)
		return nil
	})

	if err != nil {
		v.statusMsg = "Error loading workflows: " + err.Error()
		v.initialized = true
		return nil
	}

	v.selectedIdx = 0
	if len(v.workflows) > 0 {
		v.statusMsg = "Ready"
	} else {
		v.statusMsg = "No workflows found. Create one with 'goflow init <name>'"
	}
	v.initialized = true

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *WorkflowExplorerView) Cleanup() error {
	// Preserve state for when we return to this view
	return nil
}

// HandleKey processes keyboard input events
func (v *WorkflowExplorerView) HandleKey(event KeyEvent) error {
	// Keyboard navigation:
	// - j/k: move selection up/down
	// - Enter: open selected workflow in builder
	// - /: start search (not yet implemented)
	// - n: create new workflow (not yet implemented)
	// - d: delete selected workflow (not yet implemented)
	// - r: rename selected workflow (not yet implemented)

	switch {
	case event.Key == 'j':
		// Move selection down
		if len(v.workflows) > 0 && v.selectedIdx < len(v.workflows)-1 {
			v.selectedIdx++
		}
	case event.Key == 'k':
		// Move selection up
		if v.selectedIdx > 0 {
			v.selectedIdx--
		}
	case event.IsSpecial && event.Special == "Enter":
		// Open selected workflow in builder
		if len(v.workflows) == 0 {
			v.statusMsg = "No workflows to open"
			return nil
		}

		if v.selectedIdx >= len(v.workflows) {
			v.statusMsg = "Invalid selection"
			return nil
		}

		// Get selected workflow path
		selectedWorkflow := v.workflows[v.selectedIdx]
		workflowPath := filepath.Join(v.workflowsDir, selectedWorkflow)

		// Set status before switching
		v.statusMsg = "Opening " + selectedWorkflow + "..."

		// Switch to builder view
		if v.viewSwitcher != nil {
			// Get the builder view and set the workflow
			// Note: We need to pass the workflow path to the builder somehow
			// For now, we'll use environment variable or global state
			_ = os.Setenv("GOFLOW_CURRENT_WORKFLOW", workflowPath) // Best effort; ignore error

			if err := v.viewSwitcher.SwitchToView("builder"); err != nil {
				v.statusMsg = "Error opening workflow: " + err.Error()
				return nil
			}
		} else {
			v.statusMsg = "Cannot switch views: no view switcher configured"
		}

	case event.Key == '/':
		// Start search mode
		v.statusMsg = "Search mode (not yet implemented)"
	case event.Key == 'n':
		// Create new workflow
		v.statusMsg = "Create new workflow (not yet implemented)"
	}

	return nil
}

// Render draws the workflow explorer to the screen
func (v *WorkflowExplorerView) Render(screen *goterm.Screen) error {
	// TODO: Implement rendering
	// Layout:
	// +----------------------------------+
	// | Workflow Explorer    [Tab: Next] |
	// +----------------------------------+
	// |                                  |
	// | > example-workflow.yaml          |
	// |   data-pipeline.yaml             |
	// |   api-integration.yaml           |
	// |                                  |
	// +----------------------------------+
	// | Status: Ready          [?: Help] |
	// +----------------------------------+

	_, height := screen.Size()
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Clear screen
	screen.Clear()

	// Title bar
	title := "Workflow Explorer [Tab: Switch View] [?: Help]"
	screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	// Workflow list
	for i, workflow := range v.workflows {
		y := 2 + i
		if y >= height-1 {
			break // don't overflow screen
		}

		prefix := "  "
		style := goterm.StyleNone
		if i == v.selectedIdx {
			prefix = "> "
			style = goterm.StyleReverse
		}

		screen.DrawText(0, y, prefix+workflow, fg, bg, style)
	}

	// Status bar (bottom line)
	statusLine := "Status: " + v.statusMsg
	screen.DrawText(0, height-1, statusLine, fg, bg, goterm.StyleNone)

	return nil
}

// IsActive returns whether this view is currently active
func (v *WorkflowExplorerView) IsActive() bool {
	return v.active
}

// SetActive updates the active state of the view
func (v *WorkflowExplorerView) SetActive(active bool) {
	v.active = active
}

// SetBounds sets the view dimensions
func (v *WorkflowExplorerView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
