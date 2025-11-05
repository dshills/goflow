package tui_test

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

// TestEdgeCreation_EnterEdgeCreationMode tests that pressing 'e' enters edge creation mode
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_EnterEdgeCreationMode(t *testing.T) {
	tests := []struct {
		name           string
		initialState   TUIState
		keyPress       string
		expectedMode   string
		expectedStatus string
	}{
		{
			name: "enter edge creation mode from normal mode",
			initialState: TUIState{
				Mode:     "normal",
				Workflow: createTestWorkflow(t),
			},
			keyPress:       "e",
			expectedMode:   "edge_creation",
			expectedStatus: "Select source node for edge",
		},
		{
			name: "cannot enter edge creation with no nodes",
			initialState: TUIState{
				Mode:     "normal",
				Workflow: createEmptyWorkflow(t),
			},
			keyPress:       "e",
			expectedMode:   "normal",
			expectedStatus: "Error: Need at least 2 nodes to create edge",
		},
		{
			name: "cannot enter edge creation with only one node",
			initialState: TUIState{
				Mode:     "normal",
				Workflow: createWorkflowWithOneNode(t),
			},
			keyPress:       "e",
			expectedMode:   "normal",
			expectedStatus: "Error: Need at least 2 nodes to create edge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			tui := NewTestTUI(tt.initialState)
			tui.SendKey(tt.keyPress)

			if tui.GetMode() != tt.expectedMode {
				t.Errorf("Expected mode %q, got %q", tt.expectedMode, tui.GetMode())
			}

			if tui.GetStatusMessage() != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, tui.GetStatusMessage())
			}
		})
	}
}

// TestEdgeCreation_SelectSourceNode tests selecting the source node for an edge
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_SelectSourceNode(t *testing.T) {
	tests := []struct {
		name             string
		setupWorkflow    func(*testing.T) *workflow.Workflow
		nodeToSelect     string
		expectedSelected string
		expectedStatus   string
	}{
		{
			name:             "select valid source node",
			setupWorkflow:    createTestWorkflow,
			nodeToSelect:     "start",
			expectedSelected: "start",
			expectedStatus:   "Source: start | Select target node",
		},
		{
			name:             "select MCP tool node as source",
			setupWorkflow:    createTestWorkflow,
			nodeToSelect:     "tool-1",
			expectedSelected: "tool-1",
			expectedStatus:   "Source: tool-1 | Select target node",
		},
		{
			name:             "select condition node as source",
			setupWorkflow:    createWorkflowWithConditionNode,
			nodeToSelect:     "cond-1",
			expectedSelected: "cond-1",
			expectedStatus:   "Source: cond-1 | Select target node (conditional branch)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := tt.setupWorkflow(t)
			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			tui.SelectNode(tt.nodeToSelect)

			if tui.GetSelectedSourceNode() != tt.expectedSelected {
				t.Errorf("Expected source node %q, got %q", tt.expectedSelected, tui.GetSelectedSourceNode())
			}

			if tui.GetStatusMessage() != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, tui.GetStatusMessage())
			}

			// Verify visual highlighting
			if !tui.IsNodeHighlighted(tt.nodeToSelect) {
				t.Error("Source node should be highlighted")
			}
		})
	}
}

