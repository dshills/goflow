package components

import (
	"strings"

	"github.com/dshills/goterm"
)

// ModalType defines the type of modal dialog
type ModalType int

const (
	// ModalTypeInfo is an informational modal with OK button
	ModalTypeInfo ModalType = iota
	// ModalTypeConfirm is a confirmation modal with OK/Cancel buttons
	ModalTypeConfirm
	// ModalTypeInput is an input modal with text field and OK/Cancel buttons
	ModalTypeInput
)

// ModalResult represents the result of a modal interaction
type ModalResult struct {
	Confirmed bool
	Input     string
}

// Modal represents a modal dialog component
type Modal struct {
	title        string
	message      string
	modalType    ModalType
	width        int
	height       int
	visible      bool
	input        string
	cursorPos    int
	focusedBtn   int // 0 for OK, 1 for Cancel
	okButton     *Button
	cancelButton *Button
	onClose      func(ModalResult)
	style        ModalStyle
}

// ModalStyle defines visual appearance of a modal
type ModalStyle struct {
	TitleFg    goterm.Color
	TitleBg    goterm.Color
	BorderFg   goterm.Color
	BorderBg   goterm.Color
	MessageFg  goterm.Color
	MessageBg  goterm.Color
	BackdropFg goterm.Color
	BackdropBg goterm.Color
	InputFg    goterm.Color
	InputBg    goterm.Color
}

// DefaultModalStyle returns the default modal style
func DefaultModalStyle() ModalStyle {
	return ModalStyle{
		TitleFg:    goterm.ColorRGB(255, 255, 255),
		TitleBg:    goterm.ColorRGB(40, 80, 120),
		BorderFg:   goterm.ColorRGB(150, 150, 200),
		BorderBg:   goterm.ColorDefault(),
		MessageFg:  goterm.ColorRGB(220, 220, 220),
		MessageBg:  goterm.ColorDefault(),
		BackdropFg: goterm.ColorRGB(0, 0, 0),
		BackdropBg: goterm.ColorRGB(0, 0, 0),
		InputFg:    goterm.ColorRGB(255, 255, 255),
		InputBg:    goterm.ColorRGB(30, 30, 30),
	}
}

// NewModal creates a new modal dialog
func NewModal(title, message string, modalType ModalType, onClose func(ModalResult)) *Modal {
	width := 50
	height := 10

	// Adjust height based on type
	if modalType == ModalTypeInput {
		height = 12
	}

	m := &Modal{
		title:      title,
		message:    message,
		modalType:  modalType,
		width:      width,
		height:     height,
		visible:    false,
		input:      "",
		cursorPos:  0,
		focusedBtn: 0,
		onClose:    onClose,
		style:      DefaultModalStyle(),
	}

	// Create buttons based on type
	if modalType == ModalTypeInfo {
		m.okButton = NewButton("OK", 0, 0, func() {
			m.Close(ModalResult{Confirmed: true})
		})
	} else {
		m.okButton = NewButton("OK", 0, 0, func() {
			m.Close(ModalResult{Confirmed: true, Input: m.input})
		})
		m.cancelButton = NewButton("Cancel", 0, 0, func() {
			m.Close(ModalResult{Confirmed: false})
		})
		m.okButton.SetFocused(true)
	}

	return m
}

// NewInfoModal creates an info modal (OK only)
func NewInfoModal(title, message string, onClose func()) *Modal {
	return NewModal(title, message, ModalTypeInfo, func(result ModalResult) {
		if onClose != nil {
			onClose()
		}
	})
}

// NewConfirmModal creates a confirmation modal (OK/Cancel)
func NewConfirmModal(title, message string, onConfirm func(bool)) *Modal {
	return NewModal(title, message, ModalTypeConfirm, func(result ModalResult) {
		if onConfirm != nil {
			onConfirm(result.Confirmed)
		}
	})
}

// NewInputModal creates an input modal (text field + OK/Cancel)
func NewInputModal(title, message, defaultValue string, onSubmit func(bool, string)) *Modal {
	m := NewModal(title, message, ModalTypeInput, func(result ModalResult) {
		if onSubmit != nil {
			onSubmit(result.Confirmed, result.Input)
		}
	})
	m.input = defaultValue
	m.cursorPos = len(defaultValue)
	return m
}

// Show displays the modal
func (m *Modal) Show() {
	m.visible = true
}

// Hide hides the modal
func (m *Modal) Hide() {
	m.visible = false
}

