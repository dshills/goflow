package components

import (
	"strings"

	"github.com/dshills/goterm"
)

// Panel represents a bordered panel with title and scrollable content
type Panel struct {
	title     string
	x         int
	y         int
	width     int
	height    int
	content   []string
	scrollTop int
	border    bool
	style     PanelStyle
	focused   bool
}

// PanelStyle defines visual appearance of a panel
type PanelStyle struct {
	TitleFg   goterm.Color
	TitleBg   goterm.Color
	BorderFg  goterm.Color
	BorderBg  goterm.Color
	ContentFg goterm.Color
	ContentBg goterm.Color
	FocusedFg goterm.Color
	FocusedBg goterm.Color
}

// DefaultPanelStyle returns the default panel style
func DefaultPanelStyle() PanelStyle {
	return PanelStyle{
		TitleFg:   goterm.ColorRGB(255, 255, 255),
		TitleBg:   goterm.ColorRGB(40, 40, 80),
		BorderFg:  goterm.ColorRGB(128, 128, 128),
		BorderBg:  goterm.ColorDefault(),
		ContentFg: goterm.ColorRGB(220, 220, 220),
		ContentBg: goterm.ColorDefault(),
		FocusedFg: goterm.ColorRGB(100, 200, 255),
		FocusedBg: goterm.ColorDefault(),
	}
}

// NewPanel creates a new panel component
func NewPanel(title string, x, y, width, height int) *Panel {
	return &Panel{
		title:     title,
		x:         x,
		y:         y,
		width:     width,
		height:    height,
		content:   []string{},
		scrollTop: 0,
		border:    true,
		style:     DefaultPanelStyle(),
		focused:   false,
	}
}

// SetPosition sets the panel position
func (p *Panel) SetPosition(x, y int) {
	p.x = x
	p.y = y
}

// GetPosition returns the panel position
func (p *Panel) GetPosition() (int, int) {
	return p.x, p.y
}

