package mcpserver

// Tool represents an MCP tool discovered from a server
type Tool struct {
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	InputSchema  *ToolSchema `json:"inputSchema,omitempty"`
	OutputSchema *ToolSchema `json:"outputSchema,omitempty"`
}

// NewTool creates a new Tool
func NewTool(name, description string) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
	}
}

// Validate checks if the tool is valid
func (t *Tool) Validate() error {
	if t.Name == "" {
		return NewValidationError("tool: name cannot be empty")
	}

	if t.InputSchema != nil {
		if err := t.InputSchema.Validate(); err != nil {
			return err
		}
	}

	if t.OutputSchema != nil {
		if err := t.OutputSchema.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// WithInputSchema sets the input schema for the tool
func (t *Tool) WithInputSchema(schema *ToolSchema) *Tool {
	t.InputSchema = schema
	return t
}

// WithOutputSchema sets the output schema for the tool
func (t *Tool) WithOutputSchema(schema *ToolSchema) *Tool {
	t.OutputSchema = schema
	return t
}