// IsVisible returns whether the modal is visible
func (m *Modal) IsVisible() bool {
	return m.visible
}

// Close closes the modal and triggers callback
func (m *Modal) Close(result ModalResult) {
	m.Hide()
	if m.onClose != nil {
		m.onClose(result)
	}
}

// SetTitle sets the modal title
func (m *Modal) SetTitle(title string) {
	m.title = title
}

// SetMessage sets the modal message
func (m *Modal) SetMessage(message string) {
	m.message = message
}

// SetInput sets the input field value
func (m *Modal) SetInput(input string) {
	m.input = input
	m.cursorPos = len(input)
}

// GetInput returns the current input value
func (m *Modal) GetInput() string {
	return m.input
}

// SetStyle sets the modal style
func (m *Modal) SetStyle(style ModalStyle) {
	m.style = style
}

// Render renders the modal to the screen
func (m *Modal) Render(screen *goterm.Screen) {
	if !m.visible || screen == nil {
		return
	}

	// Calculate centered position
	width, height := screen.Size()
	x := (width - m.width) / 2
	y := (height - m.height) / 2

	// Draw backdrop (dim the background)
	m.drawBackdrop(screen)

	// Draw modal box
	m.drawBorder(screen, x, y)
	m.drawTitle(screen, x, y)
	m.drawMessage(screen, x, y)

	// Draw input field if input modal
	if m.modalType == ModalTypeInput {
		m.drawInputField(screen, x, y)
	}

	// Draw buttons
	m.drawButtons(screen, x, y)
}

// drawBackdrop draws a dimmed backdrop
func (m *Modal) drawBackdrop(screen *goterm.Screen) {
	// Simple backdrop - just dim with semi-transparent effect
	// In practice, you might render a semi-transparent overlay
	width, height := screen.Size()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cell := screen.GetCell(x, y)
			// Dim the existing cell by adjusting colors
			screen.SetCell(x, y, goterm.NewCell(cell.Ch, m.style.BackdropFg, m.style.BackdropBg, goterm.StyleDim))
		}
	}
}

// drawBorder draws the modal border
func (m *Modal) drawBorder(screen *goterm.Screen, x, y int) {
	fg := m.style.BorderFg
	bg := m.style.BorderBg

	// Draw corners
	screen.SetCell(x, y, goterm.NewCell('┌', fg, bg, goterm.StyleNone))
	screen.SetCell(x+m.width-1, y, goterm.NewCell('┐', fg, bg, goterm.StyleNone))
	screen.SetCell(x, y+m.height-1, goterm.NewCell('└', fg, bg, goterm.StyleNone))
	screen.SetCell(x+m.width-1, y+m.height-1, goterm.NewCell('┘', fg, bg, goterm.StyleNone))

	// Draw horizontal borders
	for i := 1; i < m.width-1; i++ {
		screen.SetCell(x+i, y, goterm.NewCell('─', fg, bg, goterm.StyleNone))
		screen.SetCell(x+i, y+m.height-1, goterm.NewCell('─', fg, bg, goterm.StyleNone))
	}

	// Draw vertical borders
	for i := 1; i < m.height-1; i++ {
		screen.SetCell(x, y+i, goterm.NewCell('│', fg, bg, goterm.StyleNone))
		screen.SetCell(x+m.width-1, y+i, goterm.NewCell('│', fg, bg, goterm.StyleNone))
	}

	// Fill background
	for i := 1; i < m.height-1; i++ {
		for j := 1; j < m.width-1; j++ {
			screen.SetCell(x+j, y+i, goterm.NewCell(' ', fg, bg, goterm.StyleNone))
		}
	}
}

// drawTitle draws the modal title
func (m *Modal) drawTitle(screen *goterm.Screen, x, y int) {
	if m.title == "" {
		return
	}

	title := " " + m.title + " "
	titleX := x + 2
	maxLen := m.width - 4

	if len(title) > maxLen {
		title = title[:maxLen]
	}

	fg := m.style.TitleFg
	bg := m.style.TitleBg

	for i, ch := range title {
		screen.SetCell(titleX+i, y, goterm.NewCell(ch, fg, bg, goterm.StyleBold))
	}
}

