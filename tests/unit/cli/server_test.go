package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/cli"
)

// TestServerAddCommand_Basic tests basic server add command
func TestServerAddCommand_Basic(t *testing.T) {
	// Use temporary directory for server config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "servers.yaml")

	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Set config path via environment or flag
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	cmd.SetArgs([]string{"add", "test-server", "npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server add, got error: %v", err)
	}

	// Verify success message
	output := stdout.String()
	if !strings.Contains(output, "added") && !strings.Contains(output, "success") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected server config file to be created")
	}
}

// TestServerAddCommand_WithDescription tests server add with description
func TestServerAddCommand_WithDescription(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{
		"add",
		"test-server",
		"npx",
		"-y",
		"@modelcontextprotocol/server-filesystem",
		"/tmp",
		"--description",
		"Test filesystem server",
	})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server add with description, got error: %v", err)
	}
}

// TestServerAddCommand_DuplicateID tests error handling for duplicate server ID
func TestServerAddCommand_DuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Add server first time
	cmd.SetArgs([]string{"add", "test-server", "echo", "test"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("First add failed: %v", err)
	}

	// Try to add same server ID again
	cmd2 := cli.NewServerCommand()
	cmd2.SetOut(&stdout)
	cmd2.SetErr(&stderr)
	cmd2.SetArgs([]string{"add", "test-server", "echo", "test2"})

	err = cmd2.Execute()
	if err == nil {
		t.Error("Expected error for duplicate server ID, got nil")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "already exists") && !strings.Contains(errOutput, "duplicate") {
		t.Errorf("Expected duplicate error message, got: %s", errOutput)
	}
}

// TestServerAddCommand_InvalidServerID tests error for invalid server ID format
func TestServerAddCommand_InvalidServerID(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Use invalid server ID (with spaces or special characters)
	cmd.SetArgs([]string{"add", "invalid server id!", "echo", "test"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid server ID, got nil")
	}
}

// TestServerListCommand_Empty tests listing servers when none exist
func TestServerListCommand_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful list command, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No servers") && !strings.Contains(output, "empty") && len(output) != 0 {
		t.Errorf("Expected empty list message, got: %s", output)
	}
}

// TestServerListCommand_WithServers tests listing configured servers
func TestServerListCommand_WithServers(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add some servers first
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "server1", "echo", "test1"})
	_ = addCmd.Execute()

	addCmd2 := cli.NewServerCommand()
	addCmd2.SetArgs([]string{"add", "server2", "echo", "test2"})
	_ = addCmd2.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	listCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	listCmd.SetOut(&stdout)
	listCmd.SetErr(&stderr)

	listCmd.SetArgs([]string{"list"})

	err := listCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful list command, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "server1") {
		t.Error("Expected output to contain server1")
	}
	if !strings.Contains(output, "server2") {
		t.Error("Expected output to contain server2")
	}
}

// TestServerListCommand_JSONFormat tests listing servers in JSON format
func TestServerListCommand_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "test-server", "echo", "test"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	listCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	listCmd.SetOut(&stdout)
	listCmd.SetErr(&stderr)

	listCmd.SetArgs([]string{"list", "--output", "json"})

	err := listCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful list command, got error: %v", err)
	}

	output := stdout.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "[") && !strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Errorf("Expected JSON output, got: %s", output)
	}
}

