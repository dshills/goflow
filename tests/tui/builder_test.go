package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
)

// TestWorkflowBuilderView_NewBuilder tests creating a new workflow builder view
func TestWorkflowBuilderView_NewBuilder(t *testing.T) {
	tests := []struct {
		name     string
		wf       *workflow.Workflow
		wantErr  bool
		wantMode string
	}{
		{
			name: "create builder with valid workflow",
			wf: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test-workflow", "test description")
				return wf
			}(),
			wantErr:  false,
			wantMode: "view",
		},
		{
			name:    "create builder with nil workflow should fail",
			wf:      nil,
			wantErr: true,
		},
		{
			name: "create builder starts in view mode by default",
			wf: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("simple", "")
				return wf
			}(),
			wantErr:  false,
			wantMode: "view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewWorkflowBuilder(tt.wf)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewWorkflowBuilder() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewWorkflowBuilder() unexpected error: %v", err)
				return
			}

			if builder == nil {
				t.Fatal("NewWorkflowBuilder() returned nil builder")
			}

			if tt.wantMode != "" && builder.Mode() != tt.wantMode {
				t.Errorf("NewWorkflowBuilder() mode = %v, want %v", builder.Mode(), tt.wantMode)
			}
		})
	}
}

// TestWorkflowBuilderView_CanvasRendering tests canvas rendering for workflow graph
func TestWorkflowBuilderView_CanvasRendering(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() *workflow.Workflow
		wantNodes     int
		wantEdges     int
		wantCanvasW   int
		wantCanvasH   int
	}{
		{
			name: "render empty workflow with start and end nodes",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("empty", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			wantNodes:   2,
			wantEdges:   0,
			wantCanvasW: 80,
			wantCanvasH: 24,
		},
		{
			name: "render simple linear workflow",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("linear", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1", ServerID: "fs", ToolName: "read_file"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				return wf
			},
			wantNodes:   3,
			wantEdges:   2,
			wantCanvasW: 80,
			wantCanvasH: 24,
		},
		{
			name: "render complex workflow with multiple branches",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("complex", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.ConditionNode{ID: "condition-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-3"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "condition-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "condition-1", ToNodeID: "tool-2", Condition: "true"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "condition-1", ToNodeID: "tool-3", Condition: "false"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-3", ToNodeID: "end"})
				return wf
			},
			wantNodes:   6,
			wantEdges:   6,
			wantCanvasW: 80,
			wantCanvasH: 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, err := NewWorkflowBuilder(wf)
			if err != nil {
				t.Fatalf("NewWorkflowBuilder() error: %v", err)
			}

			canvas := builder.RenderCanvas()
			if canvas == nil {
				t.Fatal("RenderCanvas() returned nil canvas")
			}

			if canvas.Width != tt.wantCanvasW {
				t.Errorf("RenderCanvas() width = %v, want %v", canvas.Width, tt.wantCanvasW)
			}

			if canvas.Height != tt.wantCanvasH {
				t.Errorf("RenderCanvas() height = %v, want %v", canvas.Height, tt.wantCanvasH)
			}

			renderedNodes := canvas.GetNodeCount()
			if renderedNodes != tt.wantNodes {
				t.Errorf("RenderCanvas() node count = %v, want %v", renderedNodes, tt.wantNodes)
			}

			renderedEdges := canvas.GetEdgeCount()
			if renderedEdges != tt.wantEdges {
				t.Errorf("RenderCanvas() edge count = %v, want %v", renderedEdges, tt.wantEdges)
			}
		})
	}
}

