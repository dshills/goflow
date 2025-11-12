package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/dshills/goflow/pkg/tui/components"
	"github.com/dshills/goterm"
)

// ServerRegistryView manages MCP server configurations
// T196: Server registry view with list, navigation, and actions
// T197: Server add dialog for registering new servers
// T198: Health status display with real-time updates
// T199: Tool schema viewer for server capabilities
type ServerRegistryView struct {
	name           string
	active         bool
	registry       mcpserver.ServerRepository
	servers        []*mcpserver.MCPServer
	selectedIdx    int
	statusMsg      string
	initialized    bool
	showDetails    bool              // T199: Show detailed server info and tools
	showToolSchema bool              // T199: Show tool schema details
	selectedTool   int               // T199: Selected tool index in schema view
	currentModal   *components.Modal // T197: Modal for add/edit dialogs
	addDialogState *addServerDialogState
	autoRefresh    bool      // T198: Auto-refresh health status
	lastRefresh    time.Time // T198: Last health check time
	errorMsg       string    // Error message display
	width          int
	height         int
}

// addServerDialogState tracks the state of the add server dialog (T197)
type addServerDialogState struct {
	step          int // 0=ID, 1=name, 2=transport, 3=transport-specific fields
	serverID      string
	serverName    string
	transportType mcpserver.TransportType
	command       string // For stdio
	args          string // For stdio (comma-separated)
	url           string // For SSE/HTTP
	headers       string // For SSE/HTTP (key:value pairs, comma-separated)
	currentField  string // Current field being edited
}

// NewServerRegistryView creates a new server registry view
func NewServerRegistryView() *ServerRegistryView {
	return &ServerRegistryView{
		name:           "registry",
		active:         false,
		registry:       mcpserver.NewRegistry(), // Default in-memory registry
		servers:        make([]*mcpserver.MCPServer, 0),
		selectedIdx:    0,
		showDetails:    false,
		showToolSchema: false,
		selectedTool:   0,
		autoRefresh:    true, // T198: Enable auto-refresh by default
		lastRefresh:    time.Time{},
	}
}

// SetRegistry sets the server repository to use
func (v *ServerRegistryView) SetRegistry(registry mcpserver.ServerRepository) {
	v.registry = registry
}

// Name returns the unique identifier for this view
func (v *ServerRegistryView) Name() string {
	return v.name
}

// Init initializes the server registry view
func (v *ServerRegistryView) Init() error {
	if v.initialized {
		// Refresh server list
		return v.loadServers()
	}

	// Load servers from repository
	if err := v.loadServers(); err != nil {
		v.statusMsg = fmt.Sprintf("Error loading servers: %v", err)
		// Don't fail initialization, just show error
	}

	v.selectedIdx = 0
	v.statusMsg = "Ready"
	v.initialized = true
	v.lastRefresh = time.Now()

	return nil
}

// loadServers loads servers from the registry
func (v *ServerRegistryView) loadServers() error {
	if v.registry == nil {
		return fmt.Errorf("no registry configured")
	}

	servers, err := v.registry.List()
	if err != nil {
		return err
	}

	v.servers = servers

	// Adjust selected index if needed
	if v.selectedIdx >= len(v.servers) && len(v.servers) > 0 {
		v.selectedIdx = len(v.servers) - 1
	} else if len(v.servers) == 0 {
		v.selectedIdx = 0
	}

	return nil
}

// Cleanup releases resources when view is deactivated
func (v *ServerRegistryView) Cleanup() error {
	// Preserve state for when we return to this view
	return nil
}

