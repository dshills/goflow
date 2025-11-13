package tui

import (
	"testing"

	"github.com/dshills/goflow/pkg/workflow"
)

func TestConditionNodeInPalette(t *testing.T) {
	// T121: Verify condition node is in the palette
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	palette := builder.GetNodePalette()

	// Filter for "Condition" node type
	filtered := palette.Filter("condition")

	found := false
	for _, nodeType := range filtered {
		if nodeType.typeName == "Condition" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Condition node not found in palette")
	}
}

func TestAddConditionNodeToCanvas(t *testing.T) {
	// T121: Verify condition node can be added to canvas
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add condition node
	err = builder.AddNodeAtPosition("Condition", Position{X: 10, Y: 10})
	if err != nil {
		t.Errorf("Failed to add condition node: %v", err)
	}

	// Verify node was added
	nodes := builder.GetWorkflow().Nodes
	if len(nodes) == 0 {
		t.Fatal("No nodes added to workflow")
	}

	condNode, ok := nodes[0].(*workflow.ConditionNode)
	if !ok {
		t.Errorf("Added node is not a ConditionNode, got: %T", nodes[0])
	}

	if condNode.Type() != "condition" {
		t.Errorf("Expected node type 'condition', got: %s", condNode.Type())
	}
}

func TestConditionNodePropertyPanel(t *testing.T) {
	// T122: Verify condition expression editor in property panel
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add condition node
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "price > 100",
	}
	err = builder.AddNodeToCanvas(condNode)
	if err != nil {
		t.Fatalf("Failed to add condition node: %v", err)
	}

	// Show property panel
	err = builder.ShowPropertyPanel("cond_1")
	if err != nil {
		t.Fatalf("Failed to show property panel: %v", err)
	}

	panel := builder.GetPropertyPanel()
	if !panel.IsVisible() {
		t.Error("Property panel should be visible")
	}

	if panel.GetNodeType() != "condition" {
		t.Errorf("Expected node type 'condition', got: %s", panel.GetNodeType())
	}

	fields := panel.GetFields()
	foundConditionField := false
	for _, field := range fields {
		if field.label == "Condition Expression" {
			foundConditionField = true
			if field.value != "price > 100" {
				t.Errorf("Expected condition 'price > 100', got: %s", field.value)
			}
			if field.fieldType != "condition" {
				t.Errorf("Expected field type 'condition', got: %s", field.fieldType)
			}
		}
	}

	if !foundConditionField {
		t.Error("Condition Expression field not found in property panel")
	}
}

func TestConditionExpressionValidation(t *testing.T) {
	// T124: Verify expression validation in property panel
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add a variable for testing
	wf.AddVariable(&workflow.Variable{
		Name: "price",
		Type: "number",
	})

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add condition node
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "",
	}
	err = builder.AddNodeToCanvas(condNode)
	if err != nil {
		t.Fatalf("Failed to add condition node: %v", err)
	}

	// Show property panel
	err = builder.ShowPropertyPanel("cond_1")
	if err != nil {
		t.Fatalf("Failed to show property panel: %v", err)
	}

	// Test valid expression
	validExpr := "price > 100"
	err = builder.UpdatePropertyField(1, validExpr) // Index 1 is the condition field
	if err != nil {
		t.Errorf("Valid expression should not produce error: %v", err)
	}

	panel := builder.GetPropertyPanel()
	if panel.GetValidationMessage() != "" {
		t.Errorf("Expected no validation message for valid expression, got: %s", panel.GetValidationMessage())
	}

	// Test invalid expression (syntax error)
	invalidExpr := "price > > 100" // Invalid syntax
	err = builder.UpdatePropertyField(1, invalidExpr)
	if err == nil {
		t.Error("Invalid expression should produce error")
	}
}

func TestConditionalEdgeLabels(t *testing.T) {
	// T123: Verify conditional edge labels
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add nodes
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "x > 0",
	}
	trueNode := &workflow.PassthroughNode{ID: "true_path"}
	falseNode := &workflow.PassthroughNode{ID: "false_path"}

	builder.AddNodeToCanvas(condNode)
	builder.AddNodeToCanvas(trueNode)
	builder.AddNodeToCanvas(falseNode)

	// Create conditional edges
	err = builder.CreateConditionalEdge("cond_1", "true_path", "true")
	if err != nil {
		t.Fatalf("Failed to create true edge: %v", err)
	}

	err = builder.CreateConditionalEdge("cond_1", "false_path", "false")
	if err != nil {
		t.Fatalf("Failed to create false edge: %v", err)
	}

	// Verify edges have labels
	edges := builder.GetWorkflow().Edges
	if len(edges) != 2 {
		t.Fatalf("Expected 2 edges, got: %d", len(edges))
	}

	trueEdgeFound := false
	falseEdgeFound := false

	for _, edge := range edges {
		label := builder.GetEdgeLabel(edge)
		if label == "true" {
			trueEdgeFound = true
			if edge.ToNodeID != "true_path" {
				t.Errorf("True edge should go to true_path, got: %s", edge.ToNodeID)
			}
		} else if label == "false" {
			falseEdgeFound = true
			if edge.ToNodeID != "false_path" {
				t.Errorf("False edge should go to false_path, got: %s", edge.ToNodeID)
			}
		}
	}

	if !trueEdgeFound {
		t.Error("True edge not found")
	}
	if !falseEdgeFound {
		t.Error("False edge not found")
	}
}

