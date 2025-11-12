package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	execpkg "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goterm"
)

// LogViewerPanel displays chronological execution logs with filtering.
type LogViewerPanel struct {
	x, y, width, height int
	entries             []LogEntry
	scrollOffset        int
	autoScroll          bool
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	NodeID    types.NodeID
	Message   string
	EventType execpkg.ExecutionEventType
}

func NewLogViewerPanel(x, y, width, height int) *LogViewerPanel {
	return &LogViewerPanel{
		x:          x,
		y:          y,
		width:      width,
		height:     height,
		entries:    make([]LogEntry, 0),
		autoScroll: true,
	}
}

func (p *LogViewerPanel) AddEvent(event execpkg.ExecutionEvent) {
	entry := LogEntry{
		Timestamp: event.Timestamp,
		NodeID:    event.NodeID,
		EventType: event.Type,
	}

	// Format message based on event type
	switch event.Type {
	case execpkg.EventExecutionStarted:
		entry.Level = "info"
		entry.Message = "Execution started"
	case execpkg.EventExecutionCompleted:
		entry.Level = "info"
		entry.Message = "Execution completed"
	case execpkg.EventExecutionFailed:
		entry.Level = "error"
		entry.Message = "Execution failed"
	case execpkg.EventNodeStarted:
		entry.Level = "info"
		// Check if this is a loop or parallel node for special formatting
		if event.Metadata != nil {
			if iteration, ok := event.Metadata["iteration"]; ok {
				entry.Message = fmt.Sprintf("Node '%s' started (iteration %v)", event.NodeID, iteration)
			} else if branch, ok := event.Metadata["branch"]; ok {
				entry.Message = fmt.Sprintf("Node '%s' started (branch %v)", event.NodeID, branch)
			} else {
				entry.Message = fmt.Sprintf("Node '%s' started", event.NodeID)
			}
		} else {
			entry.Message = fmt.Sprintf("Node '%s' started", event.NodeID)
		}
	case execpkg.EventNodeCompleted:
		entry.Level = "info"
		// Check for iteration/branch metadata
		if event.Metadata != nil {
			if iteration, ok := event.Metadata["iteration"]; ok {
				entry.Message = fmt.Sprintf("Node '%s' completed (iteration %v)", event.NodeID, iteration)
			} else if branch, ok := event.Metadata["branch"]; ok {
				entry.Message = fmt.Sprintf("Node '%s' completed (branch %v)", event.NodeID, branch)
			} else {
				entry.Message = fmt.Sprintf("Node '%s' completed", event.NodeID)
			}
		} else {
			entry.Message = fmt.Sprintf("Node '%s' completed", event.NodeID)
		}
	case execpkg.EventNodeFailed:
		entry.Level = "error"
		entry.Message = fmt.Sprintf("Node '%s' failed", event.NodeID)
	case execpkg.EventVariableChanged:
		entry.Level = "debug"
		entry.Message = "Variables updated"
	default:
		entry.Level = "debug"
		entry.Message = string(event.Type)
	}

	p.entries = append(p.entries, entry)

	if p.autoScroll {
		// Keep scroll at bottom
		maxScroll := len(p.entries) - (p.height - 3)
		if maxScroll > 0 {
			p.scrollOffset = maxScroll
		}
	}
}

func (p *LogViewerPanel) AddNodeExecution(nodeExec *execution.NodeExecution) {
	// Add start event
	p.entries = append(p.entries, LogEntry{
		Timestamp: nodeExec.StartedAt,
		Level:     "info",
		NodeID:    nodeExec.NodeID,
		Message:   fmt.Sprintf("Node '%s' started", nodeExec.NodeID),
	})

	// Add completion event if finished
	if !nodeExec.CompletedAt.IsZero() {
		level := "info"
		message := fmt.Sprintf("Node '%s' completed (%.2fs)", nodeExec.NodeID, nodeExec.Duration().Seconds())

		if nodeExec.Status == execution.NodeStatusFailed {
			level = "error"
			message = fmt.Sprintf("Node '%s' failed", nodeExec.NodeID)
		}

		p.entries = append(p.entries, LogEntry{
			Timestamp: nodeExec.CompletedAt,
			Level:     level,
			NodeID:    nodeExec.NodeID,
			Message:   message,
		})
	}

	// Update scroll position if auto-scroll is enabled
	if p.autoScroll {
		maxScroll := len(p.entries) - (p.height - 3)
		if maxScroll > 0 {
			p.scrollOffset = maxScroll
		}
	}
}

