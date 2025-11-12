package tui

import (
	"fmt"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/dshills/goterm"
)

// TestServerRegistryView_NewServerRegistryView tests creating a new server registry view
func TestServerRegistryView_NewServerRegistryView(t *testing.T) {
	view := NewServerRegistryView()

	if view == nil {
		t.Fatal("NewServerRegistryView returned nil")
	}

	if view.Name() != "registry" {
		t.Errorf("expected name 'registry', got %q", view.Name())
	}

	if view.IsActive() {
		t.Error("expected view to not be active initially")
	}

	if view.registry == nil {
		t.Error("expected registry to be initialized")
	}

	if view.autoRefresh != true {
		t.Error("expected auto-refresh to be enabled by default")
	}
}

// TestServerRegistryView_Init tests view initialization
func TestServerRegistryView_Init(t *testing.T) {
	view := NewServerRegistryView()
	registry := mcpserver.NewRegistry()
	view.SetRegistry(registry)

	// Add some test servers
	server1, _ := mcpserver.NewMCPServer("test1", "echo", []string{"hello"}, mcpserver.TransportStdio)
	server2, _ := mcpserver.NewMCPServer("test2", "http://localhost:3000", nil, mcpserver.TransportSSE)
	registry.Register(server1)
	registry.Register(server2)

	err := view.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !view.initialized {
		t.Error("expected initialized to be true")
	}

	if len(view.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(view.servers))
	}

	if view.statusMsg != "Ready" {
		t.Errorf("expected status 'Ready', got %q", view.statusMsg)
	}
}

// TestServerRegistryView_HandleKey_Navigation tests j/k navigation
func TestServerRegistryView_HandleKey_Navigation(t *testing.T) {
	view := setupTestView(t, 3)

	// Initial selection should be 0
	if view.selectedIdx != 0 {
		t.Errorf("expected initial selection 0, got %d", view.selectedIdx)
	}

	// Move down with j
	view.HandleKey(KeyEvent{Key: 'j'})
	if view.selectedIdx != 1 {
		t.Errorf("expected selection 1 after 'j', got %d", view.selectedIdx)
	}

	// Move down with Down arrow
	view.HandleKey(KeyEvent{IsSpecial: true, Special: "Down"})
	if view.selectedIdx != 2 {
		t.Errorf("expected selection 2 after Down, got %d", view.selectedIdx)
	}

	// Try to move past end
	view.HandleKey(KeyEvent{Key: 'j'})
	if view.selectedIdx != 2 {
		t.Errorf("expected selection to stay at 2 at end, got %d", view.selectedIdx)
	}

	// Move up with k
	view.HandleKey(KeyEvent{Key: 'k'})
	if view.selectedIdx != 1 {
		t.Errorf("expected selection 1 after 'k', got %d", view.selectedIdx)
	}

	// Move up with Up arrow
	view.HandleKey(KeyEvent{IsSpecial: true, Special: "Up"})
	if view.selectedIdx != 0 {
		t.Errorf("expected selection 0 after Up, got %d", view.selectedIdx)
	}

	// Try to move before start
	view.HandleKey(KeyEvent{Key: 'k'})
	if view.selectedIdx != 0 {
		t.Errorf("expected selection to stay at 0 at start, got %d", view.selectedIdx)
	}
}

// TestServerRegistryView_HandleKey_GoToTopBottom tests g/G navigation
func TestServerRegistryView_HandleKey_GoToTopBottom(t *testing.T) {
	view := setupTestView(t, 5)

	// Go to bottom with G
	view.HandleKey(KeyEvent{Key: 'G', Shift: true})
	if view.selectedIdx != 4 {
		t.Errorf("expected selection 4 after 'G', got %d", view.selectedIdx)
	}

	// Go to top with g
	view.HandleKey(KeyEvent{Key: 'g'})
	if view.selectedIdx != 0 {
		t.Errorf("expected selection 0 after 'g', got %d", view.selectedIdx)
	}
}

// TestServerRegistryView_HandleKey_ToggleDetails tests Enter and i keys
func TestServerRegistryView_HandleKey_ToggleDetails(t *testing.T) {
	view := setupTestView(t, 1)

	if view.showDetails {
		t.Error("expected showDetails to be false initially")
	}

	// Toggle details with Enter
	view.HandleKey(KeyEvent{IsSpecial: true, Special: "Enter"})
	if !view.showDetails {
		t.Error("expected showDetails to be true after Enter")
	}

	// Toggle off with Enter
	view.HandleKey(KeyEvent{IsSpecial: true, Special: "Enter"})
	if view.showDetails {
		t.Error("expected showDetails to be false after second Enter")
	}

	// Toggle with i
	view.HandleKey(KeyEvent{Key: 'i'})
	if !view.showDetails {
		t.Error("expected showDetails to be true after 'i'")
	}
}

