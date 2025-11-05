package tui

import (
	"fmt"
	"strings"
)

// DefaultBindingsConfig contains callbacks for default vim-style operations
type DefaultBindingsConfig struct {
	// Navigation callbacks
	OnMoveLeft     func() error
	OnMoveRight    func() error
	OnMoveUp       func() error
	OnMoveDown     func() error
	OnWordForward  func() error
	OnWordBackward func() error
	OnGoToTop      func() error
	OnGoToBottom   func() error
	OnPageUp       func() error
	OnPageDown     func() error

	// Mode switching callbacks
	OnEnterInsertMode  func() error
	OnEnterVisualMode  func() error
	OnEnterCommandMode func() error
	OnEnterNormalMode  func() error

	// Operation callbacks
	OnAddNode    func() error
	OnCreateEdge func() error
	OnDelete     func() error
	OnRename     func() error
	OnCopy       func() error
	OnPaste      func() error
	OnUndo       func() error
	OnRedo       func() error
	OnSearch     func() error
	OnNextSearch func() error
	OnPrevSearch func() error
	OnToggleHelp func() error
	OnQuit       func() error

	// Command execution
	OnExecuteCommand func(command string) error

	// Insert mode text input
	OnInsertChar func(ch rune) error
	OnBackspace  func() error

	// Command mode text input
	OnCommandChar      func(ch rune) error
	OnCommandBackspace func() error
}

// RegisterDefaultBindings configures all default vim-style keybindings
func (kh *KeyboardHandler) RegisterDefaultBindings(config DefaultBindingsConfig) error {
	// Normal Mode - Navigation (h/j/k/l)
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'h'},
		wrapHandler(config.OnMoveLeft), "Move cursor left"); err != nil {
		return fmt.Errorf("register h: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'j'},
		wrapHandler(config.OnMoveDown), "Move cursor down"); err != nil {
		return fmt.Errorf("register j: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'k'},
		wrapHandler(config.OnMoveUp), "Move cursor up"); err != nil {
		return fmt.Errorf("register k: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'l'},
		wrapHandler(config.OnMoveRight), "Move cursor right"); err != nil {
		return fmt.Errorf("register l: %w", err)
	}

	// Normal Mode - Word movement
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'w'},
		wrapHandler(config.OnWordForward), "Move to next word"); err != nil {
		return fmt.Errorf("register w: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'b'},
		wrapHandler(config.OnWordBackward), "Move to previous word"); err != nil {
		return fmt.Errorf("register b: %w", err)
	}

	// Normal Mode - Jump to top/bottom
	// Note: 'gg' is handled as a special sequence in HandleKey
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'g'},
		func(event KeyEvent) error {
			// This is handled by the multi-key sequence logic
			return nil
		}, "Start of 'gg' sequence"); err != nil {
		return fmt.Errorf("register g: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'G', Shift: true},
		wrapHandler(config.OnGoToBottom), "Go to bottom"); err != nil {
		return fmt.Errorf("register G: %w", err)
	}

	// Register 'gg' sequence (handled specially)
	if err := kh.registerSequence(ModeNormal, "gg",
		wrapHandler(config.OnGoToTop), "Go to top"); err != nil {
		return fmt.Errorf("register gg: %w", err)
	}

	// Normal Mode - Page navigation
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'u', Ctrl: true},
		wrapHandler(config.OnPageUp), "Page up"); err != nil {
		return fmt.Errorf("register Ctrl-u: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'd', Ctrl: true},
		wrapHandler(config.OnPageDown), "Page down"); err != nil {
		return fmt.Errorf("register Ctrl-d: %w", err)
	}

	// Normal Mode - Mode switching
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'i'},
		func(event KeyEvent) error {
			kh.SetMode(ModeInsert)
			if config.OnEnterInsertMode != nil {
				return config.OnEnterInsertMode()
			}
			return nil
		}, "Enter insert mode"); err != nil {
		return fmt.Errorf("register i: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'v'},
		func(event KeyEvent) error {
			kh.SetMode(ModeVisual)
			if config.OnEnterVisualMode != nil {
				return config.OnEnterVisualMode()
			}
			return nil
		}, "Enter visual mode"); err != nil {
		return fmt.Errorf("register v: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: ':'},
		func(event KeyEvent) error {
			kh.SetMode(ModeCommand)
			if config.OnEnterCommandMode != nil {
				return config.OnEnterCommandMode()
			}
			return nil
		}, "Enter command mode"); err != nil {
		return fmt.Errorf("register :: %w", err)
	}

	// Normal Mode - Operations
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'a'},
		wrapHandler(config.OnAddNode), "Add node"); err != nil {
		return fmt.Errorf("register a: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'e'},
		wrapHandler(config.OnCreateEdge), "Create edge"); err != nil {
		return fmt.Errorf("register e: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'd'},
		wrapHandler(config.OnDelete), "Delete"); err != nil {
		return fmt.Errorf("register d: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'r'},
		wrapHandler(config.OnRename), "Rename"); err != nil {
		return fmt.Errorf("register r: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'y'},
		wrapHandler(config.OnCopy), "Copy (yank)"); err != nil {
		return fmt.Errorf("register y: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'p'},
		wrapHandler(config.OnPaste), "Paste"); err != nil {
		return fmt.Errorf("register p: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'u'},
		wrapHandler(config.OnUndo), "Undo"); err != nil {
		return fmt.Errorf("register u: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'r', Ctrl: true},
		wrapHandler(config.OnRedo), "Redo"); err != nil {
		return fmt.Errorf("register Ctrl-r: %w", err)
	}

	// Normal Mode - Search
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: '/'},
		func(event KeyEvent) error {
			kh.SetMode(ModeCommand)
			if config.OnSearch != nil {
				return config.OnSearch()
			}
			return nil
		}, "Search"); err != nil {
		return fmt.Errorf("register /: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'n'},
		wrapHandler(config.OnNextSearch), "Next search result"); err != nil {
		return fmt.Errorf("register n: %w", err)
	}

	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'N', Shift: true},
		wrapHandler(config.OnPrevSearch), "Previous search result"); err != nil {
		return fmt.Errorf("register N: %w", err)
	}

	// Normal Mode - Help
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: '?', Shift: true},
		wrapHandler(config.OnToggleHelp), "Toggle help"); err != nil {
		return fmt.Errorf("register ?: %w", err)
	}

	// Normal Mode - Quit
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'q'},
		wrapHandler(config.OnQuit), "Quit"); err != nil {
		return fmt.Errorf("register q: %w", err)
	}

	// Insert Mode - Escape to normal
	if err := kh.RegisterBinding(ModeInsert, KeyEvent{IsSpecial: true, Special: "Escape"},
		func(event KeyEvent) error {
			kh.SetMode(ModeNormal)
			if config.OnEnterNormalMode != nil {
				return config.OnEnterNormalMode()
			}
			return nil
		}, "Exit to normal mode"); err != nil {
		return fmt.Errorf("register Escape in insert: %w", err)
	}

	// Visual Mode - Toggle back to normal with 'v'
	if err := kh.RegisterBinding(ModeVisual, KeyEvent{Key: 'v'},
		func(event KeyEvent) error {
			kh.SetMode(ModeNormal)
			if config.OnEnterNormalMode != nil {
				return config.OnEnterNormalMode()
			}
			return nil
		}, "Exit visual mode"); err != nil {
		return fmt.Errorf("register v in visual: %w", err)
	}

	// Visual Mode - Escape to normal
	if err := kh.RegisterBinding(ModeVisual, KeyEvent{IsSpecial: true, Special: "Escape"},
		func(event KeyEvent) error {
			kh.SetMode(ModeNormal)
			if config.OnEnterNormalMode != nil {
				return config.OnEnterNormalMode()
			}
			return nil
		}, "Exit to normal mode"); err != nil {
		return fmt.Errorf("register Escape in visual: %w", err)
	}

	// Command Mode - Escape to normal
	if err := kh.RegisterBinding(ModeCommand, KeyEvent{IsSpecial: true, Special: "Escape"},
		func(event KeyEvent) error {
			kh.SetMode(ModeNormal)
			if config.OnEnterNormalMode != nil {
				return config.OnEnterNormalMode()
			}
			return nil
		}, "Cancel command"); err != nil {
		return fmt.Errorf("register Escape in command: %w", err)
	}

	// Command Mode - Enter to execute
	if err := kh.RegisterBinding(ModeCommand, KeyEvent{IsSpecial: true, Special: "Enter"},
		func(event KeyEvent) error {
			kh.SetMode(ModeNormal)
			// Command execution handled separately
			if config.OnEnterNormalMode != nil {
				return config.OnEnterNormalMode()
			}
			return nil
		}, "Execute command"); err != nil {
		return fmt.Errorf("register Enter in command: %w", err)
	}

	// Global - Escape always returns to normal (if help is open, close it first)
	if err := kh.RegisterBinding(ModeNormal, KeyEvent{IsSpecial: true, Special: "Escape"},
		func(event KeyEvent) error {
			// In normal mode, Escape might close help overlay
			if config.OnToggleHelp != nil {
				return config.OnToggleHelp()
			}
			return nil
		}, "Close overlays"); err != nil {
		return fmt.Errorf("register Escape in normal: %w", err)
	}

	return nil
}

