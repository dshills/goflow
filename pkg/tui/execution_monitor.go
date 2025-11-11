package tui

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	execpkg "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// ExecutionMonitor is a comprehensive TUI view for monitoring workflow execution.
// It provides real-time visualization of execution progress, variables, logs, and errors.
// This view integrates multiple panels:
// - Workflow graph with node highlighting
// - Variable inspector for runtime state
// - Execution log viewer with filtering
// - Error detail view with stack traces
// - Performance metrics display
type ExecutionMonitor struct {
	mu sync.RWMutex

	// Core data
	exec     *execution.Execution
	workflow *workflow.Workflow
	screen   *goterm.Screen

	// Event subscription
	eventMonitor execpkg.ExecutionMonitor
	eventChan    <-chan execpkg.ExecutionEvent
	stopChan     chan struct{}

	// Panels
	workflowPanel *WorkflowGraphPanel
	variablePanel *VariableInspectorPanel
	logPanel      *LogViewerPanel
	errorPanel    *ErrorDetailPanel
	metricsPanel  *MetricsPanel
	helpView      *ExecutionHelpPanel

	// State
	activePanel       string // "workflow", "variables", "logs", "error", "metrics", "help"
	lastAction        string
	needsRefresh      bool
	updatedComponents map[string]bool

	// Layout
	width  int
	height int
}

// NewExecutionMonitor creates a new execution monitor view.
// It initializes all panels and subscribes to execution events.
func NewExecutionMonitor(exec *execution.Execution, wf *workflow.Workflow, screen *goterm.Screen) *ExecutionMonitor {
	width, height := screen.Size()

	// Calculate panel dimensions
	// Layout:
	// ┌─────────────────────────────────────────┐ ← Header (3 lines)
	// ├───────────────────┬─────────────────────┤
	// │   Workflow        │  Variables          │
	// │   Graph           │  Inspector          │
	// │   (left 60%)      │  (right 40%)        │
	// │                   ├─────────────────────┤
	// │                   │  Metrics            │
	// │                   │  (right 40%)        │
	// ├───────────────────┴─────────────────────┤
	// │   Logs / Error Detail                   │
	// │   (bottom 30%)                          │
	// └─────────────────────────────────────────┘ ← Status bar (1 line)

	headerHeight := 3
	statusHeight := 1
	contentHeight := height - headerHeight - statusHeight

	logHeight := contentHeight * 3 / 10
	graphHeight := contentHeight - logHeight

	graphWidth := width * 6 / 10
	sideWidth := width - graphWidth

	metricsHeight := graphHeight / 3
	varHeight := graphHeight - metricsHeight

	em := &ExecutionMonitor{
		exec:              exec,
		workflow:          wf,
		screen:            screen,
		activePanel:       "workflow",
		updatedComponents: make(map[string]bool),
		stopChan:          make(chan struct{}),
		width:             width,
		height:            height,
	}

	// Initialize panels
	em.workflowPanel = NewWorkflowGraphPanel(0, headerHeight, graphWidth, graphHeight, wf)
	em.variablePanel = NewVariableInspectorPanel(graphWidth, headerHeight, sideWidth, varHeight)
	em.metricsPanel = NewMetricsPanel(graphWidth, headerHeight+varHeight, sideWidth, metricsHeight)
	em.logPanel = NewLogViewerPanel(0, headerHeight+graphHeight, width, logHeight)
	em.errorPanel = NewErrorDetailPanel(0, headerHeight, width, contentHeight)
	em.helpView = NewExecutionHelpPanel(0, headerHeight, width, contentHeight)

	// Update panels with execution data
	em.updatePanelsFromExecution()

	return em
}

// SetEventMonitor configures the event monitor for real-time updates.
func (em *ExecutionMonitor) SetEventMonitor(monitor execpkg.ExecutionMonitor) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.eventMonitor = monitor
	if monitor != nil {
		em.eventChan = monitor.Subscribe()
		go em.watchEvents()
	}
}