// TestEdgeCreation_SelectTargetNode tests selecting the target node to complete edge creation
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_SelectTargetNode(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func(*testing.T) *workflow.Workflow
		sourceNode    string
		targetNode    string
		expectSuccess bool
		expectedError string
		expectedEdges int
	}{
		{
			name:          "create valid edge from start to tool",
			setupWorkflow: createTestWorkflow,
			sourceNode:    "start",
			targetNode:    "tool-1",
			expectSuccess: true,
			expectedEdges: 1,
		},
		{
			name:          "create valid edge from tool to end",
			setupWorkflow: createTestWorkflow,
			sourceNode:    "tool-1",
			targetNode:    "end",
			expectSuccess: true,
			expectedEdges: 1,
		},
		{
			name:          "prevent self-loop edge",
			setupWorkflow: createTestWorkflow,
			sourceNode:    "tool-1",
			targetNode:    "tool-1",
			expectSuccess: false,
			expectedError: "Cannot create self-loop edge",
			expectedEdges: 0,
		},
		{
			name:          "prevent duplicate edge",
			setupWorkflow: createWorkflowWithExistingEdge,
			sourceNode:    "start",
			targetNode:    "tool-1",
			expectSuccess: false,
			expectedError: "Edge already exists from start to tool-1",
			expectedEdges: 1, // existing edge count
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := tt.setupWorkflow(t)
			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			// Select source node
			tui.SelectNode(tt.sourceNode)

			// Select target node
			result := tui.SelectNode(tt.targetNode)

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected edge creation to succeed, got error: %v", result.Error)
				}

				// Verify edge was added to workflow
				if len(wf.Edges) != tt.expectedEdges {
					t.Errorf("Expected %d edges, got %d", tt.expectedEdges, len(wf.Edges))
				}

				// Verify edge properties
				edge := wf.Edges[len(wf.Edges)-1]
				if edge.FromNodeID != tt.sourceNode {
					t.Errorf("Expected edge from %q, got %q", tt.sourceNode, edge.FromNodeID)
				}
				if edge.ToNodeID != tt.targetNode {
					t.Errorf("Expected edge to %q, got %q", tt.targetNode, edge.ToNodeID)
				}

				// Verify mode returned to normal
				if tui.GetMode() != "normal" {
					t.Errorf("Expected mode to return to 'normal', got %q", tui.GetMode())
				}

				// Verify edge is visible in TUI
				if !tui.IsEdgeDisplayed(edge.ID) {
					t.Error("Created edge should be visible in TUI")
				}
			} else {
				if result.Success {
					t.Error("Expected edge creation to fail")
				}

				if result.Error != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, result.Error)
				}

				// Verify edge count unchanged
				if len(wf.Edges) != tt.expectedEdges {
					t.Errorf("Expected %d edges (unchanged), got %d", tt.expectedEdges, len(wf.Edges))
				}

				// Verify mode returned to normal
				if tui.GetMode() != "normal" {
					t.Errorf("Expected mode to return to 'normal' after error, got %q", tui.GetMode())
				}
			}
		})
	}
}

// TestEdgeCreation_ConditionalEdges tests creating edges from ConditionNode with true/false branches
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_ConditionalEdges(t *testing.T) {
	tests := []struct {
		name          string
		sourceNode    string
		targetNode    string
		branchLabel   string
		expectedCond  string
		existingEdges int // how many edges already exist from this condition node
		expectSuccess bool
		expectedError string
	}{
		{
			name:          "create first conditional edge (true branch)",
			sourceNode:    "cond-1",
			targetNode:    "tool-true",
			branchLabel:   "true",
			expectedCond:  "${result == true}",
			existingEdges: 0,
			expectSuccess: true,
		},
		{
			name:          "create second conditional edge (false branch)",
			sourceNode:    "cond-1",
			targetNode:    "tool-false",
			branchLabel:   "false",
			expectedCond:  "${result == false}",
			existingEdges: 1,
			expectSuccess: true,
		},
		{
			name:          "prevent third edge from condition node",
			sourceNode:    "cond-1",
			targetNode:    "tool-extra",
			branchLabel:   "extra",
			existingEdges: 2,
			expectSuccess: false,
			expectedError: "Condition nodes can only have 2 outgoing edges",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createWorkflowWithConditionNode(t)

			// Add existing edges if specified
			if tt.existingEdges >= 1 {
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "cond-1",
					ToNodeID:   "tool-true",
					Condition:  "${result == true}",
					Label:      "true",
				})
			}
			if tt.existingEdges >= 2 {
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "cond-1",
					ToNodeID:   "tool-false",
					Condition:  "${result == false}",
					Label:      "false",
				})
			}

			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			// Select source (condition node)
			tui.SelectNode(tt.sourceNode)

			// Should prompt for branch type
			if tui.GetStatusMessage() != "Source: cond-1 | Select target node (conditional branch)" {
				t.Errorf("Expected conditional branch prompt, got %q", tui.GetStatusMessage())
			}

			// Select target node
			result := tui.SelectNode(tt.targetNode)

			if tt.expectSuccess {
				// Should prompt for branch label (true/false)
				if !tui.IsPromptingForBranchLabel() {
					t.Error("Should prompt for branch label (true/false)")
				}

				// Select branch label
				tui.SelectBranchLabel(tt.branchLabel)

				// Verify edge was created with correct condition
				edge := findEdge(wf, tt.sourceNode, tt.targetNode)
				if edge == nil {
					t.Fatal("Edge should be created")
				}

				if edge.Condition != tt.expectedCond {
					t.Errorf("Expected condition %q, got %q", tt.expectedCond, edge.Condition)
				}

				if edge.Label != tt.branchLabel {
					t.Errorf("Expected label %q, got %q", tt.branchLabel, edge.Label)
				}
			} else {
				if result.Success {
					t.Error("Expected edge creation to fail")
				}

				if result.Error != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, result.Error)
				}
			}
		})
	}
}