// TestServerRegistryView_HandleKey_ToolSchema tests tool schema view (T199)
func TestServerRegistryView_HandleKey_ToolSchema(t *testing.T) {
	view := setupTestView(t, 1)

	// Add tools to server
	server := view.servers[0]
	server.Tools = []mcpserver.Tool{
		{Name: "tool1", Description: "Test tool 1"},
		{Name: "tool2", Description: "Test tool 2"},
		{Name: "tool3", Description: "Test tool 3"},
	}

	// Toggle tool schema view with s
	view.HandleKey(KeyEvent{Key: 's'})
	if !view.showToolSchema {
		t.Error("expected showToolSchema to be true after 's'")
	}

	if view.showDetails {
		t.Error("expected showDetails to be false when showToolSchema is true")
	}

	if view.selectedTool != 0 {
		t.Errorf("expected selectedTool to be 0, got %d", view.selectedTool)
	}

	// Navigate tools with j
	view.HandleKey(KeyEvent{Key: 'j'})
	if view.selectedTool != 1 {
		t.Errorf("expected selectedTool to be 1 after 'j', got %d", view.selectedTool)
	}

	// Navigate tools with k
	view.HandleKey(KeyEvent{Key: 'k'})
	if view.selectedTool != 0 {
		t.Errorf("expected selectedTool to be 0 after 'k', got %d", view.selectedTool)
	}

	// Exit tool schema view with Escape
	view.HandleKey(KeyEvent{IsSpecial: true, Special: "Escape"})
	if view.showToolSchema {
		t.Error("expected showToolSchema to be false after Escape")
	}
}

// TestServerRegistryView_HandleKey_AutoRefresh tests auto-refresh toggle (T198)
func TestServerRegistryView_HandleKey_AutoRefresh(t *testing.T) {
	view := setupTestView(t, 1)

	if !view.autoRefresh {
		t.Error("expected autoRefresh to be true initially")
	}

	// Toggle auto-refresh off with R
	view.HandleKey(KeyEvent{Key: 'R', Shift: true})
	if view.autoRefresh {
		t.Error("expected autoRefresh to be false after 'R'")
	}

	if view.statusMsg != "Auto-refresh disabled" {
		t.Errorf("expected status 'Auto-refresh disabled', got %q", view.statusMsg)
	}

	// Toggle auto-refresh on with R
	view.HandleKey(KeyEvent{Key: 'R', Shift: true})
	if !view.autoRefresh {
		t.Error("expected autoRefresh to be true after second 'R'")
	}

	if view.statusMsg != "Auto-refresh enabled" {
		t.Errorf("expected status 'Auto-refresh enabled', got %q", view.statusMsg)
	}
}

// TestServerRegistryView_ConnectServer tests server connection
func TestServerRegistryView_ConnectServer(t *testing.T) {
	view := setupTestView(t, 1)
	server := view.servers[0]

	if server.Connection.GetState() != mcpserver.StateDisconnected {
		t.Errorf("expected initial state disconnected, got %s", server.Connection.GetState())
	}

	// Connect server
	view.connectServer()

	if server.Connection.GetState() != mcpserver.StateConnected {
		t.Errorf("expected state connected, got %s", server.Connection.GetState())
	}

	// Try to connect again (should show already connected)
	oldMsg := view.statusMsg
	view.connectServer()

	if view.statusMsg != "Server already connected" {
		t.Errorf("expected 'Server already connected', got %q", view.statusMsg)
	}

	// Verify status message from first connect
	if oldMsg == "" {
		t.Error("expected status message after successful connect")
	}
}

// TestServerRegistryView_DisconnectServer tests server disconnection
func TestServerRegistryView_DisconnectServer(t *testing.T) {
	view := setupTestView(t, 1)
	server := view.servers[0]

	// Connect first
	view.connectServer()

	if server.Connection.GetState() != mcpserver.StateConnected {
		t.Fatalf("server should be connected, got %s", server.Connection.GetState())
	}

	// Disconnect
	view.disconnectServer()

	if server.Connection.GetState() != mcpserver.StateDisconnected {
		t.Errorf("expected state disconnected after disconnect, got %s", server.Connection.GetState())
	}

	// Try to disconnect again (should show not connected)
	view.disconnectServer()

	if view.statusMsg != "Server not connected" {
		t.Errorf("expected 'Server not connected', got %q", view.statusMsg)
	}
}