// watchEvents runs in a goroutine to process execution events.
func (em *ExecutionMonitor) watchEvents() {
	for {
		select {
		case event, ok := <-em.eventChan:
			if !ok {
				return
			}
			em.handleEvent(event)
		case <-em.stopChan:
			return
		}
	}
}

// handleEvent processes an execution event and updates relevant panels.
func (em *ExecutionMonitor) handleEvent(event execpkg.ExecutionEvent) {
	em.mu.Lock()
	defer em.mu.Unlock()

	switch event.Type {
	case execpkg.EventExecutionStarted:
		em.markUpdated("status", "metrics")
	case execpkg.EventExecutionCompleted, execpkg.EventExecutionFailed, execpkg.EventExecutionCancelled:
		em.markUpdated("status", "metrics", "logs")
	case execpkg.EventNodeStarted, execpkg.EventNodeCompleted, execpkg.EventNodeFailed, execpkg.EventNodeSkipped:
		em.workflowPanel.UpdateNodeStatus(event.NodeID, event.Status)
		em.markUpdated("workflow", "logs", "metrics")
	case execpkg.EventVariableChanged:
		em.variablePanel.UpdateVariables(event.Variables)
		em.markUpdated("variables")
	case execpkg.EventProgressUpdate:
		if em.eventMonitor != nil {
			progress := em.eventMonitor.GetProgress()
			em.metricsPanel.UpdateProgress(progress)
		}
		em.markUpdated("metrics")
	}

	// Add log entry for this event
	em.logPanel.AddEvent(event)

	em.needsRefresh = true
}

// markUpdated marks components as updated for tracking.
func (em *ExecutionMonitor) markUpdated(components ...string) {
	for _, comp := range components {
		em.updatedComponents[comp] = true
	}
}

// updatePanelsFromExecution updates all panels from the current execution state.
func (em *ExecutionMonitor) updatePanelsFromExecution() {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Update workflow panel with node execution states
	for _, nodeExec := range em.exec.NodeExecutions {
		em.workflowPanel.UpdateNodeStatus(nodeExec.NodeID, nodeExec.Status)
	}

	// Update variables panel
	if em.exec.Context != nil {
		vars := em.exec.Context.GetVariableSnapshot()
		em.variablePanel.UpdateVariables(vars)
	}

	// Update error panel if execution failed
	if em.exec.Error != nil {
		em.errorPanel.SetError(em.exec.Error)
	}

	// Update logs from node executions
	for _, nodeExec := range em.exec.NodeExecutions {
		// Create synthetic events from node execution history
		em.logPanel.AddNodeExecution(nodeExec)
	}

	// Update metrics
	em.updateMetrics()
}

// updateMetrics calculates and updates performance metrics.
func (em *ExecutionMonitor) updateMetrics() {
	var completedNodes, failedNodes, skippedNodes int
	for _, nodeExec := range em.exec.NodeExecutions {
		switch nodeExec.Status {
		case execution.NodeStatusCompleted:
			completedNodes++
		case execution.NodeStatusFailed:
			failedNodes++
		case execution.NodeStatusSkipped:
			skippedNodes++
		}
	}

	totalNodes := len(em.workflow.Nodes)
	percentComplete := float64(completedNodes+failedNodes+skippedNodes) / float64(totalNodes) * 100.0

	progress := execpkg.ExecutionProgress{
		TotalNodes:      totalNodes,
		CompletedNodes:  completedNodes,
		FailedNodes:     failedNodes,
		SkippedNodes:    skippedNodes,
		PercentComplete: percentComplete,
	}

	em.metricsPanel.UpdateProgress(progress)
	em.metricsPanel.UpdateExecution(em.exec)
}