// HandleKey processes keyboard input events
func (v *ServerRegistryView) HandleKey(event KeyEvent) error {
	// Handle modal input first
	if v.currentModal != nil && v.currentModal.IsVisible() {
		keyStr := v.keyEventToString(event)
		v.currentModal.HandleKey(keyStr)
		return nil
	}

	// Tool schema view navigation (T199)
	if v.showToolSchema {
		return v.handleToolSchemaKeys(event)
	}

	// Normal view navigation
	switch {
	case event.Key == 'j' || (event.IsSpecial && event.Special == "Down"):
		// Move selection down
		if len(v.servers) > 0 && v.selectedIdx < len(v.servers)-1 {
			v.selectedIdx++
			v.showDetails = false // Reset details view
			v.showToolSchema = false
		}
	case event.Key == 'k' || (event.IsSpecial && event.Special == "Up"):
		// Move selection up
		if len(v.servers) > 0 && v.selectedIdx > 0 {
			v.selectedIdx--
			v.showDetails = false // Reset details view
			v.showToolSchema = false
		}
	case event.Key == 'g':
		// Go to top
		if len(v.servers) > 0 {
			v.selectedIdx = 0
			v.showDetails = false
			v.showToolSchema = false
		}
	case event.Key == 'G':
		// Go to bottom
		if len(v.servers) > 0 {
			v.selectedIdx = len(v.servers) - 1
			v.showDetails = false
			v.showToolSchema = false
		}
	case event.IsSpecial && event.Special == "Enter":
		// Toggle detailed info view (T198, T199)
		if len(v.servers) > 0 {
			v.showDetails = !v.showDetails
			v.showToolSchema = false
			v.selectedTool = 0
			if v.showDetails {
				v.statusMsg = "Viewing server details"
			} else {
				v.statusMsg = "Ready"
			}
		}
	case event.Key == 'i':
		// Toggle detailed info view (T198, T199)
		if len(v.servers) > 0 {
			v.showDetails = !v.showDetails
			v.showToolSchema = false
			v.selectedTool = 0
			if v.showDetails {
				v.statusMsg = "Viewing server details"
			} else {
				v.statusMsg = "Ready"
			}
		}
	case event.Key == 's':
		// T199: Show tool schema for selected server
		if len(v.servers) > 0 {
			v.showToolSchema = !v.showToolSchema
			v.showDetails = false
			v.selectedTool = 0
			if v.showToolSchema {
				v.statusMsg = "Viewing tool schemas (j/k: navigate, Enter: details, Esc: back)"
			} else {
				v.statusMsg = "Ready"
			}
		}
	case event.Key == 'a':
		// T197: Add new server dialog
		v.showAddServerDialog()
	case event.Key == 'd':
		// Delete server
		if len(v.servers) > 0 {
			v.showDeleteConfirmation()
		}
	case event.Key == 't':
		// T198: Test server connection (connect and health check)
		if len(v.servers) > 0 {
			v.testServerConnection()
		}
	case event.Key == 'r':
		// T198: Refresh server status (manual health check)
		v.refreshServerStatus()
	case event.Key == 'c':
		// Connect to server
		if len(v.servers) > 0 {
			v.connectServer()
		}
	case event.Key == 'x':
		// Disconnect from server
		if len(v.servers) > 0 {
			v.disconnectServer()
		}
	case event.Key == 'R':
		// T198: Toggle auto-refresh
		v.autoRefresh = !v.autoRefresh
		if v.autoRefresh {
			v.statusMsg = "Auto-refresh enabled"
			v.lastRefresh = time.Now()
		} else {
			v.statusMsg = "Auto-refresh disabled"
		}
	case event.Key == '?':
		// Show help
		v.showHelp()
	case event.IsSpecial && event.Special == "Escape":
		// Exit details/schema view
		if v.showDetails || v.showToolSchema {
			v.showDetails = false
			v.showToolSchema = false
			v.selectedTool = 0
			v.statusMsg = "Ready"
		}
	}

	return nil
}

// handleToolSchemaKeys handles keyboard input in tool schema view (T199)
func (v *ServerRegistryView) handleToolSchemaKeys(event KeyEvent) error {
	if v.selectedIdx >= len(v.servers) {
		return nil
	}

	server := v.servers[v.selectedIdx]
	toolCount := len(server.Tools)

	switch {
	case event.Key == 'j' || (event.IsSpecial && event.Special == "Down"):
		if v.selectedTool < toolCount-1 {
			v.selectedTool++
		}
	case event.Key == 'k' || (event.IsSpecial && event.Special == "Up"):
		if v.selectedTool > 0 {
			v.selectedTool--
		}
	case event.Key == 'g':
		v.selectedTool = 0
	case event.Key == 'G':
		if toolCount > 0 {
			v.selectedTool = toolCount - 1
		}
	case event.IsSpecial && event.Special == "Escape":
		v.showToolSchema = false
		v.selectedTool = 0
		v.statusMsg = "Ready"
	case event.IsSpecial && event.Special == "Enter":
		// TODO: Copy tool name to clipboard for workflow building
		if v.selectedTool < toolCount {
			toolName := server.Tools[v.selectedTool].Name
			v.statusMsg = fmt.Sprintf("Selected: %s", toolName)
		}
	}

	return nil
}