func TestConditionalEdgeStyles(t *testing.T) {
	// T123: Verify edge styles for true/false branches
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add nodes
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "x > 0",
	}
	trueNode := &workflow.PassthroughNode{ID: "true_path"}
	falseNode := &workflow.PassthroughNode{ID: "false_path"}

	builder.AddNodeToCanvas(condNode)
	builder.AddNodeToCanvas(trueNode)
	builder.AddNodeToCanvas(falseNode)

	// Create conditional edges
	builder.CreateConditionalEdge("cond_1", "true_path", "true")
	builder.CreateConditionalEdge("cond_1", "false_path", "false")

	// Verify edge styles
	edges := builder.GetWorkflow().Edges
	for _, edge := range edges {
		style := builder.GetEdgeStyle(edge)
		if edge.Condition == "true" {
			if style != "solid" {
				t.Errorf("True edge should have solid style, got: %s", style)
			}
		} else if edge.Condition == "false" {
			if style != "dashed" {
				t.Errorf("False edge should have dashed style, got: %s", style)
			}
		}
	}
}

func TestOnlyOneEdgePerCondition(t *testing.T) {
	// T123: Verify only one true edge and one false edge per condition node
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add nodes
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "x > 0",
	}
	node1 := &workflow.PassthroughNode{ID: "node_1"}
	node2 := &workflow.PassthroughNode{ID: "node_2"}

	builder.AddNodeToCanvas(condNode)
	builder.AddNodeToCanvas(node1)
	builder.AddNodeToCanvas(node2)

	// Create first true edge
	err = builder.CreateConditionalEdge("cond_1", "node_1", "true")
	if err != nil {
		t.Fatalf("Failed to create first true edge: %v", err)
	}

	// Try to create second true edge (should fail)
	err = builder.CreateConditionalEdge("cond_1", "node_2", "true")
	if err == nil {
		t.Error("Should not allow two true edges from same condition node")
	}
}

func TestPropertyPanelRendering(t *testing.T) {
	// T122: Verify property panel can render
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Add condition node
	condNode := &workflow.ConditionNode{
		ID:        "cond_1",
		Condition: "price > 100 && inStock",
	}
	builder.AddNodeToCanvas(condNode)

	// Show property panel
	builder.ShowPropertyPanel("cond_1")
	panel := builder.GetPropertyPanel()

	// Render the panel
	output := panel.RenderPropertyPanel()
	if output == "" {
		t.Error("Property panel rendering should not be empty")
	}

	// Check that output contains key elements
	if !stringContains(output, "Condition Node Properties") {
		t.Error("Output should contain node type header")
	}
	if !stringContains(output, "Condition Expression") {
		t.Error("Output should contain condition expression field")
	}
	if !stringContains(output, "price > 100 && inStock") {
		t.Error("Output should contain the actual condition value")
	}
}

func TestVariableListInPropertyPanel(t *testing.T) {
	// T122: Verify variable suggestions are available
	wf, err := workflow.NewWorkflow("test", "test workflow")
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Add variables
	wf.AddVariable(&workflow.Variable{Name: "price", Type: "number"})
	wf.AddVariable(&workflow.Variable{Name: "quantity", Type: "number"})
	wf.AddVariable(&workflow.Variable{Name: "inStock", Type: "boolean"})

	builder, err := NewWorkflowBuilder(wf)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Get variable list
	vars := builder.GetVariableList()
	if len(vars) != 3 {
		t.Errorf("Expected 3 variables, got: %d", len(vars))
	}

	expectedVars := map[string]bool{
		"price":    true,
		"quantity": true,
		"inStock":  true,
	}

	for _, v := range vars {
		if !expectedVars[v] {
			t.Errorf("Unexpected variable: %s", v)
		}
		delete(expectedVars, v)
	}

	if len(expectedVars) > 0 {
		t.Errorf("Missing variables: %v", expectedVars)
	}
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