// registerSequence registers a multi-key sequence binding
func (kh *KeyboardHandler) registerSequence(mode Mode, sequence string, handler KeyHandler, label string) error {
	if len(sequence) != 2 {
		return fmt.Errorf("only 2-key sequences supported, got: %s", sequence)
	}

	// Store the sequence with a special key format
	key := KeyEvent{Key: rune(sequence[0])}
	keyStr := keyEventToString(key) + string(sequence[1])

	kh.bindings[mode][keyStr] = &KeyBinding{
		Key:      key,
		Handler:  handler,
		Mode:     mode,
		IsGlobal: false,
		Label:    label,
	}

	return nil
}

// wrapHandler wraps a callback function to handle nil callbacks gracefully
func wrapHandler(handler func() error) KeyHandler {
	return func(event KeyEvent) error {
		if handler != nil {
			return handler()
		}
		return nil
	}
}

// ExecuteCommand processes a command string (e.g., "w", "q", "wq")
func ExecuteCommand(command string, config DefaultBindingsConfig) error {
	cmd := strings.TrimSpace(command)

	switch cmd {
	case "w":
		// Save workflow
		if config.OnExecuteCommand != nil {
			return config.OnExecuteCommand("save")
		}
		return nil

	case "q":
		// Quit
		if config.OnQuit != nil {
			return config.OnQuit()
		}
		return nil

	case "wq":
		// Save and quit
		if config.OnExecuteCommand != nil {
			if err := config.OnExecuteCommand("save"); err != nil {
				return err
			}
		}
		if config.OnQuit != nil {
			return config.OnQuit()
		}
		return nil

	case "q!":
		// Force quit without saving
		if config.OnExecuteCommand != nil {
			return config.OnExecuteCommand("force_quit")
		}
		return nil

	default:
		// Unknown command
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