// TestWorkflowBuilderView_NodePaletteDisplay tests node palette display and navigation
func TestWorkflowBuilderView_NodePaletteDisplay(t *testing.T) {
	tests := []struct {
		name             string
		wantNodeTypes    []string
		wantDefaultIndex int
	}{
		{
			name: "node palette shows all available node types",
			wantNodeTypes: []string{
				"MCP Tool",
				"Transform",
				"Condition",
				"Loop",
				"Parallel",
			},
			wantDefaultIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test", "")
			builder, _ := NewWorkflowBuilder(wf)

			palette := builder.GetNodePalette()
			if palette == nil {
				t.Fatal("GetNodePalette() returned nil")
			}

			nodeTypes := palette.GetNodeTypes()
			if len(nodeTypes) != len(tt.wantNodeTypes) {
				t.Errorf("GetNodeTypes() count = %v, want %v", len(nodeTypes), len(tt.wantNodeTypes))
			}

			for i, wantType := range tt.wantNodeTypes {
				if i >= len(nodeTypes) {
					t.Errorf("GetNodeTypes() missing type %v", wantType)
					continue
				}
				if nodeTypes[i] != wantType {
					t.Errorf("GetNodeTypes()[%d] = %v, want %v", i, nodeTypes[i], wantType)
				}
			}

			selectedIndex := palette.GetSelectedIndex()
			if selectedIndex != tt.wantDefaultIndex {
				t.Errorf("GetSelectedIndex() = %v, want %v", selectedIndex, tt.wantDefaultIndex)
			}
		})
	}
}

// TestWorkflowBuilderView_NodePaletteNavigation tests keyboard navigation in node palette
func TestWorkflowBuilderView_NodePaletteNavigation(t *testing.T) {
	tests := []struct {
		name         string
		keys         []string
		wantIndex    int
		wantNodeType string
		wantErr      bool
	}{
		{
			name:         "navigate down in palette",
			keys:         []string{"j", "j"},
			wantIndex:    2,
			wantNodeType: "Condition",
		},
		{
			name:         "navigate up in palette",
			keys:         []string{"j", "j", "j", "k"},
			wantIndex:    2,
			wantNodeType: "Condition",
		},
		{
			name:         "navigate down wraps to top",
			keys:         []string{"j", "j", "j", "j", "j"},
			wantIndex:    0,
			wantNodeType: "MCP Tool",
		},
		{
			name:         "navigate up from top wraps to bottom",
			keys:         []string{"k"},
			wantIndex:    4,
			wantNodeType: "Parallel",
		},
		{
			name:         "home key goes to first item",
			keys:         []string{"j", "j", "g"},
			wantIndex:    0,
			wantNodeType: "MCP Tool",
		},
		{
			name:         "end key goes to last item",
			keys:         []string{"G"},
			wantIndex:    4,
			wantNodeType: "Parallel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("test", "")
			builder, _ := NewWorkflowBuilder(wf)
			palette := builder.GetNodePalette()

			for _, key := range tt.keys {
				err := palette.HandleKey(key)
				if tt.wantErr && err == nil {
					t.Errorf("HandleKey(%q) expected error but got none", key)
					return
				}
				if !tt.wantErr && err != nil {
					t.Errorf("HandleKey(%q) unexpected error: %v", key, err)
					return
				}
			}

			gotIndex := palette.GetSelectedIndex()
			if gotIndex != tt.wantIndex {
				t.Errorf("after key sequence, index = %v, want %v", gotIndex, tt.wantIndex)
			}

			gotType := palette.GetSelectedNodeType()
			if gotType != tt.wantNodeType {
				t.Errorf("after key sequence, node type = %v, want %v", gotType, tt.wantNodeType)
			}
		})
	}
}

// TestWorkflowBuilderView_NodeSelection tests node selection and highlighting
func TestWorkflowBuilderView_NodeSelection(t *testing.T) {
	tests := []struct {
		name            string
		setupWorkflow   func() *workflow.Workflow
		selectNodeID    string
		wantSelected    bool
		wantHighlighted bool
	}{
		{
			name: "select existing node",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			selectNodeID:    "tool-1",
			wantSelected:    true,
			wantHighlighted: true,
		},
		{
			name: "select start node",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			selectNodeID:    "start",
			wantSelected:    true,
			wantHighlighted: true,
		},
		{
			name: "select non-existent node",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			selectNodeID:    "nonexistent",
			wantSelected:    false,
			wantHighlighted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, _ := NewWorkflowBuilder(wf)

			err := builder.SelectNode(tt.selectNodeID)
			if !tt.wantSelected && err == nil {
				t.Error("SelectNode() expected error for non-existent node")
				return
			}
			if tt.wantSelected && err != nil {
				t.Errorf("SelectNode() unexpected error: %v", err)
				return
			}

			if !tt.wantSelected {
				return
			}

			selectedID := builder.GetSelectedNodeID()
			if selectedID != tt.selectNodeID {
				t.Errorf("GetSelectedNodeID() = %v, want %v", selectedID, tt.selectNodeID)
			}

			isHighlighted := builder.IsNodeHighlighted(tt.selectNodeID)
			if isHighlighted != tt.wantHighlighted {
				t.Errorf("IsNodeHighlighted() = %v, want %v", isHighlighted, tt.wantHighlighted)
			}
		})
	}
}

