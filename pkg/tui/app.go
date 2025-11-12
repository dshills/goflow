package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dshills/goterm"
)

// App represents the TUI application root
type App struct {
	screen        *goterm.Screen
	viewManager   *ViewManager
	keyboard      *KeyboardHandler
	running       bool
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	inputChan     chan KeyEvent
	lastFrameTime time.Time
}

// NewApp creates a new TUI application instance
func NewApp() (*App, error) {
	// Initialize terminal screen
	screen, err := goterm.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize terminal: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create view manager
	viewManager := NewViewManager()

	// Create keyboard handler
	keyboard := NewKeyboardHandler()

	app := &App{
		screen:        screen,
		viewManager:   viewManager,
		keyboard:      keyboard,
		running:       false,
		ctx:           ctx,
		cancel:        cancel,
		inputChan:     make(chan KeyEvent, 100),
		lastFrameTime: time.Now(),
	}

	// Register default views
	if err := app.registerViews(); err != nil {
		screen.Close()
		return nil, fmt.Errorf("failed to register views: %w", err)
	}

	// Register default keybindings
	if err := app.registerGlobalKeybindings(); err != nil {
		screen.Close()
		return nil, fmt.Errorf("failed to register keybindings: %w", err)
	}

	// Initialize view manager with workflow explorer
	if err := viewManager.Initialize("explorer"); err != nil {
		screen.Close()
		return nil, fmt.Errorf("failed to initialize view manager: %w", err)
	}

	return app, nil
}

// registerViews registers all available views
func (a *App) registerViews() error {
	// Register workflow explorer view
	explorerView := NewWorkflowExplorerView()
	if err := a.viewManager.RegisterView(explorerView); err != nil {
		return fmt.Errorf("failed to register explorer view: %w", err)
	}

	// Register workflow builder view
	builderView := NewWorkflowBuilderView()
	if err := a.viewManager.RegisterView(builderView); err != nil {
		return fmt.Errorf("failed to register builder view: %w", err)
	}

	// Register execution monitor view
	monitorView := NewExecutionMonitorView()
	if err := a.viewManager.RegisterView(monitorView); err != nil {
		return fmt.Errorf("failed to register monitor view: %w", err)
	}

	// Register server registry view
	registryView := NewServerRegistryView()
	if err := a.viewManager.RegisterView(registryView); err != nil {
		return fmt.Errorf("failed to register registry view: %w", err)
	}

	return nil
}

// registerGlobalKeybindings registers application-wide keybindings
func (a *App) registerGlobalKeybindings() error {
	// Ctrl+C: Quit application
	if err := a.keyboard.RegisterGlobalBinding(
		KeyEvent{Key: 'c', Ctrl: true},
		func(event KeyEvent) error {
			a.cancel()
			return nil
		},
		"Quit application",
	); err != nil {
		return err
	}

	// q: Quit application (in normal mode only)
	if err := a.keyboard.RegisterBinding(
		ModeNormal,
		KeyEvent{Key: 'q'},
		func(event KeyEvent) error {
			a.cancel()
			return nil
		},
		"Quit application",
	); err != nil {
		return err
	}

	// Tab: Switch to next view
	if err := a.keyboard.RegisterGlobalBinding(
		KeyEvent{Key: '\t', IsSpecial: true, Special: "Tab"},
		func(event KeyEvent) error {
			return a.viewManager.NextView()
		},
		"Switch to next view",
	); err != nil {
		return err
	}

	// ?: Show help (will be implemented by views)
	if err := a.keyboard.RegisterBinding(
		ModeNormal,
		KeyEvent{Key: '?'},
		func(event KeyEvent) error {
			// TODO: Show help overlay
			return nil
		},
		"Show help",
	); err != nil {
		return err
	}

	return nil
}

// Run starts the TUI application main loop
func (a *App) Run() error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start keyboard input goroutine
	go a.readKeyboardInput()

	// Render loop targeting 60 FPS (16ms frame time)
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	// Initial render
	if err := a.render(); err != nil {
		return fmt.Errorf("initial render failed: %w", err)
	}

	for {
		select {
		case <-a.ctx.Done():
			return nil

		case <-sigChan:
			a.cancel()
			return nil

		case event := <-a.inputChan:
			if err := a.handleKeyEvent(event); err != nil {
				return err
			}
			// Render immediately after input
			if err := a.render(); err != nil {
				return err
			}

		case <-ticker.C:
			// Regular frame update
			if err := a.render(); err != nil {
				return err
			}
		}
	}
}

