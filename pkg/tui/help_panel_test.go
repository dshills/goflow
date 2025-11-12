package tui

import (
	"testing"
)

func TestHelpPanel_NewHelpPanel(t *testing.T) {
	panel := NewHelpPanel()

	if panel == nil {
		t.Fatal("NewHelpPanel returned nil")
	}

	if panel.visible {
		t.Error("expected panel to be hidden initially")
	}

	if panel.currentSection != "general" {
		t.Errorf("expected current section 'general', got '%s'", panel.currentSection)
	}

	if len(panel.keyBindings) == 0 {
		t.Error("expected key bindings to be populated")
	}
}

func TestHelpPanel_Toggle(t *testing.T) {
	panel := NewHelpPanel()

	// Initially hidden
	if panel.visible {
		t.Error("expected panel to be hidden initially")
	}

	// Toggle to show
	panel.Toggle()
	if !panel.visible {
		t.Error("expected panel to be visible after toggle")
	}

	// Toggle to hide
	panel.Toggle()
	if panel.visible {
		t.Error("expected panel to be hidden after second toggle")
	}
}

func TestHelpPanel_Show(t *testing.T) {
	panel := NewHelpPanel()

	panel.Show()
	if !panel.visible {
		t.Error("expected panel to be visible after Show()")
	}

	// Calling Show again should keep it visible
	panel.Show()
	if !panel.visible {
		t.Error("expected panel to remain visible")
	}
}

func TestHelpPanel_Hide(t *testing.T) {
	panel := NewHelpPanel()

	panel.Show()
	panel.Hide()
	if panel.visible {
		t.Error("expected panel to be hidden after Hide()")
	}

	// Calling Hide again should keep it hidden
	panel.Hide()
	if panel.visible {
		t.Error("expected panel to remain hidden")
	}
}

func TestHelpPanel_SetSection(t *testing.T) {
	panel := NewHelpPanel()

	sections := []string{"general", "navigation", "node", "edit", "palette"}

	for _, section := range sections {
		panel.SetSection(section)
		if panel.currentSection != section {
			t.Errorf("expected section '%s', got '%s'", section, panel.currentSection)
		}
	}
}

