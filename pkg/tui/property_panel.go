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
// x, y: top-left corner position
// width, height: available space
func (p *PropertyPanel) Render(screen interface{}, x, y, width, height int) error {
	if !p.visible || p.node == nil {
		return nil
	}

	// Type assert to screen interface
	type Screen interface {
		SetCell(cellX, cellY int, cell interface{})
		Size() (int, int)
	}

	scr, ok := screen.(Screen)
	if !ok {
		return fmt.Errorf("invalid screen type")
	}

	// Colors
	fgColor := goterm.ColorRGB(255, 255, 255)      // White text
	bgColor := goterm.ColorRGB(30, 30, 30)         // Dark background
	selectedBgColor := goterm.ColorRGB(58, 58, 58) // Gray background for selected
	borderFg := goterm.ColorRGB(136, 136, 136)     // Gray border
	errorFg := goterm.ColorRGB(255, 100, 100)      // Light red for errors
	successFg := goterm.ColorRGB(100, 255, 100)    // Light green for valid fields

	// Draw border
	// Top border
	for i := 0; i < width; i++ {
		char := '─'
		switch i {
		case 0:
			char = '┌'
		case width - 1:
			char = '┐'
		}
		cell := goterm.NewCell(char, borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+i, y, cell)
	}

	// Title: "Properties: <NodeType>"
	title := fmt.Sprintf("Properties: %s", p.node.Type())
	titlePadding := (width - 2 - len(title)) / 2
	for i, ch := range title {
		if i+titlePadding+1 < width-1 {
			cell := goterm.NewCell(ch, fgColor, bgColor, goterm.StyleBold)
			scr.SetCell(x+1+titlePadding+i, y, cell)
		}
	}

	// Middle rows - show property fields
	currentY := y + 1
	for i, field := range p.fields {
		if currentY >= y+height-3 { // Leave room for validation message and bottom border
			break
		}

		// Left border
		cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x, currentY, cell)

		// Determine background for this row
		rowBg := bgColor
		if i == p.editIndex {
			rowBg = selectedBgColor
		}

		// Field content: "Label: value" with validation indicator
		validIndicator := " "
		fieldFg := fgColor
		if field.valid {
			validIndicator = "✓"
			fieldFg = successFg
		} else if field.value != "" {
			validIndicator = "✗"
			fieldFg = errorFg
		}

		requiredMark := ""
		if field.required {
			requiredMark = "*"
		}

		content := fmt.Sprintf("%s %s%s: %s", validIndicator, field.label, requiredMark, field.value)
		if len(content) > width-4 {
			content = content[:width-7] + "..."
		}

		// Draw content
		for j := 0; j < width-2; j++ {
			var ch rune
			if j < len(content) {
				ch = rune(content[j])
			} else {
				ch = ' '
			}
			cell := goterm.NewCell(ch, fieldFg, rowBg, goterm.StyleNone)
			scr.SetCell(x+1+j, currentY, cell)
		}

		// Right border
		cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+width-1, currentY, cell)

		currentY++

		// Show help text for focused field
		if i == p.editIndex && field.helpText != "" {
			// Help text line
			if currentY < y+height-2 {
				cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
				scr.SetCell(x, currentY, cell)

				helpContent := fmt.Sprintf("  ℹ %s", field.helpText)
				if len(helpContent) > width-4 {
					helpContent = helpContent[:width-7] + "..."
				}

				helpFg := goterm.ColorRGB(150, 150, 150) // Gray
				for j := 0; j < width-2; j++ {
					var ch rune
					if j < len(helpContent) {
						ch = rune(helpContent[j])
					} else {
						ch = ' '
					}
					cell := goterm.NewCell(ch, helpFg, bgColor, goterm.StyleDim)
					scr.SetCell(x+1+j, currentY, cell)
				}

				cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
				scr.SetCell(x+width-1, currentY, cell)

				currentY++
			}
		}
	}

	// Fill remaining space before validation message
	for currentY < y+height-2 {
		cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x, currentY, cell)

		for j := 1; j < width-1; j++ {
			cell := goterm.NewCell(' ', fgColor, bgColor, goterm.StyleNone)
			scr.SetCell(x+j, currentY, cell)
		}

		cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+width-1, currentY, cell)

		currentY++
	}

	// Validation message line (if any)
	if currentY < y+height-1 && p.validationMessage != "" {
		cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x, currentY, cell)

		msgContent := fmt.Sprintf("! %s", p.validationMessage)
		if len(msgContent) > width-4 {
			msgContent = msgContent[:width-7] + "..."
		}

		for j := 0; j < width-2; j++ {
			var ch rune
			if j < len(msgContent) {
				ch = rune(msgContent[j])
			} else {
				ch = ' '
			}
			cell := goterm.NewCell(ch, errorFg, bgColor, goterm.StyleNone)
			scr.SetCell(x+1+j, currentY, cell)
		}

		cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+width-1, currentY, cell)

		currentY++
	}

	// Bottom border
	if currentY < y+height {
		for i := 0; i < width; i++ {
			char := '─'
			switch i {
			case 0:
				char = '└'
			case width - 1:
				char = '┘'
			}
			cell := goterm.NewCell(char, borderFg, bgColor, goterm.StyleNone)
			scr.SetCell(x+i, currentY, cell)
		}
	}

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
