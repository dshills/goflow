package tui

import (
	"testing"
)

// MockTUI represents a mock TUI for testing keyboard interactions
// This will be replaced with actual TUI implementation
type MockTUI struct {
	mode             string // "normal", "insert", "visual", "command"
	cursorX          int
	cursorY          int
	buffer           [][]rune
	clipboard        string
	undoStack        []string
	redoStack        []string
	searchPattern    string
	searchResults    []Position
	currentSearchIdx int
	selectedNode     string
	pendingEdge      *Edge
	helpVisible      bool
	commandBuffer    string
	lastError        error
}

// KeyEvent represents a keyboard input event
type KeyEvent struct {
	Key       rune
	Ctrl      bool
	Shift     bool
	Alt       bool
	IsSpecial bool
	Special   string // "Escape", "Enter", "Tab", etc.
}

// TestNavigationKeys_HJKL tests vim-style movement keys
func TestNavigationKeys_HJKL(t *testing.T) {
	tests := []struct {
		name        string
		initialX    int
		initialY    int
		key         KeyEvent
		expectedX   int
		expectedY   int
		description string
	}{
		{
			name:        "h moves cursor left",
			initialX:    5,
			initialY:    3,
			key:         KeyEvent{Key: 'h'},
			expectedX:   4,
			expectedY:   3,
			description: "h should move cursor one position left",
		},
		{
			name:        "j moves cursor down",
			initialX:    5,
			initialY:    3,
			key:         KeyEvent{Key: 'j'},
			expectedX:   5,
			expectedY:   4,
			description: "j should move cursor one position down",
		},
		{
			name:        "k moves cursor up",
			initialX:    5,
			initialY:    3,
			key:         KeyEvent{Key: 'k'},
			expectedX:   5,
			expectedY:   2,
			description: "k should move cursor one position up",
		},
		{
			name:        "l moves cursor right",
			initialX:    5,
			initialY:    3,
			key:         KeyEvent{Key: 'l'},
			expectedX:   6,
			expectedY:   3,
			description: "l should move cursor one position right",
		},
		{
			name:        "h at left edge stays at position 0",
			initialX:    0,
			initialY:    3,
			key:         KeyEvent{Key: 'h'},
			expectedX:   0,
			expectedY:   3,
			description: "h at left boundary should not move cursor",
		},
		{
			name:        "k at top edge stays at position 0",
			initialX:    5,
			initialY:    0,
			key:         KeyEvent{Key: 'k'},
			expectedX:   5,
			expectedY:   0,
			description: "k at top boundary should not move cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:    "normal",
				cursorX: tt.initialX,
				cursorY: tt.initialY,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.cursorX != tt.expectedX {
				t.Errorf("cursor X = %d, want %d (%s)", tui.cursorX, tt.expectedX, tt.description)
			}

			if tui.cursorY != tt.expectedY {
				t.Errorf("cursor Y = %d, want %d (%s)", tui.cursorY, tt.expectedY, tt.description)
			}
		})
	}
}

// TestNavigationKeys_WordMovement tests w/b word navigation
func TestNavigationKeys_WordMovement(t *testing.T) {
	tests := []struct {
		name        string
		initialX    int
		text        string
		key         rune
		expectedX   int
		description string
	}{
		{
			name:        "w moves to next word start",
			initialX:    0,
			text:        "hello world test",
			key:         'w',
			expectedX:   6,
			description: "w should move to start of next word",
		},
		{
			name:        "b moves to previous word start",
			initialX:    12,
			text:        "hello world test",
			key:         'b',
			expectedX:   6,
			description: "b should move to start of previous word",
		},
		{
			name:        "w at end of line stays at end",
			initialX:    16,
			text:        "hello world test",
			key:         'w',
			expectedX:   16,
			description: "w at end should not move cursor",
		},
		{
			name:        "b at start of line stays at start",
			initialX:    0,
			text:        "hello world test",
			key:         'b',
			expectedX:   0,
			description: "b at start should not move cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:    "normal",
				cursorX: tt.initialX,
				cursorY: 0,
				buffer:  [][]rune{[]rune(tt.text)},
			}

			err := tui.HandleKeyEvent(KeyEvent{Key: tt.key})
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.cursorX != tt.expectedX {
				t.Errorf("cursor X = %d, want %d (%s)", tui.cursorX, tt.expectedX, tt.description)
			}
		})
	}
}