func TestHelpPanel_GetBindingsForMode(t *testing.T) {
	panel := NewHelpPanel()

	tests := []struct {
		name          string
		mode          string
		expectCount   int
		shouldContain []string
	}{
		{
			name:          "normal mode",
			mode:          "normal",
			expectCount:   10, // At least 10 bindings for normal mode
			shouldContain: []string{"h/j/k/l", "a", "d", "s"},
		},
		{
			name:          "edit mode",
			mode:          "edit",
			expectCount:   5, // At least 5 bindings for edit mode
			shouldContain: []string{"Tab", "Esc", "Ctrl+S"},
		},
		{
			name:          "palette mode",
			mode:          "palette",
			expectCount:   3, // At least 3 bindings for palette mode
			shouldContain: []string{"Enter", "Esc"},
		},
		{
			name:          "all modes (*)",
			mode:          "*",
			expectCount:   3, // At least 3 global bindings
			shouldContain: []string{"?", "Esc", "q"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings := panel.GetBindingsForMode(tt.mode)

			if len(bindings) < tt.expectCount {
				t.Errorf("expected at least %d bindings for mode '%s', got %d",
					tt.expectCount, tt.mode, len(bindings))
			}

			// Check that expected bindings are present
			for _, expected := range tt.shouldContain {
				found := false
				for _, binding := range bindings {
					for _, key := range binding.Keys {
						if key == expected {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					t.Errorf("expected to find binding '%s' in mode '%s'", expected, tt.mode)
				}
			}
		})
	}
}

func TestHelpPanel_GetBindingsForCategory(t *testing.T) {
	panel := NewHelpPanel()

	tests := []struct {
		name          string
		category      string
		expectCount   int
		shouldContain []string
	}{
		{
			name:          "navigation category",
			category:      "Navigation",
			expectCount:   5,
			shouldContain: []string{"h/j/k/l"},
		},
		{
			name:          "editing category",
			category:      "Editing",
			expectCount:   3,
			shouldContain: []string{"Enter", "Esc"},
		},
		{
			name:          "workflow category",
			category:      "Workflow",
			expectCount:   5,
			shouldContain: []string{"s", "v", "u"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings := panel.GetBindingsForCategory(tt.category)

			if len(bindings) < tt.expectCount {
				t.Errorf("expected at least %d bindings for category '%s', got %d",
					tt.expectCount, tt.category, len(bindings))
			}

			// Check that expected bindings are present
			for _, expected := range tt.shouldContain {
				found := false
				for _, binding := range bindings {
					for _, key := range binding.Keys {
						if key == expected {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					t.Errorf("expected to find binding '%s' in category '%s'", expected, tt.category)
				}
			}
		})
	}
}

func TestHelpPanel_LookupKeyBinding(t *testing.T) {
	panel := NewHelpPanel()

	tests := []struct {
		name          string
		key           string
		mode          string
		shouldFind    bool
		expectedDescr string
	}{
		{
			name:          "find help key in all modes",
			key:           "?",
			mode:          "*",
			shouldFind:    true,
			expectedDescr: "Toggle help panel",
		},
		{
			name:          "find add node in normal mode",
			key:           "a",
			mode:          "normal",
			shouldFind:    true,
			expectedDescr: "Add node",
		},
		{
			name:          "find save in normal mode",
			key:           "s",
			mode:          "normal",
			shouldFind:    true,
			expectedDescr: "Save workflow",
		},
		{
			name:       "key not in mode",
			key:        "a",
			mode:       "edit",
			shouldFind: false,
		},
		{
			name:       "non-existent key",
			key:        "xyz",
			mode:       "normal",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := panel.LookupKeyBinding(tt.key, tt.mode)

			if tt.shouldFind {
				if binding == nil {
					t.Errorf("expected to find binding for key '%s' in mode '%s'", tt.key, tt.mode)
					return
				}
				if binding.Description != tt.expectedDescr {
					t.Errorf("expected description '%s', got '%s'", tt.expectedDescr, binding.Description)
				}
			} else {
				if binding != nil {
					t.Errorf("expected not to find binding for key '%s' in mode '%s'", tt.key, tt.mode)
				}
			}
		})
	}
}

func TestHelpPanel_ContentCompleteness(t *testing.T) {
	panel := NewHelpPanel()

	// Verify all required modes have bindings
	requiredModes := []string{"normal", "edit", "palette", "*"}
	for _, mode := range requiredModes {
		bindings := panel.GetBindingsForMode(mode)
		if len(bindings) == 0 {
			t.Errorf("mode '%s' has no key bindings", mode)
		}
	}

	// Verify all required categories have bindings
	requiredCategories := []string{"Navigation", "Editing", "Workflow", "Node Operations"}
	for _, category := range requiredCategories {
		bindings := panel.GetBindingsForCategory(category)
		if len(bindings) == 0 {
			t.Errorf("category '%s' has no key bindings", category)
		}
	}

	// Verify essential keys are present
	essentialKeys := []struct {
		key  string
		mode string
	}{
		{"?", "*"},
		{"Esc", "*"},
		{"q", "*"},
		{"h/j/k/l", "normal"},
		{"a", "normal"},
		{"d", "normal"},
		{"s", "normal"},
		{"Enter", "edit"},
		{"Tab", "edit"},
	}

	for _, essential := range essentialKeys {
		binding := panel.LookupKeyBinding(essential.key, essential.mode)
		if binding == nil {
			t.Errorf("essential key '%s' not found in mode '%s'", essential.key, essential.mode)
		}
	}
}

func TestHelpPanel_ScrollState(t *testing.T) {
	panel := NewHelpPanel()

	// Initially at top
	if panel.scrollOffset != 0 {
		t.Error("expected initial scroll offset to be 0")
	}

	// Scroll down
	panel.ScrollDown()
	if panel.scrollOffset != 1 {
		t.Errorf("expected scroll offset 1, got %d", panel.scrollOffset)
	}

	// Scroll up
	panel.ScrollUp()
	if panel.scrollOffset != 0 {
		t.Errorf("expected scroll offset 0 after scrolling up, got %d", panel.scrollOffset)
	}

	// Can't scroll up past 0
	panel.ScrollUp()
	if panel.scrollOffset != 0 {
		t.Errorf("expected scroll offset to stay at 0, got %d", panel.scrollOffset)
	}
}

func TestHelpPanel_ScrollWithContent(t *testing.T) {
	panel := NewHelpPanel()

	// Set a max scroll based on content
	maxScroll := 10
	panel.SetMaxScroll(maxScroll)

	// Scroll to max
	for i := 0; i < maxScroll+5; i++ {
		panel.ScrollDown()
	}

	if panel.scrollOffset > maxScroll {
		t.Errorf("scroll offset exceeded max: expected max %d, got %d", maxScroll, panel.scrollOffset)
	}

	// Scroll back to top
	for i := 0; i < maxScroll+5; i++ {
		panel.ScrollUp()
	}

	if panel.scrollOffset != 0 {
		t.Errorf("expected scroll offset 0 after scrolling to top, got %d", panel.scrollOffset)
	}
}
