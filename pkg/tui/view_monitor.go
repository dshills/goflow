package tui

import (
	"github.com/dshills/goterm"
)

// ExecutionMonitorView displays real-time workflow execution status
// Shows node execution progress, logs, and error information
type ExecutionMonitorView struct {
	name        string
	active      bool
	executionID string   // Current execution being monitored
	nodes       []string // Execution nodes with status
	logs        []string // Execution log entries
	selectedIdx int      // Currently selected item
	autoScroll  bool     // Auto-scroll to latest log entry
	statusMsg   string   // Status message to display
	initialized bool
	showLogs    bool // Toggle between node view and log view
	width       int  // View width
	height      int  // View height
}

// NewExecutionMonitorView creates a new execution monitor view
func NewExecutionMonitorView() *ExecutionMonitorView {
	return &ExecutionMonitorView{
		name:        "monitor",
		active:      false,
		nodes:       make([]string, 0),
		logs:        make([]string, 0),
		selectedIdx: 0,
		autoScroll:  true,
		showLogs:    false,
	}
}

// Name returns the unique identifier for this view
func (v *ExecutionMonitorView) Name() string {
	return v.name
}

// Init initializes the execution monitor view
func (v *ExecutionMonitorView) Init() error {
	if v.initialized {
		return nil // already initialized, preserve state
	}

	// TODO: Load execution data from storage
	// For now, use placeholder data
	v.nodes = []string{
		"[✓] Start (completed)",
		"[→] Fetch Data (running)",
		"[ ] Process (pending)",
		"[ ] End (pending)",
	}
	v.logs = []string{
		"[12:00:00] Execution started",
		"[12:00:01] Start node completed",
		"[12:00:02] Fetch Data node started",
		"[12:00:03] Connecting to API...",
		"[12:00:04] Fetching data...",
	}
	v.selectedIdx = 0
	v.statusMsg = "Monitoring execution"
	v.initialized = true

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *ExecutionMonitorView) Cleanup() error {
	// Preserve state for when we return to this view
	return nil
}

// HandleKey processes keyboard input events
func (v *ExecutionMonitorView) HandleKey(event KeyEvent) error {
	// TODO: Implement keyboard navigation
	// - j/k: scroll through nodes or logs
	// - l: toggle log view
	// - a: toggle auto-scroll
	// - Enter: view detailed node execution info
	// - /: search logs
	// - r: refresh execution status

	switch {
	case event.Key == 'j':
		// Scroll down
		maxIdx := len(v.nodes) - 1
		if v.showLogs {
			maxIdx = len(v.logs) - 1
		}
		if v.selectedIdx < maxIdx {
			v.selectedIdx++
		}
	case event.Key == 'k':
		// Scroll up
		if v.selectedIdx > 0 {
			v.selectedIdx--
		}
	case event.Key == 'l':
		// Toggle log view
		v.showLogs = !v.showLogs
		v.selectedIdx = 0
		if v.showLogs {
			v.statusMsg = "Viewing logs"
		} else {
			v.statusMsg = "Viewing nodes"
		}
	case event.Key == 'a':
		// Toggle auto-scroll
		v.autoScroll = !v.autoScroll
		if v.autoScroll {
			v.statusMsg = "Auto-scroll enabled"
		} else {
			v.statusMsg = "Auto-scroll disabled"
		}
	case event.Key == 'r':
		// Refresh execution status
		v.statusMsg = "Refreshing... (not yet implemented)"
	case event.IsSpecial && event.Special == "Enter":
		// View detailed info
		v.statusMsg = "Detailed view (not yet implemented)"
	}

	return nil
}

// Render draws the execution monitor to the screen
func (v *ExecutionMonitorView) Render(screen *goterm.Screen) error {
	// TODO: Implement rendering with real-time updates
	// Layout:
	// +----------------------------------+
	// | Execution Monitor   [Tab: Next]  |
	// +----------------------------------+
	// |                                  |
	// | Execution: exec-123              |
	// |                                  |
	// | Nodes:                           |
	// | > [✓] Start (completed)          |
	// |   [→] Fetch Data (running)       |
	// |   [ ] Process (pending)          |
	// |   [ ] End (pending)              |
	// |                                  |
	// | Or when showing logs:            |
	// | > [12:00:00] Execution started   |
	// |   [12:00:01] Start completed     |
	// |   [12:00:02] Fetch Data started  |
	// |                                  |
	// +----------------------------------+
	// | AutoScroll: ON  [l: toggle logs] |
	// +----------------------------------+

	_, height := screen.Size()
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Clear screen
	screen.Clear()

	// Title bar
	title := "Execution Monitor [Tab: Switch View] [l: Toggle Logs]"
	screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	// Execution ID
	y := 2
	if v.executionID != "" {
		screen.DrawText(0, y, "Execution: "+v.executionID, fg, bg, goterm.StyleNone)
	} else {
		screen.DrawText(0, y, "No active execution", fg, bg, goterm.StyleDim)
	}
	y += 2

	// Content based on mode
	if v.showLogs {
		screen.DrawText(0, y, "Logs:", fg, bg, goterm.StyleBold)
		y++

		startIdx := 0
		if v.autoScroll && len(v.logs) > height-6 {
			startIdx = len(v.logs) - (height - 6)
		}

		for i := startIdx; i < len(v.logs); i++ {
			if y >= height-1 {
				break
			}

			prefix := "  "
			style := goterm.StyleNone
			if i == v.selectedIdx {
				prefix = "> "
				style = goterm.StyleReverse
			}

			screen.DrawText(0, y, prefix+v.logs[i], fg, bg, style)
			y++
		}
	} else {
		screen.DrawText(0, y, "Nodes:", fg, bg, goterm.StyleBold)
		y++

		for i, node := range v.nodes {
			if y >= height-1 {
				break
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
	}

	// Status bar (bottom line)
	scrollStatus := "AutoScroll: OFF"
	if v.autoScroll {
		scrollStatus = "AutoScroll: ON"
	}
	statusLine := scrollStatus + "    " + v.statusMsg
	screen.DrawText(0, height-1, statusLine, fg, bg, goterm.StyleNone)

	return nil
}

// IsActive returns whether this view is currently active
func (v *ExecutionMonitorView) IsActive() bool {
	return v.active
}

// SetActive updates the active state of the view
func (v *ExecutionMonitorView) SetActive(active bool) {
	v.active = active
}

// SetExecution sets the execution to be monitored
func (v *ExecutionMonitorView) SetExecution(executionID string) {
	v.executionID = executionID
	v.initialized = false // force reload on next Init()
}

// SetBounds sets the view dimensions
func (v *ExecutionMonitorView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
