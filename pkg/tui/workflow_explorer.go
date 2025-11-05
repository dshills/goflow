package tui

import (
	"fmt"
	"strings"

	"github.com/dshills/goflow/pkg/tui/components"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
)

// WorkflowExplorer displays and manages a list of workflows
type WorkflowExplorer struct {
	repo              workflow.WorkflowRepository
	screen            *goterm.Screen
	workflows         []*workflow.Workflow
	filteredWorkflows []*workflow.Workflow
	selectedIndex     int
	searchMode        bool
	searchQuery       string
	currentModal      *components.Modal

	// Callbacks
	onSelectCallback            func(*workflow.Workflow)
	onNewWorkflowDialogCallback func()
	onDeleteConfirmCallback     func(*workflow.Workflow) bool
	onRenameDialogCallback      func(*workflow.Workflow) string
}

// NewWorkflowExplorer creates a new workflow explorer
func NewWorkflowExplorer(repo workflow.WorkflowRepository, screen *goterm.Screen) *WorkflowExplorer {
	explorer := &WorkflowExplorer{
		repo:              repo,
		screen:            screen,
		workflows:         []*workflow.Workflow{},
		filteredWorkflows: []*workflow.Workflow{},
		selectedIndex:     0,
		searchMode:        false,
		searchQuery:       "",
	}

	// Load workflows from repository
	explorer.loadWorkflows()

	return explorer
}

// loadWorkflows loads all workflows from the repository
func (e *WorkflowExplorer) loadWorkflows() error {
	workflows, err := e.repo.List()
	if err != nil {
		return err
	}

	e.workflows = workflows
	e.updateFilteredWorkflows()

	// Ensure selected index is valid
	if e.selectedIndex >= len(e.filteredWorkflows) {
		if len(e.filteredWorkflows) > 0 {
			e.selectedIndex = len(e.filteredWorkflows) - 1
		} else {
			e.selectedIndex = 0
		}
	}

	return nil
}

// updateFilteredWorkflows updates the filtered workflow list based on search query
func (e *WorkflowExplorer) updateFilteredWorkflows() {
	if e.searchQuery == "" {
		e.filteredWorkflows = e.workflows
		return
	}

	query := strings.ToLower(e.searchQuery)
	filtered := []*workflow.Workflow{}

	for _, wf := range e.workflows {
		// Search in name, description, and tags
		if strings.Contains(strings.ToLower(wf.Name), query) ||
			strings.Contains(strings.ToLower(wf.Description), query) {
			filtered = append(filtered, wf)
			continue
		}

		// Search in tags
		for _, tag := range wf.Metadata.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				filtered = append(filtered, wf)
				break
			}
		}
	}

	e.filteredWorkflows = filtered

	// Adjust selected index if needed
	if e.selectedIndex >= len(e.filteredWorkflows) {
		if len(e.filteredWorkflows) > 0 {
			e.selectedIndex = len(e.filteredWorkflows) - 1
		} else {
			e.selectedIndex = 0
		}
	}
}

// SetSelectedIndex sets the selected workflow index
func (e *WorkflowExplorer) SetSelectedIndex(index int) {
	if index >= 0 && index < len(e.filteredWorkflows) {
		e.selectedIndex = index
	}
}

// GetSelectedIndex returns the currently selected workflow index
func (e *WorkflowExplorer) GetSelectedIndex() int {
	return e.selectedIndex
}

// GetSelectedWorkflow returns the currently selected workflow
func (e *WorkflowExplorer) GetSelectedWorkflow() *workflow.Workflow {
	if e.selectedIndex >= 0 && e.selectedIndex < len(e.filteredWorkflows) {
		return e.filteredWorkflows[e.selectedIndex]
	}
	return nil
}

// GetFilteredWorkflows returns the filtered workflow list
func (e *WorkflowExplorer) GetFilteredWorkflows() []*workflow.Workflow {
	return e.filteredWorkflows
}

// IsSearchMode returns whether search mode is active
func (e *WorkflowExplorer) IsSearchMode() bool {
	return e.searchMode
}

// SetScreen updates the screen reference
func (e *WorkflowExplorer) SetScreen(screen *goterm.Screen) {
	e.screen = screen
}

// OnSelect registers a callback for workflow selection
func (e *WorkflowExplorer) OnSelect(callback func(*workflow.Workflow)) {
	e.onSelectCallback = callback
}

