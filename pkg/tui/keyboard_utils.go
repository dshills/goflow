package tui

import (
	"fmt"
	"strings"
	"unicode"
)

// KeyEventBuilder provides a fluent interface for creating KeyEvents
type KeyEventBuilder struct {
	event KeyEvent
}

// NewKeyEvent creates a new KeyEventBuilder for a regular character key
func NewKeyEvent(key rune) *KeyEventBuilder {
	return &KeyEventBuilder{
		event: KeyEvent{
			Key:       key,
			IsSpecial: false,
		},
	}
}

// NewSpecialKeyEvent creates a new KeyEventBuilder for a special key
func NewSpecialKeyEvent(special string) *KeyEventBuilder {
	return &KeyEventBuilder{
		event: KeyEvent{
			IsSpecial: true,
			Special:   special,
		},
	}
}

// WithCtrl adds the Ctrl modifier
func (b *KeyEventBuilder) WithCtrl() *KeyEventBuilder {
	b.event.Ctrl = true
	return b
}

// WithShift adds the Shift modifier
func (b *KeyEventBuilder) WithShift() *KeyEventBuilder {
	b.event.Shift = true
	return b
}

// WithAlt adds the Alt modifier
func (b *KeyEventBuilder) WithAlt() *KeyEventBuilder {
	b.event.Alt = true
	return b
}

// Build returns the constructed KeyEvent
func (b *KeyEventBuilder) Build() KeyEvent {
	return b.event
}

// KeyEventFromString parses a string representation of a key into a KeyEvent
// Examples: "h", "Ctrl-d", "Shift-G", "Escape", "Enter", "Ctrl-Alt-x"
func KeyEventFromString(s string) (KeyEvent, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return KeyEvent{}, fmt.Errorf("empty key string")
	}

	event := KeyEvent{}
	parts := strings.Split(s, "-")

	// Parse modifiers
	for i := 0; i < len(parts)-1; i++ {
		switch strings.ToLower(parts[i]) {
		case "ctrl", "control":
			event.Ctrl = true
		case "shift":
			event.Shift = true
		case "alt":
			event.Alt = true
		default:
			return KeyEvent{}, fmt.Errorf("unknown modifier: %s", parts[i])
		}
	}

	// Parse the actual key
	keyPart := parts[len(parts)-1]

	// Check if it's a special key
	specialKeys := []string{"Escape", "Enter", "Tab", "Backspace", "Delete",
		"Up", "Down", "Left", "Right", "Home", "End", "PageUp", "PageDown",
		"F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"}

	for _, special := range specialKeys {
		if strings.EqualFold(keyPart, special) {
			event.IsSpecial = true
			event.Special = special
			return event, nil
		}
	}

	// It's a regular character key
	if len(keyPart) != 1 {
		return KeyEvent{}, fmt.Errorf("invalid key: %s", keyPart)
	}

	event.Key = rune(keyPart[0])
	return event, nil
}

// FormatKeyEvent returns a human-readable string representation of a KeyEvent
func FormatKeyEvent(event KeyEvent) string {
	parts := make([]string, 0, 3)

	if event.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if event.Alt {
		parts = append(parts, "Alt")
	}
	if event.Shift && !event.IsSpecial {
		parts = append(parts, "Shift")
	}

	if event.IsSpecial {
		parts = append(parts, event.Special)
	} else {
		key := string(event.Key)
		// Show uppercase letters directly, not as Shift-x
		if event.Shift && event.Key >= 'a' && event.Key <= 'z' {
			key = strings.ToUpper(key)
		}
		parts = append(parts, key)
	}

	return strings.Join(parts, "-")
}

// IsWordBoundary returns true if the character is a word boundary
func IsWordBoundary(ch rune) bool {
	return unicode.IsSpace(ch) || unicode.IsPunct(ch)
}

// FindNextWord finds the next word boundary starting from pos in text
func FindNextWord(text []rune, pos int) int {
	if pos >= len(text) {
		return len(text)
	}

	// Skip current word
	for pos < len(text) && !IsWordBoundary(text[pos]) {
		pos++
	}

	// Skip whitespace/punctuation
	for pos < len(text) && IsWordBoundary(text[pos]) {
		pos++
	}

	return pos
}

// FindPrevWord finds the previous word boundary starting from pos in text
func FindPrevWord(text []rune, pos int) int {
	if pos <= 0 {
		return 0
	}

	pos-- // Move back one position

	// Skip whitespace/punctuation
	for pos > 0 && IsWordBoundary(text[pos]) {
		pos--
	}

	// Find start of current word
	for pos > 0 && !IsWordBoundary(text[pos-1]) {
		pos--
	}

	return pos
}

// ClampPosition clamps a position within boundaries
func ClampPosition(x, y, maxX, maxY int) (int, int) {
	if x < 0 {
		x = 0
	}
	if x > maxX {
		x = maxX
	}
	if y < 0 {
		y = 0
	}
	if y > maxY {
		y = maxY
	}
	return x, y
}