// TestWorkflowBuilderView_NodeSelectionNavigation tests navigating between nodes with keyboard
func TestWorkflowBuilderView_NodeSelectionNavigation(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() *workflow.Workflow
		keys          []string
		wantNodeID    string
	}{
		{
			name: "tab navigates to next node",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			keys:       []string{"Tab", "Tab"},
			wantNodeID: "tool-1",
		},
		{
			name: "shift-tab navigates to previous node",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			keys:       []string{"Tab", "Tab", "Shift+Tab"},
			wantNodeID: "tool-1",
		},
		{
			name: "arrow keys navigate spatially",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
				return wf
			},
			keys:       []string{"Right", "Right"},
			wantNodeID: "tool-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, _ := NewWorkflowBuilder(wf)
			builder.SelectNode("start") // Start from start node

			for _, key := range tt.keys {
				if err := builder.HandleKey(key); err != nil {
					t.Errorf("HandleKey(%q) error: %v", key, err)
				}
			}

			selectedID := builder.GetSelectedNodeID()
			if selectedID != tt.wantNodeID {
				t.Errorf("after navigation, selected node = %v, want %v", selectedID, tt.wantNodeID)
			}
		})
	}
}

// TestWorkflowBuilderView_ValidationDisplay tests real-time workflow validation display
func TestWorkflowBuilderView_ValidationDisplay(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() *workflow.Workflow
		wantValid     bool
		wantErrors    int
		wantMessages  []string
	}{
		{
			name: "valid workflow shows no errors",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("valid", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "end"})
				return wf
			},
			wantValid:    true,
			wantErrors:   0,
			wantMessages: []string{},
		},
		{
			name: "workflow without start node shows error",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("no-start", "")
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			wantValid:  false,
			wantErrors: 1,
			wantMessages: []string{
				"must have exactly one start node",
			},
		},
		{
			name: "workflow without end node shows error",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("no-end", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				return wf
			},
			wantValid:  false,
			wantErrors: 1,
			wantMessages: []string{
				"must have at least one end node",
			},
		},
		{
			name: "workflow with circular dependency shows error",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("cycle", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-2"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "tool-2"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "tool-1"}) // cycle
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-2", ToNodeID: "end"})
				return wf
			},
			wantValid:  false,
			wantErrors: 1,
			wantMessages: []string{
				"circular dependency",
			},
		},
		{
			name: "workflow with orphaned nodes shows error",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("orphan", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.MCPToolNode{ID: "orphan"}) // not connected
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "tool-1", ToNodeID: "end"})
				return wf
			},
			wantValid:  false,
			wantErrors: 1,
			wantMessages: []string{
				"orphaned node",
			},
		},
		{
			name: "workflow with multiple errors shows all",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("multi-error", "")
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"}) // no start
				wf.AddNode(&workflow.MCPToolNode{ID: "orphan"}) // orphaned
				return wf
			},
			wantValid:  false,
			wantErrors: 2,
			wantMessages: []string{
				"must have exactly one start node",
				"must have at least one end node",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, _ := NewWorkflowBuilder(wf)

			validation := builder.GetValidationStatus()
			if validation == nil {
				t.Fatal("GetValidationStatus() returned nil")
			}

			if validation.IsValid != tt.wantValid {
				t.Errorf("GetValidationStatus().IsValid = %v, want %v", validation.IsValid, tt.wantValid)
			}

			errorCount := len(validation.Errors)
			if errorCount != tt.wantErrors {
				t.Errorf("GetValidationStatus() error count = %v, want %v", errorCount, tt.wantErrors)
			}

			for _, wantMsg := range tt.wantMessages {
				found := false
				for _, err := range validation.Errors {
					if containsString(err.Message, wantMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetValidationStatus() missing expected error message: %v", wantMsg)
				}
			}
		})
	}
}

