package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// TestExport_WorkflowWithInlineCredentials tests that inline credentials are stripped from exported YAML
func TestExport_WorkflowWithInlineCredentials(t *testing.T) {
	// Create a workflow with inline credentials in server configs
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow with credentials")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add server config with inline credentials (API key in env)
	serverConfig := &workflow.ServerConfig{
		ID:      "api-server",
		Name:    "API Server",
		Command: "mcp-api-server",
		Args:    []string{"--host", "api.example.com"},
		Env: map[string]string{
			"API_KEY":    "sk-1234567890abcdef", // Should be stripped
			"API_SECRET": "secret_abc123xyz",    // Should be stripped
			"API_TOKEN":  "token_9876543210",    // Should be stripped
			"PASSWORD":   "mypassword123",       // Should be stripped
			"HOST":       "api.example.com",     // Should be preserved
			"PORT":       "8080",                // Should be preserved
		},
	}
	wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)

	// Add basic nodes to make workflow valid
	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)

	edge := &workflow.Edge{
		ID:         "e1",
		FromNodeID: "start",
		ToNodeID:   "end",
	}
	wf.AddEdge(edge)

	// Export workflow - this should strip credentials
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse exported YAML back to check credentials were stripped
	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	// Check that servers section exists
	servers, ok := exportedData["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		t.Fatal("Exported YAML missing servers section")
	}

	serverData := servers[0].(map[string]interface{})
	env, ok := serverData["env"].(map[string]interface{})
	if !ok {
		t.Fatal("Server config missing env section")
	}

	// Verify sensitive credentials are removed
	sensitiveKeys := []string{"API_KEY", "API_SECRET", "API_TOKEN", "PASSWORD"}
	for _, key := range sensitiveKeys {
		if _, exists := env[key]; exists {
			t.Errorf("Sensitive credential %s was not stripped from export", key)
		}
	}

	// Verify non-sensitive values are preserved
	if host, ok := env["HOST"].(string); !ok || host != "api.example.com" {
		t.Error("Non-sensitive env var HOST was incorrectly removed or modified")
	}
	if port, ok := env["PORT"].(string); !ok || port != "8080" {
		t.Error("Non-sensitive env var PORT was incorrectly removed or modified")
	}

	// Verify placeholder comment or instruction is added
	yamlString := string(exportedYAML)
	if !strings.Contains(yamlString, "# CREDENTIAL") && !strings.Contains(yamlString, "credential") {
		t.Error("Export should include credential placeholder comments or instructions")
	}
}

// TestExport_WorkflowWithCredentialReferences tests that credential references are preserved with placeholders
func TestExport_WorkflowWithCredentialReferences(t *testing.T) {
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow with credential references")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Server config with credential reference (not inline credentials)
	serverConfig := &workflow.ServerConfig{
		ID:            "secure-server",
		Name:          "Secure API Server",
		Command:       "mcp-secure-server",
		CredentialRef: "keyring://api-credentials/production",
	}
	wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)

	// Add basic nodes
	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)
	edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
	wf.AddEdge(edge)

	// Export workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	servers := exportedData["servers"].([]interface{})
	serverData := servers[0].(map[string]interface{})

	// Credential reference should be replaced with placeholder
	credRef, ok := serverData["credential_ref"].(string)
	if !ok {
		t.Fatal("credential_ref field missing from exported server config")
	}

	// Should be a placeholder, not the actual keyring reference
	if credRef == "keyring://api-credentials/production" {
		t.Error("Credential reference was not replaced with placeholder")
	}

	expectedPlaceholder := "<CREDENTIAL_REF_REQUIRED>"
	if credRef != expectedPlaceholder {
		t.Errorf("Expected credential_ref placeholder %q, got %q", expectedPlaceholder, credRef)
	}
}

// TestExport_WorkflowWithNoCredentials tests that workflows without credentials are exported unchanged
func TestExport_WorkflowWithNoCredentials(t *testing.T) {
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow without credentials")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Server config with no credentials
	serverConfig := &workflow.ServerConfig{
		ID:      "local-server",
		Name:    "Local Server",
		Command: "mcp-local-server",
		Args:    []string{"--mode", "local"},
		Env: map[string]string{
			"MODE":      "development",
			"LOG_LEVEL": "debug",
		},
	}
	wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)

	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)
	edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
	wf.AddEdge(edge)

	// Export workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	servers := exportedData["servers"].([]interface{})
	serverData := servers[0].(map[string]interface{})
	env := serverData["env"].(map[string]interface{})

	// Non-credential env vars should be preserved
	if mode, ok := env["MODE"].(string); !ok || mode != "development" {
		t.Error("Non-credential env var MODE was incorrectly modified")
	}
	if logLevel, ok := env["LOG_LEVEL"].(string); !ok || logLevel != "debug" {
		t.Error("Non-credential env var LOG_LEVEL was incorrectly modified")
	}
}

