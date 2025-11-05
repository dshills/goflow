package tui

import (
	"github.com/dshills/goterm"
)

// ServerRegistryView manages MCP server configurations
// Users can add, remove, test, and configure MCP servers
type ServerRegistryView struct {
	name        string
	active      bool
	servers     []string // List of registered servers
	selectedIdx int      // Currently selected server index
	statusMsg   string   // Status message to display
	initialized bool
	showDetails bool // Show detailed server info
	width       int  // View width
	height      int  // View height
}

// NewServerRegistryView creates a new server registry view
func NewServerRegistryView() *ServerRegistryView {
	return &ServerRegistryView{
		name:        "registry",
		active:      false,
		servers:     make([]string, 0),
		selectedIdx: 0,
		showDetails: false,
	}
}

// Name returns the unique identifier for this view
func (v *ServerRegistryView) Name() string {
	return v.name
}

// Init initializes the server registry view
func (v *ServerRegistryView) Init() error {
	if v.initialized {
		return nil // already initialized, preserve state
	}

	// TODO: Load servers from storage
	// For now, use placeholder data
	v.servers = []string{
		"[✓] filesystem (active)",
		"[✓] github (active)",
		"[✗] slack (disconnected)",
		"[?] custom-api (not tested)",
	}
	v.selectedIdx = 0
	v.statusMsg = "Ready"
	v.initialized = true

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *ServerRegistryView) Cleanup() error {
	// Preserve state for when we return to this view
	return nil
}

// HandleKey processes keyboard input events
func (v *ServerRegistryView) HandleKey(event KeyEvent) error {
	// TODO: Implement keyboard navigation
	// - j/k: navigate server list
	// - Enter: show server details
	// - a: add new server
	// - d: delete selected server
	// - t: test server connection
	// - r: refresh server status
	// - e: edit server configuration
	// - i: toggle detailed info view

	switch {
	case event.Key == 'j':
		// Move selection down
		if v.selectedIdx < len(v.servers)-1 {
			v.selectedIdx++
		}
	case event.Key == 'k':
		// Move selection up
		if v.selectedIdx > 0 {
			v.selectedIdx--
		}
	case event.Key == 'i':
		// Toggle detailed info
		v.showDetails = !v.showDetails
		if v.showDetails {
			v.statusMsg = "Showing details"
		} else {
			v.statusMsg = "Ready"
		}
	case event.Key == 'a':
		// Add new server
		v.statusMsg = "Add server (not yet implemented)"
	case event.Key == 'd':
		// Delete server
		if len(v.servers) > 0 {
			v.statusMsg = "Delete server (not yet implemented)"
		}
	case event.Key == 't':
		// Test server connection
		if len(v.servers) > 0 {
			v.statusMsg = "Testing connection... (not yet implemented)"
		}
	case event.Key == 'r':
		// Refresh server status
		v.statusMsg = "Refreshing... (not yet implemented)"
	case event.Key == 'e':
		// Edit server configuration
		if len(v.servers) > 0 {
			v.statusMsg = "Edit server (not yet implemented)"
		}
	case event.IsSpecial && event.Special == "Enter":
		// Show server details
		if len(v.servers) > 0 {
			v.showDetails = true
			v.statusMsg = "Viewing server details"
		}
	}

	return nil
}

// Render draws the server registry to the screen
func (v *ServerRegistryView) Render(screen *goterm.Screen) error {
	// TODO: Implement rendering with server status indicators
	// Layout:
	// +----------------------------------+
	// | Server Registry     [Tab: Next]  |
	// +----------------------------------+
	// |                                  |
	// | MCP Servers:                     |
	// | > [✓] filesystem (active)        |
	// |   [✓] github (active)            |
	// |   [✗] slack (disconnected)       |
	// |   [?] custom-api (not tested)    |
	// |                                  |
	// | Or when showing details:         |
	// | Server: filesystem               |
	// | Status: Active                   |
	// | Transport: stdio                 |
	// | Command: mcp-server-filesystem   |
	// | Tools: 5 available               |
	// |                                  |
	// +----------------------------------+
	// | [a: Add] [t: Test] [d: Delete]   |
	// +----------------------------------+

	_, height := screen.Size()
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Clear screen
	screen.Clear()

	// Title bar
	title := "Server Registry [Tab: Switch View] [i: Toggle Details]"
	screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	y := 2

	if v.showDetails && v.selectedIdx < len(v.servers) {
		// Show detailed view for selected server
		screen.DrawText(0, y, "Server Details:", fg, bg, goterm.StyleBold)
		y += 2

		// TODO: Load actual server details
		details := []string{
			"Name: filesystem",
			"Status: Active",
			"Transport: stdio",
			"Command: mcp-server-filesystem",
			"Tools: 5 available",
			"",
			"Available Tools:",
			"  - read_file",
			"  - write_file",
			"  - list_directory",
			"  - search_files",
			"  - file_info",
		}

		for _, line := range details {
			if y >= height-1 {
				break
			}
			screen.DrawText(0, y, "  "+line, fg, bg, goterm.StyleNone)
			y++
		}
	} else {
		// Show server list
		screen.DrawText(0, y, "MCP Servers:", fg, bg, goterm.StyleBold)
		y++

		if len(v.servers) == 0 {
			screen.DrawText(0, y+1, "  No servers registered", fg, bg, goterm.StyleDim)
			screen.DrawText(0, y+2, "  Press 'a' to add a server", fg, bg, goterm.StyleDim)
		} else {
			for i, server := range v.servers {
				if y >= height-1 {
					break
				}

				prefix := "  "
				style := goterm.StyleNone
				if i == v.selectedIdx {
					prefix = "> "
					style = goterm.StyleReverse
				}

				screen.DrawText(0, y, prefix+server, fg, bg, style)
				y++
			}
		}
	}

	// Status bar (bottom line)
	helpText := "[a: Add] [t: Test] [d: Delete] [e: Edit]"
	statusLine := helpText + "    " + v.statusMsg
	screen.DrawText(0, height-1, statusLine, fg, bg, goterm.StyleNone)

	return nil
}

// IsActive returns whether this view is currently active
func (v *ServerRegistryView) IsActive() bool {
	return v.active
}

// SetActive updates the active state of the view
func (v *ServerRegistryView) SetActive(active bool) {
	v.active = active
}

// RefreshServers reloads the server list from storage
func (v *ServerRegistryView) RefreshServers() error {
	// TODO: Implement actual server loading
	v.initialized = false
	return v.Init()
}

// SetBounds sets the view dimensions
func (v *ServerRegistryView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
