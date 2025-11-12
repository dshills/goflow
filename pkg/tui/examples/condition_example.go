package main

import (
	"fmt"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
)

// This example demonstrates the TUI features for conditional logic
func main() {
	// Create a workflow
	wf, err := workflow.NewWorkflow("price-check", "Price threshold checker")
	if err != nil {
		panic(err)
	}

	// Add variables that will be used in conditions
	if err := wf.AddVariable(&workflow.Variable{
		Name: "price",
		Type: "number",
	}); err != nil {
		panic(fmt.Errorf("failed to add price variable: %w", err))
	}
	if err := wf.AddVariable(&workflow.Variable{
		Name: "quantity",
		Type: "number",
	}); err != nil {
		panic(fmt.Errorf("failed to add quantity variable: %w", err))
	}
	if err := wf.AddVariable(&workflow.Variable{
		Name: "inStock",
		Type: "boolean",
	}); err != nil {
		panic(fmt.Errorf("failed to add inStock variable: %w", err))
	}

	// Create TUI builder
	builder, err := tui.NewWorkflowBuilder(wf)
	if err != nil {
		panic(err)
	}

	// Add a start node
	startNode := &workflow.StartNode{ID: "start"}
	if err := builder.AddNodeToCanvas(startNode); err != nil {
		panic(fmt.Errorf("failed to add start node to canvas: %w", err))
	}

	// Add a condition node
	condNode := &workflow.ConditionNode{
		ID:        "check_threshold",
		Condition: "price > 100 && inStock",
	}
	if err := builder.AddNodeToCanvas(condNode); err != nil {
		panic(fmt.Errorf("failed to add condition node to canvas: %w", err))
	}

	// Add nodes for true and false paths
	highValueNode := &workflow.TransformNode{
		ID:             "high_value_discount",
		InputVariable:  "price",
		Expression:     "price * 0.9", // 10% discount
		OutputVariable: "final_price",
	}
	if err := builder.AddNodeToCanvas(highValueNode); err != nil {
		panic(fmt.Errorf("failed to add high value node to canvas: %w", err))
	}

	lowValueNode := &workflow.TransformNode{
		ID:             "regular_price",
		InputVariable:  "price",
		Expression:     "price",
		OutputVariable: "final_price",
	}
	if err := builder.AddNodeToCanvas(lowValueNode); err != nil {
		panic(fmt.Errorf("failed to add low value node to canvas: %w", err))
	}

	// Add end node
	endNode := &workflow.EndNode{
		ID:          "end",
		ReturnValue: "final_price",
	}
	if err := builder.AddNodeToCanvas(endNode); err != nil {
		panic(fmt.Errorf("failed to add end node to canvas: %w", err))
	}

	// Create edges
	if err := builder.CreateEdge("start", "check_threshold"); err != nil {
		panic(fmt.Errorf("failed to create edge start->check_threshold: %w", err))
	}

	// Create conditional edges with labels
	if err := builder.CreateConditionalEdge("check_threshold", "high_value_discount", "true"); err != nil {
		panic(fmt.Errorf("failed to create conditional edge (true): %w", err))
	}
	if err := builder.CreateConditionalEdge("check_threshold", "regular_price", "false"); err != nil {
		panic(fmt.Errorf("failed to create conditional edge (false): %w", err))
	}

	if err := builder.CreateEdge("high_value_discount", "end"); err != nil {
		panic(fmt.Errorf("failed to create edge high_value_discount->end: %w", err))
	}
	if err := builder.CreateEdge("regular_price", "end"); err != nil {
		panic(fmt.Errorf("failed to create edge regular_price->end: %w", err))
	}

	// Demonstrate property panel
	fmt.Println("=== Workflow Created ===")
	fmt.Printf("Workflow: %s\n", wf.Name)
	fmt.Printf("Nodes: %d\n", len(wf.Nodes))
	fmt.Printf("Edges: %d\n", len(wf.Edges))
	fmt.Println()

	// Show property panel for condition node
	fmt.Println("=== Property Panel ===")
	err = builder.ShowPropertyPanel("check_threshold")
	if err != nil {
		panic(err)
	}

	panel := builder.GetPropertyPanel()
	fmt.Println(panel.RenderPropertyPanel())

	// Show edge information
	fmt.Println("\n=== Edge Information ===")
	for _, edge := range wf.Edges {
		label := builder.GetEdgeLabel(edge)
		style := builder.GetEdgeStyle(edge)

		if label != "" {
			fmt.Printf("%s --%s--> %s [%s]\n",
				edge.FromNodeID,
				label,
				edge.ToNodeID,
				style,
			)
		} else {
			fmt.Printf("%s --> %s\n",
				edge.FromNodeID,
				edge.ToNodeID,
			)
		}
	}

	// Show variable list
	fmt.Println("\n=== Available Variables ===")
	vars := builder.GetVariableList()
	for _, v := range vars {
		fmt.Printf("  - %s\n", v)
	}

	// Demonstrate validation
	fmt.Println("\n=== Expression Validation ===")

	testExpressions := []string{
		"price > 100",               // Valid
		"price > 100 && inStock",    // Valid
		"quantity >= 10",            // Valid
		"price > > 100",             // Invalid syntax
		"unknownVar > 0",            // Invalid (undefined variable in workflow context)
		"price > 100 && os.Exit(0)", // Invalid (unsafe operation)
	}

	for _, expr := range testExpressions {
		err := workflow.ValidateExpressionSyntax(expr)
		status := "✓ Valid"
		if err != nil {
			status = fmt.Sprintf("✗ Invalid: %v", err)
		}
		fmt.Printf("  %s -> %s\n", expr, status)
	}

	fmt.Println("\n=== Workflow Visualization ===")
	fmt.Println("  [start]")
	fmt.Println("     │")
	fmt.Println("     ▼")
	fmt.Println(" [check_threshold]")
	fmt.Println("     │         \\")
	fmt.Println("     │ true     \\ false")
	fmt.Println("     │           \\")
	fmt.Println("     ▼            ▼")
	fmt.Println(" [high_value]  [regular_price]")
	fmt.Println("     │            │")
	fmt.Println("     └────┬───────┘")
	fmt.Println("          ▼")
	fmt.Println("        [end]")
}