// TestServerTestCommand_ValidServer tests testing a valid server connection
func TestServerTestCommand_ValidServer(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server that should work (echo command)
	addCmd := cli.NewServerCommand()
	mockServerPath := filepath.Join("..", "..", "..", "internal", "testutil", "mocks", "mock_mcp_server.go")
	addCmd.SetArgs([]string{"add", "test-server", "go", "run", mockServerPath, "--mode=server"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	testCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	testCmd.SetArgs([]string{"test", "test-server"})

	err := testCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server test, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "success") && !strings.Contains(output, "OK") && !strings.Contains(output, "connected") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

// TestServerTestCommand_InvalidServer tests testing with non-existent server
func TestServerTestCommand_InvalidServer(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	testCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	testCmd.SetArgs([]string{"test", "nonexistent-server"})

	err := testCmd.Execute()
	if err == nil {
		t.Error("Expected error for non-existent server, got nil")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "not found") && !strings.Contains(errOutput, "does not exist") {
		t.Errorf("Expected not found error, got: %s", errOutput)
	}
}

// TestServerTestCommand_FailedConnection tests testing server with failed connection
func TestServerTestCommand_FailedConnection(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server with invalid command
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "invalid-server", "nonexistent-command-12345"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	testCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	testCmd.SetArgs([]string{"test", "invalid-server"})

	err := testCmd.Execute()
	if err == nil {
		t.Error("Expected error for failed connection, got nil")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "failed") && !strings.Contains(errOutput, "error") {
		t.Errorf("Expected connection failure error, got: %s", errOutput)
	}
}

// TestServerRemoveCommand_Basic tests removing a server
func TestServerRemoveCommand_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server first
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "test-server", "echo", "test"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	removeCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	removeCmd.SetArgs([]string{"remove", "test-server"})

	err := removeCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server removal, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "removed") && !strings.Contains(output, "deleted") {
		t.Errorf("Expected removal confirmation, got: %s", output)
	}

	// Verify server is gone
	listCmd := cli.NewServerCommand()
	var listOut bytes.Buffer
	listCmd.SetOut(&listOut)
	listCmd.SetArgs([]string{"list"})
	_ = listCmd.Execute()

	if strings.Contains(listOut.String(), "test-server") {
		t.Error("Expected server to be removed from list")
	}
}

// TestServerRemoveCommand_NonExistent tests removing non-existent server
func TestServerRemoveCommand_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewServerCommand doesn't exist yet
	removeCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	removeCmd.SetOut(&stdout)
	removeCmd.SetErr(&stderr)

	removeCmd.SetArgs([]string{"remove", "nonexistent-server"})

	err := removeCmd.Execute()
	if err == nil {
		t.Error("Expected error for removing non-existent server, got nil")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "not found") && !strings.Contains(errOutput, "does not exist") {
		t.Errorf("Expected not found error, got: %s", errOutput)
	}
}

// TestServerUpdateCommand_Basic tests updating server configuration
func TestServerUpdateCommand_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server first
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "test-server", "echo", "test"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	updateCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	updateCmd.SetOut(&stdout)
	updateCmd.SetErr(&stderr)

	updateCmd.SetArgs([]string{
		"update",
		"test-server",
		"--description",
		"Updated description",
	})

	err := updateCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server update, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "updated") && !strings.Contains(output, "success") {
		t.Errorf("Expected update confirmation, got: %s", output)
	}
}

// TestServerShowCommand_Basic tests showing server details
func TestServerShowCommand_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// Add a server first
	addCmd := cli.NewServerCommand()
	addCmd.SetArgs([]string{"add", "test-server", "echo", "test", "--description", "Test server"})
	_ = addCmd.Execute()

	// This should fail because cli.NewServerCommand doesn't exist yet
	showCmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	showCmd.SetOut(&stdout)
	showCmd.SetErr(&stderr)

	showCmd.SetArgs([]string{"show", "test-server"})

	err := showCmd.Execute()
	if err != nil {
		t.Errorf("Expected successful server show, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "test-server") {
		t.Error("Expected output to contain server ID")
	}
	if !strings.Contains(output, "echo") {
		t.Error("Expected output to contain command")
	}
	if !strings.Contains(output, "Test server") {
		t.Error("Expected output to contain description")
	}
}

// TestServerCommand_NoSubcommand tests server command without subcommand
func TestServerCommand_NoSubcommand(t *testing.T) {
	// This should fail because cli.NewServerCommand doesn't exist yet
	cmd := cli.NewServerCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{})

	err := cmd.Execute()
	// Should either show help or error
	if err == nil {
		// If no error, should show help text
		output := stdout.String()
		if !strings.Contains(output, "server") && !strings.Contains(output, "Usage") {
			t.Error("Expected help text when no subcommand provided")
		}
	}
}
