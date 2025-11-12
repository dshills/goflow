package tui

import (
	"testing"
)

// TestKeyboardHandler_TypeSafety tests that Mode type is used consistently
func TestKeyboardHandler_TypeSafety(t *testing.T) {
	kh := NewKeyboardHandler()

	// Test that we can register bindings with Mode constants
	testHandler := func(event KeyEvent) error { return nil }

	modes := []Mode{ModeNormal, ModeInsert, ModeVisual, ModeCommand}
	for _, mode := range modes {
		err := kh.RegisterBinding(mode, KeyEvent{Key: 'x'}, testHandler, "test")
		if err != nil {
			t.Errorf("RegisterBinding(%s) failed: %v", mode, err)
		}
	}
}

// TestKeyboardHandler_ModeGlobal tests the ModeGlobal constant
func TestKeyboardHandler_ModeGlobal(t *testing.T) {
	// This test ensures ModeGlobal constant exists and is of Mode type
	var mode Mode = ModeGlobal
	if mode != "global" {
		t.Errorf("ModeGlobal = %q, want %q", mode, "global")
	}
}

// TestKeyboardHandler_GetAllBindings tests that GetAllBindings returns map[Mode]...
func TestKeyboardHandler_GetAllBindings_TypeSafety(t *testing.T) {
	kh := NewKeyboardHandler()

	// Register bindings for different modes
	testHandler := func(event KeyEvent) error { return nil }

	// Register mode-specific binding
	err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: 'h'}, testHandler, "left")
	if err != nil {
		t.Fatalf("RegisterBinding(ModeNormal) failed: %v", err)
	}

	// Register global binding
	err = kh.RegisterGlobalBinding(KeyEvent{Key: 'q'}, testHandler, "quit")
	if err != nil {
		t.Fatalf("RegisterGlobalBinding() failed: %v", err)
	}

	// Get all bindings - this must compile with map[Mode]... type
	allBindings := kh.GetAllBindings()

	// Verify return type is map[Mode][]*KeyBinding
	var _ map[Mode][]*KeyBinding = allBindings

	// Verify we can access bindings with Mode constants (type-safe at compile time)
	normalBindings, ok := allBindings[ModeNormal]
	if !ok || len(normalBindings) == 0 {
		t.Error("GetAllBindings()[ModeNormal] should have bindings")
	}

	// Verify global bindings are stored under ModeGlobal (not string "global")
	globalBindings, ok := allBindings[ModeGlobal]
	if !ok || len(globalBindings) == 0 {
		t.Error("GetAllBindings()[ModeGlobal] should have global bindings")
	}

	// Verify that string "global" does NOT work (compile-time type safety)
	// This line should NOT compile if we try to use string:
	// _ = allBindings["global"] // This would be a compile error with proper typing
}

// TestKeyboardHandler_GlobalBindings tests global binding registration and retrieval
func TestKeyboardHandler_GlobalBindings(t *testing.T) {
	kh := NewKeyboardHandler()

	testHandler := func(event KeyEvent) error {
		return nil
	}

	// Register global binding
	err := kh.RegisterGlobalBinding(KeyEvent{Key: 'q'}, testHandler, "quit")
	if err != nil {
		t.Fatalf("RegisterGlobalBinding() failed: %v", err)
	}

	// Get global bindings
	globalBindings := kh.GetGlobalBindings()
	if len(globalBindings) != 1 {
		t.Errorf("GetGlobalBindings() count = %d, want 1", len(globalBindings))
	}

	// Verify global binding appears in GetAllBindings under ModeGlobal
	allBindings := kh.GetAllBindings()
	globalFromAll, ok := allBindings[ModeGlobal]
	if !ok {
		t.Fatal("GetAllBindings()[ModeGlobal] not found")
	}
	if len(globalFromAll) != 1 {
		t.Errorf("GetAllBindings()[ModeGlobal] count = %d, want 1", len(globalFromAll))
	}

	// Verify global binding is marked as global
	if !globalFromAll[0].IsGlobal {
		t.Error("Global binding should have IsGlobal = true")
	}
}

