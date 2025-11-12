package tui

// HelpKeyBinding represents a keyboard shortcut with its description
type HelpKeyBinding struct {
	Keys        []string // Key combinations (e.g., ["h", "j", "k", "l"])
	Description string   // What this key/combo does
	Category    string   // Category (e.g., "Navigation", "Editing")
	Mode        string   // Which mode this applies to ("normal", "edit", "palette", "*" for all)
}

// HelpPanel manages context-sensitive help display
type HelpPanel struct {
	visible        bool             // Panel open/closed
	currentSection string           // Current help section ("general", "navigation", etc.)
	keyBindings    []HelpKeyBinding // All available key bindings
	scrollOffset   int              // Scroll position for long help text
	maxScroll      int              // Maximum scroll offset
}

// NewHelpPanel creates a new help panel with default key bindings
func NewHelpPanel() *HelpPanel {
	panel := &HelpPanel{
		visible:        false,
		currentSection: "general",
		keyBindings:    make([]HelpKeyBinding, 0),
		scrollOffset:   0,
		maxScroll:      0,
	}

	// Initialize with default key bindings
	panel.initializeKeyBindings()

	return panel
}

// Toggle toggles the visibility of the help panel
func (h *HelpPanel) Toggle() {
	h.visible = !h.visible
}

// Show makes the help panel visible
func (h *HelpPanel) Show() {
	h.visible = true
}

// Hide makes the help panel hidden
func (h *HelpPanel) Hide() {
	h.visible = false
}

// IsVisible returns whether the panel is visible
func (h *HelpPanel) IsVisible() bool {
	return h.visible
}

// SetSection changes the current help section
func (h *HelpPanel) SetSection(section string) {
	h.currentSection = section
	h.scrollOffset = 0 // Reset scroll when changing sections
}

// GetSection returns the current section
func (h *HelpPanel) GetSection() string {
	return h.currentSection
}

// ScrollDown scrolls the help content down one line
func (h *HelpPanel) ScrollDown() {
	// If maxScroll is set (> 0), respect it; otherwise allow unlimited scrolling
	if h.maxScroll == 0 || h.scrollOffset < h.maxScroll {
		h.scrollOffset++
	}
}

// ScrollUp scrolls the help content up one line
func (h *HelpPanel) ScrollUp() {
	if h.scrollOffset > 0 {
		h.scrollOffset--
	}
}

// SetMaxScroll sets the maximum scroll offset based on content height
func (h *HelpPanel) SetMaxScroll(max int) {
	h.maxScroll = max
	if h.scrollOffset > max {
		h.scrollOffset = max
	}
}

// GetScrollOffset returns the current scroll offset
func (h *HelpPanel) GetScrollOffset() int {
	return h.scrollOffset
}

