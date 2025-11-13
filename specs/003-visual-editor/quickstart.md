# Quickstart: Visual Workflow Editor Development

**Feature**: Complete Visual Workflow Editor
**Audience**: GoFlow developers extending or maintaining the TUI
**Date**: 2025-11-12

## Overview

This guide helps developers work with the visual workflow editor codebase. It covers common extension scenarios and testing patterns.

## Architecture Quick Reference

```
WorkflowBuilder (main orchestrator)
├── Canvas (node positioning, rendering)
├── NodePalette (node type selection)
├── PropertyPanel (node property editing)
├── HelpPanel (keyboard shortcuts)
├── ValidationPanel (error display)
└── UndoStack (undo/redo management)
```

All components operate on the `workflow.Workflow` domain model without modifying it directly. Changes go through `WorkflowBuilder` methods.

## Common Tasks

### 1. Add a New Node Type

**Step 1**: Define node type in workflow domain (if new)
```go
// In pkg/workflow/node.go (if type doesn't exist)
const NodeTypeCustom = "custom"
```

**Step 2**: Add to NodePalette
```go
// In pkg/tui/node_palette.go
func NewNodePalette() *NodePalette {
    return &NodePalette{
        nodeTypes: []nodeTypeInfo{
            // ... existing types
            {
                typeName: "Custom",
                description: "My custom node type",
                icon: "🎯",
                defaultConfig: map[string]interface{}{
                    "name": "custom-node",
                    "customField": "default-value",
                },
            },
        },
    }
}
```

**Step 3**: Add property panel fields (if needed)
```go
// In pkg/tui/property_panel.go
func (p *PropertyPanel) buildFieldsFor NodeType(nodeType string) []propertyField {
    switch nodeType {
    case workflow.NodeTypeCustom:
        return []propertyField{
            {label: "Name", value: node.Name, required: true, fieldType: "text"},
            {label: "Custom Field", value: node.Config["customField"], fieldType: "text"},
        }
    }
}
```

**Step 4**: Add validation rules (if needed)
```go
// In pkg/tui/validation.go or use existing workflow validation
func ValidateCustomNode(node workflow.Node) []validationError {
    errors := []validationError{}
    if node.Config["customField"] == "" {
        errors = append(errors, validationError{
            nodeID: node.ID,
            errorType: "missing_required_field",
            message: "Custom field is required",
        })
    }
    return errors
}
```

**Step 5**: Write tests
```go
// In pkg/tui/node_palette_test.go
func TestNodePaletteCustomNode(t *testing.T) {
    palette := NewNodePalette()
    // Find custom node type
    var customType nodeTypeInfo
    for _, nt := range palette.nodeTypes {
        if nt.typeName == "Custom" {
            customType = nt
            break
        }
    }

    assert.NotNil(t, customType)
    assert.Equal(t, "My custom node type", customType.description)

    // Create node from type
    node := palette.CreateNodeFromType(customType)
    assert.Equal(t, workflow.NodeTypeCustom, node.Type)
    assert.Equal(t, "default-value", node.Config["customField"])
}
```

---

### 2. Add a New Property Field Type

**Step 1**: Define field type constant
```go
// In pkg/tui/property_fields.go
const (
    FieldTypeText       = "text"
    FieldTypeExpression = "expression"
    // ... existing types
    FieldTypeCustom     = "custom"  // New type
)
```

**Step 2**: Implement validation function
```go
// In pkg/tui/property_fields.go
func validateCustomField(value string) error {
    // Your validation logic
    if !strings.HasPrefix(value, "custom:") {
        return fmt.Errorf("custom field must start with 'custom:'")
    }
    return nil
}
```

**Step 3**: Add to field builder
```go
// In pkg/tui/property_panel.go
func (p *PropertyPanel) createField(label, value, fieldType string, required bool) propertyField {
    field := propertyField{
        label:     label,
        value:     value,
        required:  required,
        fieldType: fieldType,
    }

    switch fieldType {
    // ... existing cases
    case FieldTypeCustom:
        field.validationFn = validateCustomField
        field.helpText = "Format: custom:<value>"
    }

    return field
}
```

