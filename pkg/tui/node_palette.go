package tui

import (
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
	"github.com/google/uuid"
)

// nodeTypeInfo defines metadata for a node type
type nodeTypeInfo struct {
	typeName      string                 // Display name: "MCP Tool", "Transform", etc.
	description   string                 // Short help text
	icon          string                 // Unicode emoji icon
	defaultConfig map[string]interface{} // Default field values
}

// NodePalette manages node type selection with search filtering
type NodePalette struct {
	nodeTypes     []nodeTypeInfo // All available node types
	selectedIndex int            // Currently highlighted type
	filterText    string         // Search filter
	visible       bool           // Palette open/closed
}

// NewNodePalette creates a node palette with all available node types
func NewNodePalette() *NodePalette {
	return &NodePalette{
		nodeTypes: []nodeTypeInfo{
			{
				typeName:    "MCP Tool",
				description: "Execute MCP server tool",
				icon:        "🔧",
				defaultConfig: map[string]interface{}{
					"name":     "tool",
					"serverID": "",
					"toolName": "",
				},
			},
			{
				typeName:    "Transform",
				description: "Transform data using JSONPath, template, or jq",
				icon:        "🔄",
				defaultConfig: map[string]interface{}{
					"name":       "transform",
					"type":       "jsonpath",
					"expression": "",
				},
			},
			{
				typeName:    "Condition",
				description: "Conditional branching",
				icon:        "❓",
				defaultConfig: map[string]interface{}{
					"name":       "condition",
					"expression": "",
				},
			},
			{
				typeName:    "Loop",
				description: "Iterate over collections",
				icon:        "🔁",
				defaultConfig: map[string]interface{}{
					"name":       "loop",
					"collection": "",
					"variable":   "",
				},
			},
			{
				typeName:    "Parallel",
				description: "Concurrent execution",
				icon:        "⚡",
				defaultConfig: map[string]interface{}{
					"name":     "parallel",
					"branches": 2,
				},
			},
			{
				typeName:    "End",
				description: "Exit point with output",
				icon:        "🏁",
				defaultConfig: map[string]interface{}{
					"name":   "end",
					"output": "",
				},
			},
		},
		selectedIndex: 0,
		filterText:    "",
		visible:       false,
	}
}

// Show opens the palette
func (p *NodePalette) Show() {
	p.visible = true
	p.selectedIndex = 0
	p.filterText = ""
}

// Hide closes the palette
func (p *NodePalette) Hide() {
	p.visible = false
}

// IsVisible returns whether palette is open
func (p *NodePalette) IsVisible() bool {
	return p.visible
}

// Next moves selection to next node type (with wrap-around)
func (p *NodePalette) Next() {
	filtered := p.Filter(p.filterText)
	if len(filtered) == 0 {
		return
	}
	p.selectedIndex = (p.selectedIndex + 1) % len(filtered)
}

// Previous moves selection to previous node type (with wrap-around)
func (p *NodePalette) Previous() {
	filtered := p.Filter(p.filterText)
	if len(filtered) == 0 {
		return
	}
	p.selectedIndex--
	if p.selectedIndex < 0 {
		p.selectedIndex = len(filtered) - 1
	}
}

// Filter updates the search filter and returns filtered node types
// Uses case-insensitive substring matching on typeName
func (p *NodePalette) Filter(text string) []nodeTypeInfo {
	p.filterText = text

	// Empty filter shows all types
	if text == "" {
		return p.nodeTypes
	}

	// Filter by substring match (case-insensitive)
	filtered := []nodeTypeInfo{}
	lowerFilter := strings.ToLower(text)

	for _, nodeType := range p.nodeTypes {
		if strings.Contains(strings.ToLower(nodeType.typeName), lowerFilter) {
			filtered = append(filtered, nodeType)
		}
	}

	// Reset selection if current selection is filtered out
	if p.selectedIndex >= len(filtered) {
		p.selectedIndex = 0
	}

	return filtered
}

// GetSelected returns the currently selected node type
func (p *NodePalette) GetSelected() nodeTypeInfo {
	filtered := p.Filter(p.filterText)
	if len(filtered) == 0 {
		return nodeTypeInfo{}
	}
	if p.selectedIndex >= len(filtered) {
		p.selectedIndex = 0
	}
	return filtered[p.selectedIndex]
}