// TestExport_ValidYAML tests that exported YAML is valid and parseable
func TestExport_ValidYAML(t *testing.T) {
	// Load an existing workflow fixture
	fixturePath := "../../internal/testutil/fixtures/simple-workflow.yaml"
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	wf, err := workflow.ParseFile(absPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Export the workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify exported YAML is valid by parsing it
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &yamlData); err != nil {
		t.Fatalf("Exported YAML is invalid: %v", err)
	}

	// Check required fields exist
	requiredFields := []string{"version", "name", "nodes", "edges"}
	for _, field := range requiredFields {
		if _, ok := yamlData[field]; !ok {
			t.Errorf("Exported YAML missing required field: %s", field)
		}
	}

	// Verify YAML is well-formed (no syntax errors)
	if len(exportedYAML) == 0 {
		t.Error("Exported YAML is empty")
	}

	// Check that it doesn't have malformed YAML indicators
	yamlString := string(exportedYAML)
	if strings.Contains(yamlString, "!!binary") {
		t.Error("Exported YAML contains unexpected binary data")
	}
}

// TestExport_WorkflowStructurePreserved tests that workflow structure and logic are preserved
func TestExport_WorkflowStructurePreserved(t *testing.T) {
	// Create a workflow with various node types
	wf, err := workflow.NewWorkflow("complex-workflow", "Complex workflow for export testing")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add variables
	wf.AddVariable(&workflow.Variable{
		Name:         "input_data",
		Type:         "string",
		DefaultValue: "test",
	})
	wf.AddVariable(&workflow.Variable{
		Name: "result",
		Type: "string",
	})

	// Add nodes of different types
	startNode := &workflow.StartNode{ID: "start"}
	mcpNode := &workflow.MCPToolNode{
		ID:             "fetch_data",
		ServerID:       "api-server",
		ToolName:       "fetch",
		OutputVariable: "fetched_data",
		Parameters: map[string]string{
			"endpoint": "/api/data",
		},
	}
	transformNode := &workflow.TransformNode{
		ID:             "transform_data",
		InputVariable:  "fetched_data",
		Expression:     "$.items[0]",
		OutputVariable: "result",
	}
	endNode := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "${result}",
	}

	wf.AddNode(startNode)
	wf.AddNode(mcpNode)
	wf.AddNode(transformNode)
	wf.AddNode(endNode)

	// Add edges
	wf.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "fetch_data"})
	wf.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "fetch_data", ToNodeID: "transform_data"})
	wf.AddEdge(&workflow.Edge{ID: "e3", FromNodeID: "transform_data", ToNodeID: "end"})

	// Export workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Parse exported YAML
	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	// Verify variables are preserved
	variables := exportedData["variables"].([]interface{})
	if len(variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(variables))
	}

	// Verify nodes are preserved with correct types
	nodes := exportedData["nodes"].([]interface{})
	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(nodes))
	}

	// Check node types
	nodeTypes := make(map[string]bool)
	for _, n := range nodes {
		node := n.(map[string]interface{})
		nodeType := node["type"].(string)
		nodeTypes[nodeType] = true
	}

	expectedTypes := []string{"start", "mcp_tool", "transform", "end"}
	for _, expectedType := range expectedTypes {
		if !nodeTypes[expectedType] {
			t.Errorf("Expected node type %s not found in export", expectedType)
		}
	}

	// Verify edges are preserved
	edges := exportedData["edges"].([]interface{})
	if len(edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(edges))
	}

	// Verify MCP tool node parameters are preserved (non-sensitive)
	for _, n := range nodes {
		node := n.(map[string]interface{})
		if node["type"].(string) == "mcp_tool" {
			params, ok := node["parameters"].(map[string]interface{})
			if !ok {
				t.Error("MCP tool node missing parameters")
			}
			if endpoint, ok := params["endpoint"].(string); !ok || endpoint != "/api/data" {
				t.Error("MCP tool node parameters not preserved correctly")
			}
		}
	}
}