// TestEdgeCreation_VisualFeedback tests that the TUI provides visual feedback during edge creation
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_VisualFeedback(t *testing.T) {
	tests := []struct {
		name                  string
		state                 string
		sourceNode            string
		targetNode            string
		expectedHighlights    []string
		expectedConnectionPrw bool // connection preview
		expectedStatusMsg     string
	}{
		{
			name:               "source node highlighted after selection",
			state:              "source_selected",
			sourceNode:         "start",
			expectedHighlights: []string{"start"},
			expectedStatusMsg:  "Source: start | Select target node",
		},
		{
			name:                  "connection preview shown when hovering target",
			state:                 "hovering_target",
			sourceNode:            "start",
			targetNode:            "tool-1",
			expectedHighlights:    []string{"start", "tool-1"},
			expectedConnectionPrw: true,
			expectedStatusMsg:     "Press Enter to create edge from start to tool-1",
		},
		{
			name:               "invalid target shows error highlight",
			state:              "hovering_invalid",
			sourceNode:         "tool-1",
			targetNode:         "tool-1",
			expectedHighlights: []string{"tool-1"},
			expectedStatusMsg:  "Error: Cannot create self-loop edge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createTestWorkflow(t)
			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			// Setup test state
			if tt.sourceNode != "" {
				tui.SelectNode(tt.sourceNode)
			}

			if tt.targetNode != "" && tt.state == "hovering_target" {
				tui.HoverNode(tt.targetNode)
			}

			// Verify highlighted nodes
			for _, nodeID := range tt.expectedHighlights {
				if !tui.IsNodeHighlighted(nodeID) {
					t.Errorf("Expected node %q to be highlighted", nodeID)
				}
			}

			// Verify connection preview
			if tt.expectedConnectionPrw {
				if !tui.IsConnectionPreviewVisible() {
					t.Error("Expected connection preview to be visible")
				}

				preview := tui.GetConnectionPreview()
				if preview.From != tt.sourceNode {
					t.Errorf("Expected preview from %q, got %q", tt.sourceNode, preview.From)
				}
				if preview.To != tt.targetNode {
					t.Errorf("Expected preview to %q, got %q", tt.targetNode, preview.To)
				}
			}

			// Verify status message
			if tui.GetStatusMessage() != tt.expectedStatusMsg {
				t.Errorf("Expected status %q, got %q", tt.expectedStatusMsg, tui.GetStatusMessage())
			}
		})
	}
}

