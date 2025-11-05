package tui

import (
	"github.com/dshills/goterm"
)

// WorkflowBuilderView provides a visual workflow editor
// Users can add nodes, create edges, and configure workflow properties
type WorkflowBuilderView struct {
	name        string
	active      bool
	workflowID  string   // Current workflow being edited
	nodes       []string // List of node IDs
	edges       []string // List of edge descriptions
	selectedIdx int      // Currently selected item
	mode        string   // "normal", "insert", "visual", "command"
	statusMsg   string   // Status message to display
	initialized bool
	width       int // View width
	height      int // View height
}

// NewWorkflowBuilderView creates a new workflow builder view
func NewWorkflowBuilderView() *WorkflowBuilderView {
	return &WorkflowBuilderView{
		name:        "builder",
		active:      false,
		nodes:       make([]string, 0),
		edges:       make([]string, 0),
		selectedIdx: 0,
		mode:        "normal",
	}
}

// Name returns the unique identifier for this view
func (v *WorkflowBuilderView) Name() string {
	return v.name
}

// Init initializes the workflow builder view
func (v *WorkflowBuilderView) Init() error {
	if v.initialized {
		return nil // already initialized, preserve state
	}

	// TODO: Load current workflow from storage
	// For now, use placeholder data
	v.nodes = []string{
		"[start] Start",
		"[tool] Fetch Data",
		"[transform] Process",
		"[end] End",
	}
	v.edges = []string{
		"start -> fetch_data",
		"fetch_data -> process",
		"process -> end",
	}
	v.selectedIdx = 0
	v.mode = "normal"
	v.statusMsg = "Ready"
	v.initialized = true

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *WorkflowBuilderView) Cleanup() error {
	// Preserve state for when we return to this view
	return nil
}

// HandleKey processes keyboard input events
func (v *WorkflowBuilderView) HandleKey(event KeyEvent) error {
	// TODO: Implement vim-style keyboard navigation
	// Normal mode:
	// - h/j/k/l: navigate nodes
	// - a: add node
	// - e: create edge
	// - d: delete node/edge
	// - r: rename node
	// - i: enter insert mode
	// - v: enter visual mode
	// - :: enter command mode
	// - ?: toggle help
	//
	// Insert mode:
	// - Escape: return to normal mode
	// - Characters: insert text
	//
	// Visual mode:
	// - Escape: return to normal mode
	// - Movement: select multiple items
	//
	// Command mode:
	// - Escape: return to normal mode
	// - Enter: execute command
	// - :w: save workflow
	// - :q: quit builder

	// Basic navigation for now
	switch {
	case event.Key == 'j' && v.mode == "normal":
		// Move selection down
		totalItems := len(v.nodes)
		if v.selectedIdx < totalItems-1 {
			v.selectedIdx++
		}
	case event.Key == 'k' && v.mode == "normal":
		// Move selection up
		if v.selectedIdx > 0 {
			v.selectedIdx--
		}
	case event.Key == 'i' && v.mode == "normal":
		// Enter insert mode
		v.mode = "insert"
		v.statusMsg = "-- INSERT --"
	case event.IsSpecial && event.Special == "Escape":
		// Return to normal mode
		if v.mode != "normal" {
			v.mode = "normal"
			v.statusMsg = "Ready"
		}
	case event.Key == 'a' && v.mode == "normal":
		// Add node
		v.statusMsg = "Add node (not yet implemented)"
	case event.Key == 'e' && v.mode == "normal":
		// Create edge
		v.statusMsg = "Create edge (not yet implemented)"
	case event.Key == 'd' && v.mode == "normal":
		// Delete item
		v.statusMsg = "Delete item (not yet implemented)"
	}

	return nil
}

// Render draws the workflow builder to the screen
func (v *WorkflowBuilderView) Render(screen *goterm.Screen) error {
	// TODO: Implement rendering with visual workflow representation
	// Layout:
	// +----------------------------------+
	// | Workflow Builder    [Tab: Next]  |
	// +----------------------------------+
	// |                                  |
	// | Nodes:                           |
	// | > [start] Start                  |
	// |   [tool] Fetch Data              |
	// |   [transform] Process            |
	// |   [end] End                      |
	// |                                  |
	// | Edges:                           |
	// |   start -> fetch_data            |
	// |   fetch_data -> process          |
	// |   process -> end                 |
	// |                                  |
	// +----------------------------------+
	// | Mode: NORMAL    Status: Ready    |
	// +----------------------------------+

	_, height := screen.Size()
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Clear screen
	screen.Clear()

	// Title bar
	title := "Workflow Builder [Tab: Switch View] [?: Help]"
	screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	// Nodes section
	y := 2
	screen.DrawText(0, y, "Nodes:", fg, bg, goterm.StyleBold)
	y++

	for i, node := range v.nodes {
		if y >= height-5 {
			break // leave room for edges and status
		}

		prefix := "  "
		style := goterm.StyleNone
		if i == v.selectedIdx {
			prefix = "> "
			style = goterm.StyleReverse
		}

		screen.DrawText(0, y, prefix+node, fg, bg, style)
		y++
	}

	// Edges section
	y++
	if y < height-2 {
		screen.DrawText(0, y, "Edges:", fg, bg, goterm.StyleBold)
		y++

		for _, edge := range v.edges {
			if y >= height-1 {
				break
			}
			screen.DrawText(0, y, "  "+edge, fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Status bar (bottom line)
	modeStr := "Mode: " + v.mode
	statusLine := modeStr + "    Status: " + v.statusMsg
	screen.DrawText(0, height-1, statusLine, fg, bg, goterm.StyleNone)

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
func (v *WorkflowBuilderView) SetWorkflow(workflowID string) {
	v.workflowID = workflowID
	v.initialized = false // force reload on next Init()
}

// SetBounds sets the view dimensions
func (v *WorkflowBuilderView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
