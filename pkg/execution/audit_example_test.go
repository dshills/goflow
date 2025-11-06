package execution_test

import (
	"fmt"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	execpkg "github.com/dshills/goflow/pkg/execution"
)

// ExampleReconstructAuditTrail demonstrates basic audit trail reconstruction.
func ExampleReconstructAuditTrail() {
	// Create and execute a workflow
	exec, _ := execution.NewExecution(
		types.WorkflowID("example-workflow"),
		"1.0.0",
		map[string]interface{}{"user_id": "user-123"},
	)
	_ = exec.Start()

	// Execute nodes
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("fetch-user"), "mcp_tool")
	node1.Start()
	time.Sleep(10 * time.Millisecond)
	node1.Complete(map[string]interface{}{"user": map[string]interface{}{"name": "John Doe"}})
	_ = exec.AddNodeExecution(node1)

	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("send-email"), "mcp_tool")
	node2.Start()
	time.Sleep(10 * time.Millisecond)
	node2.Complete(map[string]interface{}{"status": "sent"})
	_ = exec.AddNodeExecution(node2)

	_ = exec.Complete(map[string]interface{}{"result": "success"})

	// Reconstruct audit trail
	trail, _ := execpkg.ReconstructAuditTrail(exec)

	// Display summary
	fmt.Printf("Execution: %s\n", trail.ExecutionID)
	fmt.Printf("Workflow: %s\n", trail.WorkflowID)
	fmt.Printf("Status: %s\n", trail.Status)
	fmt.Printf("Nodes executed: %d\n", trail.NodeCount)
	fmt.Printf("Duration: %s\n", trail.Duration)
}

// ExampleAuditTrail_FilterEvents demonstrates filtering audit events.
func ExampleAuditTrail_FilterEvents() {
	// Create execution with various events
	exec, _ := execution.NewExecution(
		types.WorkflowID("filter-example"),
		"1.0.0",
		nil,
	)
	_ = exec.Start()

	// Add successful node
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node1.Start()
	node1.Complete(nil)
	_ = exec.AddNodeExecution(node1)

	// Add failed node
	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("node-2"), "transform")
	node2.Start()
	node2.Fail(&execution.NodeError{
		Type:    execution.ErrorTypeData,
		Message: "Transformation failed",
	})
	_ = exec.AddNodeExecution(node2)

	_ = exec.Fail(&execution.ExecutionError{
		Type:    execution.ErrorTypeExecution,
		Message: "Node execution failed",
		NodeID:  types.NodeID("node-2"),
	})

	// Reconstruct trail
	trail, _ := execpkg.ReconstructAuditTrail(exec)

	// Filter to only error events
	errorFilter := execpkg.AuditTrailFilter{
		EventTypes: []execpkg.AuditEventType{
			execpkg.AuditEventNodeFailed,
			execpkg.AuditEventExecutionFailed,
		},
	}

	errorTrail := trail.FilterEvents(errorFilter)

	fmt.Printf("Total events: %d\n", len(trail.Events))
	fmt.Printf("Error events: %d\n", len(errorTrail.Events))
	fmt.Printf("Error count: %d\n", errorTrail.ErrorCount)
}

// ExampleAuditTrail_GetEventsForNode demonstrates retrieving events for a specific node.
func ExampleAuditTrail_GetEventsForNode() {
	exec, _ := execution.NewExecution(
		types.WorkflowID("node-events-example"),
		"1.0.0",
		nil,
	)
	_ = exec.Start()

	// Add node with retry
	node := execution.NewNodeExecution(exec.ID, types.NodeID("retry-node"), "mcp_tool")
	node.RetryCount = 2 // Simulating retries
	node.Start()
	node.Complete(map[string]interface{}{"result": "success"})
	_ = exec.AddNodeExecution(node)

	_ = exec.Complete(nil)

	// Reconstruct and get events for specific node
	trail, _ := execpkg.ReconstructAuditTrail(exec)
	nodeEvents := trail.GetEventsForNode(types.NodeID("retry-node"))

	fmt.Printf("Events for retry-node: %d\n", len(nodeEvents))
	fmt.Printf("Includes retry events: %v\n", trail.RetryCount > 0)
}

