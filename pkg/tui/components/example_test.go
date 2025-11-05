package components_test

import (
	"testing"

	"github.com/dshills/goflow/pkg/tui/components"
	"github.com/dshills/goterm"
)

// TestButtonCreation tests creating a button
func TestButtonCreation(t *testing.T) {
	clicked := false
	button := components.NewButton("Test", 0, 0, func() {
		clicked = true
	})

	if button == nil {
		t.Fatal("NewButton returned nil")
	}

	if button.GetLabel() != "Test" {
		t.Errorf("GetLabel() = %v, want %v", button.GetLabel(), "Test")
	}

	if !button.IsEnabled() {
		t.Error("Button should be enabled by default")
	}

	button.Activate()
	if !clicked {
		t.Error("Button callback was not invoked")
	}
}

// TestButtonKeyHandling tests button keyboard interaction
func TestButtonKeyHandling(t *testing.T) {
	activated := false
	button := components.NewButton("Test", 0, 0, func() {
		activated = true
	})

	// Enter should activate
	if !button.HandleKey("Enter") {
		t.Error("HandleKey(Enter) should return true")
	}

	if !activated {
		t.Error("Enter should activate button")
	}

	// Space should also activate
	activated = false
	if !button.HandleKey(" ") {
		t.Error("HandleKey(Space) should return true")
	}

	if !activated {
		t.Error("Space should activate button")
	}

	// Disabled button should not activate
	activated = false
	button.SetEnabled(false)
	button.HandleKey("Enter")

	if activated {
		t.Error("Disabled button should not activate")
	}
}

// TestPanelCreation tests creating a panel
func TestPanelCreation(t *testing.T) {
	panel := components.NewPanel("Test Panel", 0, 0, 40, 20)

	if panel == nil {
		t.Fatal("NewPanel returned nil")
	}

	if panel.GetTitle() != "Test Panel" {
		t.Errorf("GetTitle() = %v, want %v", panel.GetTitle(), "Test Panel")
	}

	width, height := panel.GetSize()
	if width != 40 || height != 20 {
		t.Errorf("GetSize() = (%v, %v), want (40, 20)", width, height)
	}

	if !panel.HasBorder() {
		t.Error("Panel should have border by default")
	}
}

// TestPanelScrolling tests panel scrolling
func TestPanelScrolling(t *testing.T) {
	panel := components.NewPanel("Test", 0, 0, 40, 5)

	// Add more content than can fit
	content := []string{
		"Line 1", "Line 2", "Line 3", "Line 4", "Line 5",
		"Line 6", "Line 7", "Line 8", "Line 9", "Line 10",
	}
	panel.SetContent(content)

	// Initially at top
	if panel.GetScrollPosition() != 0 {
		t.Errorf("Initial scroll position = %v, want 0", panel.GetScrollPosition())
	}

	// Scroll down
	panel.ScrollDown(1)
	if panel.GetScrollPosition() != 1 {
		t.Errorf("After ScrollDown(1), position = %v, want 1", panel.GetScrollPosition())
	}

	// Scroll up
	panel.ScrollUp(1)
	if panel.GetScrollPosition() != 0 {
		t.Errorf("After ScrollUp(1), position = %v, want 0", panel.GetScrollPosition())
	}

	// Can't scroll up past top
	panel.ScrollUp(10)
	if panel.GetScrollPosition() != 0 {
		t.Errorf("After ScrollUp(10) from top, position = %v, want 0", panel.GetScrollPosition())
	}
}

// TestListCreation tests creating a list
func TestListCreation(t *testing.T) {
	list := components.NewList(0, 0, 40, 10)

	if list == nil {
		t.Fatal("NewList returned nil")
	}

	if list.IsMultiSelect() {
		t.Error("Multi-select should be disabled by default")
	}

	if list.IsSearchEnabled() {
		t.Error("Search should be disabled by default")
	}
}

// TestListItems tests adding and selecting items
func TestListItems(t *testing.T) {
	list := components.NewList(0, 0, 40, 10)

	// Add items
	list.AddItem(components.ListItem{Label: "Item 1", Enabled: true})
	list.AddItem(components.ListItem{Label: "Item 2", Enabled: true})
	list.AddItem(components.ListItem{Label: "Item 3", Enabled: true})

	items := list.GetItems()
	if len(items) != 3 {
		t.Errorf("GetItems() length = %v, want 3", len(items))
	}

	// Check selected index
	if list.GetSelectedIndex() != 0 {
		t.Errorf("Initial selected index = %v, want 0", list.GetSelectedIndex())
	}

	// Move down
	list.MoveDown()
	if list.GetSelectedIndex() != 1 {
		t.Errorf("After MoveDown(), selected index = %v, want 1", list.GetSelectedIndex())
	}

	// Move up
	list.MoveUp()
	if list.GetSelectedIndex() != 0 {
		t.Errorf("After MoveUp(), selected index = %v, want 0", list.GetSelectedIndex())
	}
}

