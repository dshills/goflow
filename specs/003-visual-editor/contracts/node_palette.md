# Contract: Node Palette Component

**Component**: NodePalette
**Package**: `pkg/tui`
**Purpose**: Node type selection interface with search filtering

## Interface

```go
type NodePalette struct {
    nodeTypes     []nodeTypeInfo
    selectedIndex int
    filterText    string
    visible       bool
}

// NewNodePalette creates a node palette with available node types
func NewNodePalette() *NodePalette

// Show opens the palette
func (p *NodePalette) Show()

// Hide closes the palette
func (p *NodePalette) Hide()

// IsVisible returns whether palette is open
func (p *NodePalette) IsVisible() bool

// Next moves selection to next node type
func (p *NodePalette) Next()

// Previous moves selection to previous node type
func (p *NodePalette) Previous()

// Filter updates the search filter
// Returns filtered node types
func (p *NodePalette) Filter(text string) []nodeTypeInfo

// GetSelected returns the currently selected node type
func (p *NodePalette) GetSelected() nodeTypeInfo

// CreateNode creates a workflow node of the selected type
func (p *NodePalette) CreateNode() workflow.Node

// Render draws the palette to screen
func (p *NodePalette) Render(screen *goterm.Screen) error
```

## Node Types

```go
type nodeTypeInfo struct {
    typeName      string                   // "MCP Tool", "Transform", etc.
    description   string                   // "Execute MCP server tool"
    icon          string                   // Unicode icon: "🔧", "🔄", etc.
    defaultConfig map[string]interface{}   // Default field values
}
```

| Type | Icon | Description | Default Config |
|------|------|-------------|----------------|
| MCP Tool | 🔧 | Execute MCP server tool | {name: "tool", serverID: "", toolName: ""} |
| Transform | 🔄 | Transform data (JSONPath, template, jq) | {name: "transform", type: "jsonpath", expression: ""} |
| Condition | ❓ | Conditional branching | {name: "condition", expression: ""} |
| Loop | 🔁 | Iterate over collections | {name: "loop", collection: "", variable: ""} |
| Parallel | ⚡ | Concurrent execution | {name: "parallel", branches: 2} |
| End | 🏁 | Exit point with output | {name: "end", output: ""} |

## Behavior Contracts

### Filter

**Preconditions**:
- Filter text must be string (can be empty)

**Postconditions**:
- Node types list filtered by substring match (case-insensitive)
- Selection index reset to 0 if current selection filtered out
- Empty filter shows all node types

**Algorithm**:
```go
filtered := []nodeTypeInfo{}
for _, nodeType := range allNodeTypes {
    if strings.Contains(
        strings.ToLower(nodeType.typeName),
        strings.ToLower(filterText),
    ) {
        filtered = append(filtered, nodeType)
    }
}
```

**Examples**:
- "trans" → matches "Transform"
- "mcp" → matches "MCP Tool"
- "cond" → matches "Condition"
- "" → matches all

### CreateNode

**Preconditions**:
- A node type must be selected

**Postconditions**:
- New workflow.Node created with selected type
- Default configuration applied from nodeTypeInfo
- Node ID generated (UUID or sequential)

**Error conditions**:
- Returns error if no node type selected
- Returns error if default config invalid

## Visual Layout

```
┌─── Add Node ────────────────────┐
│                                  │
│ Search: [trans___________]       │
│                                  │
│ ► 🔄 Transform                   │
│     Transform data using         │
│     JSONPath, template, or jq    │
│                                  │
│   🔧 MCP Tool                    │
│     Execute MCP server tool      │
│                                  │
│   ❓ Condition                   │
│     Conditional branching        │
│                                  │
│ [Enter: Select] [Esc: Cancel]    │
└──────────────────────────────────┘
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↓ / j | Next node type |
| ↑ / k | Previous node type |
| Enter | Select node type |
| Esc | Cancel |
| / | Focus search field |
| Type text | Filter node types |

## Testing Requirements

- `TestNodePaletteFiltering`: Test filter algorithm
- `TestNodePaletteSelection`: Test navigation
- `TestNodePaletteCreateNode`: Test node creation with defaults

## Dependencies

- `pkg/workflow`: Node types and creation