// OnNewWorkflowDialog registers a callback for new workflow dialog
func (e *WorkflowExplorer) OnNewWorkflowDialog(callback func()) {
	e.onNewWorkflowDialogCallback = callback
}

// OnDeleteConfirmation registers a callback for delete confirmation
func (e *WorkflowExplorer) OnDeleteConfirmation(callback func(*workflow.Workflow) bool) {
	e.onDeleteConfirmCallback = callback
}

// OnRenameDialog registers a callback for rename dialog
func (e *WorkflowExplorer) OnRenameDialog(callback func(*workflow.Workflow) string) {
	e.onRenameDialogCallback = callback
}

// HandleKey processes keyboard input
func (e *WorkflowExplorer) HandleKey(key rune) error {
	// If modal is open, handle modal input
	if e.currentModal != nil && e.currentModal.IsVisible() {
		keyStr := string(key)
		if key == '\n' {
			keyStr = "Enter"
		} else if key == 27 {
			keyStr = "Esc"
		} else if key == 8 || key == 127 {
			keyStr = "Backspace"
		}
		e.currentModal.HandleKey(keyStr)
		return nil
	}

	// Handle search mode
	if e.searchMode {
		switch key {
		case '\n': // Enter - execute search
			e.searchMode = false
		case 27: // ESC - cancel search
			e.searchMode = false
			e.searchQuery = ""
			e.updateFilteredWorkflows()
		case 8, 127: // Backspace
			if len(e.searchQuery) > 0 {
				e.searchQuery = e.searchQuery[:len(e.searchQuery)-1]
				e.updateFilteredWorkflows()
			}
		default:
			// Add character to search query
			if key >= 32 && key <= 126 {
				e.searchQuery += string(key)
				e.updateFilteredWorkflows()
			}
		}
		return nil
	}

	// Normal mode navigation
	switch key {
	case 'j': // Move down
		if len(e.filteredWorkflows) > 0 && e.selectedIndex < len(e.filteredWorkflows)-1 {
			e.selectedIndex++
		}
	case 'k': // Move up
		if len(e.filteredWorkflows) > 0 && e.selectedIndex > 0 {
			e.selectedIndex--
		}
	case '\n': // Enter - select workflow
		if e.selectedIndex >= 0 && e.selectedIndex < len(e.filteredWorkflows) {
			selected := e.filteredWorkflows[e.selectedIndex]
			if e.onSelectCallback != nil {
				e.onSelectCallback(selected)
			}
		}
	case '/': // Enter search mode
		e.searchMode = true
		e.searchQuery = ""
	case 'n': // New workflow
		e.showNewWorkflowDialog()
	case 'd': // Delete workflow
		e.showDeleteConfirmation()
	case 'r': // Rename workflow
		e.showRenameDialog()
	case '?': // Help
		e.showHelp()
	}

	return nil
}

// showNewWorkflowDialog shows the new workflow creation dialog
func (e *WorkflowExplorer) showNewWorkflowDialog() {
	if e.onNewWorkflowDialogCallback != nil {
		e.onNewWorkflowDialogCallback()
	}

	modal := components.NewInputModal(
		"New Workflow",
		"Name:\nDescription: (optional - can be added later)",
		"",
		func(confirmed bool, input string) {
			if confirmed && input != "" {
				// Validate name
				if !isValidWorkflowName(input) {
					return
				}

				// Check for duplicate names
				for _, wf := range e.workflows {
					if wf.Name == input {
						return
					}
				}

				// Create new workflow
				newWf, err := workflow.NewWorkflow(input, "")
				if err != nil {
					return
				}

				// Save to repository
				err = e.repo.Save(newWf)
				if err != nil {
					return
				}

				// Reload workflows and select the new one
				e.loadWorkflows()
				for i, wf := range e.filteredWorkflows {
					if wf.Name == input {
						e.selectedIndex = i
						break
					}
				}
			}
			e.currentModal = nil
		},
	)

	e.currentModal = modal
	modal.Show()
}