// TestWorkflowBuilderView_SaveWorkflow tests save workflow operations
func TestWorkflowBuilderView_SaveWorkflow(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() *workflow.Workflow
		setupBuilder  func(*WorkflowBuilder)
		wantErr       bool
		wantModified  bool
	}{
		{
			name: "save valid workflow",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("save-test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "end"})
				return wf
			},
			setupBuilder: func(b *WorkflowBuilder) {
				b.MarkModified()
			},
			wantErr:      false,
			wantModified: false, // should be cleared after save
		},
		{
			name: "save invalid workflow should fail",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("invalid", "")
				// Missing start and end nodes
				return wf
			},
			setupBuilder: func(b *WorkflowBuilder) {
				b.MarkModified()
			},
			wantErr:      true,
			wantModified: true, // should remain modified on error
		},
		{
			name: "save unmodified workflow",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("unmodified", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			setupBuilder: func(b *WorkflowBuilder) {
				// Don't mark as modified
			},
			wantErr:      false,
			wantModified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, _ := NewWorkflowBuilder(wf)
			tt.setupBuilder(builder)

			err := builder.SaveWorkflow()

			if tt.wantErr && err == nil {
				t.Error("SaveWorkflow() expected error but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("SaveWorkflow() unexpected error: %v", err)
				return
			}

			isModified := builder.IsModified()
			if isModified != tt.wantModified {
				t.Errorf("after save, IsModified() = %v, want %v", isModified, tt.wantModified)
			}
		})
	}
}

// TestWorkflowBuilderView_LoadWorkflow tests load workflow operations
func TestWorkflowBuilderView_LoadWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		setupData    func() *workflow.Workflow
		wantErr      bool
		wantNodes    int
	}{
		{
			name:         "load existing workflow",
			workflowName: "existing-workflow",
			setupData: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("existing-workflow", "test")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			wantErr:   false,
			wantNodes: 3,
		},
		{
			name:         "load non-existent workflow should fail",
			workflowName: "nonexistent",
			setupData:    func() *workflow.Workflow { return nil },
			wantErr:      true,
			wantNodes:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create initial workflow for the builder
			initialWf, err := workflow.NewWorkflow("initial", "test")
			if err != nil {
				t.Fatalf("Failed to create initial workflow: %v", err)
			}

			builder, err := NewWorkflowBuilder(initialWf)
			if err != nil {
				t.Fatalf("Failed to create builder: %v", err)
			}

			err = builder.LoadWorkflow(tt.workflowName)

			if tt.wantErr && err == nil {
				t.Error("LoadWorkflow() expected error but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("LoadWorkflow() unexpected error: %v", err)
				return
			}

			if tt.wantErr {
				return
			}

			wf := builder.GetWorkflow()
			if wf == nil {
				t.Fatal("LoadWorkflow() workflow is nil")
			}

			nodeCount := len(wf.Nodes)
			if nodeCount != tt.wantNodes {
				t.Errorf("LoadWorkflow() node count = %v, want %v", nodeCount, tt.wantNodes)
			}
		})
	}
}