// TestExport_NonSensitiveDataPreserved tests that non-sensitive configuration is preserved
func TestExport_NonSensitiveDataPreserved(t *testing.T) {
	wf, err := workflow.NewWorkflow("test-workflow", "Test non-sensitive data preservation")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add server with mix of sensitive and non-sensitive env vars
	serverConfig := &workflow.ServerConfig{
		ID:        "mixed-server",
		Name:      "Mixed Environment Server",
		Command:   "mcp-server",
		Args:      []string{"--verbose", "--config", "/etc/config.json"},
		Transport: "stdio",
		Env: map[string]string{
			"DATABASE_URL": "postgresql://user:password@localhost/db", // Sensitive
			"REDIS_HOST":   "localhost",                               // Not sensitive
			"REDIS_PORT":   "6379",                                    // Not sensitive
			"OAUTH_TOKEN":  "oauth_abc123",                            // Sensitive
			"SERVICE_NAME": "my-service",                              // Not sensitive
			"LOG_LEVEL":    "info",                                    // Not sensitive
		},
	}
	wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)

	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)
	edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
	wf.AddEdge(edge)

	// Export workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	servers := exportedData["servers"].([]interface{})
	serverData := servers[0].(map[string]interface{})

	// Verify command and args are preserved
	if cmd, ok := serverData["command"].(string); !ok || cmd != "mcp-server" {
		t.Error("Server command not preserved")
	}

	args := serverData["args"].([]interface{})
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}

	// Verify transport is preserved
	if transport, ok := serverData["transport"].(string); !ok || transport != "stdio" {
		t.Error("Server transport not preserved")
	}

	env := serverData["env"].(map[string]interface{})

	// Sensitive keys should be removed
	sensitiveKeys := []string{"DATABASE_URL", "OAUTH_TOKEN"}
	for _, key := range sensitiveKeys {
		if _, exists := env[key]; exists {
			t.Errorf("Sensitive env var %s was not stripped", key)
		}
	}

	// Non-sensitive keys should be preserved
	nonSensitiveChecks := map[string]string{
		"REDIS_HOST":   "localhost",
		"REDIS_PORT":   "6379",
		"SERVICE_NAME": "my-service",
		"LOG_LEVEL":    "info",
	}
	for key, expectedValue := range nonSensitiveChecks {
		if val, ok := env[key].(string); !ok || val != expectedValue {
			t.Errorf("Non-sensitive env var %s not preserved correctly (expected %q, got %q)", key, expectedValue, val)
		}
	}
}

// TestExport_RoundTripImport tests that exported YAML can be successfully re-imported
func TestExport_RoundTripImport(t *testing.T) {
	// Create original workflow
	original, err := workflow.NewWorkflow("roundtrip-test", "Test export/import round-trip")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add some content
	original.AddVariable(&workflow.Variable{
		Name:         "test_var",
		Type:         "string",
		DefaultValue: "test_value",
	})
	original.AddVariable(&workflow.Variable{
		Name: "result",
		Type: "string",
	})

	startNode := &workflow.StartNode{ID: "start"}
	transformNode := &workflow.TransformNode{
		ID:             "transform",
		InputVariable:  "test_var",
		Expression:     "$.value", // Use JSONPath instead of template to avoid validation issues
		OutputVariable: "result",
	}
	endNode := &workflow.EndNode{ID: "end"}

	original.AddNode(startNode)
	original.AddNode(transformNode)
	original.AddNode(endNode)

	original.AddEdge(&workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "transform"})
	original.AddEdge(&workflow.Edge{ID: "e2", FromNodeID: "transform", ToNodeID: "end"})

	// Export workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(original)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Re-import exported YAML
	reimported, err := workflow.Parse(exportedYAML)
	if err != nil {
		t.Fatalf("Failed to re-import exported workflow: %v", err)
	}

	// Verify key properties match
	if reimported.Name != original.Name {
		t.Errorf("Name mismatch: expected %q, got %q", original.Name, reimported.Name)
	}

	if len(reimported.Variables) != len(original.Variables) {
		t.Errorf("Variable count mismatch: expected %d, got %d", len(original.Variables), len(reimported.Variables))
	}

	if len(reimported.Nodes) != len(original.Nodes) {
		t.Errorf("Node count mismatch: expected %d, got %d", len(original.Nodes), len(reimported.Nodes))
	}

	if len(reimported.Edges) != len(original.Edges) {
		t.Errorf("Edge count mismatch: expected %d, got %d", len(original.Edges), len(reimported.Edges))
	}

	// Verify reimported workflow validates
	if err := reimported.Validate(); err != nil {
		t.Errorf("Reimported workflow failed validation: %v", err)
	}
}

