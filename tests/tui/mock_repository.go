package tui

import (
	"errors"
	"sync"

	"github.com/dshills/goflow/pkg/workflow"
)

// MockRepository implements workflow.WorkflowRepository for testing
// It provides an in-memory, thread-safe implementation with no filesystem dependencies
type MockRepository struct {
	mu        sync.RWMutex
	workflows map[string]*workflow.Workflow // key: workflow ID
	byName    map[string]string             // name -> ID lookup
}

// NewMockRepository creates a new in-memory workflow repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		workflows: make(map[string]*workflow.Workflow),
		byName:    make(map[string]string),
	}
}

// Save persists a workflow to in-memory storage
// Thread-safe for concurrent test execution
func (m *MockRepository) Save(wf *workflow.Workflow) error {
	if wf == nil {
		return errors.New("cannot save nil workflow")
	}

	if wf.ID == "" {
		return errors.New("workflow ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deep copy the workflow to prevent external modifications
	copied := m.deepCopyWorkflow(wf)

	// Check for name conflicts with other workflows (excluding this one)
	if existingID, exists := m.byName[wf.Name]; exists && existingID != wf.ID {
		return errors.New("workflow with this name already exists")
	}

	// Remove old name mapping if workflow name changed
	if existing, exists := m.workflows[wf.ID]; exists {
		if existing.Name != wf.Name {
			delete(m.byName, existing.Name)
		}
	}

	// Save workflow
	m.workflows[wf.ID] = copied
	m.byName[wf.Name] = wf.ID

	return nil
}

// FindByID retrieves a workflow by ID
// Returns ErrWorkflowNotFound if workflow doesn't exist
func (m *MockRepository) FindByID(id string) (*workflow.Workflow, error) {
	if id == "" {
		return nil, errors.New("workflow ID cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	wf, exists := m.workflows[id]
	if !exists {
		return nil, workflow.ErrWorkflowNotFound
	}

	// Return a deep copy to prevent external modifications
	return m.deepCopyWorkflow(wf), nil
}

// FindByName retrieves a workflow by name
// Returns ErrWorkflowNotFound if workflow doesn't exist
func (m *MockRepository) FindByName(name string) (*workflow.Workflow, error) {
	if name == "" {
		return nil, errors.New("workflow name cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.byName[name]
	if !exists {
		return nil, workflow.ErrWorkflowNotFound
	}

	wf, exists := m.workflows[id]
	if !exists {
		// Inconsistent state - should not happen
		return nil, workflow.ErrWorkflowNotFound
	}

	// Return a deep copy to prevent external modifications
	return m.deepCopyWorkflow(wf), nil
}

// List returns all workflows
func (m *MockRepository) List() ([]*workflow.Workflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a list of deep copies
	result := make([]*workflow.Workflow, 0, len(m.workflows))
	for _, wf := range m.workflows {
		result = append(result, m.deepCopyWorkflow(wf))
	}

	return result, nil
}

// Delete removes a workflow from storage
// Returns ErrWorkflowNotFound if workflow doesn't exist
func (m *MockRepository) Delete(id string) error {
	if id == "" {
		return errors.New("workflow ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	wf, exists := m.workflows[id]
	if !exists {
		return workflow.ErrWorkflowNotFound
	}

	// Remove from both maps
	delete(m.workflows, id)
	delete(m.byName, wf.Name)

	return nil
}

// deepCopyWorkflow creates a deep copy of a workflow
// This prevents test isolation issues where one test modifies another's data
func (m *MockRepository) deepCopyWorkflow(wf *workflow.Workflow) *workflow.Workflow {
	if wf == nil {
		return nil
	}

	// Copy basic fields
	copied := &workflow.Workflow{
		ID:          wf.ID,
		Name:        wf.Name,
		Version:     wf.Version,
		Description: wf.Description,
		Metadata:    wf.Metadata,
	}

	// Deep copy metadata tags
	if len(wf.Metadata.Tags) > 0 {
		copied.Metadata.Tags = make([]string, len(wf.Metadata.Tags))
		copy(copied.Metadata.Tags, wf.Metadata.Tags)
	}

	// Deep copy variables
	if len(wf.Variables) > 0 {
		copied.Variables = make([]*workflow.Variable, len(wf.Variables))
		for i, v := range wf.Variables {
			if v != nil {
				varCopy := *v
				copied.Variables[i] = &varCopy
			}
		}
	}

	// Deep copy server configs
	if len(wf.ServerConfigs) > 0 {
		copied.ServerConfigs = make([]*workflow.ServerConfig, len(wf.ServerConfigs))
		for i, s := range wf.ServerConfigs {
			if s != nil {
				serverCopy := *s
				// Deep copy args
				if len(s.Args) > 0 {
					serverCopy.Args = make([]string, len(s.Args))
					copy(serverCopy.Args, s.Args)
				}
				copied.ServerConfigs[i] = &serverCopy
			}
		}
	}

	// Deep copy nodes
	if len(wf.Nodes) > 0 {
		copied.Nodes = make([]workflow.Node, len(wf.Nodes))
		for i, node := range wf.Nodes {
			if node != nil {
				copied.Nodes[i] = m.deepCopyNode(node)
			}
		}
	}

	// Deep copy edges
	if len(wf.Edges) > 0 {
		copied.Edges = make([]*workflow.Edge, len(wf.Edges))
		for i, e := range wf.Edges {
			if e != nil {
				edgeCopy := *e
				copied.Edges[i] = &edgeCopy
			}
		}
	}

	return copied
}

// deepCopyNode creates a deep copy of a node
func (m *MockRepository) deepCopyNode(node workflow.Node) workflow.Node {
	switch n := node.(type) {
	case *workflow.StartNode:
		copied := *n
		return &copied

	case *workflow.EndNode:
		copied := *n
		return &copied

	case *workflow.MCPToolNode:
		copied := *n
		// Deep copy parameters map
		if n.Parameters != nil {
			copied.Parameters = make(map[string]string)
			for k, v := range n.Parameters {
				copied.Parameters[k] = v
			}
		}
		return &copied

	case *workflow.TransformNode:
		copied := *n
		return &copied

	case *workflow.ConditionNode:
		copied := *n
		return &copied

	case *workflow.LoopNode:
		copied := *n
		// Deep copy body slice
		if len(n.Body) > 0 {
			copied.Body = make([]string, len(n.Body))
			copy(copied.Body, n.Body)
		}
		return &copied

	case *workflow.ParallelNode:
		copied := *n
		// Deep copy branches slice of slices
		if len(n.Branches) > 0 {
			copied.Branches = make([][]string, len(n.Branches))
			for i, branch := range n.Branches {
				if len(branch) > 0 {
					copied.Branches[i] = make([]string, len(branch))
					copy(copied.Branches[i], branch)
				}
			}
		}
		return &copied

	default:
		// Unknown node type - return as-is
		return node
	}
}

// Count returns the number of workflows in the repository
// Helper method for testing
func (m *MockRepository) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.workflows)
}

// Clear removes all workflows from the repository
// Helper method for test cleanup
func (m *MockRepository) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workflows = make(map[string]*workflow.Workflow)
	m.byName = make(map[string]string)
}

// Has checks if a workflow exists by ID
// Helper method for testing
func (m *MockRepository) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.workflows[id]
	return exists
}

// HasName checks if a workflow exists by name
// Helper method for testing
func (m *MockRepository) HasName(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.byName[name]
	return exists
}