// showDeleteConfirmation shows the delete confirmation dialog
func (e *WorkflowExplorer) showDeleteConfirmation() {
	selected := e.GetSelectedWorkflow()
	if selected == nil {
		return
	}

	confirmed := false
	if e.onDeleteConfirmCallback != nil {
		confirmed = e.onDeleteConfirmCallback(selected)
	} else {
		// Default confirmation behavior
		message := fmt.Sprintf("Delete workflow '%s'? (y/n)", selected.Name)
		modal := components.NewConfirmModal(
			"Delete Workflow",
			message,
			func(result bool) {
				confirmed = result
				e.currentModal = nil
			},
		)
		e.currentModal = modal
		modal.Show()
	}

	// If we have a callback, it handles the confirmation logic
	if e.onDeleteConfirmCallback != nil {
		if confirmed {
			err := e.repo.Delete(selected.ID)
			if err == nil {
				e.loadWorkflows()
			}
		}
	} else {
		// For modal-based confirmation, deletion happens in callback
		modal := components.NewConfirmModal(
			"Delete Workflow",
			fmt.Sprintf("Delete workflow '%s'?", selected.Name),
			func(result bool) {
				if result {
					err := e.repo.Delete(selected.ID)
					if err == nil {
						e.loadWorkflows()
					}
				}
				e.currentModal = nil
			},
		)
		e.currentModal = modal
		modal.Show()
	}
}

// showRenameDialog shows the rename workflow dialog
func (e *WorkflowExplorer) showRenameDialog() {
	selected := e.GetSelectedWorkflow()
	if selected == nil {
		return
	}

	// If callback is registered, use it
	if e.onRenameDialogCallback != nil {
		e.onRenameDialogCallback(selected)
		newName := e.onRenameDialogCallback(selected)
		if newName != "" && newName != selected.Name {
			// Validate name
			if !isValidWorkflowName(newName) {
				return
			}

			// Check for duplicate names (excluding current)
			for _, wf := range e.workflows {
				if wf.Name == newName && wf.ID != selected.ID {
					return
				}
			}

			// Update workflow name
			selected.Name = newName
			err := e.repo.Save(selected)
			if err == nil {
				e.loadWorkflows()
			}
		}
		return
	}

	// Default modal-based rename
	modal := components.NewInputModal(
		"Rename Workflow",
		"Enter new name:",
		selected.Name,
		func(confirmed bool, input string) {
			if confirmed && input != "" && input != selected.Name {
				// Validate name
				if !isValidWorkflowName(input) {
					e.currentModal = nil
					return
				}

				// Check for duplicate names (excluding current)
				for _, wf := range e.workflows {
					if wf.Name == input && wf.ID != selected.ID {
						e.currentModal = nil
						return
					}
				}

				// Update workflow name
				selected.Name = input
				err := e.repo.Save(selected)
				if err == nil {
					e.loadWorkflows()
				}
			}
			e.currentModal = nil
		},
	)

	e.currentModal = modal
	modal.Show()
}

// showHelp shows the help dialog
func (e *WorkflowExplorer) showHelp() {
	helpText := `Keyboard Shortcuts

Navigate:
  j/k    Navigate up/down
  Enter  Select workflow

Create:
  n      Create new workflow

Delete:
  d      Delete workflow

Rename:
  r      Rename workflow

Search:
  /      Search workflows`

	modal := components.NewInfoModal(
		"Help",
		helpText,
		func() {
			e.currentModal = nil
		},
	)

	e.currentModal = modal
	modal.Show()
}