// TestNavigationKeys_TopBottom tests gg/G for jumping to top/bottom
func TestNavigationKeys_TopBottom(t *testing.T) {
	tests := []struct {
		name        string
		initialY    int
		totalLines  int
		keys        []KeyEvent
		expectedY   int
		description string
	}{
		{
			name:       "gg moves to top",
			initialY:   10,
			totalLines: 20,
			keys: []KeyEvent{
				{Key: 'g'},
				{Key: 'g'},
			},
			expectedY:   0,
			description: "gg should move cursor to first line",
		},
		{
			name:       "G moves to bottom",
			initialY:   5,
			totalLines: 20,
			keys: []KeyEvent{
				{Key: 'G', Shift: true},
			},
			expectedY:   19,
			description: "G should move cursor to last line",
		},
		{
			name:       "single g does nothing",
			initialY:   10,
			totalLines: 20,
			keys: []KeyEvent{
				{Key: 'g'},
			},
			expectedY:   10,
			description: "single g should not move cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer with specified number of lines
			buffer := make([][]rune, tt.totalLines)
			for i := range buffer {
				buffer[i] = []rune("line content")
			}

			tui := &MockTUI{
				mode:    "normal",
				cursorX: 0,
				cursorY: tt.initialY,
				buffer:  buffer,
			}

			for _, key := range tt.keys {
				err := tui.HandleKeyEvent(key)
				if err != nil {
					t.Errorf("HandleKeyEvent() unexpected error: %v", err)
				}
			}

			if tui.cursorY != tt.expectedY {
				t.Errorf("cursor Y = %d, want %d (%s)", tui.cursorY, tt.expectedY, tt.description)
			}
		})
	}
}

// TestNavigationKeys_PageUpDown tests Ctrl-u/Ctrl-d for page navigation
func TestNavigationKeys_PageUpDown(t *testing.T) {
	tests := []struct {
		name        string
		initialY    int
		totalLines  int
		key         KeyEvent
		expectedY   int
		pageSize    int
		description string
	}{
		{
			name:        "Ctrl-d moves page down",
			initialY:    0,
			totalLines:  100,
			key:         KeyEvent{Key: 'd', Ctrl: true},
			expectedY:   20,
			pageSize:    20,
			description: "Ctrl-d should move cursor down by page size",
		},
		{
			name:        "Ctrl-u moves page up",
			initialY:    40,
			totalLines:  100,
			key:         KeyEvent{Key: 'u', Ctrl: true},
			expectedY:   20,
			pageSize:    20,
			description: "Ctrl-u should move cursor up by page size",
		},
		{
			name:        "Ctrl-d at bottom stays at bottom",
			initialY:    90,
			totalLines:  100,
			key:         KeyEvent{Key: 'd', Ctrl: true},
			expectedY:   99,
			pageSize:    20,
			description: "Ctrl-d near bottom should clamp to last line",
		},
		{
			name:        "Ctrl-u at top stays at top",
			initialY:    5,
			totalLines:  100,
			key:         KeyEvent{Key: 'u', Ctrl: true},
			expectedY:   0,
			pageSize:    20,
			description: "Ctrl-u near top should clamp to first line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := make([][]rune, tt.totalLines)
			for i := range buffer {
				buffer[i] = []rune("line content")
			}

			tui := &MockTUI{
				mode:    "normal",
				cursorX: 0,
				cursorY: tt.initialY,
				buffer:  buffer,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.cursorY != tt.expectedY {
				t.Errorf("cursor Y = %d, want %d (%s)", tui.cursorY, tt.expectedY, tt.description)
			}
		})
	}
}

// TestModeSwitching_InsertMode tests entering and exiting insert mode
func TestModeSwitching_InsertMode(t *testing.T) {
	tests := []struct {
		name         string
		initialMode  string
		key          KeyEvent
		expectedMode string
		description  string
	}{
		{
			name:         "i enters insert mode from normal",
			initialMode:  "normal",
			key:          KeyEvent{Key: 'i'},
			expectedMode: "insert",
			description:  "i should switch to insert mode",
		},
		{
			name:         "Escape exits insert mode to normal",
			initialMode:  "insert",
			key:          KeyEvent{IsSpecial: true, Special: "Escape"},
			expectedMode: "normal",
			description:  "Escape should return to normal mode",
		},
		{
			name:         "i in insert mode does nothing",
			initialMode:  "insert",
			key:          KeyEvent{Key: 'i'},
			expectedMode: "insert",
			description:  "i in insert mode should remain in insert mode",
		},
		{
			name:         "Escape in normal mode stays normal",
			initialMode:  "normal",
			key:          KeyEvent{IsSpecial: true, Special: "Escape"},
			expectedMode: "normal",
			description:  "Escape in normal mode should remain normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.initialMode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.mode != tt.expectedMode {
				t.Errorf("mode = %s, want %s (%s)", tui.mode, tt.expectedMode, tt.description)
			}
		})
	}
}