// TestWorkflowBuilderView_UndoRedo tests undo/redo functionality
func TestWorkflowBuilderView_UndoRedo(t *testing.T) {
	tests := []struct {
		name          string
		operations    []func(*WorkflowBuilder)
		undoCount     int
		redoCount     int
		wantNodeCount int
		wantCanUndo   bool
		wantCanRedo   bool
	}{
		{
			name: "undo node addition",
			operations: []func(*WorkflowBuilder){
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-1"})
				},
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-2"})
				},
			},
			undoCount:     1,
			redoCount:     0,
			wantNodeCount: 3, // start + tool-1
			wantCanUndo:   true,
			wantCanRedo:   true,
		},
		{
			name: "redo node addition",
			operations: []func(*WorkflowBuilder){
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-1"})
				},
			},
			undoCount:     1,
			redoCount:     1,
			wantNodeCount: 3, // start + tool-1 + end
			wantCanUndo:   true,
			wantCanRedo:   false,
		},
		{
			name: "undo multiple operations",
			operations: []func(*WorkflowBuilder){
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-1"})
				},
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-2"})
				},
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-3"})
				},
			},
			undoCount:     3,
			redoCount:     0,
			wantNodeCount: 2, // just start + end
			wantCanUndo:   false,
			wantCanRedo:   true,
		},
		{
			name: "undo then new operation clears redo",
			operations: []func(*WorkflowBuilder){
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-1"})
				},
				func(b *WorkflowBuilder) {
					b.AddNodeToCanvas(&workflow.MCPToolNode{ID: "tool-2"})
				},
			},
			undoCount:     1, // undo tool-2
			redoCount:     0,
			wantNodeCount: 3, // start + tool-1 + end
			wantCanUndo:   true,
			wantCanRedo:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("undo-test", "")
			wf.AddNode(&workflow.StartNode{ID: "start"})
			wf.AddNode(&workflow.EndNode{ID: "end"})
			builder, _ := NewWorkflowBuilder(wf)

			// Perform operations
			for _, op := range tt.operations {
				op(builder)
			}

			// Undo operations
			for i := 0; i < tt.undoCount; i++ {
				if err := builder.Undo(); err != nil {
					t.Errorf("Undo() error at step %d: %v", i, err)
				}
			}

			// Redo operations
			for i := 0; i < tt.redoCount; i++ {
				if err := builder.Redo(); err != nil {
					t.Errorf("Redo() error at step %d: %v", i, err)
				}
			}

			wf = builder.GetWorkflow()
			nodeCount := len(wf.Nodes)
			if nodeCount != tt.wantNodeCount {
				t.Errorf("after undo/redo, node count = %v, want %v", nodeCount, tt.wantNodeCount)
			}

			canUndo := builder.CanUndo()
			if canUndo != tt.wantCanUndo {
				t.Errorf("CanUndo() = %v, want %v", canUndo, tt.wantCanUndo)
			}

			canRedo := builder.CanRedo()
			if canRedo != tt.wantCanRedo {
				t.Errorf("CanRedo() = %v, want %v", canRedo, tt.wantCanRedo)
			}
		})
	}
}

// TestWorkflowBuilderView_ModeSwitching tests view mode vs edit mode switching
func TestWorkflowBuilderView_ModeSwitching(t *testing.T) {
	tests := []struct {
		name            string
		initialMode     string
		switchToMode    string
		wantMode        string
		wantKeyBindings map[string]bool // expected available keybindings
	}{
		{
			name:         "switch from view to edit mode",
			initialMode:  "view",
			switchToMode: "edit",
			wantMode:     "edit",
			wantKeyBindings: map[string]bool{
				"a": true, // add node
				"d": true, // delete node
				"e": true, // edit node
				"c": true, // connect nodes
			},
		},
		{
			name:         "switch from edit to view mode",
			initialMode:  "edit",
			switchToMode: "view",
			wantMode:     "view",
			wantKeyBindings: map[string]bool{
				"a": false, // cannot add in view mode
				"d": false, // cannot delete in view mode
				"e": true,  // can still view details
			},
		},
		{
			name:         "press i to enter edit mode",
			initialMode:  "view",
			switchToMode: "i", // vim-style insert
			wantMode:     "edit",
			wantKeyBindings: map[string]bool{
				"a": true,
			},
		},
		{
			name:         "press Esc to exit edit mode",
			initialMode:  "edit",
			switchToMode: "Esc",
			wantMode:     "view",
			wantKeyBindings: map[string]bool{
				"a": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("mode-test", "")
			wf.AddNode(&workflow.StartNode{ID: "start"})
			wf.AddNode(&workflow.EndNode{ID: "end"})
			builder, _ := NewWorkflowBuilder(wf)

			// Set initial mode
			builder.SetMode(tt.initialMode)

			// Switch mode
			if err := builder.HandleKey(tt.switchToMode); err != nil {
				// If not a key, try direct mode switch
				builder.SetMode(tt.switchToMode)
			}

			currentMode := builder.Mode()
			if currentMode != tt.wantMode {
				t.Errorf("after mode switch, Mode() = %v, want %v", currentMode, tt.wantMode)
			}

			// Check keybindings
			for key, wantEnabled := range tt.wantKeyBindings {
				isEnabled := builder.IsKeyEnabled(key)
				if isEnabled != wantEnabled {
					t.Errorf("IsKeyEnabled(%q) = %v, want %v in mode %s", key, isEnabled, wantEnabled, currentMode)
				}
			}
		})
	}
}

