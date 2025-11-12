package tui

import (
	"github.com/dshills/goterm"
)

// WorkflowExplorerView displays a list of available workflows
// Users can browse, search, and select workflows to edit or execute
type WorkflowExplorerView struct {
	name        string
	active      bool
	workflows   []string // List of workflow names
	selectedIdx int      // Currently selected workflow index
	statusMsg   string   // Status message to display
	initialized bool
	width       int // View width
	height      int // View height
}

// NewWorkflowExplorerView creates a new workflow explorer view
func NewWorkflowExplorerView() *WorkflowExplorerView {
	return &WorkflowExplorerView{
		name:        "explorer",
		active:      false,
		workflows:   make([]string, 0),
		selectedIdx: 0,
	}
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

	// TODO: Load workflows from storage
	// For now, use placeholder data
	v.workflows = []string{
		"example-workflow.yaml",
		"data-pipeline.yaml",
		"api-integration.yaml",
	}
	v.selectedIdx = 0
	v.statusMsg = "Ready"
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
	// TODO: Implement keyboard navigation
	// - j/k: move selection up/down
	// - Enter: open selected workflow in builder
	// - /: start search
	// - n: create new workflow
	// - d: delete selected workflow
	// - r: rename selected workflow

	switch {
	case event.Key == 'j':
		// Move selection down
		if v.selectedIdx < len(v.workflows)-1 {
			v.selectedIdx++
		}
	case event.Key == 'k':
		// Move selection up
		if v.selectedIdx > 0 {
			v.selectedIdx--
		}
	case event.IsSpecial && event.Special == "Enter":
		// Open selected workflow
		v.statusMsg = "Opening workflow (not yet implemented)"
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
