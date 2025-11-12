package tui

import (
	"fmt"
	"os"

	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// WorkflowBuilderView provides a visual workflow editor
// This view wraps the WorkflowBuilder component
type WorkflowBuilderView struct {
	name         string
	active       bool
	builder      *WorkflowBuilder // The actual workflow builder
	statusMsg    string           // Status message to display
	initialized  bool
	width        int          // View width
	height       int          // View height
	viewSwitcher ViewSwitcher // For switching to other views
	workflowPath string       // Path to the workflow file being edited
}

// NewWorkflowBuilderView creates a new workflow builder view
func NewWorkflowBuilderView() *WorkflowBuilderView {
	return &WorkflowBuilderView{
		name:        "builder",
		active:      false,
		statusMsg:   "Ready",
		initialized: false,
	}
}

// Name returns the unique identifier for this view
func (v *WorkflowBuilderView) Name() string {
	return v.name
}

// SetViewSwitcher implements the View interface
func (v *WorkflowBuilderView) SetViewSwitcher(switcher ViewSwitcher) {
	v.viewSwitcher = switcher
}

// Init initializes the workflow builder view
func (v *WorkflowBuilderView) Init() error {
	// Check if we should load a workflow from environment
	workflowPath := os.Getenv("GOFLOW_CURRENT_WORKFLOW")
	if workflowPath == "" && v.workflowPath == "" {
		// No workflow specified - create a new empty workflow
		wf, err := workflow.NewWorkflow("new-workflow", "New Workflow")
		if err != nil {
			return fmt.Errorf("failed to create new workflow: %w", err)
		}

		// Add start node
		startNode := &workflow.StartNode{ID: "start"}
		if err := wf.AddNode(startNode); err != nil {
			return fmt.Errorf("failed to add start node: %w", err)
		}

		// Create builder with new workflow
		builder, err := NewWorkflowBuilder(wf)
		if err != nil {
			return fmt.Errorf("failed to create workflow builder: %w", err)
		}

		v.builder = builder
		v.statusMsg = "New workflow created"
		v.initialized = true
		return nil
	}

	// Use the workflow path from environment if set
	if workflowPath != "" {
		v.workflowPath = workflowPath
		// Clear the environment variable
		_ = os.Unsetenv("GOFLOW_CURRENT_WORKFLOW") // Best effort; ignore error
	}

	// Load workflow from file
	wf, err := workflow.ParseFile(v.workflowPath)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Create builder with loaded workflow
	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		return fmt.Errorf("failed to create workflow builder: %w", err)
	}

	v.builder = builder
	v.statusMsg = "Workflow loaded"
	v.initialized = true

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *WorkflowBuilderView) Cleanup() error {
	// Preserve state for when we return to this view
	// The builder maintains its own state
	return nil
}

// HandleKey processes keyboard input events
func (v *WorkflowBuilderView) HandleKey(event KeyEvent) error {
	if v.builder == nil {
		return fmt.Errorf("builder not initialized")
	}

	// Convert KeyEvent to string key for WorkflowBuilder
	// This is a simplified conversion - the WorkflowBuilder expects string keys
	keyStr := ""

	if event.IsSpecial {
		// Special keys
		keyStr = event.Special
	} else if event.Ctrl {
		// Ctrl combinations
		keyStr = fmt.Sprintf("Ctrl+%c", event.Key)
	} else if event.Shift && event.IsSpecial {
		// Shift combinations with special keys
		keyStr = "Shift+" + event.Special
	} else {
		// Regular keys
		keyStr = string(event.Key)
	}

	// Handle the key through the workflow builder
	if err := v.builder.HandleKey(keyStr); err != nil {
		v.statusMsg = "Error: " + err.Error()
		return nil // Don't propagate errors, just show in status
	}

	return nil
}

// Render draws the workflow builder to the screen
func (v *WorkflowBuilderView) Render(screen *goterm.Screen) error {
	if v.builder == nil {
		// Render error message if builder not initialized
		fg := goterm.ColorDefault()
		bg := goterm.ColorDefault()
		screen.Clear()
		screen.DrawText(0, 0, "Error: Workflow Builder not initialized", fg, bg, goterm.StyleBold)
		return nil
	}

	// Clear screen
	screen.Clear()

	// Get screen dimensions
	width, height := screen.Size()

	// Update builder dimensions if changed
	if width != v.width || height != v.height {
		v.width = width
		v.height = height
		// Notify builder of size change if needed
		if v.builder.canvas != nil {
			v.builder.canvas.Width = width
			v.builder.canvas.Height = height - 2 // Leave room for status bar
		}
	}

	// Render the workflow builder using the integrated rendering system
	// This will render canvas, panels, palette, etc.
	if err := v.builder.Render(screen, width, height-1); err != nil {
		// If rendering fails, show error message
		screen.DrawText(0, 2, fmt.Sprintf("Render error: %v", err), goterm.ColorRGB(255, 100, 100), goterm.ColorDefault(), goterm.StyleNone)
	}

	// Title bar (drawn on top of everything)
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()
	title := fmt.Sprintf("Workflow Builder: %s [Mode: %s]",
		v.builder.workflow.Name,
		v.builder.mode)
	screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	// Status bar at bottom
	statusLine := fmt.Sprintf("Status: %s | Keys: ? = help, q = quit, Tab = switch view", v.statusMsg)
	if v.builder.modified {
		statusLine += " [modified]"
	}
	screen.DrawText(0, height-1, statusLine, fg, bg, goterm.StyleReverse)

	return nil
}

// IsActive returns whether this view is currently active
func (v *WorkflowBuilderView) IsActive() bool {
	return v.active
}

// SetActive updates the active state of the view
func (v *WorkflowBuilderView) SetActive(active bool) {
	v.active = active
}

// SetWorkflow sets the workflow to be edited
func (v *WorkflowBuilderView) SetWorkflow(workflowPath string) {
	v.workflowPath = workflowPath
	v.initialized = false // force reload on next Init()
}

// SetBounds sets the view dimensions
func (v *WorkflowBuilderView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
