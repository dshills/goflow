# TUI Integration for Conditional Logic

This document describes the TUI features added for conditional logic support in GoFlow (Tasks T121-T124).

## Features Implemented

### T121: Condition Node in Palette

The "Condition" node type is now available in the node palette:

```
┌─ Node Palette ─────────────────┐
│ 1. MCP Tool                    │
│ 2. Transform                   │
│ 3. Condition      <-- NEW      │
│ 4. Loop                        │
│ 5. Parallel                    │
└────────────────────────────────┘
```

**Usage:**
1. Press `a` in edit mode to open the palette
2. Navigate to "Condition" using `j/k` or arrow keys
3. Press `Enter` to add a condition node to the canvas

**Code:**
```go
// Node is automatically added with default empty condition
builder.AddNodeAtPosition("Condition", Position{X: 10, Y: 10})
```

### T122: Condition Expression Editor

When a condition node is selected, the property panel displays an expression editor:

```
┌─ Condition Node Properties ────┐
│                                │
│   [✓] ID: cond_check_price     │
│ > [✓] Condition Expression:    │
│       $.price > 100 && $.inStock│
│     (Boolean expression,        │
│      e.g., price > 100)        │
│                                │
│ Available Variables:           │
│   - price                      │
│   - quantity                   │
│   - inStock                    │
│                                │
│ Keys: [↑↓] Navigate            │
│       [Enter] Edit [Esc] Close │
└────────────────────────────────┘
```

**Features:**
- **Real-time validation**: Expression is validated as you type
- **Variable suggestions**: Shows available variables from workflow
- **Field type hints**: Displays syntax examples for condition expressions
- **Visual indicators**: ✓ for valid, ✗ for invalid fields

**Code:**
```go
// Show property panel for a condition node
builder.ShowPropertyPanel("cond_1")

// Update the condition expression
builder.UpdatePropertyField(1, "price > 100 && inStock")

// Get validation status
panel := builder.GetPropertyPanel()
validationMsg := panel.GetValidationMessage()
```

### T123: Conditional Edge Labels

Edges from condition nodes display "true" or "false" labels:

```
Workflow Visualization:

  [start]
     │
     ▼
 [check_price] ──true──> [high_value]
     │                        │
     │                        ▼
     └──false─> [low_value] [end]
                     │
                     ▼
                   [end]
```

**Edge Styles:**
- **True edges**: Solid line (`──────>`)
- **False edges**: Dashed line (`- - - ->`)

**Code:**
```go
// Create conditional edges with labels
builder.CreateConditionalEdge("cond_1", "true_path", "true")
builder.CreateConditionalEdge("cond_1", "false_path", "false")

// Get edge label for rendering
label := builder.GetEdgeLabel(edge) // Returns "true" or "false"

// Get edge style for rendering
style := builder.GetEdgeStyle(edge) // Returns "solid" or "dashed"
```

**Validation:**
- Each condition node must have exactly ONE true edge
- Each condition node must have exactly ONE false edge
- Attempting to add a duplicate edge returns an error

### T124: Transform Expression Validator

The property panel validates all types of expressions in real-time:

#### Condition Expressions
```go
// Valid expressions
"price > 100"
"quantity >= 10 && inStock"
"status == 'active'"

// Invalid expressions
"price > > 100"     // Syntax error
"unknownVar > 0"    // Undefined variable
```

#### JSONPath Expressions
```go
// Valid JSONPath
"$.users[0].name"
"$.items[?(@.price > 100)]"

// Invalid JSONPath
"users[0]"          // Must start with $
"$.users[0"         // Unclosed bracket
```

#### Template Expressions
```go
// Valid templates
"Hello ${user.name}"
"Total: ${price * quantity}"

// Invalid templates
"Hello ${user.name"     // Unclosed brace
"Hello ${}"             // Empty variable
```

**Code:**
```go
// Validation is automatic when updating fields
err := builder.UpdatePropertyField(1, "price > 100")
if err != nil {
    // Field is invalid, error message shown in panel
}

// Direct validation functions (also available)
err = workflow.ValidateExpressionSyntax("price > 100")
err = workflow.ValidateJSONPathSyntax("$.users[0].name")
err = workflow.ValidateTemplateSyntax("Hello ${name}")
```

## API Reference

### WorkflowBuilder Methods

#### ShowPropertyPanel
```go
func (b *WorkflowBuilder) ShowPropertyPanel(nodeID string) error
```
Displays the property panel for the specified node.

#### UpdatePropertyField
```go
func (b *WorkflowBuilder) UpdatePropertyField(index int, value string) error
```
Updates a property field value and validates it.

#### GetPropertyPanel
```go
func (b *WorkflowBuilder) GetPropertyPanel() *PropertyPanel
```
Returns the property panel for direct access.

#### GetVariableList
```go
func (b *WorkflowBuilder) GetVariableList() []string
```
Returns list of variable names in the workflow.

#### GetEdgeLabel
```go
func (b *WorkflowBuilder) GetEdgeLabel(edge *workflow.Edge) string
```
Returns the label for an edge ("true", "false", or "").

#### GetEdgeStyle
```go
func (b *WorkflowBuilder) GetEdgeStyle(edge *workflow.Edge) string
```
Returns the style for an edge ("solid" or "dashed").