// MovePosition moves a position by dx, dy and clamps it within boundaries
func MovePosition(x, y, dx, dy, maxX, maxY int) (int, int) {
	return ClampPosition(x+dx, y+dy, maxX, maxY)
}

// PagePosition calculates a new position after page up/down
func PagePosition(y, pageSize, maxY int, down bool) int {
	if down {
		y += pageSize
		if y > maxY {
			y = maxY
		}
	} else {
		y -= pageSize
		if y < 0 {
			y = 0
		}
	}
	return y
}

// WrapSearchIndex wraps a search index within bounds (for n/N search navigation)
func WrapSearchIndex(index, delta, length int) int {
	if length == 0 {
		return 0
	}

	index += delta
	if index < 0 {
		index = length - 1
	} else if index >= length {
		index = 0
	}

	return index
}

// CommandParser parses command strings and extracts command and arguments
type CommandParser struct {
	command string
	args    []string
}

// ParseCommand parses a command string (e.g., ":w filename" -> "w", ["filename"])
func ParseCommand(input string) *CommandParser {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, ":")

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return &CommandParser{command: "", args: []string{}}
	}

	return &CommandParser{
		command: parts[0],
		args:    parts[1:],
	}
}

// Command returns the parsed command
func (cp *CommandParser) Command() string {
	return cp.command
}

// Args returns the command arguments
func (cp *CommandParser) Args() []string {
	return cp.args
}

// Arg returns a specific argument by index, or empty string if not present
func (cp *CommandParser) Arg(index int) string {
	if index < 0 || index >= len(cp.args) {
		return ""
	}
	return cp.args[index]
}

// HasArgs returns true if the command has any arguments
func (cp *CommandParser) HasArgs() bool {
	return len(cp.args) > 0
}

// KeySequence represents a multi-key sequence
type KeySequence struct {
	keys     []KeyEvent
	timeout  int // milliseconds
	complete bool
}

// NewKeySequence creates a new key sequence
func NewKeySequence(keys ...KeyEvent) *KeySequence {
	return &KeySequence{
		keys:     keys,
		timeout:  1000, // 1 second default timeout
		complete: false,
	}
}

// Match checks if the given keys match this sequence
func (ks *KeySequence) Match(keys []KeyEvent) bool {
	if len(keys) != len(ks.keys) {
		return false
	}

	for i := range keys {
		if !keyEventEquals(keys[i], ks.keys[i]) {
			return false
		}
	}

	return true
}

// keyEventEquals compares two KeyEvents for equality
func keyEventEquals(a, b KeyEvent) bool {
	if a.IsSpecial != b.IsSpecial {
		return false
	}

	if a.IsSpecial {
		return a.Special == b.Special &&
			a.Ctrl == b.Ctrl &&
			a.Alt == b.Alt &&
			a.Shift == b.Shift
	}

	return a.Key == b.Key &&
		a.Ctrl == b.Ctrl &&
		a.Alt == b.Alt &&
		a.Shift == b.Shift
}

// HelpFormatter formats keybindings for display in help overlay
type HelpFormatter struct {
	maxKeyWidth int
}

// NewHelpFormatter creates a new help formatter
func NewHelpFormatter() *HelpFormatter {
	return &HelpFormatter{
		maxKeyWidth: 15,
	}
}

// FormatBindings formats a list of keybindings into a help text
func (hf *HelpFormatter) FormatBindings(bindings []*KeyBinding) string {
	if len(bindings) == 0 {
		return "No keybindings registered"
	}

	var sb strings.Builder

	// Find max key width for alignment
	maxWidth := 0
	for _, binding := range bindings {
		keyStr := FormatKeyEvent(binding.Key)
		if len(keyStr) > maxWidth && len(keyStr) < hf.maxKeyWidth {
			maxWidth = len(keyStr)
		}
	}

	// Format each binding
	for _, binding := range bindings {
		keyStr := FormatKeyEvent(binding.Key)
		padding := strings.Repeat(" ", maxWidth-len(keyStr)+2)
		sb.WriteString(fmt.Sprintf("%s%s%s\n", keyStr, padding, binding.Label))
	}

	return sb.String()
}

// FormatByMode formats all bindings grouped by mode
func (hf *HelpFormatter) FormatByMode(allBindings map[Mode][]*KeyBinding) string {
	var sb strings.Builder

	modes := []Mode{ModeNormal, ModeInsert, ModeVisual, ModeCommand}

	for _, mode := range modes {
		bindings, ok := allBindings[mode]
		if !ok || len(bindings) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n=== %s Mode ===\n\n", strings.ToUpper(string(mode))))
		sb.WriteString(hf.FormatBindings(bindings))
	}

	// Add global bindings if present
	if globalBindings, ok := allBindings[ModeGlobal]; ok && len(globalBindings) > 0 {
		sb.WriteString("\n=== Global Bindings ===\n\n")
		sb.WriteString(hf.FormatBindings(globalBindings))
	}

	return sb.String()
}