// TestListMultiSelect tests multi-select functionality
func TestListMultiSelect(t *testing.T) {
	list := components.NewList(0, 0, 40, 10)
	list.SetMultiSelect(true)

	list.AddItem(components.ListItem{Label: "Item 1", Enabled: true})
	list.AddItem(components.ListItem{Label: "Item 2", Enabled: true})

	// Toggle selection
	list.ToggleSelection()

	selected := list.GetSelectedItems()
	if len(selected) != 1 {
		t.Errorf("GetSelectedItems() length = %v, want 1", len(selected))
	}

	// Move and toggle again
	list.MoveDown()
	list.ToggleSelection()

	selected = list.GetSelectedItems()
	if len(selected) != 2 {
		t.Errorf("GetSelectedItems() length = %v, want 2", len(selected))
	}
}

// TestModalCreation tests creating modals
func TestModalCreation(t *testing.T) {
	modal := components.NewInfoModal("Test", "Test message", func() {
		// Callback on close
	})

	if modal == nil {
		t.Fatal("NewInfoModal returned nil")
	}

	if modal.IsVisible() {
		t.Error("Modal should not be visible by default")
	}

	modal.Show()
	if !modal.IsVisible() {
		t.Error("Modal should be visible after Show()")
	}

	modal.Hide()
	if modal.IsVisible() {
		t.Error("Modal should not be visible after Hide()")
	}
}

// TestModalInput tests input modal
func TestModalInput(t *testing.T) {
	var result string
	var confirmed bool

	modal := components.NewInputModal("Test", "Enter value:", "default", func(ok bool, input string) {
		confirmed = ok
		result = input
	})

	// Set input
	modal.SetInput("test value")
	if modal.GetInput() != "test value" {
		t.Errorf("GetInput() = %v, want %v", modal.GetInput(), "test value")
	}

	// Simulate confirm
	modal.Close(components.ModalResult{Confirmed: true, Input: modal.GetInput()})

	if !confirmed {
		t.Error("Modal should be confirmed")
	}

	if result != "test value" {
		t.Errorf("Result = %v, want %v", result, "test value")
	}
}

// TestStatusBarCreation tests creating a status bar
func TestStatusBarCreation(t *testing.T) {
	statusBar := components.NewStatusBar(24, 80)

	if statusBar == nil {
		t.Fatal("NewStatusBar returned nil")
	}

	y, width := statusBar.GetPosition()
	if y != 24 || width != 80 {
		t.Errorf("GetPosition() = (%v, %v), want (24, 80)", y, width)
	}
}

// TestStatusBarSections tests status bar sections
func TestStatusBarSections(t *testing.T) {
	statusBar := components.NewStatusBar(24, 80)

	statusBar.SetText(components.StatusBarLeft, "Left")
	statusBar.SetText(components.StatusBarCenter, "Center")
	statusBar.SetText(components.StatusBarRight, "Right")

	if statusBar.GetText(components.StatusBarLeft) != "Left" {
		t.Errorf("Left text = %v, want Left", statusBar.GetText(components.StatusBarLeft))
	}

	if statusBar.GetText(components.StatusBarCenter) != "Center" {
		t.Errorf("Center text = %v, want Center", statusBar.GetText(components.StatusBarCenter))
	}

	if statusBar.GetText(components.StatusBarRight) != "Right" {
		t.Errorf("Right text = %v, want Right", statusBar.GetText(components.StatusBarRight))
	}
}

// TestStatusBarMode tests mode indicator
func TestStatusBarMode(t *testing.T) {
	statusBar := components.NewStatusBar(24, 80)

	statusBar.SetMode("NORMAL")
	if statusBar.GetMode() != "NORMAL" {
		t.Errorf("GetMode() = %v, want NORMAL", statusBar.GetMode())
	}
}

// TestStatusBarMessage tests temporary messages
func TestStatusBarMessage(t *testing.T) {
	statusBar := components.NewStatusBar(24, 80)

	statusBar.SetMessage("Test message", 10)
	if statusBar.GetMessage() != "Test message" {
		t.Errorf("GetMessage() = %v, want 'Test message'", statusBar.GetMessage())
	}

	// Update multiple times to decrement timer
	for i := 0; i < 10; i++ {
		statusBar.Update()
	}

	// Message should be cleared after timer expires
	if statusBar.GetMessage() != "" {
		t.Errorf("GetMessage() after timer = %v, want empty", statusBar.GetMessage())
	}
}

// TestComponentsRender tests that all components can render without panic
func TestComponentsRender(t *testing.T) {
	// Create a test screen
	screen := goterm.NewScreen(80, 24)

	// Button
	button := components.NewButton("Test", 10, 5, func() {})
	button.Render(screen)

	// Panel
	panel := components.NewPanel("Test", 0, 0, 40, 10)
	panel.SetContent([]string{"Line 1", "Line 2"})
	panel.Render(screen)

	// List
	list := components.NewList(0, 10, 40, 10)
	list.AddItem(components.ListItem{Label: "Item 1", Enabled: true})
	list.Render(screen)

	// StatusBar
	statusBar := components.NewStatusBar(23, 80)
	statusBar.SetMode("TEST")
	statusBar.Render(screen)

	// Modal (when visible)
	modal := components.NewInfoModal("Test", "Message", func() {})
	modal.Show()
	modal.Render(screen)

	// If we got here without panic, rendering works
	t.Log("All components rendered successfully")
}