#### CreateConditionalEdge
```go
func (b *WorkflowBuilder) CreateConditionalEdge(fromID, toID, condition string) error
```
Creates an edge with a condition label ("true" or "false").

### PropertyPanel Methods

#### IsVisible
```go
func (p *PropertyPanel) IsVisible() bool
```
Returns whether the property panel is visible.

#### GetFields
```go
func (p *PropertyPanel) GetFields() []propertyField
```
Returns the property fields for the current node.

#### GetValidationMessage
```go
func (p *PropertyPanel) GetValidationMessage() string
```
Returns the current validation error message.

#### RenderPropertyPanel
```go
func (p *PropertyPanel) RenderPropertyPanel() string
```
Returns a formatted string for displaying the property panel.

### Validation Functions (workflow package)

#### ValidateExpressionSyntax
```go
func ValidateExpressionSyntax(expr string) error
```
Validates boolean condition expression syntax.

#### ValidateJSONPathSyntax
```go
func ValidateJSONPathSyntax(path string) error
```
Validates JSONPath expression syntax.

#### ValidateTemplateSyntax
```go
func ValidateTemplateSyntax(template string) error
```
Validates template string syntax.

#### ExtractVariableReferences
```go
func ExtractVariableReferences(expr string) []string
```
Extracts variable names referenced in an expression.

## Usage Examples

### Creating a Conditional Workflow

```go
// Create workflow
wf, _ := workflow.NewWorkflow("price-check", "Check price threshold")

// Add variables
wf.AddVariable(&workflow.Variable{Name: "price", Type: "number"})
wf.AddVariable(&workflow.Variable{Name: "inStock", Type: "boolean"})

// Create builder
builder, _ := tui.NewWorkflowBuilder(wf)

// Add nodes
builder.AddNodeAtPosition("Condition", Position{X: 10, Y: 10})
builder.AddNodeAtPosition("Transform", Position{X: 10, Y: 20})
builder.AddNodeAtPosition("Transform", Position{X: 10, Y: 30})

// Edit condition
builder.ShowPropertyPanel("condition-0")
builder.UpdatePropertyField(1, "price > 100 && inStock")

// Create conditional edges
builder.CreateConditionalEdge("condition-0", "transform-1", "true")
builder.CreateConditionalEdge("condition-0", "transform-2", "false")
```

### Validating User Input

```go
// Get user input from TUI
userExpr := getUserInput() // e.g., "price > 100"

// Validate before applying
err := workflow.ValidateExpressionSyntax(userExpr)
if err != nil {
    displayError(fmt.Sprintf("Invalid expression: %v", err))
    return
}

// Update the node
builder.UpdatePropertyField(fieldIndex, userExpr)
```

### Rendering Edge Labels

```go
// In your rendering code
for _, edge := range workflow.Edges {
    label := builder.GetEdgeLabel(edge)
    style := builder.GetEdgeStyle(edge)

    if label != "" {
        // Render edge with label
        if style == "dashed" {
            renderDashedEdge(edge, label)
        } else {
            renderSolidEdge(edge, label)
        }
    } else {
        // Regular edge without label
        renderRegularEdge(edge)
    }
}
```

## Testing

All features have comprehensive test coverage:

```bash
# Run all condition TUI tests
go test ./pkg/tui -run TestCondition -v

# Run specific test
go test ./pkg/tui -run TestConditionNodePropertyPanel -v
```

Test files:
- `/pkg/tui/workflow_builder_condition_test.go` - New TUI integration tests
- `/pkg/workflow/expression_validator_test.go` - Expression validation tests

## Design Decisions

### Field Validation Approach
- **Real-time validation**: Fields are validated as soon as they are updated
- **Non-blocking**: Invalid fields don't prevent editing other fields
- **Clear feedback**: Visual indicators (✓/✗) and error messages
- **Context-aware**: Validation considers available variables in workflow

### Edge Creation Flow
- **Explicit condition labels**: User must specify "true" or "false" when creating edges from condition nodes
- **Validation at creation**: Prevents duplicate condition edges immediately
- **Visual distinction**: Different styles help users understand flow at a glance

### Property Panel Design
- **Type-specific fields**: Different field types (text, expression, condition) with appropriate hints
- **Keyboard navigation**: Up/down arrows to navigate fields, Enter to edit
- **Validation feedback**: Inline messages show exactly what's wrong
- **Variable suggestions**: Shows available variables to help construct expressions

## Future Enhancements

Potential improvements for future iterations:

1. **Expression Autocomplete**: Type-ahead suggestions for variable names and operators
2. **Syntax Highlighting**: Color-coded expression text for better readability
3. **Visual Expression Builder**: Drag-and-drop interface for building conditions
4. **Expression Debugger**: Test expressions with sample data before applying
5. **Edge Condition Wizard**: Guided flow for creating conditional edges
6. **Variable Type Hints**: Show variable types in suggestions
7. **Expression Templates**: Pre-built expression patterns for common cases

## Notes

- Condition nodes MUST have exactly 2 outgoing edges (one true, one false)
- Expression validation uses the same engine as runtime execution
- Variables referenced in expressions must be defined in the workflow
- Property panel state is maintained until closed or another node is selected
