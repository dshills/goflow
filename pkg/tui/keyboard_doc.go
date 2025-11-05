package tui

/*
Package tui provides a comprehensive vim-style keyboard handling system for GoFlow's TUI.

Architecture

The keyboard handler implements a mode-based input system inspired by vim, with four primary modes:
  - Normal Mode: Default navigation and command mode
  - Insert Mode: Text editing and input
  - Visual Mode: Selection and visual operations
  - Command Mode: Colon commands (e.g., :w, :q, :wq)

Key Components

1. KeyEvent: Represents a keyboard input event with support for:
   - Regular character keys
   - Special keys (Enter, Escape, Tab, Arrow keys)
   - Modifier keys (Ctrl, Shift, Alt)

2. KeyboardHandler: Central manager for keyboard input:
   - Mode management
   - Keybinding registry (per-mode and global)
   - Multi-key sequence handling (e.g., 'gg')
   - Boundary checking for navigation
   - Thread-safe operation

3. KeyBinding: Associates a key combination with a handler function:
   - Mode-specific bindings
   - Global bindings (work in all modes)
   - Human-readable labels for help text
   - Conflict detection

Default Keybindings

Normal Mode - Navigation:
  h/j/k/l     - Move cursor left/down/up/right
  w/b         - Move forward/backward by word
  gg/G        - Jump to top/bottom
  Ctrl-u      - Page up
  Ctrl-d      - Page down

Normal Mode - Mode Switching:
  i           - Enter insert mode
  v           - Enter visual mode
  :           - Enter command mode
  Escape      - Return to normal mode (from any mode)

Normal Mode - Operations:
  a           - Add node
  e           - Create edge
  d           - Delete
  r           - Rename
  y           - Copy (yank)
  p           - Paste
  u           - Undo
  Ctrl-r      - Redo

Normal Mode - Search:
  /           - Start search
  n           - Next search result
  N           - Previous search result

Normal Mode - Help & Quit:
  ?           - Toggle help overlay
  q           - Quit

Command Mode:
  :w          - Save workflow
  :q          - Quit
  :wq         - Save and quit
  :q!         - Force quit without saving
  Escape      - Cancel command
  Enter       - Execute command

Insert Mode:
  (chars)     - Insert characters
  Escape      - Return to normal mode

Visual Mode:
  v           - Toggle back to normal mode
  Escape      - Return to normal mode

Usage Example

Basic setup with default bindings:

	kh := NewKeyboardHandler()

	config := DefaultBindingsConfig{
		OnMoveLeft: func() error {
			// Move cursor left
			return nil
		},
		OnMoveRight: func() error {
			// Move cursor right
			return nil
		},
		// ... configure other handlers
	}

	if err := kh.RegisterDefaultBindings(config); err != nil {
		log.Fatal(err)
	}

	// Process keyboard events
	event := KeyEvent{Key: 'h'}
	if err := kh.HandleKey(event); err != nil {
		log.Printf("Error handling key: %v", err)
	}

Custom Keybindings

Register custom bindings for specific modes:

	// Add custom binding in normal mode
	kh.RegisterBinding(ModeNormal, KeyEvent{Key: 's'},
		func(event KeyEvent) error {
			// Custom save action
			return nil
		}, "Custom save")

	// Add global binding (works in all modes)
	kh.RegisterGlobalBinding(KeyEvent{Key: 'F1', IsSpecial: true},
		func(event KeyEvent) error {
			// Show help
			return nil
		}, "Show help")

Multi-Key Sequences

The handler supports multi-key sequences like vim's 'gg':

	// 'gg' is automatically handled by the default bindings
	// The handler tracks the first 'g' and waits for the second
	// When both are received, OnGoToTop is called

Conflict Detection

The system prevents keybinding conflicts within the same mode:

	// This will return an error if 'h' is already bound in normal mode
	err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'h'}, handler, "label")
	if err != nil {
		// Handle conflict
	}

Mode Management

Switch between modes programmatically:

	kh.SetMode(ModeInsert)
	currentMode := kh.GetMode()

Boundary Checking

Set boundaries for navigation operations:

	kh.SetBoundaries(maxX, maxY)  // Set buffer dimensions
	kh.SetPageSize(20)             // Set page size for Ctrl-u/Ctrl-d

Thread Safety

All public methods are thread-safe and can be called from multiple goroutines:

	// Safe to call from different goroutines
	go func() {
		kh.HandleKey(event1)
	}()
	go func() {
		kh.HandleKey(event2)
	}()

Integration with Views

Views can register their own custom bindings:

	// Workflow Builder view adds custom bindings
	builderView := NewWorkflowBuilder()

	kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'c'},
		func(event KeyEvent) error {
			return builderView.CreateNode()
		}, "Create node")

Help System

Generate help text from registered bindings:

	normalBindings := kh.GetBindings(ModeNormal)
	globalBindings := kh.GetGlobalBindings()

	for _, binding := range normalBindings {
		fmt.Printf("%s: %s\n", keyEventToString(binding.Key), binding.Label)
	}

Best Practices

1. Always register default bindings first, then add custom bindings
2. Use mode-specific bindings to avoid conflicts
3. Keep handler functions simple and fast
4. Return errors from handlers for user feedback
5. Use descriptive labels for auto-generated help text
6. Clear pending keys on mode changes (automatic)
7. Use global bindings sparingly (only for truly universal actions)

Performance

The keyboard handler is optimized for low latency:
  - O(1) key lookup using maps
  - Minimal allocations per keystroke
  - RW mutex for concurrent access
  - No blocking operations in critical path

Testing

The tests/tui/keyboard_test.go file contains comprehensive tests covering:
  - All navigation keys (h/j/k/l, w/b, gg/G, Ctrl-u/Ctrl-d)
  - Mode switching (i/v/:/Esc)
  - Operations (a/e/d/r/y/p/u/Ctrl-r)
  - Search (/n/N)
  - Help (?)
  - Quit (q)
  - Command execution (:w/:q/:wq/:q!)
  - Key conflict detection
  - Invalid key handling
  - Boundary checking

See keyboard_test.go for detailed test cases and usage examples.
*/
