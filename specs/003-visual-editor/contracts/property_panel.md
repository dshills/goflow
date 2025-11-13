# Contract: Property Panel Component

**Component**: PropertyPanel
**Package**: `pkg/tui`
**Purpose**: Edit node properties with real-time validation and type-specific field editors

## Interface

```go
type PropertyPanel struct {
    node              workflow.Node
    fields            []propertyField
    editIndex         int
    visible           bool
    validationMessage string
    dirty             bool
}

// NewPropertyPanel creates a property panel for the given node
func NewPropertyPanel(node workflow.Node) *PropertyPanel

// Show opens the panel for editing
func (p *PropertyPanel) Show()

// Hide closes the panel
func (p *PropertyPanel) Hide()

// IsVisible returns whether panel is open
func (p *PropertyPanel) IsVisible() bool

// NextField moves focus to next field
func (p *PropertyPanel) NextField()

// PrevField moves focus to previous field
func (p *PropertyPanel) PrevField()

// EditCurrentField enters edit mode for focused field
func (p *PropertyPanel) EditCurrentField()

// SetFieldValue updates the current field value
// Triggers real-time validation
func (p *PropertyPanel) SetFieldValue(value string) error

// SaveChanges applies changes to node
// Returns error if validation fails
func (p *PropertyPanel) SaveChanges() (*workflow.Node, error)

// CancelChanges discards all changes
func (p *PropertyPanel) CancelChanges()

// IsDirty returns true if unsaved changes exist
func (p *PropertyPanel) IsDirty() bool

// Validate runs validation on all fields
func (p *PropertyPanel) Validate() error

// Render draws the panel to screen
func (p *PropertyPanel) Render(screen *goterm.Screen) error
```

## Field Types

### Text Field
- Simple string input
- Validation: Required/optional, max length, regex
- Example: Node name, description

### Expression Field
- Sandboxed expression for data manipulation
- Validation: Parse with `expr-lang/expr`, check for unsafe operations
- Syntax hints: Show available variables and functions
- Example: `$.data | map(.price) | sum`

### Condition Field
- Boolean expression
- Validation: Must evaluate to boolean type
- Syntax hints: Comparison operators, logical operators
- Example: `count > 10 and status == "active"`

### JSONPath Field
- Data query expression
- Validation: Parse with `gjson`, check syntax
- Syntax hints: JSONPath operators ($, @, ., [], *)
- Example: `$.users[?(@.age > 18)].email`

### Template Field
- String interpolation with ${} placeholders
- Validation: Parse placeholders, check variable existence
- Syntax hints: Show available variables
- Example: `"Hello ${user.name}, you have ${count} items"`

## Behavior Contracts

### SaveChanges

**Preconditions**:
- All required fields must have values
- All field values must pass validation

**Postconditions**:
- Node updated with new values
- Panel marked as not dirty
- Validation errors cleared

**Error conditions**:
- Returns error if any required field empty
- Returns error if any field validation fails
- Does not apply changes on error

### SetFieldValue

**Preconditions**:
- Field index must be valid
- Value must be string

**Postconditions**:
- Field value updated
- Validation runs automatically on blur
- Validation status updated (valid/invalid)
- Panel marked as dirty

**Performance**: Validation < 200ms

## Visual Layout

```
┌─── Property Panel (Node: tool-1) ────┐
│                                        │
│ Name *                                 │
│ [filesystem-reader_________]          │
│                                        │
│ Server ID *                            │
│ [local-server_____________]           │
│                                        │
│ Tool Name *                            │
│ [filesystem.read__________]           │
│ ✓ Valid tool name                      │
│                                        │
│ Input Mapping (JSONPath)               │
│ [$.data.file_path_________]           │
│ ✓ Valid JSONPath expression            │
│                                        │
│ Output Variable *                      │
│ [file_contents____________]           │
│                                        │
│ [Save (Ctrl+S)]  [Cancel (Esc)]       │
└────────────────────────────────────────┘

Legend:
* = Required field
✓ = Validation passed
✗ = Validation failed (red)
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Tab / ↓ | Next field |
| Shift+Tab / ↑ | Previous field |
| Enter | Edit current field |
| Esc | Cancel edit / Close panel |
| Ctrl+S | Save changes |
| Ctrl+R | Reset field to original value |

## Validation Timing

- **On field blur**: Validate when leaving field
- **On save**: Validate all fields
- **NOT on every keystroke**: Too disruptive

## Testing Requirements

- `TestPropertyPanelTextValidation`: Test text field validation
- `TestPropertyPanelExpressionValidation`: Test expression syntax
- `TestPropertyPanelJSONPathValidation`: Test JSONPath parsing
- `TestPropertyPanelSaveChanges`: Test save with valid/invalid data
- `TestPropertyPanelDirtyState`: Test dirty flag management

## Dependencies

- `pkg/workflow`: Node types and field schemas
- `github.com/expr-lang/expr`: Expression validation
- `github.com/tidwall/gjson`: JSONPath validation