// keyEventToString converts KeyEvent to string for modal handling
func (v *ServerRegistryView) keyEventToString(event KeyEvent) string {
	if event.IsSpecial {
		return event.Special
	}
	if event.Ctrl {
		return fmt.Sprintf("Ctrl-%c", event.Key)
	}
	return string(event.Key)
}

// showAddServerDialog shows the add server dialog (T197)
func (v *ServerRegistryView) showAddServerDialog() {
	v.addDialogState = &addServerDialogState{
		step:          0,
		transportType: mcpserver.TransportStdio, // Default
		currentField:  "Server ID",
	}

	modal := components.NewInputModal(
		"Add MCP Server - Step 1/4",
		"Enter server ID (unique identifier):",
		"",
		func(confirmed bool, input string) {
			if confirmed && input != "" {
				v.addDialogState.serverID = input
				v.showAddServerDialogStep2()
			} else {
				v.currentModal = nil
				v.addDialogState = nil
				v.statusMsg = "Cancelled"
			}
		},
	)

	v.currentModal = modal
	modal.Show()
}

// showAddServerDialogStep2 shows step 2 of add dialog (server name)
func (v *ServerRegistryView) showAddServerDialogStep2() {
	modal := components.NewInputModal(
		"Add MCP Server - Step 2/4",
		"Enter server name (display name):",
		v.addDialogState.serverID, // Default to ID
		func(confirmed bool, input string) {
			if confirmed && input != "" {
				v.addDialogState.serverName = input
				v.showAddServerDialogStep3()
			} else {
				v.currentModal = nil
				v.addDialogState = nil
				v.statusMsg = "Cancelled"
			}
		},
	)

	v.currentModal = modal
	modal.Show()
}

// showAddServerDialogStep3 shows step 3 of add dialog (transport type)
func (v *ServerRegistryView) showAddServerDialogStep3() {
	modal := components.NewInputModal(
		"Add MCP Server - Step 3/4",
		"Enter transport type (stdio, sse, or http):",
		"stdio",
		func(confirmed bool, input string) {
			if !confirmed {
				v.currentModal = nil
				v.addDialogState = nil
				v.statusMsg = "Cancelled"
				return
			}

			transportType := mcpserver.TransportType(strings.ToLower(input))
			if !transportType.IsValid() {
				v.statusMsg = "Invalid transport type (use: stdio, sse, or http)"
				v.currentModal = nil
				v.addDialogState = nil
				return
			}

			v.addDialogState.transportType = transportType
			v.showAddServerDialogStep4()
		},
	)

	v.currentModal = modal
	modal.Show()
}

// showAddServerDialogStep4 shows step 4 of add dialog (transport config)
func (v *ServerRegistryView) showAddServerDialogStep4() {
	var prompt string
	var defaultVal string

	switch v.addDialogState.transportType {
	case mcpserver.TransportStdio:
		prompt = "Enter command (e.g., 'mcp-server-filesystem'):\nOptional args can be added after, separated by commas"
		defaultVal = ""
	case mcpserver.TransportSSE:
		prompt = "Enter SSE URL (e.g., 'http://localhost:3000/sse'):"
		defaultVal = ""
	case mcpserver.TransportHTTP:
		prompt = "Enter HTTP base URL (e.g., 'http://localhost:3000'):"
		defaultVal = ""
	}

	modal := components.NewInputModal(
		"Add MCP Server - Step 4/4",
		prompt,
		defaultVal,
		func(confirmed bool, input string) {
			if !confirmed || input == "" {
				v.currentModal = nil
				v.addDialogState = nil
				v.statusMsg = "Cancelled"
				return
			}

			// Parse input based on transport type
			switch v.addDialogState.transportType {
			case mcpserver.TransportStdio:
				parts := strings.Split(input, ",")
				v.addDialogState.command = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					v.addDialogState.args = strings.Join(parts[1:], ",")
				}
			case mcpserver.TransportSSE:
				v.addDialogState.url = strings.TrimSpace(input)
			case mcpserver.TransportHTTP:
				v.addDialogState.url = strings.TrimSpace(input)
			}

			// Create the server
			v.createServerFromDialog()
		},
	)

	v.currentModal = modal
	modal.Show()
}

