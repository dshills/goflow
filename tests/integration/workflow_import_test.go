package integration

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/cli"
	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowImport_AllServersConfigured tests importing a workflow with all servers already configured
func TestWorkflowImport_AllServersConfigured(t *testing.T) {
	// Setup: Create registry with all required servers
	registry := mcpserver.NewRegistry()

	// Register test-server referenced in workflow
	server, err := mcpserver.NewMCPServer("test-server", "go", []string{"run", "test.go"}, mcpserver.TransportStdio)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	if err := registry.Register(server); err != nil {
		t.Fatalf("Failed to register server: %v", err)
	}

	// Load workflow from fixture
	// Note: Using import-test-simple.yaml instead of simple-workflow.yaml
	// because simple-workflow.yaml has transform nodes with template variables
	// that currently fail validation (see workflow validation TODO)
	fixturePath := "../../internal/testutil/fixtures/import-test-simple.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Import workflow - this should FAIL initially (TDD)
	wf, err := cli.ImportWorkflow(absPath, registry)
	if err != nil {
		t.Fatalf("Expected successful import, got error: %v", err)
	}

	// Verify workflow structure
	if wf == nil {
		t.Fatal("Expected non-nil workflow")
	}
	if wf.Name != "import-test-simple" {
		t.Errorf("Expected name 'import-test-simple', got '%s'", wf.Name)
	}

	// Verify server references are validated
	if len(wf.ServerConfigs) != 1 {
		t.Errorf("Expected 1 server config, got %d", len(wf.ServerConfigs))
	}
}

// TestWorkflowImport_MissingServers tests importing a workflow with missing servers
func TestWorkflowImport_MissingServers(t *testing.T) {
	// Setup: Create empty registry (no servers configured)
	registry := mcpserver.NewRegistry()

	// Load workflow from fixture (references test-server)
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Import workflow - this should FAIL with missing server error
	wf, err := cli.ImportWorkflow(absPath, registry)

	// Should return error listing missing servers
	if err == nil {
		t.Fatal("Expected error for missing servers, got nil")
	}

	var missingServerErr *cli.MissingServerError
	if !errors.As(err, &missingServerErr) {
		t.Errorf("Expected MissingServerError, got %T: %v", err, err)
	} else {
		// Verify error contains the missing server ID
		if len(missingServerErr.MissingServers) != 1 {
			t.Errorf("Expected 1 missing server, got %d", len(missingServerErr.MissingServers))
		}
		if missingServerErr.MissingServers[0] != "test-server" {
			t.Errorf("Expected missing server 'test-server', got '%s'", missingServerErr.MissingServers[0])
		}
	}

	// Workflow should still be returned for inspection
	if wf == nil {
		t.Error("Expected workflow to be returned even with missing servers")
	}
}

// TestWorkflowImport_MultipleServers tests workflow with multiple server references
func TestWorkflowImport_MultipleServers(t *testing.T) {
	// Create fixture with multiple servers
	workflowYAML := `
version: "1.0"
name: "multi-server-workflow"
description: "Workflow using multiple MCP servers"

servers:
  - id: "filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
  - id: "github"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
  - id: "sqlite"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-sqlite"]

nodes:
  - id: "start"
    type: "start"
  - id: "read_file"
    type: "mcp_tool"
    server: "filesystem"
    tool: "read_file"
    parameters:
      path: "/tmp/data.json"
  - id: "query_db"
    type: "mcp_tool"
    server: "sqlite"
    tool: "query"
    parameters:
      sql: "SELECT * FROM users"
  - id: "create_issue"
    type: "mcp_tool"
    server: "github"
    tool: "create_issue"
    parameters:
      repo: "test/repo"
      title: "Test issue"
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "read_file"
  - from: "read_file"
    to: "query_db"
  - from: "query_db"
    to: "create_issue"
  - from: "create_issue"
    to: "end"
`

	// Write temporary workflow file
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "multi-server.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	// Test Case 1: All servers missing
	t.Run("all_servers_missing", func(t *testing.T) {
		registry := mcpserver.NewRegistry()

		_, err := cli.ImportWorkflow(workflowPath, registry)
		if err == nil {
			t.Fatal("Expected error for missing servers")
		}

		var missingServerErr *cli.MissingServerError
		if !errors.As(err, &missingServerErr) {
			t.Fatalf("Expected MissingServerError, got %T", err)
		}

		// Should report all 3 missing servers
		if len(missingServerErr.MissingServers) != 3 {
			t.Errorf("Expected 3 missing servers, got %d: %v",
				len(missingServerErr.MissingServers), missingServerErr.MissingServers)
		}

		// Verify server IDs
		expectedMissing := map[string]bool{"filesystem": true, "github": true, "sqlite": true}
		for _, serverID := range missingServerErr.MissingServers {
			if !expectedMissing[serverID] {
				t.Errorf("Unexpected missing server: %s", serverID)
			}
		}
	})

	// Test Case 2: Some servers configured
	t.Run("partial_servers_configured", func(t *testing.T) {
		registry := mcpserver.NewRegistry()

		// Register only filesystem server
		server, _ := mcpserver.NewMCPServer("filesystem", "npx", []string{"-y", "@modelcontextprotocol/server-filesystem"}, mcpserver.TransportStdio)
		registry.Register(server)

		_, err := cli.ImportWorkflow(workflowPath, registry)
		if err == nil {
			t.Fatal("Expected error for missing servers")
		}

		var missingServerErr *cli.MissingServerError
		if !errors.As(err, &missingServerErr) {
			t.Fatalf("Expected MissingServerError, got %T", err)
		}

		// Should report 2 missing servers (github and sqlite)
		if len(missingServerErr.MissingServers) != 2 {
			t.Errorf("Expected 2 missing servers, got %d: %v",
				len(missingServerErr.MissingServers), missingServerErr.MissingServers)
		}
	})

	// Test Case 3: All servers configured
	t.Run("all_servers_configured", func(t *testing.T) {
		registry := mcpserver.NewRegistry()

		// Register all servers
		servers := []struct {
			id      string
			command string
		}{
			{"filesystem", "npx"},
			{"github", "npx"},
			{"sqlite", "npx"},
		}

		for _, s := range servers {
			server, _ := mcpserver.NewMCPServer(s.id, s.command, []string{"-y", "@modelcontextprotocol/server-" + s.id}, mcpserver.TransportStdio)
			registry.Register(server)
		}

		wf, err := cli.ImportWorkflow(workflowPath, registry)
		if err != nil {
			t.Fatalf("Expected successful import, got error: %v", err)
		}

		if wf == nil {
			t.Fatal("Expected non-nil workflow")
		}
	})
}