// Render draws the execution monitor to the screen.
func (em *ExecutionMonitor) Render() (time.Duration, error) {
	start := time.Now()

	em.mu.RLock()
	defer em.mu.RUnlock()

	em.screen.Clear()

	// Render header
	em.renderHeader()

	// Render active panels based on view mode
	if em.activePanel == "help" {
		em.helpView.Render(em.screen)
	} else if em.activePanel == "error" && em.errorPanel.HasError() {
		em.errorPanel.Render(em.screen, true)
	} else {
		// Normal view: workflow + variables + metrics + logs
		em.workflowPanel.Render(em.screen, em.activePanel == "workflow")
		em.variablePanel.Render(em.screen, em.activePanel == "variables")
		em.metricsPanel.Render(em.screen, em.activePanel == "metrics")
		em.logPanel.Render(em.screen, em.activePanel == "logs")
	}

	// Render status bar
	em.renderStatusBar()

	// Note: Screen flush is handled by the application layer
	// The goterm.Screen interface doesn't have a Flush() method

	duration := time.Since(start)
	return duration, nil
}

// renderHeader draws the header section with execution info.
func (em *ExecutionMonitor) renderHeader() {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Title
	title := fmt.Sprintf("Execution Monitor: %s", em.workflow.Name)
	em.screen.DrawText(0, 0, title, fg, bg, goterm.StyleBold)

	// Execution info
	execInfo := fmt.Sprintf("ID: %s | Status: %s | Progress: %.0f%%",
		em.exec.ID.String(),
		em.formatStatus(em.exec.Status),
		em.metricsPanel.GetProgress().PercentComplete)
	em.screen.DrawText(0, 1, execInfo, fg, bg, goterm.StyleNone)

	// Separator
	separator := strings.Repeat("─", em.width)
	em.screen.DrawText(0, 2, separator, fg, bg, goterm.StyleNone)
}

// renderStatusBar draws the status bar at the bottom.
func (em *ExecutionMonitor) renderStatusBar() {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()
	y := em.height - 1

	status := fmt.Sprintf("[Tab: Switch] [j/k: Scroll] [e: Expand] [Esc: Back] [?: Help] | Active: %s",
		em.activePanel)

	em.screen.DrawText(0, y, status, fg, bg, goterm.StyleReverse)
}

// formatStatus returns a colored status string.
func (em *ExecutionMonitor) formatStatus(status execution.Status) string {
	switch status {
	case execution.StatusPending:
		return "⏸ Pending"
	case execution.StatusRunning:
		return "▶ Running"
	case execution.StatusCompleted:
		return "✓ Completed"
	case execution.StatusFailed:
		return "✗ Failed"
	case execution.StatusCancelled:
		return "⊗ Cancelled"
	default:
		return string(status)
	}
}

// HandleKey processes keyboard input.
func (em *ExecutionMonitor) HandleKey(key rune) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.lastAction = ""

	switch key {
	case '\t': // Tab
		em.switchPanel(true)
		em.lastAction = "switch_panel"
	case 'j': // Down
		em.scrollActive(1)
		em.lastAction = "scroll"
	case 'k': // Up
		em.scrollActive(-1)
		em.lastAction = "scroll"
	case 'e': // Expand
		if em.activePanel == "variables" {
			em.variablePanel.ToggleExpand()
			em.lastAction = "expand"
		}
	case 27: // Esc
		if em.activePanel == "error" || em.activePanel == "help" {
			em.activePanel = "workflow"
			em.lastAction = "close"
		}
	case '?':
		if em.activePanel == "help" {
			em.activePanel = "workflow"
		} else {
			em.activePanel = "help"
		}
		em.lastAction = "show_help"
	case 'q':
		// Quit handled by app layer
		em.lastAction = "quit"
	}

	em.needsRefresh = true
	return nil
}

// switchPanel switches to the next or previous panel.
func (em *ExecutionMonitor) switchPanel(forward bool) {
	panels := []string{"workflow", "variables", "metrics", "logs"}

	// Find current panel index
	currentIdx := 0
	for i, p := range panels {
		if p == em.activePanel {
			currentIdx = i
			break
		}
	}

	// Move to next/previous
	if forward {
		currentIdx = (currentIdx + 1) % len(panels)
	} else {
		currentIdx = (currentIdx - 1 + len(panels)) % len(panels)
	}

	em.activePanel = panels[currentIdx]
}