**Step 4**: Write tests
```go
// In pkg/tui/property_fields_test.go
func TestValidateCustomField(t *testing.T) {
    tests := []struct {
        name    string
        value   string
        wantErr bool
    }{
        {"valid", "custom:value", false},
        {"invalid_prefix", "value", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateCustomField(tt.value)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateCustomField() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

### 3. Add a New Keyboard Shortcut

**Step 1**: Define shortcut in WorkflowBuilder
```go
// In pkg/tui/workflow_builder.go
func (b *WorkflowBuilder) HandleKey(key string) error {
    // Check mode first
    if b.mode != "normal" {
        return b.handleModeSpecificKey(key)
    }

    // Add your shortcut
    switch key {
    // ... existing shortcuts
    case "x":  // New shortcut
        return b.exportWorkflow()
    }
}
```

**Step 2**: Add to help panel
```go
// In pkg/tui/help_panel.go
func (p *HelpPanel) loadKeyBindings() {
    p.keyBindings = []HelpKeyBinding{
        // ... existing bindings
        {
            keys:        []string{"x"},
            description: "Export workflow",
            category:    "Workflow",
            mode:        "normal",
        },
    }
}
```

**Step 3**: Update enabled keys map (if needed)
```go
// In pkg/tui/workflow_builder.go
func NewWorkflowBuilder(/*...*/) *WorkflowBuilder {
    return &WorkflowBuilder{
        // ...
        keyEnabled: map[string]bool{
            // ... existing keys
            "x": true,  // Enable new shortcut
        },
    }
}
```

**Step 4**: Write tests
```go
// In pkg/tui/workflow_builder_test.go
func TestWorkflowBuilder_ExportShortcut(t *testing.T) {
    builder := NewWorkflowBuilder(testWorkflow, testRepo)

    err := builder.HandleKey("x")
    assert.NoError(t, err)

    // Verify export was called
    // (may require mock or test double for repository)
}
```

---

### 4. Add a Workflow Template

**Step 1**: Define template function
```go
// In pkg/tui/templates.go
func CreateCustomTemplate() *workflow.Workflow {
    wf := workflow.NewWorkflow("custom-template")

    // Add nodes
    startNode := workflow.NewNode("start", workflow.NodeTypeStart)
    customNode := workflow.NewNode("custom-1", workflow.NodeTypeCustom)
    endNode := workflow.NewNode("end", workflow.NodeTypeEnd)

    wf.AddNode(startNode)
    wf.AddNode(customNode)
    wf.AddNode(endNode)

    // Connect nodes
    wf.AddEdge(workflow.NewEdge("start", "custom-1"))
    wf.AddEdge(workflow.NewEdge("custom-1", "end"))

    return wf
}
```

**Step 2**: Register in template registry
```go
// In pkg/tui/templates.go
var WorkflowTemplates = map[string]func() *workflow.Workflow{
    "basic":           CreateBasicTemplate,
    "etl":             CreateETLTemplate,
    // ... existing templates
    "custom":          CreateCustomTemplate,  // New template
}

var TemplateDescriptions = map[string]string{
    "basic":  "Simple workflow with 3 nodes",
    "etl":    "Extract, Transform, Load pipeline",
    // ... existing descriptions
    "custom": "My custom workflow pattern",  // New description
}
```

**Step 3**: Write tests
```go
// In pkg/tui/templates_test.go
func TestCreateCustomTemplate(t *testing.T) {
    wf := CreateCustomTemplate()

    assert.Equal(t, "custom-template", wf.Name)
    assert.Len(t, wf.Nodes, 3)  // start, custom, end
    assert.Len(t, wf.Edges, 2)  // 2 connections

    // Verify structure
    assert.Equal(t, workflow.NodeTypeStart, wf.Nodes[0].Type)
    assert.Equal(t, workflow.NodeTypeCustom, wf.Nodes[1].Type)
    assert.Equal(t, workflow.NodeTypeEnd, wf.Nodes[2].Type)
}
```

---

### 5. Test TUI Interactions

**Pattern**: Simulate keyboard events and verify state changes

```go
// In tests/tui/builder_interaction_test.go
func TestAddNodeInteraction(t *testing.T) {
    // Setup
    repo := NewInMemoryRepository()
    wf := workflow.NewWorkflow("test")
    builder := NewWorkflowBuilder(wf, repo)

    // Simulate: Press 'a' to open palette
    err := builder.HandleKey("a")
    assert.NoError(t, err)
    assert.True(t, builder.palette.IsVisible())
    assert.Equal(t, "palette", builder.mode)

    // Simulate: Navigate palette (down arrow)
    err = builder.palette.Next()
    assert.NoError(t, err)
    assert.Equal(t, 1, builder.palette.selectedIndex)

    // Simulate: Select node type (Enter)
    err = builder.HandleKey("Enter")
    assert.NoError(t, err)

    // Verify: Node added to canvas
    assert.Len(t, builder.canvas.nodes, 1)  // Start node + new node
    assert.False(t, builder.palette.IsVisible())
    assert.Equal(t, "normal", builder.mode)
}
```

**Testing checklist**:
- ✓ Keyboard navigation works in each mode
- ✓ Mode transitions are correct
- ✓ State changes propagate to all components
- ✓ Undo/redo works for the operation
- ✓ Workflow modification flag set correctly

---

## Development Workflow

### Running Tests

```bash
# All TUI tests
go test ./pkg/tui/...