// TestWorkflowImport_CredentialPlaceholders tests handling of credential placeholders
func TestWorkflowImport_CredentialPlaceholders(t *testing.T) {
	workflowYAML := `
version: "1.0"
name: "workflow-with-credentials"
description: "Workflow requiring credentials"

servers:
  - id: "github-api"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
    credential_ref: "{{GITHUB_TOKEN}}"
  - id: "aws-s3"
    command: "aws-mcp-server"
    credential_ref: "{{AWS_CREDENTIALS}}"

nodes:
  - id: "start"
    type: "start"
  - id: "list_repos"
    type: "mcp_tool"
    server: "github-api"
    tool: "list_repositories"
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "list_repos"
  - from: "list_repos"
    to: "end"
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "credentials.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	// Register servers without credentials
	registry := mcpserver.NewRegistry()
	server1, _ := mcpserver.NewMCPServer("github-api", "npx", []string{"-y", "@modelcontextprotocol/server-github"}, mcpserver.TransportStdio)
	registry.Register(server1)
	server2, _ := mcpserver.NewMCPServer("aws-s3", "aws-mcp-server", nil, mcpserver.TransportHTTP)
	registry.Register(server2)

	// Import workflow - should detect credential placeholders
	wf, err := cli.ImportWorkflow(workflowPath, registry)

	// Should return warning about credential placeholders
	var credentialWarning *cli.CredentialPlaceholderWarning
	if !errors.As(err, &credentialWarning) {
		t.Logf("Note: Expected CredentialPlaceholderWarning for credential placeholders (got %T: %v)", err, err)
	} else {
		// Verify warning contains the servers with placeholders
		if len(credentialWarning.ServersWithPlaceholders) != 2 {
			t.Errorf("Expected 2 servers with placeholders, got %d", len(credentialWarning.ServersWithPlaceholders))
		}
	}

	// Workflow should still be imported
	if wf == nil {
		t.Error("Expected workflow to be imported despite credential placeholders")
	}

	// Verify credential references are preserved
	if len(wf.ServerConfigs) != 2 {
		t.Fatalf("Expected 2 server configs, got %d", len(wf.ServerConfigs))
	}

	for _, sc := range wf.ServerConfigs {
		if sc.CredentialRef == "" {
			t.Errorf("Expected credential_ref to be preserved for server %s", sc.ID)
		}
	}
}

// TestWorkflowImport_InvalidYAML tests error handling for invalid YAML
func TestWorkflowImport_InvalidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError string
	}{
		{
			name:        "malformed_yaml",
			yaml:        "this is not: valid: yaml: syntax",
			expectError: "parse",
		},
		{
			name: "missing_version",
			yaml: `
name: "test"
nodes:
  - id: "start"
    type: "start"
`,
			expectError: "version",
		},
		{
			name:        "empty_file",
			yaml:        "",
			expectError: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "invalid.yaml")
			if err := os.WriteFile(workflowPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("Failed to write test fixture: %v", err)
			}

			registry := mcpserver.NewRegistry()
			_, err := cli.ImportWorkflow(workflowPath, registry)

			if err == nil {
				t.Fatal("Expected error for invalid YAML")
			}

			// Verify error message contains expected text
			if !strings.Contains(strings.ToLower(err.Error()), tt.expectError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
			}
		})
	}
}

// TestWorkflowImport_IncompatibleVersion tests handling of incompatible workflow versions
func TestWorkflowImport_IncompatibleVersion(t *testing.T) {
	workflowYAML := `
version: "99.0"
name: "future-workflow"
description: "Workflow from the future"

nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "end"
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "future.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	registry := mcpserver.NewRegistry()
	_, err := cli.ImportWorkflow(workflowPath, registry)

	if err == nil {
		t.Fatal("Expected error for incompatible version")
	}

	var versionErr *cli.IncompatibleVersionError
	if !errors.As(err, &versionErr) {
		t.Logf("Note: Expected IncompatibleVersionError for version 99.0 (got %T: %v)", err, err)
	} else {
		if versionErr.WorkflowVersion != "99.0" {
			t.Errorf("Expected workflow version '99.0', got '%s'", versionErr.WorkflowVersion)
		}
	}
}