// createServerFromDialog creates and registers a server from dialog state
func (v *ServerRegistryView) createServerFromDialog() {
	state := v.addDialogState

	var args []string
	if state.args != "" {
		argParts := strings.Split(state.args, ",")
		for _, arg := range argParts {
			trimmed := strings.TrimSpace(arg)
			if trimmed != "" {
				args = append(args, trimmed)
			}
		}
	}

	// For SSE/HTTP, command is the URL
	command := state.command
	if state.transportType == mcpserver.TransportSSE || state.transportType == mcpserver.TransportHTTP {
		command = state.url
	}

	// Create server
	server, err := mcpserver.NewMCPServer(state.serverID, command, args, state.transportType)
	if err != nil {
		v.statusMsg = fmt.Sprintf("Error creating server: %v", err)
		v.errorMsg = err.Error()
		v.currentModal = nil
		v.addDialogState = nil
		return
	}

	// Set server name if different from ID
	if state.serverName != "" && state.serverName != state.serverID {
		server.Name = state.serverName
	}

	// Register server
	if err := v.registry.Register(server); err != nil {
		v.statusMsg = fmt.Sprintf("Error registering server: %v", err)
		v.errorMsg = err.Error()
		v.currentModal = nil
		v.addDialogState = nil
		return
	}

	// Reload servers
	v.loadServers()

	// Select the new server
	for i, s := range v.servers {
		if s.ID == state.serverID {
			v.selectedIdx = i
			break
		}
	}

	v.statusMsg = fmt.Sprintf("Server '%s' added successfully", state.serverID)
	v.currentModal = nil
	v.addDialogState = nil
}

// showDeleteConfirmation shows delete confirmation dialog
func (v *ServerRegistryView) showDeleteConfirmation() {
	if v.selectedIdx >= len(v.servers) {
		return
	}

	server := v.servers[v.selectedIdx]
	modal := components.NewConfirmModal(
		"Delete Server",
		fmt.Sprintf("Delete server '%s'?\nThis will remove the server configuration.", server.Name),
		func(confirmed bool) {
			if confirmed {
				v.deleteSelectedServer()
			}
			v.currentModal = nil
		},
	)

	v.currentModal = modal
	modal.Show()
}

// deleteSelectedServer deletes the currently selected server
func (v *ServerRegistryView) deleteSelectedServer() {
	if v.selectedIdx >= len(v.servers) {
		return
	}

	server := v.servers[v.selectedIdx]

	// Disconnect if connected
	if server.Connection.GetState() == mcpserver.StateConnected {
		server.Disconnect()
	}

	// Unregister from registry
	if err := v.registry.Unregister(server.ID); err != nil {
		v.statusMsg = fmt.Sprintf("Error deleting server: %v", err)
		v.errorMsg = err.Error()
		return
	}

	v.statusMsg = fmt.Sprintf("Server '%s' deleted", server.Name)
	v.loadServers()
}

// testServerConnection tests the selected server connection (T198)
func (v *ServerRegistryView) testServerConnection() {
	if v.selectedIdx >= len(v.servers) {
		return
	}

	server := v.servers[v.selectedIdx]
	v.statusMsg = fmt.Sprintf("Testing connection to '%s'...", server.Name)

	// Connect if not connected
	if server.Connection.GetState() != mcpserver.StateConnected {
		if err := server.Connect(); err != nil {
			v.statusMsg = fmt.Sprintf("Connection failed: %v", err)
			v.errorMsg = err.Error()
			return
		}

		// Simulate connection completion (in real implementation, would wait for async completion)
		if err := server.CompleteConnection(); err != nil {
			v.statusMsg = fmt.Sprintf("Connection failed: %v", err)
			v.errorMsg = err.Error()
			server.FailConnection(err.Error())
			return
		}
	}

	// Perform health check
	if err := server.HealthCheck(); err != nil {
		v.statusMsg = fmt.Sprintf("Health check failed: %v", err)
		v.errorMsg = err.Error()
		return
	}

	// Discover tools if not already done
	if len(server.Tools) == 0 {
		if err := server.DiscoverTools(); err != nil {
			v.statusMsg = fmt.Sprintf("Tool discovery failed: %v", err)
			v.errorMsg = err.Error()
			return
		}
	}

	v.statusMsg = fmt.Sprintf("Connection test successful - %d tools available", len(server.Tools))
	v.lastRefresh = time.Now()
}

// refreshServerStatus refreshes health status for all servers (T198)
func (v *ServerRegistryView) refreshServerStatus() {
	v.statusMsg = "Refreshing server status..."

	healthyCount := 0
	errorCount := 0

	for _, server := range v.servers {
		// Only check health for connected servers
		if server.Connection.GetState() == mcpserver.StateConnected {
			if err := server.HealthCheck(); err != nil {
				errorCount++
			} else {
				healthyCount++
			}
		}
	}

	v.lastRefresh = time.Now()

	if errorCount > 0 {
		v.statusMsg = fmt.Sprintf("Refreshed - %d healthy, %d errors", healthyCount, errorCount)
	} else {
		v.statusMsg = fmt.Sprintf("Refreshed - %d servers healthy", healthyCount)
	}
}