// TestKeyboardHandler_ModeSpecificBindings tests mode-specific bindings
func TestKeyboardHandler_ModeSpecificBindings(t *testing.T) {
	kh := NewKeyboardHandler()

	testHandler := func(event KeyEvent) error { return nil }

	// Register bindings for all modes
	modes := []Mode{ModeNormal, ModeInsert, ModeVisual, ModeCommand}
	for i, mode := range modes {
		key := rune('a' + i)
		err := kh.RegisterBinding(mode, KeyEvent{Key: key}, testHandler, "test")
		if err != nil {
			t.Errorf("RegisterBinding(%s) failed: %v", mode, err)
		}
	}

	// Verify all bindings are retrievable with Mode type
	allBindings := kh.GetAllBindings()

	for _, mode := range modes {
		bindings, ok := allBindings[mode]
		if !ok || len(bindings) == 0 {
			t.Errorf("GetAllBindings()[%s] should have bindings", mode)
		}

		// Verify mode is set correctly
		if len(bindings) > 0 && bindings[0].Mode != mode {
			t.Errorf("Binding mode = %s, want %s", bindings[0].Mode, mode)
		}
	}
}

// TestKeyboardHandler_BindingConflicts tests duplicate binding detection
func TestKeyboardHandler_BindingConflicts(t *testing.T) {
	kh := NewKeyboardHandler()

	testHandler := func(event KeyEvent) error { return nil }
	key := KeyEvent{Key: 'h'}

	// Register first binding
	err := kh.RegisterBinding(ModeNormal, key, testHandler, "first")
	if err != nil {
		t.Fatalf("First RegisterBinding() failed: %v", err)
	}

	// Try to register conflicting binding in same mode
	err = kh.RegisterBinding(ModeNormal, key, testHandler, "second")
	if err == nil {
		t.Error("RegisterBinding() should fail on duplicate key in same mode")
	}

	// Should be able to register same key in different mode
	err = kh.RegisterBinding(ModeInsert, key, testHandler, "insert mode")
	if err != nil {
		t.Errorf("RegisterBinding(ModeInsert) should succeed: %v", err)
	}

	// Test global binding conflicts
	globalKey := KeyEvent{Key: 'q'}
	err = kh.RegisterGlobalBinding(globalKey, testHandler, "first global")
	if err != nil {
		t.Fatalf("First RegisterGlobalBinding() failed: %v", err)
	}

	err = kh.RegisterGlobalBinding(globalKey, testHandler, "second global")
	if err == nil {
		t.Error("RegisterGlobalBinding() should fail on duplicate key")
	}
}

// TestKeyboardHandler_Unregister tests binding removal
func TestKeyboardHandler_Unregister(t *testing.T) {
	kh := NewKeyboardHandler()

	testHandler := func(event KeyEvent) error { return nil }
	key := KeyEvent{Key: 'h'}

	// Register and verify
	err := kh.RegisterBinding(ModeNormal, key, testHandler, "test")
	if err != nil {
		t.Fatalf("RegisterBinding() failed: %v", err)
	}

	bindings := kh.GetBindings(ModeNormal)
	if len(bindings) == 0 {
		t.Fatal("Should have binding after registration")
	}

	// Unregister and verify
	kh.UnregisterBinding(ModeNormal, key)
	bindings = kh.GetBindings(ModeNormal)
	if len(bindings) != 0 {
		t.Errorf("Should have no bindings after unregister, got %d", len(bindings))
	}

	// Test global unregister
	globalKey := KeyEvent{Key: 'q'}
	err = kh.RegisterGlobalBinding(globalKey, testHandler, "quit")
	if err != nil {
		t.Fatalf("RegisterGlobalBinding() failed: %v", err)
	}

	kh.UnregisterGlobalBinding(globalKey)
	globalBindings := kh.GetGlobalBindings()
	if len(globalBindings) != 0 {
		t.Errorf("Should have no global bindings after unregister, got %d", len(globalBindings))
	}
}

// TestKeyboardHandler_GetBindings tests mode-specific binding retrieval
func TestKeyboardHandler_GetBindings(t *testing.T) {
	kh := NewKeyboardHandler()

	testHandler := func(event KeyEvent) error { return nil }

	// Register multiple bindings for normal mode
	keys := []rune{'h', 'j', 'k', 'l'}
	for _, key := range keys {
		err := kh.RegisterBinding(ModeNormal, KeyEvent{Key: key}, testHandler, "nav")
		if err != nil {
			t.Fatalf("RegisterBinding() failed: %v", err)
		}
	}

	// Get bindings for normal mode
	bindings := kh.GetBindings(ModeNormal)
	if len(bindings) != len(keys) {
		t.Errorf("GetBindings(ModeNormal) count = %d, want %d", len(bindings), len(keys))
	}

	// Get bindings for insert mode (should be empty)
	insertBindings := kh.GetBindings(ModeInsert)
	if len(insertBindings) != 0 {
		t.Errorf("GetBindings(ModeInsert) count = %d, want 0", len(insertBindings))
	}
}