# Specific component
go test ./pkg/tui -run TestCanvas

# With coverage
go test -cover ./pkg/tui/...

# With race detection
go test -race ./pkg/tui/...

# Integration tests
go test ./tests/tui/...
```

### Performance Benchmarks

```bash
# Canvas rendering benchmark
go test -bench=BenchmarkCanvasRender ./pkg/tui

# Auto-layout benchmark
go test -bench=BenchmarkAutoLayout ./pkg/tui

# Undo/redo benchmark
go test -bench=BenchmarkUndo ./pkg/tui
```

### Running the Builder Locally

```bash
# Build and run
go build -o goflow ./cmd/goflow
./goflow edit my-workflow

# Or run directly
go run ./cmd/goflow edit my-workflow

# With debug mode
go run ./cmd/goflow edit my-workflow --debug
```

---

## Code Conventions

### Naming

- **Components**: `Canvas`, `PropertyPanel`, `NodePalette` (PascalCase)
- **Methods**: `HandleKey`, `AddNode`, `SaveChanges` (PascalCase, verbs)
- **Internal functions**: `calculateLayout`, `routeEdge` (camelCase)
- **Constants**: `FieldTypeText`, `ModeNormal` (PascalCase)

### Error Handling

```go
// Return errors, don't panic
func (c *Canvas) AddNode(node workflow.Node, pos Position) error {
    if _, exists := c.nodes[node.ID]; exists {
        return fmt.Errorf("node with ID %s already exists", node.ID)
    }
    // ...
    return nil
}

// Use errors package for wrapping
import "github.com/dshills/goflow/pkg/errors"

func (b *WorkflowBuilder) SaveWorkflow() error {
    if err := b.repository.Save(b.workflow); err != nil {
        return errors.NewOperationalError(
            "saving workflow",
            b.workflow.ID,
            "",
            err,
        )
    }
    return nil
}
```

### Testing

```go
// Table-driven tests
func TestPropertyFieldValidation(t *testing.T) {
    tests := []struct {
        name      string
        fieldType string
        value     string
        wantErr   bool
    }{
        {"valid_text", "text", "hello", false},
        {"empty_required", "text", "", true},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateField(tt.fieldType, tt.value)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Debugging Tips

### Visual Debugging

```go
// Add debug rendering
func (c *Canvas) Render(screen *goterm.Screen) error {
    if debugMode {
        c.renderDebugInfo(screen)
    }
    // ... normal rendering
}

func (c *Canvas) renderDebugInfo(screen *goterm.Screen) {
    // Show viewport bounds
    screen.DrawBox(c.ViewportX, c.ViewportY, c.Width, c.Height, goterm.ColorRed)

    // Show node bounding boxes
    for _, node := range c.nodes {
        screen.DrawRect(node.position.X, node.position.Y, node.width, node.height, goterm.ColorYellow)
    }
}
```

### State Inspection

```go
// Add state dump method
func (b *WorkflowBuilder) DumpState() string {
    return fmt.Sprintf("Mode: %s, Selected: %s, Modified: %v, Undo: %d, Redo: %d",
        b.mode,
        b.selectedNodeID,
        b.modified,
        len(b.undoStack),
        len(b.redoStack),
    )
}

// Log state changes in development
if devMode {
    log.Printf("WorkflowBuilder state: %s", b.DumpState())
}
```

---

## Further Reading

- **Domain Model**: `pkg/workflow/README.md` - Workflow aggregate documentation
- **goterm Library**: `github.com/dshills/goterm` - Terminal UI framework
- **Constitution**: `.specify/memory/constitution.md` - Project principles
- **Contracts**: `specs/003-visual-editor/contracts/` - Component interfaces

## Need Help?

1. Check existing tests for usage examples
2. Read contract documentation for component behavior
3. Review research.md for design decisions and rationale
4. Consult CLAUDE.md for development guidance
