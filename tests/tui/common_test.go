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
// This is a combined version that supports both map-based and slice-based storage
type MockWorkflowRepository struct {
	workflows  map[string]*workflow.Workflow
	saveFunc   func(*workflow.Workflow) error
	deleteFunc func(string) error
}

func NewMockWorkflowRepository() *MockWorkflowRepository {
	return &MockWorkflowRepository{
		workflows: make(map[string]*workflow.Workflow),
	}
}

// NewMockWorkflowRepositoryWithWorkflows creates a repository pre-populated with workflows
func NewMockWorkflowRepositoryWithWorkflows(wfs []*workflow.Workflow) *MockWorkflowRepository {
	repo := NewMockWorkflowRepository()
	for _, wf := range wfs {
		repo.workflows[wf.Name] = wf
	}
	return repo
}

func (m *MockWorkflowRepository) Save(wf *workflow.Workflow) error {
	if m.saveFunc != nil {
		return m.saveFunc(wf)
	}
	m.workflows[wf.Name] = wf
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
	if wf, ok := m.workflows[name]; ok {
		return wf, nil
	}
	return nil, fmt.Errorf("workflow not found: %s", name)
}

func (m *MockWorkflowRepository) FindAll() ([]*workflow.Workflow, error) {
	result := make([]*workflow.Workflow, 0, len(m.workflows))
	for _, wf := range m.workflows {
		result = append(result, wf)
	}
	return result, nil
}

func (m *MockWorkflowRepository) List() ([]*workflow.Workflow, error) {
	return m.FindAll()
}

func (m *MockWorkflowRepository) Delete(name string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(name)
	}
	delete(m.workflows, name)
	return nil
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