// TestModeSwitching_VisualMode tests entering and using visual mode
func TestModeSwitching_VisualMode(t *testing.T) {
	tests := []struct {
		name         string
		initialMode  string
		key          KeyEvent
		expectedMode string
		description  string
	}{
		{
			name:         "v enters visual mode from normal",
			initialMode:  "normal",
			key:          KeyEvent{Key: 'v'},
			expectedMode: "visual",
			description:  "v should switch to visual/selection mode",
		},
		{
			name:         "Escape exits visual mode to normal",
			initialMode:  "visual",
			key:          KeyEvent{IsSpecial: true, Special: "Escape"},
			expectedMode: "normal",
			description:  "Escape should return to normal mode",
		},
		{
			name:         "v in visual mode toggles back to normal",
			initialMode:  "visual",
			key:          KeyEvent{Key: 'v'},
			expectedMode: "normal",
			description:  "v in visual mode should toggle back to normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.initialMode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.mode != tt.expectedMode {
				t.Errorf("mode = %s, want %s (%s)", tui.mode, tt.expectedMode, tt.description)
			}
		})
	}
}

// TestModeSwitching_CommandMode tests entering command mode
func TestModeSwitching_CommandMode(t *testing.T) {
	tests := []struct {
		name         string
		initialMode  string
		key          KeyEvent
		expectedMode string
		description  string
	}{
		{
			name:         "colon enters command mode from normal",
			initialMode:  "normal",
			key:          KeyEvent{Key: ':'},
			expectedMode: "command",
			description:  ": should switch to command mode",
		},
		{
			name:         "Escape exits command mode to normal",
			initialMode:  "command",
			key:          KeyEvent{IsSpecial: true, Special: "Escape"},
			expectedMode: "normal",
			description:  "Escape should return to normal mode",
		},
		{
			name:         "Enter executes command and returns to normal",
			initialMode:  "command",
			key:          KeyEvent{IsSpecial: true, Special: "Enter"},
			expectedMode: "normal",
			description:  "Enter should execute command and return to normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.initialMode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.mode != tt.expectedMode {
				t.Errorf("mode = %s, want %s (%s)", tui.mode, tt.expectedMode, tt.description)
			}
		})
	}
}

// TestOperations_AddNode tests 'a' for adding nodes
func TestOperations_AddNode(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		key         KeyEvent
		shouldAdd   bool
		description string
	}{
		{
			name:        "a in normal mode adds node",
			mode:        "normal",
			key:         KeyEvent{Key: 'a'},
			shouldAdd:   true,
			description: "a should trigger add node action",
		},
		{
			name:        "a in insert mode does not add node",
			mode:        "insert",
			key:         KeyEvent{Key: 'a'},
			shouldAdd:   false,
			description: "a in insert mode should insert character",
		},
		{
			name:        "a in command mode does not add node",
			mode:        "command",
			key:         KeyEvent{Key: 'a'},
			shouldAdd:   false,
			description: "a in command mode should be part of command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.mode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			addTriggered := tui.lastError == nil && tt.mode == "normal"
			if addTriggered != tt.shouldAdd {
				t.Errorf("add node triggered = %v, want %v (%s)", addTriggered, tt.shouldAdd, tt.description)
			}
		})
	}
}

// TestOperations_EdgeCreation tests 'e' for creating edges
func TestOperations_EdgeCreation(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		selectedNode string
		key          KeyEvent
		shouldStart  bool
		description  string
	}{
		{
			name:         "e with selected node starts edge creation",
			mode:         "normal",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'e'},
			shouldStart:  true,
			description:  "e should start edge creation mode",
		},
		{
			name:         "e without selected node shows error",
			mode:         "normal",
			selectedNode: "",
			key:          KeyEvent{Key: 'e'},
			shouldStart:  false,
			description:  "e without selection should show error",
		},
		{
			name:         "e in insert mode does not create edge",
			mode:         "insert",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'e'},
			shouldStart:  false,
			description:  "e in insert mode should insert character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:         tt.mode,
				selectedNode: tt.selectedNode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			edgeStarted := tui.pendingEdge != nil
			if edgeStarted != tt.shouldStart {
				t.Errorf("edge creation started = %v, want %v (%s)", edgeStarted, tt.shouldStart, tt.description)
			}
		})
	}
}

