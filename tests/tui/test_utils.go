package tui

import (
	"strings"
	"testing"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
)

// KeyboardEventSimulator simulates keyboard input for TUI testing
type KeyboardEventSimulator struct {
	events []string
	index  int
}

// NewKeyboardEventSimulator creates a new keyboard event simulator
func NewKeyboardEventSimulator(events []string) *KeyboardEventSimulator {
	return &KeyboardEventSimulator{
		events: events,
		index:  0,
	}
}

// NextKey returns the next keyboard event
// Returns empty string if no more events
func (k *KeyboardEventSimulator) NextKey() string {
	if k.index >= len(k.events) {
		return ""
	}
	key := k.events[k.index]
	k.index++
	return key
}

// HasMore returns true if there are more events to process
func (k *KeyboardEventSimulator) HasMore() bool {
	return k.index < len(k.events)
}

// Reset resets the simulator to the beginning
func (k *KeyboardEventSimulator) Reset() {
	k.index = 0
}

// RemainingCount returns the number of remaining events
func (k *KeyboardEventSimulator) RemainingCount() int {
	return len(k.events) - k.index
}

// ScreenCapture captures rendered output for testing
type ScreenCapture struct {
	lines  []string
	width  int
	height int
}

// NewScreenCapture creates a new screen capture utility
func NewScreenCapture(width, height int) *ScreenCapture {
	return &ScreenCapture{
		lines:  make([]string, 0),
		width:  width,
		height: height,
	}
}

// CaptureCanvas captures the rendered canvas output
func (s *ScreenCapture) CaptureCanvas(canvas *tui.Canvas) {
	s.lines = make([]string, 0)

	// Simple capture implementation that stores canvas state
	// In a real implementation, this would render the canvas to a buffer
	// For now, we just capture basic state
	s.lines = append(s.lines, "Canvas rendered")
}

// GetLine returns a specific line from the capture
func (s *ScreenCapture) GetLine(lineNum int) string {
	if lineNum < 0 || lineNum >= len(s.lines) {
		return ""
	}
	return s.lines[lineNum]
}

// GetLines returns all captured lines
func (s *ScreenCapture) GetLines() []string {
	return s.lines
}

// Contains checks if the capture contains a specific string
func (s *ScreenCapture) Contains(text string) bool {
	for _, line := range s.lines {
		if strings.Contains(line, text) {
			return true
		}
	}
	return false
}

// Clear clears the screen capture
func (s *ScreenCapture) Clear() {
	s.lines = make([]string, 0)
}

// WorkflowBuilderTestHarness provides utilities for testing WorkflowBuilder
type WorkflowBuilderTestHarness struct {
	builder    *tui.WorkflowBuilder
	workflow   *workflow.Workflow
	repository *MockRepository
	keyboard   *KeyboardEventSimulator
	screen     *ScreenCapture
	t          *testing.T
}

// NewWorkflowBuilderTestHarness creates a new test harness
func NewWorkflowBuilderTestHarness(t *testing.T, wf *workflow.Workflow) *WorkflowBuilderTestHarness {
	builder, err := tui.NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("failed to create workflow builder: %v", err)
	}

	repo := NewMockRepository()
	builder.SetRepository(repo)

	return &WorkflowBuilderTestHarness{
		builder:    builder,
		workflow:   wf,
		repository: repo,
		keyboard:   NewKeyboardEventSimulator([]string{}),
		screen:     NewScreenCapture(80, 24),
		t:          t,
	}
}

// SimulateKeySequence simulates a sequence of keyboard events
func (h *WorkflowBuilderTestHarness) SimulateKeySequence(keys []string) error {
	h.keyboard = NewKeyboardEventSimulator(keys)

	for h.keyboard.HasMore() {
		key := h.keyboard.NextKey()
		if err := h.builder.HandleKey(key); err != nil {
			return err
		}
	}

	return nil
}