// scrollActive scrolls the active panel.
func (em *ExecutionMonitor) scrollActive(delta int) {
	switch em.activePanel {
	case "logs":
		em.logPanel.Scroll(delta)
	case "variables":
		em.variablePanel.Scroll(delta)
	case "error":
		em.errorPanel.Scroll(delta)
	}
}

// Public accessors for testing

func (em *ExecutionMonitor) GetVariableInspector() *VariableInspectorPanel {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.variablePanel
}

func (em *ExecutionMonitor) GetErrorDetailView() *ErrorDetailPanel {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.errorPanel
}

func (em *ExecutionMonitor) GetLogViewer() *LogViewerPanel {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.logPanel
}

func (em *ExecutionMonitor) GetMetricsPanel() *MetricsPanel {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.metricsPanel
}

func (em *ExecutionMonitor) SetActivePanel(panel string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.activePanel = panel
}

func (em *ExecutionMonitor) GetActivePanel() string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.activePanel
}

func (em *ExecutionMonitor) GetLastAction() string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.lastAction
}

func (em *ExecutionMonitor) IsNodeHighlighted(nodeID string) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.workflowPanel.IsNodeHighlighted(types.NodeID(nodeID))
}

func (em *ExecutionMonitor) GetNodeHighlightStyle(nodeID string) string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.workflowPanel.GetNodeHighlightStyle(types.NodeID(nodeID))
}

func (em *ExecutionMonitor) OnExecutionEvent(exec *execution.Execution) bool {
	em.mu.Lock()
	em.exec = exec
	em.mu.Unlock()

	em.updatePanelsFromExecution()
	return true
}

func (em *ExecutionMonitor) WasComponentUpdated(component string) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.updatedComponents[component]
}

// Close stops event watching and cleans up resources.
func (em *ExecutionMonitor) Close() {
	close(em.stopChan)
	if em.eventMonitor != nil && em.eventChan != nil {
		em.eventMonitor.Unsubscribe(em.eventChan)
	}
}

// WorkflowGraphPanel displays the workflow graph with node highlighting.
type WorkflowGraphPanel struct {
	x, y, width, height int
	workflow            *workflow.Workflow
	nodeStatuses        map[types.NodeID]interface{} // execution.NodeStatus or execution.Status
	currentNode         types.NodeID
	scrollOffset        int
}

func NewWorkflowGraphPanel(x, y, width, height int, wf *workflow.Workflow) *WorkflowGraphPanel {
	return &WorkflowGraphPanel{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		workflow:     wf,
		nodeStatuses: make(map[types.NodeID]interface{}),
	}
}

func (p *WorkflowGraphPanel) UpdateNodeStatus(nodeID types.NodeID, status interface{}) {
	p.nodeStatuses[nodeID] = status
	if status == execution.NodeStatusRunning {
		p.currentNode = nodeID
	}
}

func (p *WorkflowGraphPanel) IsNodeHighlighted(nodeID types.NodeID) bool {
	_, exists := p.nodeStatuses[nodeID]
	return exists
}

func (p *WorkflowGraphPanel) GetNodeHighlightStyle(nodeID types.NodeID) string {
	status, exists := p.nodeStatuses[nodeID]
	if !exists {
		return "pending"
	}

	switch s := status.(type) {
	case execution.NodeStatus:
		switch s {
		case execution.NodeStatusRunning:
			return "running"
		case execution.NodeStatusCompleted:
			return "completed"
		case execution.NodeStatusFailed:
			return "failed"
		case execution.NodeStatusSkipped:
			return "skipped"
		default:
			return "pending"
		}
	default:
		return "pending"
	}
}

