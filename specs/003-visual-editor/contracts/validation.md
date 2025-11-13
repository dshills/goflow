# Contract: Workflow Validation Component

**Component**: ValidationStatus and ValidationPanel
**Package**: `pkg/tui`
**Purpose**: Validate workflow structure and display errors/warnings

## Interface

```go
type ValidationStatus struct {
    valid         bool
    errors        []validationError
    warnings      []validationWarning
    lastValidated time.Time
}

type validationError struct {
    nodeID    string   // Node with error ("" for global errors)
    errorType string   // Error category
    message   string   // Human-readable message
}

type validationWarning struct {
    nodeID  string
    message string
}

// ValidateWorkflow performs full workflow validation
func ValidateWorkflow(wf *workflow.Workflow) *ValidationStatus

// ValidateNode performs single node validation
func ValidateNode(node workflow.Node) []validationError

// ValidationPanel displays validation results
type ValidationPanel struct {
    status         *ValidationStatus
    selectedIndex  int
    visible        bool
}

// NewValidationPanel creates a validation panel
func NewValidationPanel(status *ValidationStatus) *ValidationPanel

// Show opens the panel
func (p *ValidationPanel) Show()

// Hide closes the panel
func (p *ValidationPanel) Hide()

// Next selects next error
func (p *ValidationPanel) Next()

// Previous selects previous error
func (p *ValidationPanel) Previous()

// GetSelectedNodeID returns node ID of selected error
func (p *ValidationPanel) GetSelectedNodeID() string

// Render draws the panel to screen
func (p *ValidationPanel) Render(screen *goterm.Screen) error
```

## Validation Rules

### Structural Validation

1. **No circular dependencies**
   - Error type: `circular_dependency`
   - Detection: Cycle detection in workflow graph (DFS)
   - Message: "Circular dependency detected: A → B → C → A"

2. **All nodes reachable from start**
   - Error type: `disconnected_node`
   - Detection: BFS from start node
   - Warning for disconnected nodes (non-blocking)
   - Message: "Node 'transform-1' is not reachable from start"

3. **All edges have valid targets**
   - Error type: `invalid_edge_target`
   - Detection: Check edge.To exists in workflow.Nodes
   - Message: "Edge from 'node-1' targets non-existent node 'node-99'"

### Node Validation

4. **Required fields populated**
   - Error type: `missing_required_field`
   - Per node type requirements
   - Message: "Required field 'tool name' missing in node 'tool-1'"

5. **Valid expressions**
   - Error type: `invalid_expression`
   - Parse with `expr-lang/expr`
   - Message: "Invalid expression syntax: unexpected token ')'"

6. **Valid JSONPath**
   - Error type: `invalid_jsonpath`
   - Parse with `gjson`
   - Message: "Invalid JSONPath: '$.invalid[' - unclosed bracket"

7. **Valid templates**
   - Error type: `invalid_template`
   - Parse `${}` placeholders
   - Message: "Template references undefined variable: '${unknown}'"

### Domain-Specific Validation

8. **Condition nodes have 2 outgoing edges**
   - Error type: `invalid_condition_edges`
   - Count outgoing edges from condition nodes
   - Message: "Condition node 'check-status' must have exactly 2 outgoing edges (true/false)"

9. **Loop nodes have valid collection source**
   - Error type: `invalid_loop_collection`
   - Check collection variable exists and is array type
   - Message: "Loop node 'process-items' collection '${items}' is not an array"

## Behavior Contracts

### ValidateWorkflow

**Preconditions**:
- Workflow must be non-null
- Workflow must have at least start node

**Postconditions**:
- ValidationStatus returned with all errors and warnings
- Errors ordered by severity (blocking first)
- Each error includes node ID for navigation

**Performance**: < 500ms for 100-node workflow

**Algorithm**:
1. Check for circular dependencies (O(V + E))
2. Check reachability from start (O(V + E))
3. Validate each node (O(V))
4. Validate each edge (O(E))
5. Check domain-specific rules (O(V + E))

### ValidateNode

**Preconditions**:
- Node must be non-null

**Postconditions**:
- List of validation errors for the node
- Empty list if node valid

**Validation order**:
1. Required fields present
2. Field values well-formed (syntax)
3. Field values semantically valid (references exist)

## Visual Layout

```
┌─── Validation Errors (3) ───────────────────┐
│                                               │
│ ► ❌ Node 'tool-1': Required field 'tool     │
│      name' missing                            │
│                                               │
│   ⚠️  Node 'transform-2': Disconnected node  │
│                                               │
│   ❌ Node 'condition-1': Condition node must │
│      have exactly 2 outgoing edges           │
│                                               │
│ [Enter: Navigate to node] [Esc: Close]       │
└───────────────────────────────────────────────┘
```

## Error Categories

| Category | Icon | Blocking | Color |
|----------|------|----------|-------|
| Error | ❌ | Yes | Red |
| Warning | ⚠️ | No | Yellow |
| Info | ℹ️ | No | Blue |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↓ / j | Next error |
| ↑ / k | Previous error |
| Enter | Navigate to node with error |
| Esc | Close panel |
| v | Toggle validation panel |

## Integration with Canvas

When validation panel displays error:
1. Highlight problematic node on canvas
2. Show error icon on node border
3. Change node border color (red for error, yellow for warning)
4. Pressing Enter navigates canvas to node

## Testing Requirements

- `TestValidateCircularDependency`: Detect cycles
- `TestValidateDisconnectedNodes`: Find unreachable nodes
- `TestValidateRequiredFields`: Check required fields
- `TestValidateExpressionSyntax`: Parse expressions
- `TestValidateJSONPath`: Parse JSONPath
- `TestValidateConditionEdges`: Check condition node edges
- `TestValidationPerformance`: Benchmark 100-node workflow

## Dependencies

- `pkg/workflow`: Workflow domain model
- `github.com/expr-lang/expr`: Expression validation
- `github.com/tidwall/gjson`: JSONPath validation

## Notes

- Validation runs asynchronously (non-blocking UI)
- Validation cached until workflow modified
- Validation triggered on: save, explicit validate command, auto (configurable interval)
