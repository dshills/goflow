package components

import (
	"strings"

	"github.com/dshills/goterm"
)

// ListItem represents an item in the list
type ListItem struct {
	Label    string
	Value    interface{}
	Selected bool
	Enabled  bool
}

// List represents a scrollable list component with selection
type List struct {
	x             int
	y             int
	width         int
	height        int
	items         []ListItem
	selectedIndex int
	scrollTop     int
	multiSelect   bool
	searchEnabled bool
	searchQuery   string
	filteredItems []int // indices of filtered items
	style         ListStyle
	focused       bool
}

// ListStyle defines visual appearance of a list
type ListStyle struct {
	ItemFg        goterm.Color
	ItemBg        goterm.Color
	SelectedFg    goterm.Color
	SelectedBg    goterm.Color
	DisabledFg    goterm.Color
	DisabledBg    goterm.Color
	MultiSelectFg goterm.Color
	MultiSelectBg goterm.Color
	SearchFg      goterm.Color
	SearchBg      goterm.Color
}

// DefaultListStyle returns the default list style
func DefaultListStyle() ListStyle {
	return ListStyle{
		ItemFg:        goterm.ColorRGB(220, 220, 220),
		ItemBg:        goterm.ColorDefault(),
		SelectedFg:    goterm.ColorRGB(0, 0, 0),
		SelectedBg:    goterm.ColorRGB(100, 200, 255),
		DisabledFg:    goterm.ColorRGB(100, 100, 100),
		DisabledBg:    goterm.ColorDefault(),
		MultiSelectFg: goterm.ColorRGB(255, 255, 255),
		MultiSelectBg: goterm.ColorRGB(40, 80, 40),
		SearchFg:      goterm.ColorRGB(255, 255, 0),
		SearchBg:      goterm.ColorRGB(40, 40, 40),
	}
}

// NewList creates a new list component
func NewList(x, y, width, height int) *List {
	return &List{
		x:             x,
		y:             y,
		width:         width,
		height:        height,
		items:         []ListItem{},
		selectedIndex: 0,
		scrollTop:     0,
		multiSelect:   false,
		searchEnabled: false,
		searchQuery:   "",
		filteredItems: []int{},
		style:         DefaultListStyle(),
		focused:       false,
	}
}

// SetPosition sets the list position
func (l *List) SetPosition(x, y int) {
	l.x = x
	l.y = y
}

// GetPosition returns the list position
func (l *List) GetPosition() (int, int) {
	return l.x, l.y
}

// SetSize sets the list dimensions
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// GetSize returns the list dimensions
func (l *List) GetSize() (int, int) {
	return l.width, l.height
}

// SetMultiSelect enables or disables multi-select mode
func (l *List) SetMultiSelect(enabled bool) {
	l.multiSelect = enabled
}

// IsMultiSelect returns whether multi-select is enabled
func (l *List) IsMultiSelect() bool {
	return l.multiSelect
}

// SetSearchEnabled enables or disables search/filter
func (l *List) SetSearchEnabled(enabled bool) {
	l.searchEnabled = enabled
	if !enabled {
		l.searchQuery = ""
		l.updateFilter()
	}
}

// IsSearchEnabled returns whether search is enabled
func (l *List) IsSearchEnabled() bool {
	return l.searchEnabled
}

// SetFocused sets the focused state
func (l *List) SetFocused(focused bool) {
	l.focused = focused
}

// IsFocused returns whether the list is focused
func (l *List) IsFocused() bool {
	return l.focused
}

// SetItems sets the list items
func (l *List) SetItems(items []ListItem) {
	l.items = items
	l.selectedIndex = 0
	l.scrollTop = 0
	l.updateFilter()
}

// AddItem adds an item to the list
func (l *List) AddItem(item ListItem) {
	l.items = append(l.items, item)
	l.updateFilter()
}

// RemoveItem removes an item at the given index
func (l *List) RemoveItem(index int) {
	if index >= 0 && index < len(l.items) {
		l.items = append(l.items[:index], l.items[index+1:]...)
		if l.selectedIndex >= len(l.items) && len(l.items) > 0 {
			l.selectedIndex = len(l.items) - 1
		}
		l.updateFilter()
	}
}

// ClearItems clears all items
func (l *List) ClearItems() {
	l.items = []ListItem{}
	l.selectedIndex = 0
	l.scrollTop = 0
	l.updateFilter()
}

// GetItems returns all list items
func (l *List) GetItems() []ListItem {
	return l.items
}

// GetSelectedIndex returns the currently selected index
func (l *List) GetSelectedIndex() int {
	return l.selectedIndex
}

// GetSelectedItem returns the currently selected item
func (l *List) GetSelectedItem() *ListItem {
	if l.selectedIndex >= 0 && l.selectedIndex < len(l.items) {
		return &l.items[l.selectedIndex]
	}
	return nil
}

