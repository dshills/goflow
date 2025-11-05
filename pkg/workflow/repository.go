package workflow

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	// Save persists a workflow to storage
	Save(workflow *Workflow) error

	// FindByID retrieves a workflow by ID
	// Returns ErrWorkflowNotFound if workflow doesn't exist
	FindByID(id string) (*Workflow, error)

	// FindByName retrieves a workflow by name
	// Returns ErrWorkflowNotFound if workflow doesn't exist
	FindByName(name string) (*Workflow, error)

	// List returns all workflows
	List() ([]*Workflow, error)

	// Delete removes a workflow from storage
	// Returns ErrWorkflowNotFound if workflow doesn't exist
	Delete(id string) error
}