// GetBindingsForMode returns all key bindings for a specific mode
// Mode can be "normal", "edit", "palette", or "*" for global bindings
func (h *HelpPanel) GetBindingsForMode(mode string) []HelpKeyBinding {
	bindings := make([]HelpKeyBinding, 0)

	for _, binding := range h.keyBindings {
		if binding.Mode == mode || binding.Mode == "*" {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

// GetBindingsForCategory returns all key bindings for a specific category
func (h *HelpPanel) GetBindingsForCategory(category string) []HelpKeyBinding {
	bindings := make([]HelpKeyBinding, 0)

	for _, binding := range h.keyBindings {
		if binding.Category == category {
			bindings = append(bindings, binding)
		}
	}

	return bindings
}

// LookupKeyBinding finds a key binding by key and mode
// Returns nil if not found
func (h *HelpPanel) LookupKeyBinding(key, mode string) *HelpKeyBinding {
	for _, binding := range h.keyBindings {
		// Check if binding applies to this mode
		if binding.Mode != mode && binding.Mode != "*" {
			continue
		}

		// Check if key matches any of the binding's keys
		for _, bindingKey := range binding.Keys {
			if bindingKey == key {
				return &binding
			}
		}
	}

	return nil
}

// AddKeyBinding adds a custom key binding to the help panel
func (h *HelpPanel) AddKeyBinding(binding HelpKeyBinding) {
	h.keyBindings = append(h.keyBindings, binding)
}

// initializeKeyBindings populates default key bindings
func (h *HelpPanel) initializeKeyBindings() {
	// Global bindings (available in all modes)
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"?"},
			Description: "Toggle help panel",
			Category:    "General",
			Mode:        "*",
		},
		{
			Keys:        []string{"Esc"},
			Description: "Close panel / Cancel operation",
			Category:    "General",
			Mode:        "*",
		},
		{
			Keys:        []string{"q"},
			Description: "Quit application",
			Category:    "General",
			Mode:        "*",
		},
	}...)

	// Navigation bindings (normal mode)
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"h/j/k/l"},
			Description: "Move selection (left/down/up/right)",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Shift+←"},
			Description: "Pan canvas left",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Shift+↑"},
			Description: "Pan canvas up",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Shift+↓"},
			Description: "Pan canvas down",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Shift+→"},
			Description: "Pan canvas right",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"+"},
			Description: "Zoom in",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"-"},
			Description: "Zoom out",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"0"},
			Description: "Reset view (100% zoom)",
			Category:    "Navigation",
			Mode:        "normal",
		},
		{
			Keys:        []string{"f"},
			Description: "Fit all nodes in viewport",
			Category:    "Navigation",
			Mode:        "normal",
		},
	}...)

	// Node operation bindings (normal mode)
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"a"},
			Description: "Add node",
			Category:    "Node Operations",
			Mode:        "normal",
		},
		{
			Keys:        []string{"d"},
			Description: "Delete selected node",
			Category:    "Node Operations",
			Mode:        "normal",
		},
		{
			Keys:        []string{"c"},
			Description: "Create edge from selected node",
			Category:    "Node Operations",
			Mode:        "normal",
		},
		{
			Keys:        []string{"y"},
			Description: "Yank (copy) selected node",
			Category:    "Node Operations",
			Mode:        "normal",
		},
		{
			Keys:        []string{"p"},
			Description: "Paste node",
			Category:    "Node Operations",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Enter"},
			Description: "Edit node properties",
			Category:    "Node Operations",
			Mode:        "normal",
		},
	}...)

	// Workflow operation bindings (normal mode)
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"s"},
			Description: "Save workflow",
			Category:    "Workflow",
			Mode:        "normal",
		},
		{
			Keys:        []string{"v"},
			Description: "Validate workflow",
			Category:    "Workflow",
			Mode:        "normal",
		},
		{
			Keys:        []string{"u"},
			Description: "Undo last change",
			Category:    "Workflow",
			Mode:        "normal",
		},
		{
			Keys:        []string{"Ctrl+R"},
			Description: "Redo last undo",
			Category:    "Workflow",
			Mode:        "normal",
		},
		{
			Keys:        []string{"t"},
			Description: "Apply template",
			Category:    "Workflow",
			Mode:        "normal",
		},
	}...)

	// Edit mode bindings
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"Tab"},
			Description: "Next field",
			Category:    "Editing",
			Mode:        "edit",
		},
		{
			Keys:        []string{"Shift+Tab"},
			Description: "Previous field",
			Category:    "Editing",
			Mode:        "edit",
		},
		{
			Keys:        []string{"Enter"},
			Description: "Edit field / Confirm",
			Category:    "Editing",
			Mode:        "edit",
		},
		{
			Keys:        []string{"Ctrl+S"},
			Description: "Save changes",
			Category:    "Editing",
			Mode:        "edit",
		},
		{
			Keys:        []string{"Esc"},
			Description: "Cancel editing",
			Category:    "Editing",
			Mode:        "edit",
		},
	}...)

	// Palette mode bindings
	h.keyBindings = append(h.keyBindings, []HelpKeyBinding{
		{
			Keys:        []string{"↓", "j"},
			Description: "Next node type",
			Category:    "Node Palette",
			Mode:        "palette",
		},
		{
			Keys:        []string{"↑", "k"},
			Description: "Previous node type",
			Category:    "Node Palette",
			Mode:        "palette",
		},
		{
			Keys:        []string{"Enter"},
			Description: "Select node type",
			Category:    "Node Palette",
			Mode:        "palette",
		},
		{
			Keys:        []string{"Esc"},
			Description: "Cancel node creation",
			Category:    "Node Palette",
			Mode:        "palette",
		},
	}...)
}