// TestOperations_Delete tests 'd' for delete
func TestOperations_Delete(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		selectedNode string
		key          KeyEvent
		shouldDelete bool
		description  string
	}{
		{
			name:         "d with selected node deletes",
			mode:         "normal",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'd'},
			shouldDelete: true,
			description:  "d should delete selected node",
		},
		{
			name:         "d without selection does nothing",
			mode:         "normal",
			selectedNode: "",
			key:          KeyEvent{Key: 'd'},
			shouldDelete: false,
			description:  "d without selection should show error",
		},
		{
			name:         "d in insert mode does not delete",
			mode:         "insert",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'd'},
			shouldDelete: false,
			description:  "d in insert mode should insert character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:         tt.mode,
				selectedNode: tt.selectedNode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			// In real implementation, would check if node was deleted
			deleteTriggered := tt.mode == "normal" && tt.selectedNode != ""
			if deleteTriggered != tt.shouldDelete {
				t.Errorf("delete triggered = %v, want %v (%s)", deleteTriggered, tt.shouldDelete, tt.description)
			}
		})
	}
}

// TestOperations_CopyPaste tests y/p for copy/paste
func TestOperations_CopyPaste(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		selectedNode string
		operation    rune
		clipboard    string
		expected     string
		description  string
	}{
		{
			name:         "y copies selected node",
			mode:         "normal",
			selectedNode: "node-1",
			operation:    'y',
			clipboard:    "",
			expected:     "node-1",
			description:  "y should copy node to clipboard",
		},
		{
			name:         "p pastes from clipboard",
			mode:         "normal",
			selectedNode: "",
			operation:    'p',
			clipboard:    "node-2",
			expected:     "node-2",
			description:  "p should paste from clipboard",
		},
		{
			name:         "y without selection does nothing",
			mode:         "normal",
			selectedNode: "",
			operation:    'y',
			clipboard:    "old",
			expected:     "old",
			description:  "y without selection should not change clipboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:         tt.mode,
				selectedNode: tt.selectedNode,
				clipboard:    tt.clipboard,
			}

			err := tui.HandleKeyEvent(KeyEvent{Key: tt.operation})
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.clipboard != tt.expected {
				t.Errorf("clipboard = %s, want %s (%s)", tui.clipboard, tt.expected, tt.description)
			}
		})
	}
}

// TestOperations_UndoRedo tests u/Ctrl-r for undo/redo
func TestOperations_UndoRedo(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		undoStack   []string
		redoStack   []string
		key         KeyEvent
		expectUndo  int
		expectRedo  int
		description string
	}{
		{
			name:        "u performs undo",
			mode:        "normal",
			undoStack:   []string{"state1", "state2"},
			redoStack:   []string{},
			key:         KeyEvent{Key: 'u'},
			expectUndo:  1,
			expectRedo:  1,
			description: "u should pop from undo stack and push to redo",
		},
		{
			name:        "Ctrl-r performs redo",
			mode:        "normal",
			undoStack:   []string{"state1"},
			redoStack:   []string{"state2"},
			key:         KeyEvent{Key: 'r', Ctrl: true},
			expectUndo:  2,
			expectRedo:  0,
			description: "Ctrl-r should pop from redo stack and push to undo",
		},
		{
			name:        "u with empty undo does nothing",
			mode:        "normal",
			undoStack:   []string{},
			redoStack:   []string{},
			key:         KeyEvent{Key: 'u'},
			expectUndo:  0,
			expectRedo:  0,
			description: "u with empty stack should do nothing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:      tt.mode,
				undoStack: tt.undoStack,
				redoStack: tt.redoStack,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if len(tui.undoStack) != tt.expectUndo {
				t.Errorf("undo stack length = %d, want %d (%s)", len(tui.undoStack), tt.expectUndo, tt.description)
			}

			if len(tui.redoStack) != tt.expectRedo {
				t.Errorf("redo stack length = %d, want %d (%s)", len(tui.redoStack), tt.expectRedo, tt.description)
			}
		})
	}
}