func (p *LogViewerPanel) Scroll(delta int) {
	p.scrollOffset += delta
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
	maxScroll := len(p.entries) - (p.height - 3)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollOffset > maxScroll {
		p.scrollOffset = maxScroll
	}

	// Disable auto-scroll when manually scrolling
	if delta != 0 {
		p.autoScroll = false
	}
}

func (p *LogViewerPanel) SetAutoScroll(enabled bool) {
	p.autoScroll = enabled
	// When disabling auto-scroll, reset to top of logs
	if !enabled {
		p.scrollOffset = 0
	}
}

func (p *LogViewerPanel) IsScrolledToBottom() bool {
	maxScroll := len(p.entries) - (p.height - 3)
	if maxScroll < 0 {
		return true
	}
	return p.scrollOffset >= maxScroll
}

func (p *LogViewerPanel) GetLogEntries() []LogEntry {
	return p.entries
}

func (p *LogViewerPanel) Render(screen *goterm.Screen, active bool) {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	titleStyle := goterm.StyleBold
	if active {
		titleStyle = goterm.StyleReverse
	}

	autoScrollStatus := ""
	if p.autoScroll {
		autoScrollStatus = " [AutoScroll: ON]"
	}

	title := fmt.Sprintf("┌─ Execution Log%s ", autoScrollStatus)
	screen.DrawText(p.x, p.y, title, fg, bg, titleStyle)
	screen.DrawText(p.x+len(title)-1, p.y, strings.Repeat("─", p.width-len(title))+"┐", fg, bg, goterm.StyleNone)

	y := p.y + 1

	// Render log entries
	visibleEntries := p.entries[p.scrollOffset:]
	for i, entry := range visibleEntries {
		if y >= p.y+p.height-1 {
			break
		}

		// Format timestamp
		timeStr := entry.Timestamp.Format("15:04:05")

		// Level icon
		levelIcon := p.getLevelIcon(entry.Level)

		// Format line
		line := fmt.Sprintf("  %s %s %s", timeStr, levelIcon, entry.Message)
		if len(line) > p.width-2 {
			line = line[:p.width-5] + "..."
		}

		style := goterm.StyleNone
		if entry.Level == "error" {
			style = goterm.StyleBold
		}

		screen.DrawText(p.x+1, y, line, fg, bg, style)
		y++

		// Show scroll indicator
		if i == 0 && p.scrollOffset > 0 {
			scrollIndicator := fmt.Sprintf("  ↑ %d more entries above", p.scrollOffset)
			screen.DrawText(p.x+1, p.y+1, scrollIndicator, fg, bg, goterm.StyleDim)
		}
	}

	// Bottom border
	screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
}

func (p *LogViewerPanel) getLevelIcon(level string) string {
	switch level {
	case "error":
		return "✗"
	case "info":
		return "▶"
	case "debug":
		return "◆"
	default:
		return "•"
	}
}

// ErrorDetailPanel displays detailed error information.
type ErrorDetailPanel struct {
	x, y, width, height int
	error               *execution.ExecutionError
	scrollOffset        int
}