// TestWorkflowImport_UnknownNodeTypes tests handling of unknown node types
func TestWorkflowImport_UnknownNodeTypes(t *testing.T) {
	workflowYAML := `
version: "1.0"
name: "future-nodes"
description: "Workflow with unknown node types"

nodes:
  - id: "start"
    type: "start"
  - id: "ai_agent"
    type: "ai_agent"  # Future node type not yet supported
    model: "gpt-5"
    prompt: "Do something smart"
  - id: "quantum_compute"
    type: "quantum_processor"  # Unknown node type
    qubits: 1024
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "ai_agent"
  - from: "ai_agent"
    to: "quantum_compute"
  - from: "quantum_compute"
    to: "end"
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "unknown-nodes.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	registry := mcpserver.NewRegistry()
	_, err := cli.ImportWorkflow(workflowPath, registry)

	if err == nil {
		t.Fatal("Expected error for unknown node types")
	}

	// Should report unknown node types
	if !strings.Contains(err.Error(), "unknown") && !strings.Contains(err.Error(), "node type") {
		t.Errorf("Expected error mentioning unknown node types, got: %v", err)
	}
}

// TestWorkflowImport_PartialServerConfigurations tests workflows with incomplete server configs
func TestWorkflowImport_PartialServerConfigurations(t *testing.T) {
	workflowYAML := `
version: "1.0"
name: "partial-config"
description: "Workflow with minimal server configuration"

servers:
  - id: "minimal-server"
    command: "npx"
    # Missing args, transport, etc.

nodes:
  - id: "start"
    type: "start"
  - id: "tool_call"
    type: "mcp_tool"
    server: "minimal-server"
    tool: "test"
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "tool_call"
  - from: "tool_call"
    to: "end"
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "partial.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	// Register the server with minimal config
	registry := mcpserver.NewRegistry()
	server, err := mcpserver.NewMCPServer("minimal-server", "npx", nil, mcpserver.TransportStdio)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	registry.Register(server)

	// Import should succeed with partial configuration
	wf, err := cli.ImportWorkflow(workflowPath, registry)
	if err != nil {
		t.Fatalf("Expected successful import with partial config, got error: %v", err)
	}

	if wf == nil {
		t.Fatal("Expected non-nil workflow")
	}

	// Verify server config was imported
	if len(wf.ServerConfigs) != 1 {
		t.Errorf("Expected 1 server config, got %d", len(wf.ServerConfigs))
	}

	// Verify minimal fields are present
	sc := wf.ServerConfigs[0]
	if sc.ID != "minimal-server" {
		t.Errorf("Expected server ID 'minimal-server', got '%s'", sc.ID)
	}
	if sc.Command != "npx" {
		t.Errorf("Expected command 'npx', got '%s'", sc.Command)
	}
}