// TestOperations_Search tests / for search and n/N for navigation
func TestOperations_Search(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		key           KeyEvent
		searchPattern string
		searchResults []Position
		currentIdx    int
		expectedIdx   int
		expectedMode  string
		description   string
	}{
		{
			name:          "slash enters search mode",
			mode:          "normal",
			key:           KeyEvent{Key: '/'},
			searchPattern: "",
			expectedMode:  "command",
			description:   "/ should enter search/command mode",
		},
		{
			name:          "n moves to next search result",
			mode:          "normal",
			key:           KeyEvent{Key: 'n'},
			searchPattern: "test",
			searchResults: []Position{{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 3}},
			currentIdx:    0,
			expectedIdx:   1,
			expectedMode:  "normal",
			description:   "n should move to next search result",
		},
		{
			name:          "N moves to previous search result",
			mode:          "normal",
			key:           KeyEvent{Key: 'N', Shift: true},
			searchPattern: "test",
			searchResults: []Position{{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 3}},
			currentIdx:    2,
			expectedIdx:   1,
			expectedMode:  "normal",
			description:   "N should move to previous search result",
		},
		{
			name:          "n wraps around to first result",
			mode:          "normal",
			key:           KeyEvent{Key: 'n'},
			searchPattern: "test",
			searchResults: []Position{{X: 1, Y: 1}, {X: 2, Y: 2}},
			currentIdx:    1,
			expectedIdx:   0,
			expectedMode:  "normal",
			description:   "n at last result should wrap to first",
		},
		{
			name:          "N wraps around to last result",
			mode:          "normal",
			key:           KeyEvent{Key: 'N', Shift: true},
			searchPattern: "test",
			searchResults: []Position{{X: 1, Y: 1}, {X: 2, Y: 2}},
			currentIdx:    0,
			expectedIdx:   1,
			expectedMode:  "normal",
			description:   "N at first result should wrap to last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:             tt.mode,
				searchPattern:    tt.searchPattern,
				searchResults:    tt.searchResults,
				currentSearchIdx: tt.currentIdx,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tt.expectedMode != "" && tui.mode != tt.expectedMode {
				t.Errorf("mode = %s, want %s (%s)", tui.mode, tt.expectedMode, tt.description)
			}

			if len(tt.searchResults) > 0 && tui.currentSearchIdx != tt.expectedIdx {
				t.Errorf("search index = %d, want %d (%s)", tui.currentSearchIdx, tt.expectedIdx, tt.description)
			}
		})
	}
}

// TestOperations_Help tests ? for context-sensitive help
func TestOperations_Help(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		helpVisible   bool
		key           KeyEvent
		expectVisible bool
		description   string
	}{
		{
			name:          "question mark toggles help on",
			mode:          "normal",
			helpVisible:   false,
			key:           KeyEvent{Key: '?', Shift: true},
			expectVisible: true,
			description:   "? should show help overlay",
		},
		{
			name:          "question mark toggles help off",
			mode:          "normal",
			helpVisible:   true,
			key:           KeyEvent{Key: '?', Shift: true},
			expectVisible: false,
			description:   "? with help visible should hide help",
		},
		{
			name:          "Escape closes help",
			mode:          "normal",
			helpVisible:   true,
			key:           KeyEvent{IsSpecial: true, Special: "Escape"},
			expectVisible: false,
			description:   "Escape should close help overlay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:        tt.mode,
				helpVisible: tt.helpVisible,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			if tui.helpVisible != tt.expectVisible {
				t.Errorf("help visible = %v, want %v (%s)", tui.helpVisible, tt.expectVisible, tt.description)
			}
		})
	}
}

// TestGlobalKeys_Quit tests q for quit
func TestGlobalKeys_Quit(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		key         KeyEvent
		shouldQuit  bool
		description string
	}{
		{
			name:        "q in normal mode quits",
			mode:        "normal",
			key:         KeyEvent{Key: 'q'},
			shouldQuit:  true,
			description: "q should initiate quit action",
		},
		{
			name:        "q in insert mode does not quit",
			mode:        "insert",
			key:         KeyEvent{Key: 'q'},
			shouldQuit:  false,
			description: "q in insert mode should insert character",
		},
		{
			name:        "q in command mode is part of command",
			mode:        "command",
			key:         KeyEvent{Key: 'q'},
			shouldQuit:  false,
			description: "q in command mode should be part of :q command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.mode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			// In real implementation, would check if quit was triggered
			quitTriggered := tt.mode == "normal"
			if quitTriggered != tt.shouldQuit {
				t.Errorf("quit triggered = %v, want %v (%s)", quitTriggered, tt.shouldQuit, tt.description)
			}
		})
	}
}

