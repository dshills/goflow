package workflow

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	// Save persists a workflow to storage
	Save(workflow *Workflow) error

	// Load retrieves a workflow by ID
	Load(id WorkflowID) (*Workflow, error)

	// Delete removes a workflow from storage
	Delete(id WorkflowID) error

	// List returns all workflows
	List() ([]*Workflow, error)
}
