package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

func TestIsValidWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple name", "workflow", true},
		{"valid with hyphen", "my-workflow", true},
		{"valid with underscore", "my_workflow", true},
		{"valid with numbers", "workflow123", true},
		{"valid mixed", "my-workflow_v1", true},
		{"invalid starts with number", "123workflow", false},
		{"invalid starts with hyphen", "-workflow", false},
		{"invalid starts with underscore", "_workflow", false},
		{"invalid special chars", "workflow@123", false},
		{"invalid spaces", "my workflow", false},
		{"valid max length", "a123456789012345678901234567890123456789012345678901234567890123", true},
		{"invalid too long", "a1234567890123456789012345678901234567890123456789012345678901234", false},
		{"invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidWorkflowName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateBasicWorkflow(t *testing.T) {
	wf, err := createBasicWorkflow("test-workflow", "Test description")
	if err != nil {
		t.Fatalf("createBasicWorkflow failed: %v", err)
	}

	// Verify workflow properties
	if wf.Name != "test-workflow" {
		t.Errorf("Name = %q, want %q", wf.Name, "test-workflow")
	}

	if wf.Description != "Test description" {
		t.Errorf("Description = %q, want %q", wf.Description, "Test description")
	}

	// Verify nodes
	if len(wf.Nodes) != 2 {
		t.Fatalf("Nodes count = %d, want 2", len(wf.Nodes))
	}

	// Check for start and end nodes
	hasStart := false
	hasEnd := false
	for _, node := range wf.Nodes {
		if node.Type() == "start" {
			hasStart = true
		}
		if node.Type() == "end" {
			hasEnd = true
		}
	}

	if !hasStart {
		t.Error("Missing start node")
	}
	if !hasEnd {
		t.Error("Missing end node")
	}

	// Verify edge
	if len(wf.Edges) != 1 {
		t.Fatalf("Edges count = %d, want 1", len(wf.Edges))
	}

	edge := wf.Edges[0]
	if edge.FromNodeID != "start" || edge.ToNodeID != "end" {
		t.Errorf("Edge = %s -> %s, want start -> end", edge.FromNodeID, edge.ToNodeID)
	}

	// Validate workflow
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

func TestCreateETLTemplate(t *testing.T) {
	wf, err := createETLTemplate("etl-test", "ETL test workflow")
	if err != nil {
		t.Fatalf("createETLTemplate failed: %v", err)
	}

	// Verify workflow has ETL nodes
	expectedNodes := []string{"start", "extract", "transform", "load", "end"}
	if len(wf.Nodes) != len(expectedNodes) {
		t.Errorf("Nodes count = %d, want %d", len(wf.Nodes), len(expectedNodes))
	}

	// Check node types
	nodeIDs := make(map[string]bool)
	for _, node := range wf.Nodes {
		nodeIDs[node.GetID()] = true
	}

	for _, expectedID := range expectedNodes {
		if !nodeIDs[expectedID] {
			t.Errorf("Missing expected node: %s", expectedID)
		}
	}

	// Verify edges form a pipeline
	if len(wf.Edges) != 4 {
		t.Errorf("Edges count = %d, want 4", len(wf.Edges))
	}

	// Add mock server for validation (since auto-config was removed for security)
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:        "data-server",
			Name:      "Mock Data Server",
			Command:   "echo",
			Args:      []string{"mock"},
			Transport: "stdio",
		},
	}

	// Validate workflow
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

func TestCreateAPIIntegrationTemplate(t *testing.T) {
	wf, err := createAPIIntegrationTemplate("api-test", "API test workflow")
	if err != nil {
		t.Fatalf("createAPIIntegrationTemplate failed: %v", err)
	}

	// Verify workflow has API nodes
	expectedNodes := []string{"start", "fetch_api", "process_response", "end"}
	if len(wf.Nodes) != len(expectedNodes) {
		t.Errorf("Nodes count = %d, want %d", len(wf.Nodes), len(expectedNodes))
	}

	// Add mock server for validation (since auto-config was removed for security)
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:        "http-server",
			Name:      "Mock HTTP Server",
			Command:   "echo",
			Args:      []string{"mock"},
			Transport: "stdio",
		},
	}

	// Validate workflow
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

func TestCreateBatchProcessingTemplate(t *testing.T) {
	wf, err := createBatchProcessingTemplate("batch-test", "Batch test workflow")
	if err != nil {
		t.Fatalf("createBatchProcessingTemplate failed: %v", err)
	}

	// Verify workflow has batch processing nodes
	expectedNodes := []string{"start", "process_batch", "end"}
	if len(wf.Nodes) != len(expectedNodes) {
		t.Errorf("Nodes count = %d, want %d", len(wf.Nodes), len(expectedNodes))
	}

	// Add mock server for validation (since auto-config was removed for security)
	wf.ServerConfigs = []*workflow.ServerConfig{
		{
			ID:        "batch-server",
			Name:      "Mock Batch Server",
			Command:   "echo",
			Args:      []string{"mock"},
			Transport: "stdio",
		},
	}

	// Validate workflow
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}

func TestCreateWorkflowFromTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    WorkflowTemplate
		expectError bool
	}{
		{"basic template", TemplateBasic, false},
		{"etl template", TemplateETL, false},
		{"api-integration template", TemplateAPIIntegration, false},
		{"batch-processing template", TemplateBatchProcess, false},
		{"unknown template", WorkflowTemplate("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := createWorkflowFromTemplate("test", "test desc", tt.template)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if wf == nil {
					t.Error("Expected workflow, got nil")
				}
			}
		})
	}
}

func TestWorkflowToYAMLMap(t *testing.T) {
	wf, err := createBasicWorkflow("test-workflow", "Test description")
	if err != nil {
		t.Fatalf("createBasicWorkflow failed: %v", err)
	}

	yamlMap := workflowToYAMLMap(wf)

	// Verify essential fields
	if yamlMap["name"] != "test-workflow" {
		t.Errorf("name = %v, want test-workflow", yamlMap["name"])
	}

	if yamlMap["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", yamlMap["version"])
	}

	// Verify nodes
	nodes, ok := yamlMap["nodes"].([]map[string]interface{})
	if !ok {
		t.Fatal("nodes not a slice of maps")
	}

	if len(nodes) != 2 {
		t.Errorf("nodes count = %d, want 2", len(nodes))
	}

	// Verify edges
	edges, ok := yamlMap["edges"].([]map[string]interface{})
	if !ok {
		t.Fatal("edges not a slice of maps")
	}

	if len(edges) != 1 {
		t.Errorf("edges count = %d, want 1", len(edges))
	}
}

func TestInitCommand_Integration(t *testing.T) {
	// Skip if running in CI without proper filesystem access
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "goflow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test config directory
	oldConfigDir := GlobalConfig.ConfigDir
	GlobalConfig.ConfigDir = tmpDir
	defer func() { GlobalConfig.ConfigDir = oldConfigDir }()

	// Create workflows directory
	workflowsDir := filepath.Join(tmpDir, "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Test creating a workflow
	wf, err := createBasicWorkflow("test-integration", "Test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Verify workflow is valid
	if err := wf.Validate(); err != nil {
		t.Errorf("Workflow validation failed: %v", err)
	}
}