// SetSize sets the panel dimensions
func (p *Panel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// GetSize returns the panel dimensions
func (p *Panel) GetSize() (int, int) {
	return p.width, p.height
}

// SetTitle sets the panel title
func (p *Panel) SetTitle(title string) {
	p.title = title
}

// GetTitle returns the panel title
func (p *Panel) GetTitle() string {
	return p.title
}

// SetBorder sets whether to draw a border
func (p *Panel) SetBorder(border bool) {
	p.border = border
}

// HasBorder returns whether the panel has a border
func (p *Panel) HasBorder() bool {
	return p.border
}

// SetFocused sets the focused state
func (p *Panel) SetFocused(focused bool) {
	p.focused = focused
}

// IsFocused returns whether the panel is focused
func (p *Panel) IsFocused() bool {
	return p.focused
}

// SetContent sets the panel content lines
func (p *Panel) SetContent(content []string) {
	p.content = content
	// Reset scroll if content changed
	if p.scrollTop >= len(content) {
		p.scrollTop = 0
	}
}

// AppendContent appends a line to the content
func (p *Panel) AppendContent(line string) {
	p.content = append(p.content, line)
}

// ClearContent clears all content
func (p *Panel) ClearContent() {
	p.content = []string{}
	p.scrollTop = 0
}

// GetContent returns the panel content
func (p *Panel) GetContent() []string {
	return p.content
}

// SetStyle sets the panel style
func (p *Panel) SetStyle(style PanelStyle) {
	p.style = style
}

// ContentHeight returns the available height for content
func (p *Panel) ContentHeight() int {
	if p.border {
		return p.height - 2 // Subtract top and bottom border
	}
	return p.height
}

// ContentWidth returns the available width for content
func (p *Panel) ContentWidth() int {
	if p.border {
		return p.width - 2 // Subtract left and right border
	}
	return p.width
}

// ScrollUp scrolls content up by n lines
func (p *Panel) ScrollUp(n int) {
	p.scrollTop -= n
	if p.scrollTop < 0 {
		p.scrollTop = 0
	}
}

// ScrollDown scrolls content down by n lines
func (p *Panel) ScrollDown(n int) {
	maxScroll := len(p.content) - p.ContentHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	p.scrollTop += n
	if p.scrollTop > maxScroll {
		p.scrollTop = maxScroll
	}
}

// ScrollToTop scrolls to the top
func (p *Panel) ScrollToTop() {
	p.scrollTop = 0
}

// ScrollToBottom scrolls to the bottom
func (p *Panel) ScrollToBottom() {
	maxScroll := len(p.content) - p.ContentHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	p.scrollTop = maxScroll
}

// GetScrollPosition returns current scroll position
func (p *Panel) GetScrollPosition() int {
	return p.scrollTop
}

// CanScrollUp returns whether content can scroll up
func (p *Panel) CanScrollUp() bool {
	return p.scrollTop > 0
}

// CanScrollDown returns whether content can scroll down
func (p *Panel) CanScrollDown() bool {
	return p.scrollTop < len(p.content)-p.ContentHeight()
}

// Render renders the panel to the screen
func (p *Panel) Render(screen *goterm.Screen) {
	if screen == nil {
		return
	}

	borderFg := p.style.BorderFg
	if p.focused {
		borderFg = p.style.FocusedFg
	}

	// Draw border if enabled
	if p.border {
		p.drawBorder(screen, borderFg)
		p.drawTitle(screen)
	}

	// Draw content
	p.drawContent(screen)
}

// drawBorder draws the panel border
func (p *Panel) drawBorder(screen *goterm.Screen, fg goterm.Color) {
	bg := p.style.BorderBg

	// Draw corners
	screen.SetCell(p.x, p.y, goterm.NewCell('┌', fg, bg, goterm.StyleNone))
	screen.SetCell(p.x+p.width-1, p.y, goterm.NewCell('┐', fg, bg, goterm.StyleNone))
	screen.SetCell(p.x, p.y+p.height-1, goterm.NewCell('└', fg, bg, goterm.StyleNone))
	screen.SetCell(p.x+p.width-1, p.y+p.height-1, goterm.NewCell('┘', fg, bg, goterm.StyleNone))

	// Draw horizontal borders
	for i := 1; i < p.width-1; i++ {
		screen.SetCell(p.x+i, p.y, goterm.NewCell('─', fg, bg, goterm.StyleNone))
		screen.SetCell(p.x+i, p.y+p.height-1, goterm.NewCell('─', fg, bg, goterm.StyleNone))
	}

	// Draw vertical borders
	for i := 1; i < p.height-1; i++ {
		screen.SetCell(p.x, p.y+i, goterm.NewCell('│', fg, bg, goterm.StyleNone))
		screen.SetCell(p.x+p.width-1, p.y+i, goterm.NewCell('│', fg, bg, goterm.StyleNone))
	}
}

// drawTitle draws the panel title in the top border
func (p *Panel) drawTitle(screen *goterm.Screen) {
	if p.title == "" {
		return
	}

	title := " " + p.title + " "
	titleX := p.x + 2
	maxLen := p.width - 4

	if len(title) > maxLen {
		title = title[:maxLen]
	}

	fg := p.style.TitleFg
	bg := p.style.TitleBg

	for i, ch := range title {
		if titleX+i >= p.x+p.width-1 {
			break
		}
		screen.SetCell(titleX+i, p.y, goterm.NewCell(ch, fg, bg, goterm.StyleBold))
	}
}

// drawContent draws the panel content
func (p *Panel) drawContent(screen *goterm.Screen) {
	contentX := p.x
	contentY := p.y
	contentWidth := p.width
	contentHeight := p.height

	if p.border {
		contentX++
		contentY++
		contentWidth -= 2
		contentHeight -= 2
	}

	fg := p.style.ContentFg
	bg := p.style.ContentBg

	for i := 0; i < contentHeight; i++ {
		lineIdx := p.scrollTop + i
		if lineIdx >= len(p.content) {
			break
		}

		line := p.content[lineIdx]
		// Truncate line if too long
		if len(line) > contentWidth {
			line = line[:contentWidth]
		}

		// Pad line to full width to clear background
		line = line + strings.Repeat(" ", contentWidth-len(line))

		width, height := screen.Size()
		for j, ch := range line {
			if contentX+j >= width || contentY+i >= height {
				break
			}
			screen.SetCell(contentX+j, contentY+i, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
		}
	}
}

// HandleKey handles keyboard input for the panel
// Returns true if the key was handled
func (p *Panel) HandleKey(key string) bool {
	switch key {
	case "j", "Down":
		p.ScrollDown(1)
		return true
	case "k", "Up":
		p.ScrollUp(1)
		return true
	case "d", "PageDown":
		p.ScrollDown(p.ContentHeight())
		return true
	case "u", "PageUp":
		p.ScrollUp(p.ContentHeight())
		return true
	case "g", "Home":
		p.ScrollToTop()
		return true
	case "G", "End":
		p.ScrollToBottom()
		return true
	}
	return false
}
