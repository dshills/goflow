package mcpserver

import "encoding/json"

// ToolSchema represents a JSON Schema for tool input/output validation
type ToolSchema struct {
	Type       string                 `json:"type,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// NewToolSchema creates a new ToolSchema
func NewToolSchema(schemaType string) *ToolSchema {
	return &ToolSchema{
		Type:       schemaType,
		Properties: make(map[string]interface{}),
		Required:   []string{},
	}
}

// AddProperty adds a property to the schema
func (ts *ToolSchema) AddProperty(name string, property interface{}) {
	if ts.Properties == nil {
		ts.Properties = make(map[string]interface{})
	}
	ts.Properties[name] = property
}

// AddRequired marks a property as required
func (ts *ToolSchema) AddRequired(name string) {
	ts.Required = append(ts.Required, name)
}

// Validate checks if the schema is valid
func (ts *ToolSchema) Validate() error {
	if ts.Type == "" {
		return NewValidationError("tool schema: type cannot be empty")
	}
	return nil
}

// MarshalJSON implements json.Marshaler
func (ts *ToolSchema) MarshalJSON() ([]byte, error) {
	type Alias ToolSchema
	return json.Marshal((*Alias)(ts))
}

// UnmarshalJSON implements json.Unmarshaler
func (ts *ToolSchema) UnmarshalJSON(data []byte) error {
	type Alias ToolSchema
	aux := (*Alias)(ts)
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	return nil
}