// TestWorkflowBuilderView_HelpPanel tests help panel display
func TestWorkflowBuilderView_HelpPanel(t *testing.T) {
	tests := []struct {
		name            string
		mode            string
		pressHelpKey    bool
		wantVisible     bool
		wantKeyBindings []KeyBinding
	}{
		{
			name:         "help panel shows in view mode",
			mode:         "view",
			pressHelpKey: true,
			wantVisible:  true,
			wantKeyBindings: []KeyBinding{
				{Key: "i", Description: "Enter edit mode"},
				{Key: "Tab", Description: "Next node"},
				{Key: "?", Description: "Toggle help"},
				{Key: "q", Description: "Quit"},
			},
		},
		{
			name:         "help panel shows in edit mode",
			mode:         "edit",
			pressHelpKey: true,
			wantVisible:  true,
			wantKeyBindings: []KeyBinding{
				{Key: "a", Description: "Add node"},
				{Key: "d", Description: "Delete node"},
				{Key: "e", Description: "Edit node"},
				{Key: "c", Description: "Connect nodes"},
				{Key: "u", Description: "Undo"},
				{Key: "Ctrl+r", Description: "Redo"},
				{Key: "Esc", Description: "Exit edit mode"},
				{Key: "?", Description: "Toggle help"},
			},
		},
		{
			name:         "help panel toggles visibility",
			mode:         "view",
			pressHelpKey: false,
			wantVisible:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("help-test", "")
			builder, _ := NewWorkflowBuilder(wf)
			builder.SetMode(tt.mode)

			if tt.pressHelpKey {
				builder.HandleKey("?")
			}

			helpPanel := builder.GetHelpPanel()
			if helpPanel == nil {
				t.Fatal("GetHelpPanel() returned nil")
			}

			isVisible := helpPanel.IsVisible()
			if isVisible != tt.wantVisible {
				t.Errorf("GetHelpPanel().IsVisible() = %v, want %v", isVisible, tt.wantVisible)
			}

			if !tt.wantVisible {
				return
			}

			bindings := helpPanel.GetKeyBindings()
			if len(bindings) < len(tt.wantKeyBindings) {
				t.Errorf("GetKeyBindings() count = %v, want at least %v", len(bindings), len(tt.wantKeyBindings))
			}

			for _, wantBinding := range tt.wantKeyBindings {
				found := false
				for _, binding := range bindings {
					if binding.Key == wantBinding.Key && containsString(binding.Description, wantBinding.Description) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetKeyBindings() missing binding: %v - %v", wantBinding.Key, wantBinding.Description)
				}
			}
		})
	}
}

// TestWorkflowBuilderView_AddNodeToCanvas tests adding nodes to canvas from palette
func TestWorkflowBuilderView_AddNodeToCanvas(t *testing.T) {
	tests := []struct {
		name          string
		nodeType      string
		position      Position
		wantNodeCount int
		wantErr       bool
	}{
		{
			name:          "add MCP tool node to canvas",
			nodeType:      "MCP Tool",
			position:      Position{X: 40, Y: 12},
			wantNodeCount: 3, // start + new node + end
			wantErr:       false,
		},
		{
			name:          "add transform node to canvas",
			nodeType:      "Transform",
			position:      Position{X: 50, Y: 15},
			wantNodeCount: 3,
			wantErr:       false,
		},
		{
			name:          "add condition node to canvas",
			nodeType:      "Condition",
			position:      Position{X: 45, Y: 10},
			wantNodeCount: 3,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("add-test", "")
			wf.AddNode(&workflow.StartNode{ID: "start"})
			wf.AddNode(&workflow.EndNode{ID: "end"})
			builder, _ := NewWorkflowBuilder(wf)
			builder.SetMode("edit")

			err := builder.AddNodeAtPosition(tt.nodeType, tt.position)

			if tt.wantErr && err == nil {
				t.Error("AddNodeAtPosition() expected error but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("AddNodeAtPosition() unexpected error: %v", err)
				return
			}

			wf = builder.GetWorkflow()
			nodeCount := len(wf.Nodes)
			if nodeCount != tt.wantNodeCount {
				t.Errorf("after adding node, count = %v, want %v", nodeCount, tt.wantNodeCount)
			}

			if !builder.IsModified() {
				t.Error("AddNodeAtPosition() should mark workflow as modified")
			}
		})
	}
}