func NewErrorDetailPanel(x, y, width, height int) *ErrorDetailPanel {
	return &ErrorDetailPanel{
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
}

func (p *ErrorDetailPanel) SetError(err *execution.ExecutionError) {
	p.error = err
	// TODO: Convert to EnhancedExecutionError if available
}

func (p *ErrorDetailPanel) HasError() bool {
	return p.error != nil
}

func (p *ErrorDetailPanel) GetErrorDetails() *execution.ExecutionError {
	return p.error
}

func (p *ErrorDetailPanel) Scroll(delta int) {
	p.scrollOffset += delta
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

// Render draws the full error detail panel (full screen mode)
func (p *ErrorDetailPanel) Render(screen *goterm.Screen, active bool) {
	if p.error == nil {
		return
	}

	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	titleStyle := goterm.StyleBold
	screen.DrawText(p.x, p.y, "┌─ Error Details ", fg, bg, titleStyle)
	screen.DrawText(p.x+16, p.y, strings.Repeat("─", p.width-17)+"┐", fg, bg, goterm.StyleNone)

	y := p.y + 1

	// Error type and message
	errorType := fmt.Sprintf("  Type: %s", p.error.Type)
	screen.DrawText(p.x+1, y, errorType, fg, bg, goterm.StyleBold)
	y++

	message := fmt.Sprintf("  Message: %s", p.error.Message)
	if len(message) > p.width-2 {
		// Wrap message
		wrapped := p.wrapText(message, p.width-4)
		for _, line := range wrapped {
			if y >= p.y+p.height-1 {
				break
			}
			screen.DrawText(p.x+1, y, "  "+line, fg, bg, goterm.StyleNone)
			y++
		}
	} else {
		screen.DrawText(p.x+1, y, message, fg, bg, goterm.StyleNone)
		y++
	}

	// Node ID if available
	if p.error.NodeID != "" {
		nodeInfo := fmt.Sprintf("  Node: %s", p.error.NodeID)
		screen.DrawText(p.x+1, y, nodeInfo, fg, bg, goterm.StyleNone)
		y++
	}

	// Timestamp
	if !p.error.Timestamp.IsZero() {
		timeInfo := fmt.Sprintf("  Time: %s", p.error.Timestamp.Format(time.RFC3339))
		screen.DrawText(p.x+1, y, timeInfo, fg, bg, goterm.StyleDim)
		y++
	}

	// Context if available
	if len(p.error.Context) > 0 {
		y++
		screen.DrawText(p.x+1, y, "  Context:", fg, bg, goterm.StyleBold)
		y++

		for key, value := range p.error.Context {
			if y >= p.y+p.height-1 {
				break
			}
			contextLine := fmt.Sprintf("    %s: %v", key, value)
			if len(contextLine) > p.width-2 {
				contextLine = contextLine[:p.width-5] + "..."
			}
			screen.DrawText(p.x+1, y, contextLine, fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Recovery suggestion
	if p.error.Recoverable {
		y++
		screen.DrawText(p.x+1, y, "  This error is recoverable - retry may succeed", fg, bg, goterm.StyleDim)
	}

	// Bottom border
	screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
}

// RenderInline draws error info inline (embedded in normal view)
func (p *ErrorDetailPanel) RenderInline(screen *goterm.Screen) {
	if p.error == nil {
		return
	}

	// Get screen dimensions to place error at bottom
	width, height := screen.Size()
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Draw error notification at the bottom (above status bar)
	y := height - 3

	// Compact error line with type and message
	errorLine := fmt.Sprintf("✗ Error [%s]: %s", p.error.Type, p.error.Message)
	if len(errorLine) > width-2 {
		errorLine = errorLine[:width-5] + "..."
	}
	screen.DrawText(0, y, errorLine, fg, bg, goterm.StyleBold)

	// Show node ID if available
	if p.error.NodeID != "" {
		nodeLine := fmt.Sprintf("  Node: %s | Press 'e' for details", p.error.NodeID)
		screen.DrawText(0, y+1, nodeLine, fg, bg, goterm.StyleDim)
	} else {
		helpLine := "  Press 'e' for error details"
		screen.DrawText(0, y+1, helpLine, fg, bg, goterm.StyleDim)
	}
}

func (p *ErrorDetailPanel) wrapText(text string, maxWidth int) []string {
	words := strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= maxWidth {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// MetricsPanel displays performance metrics.
type MetricsPanel struct {
	x, y, width, height int
	progress            execpkg.ExecutionProgress
	exec                *execution.Execution
	metrics             map[string]interface{}
}

func NewMetricsPanel(x, y, width, height int) *MetricsPanel {
	return &MetricsPanel{
		x:       x,
		y:       y,
		width:   width,
		height:  height,
		metrics: make(map[string]interface{}),
	}
}

func (p *MetricsPanel) UpdateProgress(progress execpkg.ExecutionProgress) {
	p.progress = progress
}

func (p *MetricsPanel) UpdateExecution(exec *execution.Execution) {
	p.exec = exec

	// Calculate metrics
	p.metrics["Total Nodes"] = p.progress.TotalNodes
	p.metrics["Completed"] = p.progress.CompletedNodes
	p.metrics["Failed"] = p.progress.FailedNodes
	p.metrics["Skipped"] = p.progress.SkippedNodes
	p.metrics["Nodes Executed"] = p.progress.CompletedNodes + p.progress.FailedNodes + p.progress.SkippedNodes
	p.metrics["Status"] = exec.Status

	// Set duration based on execution state
	if !exec.CompletedAt.IsZero() {
		p.metrics["Duration"] = exec.Duration()
	} else if exec.Status == execution.StatusRunning {
		p.metrics["Duration"] = time.Since(exec.StartedAt)
	}

	// Add Failed Node info if execution failed
	if exec.Status == execution.StatusFailed && exec.Error != nil && exec.Error.NodeID != "" {
		p.metrics["Failed Node"] = exec.Error.NodeID
	}
}

func (p *MetricsPanel) GetProgress() execpkg.ExecutionProgress {
	return p.progress
}

func (p *MetricsPanel) GetMetrics() map[string]interface{} {
	return p.metrics
}

func (p *MetricsPanel) Render(screen *goterm.Screen, active bool) {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	titleStyle := goterm.StyleBold
	if active {
		titleStyle = goterm.StyleReverse
	}
	screen.DrawText(p.x, p.y, "┌─ Metrics ", fg, bg, titleStyle)
	screen.DrawText(p.x+10, p.y, strings.Repeat("─", p.width-11)+"┐", fg, bg, goterm.StyleNone)

	y := p.y + 1

	// Progress bar
	progressBar := p.renderProgressBar(int(p.progress.PercentComplete), p.width-4)
	progressText := fmt.Sprintf("  %s %.0f%%", progressBar, p.progress.PercentComplete)
	screen.DrawText(p.x+1, y, progressText, fg, bg, goterm.StyleNone)
	y++

	// Status
	if status, ok := p.metrics["Status"].(execution.Status); ok {
		statusLine := fmt.Sprintf("  Status: %s", status)
		screen.DrawText(p.x+1, y, statusLine, fg, bg, goterm.StyleNone)
		y++
	}

	// Nodes Executed
	if nodesExecuted, ok := p.metrics["Nodes Executed"].(int); ok {
		nodesLine := fmt.Sprintf("  Nodes Executed: %d/%d", nodesExecuted, p.progress.TotalNodes)
		screen.DrawText(p.x+1, y, nodesLine, fg, bg, goterm.StyleNone)
		y++
	}

	// Failed nodes
	if p.progress.FailedNodes > 0 {
		failedStatus := fmt.Sprintf("  Failed: %d", p.progress.FailedNodes)
		screen.DrawText(p.x+1, y, failedStatus, fg, bg, goterm.StyleBold)
		y++

		// Show Failed Node ID if available
		if failedNode, ok := p.metrics["Failed Node"].(types.NodeID); ok {
			failedNodeLine := fmt.Sprintf("  Failed Node: %s", failedNode)
			screen.DrawText(p.x+1, y, failedNodeLine, fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Duration / Total Time
	if duration, ok := p.metrics["Duration"].(time.Duration); ok {
		durationStr := fmt.Sprintf("  Duration: %v", duration.Round(time.Millisecond))
		screen.DrawText(p.x+1, y, durationStr, fg, bg, goterm.StyleNone)
		y++

		// Also show Total Time label for compatibility with tests
		totalTimeStr := fmt.Sprintf("  Total Time: %v", duration.Round(time.Millisecond))
		screen.DrawText(p.x+1, y, totalTimeStr, fg, bg, goterm.StyleNone)
		y++
	}

	// Show concurrent branches if available
	if val, ok := p.metrics["Active Branches"]; ok {
		var activeBranches int
		switch v := val.(type) {
		case int:
			activeBranches = v
		case int32:
			activeBranches = int(v)
		case int64:
			activeBranches = int(v)
		case float64:
			activeBranches = int(v)
		}
		if activeBranches > 0 {
			branchStatus := fmt.Sprintf("  Parallel: %d branches active", activeBranches)
			screen.DrawText(p.x+1, y, branchStatus, fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Show loop iteration if available
	if val, ok := p.metrics["Loop Iteration"]; ok {
		var iteration int
		switch v := val.(type) {
		case int:
			iteration = v
		case int32:
			iteration = int(v)
		case int64:
			iteration = int(v)
		case float64:
			iteration = int(v)
		}
		if iteration > 0 {
			iterStatus := fmt.Sprintf("  Loop: iteration %d", iteration)
			screen.DrawText(p.x+1, y, iterStatus, fg, bg, goterm.StyleNone)
			y++
		}
	}

	// Bottom border
	if y < p.y+p.height {
		screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
	}
}

func (p *MetricsPanel) renderProgressBar(percent, width int) string {
	if width < 10 {
		width = 10
	}

	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

// ExecutionHelpPanel displays keyboard shortcuts and usage information.
type ExecutionHelpPanel struct {
	x, y, width, height int
}

func NewExecutionHelpPanel(x, y, width, height int) *ExecutionHelpPanel {
	return &ExecutionHelpPanel{
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}
}

func (p *ExecutionHelpPanel) Render(screen *goterm.Screen) {
	fg := goterm.ColorDefault()
	bg := goterm.ColorDefault()

	// Border
	screen.DrawText(p.x, p.y, "┌─ Help ", fg, bg, goterm.StyleBold)
	screen.DrawText(p.x+7, p.y, strings.Repeat("─", p.width-8)+"┐", fg, bg, goterm.StyleNone)

	y := p.y + 1

	helpItems := []struct {
		key  string
		desc string
	}{
		{"Tab", "Switch between panels"},
		{"Shift+Tab", "Switch backward"},
		{"j / k", "Scroll down / up"},
		{"e", "Expand variable details"},
		{"Esc", "Close help or error view"},
		{"?", "Toggle help"},
		{"q", "Quit monitor"},
	}

	screen.DrawText(p.x+1, y, "Keyboard Shortcuts:", fg, bg, goterm.StyleBold)
	y += 2

	for _, item := range helpItems {
		if y >= p.y+p.height-1 {
			break
		}

		line := fmt.Sprintf("  %-12s - %s", item.key, item.desc)
		screen.DrawText(p.x+1, y, line, fg, bg, goterm.StyleNone)
		y++
	}

	y++
	screen.DrawText(p.x+1, y, "Panels:", fg, bg, goterm.StyleBold)
	y += 2

	panels := []struct {
		name string
		desc string
	}{
		{"Workflow", "Shows execution progress through workflow graph"},
		{"Variables", "Displays current variable values"},
		{"Metrics", "Shows performance and progress metrics"},
		{"Logs", "Chronological execution events"},
	}

	for _, panel := range panels {
		if y >= p.y+p.height-1 {
			break
		}

		line := fmt.Sprintf("  %-12s - %s", panel.name, panel.desc)
		screen.DrawText(p.x+1, y, line, fg, bg, goterm.StyleNone)
		y++
	}

	// Bottom border
	screen.DrawText(p.x, p.y+p.height-1, "└"+strings.Repeat("─", p.width-2)+"┘", fg, bg, goterm.StyleNone)
}
