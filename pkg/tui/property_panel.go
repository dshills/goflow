package tui

import (
	"fmt"

	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// PropertyPanel manages node property editing with validation
// Note: PropertyPanel struct is defined in workflow_builder.go

// NewPropertyPanel creates a property panel for the given node
func NewPropertyPanel(node workflow.Node) *PropertyPanel {
	panel := &PropertyPanel{
		node:              node,
		fields:            buildFieldsForNode(node),
		editIndex:         0,
		visible:           false,
		validationMessage: "",
	}
	return panel
}

// Show opens the panel for editing
func (p *PropertyPanel) Show() {
	p.visible = true
	p.editIndex = 0
	p.validationMessage = ""
}

// Hide closes the panel
func (p *PropertyPanel) Hide() {
	p.visible = false
}

// IsVisible returns whether panel is open
func (p *PropertyPanel) IsVisible() bool {
	return p.visible
}

// NextField moves focus to next field (with wrap-around)
func (p *PropertyPanel) NextField() {
	if len(p.fields) == 0 {
		return
	}
	p.editIndex = (p.editIndex + 1) % len(p.fields)
	p.validationMessage = ""
}

// PrevField moves focus to previous field (with wrap-around)
func (p *PropertyPanel) PrevField() {
	if len(p.fields) == 0 {
		return
	}
	p.editIndex = (p.editIndex - 1 + len(p.fields)) % len(p.fields)
	p.validationMessage = ""
}

// EditCurrentField enters edit mode for focused field
// This is a no-op in the current implementation since editing happens inline
func (p *PropertyPanel) EditCurrentField() {
	// Field editing is handled by SetFieldValue
}

// SetFieldValue updates the current field value
// Triggers validation on the field
func (p *PropertyPanel) SetFieldValue(value string) error {
	if p.editIndex < 0 || p.editIndex >= len(p.fields) {
		return fmt.Errorf("invalid field index: %d", p.editIndex)
	}

	field := &p.fields[p.editIndex]
	field.value = value

	// Validate the field
	err := field.validate()
	if err != nil {
		p.validationMessage = err.Error()
		return err
	}

	p.validationMessage = ""
	return nil
}

// SaveChanges applies changes to node
// Returns the updated node or error if validation fails
func (p *PropertyPanel) SaveChanges() (*workflow.Node, error) {
	// Validate all fields
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Apply changes to node based on node type
	updatedNode, err := applyFieldsToNode(p.node, p.fields)
	if err != nil {
		return nil, fmt.Errorf("failed to apply changes: %w", err)
	}

	p.node = updatedNode
	return &updatedNode, nil
}

// CancelChanges discards all changes and rebuilds fields from node
func (p *PropertyPanel) CancelChanges() {
	p.fields = buildFieldsForNode(p.node)
	p.editIndex = 0
	p.validationMessage = ""
}

// IsDirty returns true if unsaved changes exist
func (p *PropertyPanel) IsDirty() bool {
	// Compare current field values with node values
	originalFields := buildFieldsForNode(p.node)

	if len(p.fields) != len(originalFields) {
		return true
	}

	for i := range p.fields {
		if p.fields[i].value != originalFields[i].value {
			return true
		}
	}

	return false
}

// Validate runs validation on all fields
func (p *PropertyPanel) Validate() error {
	for i := range p.fields {
		field := &p.fields[i]

		// Check required fields
		if field.required && field.value == "" {
			return fmt.Errorf("field '%s' is required", field.label)
		}

		// Run field validation
		if err := field.validate(); err != nil {
			return fmt.Errorf("field '%s': %w", field.label, err)
		}
	}

	return nil
}

// GetFields returns the property fields (for testing)
func (p *PropertyPanel) GetFields() []propertyField {
	return p.fields
}

// GetEditIndex returns the currently edited field index (for testing)
func (p *PropertyPanel) GetEditIndex() int {
	return p.editIndex
}

// GetValidationMessage returns the current validation message (for testing)
func (p *PropertyPanel) GetValidationMessage() string {
	return p.validationMessage
}

// GetNodeType returns the type of node being edited (for testing)
func (p *PropertyPanel) GetNodeType() string {
	if p.node == nil {
		return ""
	}
	return p.node.Type()
}

// Render draws the panel to screen
func (p *PropertyPanel) Render(screen *goterm.Screen) error {
	if !p.visible {
		return nil
	}

	// TODO: Implement rendering when goterm Screen API is available
	// For now, this is a placeholder
	return nil
}

// buildFieldsForNode creates property fields based on node type
func buildFieldsForNode(node workflow.Node) []propertyField {
	fields := make([]propertyField, 0)

	// If node is nil, return empty fields
	if node == nil {
		return fields
	}

	// Common field: ID (read-only, but included for display)
	fields = append(fields, newPropertyField("Node ID", node.GetID(), "text", true))

	// Type-specific fields
	switch n := node.(type) {
	case *workflow.MCPToolNode:
		fields = append(fields,
			newPropertyField("Server ID", n.ServerID, "text", true),
			newPropertyField("Tool Name", n.ToolName, "text", true),
			newPropertyField("Output Variable", n.OutputVariable, "text", true),
		)

	case *workflow.TransformNode:
		fields = append(fields,
			newPropertyField("Input Variable", n.InputVariable, "text", true),
			newPropertyField("Expression", n.Expression, "expression", true),
			newPropertyField("Output Variable", n.OutputVariable, "text", true),
		)

	case *workflow.ConditionNode:
		fields = append(fields,
			newPropertyField("Condition", n.Condition, "condition", true),
		)

	case *workflow.EndNode:
		fields = append(fields,
			newPropertyField("Return Value", n.ReturnValue, "template", false),
		)

	case *workflow.LoopNode:
		fields = append(fields,
			newPropertyField("Collection", n.Collection, "jsonpath", true),
			newPropertyField("Item Variable", n.ItemVariable, "text", true),
			newPropertyField("Break Condition", n.BreakCondition, "condition", false),
		)

		// StartNode and PassthroughNode have no editable fields beyond ID
	}

	return fields
}

// applyFieldsToNode creates an updated node with field values applied
func applyFieldsToNode(node workflow.Node, fields []propertyField) (workflow.Node, error) {
	// Create a copy of the node with updated values
	switch n := node.(type) {
	case *workflow.MCPToolNode:
		updated := &workflow.MCPToolNode{
			ID:             n.ID,
			ServerID:       getFieldValue(fields, "Server ID"),
			ToolName:       getFieldValue(fields, "Tool Name"),
			OutputVariable: getFieldValue(fields, "Output Variable"),
			Parameters:     n.Parameters, // Keep existing parameters
			Retry:          n.Retry,      // Keep existing retry policy
		}
		return updated, nil

	case *workflow.TransformNode:
		updated := &workflow.TransformNode{
			ID:             n.ID,
			InputVariable:  getFieldValue(fields, "Input Variable"),
			Expression:     getFieldValue(fields, "Expression"),
			OutputVariable: getFieldValue(fields, "Output Variable"),
			Retry:          n.Retry,
		}
		return updated, nil

	case *workflow.ConditionNode:
		updated := &workflow.ConditionNode{
			ID:        n.ID,
			Condition: getFieldValue(fields, "Condition"),
		}
		return updated, nil

	case *workflow.EndNode:
		updated := &workflow.EndNode{
			ID:          n.ID,
			ReturnValue: getFieldValue(fields, "Return Value"),
		}
		return updated, nil

	case *workflow.LoopNode:
		updated := &workflow.LoopNode{
			ID:             n.ID,
			Collection:     getFieldValue(fields, "Collection"),
			ItemVariable:   getFieldValue(fields, "Item Variable"),
			Body:           n.Body, // Keep existing body
			BreakCondition: getFieldValue(fields, "Break Condition"),
		}
		return updated, nil

	case *workflow.StartNode:
		// StartNode has no editable fields
		return n, nil

	case *workflow.PassthroughNode:
		// PassthroughNode has no editable fields
		return n, nil

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// getFieldValue retrieves a field value by label
func getFieldValue(fields []propertyField, label string) string {
	for _, field := range fields {
		if field.label == label {
			return field.value
		}
	}
	return ""
}