// connectServer connects the selected server
func (v *ServerRegistryView) connectServer() {
	if v.selectedIdx >= len(v.servers) {
		return
	}

	server := v.servers[v.selectedIdx]

	if server.Connection.GetState() == mcpserver.StateConnected {
		v.statusMsg = "Server already connected"
		return
	}

	if err := server.Connect(); err != nil {
		v.statusMsg = fmt.Sprintf("Connect failed: %v", err)
		v.errorMsg = err.Error()
		return
	}

	// Simulate connection completion
	if err := server.CompleteConnection(); err != nil {
		v.statusMsg = fmt.Sprintf("Connect failed: %v", err)
		v.errorMsg = err.Error()
		server.FailConnection(err.Error())
		return
	}

	v.statusMsg = fmt.Sprintf("Connected to '%s'", server.Name)
}

// disconnectServer disconnects the selected server
func (v *ServerRegistryView) disconnectServer() {
	if v.selectedIdx >= len(v.servers) {
		return
	}

	server := v.servers[v.selectedIdx]

	if server.Connection.GetState() != mcpserver.StateConnected {
		v.statusMsg = "Server not connected"
		return
	}

	if err := server.Disconnect(); err != nil {
		v.statusMsg = fmt.Sprintf("Disconnect failed: %v", err)
		v.errorMsg = err.Error()
		return
	}

	v.statusMsg = fmt.Sprintf("Disconnected from '%s'", server.Name)
}

// showHelp shows the help modal
func (v *ServerRegistryView) showHelp() {
	helpText := `Server Registry Help

Navigation:
  j/k       Move up/down
  g/G       Go to top/bottom
  Enter/i   Toggle server details
  s         View tool schemas
  Esc       Exit details/schema view

Server Management:
  a         Add new server
  d         Delete selected server
  t         Test server connection
  c         Connect to server
  x         Disconnect from server

Health Status:
  r         Refresh server status
  R         Toggle auto-refresh

General:
  Tab       Switch to next view
  ?         Show this help
  q         Quit application`

	modal := components.NewInfoModal(
		"Help - Server Registry",
		helpText,
		func() {
			v.currentModal = nil
		},
	)

	v.currentModal = modal
	modal.Show()
}