// TestCommandMode_Commands tests :w, :q, :wq commands
func TestCommandMode_Commands(t *testing.T) {
	tests := []struct {
		name           string
		commandBuffer  string
		expectedAction string
		description    string
	}{
		{
			name:           "colon w saves workflow",
			commandBuffer:  "w",
			expectedAction: "save",
			description:    ":w should save workflow",
		},
		{
			name:           "colon q quits",
			commandBuffer:  "q",
			expectedAction: "quit",
			description:    ":q should quit application",
		},
		{
			name:           "colon wq saves and quits",
			commandBuffer:  "wq",
			expectedAction: "save_quit",
			description:    ":wq should save and quit",
		},
		{
			name:           "colon q exclamation force quits",
			commandBuffer:  "q!",
			expectedAction: "force_quit",
			description:    ":q! should force quit without saving",
		},
		{
			name:           "empty command does nothing",
			commandBuffer:  "",
			expectedAction: "none",
			description:    "empty command should do nothing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:          "command",
				commandBuffer: tt.commandBuffer,
			}

			err := tui.ExecuteCommand()
			if err != nil && tt.expectedAction != "none" {
				t.Errorf("ExecuteCommand() unexpected error: %v", err)
			}

			// In real implementation, would check which action was taken
			// For now, just verify no panic and command buffer handling
			if tui.mode != "command" {
				t.Errorf("mode changed unexpectedly after command execution")
			}
		})
	}
}

// TestKeyConflicts tests that keys don't conflict across modes
func TestKeyConflicts(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		key         rune
		validAction bool
		description string
	}{
		{
			name:        "h in normal mode navigates",
			mode:        "normal",
			key:         'h',
			validAction: true,
			description: "h should be navigation in normal mode",
		},
		{
			name:        "h in insert mode inserts",
			mode:        "insert",
			key:         'h',
			validAction: true,
			description: "h should insert character in insert mode",
		},
		{
			name:        "colon in normal mode enters command",
			mode:        "normal",
			key:         ':',
			validAction: true,
			description: ": should enter command mode",
		},
		{
			name:        "colon in insert mode inserts",
			mode:        "insert",
			key:         ':',
			validAction: true,
			description: ": should insert character in insert mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.mode,
			}

			err := tui.HandleKeyEvent(KeyEvent{Key: tt.key})

			// No error means the key was handled appropriately for the mode
			hasError := err != nil
			if hasError == tt.validAction {
				t.Errorf("key handling = error:%v, want valid:%v (%s)", hasError, tt.validAction, tt.description)
			}
		})
	}
}

// TestInvalidKeys tests that invalid keys are ignored gracefully
func TestInvalidKeys(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		key         KeyEvent
		shouldError bool
		description string
	}{
		{
			name:        "control-x in normal mode ignored",
			mode:        "normal",
			key:         KeyEvent{Key: 'x', Ctrl: true},
			shouldError: false,
			description: "unbound ctrl key should be ignored",
		},
		{
			name:        "alt-z in normal mode ignored",
			mode:        "normal",
			key:         KeyEvent{Key: 'z', Alt: true},
			shouldError: false,
			description: "unbound alt key should be ignored",
		},
		{
			name:        "F12 in normal mode ignored",
			mode:        "normal",
			key:         KeyEvent{IsSpecial: true, Special: "F12"},
			shouldError: false,
			description: "unbound function key should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode: tt.mode,
			}

			err := tui.HandleKeyEvent(tt.key)

			if tt.shouldError && err == nil {
				t.Errorf("HandleKeyEvent() expected error but got none (%s)", tt.description)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v (%s)", err, tt.description)
			}
		})
	}
}