// GetSelectedItems returns all selected items (for multi-select)
func (l *List) GetSelectedItems() []ListItem {
	var selected []ListItem
	for _, item := range l.items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

// SelectItem sets the selected index
func (l *List) SelectItem(index int) {
	if index >= 0 && index < len(l.items) {
		l.selectedIndex = index
		l.ensureVisible()
	}
}

// ToggleSelection toggles selection for multi-select mode
func (l *List) ToggleSelection() {
	if l.multiSelect && l.selectedIndex >= 0 && l.selectedIndex < len(l.items) {
		l.items[l.selectedIndex].Selected = !l.items[l.selectedIndex].Selected
	}
}

// SetStyle sets the list style
func (l *List) SetStyle(style ListStyle) {
	l.style = style
}

// SetSearchQuery sets the search query and updates filter
func (l *List) SetSearchQuery(query string) {
	l.searchQuery = query
	l.updateFilter()
}

// GetSearchQuery returns the current search query
func (l *List) GetSearchQuery() string {
	return l.searchQuery
}

// updateFilter updates the filtered items based on search query
func (l *List) updateFilter() {
	l.filteredItems = []int{}

	if l.searchQuery == "" {
		// No filter - all items visible
		for i := range l.items {
			l.filteredItems = append(l.filteredItems, i)
		}
	} else {
		// Filter items by search query
		query := strings.ToLower(l.searchQuery)
		for i, item := range l.items {
			if strings.Contains(strings.ToLower(item.Label), query) {
				l.filteredItems = append(l.filteredItems, i)
			}
		}
	}

	// Adjust selected index if needed
	if len(l.filteredItems) > 0 {
		// Find current selection in filtered list
		found := false
		for _, idx := range l.filteredItems {
			if idx == l.selectedIndex {
				found = true
				break
			}
		}
		if !found {
			l.selectedIndex = l.filteredItems[0]
		}
	}
}

// ensureVisible ensures the selected item is visible
func (l *List) ensureVisible() {
	// Find position of selected item in filtered list
	selectedPos := -1
	for pos, idx := range l.filteredItems {
		if idx == l.selectedIndex {
			selectedPos = pos
			break
		}
	}

	if selectedPos == -1 {
		return
	}

	// Adjust scroll to keep selected item visible
	if selectedPos < l.scrollTop {
		l.scrollTop = selectedPos
	} else if selectedPos >= l.scrollTop+l.height {
		l.scrollTop = selectedPos - l.height + 1
	}
}

// MoveUp moves selection up
func (l *List) MoveUp() {
	if len(l.filteredItems) == 0 {
		return
	}

	// Find current position in filtered list
	currentPos := -1
	for pos, idx := range l.filteredItems {
		if idx == l.selectedIndex {
			currentPos = pos
			break
		}
	}

	if currentPos > 0 {
		l.selectedIndex = l.filteredItems[currentPos-1]
		l.ensureVisible()
	}
}

// MoveDown moves selection down
func (l *List) MoveDown() {
	if len(l.filteredItems) == 0 {
		return
	}

	// Find current position in filtered list
	currentPos := -1
	for pos, idx := range l.filteredItems {
		if idx == l.selectedIndex {
			currentPos = pos
			break
		}
	}

	if currentPos >= 0 && currentPos < len(l.filteredItems)-1 {
		l.selectedIndex = l.filteredItems[currentPos+1]
		l.ensureVisible()
	}
}

// MoveToTop moves selection to first item
func (l *List) MoveToTop() {
	if len(l.filteredItems) > 0 {
		l.selectedIndex = l.filteredItems[0]
		l.scrollTop = 0
	}
}

// MoveToBottom moves selection to last item
func (l *List) MoveToBottom() {
	if len(l.filteredItems) > 0 {
		l.selectedIndex = l.filteredItems[len(l.filteredItems)-1]
		l.ensureVisible()
	}
}

// PageUp moves up by page height
func (l *List) PageUp() {
	if len(l.filteredItems) == 0 {
		return
	}

	// Find current position
	currentPos := -1
	for pos, idx := range l.filteredItems {
		if idx == l.selectedIndex {
			currentPos = pos
			break
		}
	}

	if currentPos > 0 {
		newPos := currentPos - l.height
		if newPos < 0 {
			newPos = 0
		}
		l.selectedIndex = l.filteredItems[newPos]
		l.ensureVisible()
	}
}

// PageDown moves down by page height
func (l *List) PageDown() {
	if len(l.filteredItems) == 0 {
		return
	}

	// Find current position
	currentPos := -1
	for pos, idx := range l.filteredItems {
		if idx == l.selectedIndex {
			currentPos = pos
			break
		}
	}

	if currentPos >= 0 {
		newPos := currentPos + l.height
		if newPos >= len(l.filteredItems) {
			newPos = len(l.filteredItems) - 1
		}
		l.selectedIndex = l.filteredItems[newPos]
		l.ensureVisible()
	}
}

// Render renders the list to the screen
func (l *List) Render(screen *goterm.Screen) {
	if screen == nil {
		return
	}

	// Draw search bar if enabled
	searchHeight := 0
	if l.searchEnabled {
		searchHeight = 1
		l.drawSearchBar(screen)
	}

	// Draw list items
	l.drawItems(screen, searchHeight)
}

// drawSearchBar draws the search input bar
func (l *List) drawSearchBar(screen *goterm.Screen) {
	fg := l.style.SearchFg
	bg := l.style.SearchBg

	// Draw search prompt
	prompt := "Search: "
	width, _ := screen.Size()
	for i, ch := range prompt {
		if l.x+i >= width {
			break
		}
		screen.SetCell(l.x+i, l.y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}

	// Draw search query
	queryX := l.x + len(prompt)
	maxQueryLen := l.width - len(prompt) - 2
	query := l.searchQuery
	if len(query) > maxQueryLen {
		query = query[len(query)-maxQueryLen:]
	}

	for i, ch := range query {
		if queryX+i >= l.x+l.width || queryX+i >= width {
			break
		}
		screen.SetCell(queryX+i, l.y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}

	// Draw cursor
	cursorX := queryX + len(query)
	if cursorX < l.x+l.width && cursorX < width {
		screen.SetCell(cursorX, l.y, goterm.NewCell('_', fg, bg, goterm.StyleSlowBlink))
	}

	// Fill rest of line
	for i := cursorX + 1; i < l.x+l.width && i < width; i++ {
		screen.SetCell(i, l.y, goterm.NewCell(' ', fg, bg, goterm.StyleNone))
	}
}

// drawItems draws the list items
func (l *List) drawItems(screen *goterm.Screen, offset int) {
	itemY := l.y + offset
	availableHeight := l.height - offset

	for i := 0; i < availableHeight; i++ {
		displayIdx := l.scrollTop + i
		if displayIdx >= len(l.filteredItems) {
			break
		}

		itemIdx := l.filteredItems[displayIdx]
		item := l.items[itemIdx]
		isSelected := itemIdx == l.selectedIndex

		l.drawItem(screen, itemY+i, item, isSelected)
	}
}

// drawItem draws a single list item
func (l *List) drawItem(screen *goterm.Screen, y int, item ListItem, isSelected bool) {
	// Determine colors
	var fg, bg goterm.Color
	if !item.Enabled {
		fg = l.style.DisabledFg
		bg = l.style.DisabledBg
	} else if isSelected {
		fg = l.style.SelectedFg
		bg = l.style.SelectedBg
	} else if item.Selected && l.multiSelect {
		fg = l.style.MultiSelectFg
		bg = l.style.MultiSelectBg
	} else {
		fg = l.style.ItemFg
		bg = l.style.ItemBg
	}

	// Draw selection indicator
	prefix := " "
	if l.multiSelect {
		if item.Selected {
			prefix = "✓"
		} else {
			prefix = " "
		}
		prefix += " "
	}

	if isSelected {
		prefix = "►" + prefix[1:]
	}

	// Draw item text
	text := prefix + item.Label
	maxLen := l.width
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	// Pad to full width
	text = text + strings.Repeat(" ", maxLen-len(text))

	width, height := screen.Size()
	for i, ch := range text {
		if l.x+i >= width || y >= height {
			break
		}
		screen.SetCell(l.x+i, y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}
}

// HandleKey handles keyboard input for the list
// Returns true if the key was handled
func (l *List) HandleKey(key string) bool {
	// Handle search input
	if l.searchEnabled {
		switch key {
		case "Backspace":
			if len(l.searchQuery) > 0 {
				l.searchQuery = l.searchQuery[:len(l.searchQuery)-1]
				l.updateFilter()
			}
			return true
		case "Esc":
			if l.searchQuery != "" {
				l.searchQuery = ""
				l.updateFilter()
				return true
			}
		default:
			// Regular character input
			if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
				l.searchQuery += key
				l.updateFilter()
				return true
			}
		}
	}

	// Handle navigation
	switch key {
	case "j", "Down":
		l.MoveDown()
		return true
	case "k", "Up":
		l.MoveUp()
		return true
	case "g", "Home":
		l.MoveToTop()
		return true
	case "G", "End":
		l.MoveToBottom()
		return true
	case "d", "PageDown":
		l.PageDown()
		return true
	case "u", "PageUp":
		l.PageUp()
		return true
	case " ":
		if l.multiSelect {
			l.ToggleSelection()
			return true
		}
	case "/":
		if !l.searchEnabled {
			l.SetSearchEnabled(true)
			return true
		}
	}

	return false
}
