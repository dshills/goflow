package tui

import (
	"fmt"
	"sync"
)

// Mode represents the current keyboard input mode
type Mode string

const (
	// ModeNormal is the default navigation and command mode
	ModeNormal Mode = "normal"
	// ModeInsert is the text editing mode
	ModeInsert Mode = "insert"
	// ModeVisual is the selection mode
	ModeVisual Mode = "visual"
	// ModeCommand is the command input mode (: commands)
	ModeCommand Mode = "command"
)

// KeyEvent represents a keyboard input event
type KeyEvent struct {
	Key       rune   // The character pressed
	Ctrl      bool   // Ctrl modifier
	Shift     bool   // Shift modifier
	Alt       bool   // Alt modifier
	IsSpecial bool   // Whether this is a special key
	Special   string // Special key name (Enter, Escape, Tab, etc.)
}

// KeyHandler is a function that handles a key event
type KeyHandler func(event KeyEvent) error

// KeyBinding represents a registered keybinding
type KeyBinding struct {
	Key      KeyEvent
	Handler  KeyHandler
	Mode     Mode
	IsGlobal bool   // If true, works in all modes
	Label    string // Description for help text
}

// KeyboardHandler manages vim-style keyboard input
type KeyboardHandler struct {
	mu sync.RWMutex

	// Current mode
	currentMode Mode

	// Keybindings registry organized by mode
	bindings map[Mode]map[string]*KeyBinding

	// Global bindings that work in any mode
	globalBindings map[string]*KeyBinding

	// Pending key for multi-key sequences (e.g., 'gg')
	pendingKey rune

	// Buffer dimensions for boundary checks
	maxX int
	maxY int

	// Page size for Ctrl-u/Ctrl-d
	pageSize int
}

// NewKeyboardHandler creates a new keyboard handler with default vim bindings
func NewKeyboardHandler() *KeyboardHandler {
	kh := &KeyboardHandler{
		currentMode:    ModeNormal,
		bindings:       make(map[Mode]map[string]*KeyBinding),
		globalBindings: make(map[string]*KeyBinding),
		pendingKey:     0,
		pageSize:       20, // Default page size
	}

	// Initialize mode maps
	for _, mode := range []Mode{ModeNormal, ModeInsert, ModeVisual, ModeCommand} {
		kh.bindings[mode] = make(map[string]*KeyBinding)
	}

	return kh
}

// SetMode changes the current input mode
func (kh *KeyboardHandler) SetMode(mode Mode) {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.currentMode = mode
	kh.pendingKey = 0 // Clear pending keys on mode change
}

// GetMode returns the current input mode
func (kh *KeyboardHandler) GetMode() Mode {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.currentMode
}

// SetBoundaries sets the buffer boundaries for navigation
func (kh *KeyboardHandler) SetBoundaries(maxX, maxY int) {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.maxX = maxX
	kh.maxY = maxY
}

// SetPageSize sets the page size for Ctrl-u/Ctrl-d navigation
func (kh *KeyboardHandler) SetPageSize(size int) {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.pageSize = size
}

// RegisterBinding registers a new keybinding for a specific mode
func (kh *KeyboardHandler) RegisterBinding(mode Mode, key KeyEvent, handler KeyHandler, label string) error {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	keyStr := keyEventToString(key)

	// Check for conflicts in the same mode
	if _, exists := kh.bindings[mode][keyStr]; exists {
		return fmt.Errorf("keybinding conflict: %s already registered in %s mode", keyStr, mode)
	}

	kh.bindings[mode][keyStr] = &KeyBinding{
		Key:      key,
		Handler:  handler,
		Mode:     mode,
		IsGlobal: false,
		Label:    label,
	}

	return nil
}

// RegisterGlobalBinding registers a keybinding that works in all modes
func (kh *KeyboardHandler) RegisterGlobalBinding(key KeyEvent, handler KeyHandler, label string) error {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	keyStr := keyEventToString(key)

	// Check for conflicts
	if _, exists := kh.globalBindings[keyStr]; exists {
		return fmt.Errorf("global keybinding conflict: %s already registered", keyStr)
	}

	kh.globalBindings[keyStr] = &KeyBinding{
		Key:      key,
		Handler:  handler,
		IsGlobal: true,
		Label:    label,
	}

	return nil
}

// UnregisterBinding removes a keybinding from a specific mode
func (kh *KeyboardHandler) UnregisterBinding(mode Mode, key KeyEvent) {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	keyStr := keyEventToString(key)
	delete(kh.bindings[mode], keyStr)
}

// UnregisterGlobalBinding removes a global keybinding
func (kh *KeyboardHandler) UnregisterGlobalBinding(key KeyEvent) {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	keyStr := keyEventToString(key)
	delete(kh.globalBindings, keyStr)
}