// TestEdgeCreation_CancelOperation tests canceling edge creation with Esc key
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_CancelOperation(t *testing.T) {
	tests := []struct {
		name          string
		stage         string // "no_selection", "source_selected"
		sourceNode    string
		expectedMode  string
		expectedEdges int
	}{
		{
			name:          "cancel before selecting source",
			stage:         "no_selection",
			expectedMode:  "normal",
			expectedEdges: 0,
		},
		{
			name:          "cancel after selecting source",
			stage:         "source_selected",
			sourceNode:    "start",
			expectedMode:  "normal",
			expectedEdges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createTestWorkflow(t)
			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			if tt.sourceNode != "" {
				tui.SelectNode(tt.sourceNode)
			}

			// Press Esc to cancel
			tui.SendKey("Esc")

			// Verify mode returned to normal
			if tui.GetMode() != tt.expectedMode {
				t.Errorf("Expected mode %q, got %q", tt.expectedMode, tui.GetMode())
			}

			// Verify no edge was created
			if len(wf.Edges) != tt.expectedEdges {
				t.Errorf("Expected %d edges, got %d", tt.expectedEdges, len(wf.Edges))
			}

			// Verify no nodes are highlighted
			highlightedNodes := tui.GetHighlightedNodes()
			if len(highlightedNodes) > 0 {
				t.Errorf("Expected no highlighted nodes after cancel, got %v", highlightedNodes)
			}

			// Verify status cleared
			if tui.GetStatusMessage() != "" {
				t.Errorf("Expected empty status message, got %q", tui.GetStatusMessage())
			}
		})
	}
}

// TestEdgeCreation_CycleDetection tests that creating edges that would form cycles is prevented
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_CycleDetection(t *testing.T) {
	tests := []struct {
		name          string
		setupEdges    func(*workflow.Workflow)
		sourceNode    string
		targetNode    string
		expectSuccess bool
		expectedError string
	}{
		{
			name: "simple two-node cycle prevention",
			setupEdges: func(wf *workflow.Workflow) {
				// Create edge: tool-1 -> tool-2
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "tool-1",
					ToNodeID:   "tool-2",
				})
			},
			sourceNode:    "tool-2",
			targetNode:    "tool-1",
			expectSuccess: false,
			expectedError: "Would create circular dependency",
		},
		{
			name: "three-node cycle prevention",
			setupEdges: func(wf *workflow.Workflow) {
				// Create chain: tool-1 -> tool-2 -> tool-3
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "tool-1",
					ToNodeID:   "tool-2",
				})
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "tool-2",
					ToNodeID:   "tool-3",
				})
			},
			sourceNode:    "tool-3",
			targetNode:    "tool-1",
			expectSuccess: false,
			expectedError: "Would create circular dependency",
		},
		{
			name: "diamond pattern should succeed",
			setupEdges: func(wf *workflow.Workflow) {
				// Create diamond: start -> tool-1 -> tool-3
				//                      -> tool-2 -> tool-3
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "start",
					ToNodeID:   "tool-1",
				})
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "start",
					ToNodeID:   "tool-2",
				})
				wf.AddEdge(&workflow.Edge{
					FromNodeID: "tool-1",
					ToNodeID:   "tool-3",
				})
			},
			sourceNode:    "tool-2",
			targetNode:    "tool-3",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createComplexTestWorkflow(t)
			tt.setupEdges(wf)

			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			// Select source
			tui.SelectNode(tt.sourceNode)

			// Try to select target
			result := tui.SelectNode(tt.targetNode)

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected edge creation to succeed, got error: %v", result.Error)
				}
			} else {
				if result.Success {
					t.Error("Expected edge creation to fail with cycle detection")
				}

				if result.Error != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, result.Error)
				}
			}
		})
	}
}