// drawMessage draws the modal message
func (m *Modal) drawMessage(screen *goterm.Screen, x, y int) {
	contentWidth := m.width - 4
	messageX := x + 2
	messageY := y + 2

	// Word wrap message
	lines := wrapText(m.message, contentWidth)

	fg := m.style.MessageFg
	bg := m.style.MessageBg

	for i, line := range lines {
		if i >= m.height-6 { // Leave room for input and buttons
			break
		}
		for j, ch := range line {
			screen.SetCell(messageX+j, messageY+i, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
		}
	}
}

// drawInputField draws the input field for input modals
func (m *Modal) drawInputField(screen *goterm.Screen, x, y int) {
	inputY := y + m.height - 5
	inputX := x + 2
	inputWidth := m.width - 4

	fg := m.style.InputFg
	bg := m.style.InputBg

	// Draw input field background
	for i := 0; i < inputWidth; i++ {
		screen.SetCell(inputX+i, inputY, goterm.NewCell(' ', fg, bg, goterm.StyleNone))
	}

	// Draw input text
	displayText := m.input
	if len(displayText) > inputWidth-2 {
		// Scroll text if too long
		start := len(displayText) - (inputWidth - 2)
		displayText = displayText[start:]
	}

	for i, ch := range displayText {
		screen.SetCell(inputX+1+i, inputY, goterm.NewCell(ch, fg, bg, goterm.StyleNone))
	}

	// Draw cursor
	cursorX := inputX + 1 + len(displayText)
	if cursorX < inputX+inputWidth-1 {
		screen.SetCell(cursorX, inputY, goterm.NewCell('_', fg, bg, goterm.StyleSlowBlink))
	}
}

// drawButtons draws the modal buttons
func (m *Modal) drawButtons(screen *goterm.Screen, x, y int) {
	buttonY := y + m.height - 3

	if m.modalType == ModalTypeInfo {
		// Center single OK button
		btnX := x + (m.width-m.okButton.Width())/2
		m.okButton.SetPosition(btnX, buttonY)
		m.okButton.SetFocused(true)
		m.okButton.Render(screen)
	} else {
		// Position OK and Cancel buttons
		totalBtnWidth := m.okButton.Width() + m.cancelButton.Width() + 4
		startX := x + (m.width-totalBtnWidth)/2

		m.okButton.SetPosition(startX, buttonY)
		m.cancelButton.SetPosition(startX+m.okButton.Width()+4, buttonY)

		m.okButton.SetFocused(m.focusedBtn == 0)
		m.cancelButton.SetFocused(m.focusedBtn == 1)

		m.okButton.Render(screen)
		m.cancelButton.Render(screen)
	}
}

// HandleKey handles keyboard input for the modal
// Returns true if the key was handled
func (m *Modal) HandleKey(key string) bool {
	if !m.visible {
		return false
	}

	// ESC always cancels
	if key == "Esc" {
		m.Close(ModalResult{Confirmed: false})
		return true
	}

	// Handle input field typing
	if m.modalType == ModalTypeInput {
		switch key {
		case "Backspace":
			if len(m.input) > 0 && m.cursorPos > 0 {
				m.input = m.input[:m.cursorPos-1] + m.input[m.cursorPos:]
				m.cursorPos--
			}
			return true
		case "Delete":
			if m.cursorPos < len(m.input) {
				m.input = m.input[:m.cursorPos] + m.input[m.cursorPos+1:]
			}
			return true
		case "Left":
			if m.cursorPos > 0 {
				m.cursorPos--
			}
			return true
		case "Right":
			if m.cursorPos < len(m.input) {
				m.cursorPos++
			}
			return true
		case "Home":
			m.cursorPos = 0
			return true
		case "End":
			m.cursorPos = len(m.input)
			return true
		default:
			// Regular character input
			if len(key) == 1 {
				m.input = m.input[:m.cursorPos] + key + m.input[m.cursorPos:]
				m.cursorPos++
				return true
			}
		}
	}

	// Handle button navigation and activation
	if m.modalType != ModalTypeInfo {
		switch key {
		case "Tab", "Right", "l":
			m.focusedBtn = (m.focusedBtn + 1) % 2
			return true
		case "Shift+Tab", "Left", "h":
			m.focusedBtn = (m.focusedBtn + 1) % 2
			return true
		}
	}

	// Enter activates focused button
	if key == "Enter" {
		if m.modalType == ModalTypeInfo || m.focusedBtn == 0 {
			m.okButton.Activate()
		} else {
			m.cancelButton.Activate()
		}
		return true
	}

	return false
}

// wrapText wraps text to fit within a given width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