// TestExport_VariousCredentialPatterns tests detection of various credential patterns
func TestExport_VariousCredentialPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		envKey      string
		envValue    string
		shouldStrip bool
	}{
		// Should be stripped
		{"api_key", "API_KEY", "sk-abc123", true},
		{"secret_key", "SECRET_KEY", "secret_xyz", true},
		{"access_token", "ACCESS_TOKEN", "token_123", true},
		{"password", "PASSWORD", "mypass123", true},
		{"auth_token", "AUTH_TOKEN", "auth_abc", true},
		{"bearer_token", "BEARER_TOKEN", "bearer_xyz", true},
		{"client_secret", "CLIENT_SECRET", "client_secret_123", true},
		{"private_key", "PRIVATE_KEY", "-----BEGIN PRIVATE KEY-----", true},
		{"credential", "CREDENTIAL", "cred_123", true},
		{"passphrase", "PASSPHRASE", "my_passphrase", true},

		// Should NOT be stripped
		{"host", "HOST", "localhost", false},
		{"port", "PORT", "8080", false},
		{"service_name", "SERVICE_NAME", "my-service", false},
		{"log_level", "LOG_LEVEL", "debug", false},
		{"timeout", "TIMEOUT", "30", false},
		{"max_retries", "MAX_RETRIES", "3", false},
		{"endpoint", "ENDPOINT", "/api/v1", false},
		{"region", "REGION", "us-east-1", false},
		{"namespace", "NAMESPACE", "default", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wf, err := workflow.NewWorkflow("test-"+tc.name, "Test credential pattern detection")
			if err != nil {
				t.Fatalf("Failed to create workflow: %v", err)
			}

			serverConfig := &workflow.ServerConfig{
				ID:      "test-server",
				Command: "test-command",
				Env: map[string]string{
					tc.envKey: tc.envValue,
				},
			}
			wf.ServerConfigs = append(wf.ServerConfigs, serverConfig)

			startNode := &workflow.StartNode{ID: "start"}
			endNode := &workflow.EndNode{ID: "end"}
			wf.AddNode(startNode)
			wf.AddNode(endNode)
			edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
			wf.AddEdge(edge)

			// Export workflow
			// This will FAIL until workflow.Export() is implemented
			exportedYAML, err := workflow.Export(wf)
			if err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			var exportedData map[string]interface{}
			if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
				t.Fatalf("Failed to parse exported YAML: %v", err)
			}

			servers := exportedData["servers"].([]interface{})
			serverData := servers[0].(map[string]interface{})

			// Check if env field exists (it may be omitted if all vars were stripped)
			envData, envExists := serverData["env"]
			exists := false
			if envExists && envData != nil {
				if env, ok := envData.(map[string]interface{}); ok {
					_, exists = env[tc.envKey]
				}
			}

			if tc.shouldStrip && exists {
				t.Errorf("Expected env var %s to be stripped, but it exists in export", tc.envKey)
			}

			if !tc.shouldStrip && !exists {
				t.Errorf("Expected env var %s to be preserved, but it was stripped", tc.envKey)
			}

			if !tc.shouldStrip && exists {
				env := envData.(map[string]interface{})
				if val, ok := env[tc.envKey].(string); !ok || val != tc.envValue {
					t.Errorf("Expected env var %s to have value %q, got %q", tc.envKey, tc.envValue, val)
				}
			}
		})
	}
}

// TestExport_ToFile tests exporting workflow to a file
func TestExport_ToFile(t *testing.T) {
	wf, err := workflow.NewWorkflow("file-export-test", "Test file export")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)
	edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
	wf.AddEdge(edge)

	// Create temp file for export
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "exported-workflow.yaml")

	// Export to file
	// This will FAIL until workflow.ExportFile() is implemented
	err = workflow.ExportFile(wf, exportPath)
	if err != nil {
		t.Fatalf("ExportFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatal("Exported file was not created")
	}

	// Read and verify file content
	content, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(content, &exportedData); err != nil {
		t.Fatalf("Exported file contains invalid YAML: %v", err)
	}

	// Verify workflow name
	if name, ok := exportedData["name"].(string); !ok || name != "file-export-test" {
		t.Error("Exported file does not contain correct workflow name")
	}
}

// TestExport_NilWorkflow tests that exporting nil workflow returns error
func TestExport_NilWorkflow(t *testing.T) {
	// This will FAIL until workflow.Export() is implemented
	_, err := workflow.Export(nil)
	if err == nil {
		t.Error("Expected error when exporting nil workflow, got nil")
	}

	expectedError := "workflow cannot be nil"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain %q, got %q", expectedError, err.Error())
	}
}

// TestExport_EmptyWorkflow tests exporting a minimal valid workflow
func TestExport_EmptyWorkflow(t *testing.T) {
	wf, err := workflow.NewWorkflow("empty-workflow", "Minimal workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add minimal required nodes for valid workflow
	startNode := &workflow.StartNode{ID: "start"}
	endNode := &workflow.EndNode{ID: "end"}
	wf.AddNode(startNode)
	wf.AddNode(endNode)
	edge := &workflow.Edge{ID: "e1", FromNodeID: "start", ToNodeID: "end"}
	wf.AddEdge(edge)

	// Export should succeed even with minimal workflow
	// This will FAIL until workflow.Export() is implemented
	exportedYAML, err := workflow.Export(wf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(exportedYAML) == 0 {
		t.Error("Exported YAML is empty")
	}

	// Verify it's valid YAML
	var exportedData map[string]interface{}
	if err := yaml.Unmarshal(exportedYAML, &exportedData); err != nil {
		t.Fatalf("Exported YAML is invalid: %v", err)
	}
}
