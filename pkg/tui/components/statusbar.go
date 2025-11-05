package components

import (
	"strings"

	"github.com/dshills/goterm"
)

// StatusBarSection defines which section of the status bar
type StatusBarSection int

const (
	// StatusBarLeft is the left section
	StatusBarLeft StatusBarSection = iota
	// StatusBarCenter is the center section
	StatusBarCenter
	// StatusBarRight is the right section
	StatusBarRight
)

// StatusBar represents a status bar component at bottom of screen
type StatusBar struct {
	y            int
	width        int
	leftText     string
	centerText   string
	rightText    string
	mode         string
	message      string
	messageTimer int // frames remaining for temporary message
	style        StatusBarStyle
}

// StatusBarStyle defines visual appearance of a status bar
type StatusBarStyle struct {
	Fg        goterm.Color
	Bg        goterm.Color
	ModeFg    goterm.Color
	ModeBg    goterm.Color
	MessageFg goterm.Color
	MessageBg goterm.Color
}

// DefaultStatusBarStyle returns the default status bar style
func DefaultStatusBarStyle() StatusBarStyle {
	return StatusBarStyle{
		Fg:        goterm.ColorRGB(220, 220, 220),
		Bg:        goterm.ColorRGB(40, 40, 40),
		ModeFg:    goterm.ColorRGB(0, 0, 0),
		ModeBg:    goterm.ColorRGB(100, 200, 255),
		MessageFg: goterm.ColorRGB(255, 255, 0),
		MessageBg: goterm.ColorRGB(40, 40, 40),
	}
}

// NewStatusBar creates a new status bar component
// y should typically be screen height - 1
func NewStatusBar(y, width int) *StatusBar {
	return &StatusBar{
		y:            y,
		width:        width,
		leftText:     "",
		centerText:   "",
		rightText:    "",
		mode:         "",
		message:      "",
		messageTimer: 0,
		style:        DefaultStatusBarStyle(),
	}
}

// SetPosition sets the status bar Y position and width
func (s *StatusBar) SetPosition(y, width int) {
	s.y = y
	s.width = width
}

// GetPosition returns the status bar Y position and width
func (s *StatusBar) GetPosition() (int, int) {
	return s.y, s.width
}

// SetText sets text for a specific section
func (s *StatusBar) SetText(section StatusBarSection, text string) {
	switch section {
	case StatusBarLeft:
		s.leftText = text
	case StatusBarCenter:
		s.centerText = text
	case StatusBarRight:
		s.rightText = text
	}
}

// GetText returns text for a specific section
func (s *StatusBar) GetText(section StatusBarSection) string {
	switch section {
	case StatusBarLeft:
		return s.leftText
	case StatusBarCenter:
		return s.centerText
	case StatusBarRight:
		return s.rightText
	}
	return ""
}

// SetMode sets the mode indicator (e.g., "NORMAL", "INSERT", "VISUAL")
func (s *StatusBar) SetMode(mode string) {
	s.mode = mode
}

// GetMode returns the current mode
func (s *StatusBar) GetMode() string {
	return s.mode
}

// SetMessage displays a temporary message
// duration is in render frames (e.g., 60 for 1 second at 60 FPS)
func (s *StatusBar) SetMessage(message string, duration int) {
	s.message = message
	s.messageTimer = duration
}

// ClearMessage clears any displayed message
func (s *StatusBar) ClearMessage() {
	s.message = ""
	s.messageTimer = 0
}

// GetMessage returns the current message
func (s *StatusBar) GetMessage() string {
	return s.message
}

// SetStyle sets the status bar style
func (s *StatusBar) SetStyle(style StatusBarStyle) {
	s.style = style
}

// Update updates the status bar state (call each frame)
func (s *StatusBar) Update() {
	// Decrement message timer
	if s.messageTimer > 0 {
		s.messageTimer--
		if s.messageTimer == 0 {
			s.message = ""
		}
	}
}

// Render renders the status bar to the screen
func (s *StatusBar) Render(screen *goterm.Screen) {
	if screen == nil {
		return
	}

	// Update width if screen size changed
	width, _ := screen.Size()
	if width != s.width {
		s.width = width
	}

	// Clear status bar line
	s.clearLine(screen)

	// Draw mode indicator on left
	x := 0
	if s.mode != "" {
		x = s.drawMode(screen, x)
		x++ // Add space after mode
	}

	// Draw temporary message if active
	if s.message != "" && s.messageTimer > 0 {
		s.drawMessage(screen, x)
		return
	}

	// Draw left text
	if s.leftText != "" {
		x = s.drawText(screen, x, s.leftText, s.style.Fg, s.style.Bg)
		x += 2 // Add spacing
	}

	// Draw right text (from right edge)
	if s.rightText != "" {
		rightX := s.width - len(s.rightText)
		if rightX > x {
			s.drawText(screen, rightX, s.rightText, s.style.Fg, s.style.Bg)
		}
	}

	// Draw center text (centered)
	if s.centerText != "" {
		centerX := (s.width - len(s.centerText)) / 2
		if centerX > x && centerX+len(s.centerText) < s.width-len(s.rightText) {
			s.drawText(screen, centerX, s.centerText, s.style.Fg, s.style.Bg)
		}
	}
}

// clearLine clears the status bar line
func (s *StatusBar) clearLine(screen *goterm.Screen) {
	fg := s.style.Fg
	bg := s.style.Bg

	for x := 0; x < s.width; x++ {
		screen.SetCell(x, s.y, goterm.NewCell(' ', fg, bg, goterm.StyleNone))
	}
}

// drawMode draws the mode indicator
func (s *StatusBar) drawMode(screen *goterm.Screen, x int) int {
	mode := " " + strings.ToUpper(s.mode) + " "
	fg := s.style.ModeFg
	bg := s.style.ModeBg

	for i, ch := range mode {
		if x+i >= s.width {
			break
		}
		screen.SetCell(x+i, s.y, goterm.NewCell(ch, fg, bg, goterm.StyleBold))
	}

	return x + len(mode)
}

// drawMessage draws a temporary message
func (s *StatusBar) drawMessage(screen *goterm.Screen, x int) {
	fg := s.style.MessageFg
	bg := s.style.MessageBg

	maxLen := s.width - x
	message := s.message
	if len(message) > maxLen {
		message = message[:maxLen]
	}

	for i, ch := range message {
		if x+i >= s.width {
			break
		}
		screen.SetCell(x+i, s.y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}
}

// drawText draws text at a specific position
func (s *StatusBar) drawText(screen *goterm.Screen, x int, text string, fg, bg goterm.Color) int {
	maxLen := s.width - x
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	for i, ch := range text {
		if x+i >= s.width {
			break
		}
		screen.SetCell(x+i, s.y, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}

	return x + len(text)
}

// SetWidth sets the status bar width (useful for resize events)
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// GetWidth returns the status bar width
func (s *StatusBar) GetWidth() int {
	return s.width
}

// Height returns the status bar height (always 1)
func (s *StatusBar) Height() int {
	return 1
}