// Render draws the server registry to the screen
func (v *ServerRegistryView) Render(screen *goterm.Screen) error {
	if screen == nil {
		return fmt.Errorf("screen is nil")
	}

	width, height := screen.Size()
	v.width = width
	v.height = height

	// Auto-refresh health status (T198)
	if v.autoRefresh && time.Since(v.lastRefresh) > 10*time.Second {
		// Perform background health check for connected servers
		for _, server := range v.servers {
			if server.Connection.GetState() == mcpserver.StateConnected {
				// Non-blocking health check
				go func(s *mcpserver.MCPServer) {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()

					// Simple ping check
					select {
					case <-ctx.Done():
						s.RecordUnhealthy("ping timeout")
					default:
						s.HealthCheck()
					}
				}(server)
			}
		}
		v.lastRefresh = time.Now()
	}

	// Clear screen
	screen.Clear()

	fg := goterm.ColorRGB(220, 220, 220)
	bg := goterm.ColorDefault()

	// Title bar
	title := "Server Registry"
	helpLine := "[j/k: Navigate] [i: Details] [s: Tools] [a: Add] [d: Delete] [t: Test] [r: Refresh] [?: Help]"

	// Draw title
	for i, ch := range title {
		if i >= width {
			break
		}
		screen.SetCell(i, 0, goterm.NewCell(ch, fg, bg, goterm.StyleBold))
	}

	// Draw help line (abbreviated if needed)
	helpX := len(title) + 2
	if width-helpX > len(helpLine) {
		for i, ch := range helpLine {
			x := helpX + i
			if x < width {
				screen.SetCell(x, 0, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
			}
		}
	}

	y := 2

	// Render based on current mode
	if v.showToolSchema && v.selectedIdx < len(v.servers) {
		// T199: Tool schema viewer
		y = v.renderToolSchemaView(screen, y)
	} else if v.showDetails && v.selectedIdx < len(v.servers) {
		// T198: Server details view
		y = v.renderServerDetailsView(screen, y)
	} else {
		// T196: Server list view
		y = v.renderServerListView(screen, y)
	}

	// Status bar (bottom)
	statusY := height - 1

	// Auto-refresh indicator (T198)
	refreshIndicator := ""
	if v.autoRefresh {
		refreshIndicator = " [Auto-refresh: ON]"
	}

	statusLine := v.statusMsg + refreshIndicator

	// Show last refresh time if auto-refresh is on
	if v.autoRefresh && !v.lastRefresh.IsZero() {
		timeSince := time.Since(v.lastRefresh)
		statusLine += fmt.Sprintf(" (Last: %ds ago)", int(timeSince.Seconds()))
	}

	// Truncate if too long
	if len(statusLine) > width {
		statusLine = statusLine[:width]
	}

	for i, ch := range statusLine {
		if i >= width {
			break
		}
		screen.SetCell(i, statusY, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
	}

	// Render modal if visible
	if v.currentModal != nil && v.currentModal.IsVisible() {
		v.currentModal.Render(screen)
	}

	return nil
}

// renderServerListView renders the main server list (T196)
func (v *ServerRegistryView) renderServerListView(screen *goterm.Screen, startY int) int {
	fg := goterm.ColorRGB(220, 220, 220)
	bg := goterm.ColorDefault()

	// Header
	screen.DrawText(0, startY, "MCP Servers:", fg, bg, goterm.StyleBold)
	y := startY + 1

	if len(v.servers) == 0 {
		screen.DrawText(0, y+1, "  No servers registered", goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
		screen.DrawText(0, y+2, "  Press 'a' to add a server", goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
		return y + 3
	}

	// Server list with health status indicators (T198)
	for i, server := range v.servers {
		if y >= v.height-2 {
			break
		}

		prefix := "  "
		style := goterm.StyleNone
		itemFg := fg
		itemBg := bg

		if i == v.selectedIdx {
			prefix = "> "
			itemFg = goterm.ColorRGB(0, 0, 0)
			itemBg = goterm.ColorRGB(100, 200, 255)
			style = goterm.StyleBold
		}

		// Health status indicator (T198)
		statusIcon := v.getHealthStatusIcon(server)
		transportBadge := v.getTransportBadge(server)

		// Format: > [✓] server-name (stdio) - 5 tools
		line := fmt.Sprintf("%s%s %s %s", prefix, statusIcon, server.Name, transportBadge)

		// Add tool count if tools discovered
		if len(server.Tools) > 0 {
			line += fmt.Sprintf(" - %d tools", len(server.Tools))
		}

		// Truncate if too long
		if len(line) > v.width {
			line = line[:v.width]
		}

		// Pad to full width if selected
		if i == v.selectedIdx {
			line += strings.Repeat(" ", v.width-len(line))
		}

		for x, ch := range line {
			if x >= v.width {
				break
			}
			screen.SetCell(x, y, goterm.NewCell(ch, itemFg, itemBg, style))
		}

		y++
	}

	return y
}

// renderServerDetailsView renders detailed server information (T198)
func (v *ServerRegistryView) renderServerDetailsView(screen *goterm.Screen, startY int) int {
	server := v.servers[v.selectedIdx]

	fg := goterm.ColorRGB(220, 220, 220)
	bg := goterm.ColorDefault()

	screen.DrawText(0, startY, "Server Details:", fg, bg, goterm.StyleBold)
	y := startY + 2

	// Basic information
	details := []struct {
		label string
		value string
	}{
		{"ID:", server.ID},
		{"Name:", server.Name},
		{"Transport:", string(server.Transport.Type())},
		{"Status:", v.getConnectionStateLabel(server)},
		{"Health:", v.getHealthStatusLabel(server)},
	}

	for _, detail := range details {
		if y >= v.height-2 {
			break
		}
		line := fmt.Sprintf("  %-15s %s", detail.label, detail.value)
		screen.DrawText(0, y, line, fg, bg, goterm.StyleNone)
		y++
	}

	// Transport-specific details
	y++
	if y < v.height-2 {
		screen.DrawText(0, y, "Transport Configuration:", fg, bg, goterm.StyleBold)
		y++
	}

	switch cfg := server.Transport.(type) {
	case *mcpserver.StdioTransportConfig:
		if y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("  Command:  %s", cfg.Command), fg, bg, goterm.StyleNone)
			y++
		}
		if len(cfg.Args) > 0 && y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("  Args:     %v", cfg.Args), fg, bg, goterm.StyleNone)
			y++
		}
	case *mcpserver.SSETransportConfig:
		if y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("  URL:      %s", cfg.URL), fg, bg, goterm.StyleNone)
			y++
		}
	case *mcpserver.HTTPTransportConfig:
		if y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("  Base URL: %s", cfg.BaseURL), fg, bg, goterm.StyleNone)
			y++
		}
		if y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("  Timeout:  %v", cfg.Timeout), fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Connection statistics (T198)
	y++
	if y < v.height-2 {
		screen.DrawText(0, y, "Connection Statistics:", fg, bg, goterm.StyleBold)
		y++
	}

	if !server.Connection.ConnectedAt.IsZero() && y < v.height-2 {
		screen.DrawText(0, y, fmt.Sprintf("  Connected:    %s", server.Connection.ConnectedAt.Format("2006-01-02 15:04:05")), fg, bg, goterm.StyleNone)
		y++
	}

	lastActivity := server.Connection.GetLastActivity()
	if !lastActivity.IsZero() && y < v.height-2 {
		screen.DrawText(0, y, fmt.Sprintf("  Last Activity: %s", lastActivity.Format("2006-01-02 15:04:05")), fg, bg, goterm.StyleNone)
		y++
	}

	// Get error info using thread-safe getters
	errorCount := server.Connection.GetErrorCount()
	lastError := server.Connection.GetLastError()

	if errorCount > 0 && y < v.height-2 {
		screen.DrawText(0, y, fmt.Sprintf("  Error Count:  %d", errorCount), fg, bg, goterm.StyleNone)
		y++
	}

	if lastError != "" && y < v.height-2 {
		screen.DrawText(0, y, fmt.Sprintf("  Last Error:   %s", lastError), goterm.ColorRGB(255, 100, 100), bg, goterm.StyleNone)
		y++
	}

	// Health check info (T198)
	if !server.LastHealthCheck.IsZero() && y < v.height-2 {
		y++
		screen.DrawText(0, y, fmt.Sprintf("  Last Health Check: %s", server.LastHealthCheck.Format("2006-01-02 15:04:05")), fg, bg, goterm.StyleNone)
		y++
	}

	// Tools summary
	if len(server.Tools) > 0 {
		y++
		if y < v.height-2 {
			screen.DrawText(0, y, fmt.Sprintf("Available Tools: %d", len(server.Tools)), fg, bg, goterm.StyleBold)
			y++
		}

		// Show first few tools
		toolsToShow := 5
		for i, tool := range server.Tools {
			if i >= toolsToShow || y >= v.height-2 {
				if len(server.Tools) > toolsToShow && y < v.height-2 {
					screen.DrawText(0, y, fmt.Sprintf("  ... and %d more (press 's' to view all)", len(server.Tools)-toolsToShow), goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
				}
				break
			}
			screen.DrawText(0, y, fmt.Sprintf("  - %s", tool.Name), fg, bg, goterm.StyleNone)
			y++
		}
	}

	return y
}