// handleKeyEvent processes keyboard input through the keyboard handler
func (a *App) handleKeyEvent(event KeyEvent) error {
	// First, let the keyboard handler process global bindings
	if err := a.keyboard.HandleKey(event); err != nil {
		return fmt.Errorf("keyboard handler error: %w", err)
	}

	// Then pass to the current view
	currentView := a.viewManager.GetCurrentView()
	if currentView != nil {
		if err := currentView.HandleKey(event); err != nil {
			return fmt.Errorf("view key handler error: %w", err)
		}
	}

	return nil
}

// render draws the current view to the screen
func (a *App) render() error {
	start := time.Now()

	// Get current view
	currentView := a.viewManager.GetCurrentView()

	// Clear screen buffer
	a.screen.Clear()

	// Render current view
	if currentView != nil {
		if err := currentView.Render(a.screen); err != nil {
			return fmt.Errorf("view render failed: %w", err)
		}
	}

	// Show the screen
	if err := a.screen.Show(); err != nil {
		return fmt.Errorf("screen show failed: %w", err)
	}

	// Track frame time for performance monitoring
	frameTime := time.Since(start)
	a.lastFrameTime = start

	// Log warning if frame time exceeds target (16ms for 60 FPS)
	if frameTime > 16*time.Millisecond {
		// In production, this would use proper logging
		// Constitutional requirement: < 16ms frame time
		_ = frameTime
	}

	return nil
}

// readKeyboardInput reads keyboard input in a background goroutine
func (a *App) readKeyboardInput() {
	// Read from stdin in raw mode (blocking)
	buf := make([]byte, 32)

	for {
		// Check for context cancellation before each read
		select {
		case <-a.ctx.Done():
			return
		default:
		}

		// Blocking read - terminal is already in raw mode from goterm
		n, err := os.Stdin.Read(buf)
		if err != nil {
			// Handle EOF gracefully (stdin closed)
			if err == io.EOF {
				return
			}
			// On other errors, continue to next iteration
			continue
		}

		if n > 0 {
			// Parse input and send to input channel
			event := a.parseKeyInput(buf[:n])
			select {
			case a.inputChan <- event:
			case <-a.ctx.Done():
				return
			}
		}
	}
}

// parseKeyInput converts raw bytes into a KeyEvent
func (a *App) parseKeyInput(buf []byte) KeyEvent {
	if len(buf) == 0 {
		return KeyEvent{}
	}

	// Handle escape sequences (arrow keys, etc.)
	if buf[0] == 27 {
		if len(buf) == 1 {
			return KeyEvent{IsSpecial: true, Special: "Escape"}
		}
		if len(buf) > 1 && buf[1] == '[' && len(buf) > 2 {
			switch buf[2] {
			case 'A':
				return KeyEvent{IsSpecial: true, Special: "Up"}
			case 'B':
				return KeyEvent{IsSpecial: true, Special: "Down"}
			case 'C':
				return KeyEvent{IsSpecial: true, Special: "Right"}
			case 'D':
				return KeyEvent{IsSpecial: true, Special: "Left"}
			}
		}
		return KeyEvent{IsSpecial: true, Special: "Escape"}
	}

	// Handle special keys
	switch buf[0] {
	case 9: // Tab
		return KeyEvent{IsSpecial: true, Special: "Tab"}
	case 13: // Enter
		return KeyEvent{IsSpecial: true, Special: "Enter"}
	case 127: // Backspace
		return KeyEvent{IsSpecial: true, Special: "Backspace"}
	}

	// Handle Ctrl combinations
	if buf[0] < 32 {
		return KeyEvent{
			Key:  rune(buf[0] + 'a' - 1), // Convert to letter
			Ctrl: true,
		}
	}

	// Regular character
	key := rune(buf[0])
	shift := false
	if key >= 'A' && key <= 'Z' {
		shift = true
	}

	return KeyEvent{
		Key:   key,
		Shift: shift,
	}
}

// Close performs cleanup and restores terminal state
func (a *App) Close() error {
	a.cancel()

	// Shutdown view manager (cleans up all views)
	if err := a.viewManager.Shutdown(); err != nil {
		// Log error but continue cleanup
		_ = err
	}

	// Close screen (restores terminal)
	if err := a.screen.Close(); err != nil {
		return fmt.Errorf("failed to close screen: %w", err)
	}

	return nil
}

// GetViewManager returns the view manager instance
func (a *App) GetViewManager() *ViewManager {
	return a.viewManager
}