// TestServerRegistryView_TestConnection tests the test connection feature (T198)
func TestServerRegistryView_TestConnection(t *testing.T) {
	view := setupTestView(t, 1)
	server := view.servers[0]

	// Test connection (connects, health checks, discovers tools)
	view.testServerConnection()

	if server.Connection.GetState() != mcpserver.StateConnected {
		t.Errorf("expected connected state after test, got %s", server.Connection.GetState())
	}

	if server.HealthStatus == mcpserver.HealthUnknown {
		t.Error("expected health status to be checked")
	}

	// Verify status message indicates success
	if view.statusMsg == "" {
		t.Error("expected status message after connection test")
	}
}

// TestServerRegistryView_RefreshServerStatus tests status refresh (T198)
func TestServerRegistryView_RefreshServerStatus(t *testing.T) {
	view := setupTestView(t, 3)

	// Connect all servers
	for _, server := range view.servers {
		server.Connect()
		server.CompleteConnection()
	}

	oldRefreshTime := view.lastRefresh

	// Refresh status
	view.refreshServerStatus()

	// Verify refresh time was updated
	if !view.lastRefresh.After(oldRefreshTime) {
		t.Error("expected lastRefresh to be updated")
	}

	// Verify status message
	if view.statusMsg == "" {
		t.Error("expected status message after refresh")
	}
}

// TestServerRegistryView_Render tests basic rendering
func TestServerRegistryView_Render(t *testing.T) {
	view := setupTestView(t, 2)

	// Create a mock screen
	screen, err := goterm.Init()
	if err != nil {
		t.Skip("Cannot initialize terminal for render test")
	}
	defer screen.Close()

	// Test rendering server list
	err = view.Render(screen)
	if err != nil {
		t.Errorf("Render failed: %v", err)
	}

	// Test rendering with details view
	view.showDetails = true
	err = view.Render(screen)
	if err != nil {
		t.Errorf("Render with details failed: %v", err)
	}

	// Test rendering with tool schema view
	view.showDetails = false
	view.showToolSchema = true
	view.servers[0].Tools = []mcpserver.Tool{
		{Name: "test_tool", Description: "A test tool"},
	}
	err = view.Render(screen)
	if err != nil {
		t.Errorf("Render with tool schema failed: %v", err)
	}
}

// TestServerRegistryView_GetHealthStatusIcon tests health status icons (T198)
func TestServerRegistryView_GetHealthStatusIcon(t *testing.T) {
	view := NewServerRegistryView()

	testCases := []struct {
		status       mcpserver.HealthStatus
		expectedIcon string
	}{
		{mcpserver.HealthHealthy, "[✓]"},
		{mcpserver.HealthUnhealthy, "[✗]"},
		{mcpserver.HealthDisconnected, "[○]"},
		{mcpserver.HealthUnknown, "[?]"},
	}

	for _, tc := range testCases {
		server := &mcpserver.MCPServer{
			HealthStatus: tc.status,
		}

		icon := view.getHealthStatusIcon(server)
		if icon != tc.expectedIcon {
			t.Errorf("for status %s, expected icon %q, got %q", tc.status, tc.expectedIcon, icon)
		}
	}
}

// TestServerRegistryView_GetHealthStatusLabel tests health status labels (T198)
func TestServerRegistryView_GetHealthStatusLabel(t *testing.T) {
	view := NewServerRegistryView()

	testCases := []struct {
		status        mcpserver.HealthStatus
		expectedLabel string
	}{
		{mcpserver.HealthHealthy, "Healthy"},
		{mcpserver.HealthUnhealthy, "Unhealthy"},
		{mcpserver.HealthDisconnected, "Disconnected"},
		{mcpserver.HealthUnknown, "Unknown"},
	}

	for _, tc := range testCases {
		server := &mcpserver.MCPServer{
			HealthStatus: tc.status,
		}

		label := view.getHealthStatusLabel(server)
		if label != tc.expectedLabel {
			t.Errorf("for status %s, expected label %q, got %q", tc.status, tc.expectedLabel, label)
		}
	}
}

// TestServerRegistryView_GetConnectionStateLabel tests connection state labels
func TestServerRegistryView_GetConnectionStateLabel(t *testing.T) {
	view := NewServerRegistryView()

	testCases := []struct {
		state         mcpserver.ConnectionState
		expectedLabel string
	}{
		{mcpserver.StateConnected, "Connected"},
		{mcpserver.StateConnecting, "Connecting..."},
		{mcpserver.StateDisconnected, "Disconnected"},
		{mcpserver.StateFailed, "Failed"},
	}

	for _, tc := range testCases {
		server := &mcpserver.MCPServer{
			Connection: &mcpserver.Connection{
				State: tc.state,
			},
		}

		label := view.getConnectionStateLabel(server)
		if label != tc.expectedLabel {
			t.Errorf("for state %s, expected label %q, got %q", tc.state, tc.expectedLabel, label)
		}
	}
}