// HandleKey processes a key event and dispatches to the appropriate handler
func (kh *KeyboardHandler) HandleKey(event KeyEvent) error {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	keyStr := keyEventToString(event)

	// Check for global bindings first
	if binding, exists := kh.globalBindings[keyStr]; exists {
		return binding.Handler(event)
	}

	// Check for mode-specific bindings
	if binding, exists := kh.bindings[kh.currentMode][keyStr]; exists {
		return binding.Handler(event)
	}

	// Handle multi-key sequences (e.g., gg)
	if kh.pendingKey != 0 {
		sequence := string([]rune{kh.pendingKey, event.Key})
		kh.pendingKey = 0 // Clear pending key

		// Check for sequence bindings
		seqKey := KeyEvent{Key: rune(sequence[0]), Shift: event.Shift}
		seqKeyStr := keyEventToString(seqKey) + string(sequence[1])

		if binding, exists := kh.bindings[kh.currentMode][seqKeyStr]; exists {
			return binding.Handler(event)
		}
	}

	// Check if this starts a multi-key sequence
	if event.Key == 'g' && kh.currentMode == ModeNormal {
		kh.pendingKey = 'g'
		return nil // Wait for next key
	}

	// No binding found - in insert/command mode, this might be input
	if kh.currentMode == ModeInsert || kh.currentMode == ModeCommand {
		return nil // Allow character insertion
	}

	// In normal/visual mode, unbound keys are ignored
	return nil
}

// GetBindings returns all bindings for a specific mode
func (kh *KeyboardHandler) GetBindings(mode Mode) []*KeyBinding {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	bindings := make([]*KeyBinding, 0, len(kh.bindings[mode]))
	for _, binding := range kh.bindings[mode] {
		bindings = append(bindings, binding)
	}

	return bindings
}

// GetGlobalBindings returns all global bindings
func (kh *KeyboardHandler) GetGlobalBindings() []*KeyBinding {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	bindings := make([]*KeyBinding, 0, len(kh.globalBindings))
	for _, binding := range kh.globalBindings {
		bindings = append(bindings, binding)
	}

	return bindings
}

// GetAllBindings returns all bindings for all modes
func (kh *KeyboardHandler) GetAllBindings() map[Mode][]*KeyBinding {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	result := make(map[Mode][]*KeyBinding)

	// Add mode-specific bindings
	for mode, modeBindings := range kh.bindings {
		bindings := make([]*KeyBinding, 0, len(modeBindings))
		for _, binding := range modeBindings {
			bindings = append(bindings, binding)
		}
		result[mode] = bindings
	}

	// Add global bindings
	globalBindings := make([]*KeyBinding, 0, len(kh.globalBindings))
	for _, binding := range kh.globalBindings {
		globalBindings = append(globalBindings, binding)
	}
	result["global"] = globalBindings

	return result
}

// ClearPendingKeys clears any pending multi-key sequences
func (kh *KeyboardHandler) ClearPendingKeys() {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	kh.pendingKey = 0
}

// HasPendingKey returns true if there's a pending key in a multi-key sequence
func (kh *KeyboardHandler) HasPendingKey() bool {
	kh.mu.RLock()
	defer kh.mu.RUnlock()

	return kh.pendingKey != 0
}

// keyEventToString converts a KeyEvent to a string for lookup
func keyEventToString(event KeyEvent) string {
	if event.IsSpecial {
		base := event.Special
		if event.Ctrl {
			base = "Ctrl-" + base
		}
		if event.Alt {
			base = "Alt-" + base
		}
		if event.Shift {
			base = "Shift-" + base
		}
		return base
	}

	key := string(event.Key)
	if event.Ctrl {
		key = fmt.Sprintf("Ctrl-%c", event.Key)
	}
	if event.Alt {
		key = fmt.Sprintf("Alt-%c", event.Key)
	}
	if event.Shift && event.Key >= 'a' && event.Key <= 'z' {
		// Shift+letter is represented as uppercase
		key = string(event.Key - 32)
	}

	return key
}

// Helper functions for common key checks

// IsNavigationKey returns true if the key is a navigation key (h/j/k/l)
func IsNavigationKey(event KeyEvent) bool {
	if event.IsSpecial || event.Ctrl || event.Alt {
		return false
	}
	return event.Key == 'h' || event.Key == 'j' || event.Key == 'k' || event.Key == 'l'
}

// IsModeSwitchKey returns true if the key switches modes (i/v/:/Esc)
func IsModeSwitchKey(event KeyEvent) bool {
	if event.IsSpecial && event.Special == "Escape" {
		return true
	}
	if event.IsSpecial {
		return false
	}
	return event.Key == 'i' || event.Key == 'v' || event.Key == ':'
}

// IsOperationKey returns true if the key performs an operation (a/e/d/r/y/p/u)
func IsOperationKey(event KeyEvent) bool {
	if event.IsSpecial {
		return false
	}
	operations := "aedrnyp"
	for _, op := range operations {
		if event.Key == op {
			return true
		}
	}
	return false
}