// TestEdgeCreation_EdgeDeletion tests selecting and deleting an edge
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_EdgeDeletion(t *testing.T) {
	tests := []struct {
		name          string
		setupEdges    func(*workflow.Workflow)
		edgeToSelect  string
		expectedEdges int
	}{
		{
			name: "delete selected edge",
			setupEdges: func(wf *workflow.Workflow) {
				wf.AddEdge(&workflow.Edge{
					ID:         "edge-1",
					FromNodeID: "start",
					ToNodeID:   "tool-1",
				})
				wf.AddEdge(&workflow.Edge{
					ID:         "edge-2",
					FromNodeID: "tool-1",
					ToNodeID:   "end",
				})
			},
			edgeToSelect:  "edge-1",
			expectedEdges: 1,
		},
		{
			name: "delete last remaining edge",
			setupEdges: func(wf *workflow.Workflow) {
				wf.AddEdge(&workflow.Edge{
					ID:         "edge-1",
					FromNodeID: "start",
					ToNodeID:   "end",
				})
			},
			edgeToSelect:  "edge-1",
			expectedEdges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createTestWorkflow(t)
			tt.setupEdges(wf)

			initialEdgeCount := len(wf.Edges)
			tui := NewTestTUI(TUIState{Mode: "normal", Workflow: wf})

			// Click on edge to select it
			tui.SelectEdge(tt.edgeToSelect)

			// Verify edge is highlighted/selected
			if !tui.IsEdgeSelected(tt.edgeToSelect) {
				t.Error("Edge should be selected after clicking")
			}

			// Press Delete key
			tui.SendKey("Delete")

			// Verify edge was removed
			if len(wf.Edges) != tt.expectedEdges {
				t.Errorf("Expected %d edges after deletion, got %d", tt.expectedEdges, len(wf.Edges))
			}

			// Verify edge no longer displayed
			if tui.IsEdgeDisplayed(tt.edgeToSelect) {
				t.Error("Deleted edge should not be displayed")
			}

			// Verify status message
			expectedStatus := "Deleted edge from " + getEdgeFromNode(tt.edgeToSelect, initialEdgeCount) + " to " + getEdgeToNode(tt.edgeToSelect, initialEdgeCount)
			if tui.GetStatusMessage() != expectedStatus {
				t.Errorf("Expected status %q, got %q", expectedStatus, tui.GetStatusMessage())
			}
		})
	}
}

// TestEdgeCreation_EdgeLabels tests that edge labels are displayed properly
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_EdgeLabels(t *testing.T) {
	tests := []struct {
		name          string
		edge          *workflow.Edge
		expectedLabel string
	}{
		{
			name: "simple edge with no label",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "start",
				ToNodeID:   "tool-1",
			},
			expectedLabel: "",
		},
		{
			name: "edge with custom label",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "tool-1",
				ToNodeID:   "tool-2",
				Label:      "on success",
			},
			expectedLabel: "on success",
		},
		{
			name: "conditional edge with true branch",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "cond-1",
				ToNodeID:   "tool-true",
				Condition:  "${result == true}",
				Label:      "true",
			},
			expectedLabel: "true",
		},
		{
			name: "conditional edge with false branch",
			edge: &workflow.Edge{
				ID:         "edge-1",
				FromNodeID: "cond-1",
				ToNodeID:   "tool-false",
				Condition:  "${result == false}",
				Label:      "false",
			},
			expectedLabel: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createTestWorkflow(t)
			wf.AddEdge(tt.edge)

			tui := NewTestTUI(TUIState{Mode: "normal", Workflow: wf})

			// Verify edge is displayed
			if !tui.IsEdgeDisplayed(tt.edge.ID) {
				t.Error("Edge should be displayed")
			}

			// Verify label is shown correctly
			displayedLabel := tui.GetEdgeLabel(tt.edge.ID)
			if displayedLabel != tt.expectedLabel {
				t.Errorf("Expected label %q, got %q", tt.expectedLabel, displayedLabel)
			}

			// If edge has label, it should be visible near the edge
			if tt.expectedLabel != "" {
				if !tui.IsEdgeLabelVisible(tt.edge.ID) {
					t.Error("Edge label should be visible in TUI")
				}
			}
		})
	}
}