func (p *WorkflowGraphPanel) Render(screen *goterm.Screen, active bool) {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	titleStyle := goterm.StyleBold
	if active {
		titleStyle = goterm.StyleReverse
	}
	screen.DrawText(p.x, p.y, "┌─ Workflow Graph ", fg, bg, titleStyle)
	screen.DrawText(p.x+17, p.y, strings.Repeat("─", p.width-18)+"┐", fg, bg, goterm.StyleNone)

	// Render nodes as a tree
	y := p.y + 1
	renderedNodes := make(map[string]bool)

	// Find start node
	var startNode workflow.Node
	for _, node := range p.workflow.Nodes {
		if _, ok := node.(*workflow.StartNode); ok {
			startNode = node
			break
		}
	}

	if startNode != nil {
		y = p.renderNodeTree(screen, startNode, y, 2, renderedNodes)
	}

	// Draw bottom border
	if y < p.y+p.height {
		screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
	}
}

func (p *WorkflowGraphPanel) renderNodeTree(screen *goterm.Screen, node workflow.Node, y, indent int, rendered map[string]bool) int {
	if y >= p.y+p.height-1 {
		return y
	}

	nodeID := string(node.GetID())
	if rendered[nodeID] {
		return y
	}
	rendered[nodeID] = true

	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Get node symbol and style
	symbol, style := p.getNodeSymbol(types.NodeID(nodeID))

	// Indent
	prefix := strings.Repeat(" ", indent)

	// Node line
	nodeType := p.getNodeType(node)
	line := fmt.Sprintf("%s%s %s (%s)", prefix, symbol, nodeID, nodeType)
	screen.DrawText(p.x+1, y, line, fg, bg, style)
	y++

	// Show additional details for parallel and loop nodes
	if parallelNode, ok := node.(*workflow.ParallelNode); ok && y < p.y+p.height-1 {
		details := fmt.Sprintf("%s  Strategy: %s", prefix, parallelNode.MergeStrategy)
		screen.DrawText(p.x+1, y, details, fg, bg, goterm.StyleDim)
		y++
	} else if loopNode, ok := node.(*workflow.LoopNode); ok && y < p.y+p.height-1 {
		details := fmt.Sprintf("%s  Over: %s → %s", prefix, loopNode.Collection, loopNode.ItemVariable)
		screen.DrawText(p.x+1, y, details, fg, bg, goterm.StyleDim)
		y++
	}

	// Find children via edges (convert back to NodeID for lookup)
	children := p.findChildren(types.NodeID(nodeID))
	for _, child := range children {
		// Draw connector
		if y < p.y+p.height-1 {
			connector := strings.Repeat(" ", indent) + "  ↓"
			screen.DrawText(p.x+1, y, connector, fg, bg, goterm.StyleDim)
			y++
		}

		// Render child
		y = p.renderNodeTree(screen, child, y, indent+2, rendered)
	}

	return y
}

func (p *WorkflowGraphPanel) getNodeSymbol(nodeID types.NodeID) (string, goterm.Style) {
	style := p.GetNodeHighlightStyle(nodeID)

	switch style {
	case "running":
		return "⟳", goterm.StyleBold
	case "completed":
		return "✓", goterm.StyleNone
	case "failed":
		return "✗", goterm.StyleBold
	case "skipped":
		return "⊘", goterm.StyleDim
	default:
		return "○", goterm.StyleDim
	}
}

func (p *WorkflowGraphPanel) getNodeType(node workflow.Node) string {
	switch n := node.(type) {
	case *workflow.StartNode:
		return "start"
	case *workflow.EndNode:
		return "end"
	case *workflow.MCPToolNode:
		return "tool"
	case *workflow.TransformNode:
		return "transform"
	case *workflow.ConditionNode:
		return "condition"
	case *workflow.LoopNode:
		return fmt.Sprintf("loop[%d nodes]", len(n.Body))
	case *workflow.ParallelNode:
		return fmt.Sprintf("parallel[%d branches]", len(n.Branches))
	default:
		return "unknown"
	}
}

