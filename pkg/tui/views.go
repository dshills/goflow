package tui

import (
	"fmt"
	"sync"

	"github.com/dshills/goterm"
)

// View defines the interface that all TUI views must implement
type View interface {
	// Name returns the unique identifier for this view
	Name() string

	// Init initializes the view, called before first render
	Init() error

	// Cleanup releases resources when view is deactivated
	Cleanup() error

	// HandleKey processes keyboard input events
	HandleKey(event KeyEvent) error

	// Render draws the view to the screen
	Render(screen *goterm.Screen) error

	// IsActive returns whether this view is currently active
	IsActive() bool

	// SetActive updates the active state of the view
	SetActive(active bool)
}

// ViewManager manages view registration, switching, and lifecycle
type ViewManager struct {
	views       map[string]View
	activeView  View
	history     []string // view name history for back navigation
	mu          sync.RWMutex
	onSwitch    []func(from, to View) error // transition hooks
	initialized bool
}

// NewViewManager creates a new view manager
func NewViewManager() *ViewManager {
	return &ViewManager{
		views:    make(map[string]View),
		history:  make([]string, 0),
		onSwitch: make([]func(from, to View) error, 0),
	}
}

// RegisterView adds a view to the manager's registry
func (vm *ViewManager) RegisterView(view View) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if view == nil {
		return fmt.Errorf("cannot register nil view")
	}

	name := view.Name()
	if name == "" {
		return fmt.Errorf("view name cannot be empty")
	}

	if _, exists := vm.views[name]; exists {
		return fmt.Errorf("view %q already registered", name)
	}

	vm.views[name] = view
	return nil
}

// SwitchTo switches to the named view, handling cleanup and initialization
func (vm *ViewManager) SwitchTo(viewName string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	newView, exists := vm.views[viewName]
	if !exists {
		return fmt.Errorf("view %q not found", viewName)
	}

	// Same view, nothing to do
	if vm.activeView == newView {
		return nil
	}

	oldView := vm.activeView

	// Cleanup old view
	if oldView != nil {
		if err := oldView.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup view %q: %w", oldView.Name(), err)
		}
		oldView.SetActive(false)

		// Add to history for back navigation
		vm.history = append(vm.history, oldView.Name())
	}

	// Run transition hooks
	for _, hook := range vm.onSwitch {
		if err := hook(oldView, newView); err != nil {
			// Attempt to restore old view on hook failure
			if oldView != nil {
				oldView.SetActive(true)
				_ = oldView.Init() // ignore init error on rollback
			}
			return fmt.Errorf("view switch hook failed: %w", err)
		}
	}

	// Initialize new view
	if err := newView.Init(); err != nil {
		// Attempt to restore old view on init failure
		if oldView != nil {
			oldView.SetActive(true)
			_ = oldView.Init() // ignore init error on rollback
		}
		return fmt.Errorf("failed to initialize view %q: %w", viewName, err)
	}

	newView.SetActive(true)
	vm.activeView = newView
	return nil
}

// GetCurrentView returns the currently active view
func (vm *ViewManager) GetCurrentView() View {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.activeView
}

// GoBack returns to the previous view in the history
func (vm *ViewManager) GoBack() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if len(vm.history) == 0 {
		return fmt.Errorf("no previous view in history")
	}

	// Get last view from history
	prevViewName := vm.history[len(vm.history)-1]
	vm.history = vm.history[:len(vm.history)-1]

	prevView, exists := vm.views[prevViewName]
	if !exists {
		return fmt.Errorf("previous view %q no longer exists", prevViewName)
	}

	// Cleanup current view
	if vm.activeView != nil {
		if err := vm.activeView.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup current view: %w", err)
		}
		vm.activeView.SetActive(false)
	}

	// Initialize previous view
	if err := prevView.Init(); err != nil {
		return fmt.Errorf("failed to initialize previous view: %w", err)
	}

	prevView.SetActive(true)
	vm.activeView = prevView
	return nil
}

// ListViews returns the names of all registered views
func (vm *ViewManager) ListViews() []string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	names := make([]string, 0, len(vm.views))
	for name := range vm.views {
		names = append(names, name)
	}
	return names
}

// GetView retrieves a view by name
func (vm *ViewManager) GetView(name string) (View, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	view, exists := vm.views[name]
	if !exists {
		return nil, fmt.Errorf("view %q not found", name)
	}
	return view, nil
}

// AddSwitchHook registers a callback that runs during view transitions
func (vm *ViewManager) AddSwitchHook(hook func(from, to View) error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.onSwitch = append(vm.onSwitch, hook)
}

// ClearHistory removes all view history
func (vm *ViewManager) ClearHistory() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.history = make([]string, 0)
}

// Initialize sets up the view manager and activates the initial view
func (vm *ViewManager) Initialize(initialViewName string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.initialized {
		return fmt.Errorf("view manager already initialized")
	}

	view, exists := vm.views[initialViewName]
	if !exists {
		return fmt.Errorf("initial view %q not found", initialViewName)
	}

	if err := view.Init(); err != nil {
		return fmt.Errorf("failed to initialize initial view: %w", err)
	}

	view.SetActive(true)
	vm.activeView = view
	vm.initialized = true
	return nil
}

// Shutdown cleanups all views and releases resources
func (vm *ViewManager) Shutdown() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Cleanup active view
	if vm.activeView != nil {
		if err := vm.activeView.Cleanup(); err != nil {
			return fmt.Errorf("failed to cleanup active view: %w", err)
		}
	}

	vm.activeView = nil
	vm.history = make([]string, 0)
	vm.initialized = false
	return nil
}

// NextView cycles to the next view in alphabetical order
// Used for Tab key navigation
func (vm *ViewManager) NextView() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if len(vm.views) == 0 {
		return fmt.Errorf("no views registered")
	}

	// Get sorted view names
	names := make([]string, 0, len(vm.views))
	for name := range vm.views {
		names = append(names, name)
	}

	if len(names) == 1 {
		return nil // only one view, nothing to switch
	}

	// Sort names for consistent ordering
	sortedNames := sortStrings(names)

	// Find current view index
	currentIdx := -1
	if vm.activeView != nil {
		for i, name := range sortedNames {
			if name == vm.activeView.Name() {
				currentIdx = i
				break
			}
		}
	}

	// Calculate next index
	nextIdx := (currentIdx + 1) % len(sortedNames)
	nextViewName := sortedNames[nextIdx]

	// Unlock before calling SwitchTo to avoid deadlock
	vm.mu.Unlock()
	err := vm.SwitchTo(nextViewName)
	vm.mu.Lock()

	return err
}

// sortStrings provides a simple string sort without external dependencies
func sortStrings(strs []string) []string {
	// Simple insertion sort for small arrays
	sorted := make([]string, len(strs))
	copy(sorted, strs)

	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	return sorted
}