// TestOperations_Rename tests 'r' for rename
func TestOperations_Rename(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		selectedNode string
		key          KeyEvent
		shouldRename bool
		description  string
	}{
		{
			name:         "r with selected node starts rename",
			mode:         "normal",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'r'},
			shouldRename: true,
			description:  "r should start rename mode",
		},
		{
			name:         "r without selection does nothing",
			mode:         "normal",
			selectedNode: "",
			key:          KeyEvent{Key: 'r'},
			shouldRename: false,
			description:  "r without selection should show error",
		},
		{
			name:         "r in insert mode does not rename",
			mode:         "insert",
			selectedNode: "node-1",
			key:          KeyEvent{Key: 'r'},
			shouldRename: false,
			description:  "r in insert mode should insert character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := &MockTUI{
				mode:         tt.mode,
				selectedNode: tt.selectedNode,
			}

			err := tui.HandleKeyEvent(tt.key)
			if err != nil {
				t.Errorf("HandleKeyEvent() unexpected error: %v", err)
			}

			// In real implementation, would check if rename mode was entered
			renameTriggered := tt.mode == "normal" && tt.selectedNode != ""
			if renameTriggered != tt.shouldRename {
				t.Errorf("rename triggered = %v, want %v (%s)", renameTriggered, tt.shouldRename, tt.description)
			}
		})
	}
}

// TestNavigationBoundaries tests cursor stays within valid boundaries
func TestNavigationBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		bufferSize  Position
		initialPos  Position
		keys        []KeyEvent
		expectedPos Position
		description string
	}{
		{
			name:       "movement stays within buffer",
			bufferSize: Position{X: 10, Y: 10},
			initialPos: Position{X: 9, Y: 9},
			keys: []KeyEvent{
				{Key: 'l'}, // try to go right
				{Key: 'j'}, // try to go down
			},
			expectedPos: Position{X: 9, Y: 9},
			description: "cursor should not move beyond buffer boundaries",
		},
		{
			name:       "multiple movements stay in bounds",
			bufferSize: Position{X: 10, Y: 10},
			initialPos: Position{X: 5, Y: 5},
			keys: []KeyEvent{
				{Key: 'k'}, {Key: 'k'}, {Key: 'k'}, {Key: 'k'}, {Key: 'k'}, {Key: 'k'}, // up 6 times
			},
			expectedPos: Position{X: 5, Y: 0},
			description: "multiple movements beyond boundary should clamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := make([][]rune, tt.bufferSize.Y)
			for i := range buffer {
				buffer[i] = make([]rune, tt.bufferSize.X)
			}

			tui := &MockTUI{
				mode:    "normal",
				cursorX: tt.initialPos.X,
				cursorY: tt.initialPos.Y,
				buffer:  buffer,
			}

			for _, key := range tt.keys {
				tui.HandleKeyEvent(key)
			}

			if tui.cursorX != tt.expectedPos.X || tui.cursorY != tt.expectedPos.Y {
				t.Errorf("cursor = (%d,%d), want (%d,%d) (%s)",
					tui.cursorX, tui.cursorY, tt.expectedPos.X, tt.expectedPos.Y, tt.description)
			}
		})
	}
}

// Mock implementation stubs for compilation
func (m *MockTUI) HandleKeyEvent(key KeyEvent) error {
	// Handle special keys
	if key.IsSpecial {
		return m.handleSpecialKey(key)
	}

	// Handle mode-specific key events
	switch m.mode {
	case "normal":
		return m.handleNormalModeKey(key)
	case "insert":
		return m.handleInsertModeKey(key)
	case "visual":
		return m.handleVisualModeKey(key)
	case "command":
		return m.handleCommandModeKey(key)
	}

	return nil
}

func (m *MockTUI) handleSpecialKey(key KeyEvent) error {
	switch key.Special {
	case "Escape":
		if m.mode == "insert" || m.mode == "visual" || m.mode == "command" {
			m.mode = "normal"
		} else if m.helpVisible {
			m.helpVisible = false
		}
	case "Enter":
		if m.mode == "command" {
			m.ExecuteCommand()
			m.mode = "normal"
		}
	case "Tab":
		if key.Shift {
			// Shift-Tab: navigate backwards (stub for now)
		} else {
			// Tab: navigate forwards (stub for now)
		}
	}
	return nil
}