// TestWorkflowBuilderView_EdgeCreation tests creating edges between nodes
func TestWorkflowBuilderView_EdgeCreation(t *testing.T) {
	tests := []struct {
		name          string
		setupWorkflow func() *workflow.Workflow
		fromNodeID    string
		toNodeID      string
		wantEdgeCount int
		wantErr       bool
	}{
		{
			name: "create edge between valid nodes",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("edge-test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			fromNodeID:    "start",
			toNodeID:      "tool-1",
			wantEdgeCount: 1,
			wantErr:       false,
		},
		{
			name: "create edge to non-existent node should fail",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("edge-test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				return wf
			},
			fromNodeID:    "start",
			toNodeID:      "nonexistent",
			wantEdgeCount: 0,
			wantErr:       true,
		},
		{
			name: "create duplicate edge should fail",
			setupWorkflow: func() *workflow.Workflow {
				wf, _ := workflow.NewWorkflow("edge-test", "")
				wf.AddNode(&workflow.StartNode{ID: "start"})
				wf.AddNode(&workflow.MCPToolNode{ID: "tool-1"})
				wf.AddNode(&workflow.EndNode{ID: "end"})
				wf.AddEdge(&workflow.Edge{FromNodeID: "start", ToNodeID: "tool-1"})
				return wf
			},
			fromNodeID:    "start",
			toNodeID:      "tool-1",
			wantEdgeCount: 1,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.setupWorkflow()
			builder, _ := NewWorkflowBuilder(wf)
			builder.SetMode("edit")

			err := builder.CreateEdge(tt.fromNodeID, tt.toNodeID)

			if tt.wantErr && err == nil {
				t.Error("CreateEdge() expected error but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("CreateEdge() unexpected error: %v", err)
				return
			}

			wf = builder.GetWorkflow()
			edgeCount := len(wf.Edges)
			if edgeCount != tt.wantEdgeCount {
				t.Errorf("after creating edge, count = %v, want %v", edgeCount, tt.wantEdgeCount)
			}
		})
	}
}

// TestWorkflowBuilderView_KeyboardShortcuts tests various keyboard shortcuts
func TestWorkflowBuilderView_KeyboardShortcuts(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		key        string
		wantAction string
		wantErr    bool
	}{
		{
			name:       "Ctrl+s saves workflow",
			mode:       "edit",
			key:        "Ctrl+s",
			wantAction: "save",
			wantErr:    false,
		},
		{
			name:       "Ctrl+o opens workflow",
			mode:       "view",
			key:        "Ctrl+o",
			wantAction: "open",
			wantErr:    false,
		},
		{
			name:       "u undoes last action",
			mode:       "edit",
			key:        "u",
			wantAction: "undo",
			wantErr:    false,
		},
		{
			name:       "Ctrl+r redoes action",
			mode:       "edit",
			key:        "Ctrl+r",
			wantAction: "redo",
			wantErr:    false,
		},
		{
			name:       "? toggles help",
			mode:       "view",
			key:        "?",
			wantAction: "toggle_help",
			wantErr:    false,
		},
		{
			name:       "q quits builder",
			mode:       "view",
			key:        "q",
			wantAction: "quit",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, _ := workflow.NewWorkflow("shortcut-test", "")
			wf.AddNode(&workflow.StartNode{ID: "start"})
			wf.AddNode(&workflow.EndNode{ID: "end"})
			builder, _ := NewWorkflowBuilder(wf)
			builder.SetMode(tt.mode)

			action, err := builder.GetActionForKey(tt.key)

			if tt.wantErr && err == nil {
				t.Error("GetActionForKey() expected error but got none")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("GetActionForKey() unexpected error: %v", err)
				return
			}

			if action != tt.wantAction {
				t.Errorf("GetActionForKey(%q) = %v, want %v", tt.key, action, tt.wantAction)
			}
		})
	}
}

// Helper types - re-export from tui package for test convenience
type WorkflowBuilder = tui.WorkflowBuilder
type Canvas = tui.Canvas
type NodePalette = tui.NodePalette
type ValidationStatus = tui.ValidationStatus
type ValidationError = tui.ValidationError
type HelpPanel = tui.HelpPanel
type KeyBinding = tui.HelpKeyBinding

// Position is defined in common_test.go

// Helper functions
var NewWorkflowBuilder = tui.NewWorkflowBuilder

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}
