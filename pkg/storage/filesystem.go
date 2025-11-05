package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// FilesystemWorkflowRepository implements WorkflowRepository using filesystem storage.
// Workflows are stored as YAML files in ~/.goflow/workflows/
type FilesystemWorkflowRepository struct {
	baseDir string
}

// NewFilesystemWorkflowRepository creates a new filesystem-based workflow repository.
// It ensures the workflows directory exists.
func NewFilesystemWorkflowRepository() (*FilesystemWorkflowRepository, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".goflow")
	workflowsDir := filepath.Join(baseDir, "workflows")

	// Create directories if they don't exist
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflows directory: %w", err)
	}

	return &FilesystemWorkflowRepository{
		baseDir: workflowsDir,
	}, nil
}

// NewFilesystemWorkflowRepositoryWithPath creates a repository with a custom base directory.
// Useful for testing or custom configurations.
func NewFilesystemWorkflowRepositoryWithPath(baseDir string) (*FilesystemWorkflowRepository, error) {
	workflowsDir := filepath.Join(baseDir, "workflows")

	// Create directories if they don't exist
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflows directory: %w", err)
	}

	return &FilesystemWorkflowRepository{
		baseDir: workflowsDir,
	}, nil
}

// Save persists a workflow to the filesystem as a YAML file.
// The filename is derived from the workflow ID with .yaml extension.
func (r *FilesystemWorkflowRepository) Save(wf *workflow.Workflow) error {
	if wf == nil {
		return fmt.Errorf("cannot save nil workflow")
	}

	if wf.ID == "" {
		return fmt.Errorf("workflow must have an ID")
	}

	// Serialize to YAML
	data, err := yaml.Marshal(wf)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow to YAML: %w", err)
	}

	// Write to file atomically using a temp file + rename
	filePath := r.workflowPath(workflow.WorkflowID(wf.ID))
	tempPath := filePath + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to save workflow file: %w", err)
	}

	return nil
}

// Load retrieves a workflow from the filesystem by its ID.
func (r *FilesystemWorkflowRepository) Load(id workflow.WorkflowID) (*workflow.Workflow, error) {
	if id == "" {
		return nil, fmt.Errorf("workflow ID cannot be empty")
	}

	filePath := r.workflowPath(id)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Deserialize YAML
	var wf workflow.Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &wf, nil
}

// Delete removes a workflow from the filesystem.
func (r *FilesystemWorkflowRepository) Delete(id workflow.WorkflowID) error {
	if id == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	filePath := r.workflowPath(id)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("workflow not found: %s", id)
	}

	// Remove file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete workflow file: %w", err)
	}

	return nil
}

// List returns all workflows stored in the repository.
func (r *FilesystemWorkflowRepository) List() ([]*workflow.Workflow, error) {
	// Read directory entries
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	workflows := make([]*workflow.Workflow, 0)

	for _, entry := range entries {
		// Skip non-YAML files and directories
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Extract workflow ID from filename
		id := strings.TrimSuffix(entry.Name(), ".yaml")

		// Load workflow
		wf, err := r.Load(workflow.WorkflowID(id))
		if err != nil {
			// Log error but continue with other workflows
			// In production, this would use a proper logger
			continue
		}

		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// workflowPath returns the full filesystem path for a workflow ID.
func (r *FilesystemWorkflowRepository) workflowPath(id workflow.WorkflowID) string {
	return filepath.Join(r.baseDir, id.String()+".yaml")
}