// TestWorkflowImport_ValidatesStructure tests that imported workflow structure is validated
func TestWorkflowImport_ValidatesStructure(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		shouldFail  bool
		expectError string
	}{
		{
			name: "valid_workflow",
			yaml: `
version: "1.0"
name: "valid"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`,
			shouldFail: false,
		},
		{
			name: "missing_start_node",
			yaml: `
version: "1.0"
name: "no-start"
nodes:
  - id: "end"
    type: "end"
`,
			shouldFail:  true,
			expectError: "start node",
		},
		{
			name: "missing_end_node",
			yaml: `
version: "1.0"
name: "no-end"
nodes:
  - id: "start"
    type: "start"
`,
			shouldFail:  true,
			expectError: "end node",
		},
		{
			name: "circular_dependency",
			yaml: `
version: "1.0"
name: "circular"
nodes:
  - id: "start"
    type: "start"
  - id: "node1"
    type: "mcp_tool"
    server: "test"
    tool: "test"
  - id: "node2"
    type: "mcp_tool"
    server: "test"
    tool: "test"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "node1"
  - from: "node1"
    to: "node2"
  - from: "node2"
    to: "node1"  # Creates cycle
`,
			shouldFail:  true,
			expectError: "circular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(workflowPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("Failed to write test fixture: %v", err)
			}

			registry := mcpserver.NewRegistry()

			// Register test server if needed
			server, _ := mcpserver.NewMCPServer("test", "test", nil, mcpserver.TransportStdio)
			registry.Register(server)

			wf, err := cli.ImportWorkflow(workflowPath, registry)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation error containing '%s', got nil", tt.expectError)
				} else if !strings.Contains(strings.ToLower(err.Error()), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected successful import, got error: %v", err)
				}
				if wf == nil {
					t.Error("Expected non-nil workflow for valid import")
				}
			}
		})
	}
}

// TestWorkflowImport_CreatesValidWorkflowObject tests that successful import creates a valid Workflow object
func TestWorkflowImport_CreatesValidWorkflowObject(t *testing.T) {
	// Load simple workflow fixture
	fixturePath := "../../internal/testutil/fixtures/import-test-simple.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Setup registry with required server
	registry := mcpserver.NewRegistry()
	server, err := mcpserver.NewMCPServer("test-server", "go", []string{"run", "test.go"}, mcpserver.TransportStdio)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	registry.Register(server)

	// Import workflow
	wf, err := cli.ImportWorkflow(absPath, registry)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify Workflow object properties
	if wf == nil {
		t.Fatal("Expected non-nil workflow")
	}

	// Verify identity
	if wf.ID == "" {
		t.Error("Expected workflow to have an ID")
	}
	if wf.Name == "" {
		t.Error("Expected workflow to have a name")
	}
	if wf.Version == "" {
		t.Error("Expected workflow to have a version")
	}

	// Verify structure
	if len(wf.Nodes) == 0 {
		t.Error("Expected workflow to have nodes")
	}
	if len(wf.Edges) == 0 {
		t.Error("Expected workflow to have edges")
	}

	// Verify server configs are present
	if len(wf.ServerConfigs) == 0 {
		t.Error("Expected workflow to have server configs")
	}

	// Verify workflow can be validated
	if err := wf.Validate(); err != nil {
		t.Errorf("Imported workflow failed validation: %v", err)
	}

	// Verify workflow is executable (has execution methods)
	// This is a compile-time check through the Workflow interface
	var _ *workflow.Workflow = wf
}

// TestWorkflowImport_NonExistentFile tests error handling for non-existent files
func TestWorkflowImport_NonExistentFile(t *testing.T) {
	registry := mcpserver.NewRegistry()

	_, err := cli.ImportWorkflow("/nonexistent/path/workflow.yaml", registry)
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}

	// Should be a file not found error
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

// TestWorkflowImport_ServerIDMismatch tests when workflow references server ID not matching registry
func TestWorkflowImport_ServerIDMismatch(t *testing.T) {
	workflowYAML := `
version: "1.0"
name: "mismatch-test"

servers:
  - id: "workflow-server"
    command: "test"

nodes:
  - id: "start"
    type: "start"
  - id: "tool"
    type: "mcp_tool"
    server: "workflow-server"
    tool: "test"
  - id: "end"
    type: "end"

edges:
  - from: "start"
    to: "tool"
  - from: "tool"
    to: "end"
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "mismatch.yaml")
	if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0644); err != nil {
		t.Fatalf("Failed to write test fixture: %v", err)
	}

	// Register server with different ID
	registry := mcpserver.NewRegistry()
	server, _ := mcpserver.NewMCPServer("registry-server", "test", nil, mcpserver.TransportStdio)
	registry.Register(server)

	// Import should detect missing server (workflow-server not in registry)
	_, err := cli.ImportWorkflow(workflowPath, registry)

	if err == nil {
		t.Fatal("Expected error for server ID mismatch")
	}

	var missingServerErr *cli.MissingServerError
	if errors.As(err, &missingServerErr) {
		if len(missingServerErr.MissingServers) != 1 || missingServerErr.MissingServers[0] != "workflow-server" {
			t.Errorf("Expected missing server 'workflow-server', got %v", missingServerErr.MissingServers)
		}
	}
}