func (p *WorkflowGraphPanel) findChildren(nodeID types.NodeID) []workflow.Node {
	var children []workflow.Node
	for _, edge := range p.workflow.Edges {
		if string(edge.FromNodeID) == string(nodeID) {
			for _, node := range p.workflow.Nodes {
				if node.GetID() == edge.ToNodeID {
					children = append(children, node)
					break
				}
			}
		}
	}
	return children
}

// VariableInspectorPanel displays workflow variables with expansion.
type VariableInspectorPanel struct {
	x, y, width, height int
	variables           map[string]interface{}
	expandedVars        map[string]bool
	scrollOffset        int
	selectedIdx         int
}

func NewVariableInspectorPanel(x, y, width, height int) *VariableInspectorPanel {
	return &VariableInspectorPanel{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		variables:    make(map[string]interface{}),
		expandedVars: make(map[string]bool),
	}
}

func (p *VariableInspectorPanel) UpdateVariables(vars map[string]interface{}) {
	p.variables = vars
}

func (p *VariableInspectorPanel) ToggleExpand() {
	varNames := p.getSortedVarNames()
	if p.selectedIdx < len(varNames) {
		varName := varNames[p.selectedIdx]
		p.expandedVars[varName] = !p.expandedVars[varName]
	}
}

func (p *VariableInspectorPanel) Scroll(delta int) {
	p.scrollOffset += delta
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

func (p *VariableInspectorPanel) IsVisible() bool {
	return true
}

func (p *VariableInspectorPanel) GetDisplayedVariables() map[string]interface{} {
	return p.variables
}

func (p *VariableInspectorPanel) Render(screen *goterm.Screen, active bool) {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	titleStyle := goterm.StyleBold
	if active {
		titleStyle = goterm.StyleReverse
	}
	screen.DrawText(p.x, p.y, "┌─ Variables ", fg, bg, titleStyle)
	screen.DrawText(p.x+12, p.y, strings.Repeat("─", p.width-13)+"┐", fg, bg, goterm.StyleNone)

	y := p.y + 1
	varNames := p.getSortedVarNames()

	for i, name := range varNames {
		if i < p.scrollOffset {
			continue
		}
		if y >= p.y+p.height-1 {
			break
		}

		value := p.variables[name]
		valueStr := p.formatValue(value)

		line := fmt.Sprintf("  %s = %s", name, valueStr)
		if len(line) > p.width-2 {
			line = line[:p.width-5] + "..."
		}

		screen.DrawText(p.x+1, y, line, fg, bg, goterm.StyleNone)
		y++

		// Show expanded view if toggled
		if p.expandedVars[name] {
			expandedLines := p.formatExpanded(value)
			for _, expLine := range expandedLines {
				if y >= p.y+p.height-1 {
					break
				}
				screen.DrawText(p.x+3, y, expLine, fg, bg, goterm.StyleDim)
				y++
			}
		}
	}

	// Bottom border
	if y < p.y+p.height {
		screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
	}
}

func (p *VariableInspectorPanel) getSortedVarNames() []string {
	names := make([]string, 0, len(p.variables))
	for name := range p.variables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *VariableInspectorPanel) formatValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		if len(v) > 30 {
			return fmt.Sprintf("%q...", v[:30])
		}
		return fmt.Sprintf("%q", v)
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	case map[string]interface{}:
		return fmt.Sprintf("{%d fields}", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *VariableInspectorPanel) formatExpanded(value interface{}) []string {
	var lines []string

	switch v := value.(type) {
	case []interface{}:
		for i, item := range v {
			if i >= 10 { // Limit to 10 items
				lines = append(lines, fmt.Sprintf("  ... and %d more", len(v)-10))
				break
			}
			lines = append(lines, fmt.Sprintf("  [%d]: %v", i, item))
		}
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i >= 10 {
				lines = append(lines, fmt.Sprintf("  ... and %d more", len(keys)-10))
				break
			}
			lines = append(lines, fmt.Sprintf("  %s: %v", k, v[k]))
		}
	default:
		lines = append(lines, fmt.Sprintf("  %v", value))
	}

	return lines
}

// Continue in next chunk...