// AssertNodeSelected asserts that a specific node is selected
func (h *WorkflowBuilderTestHarness) AssertNodeSelected(nodeID string) {
	selectedID := h.builder.GetSelectedNodeID()
	if selectedID != nodeID {
		h.t.Errorf("expected node %s to be selected, got %s", nodeID, selectedID)
	}
}

// AssertNodeCount asserts the number of nodes in the workflow
func (h *WorkflowBuilderTestHarness) AssertNodeCount(expected int) {
	actual := len(h.workflow.Nodes)
	if actual != expected {
		h.t.Errorf("expected %d nodes, got %d", expected, actual)
	}
}

// AssertEdgeCount asserts the number of edges in the workflow
func (h *WorkflowBuilderTestHarness) AssertEdgeCount(expected int) {
	actual := len(h.workflow.Edges)
	if actual != expected {
		h.t.Errorf("expected %d edges, got %d", expected, actual)
	}
}

// AssertModified asserts whether the workflow is marked as modified
func (h *WorkflowBuilderTestHarness) AssertModified(expected bool) {
	actual := h.builder.IsModified()
	if actual != expected {
		h.t.Errorf("expected modified=%v, got modified=%v", expected, actual)
	}
}

// AssertMode asserts the current builder mode
func (h *WorkflowBuilderTestHarness) AssertMode(expectedMode string) {
	actualMode := h.builder.Mode()
	if actualMode != expectedMode {
		h.t.Errorf("expected mode %s, got %s", expectedMode, actualMode)
	}
}

// AssertValidationErrors asserts the number of validation errors
func (h *WorkflowBuilderTestHarness) AssertValidationErrors(expected int) {
	status := h.builder.GetValidationStatus()
	actual := len(status.Errors)
	if actual != expected {
		h.t.Errorf("expected %d validation errors, got %d", expected, actual)
	}
}

// AssertValidationValid asserts whether the workflow is valid
func (h *WorkflowBuilderTestHarness) AssertValidationValid(expected bool) {
	status := h.builder.GetValidationStatus()
	if status.IsValid != expected {
		h.t.Errorf("expected validation valid=%v, got valid=%v", expected, status.IsValid)
	}
}

// AssertCanUndo asserts whether undo is available
func (h *WorkflowBuilderTestHarness) AssertCanUndo(expected bool) {
	actual := h.builder.CanUndo()
	if actual != expected {
		h.t.Errorf("expected CanUndo=%v, got %v", expected, actual)
	}
}

// AssertCanRedo asserts whether redo is available
func (h *WorkflowBuilderTestHarness) AssertCanRedo(expected bool) {
	actual := h.builder.CanRedo()
	if actual != expected {
		h.t.Errorf("expected CanRedo=%v, got %v", expected, actual)
	}
}

// GetBuilder returns the workflow builder instance
func (h *WorkflowBuilderTestHarness) GetBuilder() *tui.WorkflowBuilder {
	return h.builder
}

// GetWorkflow returns the workflow instance
func (h *WorkflowBuilderTestHarness) GetWorkflow() *workflow.Workflow {
	return h.workflow
}

// GetRepository returns the mock repository
func (h *WorkflowBuilderTestHarness) GetRepository() *MockRepository {
	return h.repository
}

// AddNode is a helper to add a node to the workflow
func (h *WorkflowBuilderTestHarness) AddNode(node workflow.Node) error {
	return h.builder.AddNodeToCanvas(node)
}

// CreateEdge is a helper to create an edge
func (h *WorkflowBuilderTestHarness) CreateEdge(fromID, toID string) error {
	return h.builder.CreateEdge(fromID, toID)
}

// SelectNode is a helper to select a node
func (h *WorkflowBuilderTestHarness) SelectNode(nodeID string) error {
	return h.builder.SelectNode(nodeID)
}

// SaveWorkflow is a helper to save the workflow
func (h *WorkflowBuilderTestHarness) SaveWorkflow() error {
	return h.builder.SaveWorkflow()
}