// renderToolSchemaView renders the tool schema viewer (T199)
func (v *ServerRegistryView) renderToolSchemaView(screen *goterm.Screen, startY int) int {
	server := v.servers[v.selectedIdx]

	fg := goterm.ColorRGB(220, 220, 220)
	bg := goterm.ColorDefault()

	screen.DrawText(0, startY, fmt.Sprintf("Tool Schemas - %s:", server.Name), fg, bg, goterm.StyleBold)
	y := startY + 2

	if len(server.Tools) == 0 {
		screen.DrawText(0, y, "  No tools discovered yet", goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
		screen.DrawText(0, y+1, "  Press 't' to test connection and discover tools", goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
		return y + 2
	}

	// Tool list
	maxToolsToShow := (v.height - y - 5) / 2 // Reserve space for details
	if maxToolsToShow < 5 {
		maxToolsToShow = 5
	}

	for i, tool := range server.Tools {
		if i >= maxToolsToShow || y >= v.height-2 {
			if len(server.Tools) > maxToolsToShow && y < v.height-2 {
				screen.DrawText(0, y, fmt.Sprintf("  ... and %d more tools", len(server.Tools)-maxToolsToShow), goterm.ColorRGB(150, 150, 150), bg, goterm.StyleDim)
				y++
			}
			break
		}

		prefix := "  "
		itemFg := fg
		itemBg := bg
		style := goterm.StyleNone

		if i == v.selectedTool {
			prefix = "> "
			itemFg = goterm.ColorRGB(0, 0, 0)
			itemBg = goterm.ColorRGB(100, 200, 255)
			style = goterm.StyleBold
		}

		line := fmt.Sprintf("%s%s", prefix, tool.Name)

		// Add description if available and not selected
		if tool.Description != "" && i != v.selectedTool {
			maxDescLen := v.width - len(line) - 3
			desc := tool.Description
			if len(desc) > maxDescLen && maxDescLen > 3 {
				desc = desc[:maxDescLen-3] + "..."
			}
			line += " - " + desc
		}

		// Truncate if too long
		if len(line) > v.width {
			line = line[:v.width]
		}

		// Pad if selected
		if i == v.selectedTool {
			line += strings.Repeat(" ", v.width-len(line))
		}

		for x, ch := range line {
			if x >= v.width {
				break
			}
			screen.SetCell(x, y, goterm.NewCell(ch, itemFg, itemBg, style))
		}

		y++
	}

	// Show selected tool details
	if v.selectedTool < len(server.Tools) {
		y++
		if y < v.height-2 {
			selectedTool := server.Tools[v.selectedTool]
			screen.DrawText(0, y, "Tool Details:", fg, bg, goterm.StyleBold)
			y++

			if y < v.height-2 {
				screen.DrawText(0, y, fmt.Sprintf("  Name: %s", selectedTool.Name), fg, bg, goterm.StyleNone)
				y++
			}

			if selectedTool.Description != "" && y < v.height-2 {
				// Word wrap description
				descLines := v.wrapText(selectedTool.Description, v.width-4)
				for _, line := range descLines {
					if y >= v.height-2 {
						break
					}
					screen.DrawText(0, y, "  "+line, fg, bg, goterm.StyleNone)
					y++
				}
			}

			// Show schema info if available
			if selectedTool.InputSchema != nil && y < v.height-2 {
				y++
				screen.DrawText(0, y, "  Input Schema:", fg, bg, goterm.StyleNone)
				y++

				if y < v.height-2 {
					screen.DrawText(0, y, fmt.Sprintf("    Type: %s", selectedTool.InputSchema.Type), fg, bg, goterm.StyleNone)
					y++
				}

				if len(selectedTool.InputSchema.Required) > 0 && y < v.height-2 {
					screen.DrawText(0, y, fmt.Sprintf("    Required: %v", selectedTool.InputSchema.Required), fg, bg, goterm.StyleNone)
					y++
				}
			}
		}
	}

	return y
}

// Helper functions

// getHealthStatusIcon returns the icon for a server's health status (T198)
func (v *ServerRegistryView) getHealthStatusIcon(server *mcpserver.MCPServer) string {
	switch server.HealthStatus {
	case mcpserver.HealthHealthy:
		return "[✓]"
	case mcpserver.HealthUnhealthy:
		return "[✗]"
	case mcpserver.HealthDisconnected:
		return "[○]"
	case mcpserver.HealthUnknown:
		return "[?]"
	default:
		return "[?]"
	}
}

// getHealthStatusLabel returns a detailed label for health status (T198)
func (v *ServerRegistryView) getHealthStatusLabel(server *mcpserver.MCPServer) string {
	switch server.HealthStatus {
	case mcpserver.HealthHealthy:
		return "Healthy"
	case mcpserver.HealthUnhealthy:
		return "Unhealthy"
	case mcpserver.HealthDisconnected:
		return "Disconnected"
	case mcpserver.HealthUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// getConnectionStateLabel returns a label for connection state
func (v *ServerRegistryView) getConnectionStateLabel(server *mcpserver.MCPServer) string {
	switch server.Connection.State {
	case mcpserver.StateConnected:
		return "Connected"
	case mcpserver.StateConnecting:
		return "Connecting..."
	case mcpserver.StateDisconnected:
		return "Disconnected"
	case mcpserver.StateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// getTransportBadge returns a badge string for transport type
func (v *ServerRegistryView) getTransportBadge(server *mcpserver.MCPServer) string {
	return fmt.Sprintf("(%s)", server.Transport.Type())
}

// wrapText wraps text to fit within a given width
func (v *ServerRegistryView) wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
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
	return v.loadServers()
}

// SetBounds sets the view dimensions (for testing)
func (v *ServerRegistryView) SetBounds(width, height int) {
	v.width = width
	v.height = height
}