// Render renders the workflow explorer to the screen
func (e *WorkflowExplorer) Render() (string, error) {
	if e.screen == nil {
		return "", fmt.Errorf("screen not initialized")
	}

	width, height := e.screen.Size()

	// Clear screen
	e.screen.Clear()

	fg := goterm.ColorRGB(220, 220, 220)
	bg := goterm.ColorDefault()

	// Draw title bar
	titleText := "Workflow Explorer"
	for i, ch := range titleText {
		if i >= width {
			break
		}
		e.screen.SetCell(i, 0, goterm.NewCell(ch, fg, bg, goterm.StyleBold))
	}

	// Draw help text on the right side of title bar
	helpText := "[j/k: Navigate] [Enter: Select] [n: New] [d: Delete] [r: Rename] [/: Search] [?: Help]"

	// Position help text after title
	helpStartX := len(titleText) + 2
	if width >= helpStartX+len(helpText) {
		// Full help text fits
		for i, ch := range helpText {
			x := helpStartX + i
			if x < width {
				e.screen.SetCell(x, 0, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
			}
		}
	} else if width >= 60 {
		// Medium screen - show abbreviated help
		helpText = "[j/k] [Enter] [n] [d] [r] [/] [?]"
		for i, ch := range helpText {
			x := helpStartX + i
			if x < width {
				e.screen.SetCell(x, 0, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
			}
		}
	}

	contentY := 2

	// Draw search bar if in search mode
	if e.searchMode {
		searchText := "Search: " + e.searchQuery + "_"
		for i, ch := range searchText {
			if i >= width {
				break
			}
			e.screen.SetCell(i, contentY, goterm.NewCell(ch, goterm.ColorRGB(255, 255, 0), goterm.ColorRGB(40, 40, 40), goterm.StyleNone))
		}
		contentY++
	} else if e.searchQuery != "" {
		// Show active search filter
		searchText := fmt.Sprintf("Filter: %s (press / to edit)", e.searchQuery)
		for i, ch := range searchText {
			if i >= width {
				break
			}
			e.screen.SetCell(i, contentY, goterm.NewCell(ch, goterm.ColorRGB(200, 200, 100), bg, goterm.StyleNone))
		}
		contentY++
	}

	contentY++ // Add spacing

	// Draw workflow list or empty state
	if len(e.filteredWorkflows) == 0 {
		emptyMsg := "No workflows found"
		helpMsg := "Press 'n' to create a new workflow"
		if width < 40 {
			helpMsg = "Press 'n' to create"
		}

		// Center empty message
		emptyY := height / 2
		emptyX := (width - len(emptyMsg)) / 2
		if emptyX < 0 {
			emptyX = 0
		}
		for i, ch := range emptyMsg {
			if emptyX+i >= width {
				break
			}
			e.screen.SetCell(emptyX+i, emptyY, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
		}

		helpX := (width - len(helpMsg)) / 2
		if helpX < 0 {
			helpX = 0
		}
		for i, ch := range helpMsg {
			if helpX+i >= width {
				break
			}
			e.screen.SetCell(helpX+i, emptyY+1, goterm.NewCell(ch, goterm.ColorRGB(100, 100, 100), bg, goterm.StyleNone))
		}
	} else {
		// Draw workflow list
		y := contentY
		for i, wf := range e.filteredWorkflows {
			if y >= height-2 {
				break // Leave room for status bar
			}

			isSelected := i == e.selectedIndex
			itemFg := fg
			itemBg := bg
			style := goterm.StyleNone

			if isSelected {
				itemFg = goterm.ColorRGB(0, 0, 0)
				itemBg = goterm.ColorRGB(100, 200, 255)
				style = goterm.StyleBold
			}

			// Format: [icon] name - description
			icon := wf.Metadata.Icon
			if icon == "" {
				icon = "📄"
			}

			prefix := "  "
			if isSelected {
				prefix = "► "
			}

			line := fmt.Sprintf("%s%s %s", prefix, icon, wf.Name)
			if len(wf.Description) > 0 && width > 40 {
				maxDescLen := width - len(line) - 5
				desc := wf.Description
				if len(desc) > maxDescLen && maxDescLen > 3 {
					desc = desc[:maxDescLen-3] + "..."
				}
				line += " - " + desc
			}

			// Truncate if too long
			if len(line) > width {
				line = line[:width]
			}

			// Pad to full width for selection highlight
			if isSelected {
				line += strings.Repeat(" ", width-len(line))
			}

			for x, ch := range line {
				if x >= width {
					break
				}
				e.screen.SetCell(x, y, goterm.NewCell(ch, itemFg, itemBg, style))
			}

			y++
		}
	}

	// Draw status bar
	statusY := height - 1
	statusText := fmt.Sprintf("%d workflow", len(e.filteredWorkflows))
	if len(e.filteredWorkflows) != 1 {
		statusText += "s"
	}

	for i, ch := range statusText {
		if i >= width {
			break
		}
		e.screen.SetCell(i, statusY, goterm.NewCell(ch, goterm.ColorRGB(150, 150, 150), bg, goterm.StyleNone))
	}

	// Render modal if open
	if e.currentModal != nil && e.currentModal.IsVisible() {
		e.currentModal.Render(e.screen)
	}

	return "", nil
}

// isValidWorkflowName checks if a workflow name is valid
func isValidWorkflowName(name string) bool {
	if name == "" {
		return false
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_') {
			return false
		}
	}

	return true
}
