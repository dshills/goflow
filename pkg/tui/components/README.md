# TUI Components

Reusable UI components for the GoFlow terminal user interface, built on the `goterm` library.

## Components

### Button (`button.go`)

A clickable button component with keyboard focus support.

**Features:**
- Label with automatic padding
- Enabled/disabled states
- Keyboard focus with visual highlighting
- Click callback support
- Customizable styling (normal, focused, disabled)
- Vim-style activation (Enter or Space)

**Usage:**
```go
button := components.NewButton("Save", 10, 5, func() {
    // Handle button click
    fmt.Println("Button clicked!")
})

button.SetFocused(true)
button.Render(screen)

// Handle keyboard
if button.HandleKey("Enter") {
    // Button was activated
}
```

### Panel (`panel.go`)

A bordered panel with title and scrollable content.

**Features:**
- Optional border with Unicode box-drawing characters
- Title display in top border
- Scrollable content with vim-style navigation (j/k/g/G)
- Automatic scroll management
- Resize handling
- Focus indicator
- Content area calculations

**Usage:**
```go
panel := components.NewPanel("Workflow Details", 0, 0, 40, 20)
panel.SetContent([]string{
    "Name: my-workflow",
    "Status: Running",
    "Nodes: 5",
})

panel.SetFocused(true)
panel.Render(screen)

// Scroll with j/k
panel.HandleKey("j") // scroll down
panel.HandleKey("k") // scroll up
```

### Modal (`modal.go`)

A modal dialog with three variants: Info, Confirm, and Input.

**Features:**
- Center screen positioning
- Backdrop dimming
- Three modal types:
  - Info: Single OK button
  - Confirm: OK/Cancel buttons
  - Input: Text field with OK/Cancel
- ESC to cancel
- Enter to confirm
- Tab navigation between buttons
- Text input with cursor, Home/End, Backspace/Delete
- Word-wrapped message text
- Callback on close

**Usage:**
```go
// Info modal
modal := components.NewInfoModal("Success", "Workflow saved successfully!", func() {
    fmt.Println("User acknowledged")
})

// Confirm modal
modal := components.NewConfirmModal("Delete?", "Are you sure?", func(confirmed bool) {
    if confirmed {
        // Delete the item
    }
})

// Input modal
modal := components.NewInputModal("New Workflow", "Enter workflow name:", "my-workflow",
    func(confirmed bool, input string) {
        if confirmed {
            createWorkflow(input)
        }
    })

modal.Show()
modal.Render(screen)

// Handle keys
modal.HandleKey("Enter") // confirm
modal.HandleKey("Esc")   // cancel
```

### List (`list.go`)

A scrollable list with selection, multi-select, and search/filter capabilities.

**Features:**
- Single or multi-select mode
- Vim-style navigation (j/k/g/G/u/d)
- Visual selection indicator (►)
- Multi-select checkmarks (✓)
- Search/filter with live filtering (/)
- Scroll management
- Disabled items support
- Custom styling per item state
- Page up/down support

**Usage:**
```go
list := components.NewList(0, 0, 40, 20)

// Add items
list.AddItem(components.ListItem{
    Label:   "Start Node",
    Value:   startNode,
    Enabled: true,
})
list.AddItem(components.ListItem{
    Label:   "MCP Tool",
    Value:   toolNode,
    Enabled: true,
})

// Enable multi-select
list.SetMultiSelect(true)

// Enable search
list.SetSearchEnabled(true)

list.Render(screen)

// Navigate
list.HandleKey("j") // move down
list.HandleKey("k") // move up
list.HandleKey(" ") // toggle selection (multi-select)
list.HandleKey("/") // activate search

// Get selected
selected := list.GetSelectedItem()
allSelected := list.GetSelectedItems() // for multi-select
```

### StatusBar (`statusbar.go`)

A status bar component positioned at the bottom of the screen.

**Features:**
- Three sections: left, center, right
- Mode indicator (e.g., NORMAL, INSERT, VISUAL)
- Temporary message display with auto-clear
- Full-width background
- Automatic width adjustment on resize
- Custom styling for mode and messages

**Usage:**
```go
statusBar := components.NewStatusBar(screenHeight-1, screenWidth)

// Set mode
statusBar.SetMode("NORMAL")

// Set section text
statusBar.SetText(components.StatusBarLeft, "Workflow: my-workflow")
statusBar.SetText(components.StatusBarCenter, "5 nodes, 4 edges")
statusBar.SetText(components.StatusBarRight, "Modified")

// Show temporary message (60 frames = ~1 second at 60 FPS)
statusBar.SetMessage("Workflow saved!", 60)

// Update each frame (decrements message timer)
statusBar.Update()

statusBar.Render(screen)
```

## Design Principles

All components follow these principles:

1. **Self-contained**: Each component manages its own state and rendering
2. **Composable**: Components can be combined to build complex UIs
3. **Keyboard-first**: Full keyboard navigation with vim-style bindings
4. **Customizable**: Styling via style structs with sensible defaults
5. **Consistent API**: Similar methods across components (Render, HandleKey, SetFocused, etc.)
6. **Screen-relative**: Position and size in terminal character coordinates

## Common Patterns

### Focus Management

Components that accept input support focus:

```go
component.SetFocused(true)
if component.IsFocused() {
    // Handle input only when focused
    component.HandleKey(key)
}
```

### Rendering Pipeline

1. Create component instances
2. Set positions and sizes
3. Update component state
4. Call Render() with goterm Screen
5. Handle input via HandleKey()

```go
// Setup
button := components.NewButton("Save", 10, 5, onSave)
panel := components.NewPanel("Info", 0, 0, 40, 20)

// Render loop
for {
    screen.Clear()

    panel.Render(screen)
    button.Render(screen)

    screen.Flush()

    // Handle events
    event := screen.PollEvent()
    // ... dispatch to components
}
```

### Styling

Each component has a default style and allows customization:

```go
style := components.DefaultButtonStyle()
style.FocusedBg = goterm.ColorRGB(255, 100, 100) // Custom focused color
button.SetStyle(style)
```

## Integration with GoFlow

These components are designed for use in GoFlow's TUI views:

- **Workflow Explorer**: List component for workflow selection
- **Workflow Builder**: Panel for properties, Button for actions, Modal for confirmations
- **Execution Monitor**: Panel for logs, StatusBar for execution status
- **Server Registry**: List for servers, Modal for add/edit dialogs

## Testing

Components are testable by:
1. Creating instances with known dimensions
2. Calling methods and verifying state
3. Testing HandleKey() return values
4. Verifying Render() produces expected output (can mock Screen)

Example:
```go
func TestButtonActivation(t *testing.T) {
    activated := false
    button := NewButton("Test", 0, 0, func() {
        activated = true
    })

    button.HandleKey("Enter")

    if !activated {
        t.Error("Button should have been activated")
    }
}
```

## Future Enhancements

Potential additions:
- Tab/FormField component for structured input
- ProgressBar component for long operations
- Menu/Dropdown component for hierarchical options
- SplitPane component for resizable layouts
- TreeView component for hierarchical data
- Chart/Graph component for execution visualization