// TestEdgeCreation_MultipleEdgeTypes tests creating edges between different node type combinations
// EXPECTED FAILURE: TUI not yet implemented
func TestEdgeCreation_MultipleEdgeTypes(t *testing.T) {
	tests := []struct {
		name           string
		sourceNodeType string
		targetNodeType string
		expectSuccess  bool
		expectedError  string
	}{
		{
			name:           "start to mcp_tool",
			sourceNodeType: "start",
			targetNodeType: "mcp_tool",
			expectSuccess:  true,
		},
		{
			name:           "mcp_tool to transform",
			sourceNodeType: "mcp_tool",
			targetNodeType: "transform",
			expectSuccess:  true,
		},
		{
			name:           "transform to condition",
			sourceNodeType: "transform",
			targetNodeType: "condition",
			expectSuccess:  true,
		},
		{
			name:           "condition to mcp_tool (via branch)",
			sourceNodeType: "condition",
			targetNodeType: "mcp_tool",
			expectSuccess:  true,
		},
		{
			name:           "mcp_tool to end",
			sourceNodeType: "mcp_tool",
			targetNodeType: "end",
			expectSuccess:  true,
		},
		{
			name:           "transform to end",
			sourceNodeType: "transform",
			targetNodeType: "end",
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This test will fail until TUI is implemented
			t.Skip("TUI not yet implemented - TEST-FIRST approach")

			wf := createWorkflowWithMultipleNodeTypes(t)
			tui := NewTestTUI(TUIState{Mode: "edge_creation", Workflow: wf})

			// Find nodes of the specified types
			sourceNode := findNodeByType(wf, tt.sourceNodeType)
			targetNode := findNodeByType(wf, tt.targetNodeType)

			if sourceNode == nil {
				t.Fatalf("Could not find source node of type %q", tt.sourceNodeType)
			}
			if targetNode == nil {
				t.Fatalf("Could not find target node of type %q", tt.targetNodeType)
			}

			// Select source
			tui.SelectNode(sourceNode.GetID())

			// Select target
			result := tui.SelectNode(targetNode.GetID())

			// Handle conditional edges differently
			if tt.sourceNodeType == "condition" {
				if !tui.IsPromptingForBranchLabel() {
					t.Error("Should prompt for branch label when source is condition node")
				}
				tui.SelectBranchLabel("true")
			}

			if tt.expectSuccess {
				if !result.Success {
					t.Errorf("Expected edge creation to succeed, got error: %v", result.Error)
				}

				// Verify edge exists
				edge := findEdge(wf, sourceNode.GetID(), targetNode.GetID())
				if edge == nil {
					t.Error("Edge should exist in workflow")
				}
			} else {
				if result.Success {
					t.Error("Expected edge creation to fail")
				}

				if result.Error != tt.expectedError {
					t.Errorf("Expected error %q, got %q", tt.expectedError, result.Error)
				}
			}
		})
	}
}

// Helper types and functions for TUI testing (will be implemented with TUI)

// TUIState represents the state of the TUI for testing
type TUIState struct {
	Mode     string
	Workflow *workflow.Workflow
}

// TestTUI is a mock TUI for testing (to be replaced with actual implementation)
type TestTUI struct {
	state TUIState
}

// EdgeOperationResult represents the result of an edge operation
type EdgeOperationResult struct {
	Success bool
	Error   string
}

// ConnectionPreview represents a preview of an edge being created
type ConnectionPreview struct {
	From string
	To   string
}

// NewTestTUI creates a new test TUI instance
func NewTestTUI(state TUIState) *TestTUI {
	return &TestTUI{state: state}
}

// Mock methods (to be implemented when TUI is built)
func (t *TestTUI) SendKey(key string) {}
func (t *TestTUI) GetMode() string {
	return ""
}
func (t *TestTUI) GetStatusMessage() string {
	return ""
}
func (t *TestTUI) SelectNode(nodeID string) EdgeOperationResult {
	return EdgeOperationResult{}
}
func (t *TestTUI) GetSelectedSourceNode() string {
	return ""
}
func (t *TestTUI) IsNodeHighlighted(nodeID string) bool {
	return false
}
func (t *TestTUI) HoverNode(nodeID string) {}
func (t *TestTUI) IsConnectionPreviewVisible() bool {
	return false
}
func (t *TestTUI) GetConnectionPreview() ConnectionPreview {
	return ConnectionPreview{}
}
func (t *TestTUI) GetHighlightedNodes() []string {
	return nil
}
func (t *TestTUI) IsPromptingForBranchLabel() bool {
	return false
}
func (t *TestTUI) SelectBranchLabel(label string) {}
func (t *TestTUI) IsEdgeDisplayed(edgeID string) bool {
	return false
}
func (t *TestTUI) SelectEdge(edgeID string) {}
func (t *TestTUI) IsEdgeSelected(edgeID string) bool {
	return false
}
func (t *TestTUI) GetEdgeLabel(edgeID string) string {
	return ""
}
func (t *TestTUI) IsEdgeLabelVisible(edgeID string) bool {
	return false
}

