package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/cli"
)

// TestRunCommand_Basic tests basic run command parsing and execution
func TestRunCommand_Basic(t *testing.T) {
	// Create a temporary workflow file
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	// Capture stdout
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Set arguments - use workflow name, not path
	cmd.SetArgs([]string{"test-workflow"})

	// Execute command
	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	// Verify output contains success message
	output := stdout.String()
	if !strings.Contains(output, "success") && !strings.Contains(output, "completed") {
		t.Errorf("Expected success message in output, got: %s", output)
	}
}

// TestRunCommand_WithInputFile tests run command with input variable file
func TestRunCommand_WithInputFile(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")
	inputPath := filepath.Join(tmpDir, "input.json")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
variables:
  - name: "test_var"
    type: "string"
    required: true
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	inputJSON := `{"test_var": "test value"}`
	err = os.WriteFile(inputPath, []byte(inputJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write input file: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow", "--input", inputPath})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution with input file, got error: %v", err)
	}
}

// TestRunCommand_WithInlineVariables tests run command with inline variable values
func TestRunCommand_WithInlineVariables(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
variables:
  - name: "var1"
    type: "string"
  - name: "var2"
    type: "string"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{
		"test-workflow",
		"--var", "var1=value1",
		"--var", "var2=value2",
	})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution with inline variables, got error: %v", err)
	}
}

// TestRunCommand_DebugMode tests run command with debug flag
func TestRunCommand_DebugMode(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow", "--debug"})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution in debug mode, got error: %v", err)
	}

	// In debug mode, expect more verbose output
	output := stdout.String()
	if len(output) == 0 {
		t.Error("Expected debug output, got empty string")
	}
}

// TestRunCommand_WatchMode tests run command with watch flag
func TestRunCommand_WatchMode(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Use context with timeout for watch mode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd.SetArgs([]string{"test-workflow", "--watch"})

	// Watch mode should start execution
	// For testing, we'd cancel after a short time
	go func() {
		_ = cmd.ExecuteContext(ctx)
	}()

	// Cancel watch mode
	cancel()
}

// TestRunCommand_NonExistentWorkflow tests error handling for missing workflow file
func TestRunCommand_NonExistentWorkflow(t *testing.T) {
	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"/nonexistent/workflow.yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for non-existent workflow, got nil")
	}

	// Verify error message
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "not found") && !strings.Contains(errOutput, "no such file") {
		t.Errorf("Expected file not found error, got: %s", errOutput)
	}
}

// TestRunCommand_InvalidWorkflow tests error handling for invalid workflow YAML
func TestRunCommand_InvalidWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "invalid-workflow.yaml")

	invalidYAML := `
version: "1.0"
name: "invalid-workflow"
# Missing required nodes field
`
	err = os.WriteFile(workflowPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"invalid-workflow"})

	err = cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid workflow, got nil")
	}
}

// TestRunCommand_MissingRequiredVariable tests error for missing required variables
func TestRunCommand_MissingRequiredVariable(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
variables:
  - name: "required_var"
    type: "string"
    required: true
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow"})

	err = cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing required variable, got nil")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "required") && !strings.Contains(errOutput, "missing") {
		t.Errorf("Expected missing required variable error, got: %s", errOutput)
	}
}

// TestRunCommand_OutputFormatJSON tests run command with JSON output format
func TestRunCommand_OutputFormatJSON(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
    return: "test result"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow", "--output", "json"})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}

	// Verify output is valid JSON
	output := stdout.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Errorf("Expected JSON output, got: %s", output)
	}
}

// TestRunCommand_StdinWorkflow tests running workflow from stdin
func TestRunCommand_StdinWorkflow(t *testing.T) {
	workflowYAML := `
version: "1.0"
name: "stdin-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(workflowYAML))

	cmd.SetArgs([]string{"--stdin"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution from stdin, got error: %v", err)
	}
}

// TestRunCommand_TimeoutFlag tests run command with timeout
func TestRunCommand_TimeoutFlag(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow", "--timeout", "30"})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Expected successful execution with timeout, got error: %v", err)
	}
}

// TestRunCommand_InvalidInputFormat tests error for invalid input file format
func TestRunCommand_InvalidInputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	workflowPath := filepath.Join(workflowsDir, "test-workflow.yaml")
	inputPath := filepath.Join(tmpDir, "invalid-input.json")

	workflowYAML := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "end"
`
	err = os.WriteFile(workflowPath, []byte(workflowYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	invalidJSON := `{"invalid": json syntax`
	err = os.WriteFile(inputPath, []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid input: %v", err)
	}

	// Set config directory to temp dir
	os.Setenv("GOFLOW_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("GOFLOW_CONFIG_DIR")

	// This should fail because cli.NewRunCommand doesn't exist yet
	cmd := cli.NewRunCommand()

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"test-workflow", "--input", inputPath})

	err = cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid input format, got nil")
	}
}
