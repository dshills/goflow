package tui

import (
	"fmt"
	"strings"

	tui "github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// Common test types and helpers
// Type aliases for types defined in the tui package
type Position = tui.Position

// Edge represents a workflow edge (for test purposes)
type Edge struct {
	From string
	To   string
}

// MockWorkflowRepository implements workflow.WorkflowRepository for testing
// Uses a slice to maintain insertion order for predictable test behavior
type MockWorkflowRepository struct {
	workflows  []*workflow.Workflow
	saveFunc   func(*workflow.Workflow) error
	deleteFunc func(string) error
}

func NewMockWorkflowRepository() *MockWorkflowRepository {
	return &MockWorkflowRepository{
		workflows: make([]*workflow.Workflow, 0),
	}
}

// NewMockWorkflowRepositoryWithWorkflows creates a repository pre-populated with workflows
func NewMockWorkflowRepositoryWithWorkflows(wfs []*workflow.Workflow) *MockWorkflowRepository {
	return &MockWorkflowRepository{
		workflows: append([]*workflow.Workflow{}, wfs...), // Copy slice
	}
}

func (m *MockWorkflowRepository) Save(wf *workflow.Workflow) error {
	if m.saveFunc != nil {
		return m.saveFunc(wf)
	}

	// Update existing or append new
	for i, existing := range m.workflows {
		if existing.ID == wf.ID {
			m.workflows[i] = wf
			return nil
		}
	}
	m.workflows = append(m.workflows, wf)
	return nil
}

func (m *MockWorkflowRepository) FindByID(id string) (*workflow.Workflow, error) {
	for _, wf := range m.workflows {
		if wf.ID == id {
			return wf, nil
		}
	}
	return nil, fmt.Errorf("workflow not found: %s", id)
}

func (m *MockWorkflowRepository) FindByName(name string) (*workflow.Workflow, error) {
	for _, wf := range m.workflows {
		if wf.Name == name {
			return wf, nil
		}
	}
	return nil, fmt.Errorf("workflow not found: %s", name)
}

func (m *MockWorkflowRepository) FindAll() ([]*workflow.Workflow, error) {
	// Return a copy to prevent modifications
	result := make([]*workflow.Workflow, len(m.workflows))
	copy(result, m.workflows)
	return result, nil
}

func (m *MockWorkflowRepository) List() ([]*workflow.Workflow, error) {
	return m.FindAll()
}

func (m *MockWorkflowRepository) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}

	// Find and remove by ID
	for i, wf := range m.workflows {
		if wf.ID == id {
			m.workflows = append(m.workflows[:i], m.workflows[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("workflow not found: %s", id)
}

// screenContainsText checks if the screen buffer contains the given text
func screenContainsText(screen *goterm.Screen, text string) bool {
	width, height := screen.Size()
	for y := 0; y < height; y++ {
		line := ""
		for x := 0; x < width; x++ {
			cell := screen.GetCell(x, y)
			line += string(cell.Ch)
		}
		if strings.Contains(line, text) {
			return true
		}
	}
	return false
}