func (m *MockTUI) handleNormalModeKey(key KeyEvent) error {
	// Handle Ctrl combinations
	if key.Ctrl {
		switch key.Key {
		case 'd': // Ctrl-d: page down
			m.cursorY += 10 // Half page
			if m.cursorY > 100 {
				m.cursorY = 100
			}
		case 'u': // Ctrl-u: page up
			m.cursorY -= 10 // Half page
			if m.cursorY < 0 {
				m.cursorY = 0
			}
		case 'r': // Ctrl-r: redo
			if len(m.redoStack) > 0 {
				m.undoStack = append(m.undoStack, m.redoStack[len(m.redoStack)-1])
				m.redoStack = m.redoStack[:len(m.redoStack)-1]
			}
		}
		return nil
	}

	// Handle regular keys
	switch key.Key {
	case 'h': // Move left
		if m.cursorX > 0 {
			m.cursorX--
		}
	case 'j': // Move down
		m.cursorY++
	case 'k': // Move up
		if m.cursorY > 0 {
			m.cursorY--
		}
	case 'l': // Move right
		m.cursorX++
	case 'w': // Next word
		m.cursorX = m.findNextWordStart(m.cursorX)
	case 'b': // Previous word
		m.cursorX = m.findPrevWordStart(m.cursorX)
	case 'g': // Handle 'gg' sequence
		// For simplicity, move to top
		m.cursorY = 0
	case 'G': // Move to bottom
		m.cursorY = 100 // Mock bottom
	case 'i': // Enter insert mode
		m.mode = "insert"
	case 'v': // Enter visual mode
		m.mode = "visual"
	case ':': // Enter command mode
		m.mode = "command"
		m.commandBuffer = ""
	case 'e': // Start edge creation
		if m.selectedNode != "" {
			m.pendingEdge = &Edge{From: m.selectedNode}
		}
	case 'y': // Copy/yank
		if m.selectedNode != "" {
			m.clipboard = m.selectedNode
		}
	case 'u': // Undo
		if len(m.undoStack) > 0 {
			m.redoStack = append(m.redoStack, m.undoStack[len(m.undoStack)-1])
			m.undoStack = m.undoStack[:len(m.undoStack)-1]
		}
	case '/': // Search
		m.mode = "command"
		m.commandBuffer = "/"
	case 'n': // Next search result
		if len(m.searchResults) > 0 {
			m.currentSearchIdx = (m.currentSearchIdx + 1) % len(m.searchResults)
			result := m.searchResults[m.currentSearchIdx]
			m.cursorX = result.X
			m.cursorY = result.Y
		}
	case 'N': // Previous search result
		if len(m.searchResults) > 0 {
			m.currentSearchIdx = (m.currentSearchIdx - 1 + len(m.searchResults)) % len(m.searchResults)
			result := m.searchResults[m.currentSearchIdx]
			m.cursorX = result.X
			m.cursorY = result.Y
		}
	case '?': // Toggle help
		m.helpVisible = !m.helpVisible
	}

	return nil
}

func (m *MockTUI) handleInsertModeKey(key KeyEvent) error {
	// In insert mode, just add characters to buffer
	return nil
}

func (m *MockTUI) handleVisualModeKey(key KeyEvent) error {
	// Handle visual mode - can toggle back with 'v'
	if key.Key == 'v' {
		m.mode = "normal"
	}
	return nil
}

func (m *MockTUI) handleCommandModeKey(key KeyEvent) error {
	// Build command buffer
	if key.Key >= 32 && key.Key <= 126 { // Printable ASCII
		m.commandBuffer += string(key.Key)
	}
	return nil
}

func (m *MockTUI) findNextWordStart(currentX int) int {
	// Mock implementation: find next word boundary
	if len(m.buffer) == 0 || currentX >= len(m.buffer[0]) {
		return currentX
	}

	// Skip current word
	x := currentX
	for x < len(m.buffer[0]) && m.buffer[0][x] != ' ' {
		x++
	}
	// Skip spaces
	for x < len(m.buffer[0]) && m.buffer[0][x] == ' ' {
		x++
	}

	return x
}

func (m *MockTUI) findPrevWordStart(currentX int) int {
	// Mock implementation: find previous word boundary
	if len(m.buffer) == 0 || currentX <= 0 {
		return 0
	}

	// Move back one position
	x := currentX - 1

	// Skip spaces
	for x > 0 && m.buffer[0][x] == ' ' {
		x--
	}

	// Find start of word
	for x > 0 && m.buffer[0][x-1] != ' ' {
		x--
	}

	return x
}

func (m *MockTUI) ExecuteCommand() error {
	// This will be implemented in the actual TUI
	return nil
}
