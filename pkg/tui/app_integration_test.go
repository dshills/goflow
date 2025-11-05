package tui

import (
	"testing"
	"time"

	"github.com/dshills/goterm"
)

// TestAppIntegration tests the complete app integration
func TestAppIntegration(t *testing.T) {
	// Skip if not in a terminal environment
	screen, err := goterm.Init()
	if err != nil {
		t.Skip("Skipping: not running in a terminal environment")
		return
	}
	screen.Close()

	// Create new app
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	defer app.Close()

	// Verify screen initialized
	if app.screen == nil {
		t.Fatal("screen not initialized")
	}

	// Verify view manager initialized
	if app.viewManager == nil {
		t.Fatal("view manager not initialized")
	}

	// Verify keyboard handler initialized
	if app.keyboard == nil {
		t.Fatal("keyboard handler not initialized")
	}

	// Verify views registered
	views := app.viewManager.ListViews()
	expectedViews := []string{"explorer", "builder", "monitor", "registry"}
	if len(views) != len(expectedViews) {
		t.Errorf("expected %d views, got %d", len(expectedViews), len(views))
	}

	// Verify initial view is active
	currentView := app.viewManager.GetCurrentView()
	if currentView == nil {
		t.Fatal("no active view")
	}
	if currentView.Name() != "explorer" {
		t.Errorf("expected initial view 'explorer', got '%s'", currentView.Name())
	}

	// Test view switching
	err = app.viewManager.SwitchTo("builder")
	if err != nil {
		t.Fatalf("SwitchTo('builder') failed: %v", err)
	}

	currentView = app.viewManager.GetCurrentView()
	if currentView.Name() != "builder" {
		t.Errorf("expected current view 'builder', got '%s'", currentView.Name())
	}

	// Test keyboard handler mode
	mode := app.keyboard.GetMode()
	if mode != ModeNormal {
		t.Errorf("expected initial mode 'normal', got '%s'", mode)
	}

	// Test render performance
	start := time.Now()
	err = app.render()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("render() failed: %v", err)
	}

	// Constitutional requirement: < 16ms frame time for 60 FPS
	targetFrameTime := 16 * time.Millisecond
	if duration > targetFrameTime {
		t.Errorf("render() took %v, exceeds constitutional target of %v", duration, targetFrameTime)
	}
}

// TestAppKeyboardInput tests keyboard input parsing
func TestAppKeyboardInput(t *testing.T) {
	app := &App{} // Minimal app for testing

	tests := []struct {
		name             string
		input            []byte
		expectedKey      rune
		expectedCtrl     bool
		expectedShift    bool
		expectedAlt      bool
		expectedSpecial  string
		expectSpecialKey bool
	}{
		{
			name:        "regular character 'a'",
			input:       []byte{'a'},
			expectedKey: 'a',
		},
		{
			name:          "uppercase character 'A'",
			input:         []byte{'A'},
			expectedKey:   'A',
			expectedShift: true,
		},
		{
			name:             "tab key",
			input:            []byte{9},
			expectSpecialKey: true,
			expectedSpecial:  "Tab",
		},
		{
			name:             "escape key",
			input:            []byte{27},
			expectSpecialKey: true,
			expectedSpecial:  "Escape",
		},
		{
			name:         "ctrl+c",
			input:        []byte{3},
			expectedKey:  'c',
			expectedCtrl: true,
		},
		{
			name:             "arrow up",
			input:            []byte{27, '[', 'A'},
			expectSpecialKey: true,
			expectedSpecial:  "Up",
		},
		{
			name:             "arrow down",
			input:            []byte{27, '[', 'B'},
			expectSpecialKey: true,
			expectedSpecial:  "Down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := app.parseKeyInput(tt.input)

			if tt.expectSpecialKey {
				if !event.IsSpecial {
					t.Errorf("expected special key, got regular key")
				}
				if event.Special != tt.expectedSpecial {
					t.Errorf("expected special '%s', got '%s'", tt.expectedSpecial, event.Special)
				}
			} else {
				if event.IsSpecial {
					t.Errorf("expected regular key, got special key '%s'", event.Special)
				}
				if event.Key != tt.expectedKey {
					t.Errorf("expected key '%c', got '%c'", tt.expectedKey, event.Key)
				}
				if event.Ctrl != tt.expectedCtrl {
					t.Errorf("expected Ctrl=%v, got Ctrl=%v", tt.expectedCtrl, event.Ctrl)
				}
				if event.Shift != tt.expectedShift {
					t.Errorf("expected Shift=%v, got Shift=%v", tt.expectedShift, event.Shift)
				}
			}
		})
	}
}

// TestAppClose tests graceful shutdown
func TestAppClose(t *testing.T) {
	screen, err := goterm.Init()
	if err != nil {
		t.Skip("Skipping: not running in a terminal environment")
		return
	}
	screen.Close()

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Close should not error
	err = app.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify screen is closed (context cancelled)
	select {
	case <-app.ctx.Done():
		// Context is done, as expected
	default:
		t.Error("context not cancelled after Close()")
	}
}

// Benchmark for render performance
func BenchmarkRender(b *testing.B) {
	screen, err := goterm.Init()
	if err != nil {
		b.Skip("Skipping: not running in a terminal environment")
		return
	}
	screen.Close()

	app, err := NewApp()
	if err != nil {
		b.Fatalf("NewApp() failed: %v", err)
	}
	defer app.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := app.render(); err != nil {
			b.Fatalf("render() failed: %v", err)
		}
	}
}