// CreateNode creates a workflow node of the selected type
func (p *NodePalette) CreateNode() (workflow.Node, error) {
	selected := p.GetSelected()
	if selected.typeName == "" {
		return nil, fmt.Errorf("no node type selected")
	}

	// Generate unique node ID
	nodeID := "node-" + uuid.New().String()[:8]

	// Create node based on selected type
	switch selected.typeName {
	case "MCP Tool":
		return &workflow.MCPToolNode{
			ID:             nodeID,
			ServerID:       selected.defaultConfig["serverID"].(string),
			ToolName:       selected.defaultConfig["toolName"].(string),
			Parameters:     make(map[string]string),
			OutputVariable: "result",
		}, nil

	case "Transform":
		return &workflow.TransformNode{
			ID:             nodeID,
			InputVariable:  "input",
			Expression:     selected.defaultConfig["expression"].(string),
			OutputVariable: "output",
		}, nil

	case "Condition":
		return &workflow.ConditionNode{
			ID:        nodeID,
			Condition: selected.defaultConfig["expression"].(string),
		}, nil

	case "Loop":
		return &workflow.LoopNode{
			ID:           nodeID,
			Collection:   selected.defaultConfig["collection"].(string),
			ItemVariable: selected.defaultConfig["variable"].(string),
			Body:         []string{},
		}, nil

	case "Parallel":
		branches := selected.defaultConfig["branches"].(int)
		parallelNode := &workflow.ParallelNode{
			ID:            nodeID,
			Branches:      make([][]string, branches),
			MergeStrategy: "wait_all",
		}
		for i := 0; i < branches; i++ {
			parallelNode.Branches[i] = []string{}
		}
		return parallelNode, nil

	case "End":
		return &workflow.EndNode{
			ID:          nodeID,
			ReturnValue: selected.defaultConfig["output"].(string),
		}, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", selected.typeName)
	}
}

// Render draws the node palette to the screen
// x, y: top-left corner position
// width, height: available space
func (p *NodePalette) Render(screen interface{}, x, y, width, height int) error {
	if !p.visible {
		return nil
	}

	// Import goterm types
	type Screen interface {
		SetCell(cellX, cellY int, cell interface{})
		Size() (int, int)
	}

	scr, ok := screen.(Screen)
	if !ok {
		return fmt.Errorf("invalid screen type")
	}

	// Get filtered node types
	filtered := p.Filter(p.filterText)
	if len(filtered) == 0 {
		return nil
	}

	// Colors using goterm
	fgColor := goterm.ColorRGB(255, 255, 255)      // White text
	bgColor := goterm.ColorRGB(30, 30, 30)         // Dark background
	selectedBgColor := goterm.ColorRGB(58, 58, 58) // Gray background for selected
	borderFg := goterm.ColorRGB(136, 136, 136)     // Gray border

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

	// Title
	title := "Node Palette"
	if p.filterText != "" {
		title = fmt.Sprintf("Node Palette [%s]", p.filterText)
	}
	titlePadding := (width - 2 - len(title)) / 2
	for i, ch := range title {
		if i+titlePadding+1 < width-1 {
			cell := goterm.NewCell(ch, fgColor, bgColor, goterm.StyleBold)
			scr.SetCell(x+1+titlePadding+i, y, cell)
		}
	}

	// Middle rows
	currentY := y + 1
	for i, nodeType := range filtered {
		if currentY >= y+height-1 {
			break
		}

		// Left border
		cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x, currentY, cell)

		// Determine background for this row
		rowBg := bgColor
		if i == p.selectedIndex {
			rowBg = selectedBgColor
		}

		// Node type content: "icon TypeName - description"
		content := fmt.Sprintf("%s %s - %s", nodeType.icon, nodeType.typeName, nodeType.description)
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
			cell := goterm.NewCell(ch, fgColor, rowBg, goterm.StyleNone)
			scr.SetCell(x+1+j, currentY, cell)
		}

		// Right border
		cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+width-1, currentY, cell)

		currentY++
	}

	// Fill remaining space
	for currentY < y+height-1 {
		// Left border
		cell := goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x, currentY, cell)

		// Fill with spaces
		for j := 1; j < width-1; j++ {
			cell := goterm.NewCell(' ', fgColor, bgColor, goterm.StyleNone)
			scr.SetCell(x+j, currentY, cell)
		}

		// Right border
		cell = goterm.NewCell('│', borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+width-1, currentY, cell)

		currentY++
	}

	// Bottom border
	for i := 0; i < width; i++ {
		char := '─'
		switch i {
		case 0:
			char = '└'
		case width - 1:
			char = '┘'
		}
		cell := goterm.NewCell(char, borderFg, bgColor, goterm.StyleNone)
		scr.SetCell(x+i, y+height-1, cell)
	}

	return nil
}