// Helper functions for creating test workflows

func createEmptyWorkflow(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-empty", "Empty workflow for testing")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}
	return wf
}

func createWorkflowWithOneNode(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-one-node", "Workflow with one node")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}
	wf.AddNode(&workflow.StartNode{ID: "start"})
	return wf
}

func createTestWorkflow(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-workflow", "Test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-1",
		ServerID:       "test-server",
		ToolName:       "test-tool",
		OutputVariable: "result",
	})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	return wf
}

func createWorkflowWithConditionNode(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-condition", "Workflow with condition")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.ConditionNode{
		ID:        "cond-1",
		Condition: "${count > 10}",
	})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-true",
		ServerID:       "test-server",
		ToolName:       "on-true",
		OutputVariable: "true_result",
	})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-false",
		ServerID:       "test-server",
		ToolName:       "on-false",
		OutputVariable: "false_result",
	})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	return wf
}

func createWorkflowWithExistingEdge(t *testing.T) *workflow.Workflow {
	wf := createTestWorkflow(t)

	// Add existing edge
	wf.AddEdge(&workflow.Edge{
		ID:         "existing-edge",
		FromNodeID: "start",
		ToNodeID:   "tool-1",
	})

	return wf
}

func createComplexTestWorkflow(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-complex", "Complex workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-1",
		ServerID:       "test-server",
		ToolName:       "test-tool-1",
		OutputVariable: "result1",
	})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-2",
		ServerID:       "test-server",
		ToolName:       "test-tool-2",
		OutputVariable: "result2",
	})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-3",
		ServerID:       "test-server",
		ToolName:       "test-tool-3",
		OutputVariable: "result3",
	})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	return wf
}

func createWorkflowWithMultipleNodeTypes(t *testing.T) *workflow.Workflow {
	wf, err := workflow.NewWorkflow("test-multi-types", "Workflow with multiple node types")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	wf.AddNode(&workflow.StartNode{ID: "start"})
	wf.AddNode(&workflow.MCPToolNode{
		ID:             "tool-1",
		ServerID:       "test-server",
		ToolName:       "test-tool",
		OutputVariable: "result",
	})
	wf.AddNode(&workflow.TransformNode{
		ID:             "transform-1",
		InputVariable:  "result",
		Expression:     "$.data",
		OutputVariable: "transformed",
	})
	wf.AddNode(&workflow.ConditionNode{
		ID:        "cond-1",
		Condition: "${transformed.count > 0}",
	})
	wf.AddNode(&workflow.EndNode{ID: "end"})

	return wf
}

func findNodeByType(wf *workflow.Workflow, nodeType string) workflow.Node {
	for _, node := range wf.Nodes {
		if node.Type() == nodeType {
			return node
		}
	}
	return nil
}

func findEdge(wf *workflow.Workflow, fromNodeID, toNodeID string) *workflow.Edge {
	for _, edge := range wf.Edges {
		if edge.FromNodeID == fromNodeID && edge.ToNodeID == toNodeID {
			return edge
		}
	}
	return nil
}

func getEdgeFromNode(edgeID string, initialCount int) string {
	// Placeholder - will be implemented with actual TUI
	return "node"
}

func getEdgeToNode(edgeID string, initialCount int) string {
	// Placeholder - will be implemented with actual TUI
	return "node"
}