// TestServerRegistryView_WrapText tests text wrapping utility
func TestServerRegistryView_WrapText(t *testing.T) {
	view := NewServerRegistryView()

	testCases := []struct {
		text     string
		width    int
		expected []string
	}{
		{
			text:     "short",
			width:    20,
			expected: []string{"short"},
		},
		{
			text:     "this is a longer text that needs wrapping",
			width:    20,
			expected: []string{"this is a longer", "text that needs", "wrapping"},
		},
		{
			text:     "a b c d e f g",
			width:    5,
			expected: []string{"a b c", "d e f", "g"},
		},
		{
			text:     "",
			width:    20,
			expected: []string{""},
		},
	}

	for _, tc := range testCases {
		result := view.wrapText(tc.text, tc.width)

		if len(result) != len(tc.expected) {
			t.Errorf("for text %q width %d, expected %d lines, got %d",
				tc.text, tc.width, len(tc.expected), len(result))
			continue
		}

		for i, line := range result {
			if line != tc.expected[i] {
				t.Errorf("for text %q width %d, line %d: expected %q, got %q",
					tc.text, tc.width, i, tc.expected[i], line)
			}
		}
	}
}

// TestServerRegistryView_SetActive tests active state management
func TestServerRegistryView_SetActive(t *testing.T) {
	view := NewServerRegistryView()

	if view.IsActive() {
		t.Error("expected view to not be active initially")
	}

	view.SetActive(true)
	if !view.IsActive() {
		t.Error("expected view to be active after SetActive(true)")
	}

	view.SetActive(false)
	if view.IsActive() {
		t.Error("expected view to not be active after SetActive(false)")
	}
}

// TestServerRegistryView_RefreshServers tests server list refresh
func TestServerRegistryView_RefreshServers(t *testing.T) {
	view := setupTestView(t, 2)

	initialCount := len(view.servers)

	// Add another server to registry
	server3, _ := mcpserver.NewMCPServer("test3", "echo", []string{"world"}, mcpserver.TransportStdio)
	view.registry.Register(server3)

	// Refresh
	err := view.RefreshServers()
	if err != nil {
		t.Fatalf("RefreshServers failed: %v", err)
	}

	if len(view.servers) != initialCount+1 {
		t.Errorf("expected %d servers after refresh, got %d", initialCount+1, len(view.servers))
	}
}

// TestServerRegistryView_AutoRefreshBehavior tests auto-refresh mechanism (T198)
func TestServerRegistryView_AutoRefreshBehavior(t *testing.T) {
	view := setupTestView(t, 1)
	server := view.servers[0]

	// Connect server
	server.Connect()
	server.CompleteConnection()

	// Set last refresh to past time
	view.lastRefresh = time.Now().Add(-15 * time.Second)

	// Create mock screen for render
	screen, err := goterm.Init()
	if err != nil {
		t.Skip("Cannot initialize terminal for auto-refresh test")
	}
	defer screen.Close()

	// Render should trigger auto-refresh
	oldRefreshTime := view.lastRefresh
	view.Render(screen)

	// Verify refresh time was updated
	if !view.lastRefresh.After(oldRefreshTime) {
		t.Error("expected auto-refresh to update lastRefresh time")
	}
}

// TestServerRegistryView_EmptyServerList tests behavior with no servers
func TestServerRegistryView_EmptyServerList(t *testing.T) {
	view := NewServerRegistryView()
	view.Init()

	if len(view.servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(view.servers))
	}

	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 with empty list, got %d", view.selectedIdx)
	}

	// Test navigation on empty list
	view.HandleKey(KeyEvent{Key: 'j'})
	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx to stay 0 on empty list, got %d", view.selectedIdx)
	}
}

// Helper function to set up a test view with servers
func setupTestView(t *testing.T, serverCount int) *ServerRegistryView {
	t.Helper()

	view := NewServerRegistryView()
	registry := mcpserver.NewRegistry()
	view.SetRegistry(registry)

	// Add test servers
	for i := 0; i < serverCount; i++ {
		serverID := fmt.Sprintf("test%d", i+1)
		command := "echo"
		args := []string{fmt.Sprintf("server%d", i+1)}

		server, err := mcpserver.NewMCPServer(serverID, command, args, mcpserver.TransportStdio)
		if err != nil {
			t.Fatalf("failed to create server %d: %v", i+1, err)
		}

		server.Name = fmt.Sprintf("Test Server %d", i+1)

		if err := registry.Register(server); err != nil {
			t.Fatalf("failed to register server %d: %v", i+1, err)
		}
	}

	// Initialize view
	if err := view.Init(); err != nil {
		t.Fatalf("failed to initialize view: %v", err)
	}

	return view
}
