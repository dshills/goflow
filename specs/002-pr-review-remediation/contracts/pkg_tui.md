// Package contracts defines API contracts for the 002-pr-review-remediation feature.
//
// This file documents contract changes (API fixes) for the tui package.
// Implementation will be in pkg/tui/
package contracts

import (
	"context"
	"io"
)

// Mode represents a TUI input mode (normal, insert, visual, etc.)
//
// FIXED: Type-safe mode identifiers (was mixing Mode type and string "global")
type Mode string

const (
	ModeNormal  Mode = "normal"
	ModeInsert  Mode = "insert"
	ModeVisual  Mode = "visual"
	ModeCommand Mode = "command"
	ModeGlobal  Mode = "global" // NEW: Type-safe global mode (was string "global")
)

// KeyBinding represents a keyboard shortcut bound to an action.
//
// UNCHANGED: Structure remains the same
type KeyBinding struct {
	Key     string
	Mode    Mode
	Handler func() error
}

// KeyBindingRegistry manages keyboard bindings for different modes.
//
// FIXED: Type-safe binding storage and retrieval
type KeyBindingRegistry struct {
	bindings map[Mode]map[string]KeyBinding // FIXED: Mode type for keys
	mu       sync.RWMutex
}

// Register registers a key binding for a specific mode.
//
// Example:
//
//	registry.Register(ModeNormal, "q", func() error {
//	    return app.Quit()
//	})
func (r *KeyBindingRegistry) Register(mode Mode, key string, handler func() error)

// GetAllBindings returns all registered bindings by mode.
//
// FIXED: Returns map[Mode]... instead of mixing types
//
// Global bindings are returned under ModeGlobal key (not string "global").
//
// Example:
//
//	bindings := registry.GetAllBindings()
//	globalBindings := bindings[ModeGlobal]
//	normalBindings := bindings[ModeNormal]
func (r *KeyBindingRegistry) GetAllBindings() map[Mode]map[string]KeyBinding

// App represents the TUI application.
//
// FIXED: Input handling implementation (not visible in contract)
type App struct {
	// ... internal fields ...
}

// NewApp creates a new TUI application.
//
// Example:
//
//	app := NewApp(ctx, config)
func NewApp(ctx context.Context, config *Config) *App

// Run starts the TUI event loop.
//
// FIXED: Internally uses goroutine + blocking read pattern for input
//
// Blocks until app is quit via Quit() or context cancellation.
//
// Returns error if initialization or event loop fails.
func (a *App) Run() error

// Quit signals the app to quit gracefully.
func (a *App) Quit() error

// KeyEvent represents a keyboard input event.
//
// UNCHANGED: Event structure remains the same
type KeyEvent struct {
	Key  rune
	Ctrl bool
	Alt  bool
	// ... other fields ...
}

// INTERNAL: readKeyboardInput is the goroutine that reads keyboard input
//
// FIXED: Removed os.Stdin.SetReadDeadline() (doesn't exist)
// FIXED: Now uses blocking Read() with context cancellation
//
// This is an internal implementation detail not exposed in the public API,
// but documented here for contract clarity.
//
// Implementation pattern:
//
//	func (a *App) readKeyboardInput() {
//	    buf := make([]byte, 32)
//	    for {
//	        select {
//	        case <-a.ctx.Done():
//	            return
//	        default:
//	        }
//	        n, err := os.Stdin.Read(buf) // Blocking read
//	        if err != nil {
//	            if err == io.EOF {
//	                return
//	            }
//	            continue
//	        }
//	        if n > 0 {
//	            event := a.parseKeyInput(buf[:n])
//	            select {
//	            case a.inputChan <- event:
//	            case <-a.ctx.Done():
//	                return
//	            }
//	        }
//	    }
//	}
