package components

import (
	"github.com/dshills/goterm"
)

// Button represents a clickable button component with keyboard focus support
type Button struct {
	label   string
	x       int
	y       int
	width   int
	enabled bool
	focused bool
	onClick func()
	style   ButtonStyle
}

// ButtonStyle defines visual appearance of a button
type ButtonStyle struct {
	NormalFg   goterm.Color
	NormalBg   goterm.Color
	FocusedFg  goterm.Color
	FocusedBg  goterm.Color
	DisabledFg goterm.Color
	DisabledBg goterm.Color
}

// DefaultButtonStyle returns the default button style
func DefaultButtonStyle() ButtonStyle {
	return ButtonStyle{
		NormalFg:   goterm.ColorRGB(255, 255, 255),
		NormalBg:   goterm.ColorRGB(60, 60, 60),
		FocusedFg:  goterm.ColorRGB(0, 0, 0),
		FocusedBg:  goterm.ColorRGB(100, 200, 255),
		DisabledFg: goterm.ColorRGB(128, 128, 128),
		DisabledBg: goterm.ColorRGB(40, 40, 40),
	}
}

// NewButton creates a new button component
func NewButton(label string, x, y int, onClick func()) *Button {
	return &Button{
		label:   label,
		x:       x,
		y:       y,
		width:   len(label) + 4, // Add padding
		enabled: true,
		focused: false,
		onClick: onClick,
		style:   DefaultButtonStyle(),
	}
}

// SetPosition sets the button position
func (b *Button) SetPosition(x, y int) {
	b.x = x
	b.y = y
}

// GetPosition returns the button position
func (b *Button) GetPosition() (int, int) {
	return b.x, b.y
}

// SetEnabled sets the enabled state
func (b *Button) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// IsEnabled returns whether the button is enabled
func (b *Button) IsEnabled() bool {
	return b.enabled
}

// SetFocused sets the focused state
func (b *Button) SetFocused(focused bool) {
	b.focused = focused
}

// IsFocused returns whether the button is focused
func (b *Button) IsFocused() bool {
	return b.focused
}

// SetLabel sets the button label
func (b *Button) SetLabel(label string) {
	b.label = label
	b.width = len(label) + 4
}

// GetLabel returns the button label
func (b *Button) GetLabel() string {
	return b.label
}

// SetStyle sets the button style
func (b *Button) SetStyle(style ButtonStyle) {
	b.style = style
}

// Width returns the button width
func (b *Button) Width() int {
	return b.width
}

// Height returns the button height
func (b *Button) Height() int {
	return 1
}

// Activate triggers the button's onClick callback if enabled
func (b *Button) Activate() {
	if b.enabled && b.onClick != nil {
		b.onClick()
	}
}

// Contains checks if a coordinate is within the button bounds
func (b *Button) Contains(x, y int) bool {
	return x >= b.x && x < b.x+b.width && y == b.y
}

// Render renders the button to the screen
func (b *Button) Render(screen *goterm.Screen) {
	if screen == nil {
		return
	}

	// Determine colors based on state
	var fg, bg goterm.Color
	if !b.enabled {
		fg = b.style.DisabledFg
		bg = b.style.DisabledBg
	} else if b.focused {
		fg = b.style.FocusedFg
		bg = b.style.FocusedBg
	} else {
		fg = b.style.NormalFg
		bg = b.style.NormalBg
	}

	// Draw button background and label
	text := "[ " + b.label + " ]"
	width, height := screen.Size()
	for i, ch := range text {
		if b.x+i >= width || b.y >= height {
			break
		}
		screen.SetCell(b.x+i, b.y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}
}

// HandleKey handles keyboard input for the button
// Returns true if the key was handled
func (b *Button) HandleKey(key string) bool {
	if !b.enabled {
		return false
	}

	// Enter or Space activates the button
	if key == "Enter" || key == " " {
		b.Activate()
		return true
	}

	return false
}