// ExampleAuditTrail_FormatHumanReadable demonstrates human-readable formatting.
func ExampleAuditTrail_FormatHumanReadable() {
	exec, _ := execution.NewExecution(
		types.WorkflowID("format-example"),
		"1.0.0",
		nil,
	)
	_ = exec.Start()

	node := execution.NewNodeExecution(exec.ID, types.NodeID("task-node"), "mcp_tool")
	node.Start()
	time.Sleep(10 * time.Millisecond)
	node.Complete(map[string]interface{}{"output": "result"})
	_ = exec.AddNodeExecution(node)

	_ = exec.Complete(map[string]interface{}{"final": "output"})

	// Reconstruct and format
	trail, _ := execpkg.ReconstructAuditTrail(exec)
	output := trail.FormatHumanReadable()

	// Output will contain:
	// - Execution metadata (ID, workflow, status, duration)
	// - Summary (nodes executed, errors, variable changes, retries)
	// - Chronological event timeline with timestamps and icons
	// - Event details (inputs, outputs, errors)

	fmt.Println("Audit trail formatted successfully")
	fmt.Printf("Contains execution ID: %v\n", len(output) > 0)
}

// ExampleAuditTrail_ExportJSON demonstrates JSON export.
func ExampleAuditTrail_ExportJSON() {
	exec, _ := execution.NewExecution(
		types.WorkflowID("json-export-example"),
		"1.0.0",
		nil,
	)
	_ = exec.Start()

	node := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node.Start()
	node.Complete(nil)
	_ = exec.AddNodeExecution(node)

	_ = exec.Complete(nil)

	// Reconstruct and export as JSON
	trail, _ := execpkg.ReconstructAuditTrail(exec)

	// Pretty JSON (indented)
	prettyJSON, _ := trail.ExportJSON()
	fmt.Printf("Pretty JSON bytes: %d\n", len(prettyJSON))

	// Compact JSON (no indentation)
	compactJSON, _ := trail.ExportCompactJSON()
	fmt.Printf("Compact JSON bytes: %d\n", len(compactJSON))
	fmt.Printf("Compact is smaller: %v\n", len(compactJSON) < len(prettyJSON))
}

// ExampleAuditTrail_GetErrorEvents demonstrates retrieving all error events.
func ExampleAuditTrail_GetErrorEvents() {
	exec, _ := execution.NewExecution(
		types.WorkflowID("error-events-example"),
		"1.0.0",
		nil,
	)
	_ = exec.Start()

	// Add some successful nodes
	for i := 0; i < 3; i++ {
		node := execution.NewNodeExecution(exec.ID, types.NodeID(fmt.Sprintf("node-%d", i)), "mcp_tool")
		node.Start()
		node.Complete(nil)
		_ = exec.AddNodeExecution(node)
	}

	// Add failed node
	failedNode := execution.NewNodeExecution(exec.ID, types.NodeID("failed-node"), "mcp_tool")
	failedNode.Start()
	failedNode.Fail(&execution.NodeError{
		Type:    execution.ErrorTypeExecution,
		Message: "Execution failed",
	})
	_ = exec.AddNodeExecution(failedNode)

	_ = exec.Fail(&execution.ExecutionError{
		Type:    execution.ErrorTypeExecution,
		Message: "Workflow failed",
		NodeID:  types.NodeID("failed-node"),
	})

	// Reconstruct and get errors
	trail, _ := execpkg.ReconstructAuditTrail(exec)
	errorEvents := trail.GetErrorEvents()

	fmt.Printf("Total events: %d\n", len(trail.Events))
	fmt.Printf("Error events: %d\n", len(errorEvents))
	fmt.Printf("Error count: %d\n", trail.ErrorCount)
}

// ExampleAuditTrail_GetVariableChanges demonstrates tracking variable changes.
func ExampleAuditTrail_GetVariableChanges() {
	exec, _ := execution.NewExecution(
		types.WorkflowID("variable-tracking"),
		"1.0.0",
		map[string]interface{}{"initial": "value"},
	)
	_ = exec.Start()

	// Make variable changes
	ctx := exec.Context
	_ = ctx.SetVariable("counter", 0)
	_ = ctx.SetVariable("counter", 1)
	_ = ctx.SetVariable("counter", 2)
	_ = ctx.SetVariable("status", "processing")

	_ = exec.Complete(nil)

	// Reconstruct and get variable changes
	trail, _ := execpkg.ReconstructAuditTrail(exec)
	varChanges := trail.GetVariableChanges()

	fmt.Printf("Variable changes: %d\n", len(varChanges))
	fmt.Printf("Variable change count: %d\n", trail.VariableChangeCount)
}
