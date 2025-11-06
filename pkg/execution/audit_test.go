package execution

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReconstructAuditTrail_BasicExecution tests audit trail reconstruction for a simple execution.
func TestReconstructAuditTrail_BasicExecution(t *testing.T) {
	// Create a simple execution with a few nodes
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		map[string]interface{}{"input": "test"},
	)
	require.NoError(t, err)

	// Start execution
	require.NoError(t, exec.Start())

	// Add some node executions
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node1.Start()
	time.Sleep(10 * time.Millisecond)
	node1.Complete(map[string]interface{}{"output": "result1"})
	require.NoError(t, exec.AddNodeExecution(node1))

	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("node-2"), "transform")
	node2.Start()
	time.Sleep(10 * time.Millisecond)
	node2.Complete(map[string]interface{}{"output": "result2"})
	require.NoError(t, exec.AddNodeExecution(node2))

	// Complete execution
	require.NoError(t, exec.Complete(map[string]interface{}{"final": "output"}))

	// Reconstruct audit trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)
	require.NotNil(t, trail)

	// Verify metadata
	assert.Equal(t, exec.ID, trail.ExecutionID)
	assert.Equal(t, exec.WorkflowID, trail.WorkflowID)
	assert.Equal(t, exec.Status, trail.Status)
	assert.Equal(t, 2, trail.NodeCount)
	assert.Equal(t, 0, trail.ErrorCount)

	// Verify we have events: execution_started, 2x node_started, 2x node_completed, execution_completed
	assert.GreaterOrEqual(t, len(trail.Events), 6)

	// Verify events are chronologically ordered
	for i := 1; i < len(trail.Events); i++ {
		assert.False(t, trail.Events[i].Timestamp.Before(trail.Events[i-1].Timestamp),
			"events should be in chronological order")
	}

	// Verify first event is execution started
	assert.Equal(t, AuditEventExecutionStarted, trail.Events[0].Type)

	// Verify last event is execution completed
	lastEvent := trail.Events[len(trail.Events)-1]
	assert.Equal(t, AuditEventExecutionCompleted, lastEvent.Type)
	assert.NotNil(t, lastEvent.Duration)
}

// TestReconstructAuditTrail_WithErrors tests audit trail with failed nodes.
func TestReconstructAuditTrail_WithErrors(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("error-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add successful node
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node1.Start()
	node1.Complete(map[string]interface{}{"output": "ok"})
	require.NoError(t, exec.AddNodeExecution(node1))

	// Add failed node
	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("node-2"), "mcp_tool")
	node2.Start()
	node2.Fail(&execution.NodeError{
		Type:    execution.ErrorTypeExecution,
		Message: "Tool execution failed",
		Context: map[string]interface{}{
			"tool":   "test-tool",
			"reason": "timeout",
		},
	})
	require.NoError(t, exec.AddNodeExecution(node2))

	// Fail execution
	require.NoError(t, exec.Fail(&execution.ExecutionError{
		Type:    execution.ErrorTypeExecution,
		Message: "Node execution failed",
		NodeID:  types.NodeID("node-2"),
	}))

	// Reconstruct audit trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Verify error tracking
	assert.Equal(t, execution.StatusFailed, trail.Status)
	assert.Equal(t, 2, trail.NodeCount)
	assert.Equal(t, 1, trail.ErrorCount) // One failed node

	// Get error events
	errorEvents := trail.GetErrorEvents()
	assert.GreaterOrEqual(t, len(errorEvents), 2) // node_failed + execution_failed

	// Verify node failure event has error details
	var nodeFailedEvent *AuditEvent
	for i := range trail.Events {
		if trail.Events[i].Type == AuditEventNodeFailed {
			nodeFailedEvent = &trail.Events[i]
			break
		}
	}
	require.NotNil(t, nodeFailedEvent)
	assert.Equal(t, types.NodeID("node-2"), nodeFailedEvent.NodeID)
	assert.Contains(t, nodeFailedEvent.Details, "error_type")
	assert.Contains(t, nodeFailedEvent.Details, "error_message")
}

// TestReconstructAuditTrail_WithRetries tests audit trail with node retries.
func TestReconstructAuditTrail_WithRetries(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("retry-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add node with retries
	node := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node.RetryCount = 3 // Simulating 3 retries
	node.Start()
	node.Complete(map[string]interface{}{"output": "success"})
	require.NoError(t, exec.AddNodeExecution(node))

	require.NoError(t, exec.Complete(nil))

	// Reconstruct audit trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Verify retry tracking
	assert.Equal(t, 3, trail.RetryCount)

	// Verify retry events exist
	retryEvents := trail.GetEventsByType(AuditEventNodeRetried)
	assert.Equal(t, 3, len(retryEvents))

	// Verify retry events are ordered
	for i, event := range retryEvents {
		assert.Contains(t, event.Message, "retry attempt")
		assert.Equal(t, i+1, event.Details["retry_count"])
	}
}

// TestReconstructAuditTrail_WithVariableChanges tests variable change tracking.
func TestReconstructAuditTrail_WithVariableChanges(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("variable-workflow"),
		"1.0.0",
		map[string]interface{}{"initial": "value"},
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Simulate variable changes through context
	ctx := exec.Context
	require.NoError(t, ctx.SetVariable("var1", "value1"))
	require.NoError(t, ctx.SetVariable("var2", 42))
	require.NoError(t, ctx.SetVariable("var1", "updated_value1")) // Update existing

	// Add a node
	node := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "transform")
	node.Start()
	require.NoError(t, ctx.SetVariableWithNode("var3", "from_node", node.ID))
	node.Complete(nil)
	require.NoError(t, exec.AddNodeExecution(node))

	require.NoError(t, exec.Complete(nil))

	// Reconstruct audit trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Verify variable change tracking
	assert.Equal(t, 4, trail.VariableChangeCount) // var1, var2, var1 update, var3

	// Get variable change events
	varEvents := trail.GetVariableChanges()
	assert.Equal(t, 4, len(varEvents))

	// Verify variable initialization vs update messages
	var initCount, updateCount int
	for _, event := range varEvents {
		if strings.Contains(event.Message, "initialized") {
			initCount++
		} else if strings.Contains(event.Message, "updated") {
			updateCount++
		}
	}
	assert.Equal(t, 3, initCount)   // var1, var2, var3
	assert.Equal(t, 1, updateCount) // var1 update
}

// TestReconstructAuditTrail_SkippedNodes tests handling of skipped nodes.
func TestReconstructAuditTrail_SkippedNodes(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("conditional-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add executed node
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("condition"), "condition")
	node1.Start()
	node1.Complete(map[string]interface{}{"result": false})
	require.NoError(t, exec.AddNodeExecution(node1))

	// Add skipped node (conditional branch not taken)
	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("then-branch"), "mcp_tool")
	node2.Skip()
	require.NoError(t, exec.AddNodeExecution(node2))

	require.NoError(t, exec.Complete(nil))

	// Reconstruct audit trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Verify skipped node is tracked
	assert.Equal(t, 2, trail.NodeCount)

	// Find skipped event
	var skippedEvent *AuditEvent
	for i := range trail.Events {
		if trail.Events[i].Type == AuditEventNodeSkipped {
			skippedEvent = &trail.Events[i]
			break
		}
	}
	require.NotNil(t, skippedEvent)
	assert.Equal(t, types.NodeID("then-branch"), skippedEvent.NodeID)
}

// TestFilterEvents tests event filtering functionality.
func TestFilterEvents(t *testing.T) {
	// Create execution with various event types
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add nodes
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node1.Start()
	node1.Complete(nil)
	require.NoError(t, exec.AddNodeExecution(node1))

	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("node-2"), "transform")
	node2.Start()
	node2.Fail(&execution.NodeError{
		Type:    execution.ErrorTypeData,
		Message: "Transform failed",
	})
	require.NoError(t, exec.AddNodeExecution(node2))

	require.NoError(t, exec.Fail(&execution.ExecutionError{
		Type:    execution.ErrorTypeExecution,
		Message: "Execution failed",
		NodeID:  types.NodeID("node-2"),
	}))

	// Reconstruct full trail
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Test 1: Filter by event type (only node events)
	nodeFilter := AuditTrailFilter{
		EventTypes: []AuditEventType{
			AuditEventNodeStarted,
			AuditEventNodeCompleted,
			AuditEventNodeFailed,
		},
	}
	nodeTrail := trail.FilterEvents(nodeFilter)
	assert.Equal(t, 4, len(nodeTrail.Events)) // 2 starts + 1 completed + 1 failed

	// Test 2: Filter by node ID
	node1Filter := AuditTrailFilter{
		NodeID: types.NodeID("node-1"),
	}
	node1Trail := trail.FilterEvents(node1Filter)
	assert.GreaterOrEqual(t, len(node1Trail.Events), 2) // At least start + complete
	for _, event := range node1Trail.Events {
		assert.Equal(t, types.NodeID("node-1"), event.NodeID)
	}

	// Test 3: Filter by time range
	midTime := trail.Events[len(trail.Events)/2].Timestamp
	timeFilter := AuditTrailFilter{
		StartTime: &midTime,
	}
	timeTrail := trail.FilterEvents(timeFilter)
	assert.Less(t, len(timeTrail.Events), len(trail.Events))
	for _, event := range timeTrail.Events {
		assert.False(t, event.Timestamp.Before(midTime))
	}

	// Test 4: Exclude variable changes
	noVarFilter := AuditTrailFilter{
		IncludeVariableChanges: false,
	}
	noVarTrail := trail.FilterEvents(noVarFilter)
	for _, event := range noVarTrail.Events {
		assert.NotEqual(t, AuditEventVariableSet, event.Type)
	}
}

// TestGetEventsByType tests retrieving events by type.
func TestGetEventsByType(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add multiple nodes
	for i := 0; i < 3; i++ {
		node := execution.NewNodeExecution(exec.ID, types.NodeID("node-"+string(rune('1'+i))), "mcp_tool")
		node.Start()
		node.Complete(nil)
		require.NoError(t, exec.AddNodeExecution(node))
	}

	require.NoError(t, exec.Complete(nil))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Get node started events
	startedEvents := trail.GetEventsByType(AuditEventNodeStarted)
	assert.Equal(t, 3, len(startedEvents))

	// Get node completed events
	completedEvents := trail.GetEventsByType(AuditEventNodeCompleted)
	assert.Equal(t, 3, len(completedEvents))

	// Get execution events
	execStarted := trail.GetEventsByType(AuditEventExecutionStarted)
	assert.Equal(t, 1, len(execStarted))

	execCompleted := trail.GetEventsByType(AuditEventExecutionCompleted)
	assert.Equal(t, 1, len(execCompleted))
}

// TestGetEventsForNode tests retrieving all events for a specific node.
func TestGetEventsForNode(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add node with retries
	node := execution.NewNodeExecution(exec.ID, types.NodeID("target-node"), "mcp_tool")
	node.RetryCount = 2
	node.Start()
	node.Complete(map[string]interface{}{"output": "result"})
	require.NoError(t, exec.AddNodeExecution(node))

	// Add another node
	other := execution.NewNodeExecution(exec.ID, types.NodeID("other-node"), "transform")
	other.Start()
	other.Complete(nil)
	require.NoError(t, exec.AddNodeExecution(other))

	require.NoError(t, exec.Complete(nil))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Get events for target node
	nodeEvents := trail.GetEventsForNode(types.NodeID("target-node"))
	assert.GreaterOrEqual(t, len(nodeEvents), 4) // 2 retries + start + complete

	// Verify all events are for the correct node
	for _, event := range nodeEvents {
		assert.Equal(t, types.NodeID("target-node"), event.NodeID)
	}
}

// TestFormatHumanReadable tests human-readable formatting.
func TestFormatHumanReadable(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	node := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node.Start()
	time.Sleep(10 * time.Millisecond)
	node.Complete(map[string]interface{}{"result": "success"})
	require.NoError(t, exec.AddNodeExecution(node))

	require.NoError(t, exec.Complete(map[string]interface{}{"final": "output"}))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Format as human-readable
	output := trail.FormatHumanReadable()

	// Verify output contains key sections
	assert.Contains(t, output, "Execution Audit Trail")
	assert.Contains(t, output, "Workflow:")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "Event Timeline")
	assert.Contains(t, output, "test-workflow")
	assert.Contains(t, output, exec.ID.String())

	// Verify event icons are present
	assert.Contains(t, output, "▶") // Execution started
	assert.Contains(t, output, "→") // Node started
	assert.Contains(t, output, "✓") // Completed

	// Verify timestamps are present
	assert.Contains(t, output, "[")
	assert.Contains(t, output, "+") // Time offset
}

// TestExportJSON tests JSON export functionality.
func TestExportJSON(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("test-workflow"),
		"1.0.0",
		map[string]interface{}{"input": "test"},
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	node := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node.Start()
	node.Complete(map[string]interface{}{"output": "result"})
	require.NoError(t, exec.AddNodeExecution(node))

	require.NoError(t, exec.Complete(map[string]interface{}{"final": "output"}))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Test pretty JSON export
	prettyJSON, err := trail.ExportJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, prettyJSON)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(prettyJSON, &parsed)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, parsed, "execution_id")
	assert.Contains(t, parsed, "workflow_id")
	assert.Contains(t, parsed, "events")
	assert.Contains(t, parsed, "node_count")

	// Test compact JSON export
	compactJSON, err := trail.ExportCompactJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, compactJSON)
	assert.Less(t, len(compactJSON), len(prettyJSON)) // Compact should be smaller

	// Verify compact is valid JSON
	err = json.Unmarshal(compactJSON, &parsed)
	require.NoError(t, err)
}

// TestReconstructAuditTrail_NilExecution tests error handling for nil execution.
func TestReconstructAuditTrail_NilExecution(t *testing.T) {
	trail, err := ReconstructAuditTrail(nil)
	assert.Error(t, err)
	assert.Nil(t, trail)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestReconstructAuditTrail_EmptyExecution tests audit trail for execution with no nodes.
func TestReconstructAuditTrail_EmptyExecution(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("empty-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())
	require.NoError(t, exec.Complete(nil))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	assert.Equal(t, 0, trail.NodeCount)
	assert.Equal(t, 0, trail.ErrorCount)
	assert.GreaterOrEqual(t, len(trail.Events), 2) // At least start and complete
}

// TestReconstructAuditTrail_RunningExecution tests audit trail for in-progress execution.
func TestReconstructAuditTrail_RunningExecution(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("running-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add some completed nodes
	node1 := execution.NewNodeExecution(exec.ID, types.NodeID("node-1"), "mcp_tool")
	node1.Start()
	node1.Complete(nil)
	require.NoError(t, exec.AddNodeExecution(node1))

	// Add a running node (not completed)
	node2 := execution.NewNodeExecution(exec.ID, types.NodeID("node-2"), "mcp_tool")
	node2.Start()
	// Don't complete it
	require.NoError(t, exec.AddNodeExecution(node2))

	// Execution is still running
	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Should have execution started and node events, but no execution completed
	assert.Equal(t, execution.StatusRunning, trail.Status)
	assert.Equal(t, 2, trail.NodeCount)

	// Verify no execution completion event
	completedEvents := trail.GetEventsByType(AuditEventExecutionCompleted)
	assert.Equal(t, 0, len(completedEvents))

	// Verify node1 has completion but node2 doesn't
	node1Events := trail.GetEventsForNode(types.NodeID("node-1"))
	node2Events := trail.GetEventsForNode(types.NodeID("node-2"))

	hasNode1Complete := false
	for _, event := range node1Events {
		if event.Type == AuditEventNodeCompleted {
			hasNode1Complete = true
		}
	}
	assert.True(t, hasNode1Complete)

	hasNode2Complete := false
	for _, event := range node2Events {
		if event.Type == AuditEventNodeCompleted {
			hasNode2Complete = true
		}
	}
	assert.False(t, hasNode2Complete)
}

// TestAuditTrailFilter_ComplexScenario tests filtering with multiple criteria.
func TestAuditTrailFilter_ComplexScenario(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("complex-workflow"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	// Add multiple nodes with different outcomes
	startTime := time.Now()

	for i := 0; i < 5; i++ {
		node := execution.NewNodeExecution(exec.ID, types.NodeID("node-"+string(rune('1'+i))), "mcp_tool")
		node.StartedAt = startTime.Add(time.Duration(i) * time.Second)
		node.Start()

		if i == 2 {
			// Fail middle node
			node.Fail(&execution.NodeError{
				Type:    execution.ErrorTypeExecution,
				Message: "Failed",
			})
		} else {
			node.Complete(nil)
		}
		node.CompletedAt = node.StartedAt.Add(100 * time.Millisecond)
		require.NoError(t, exec.AddNodeExecution(node))
	}

	require.NoError(t, exec.Fail(&execution.ExecutionError{
		Type:    execution.ErrorTypeExecution,
		Message: "Node failed",
		NodeID:  types.NodeID("node-3"),
	}))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Filter 1: Only error events
	errorFilter := AuditTrailFilter{
		EventTypes: []AuditEventType{
			AuditEventNodeFailed,
			AuditEventExecutionFailed,
		},
	}

	errorFiltered := trail.FilterEvents(errorFilter)
	assert.GreaterOrEqual(t, len(errorFiltered.Events), 1) // At least the failed node

	// Verify all are error events
	for _, event := range errorFiltered.Events {
		assert.True(t, event.Type == AuditEventNodeFailed || event.Type == AuditEventExecutionFailed)
	}

	// Filter 2: Only events for node-3 (the failed node)
	nodeFilter := AuditTrailFilter{
		NodeID: types.NodeID("node-3"),
	}

	nodeFiltered := trail.FilterEvents(nodeFilter)
	assert.GreaterOrEqual(t, len(nodeFiltered.Events), 1) // At least node-3 events

	// Verify all are for node-3
	for _, event := range nodeFiltered.Events {
		assert.Equal(t, types.NodeID("node-3"), event.NodeID)
	}
}

// TestEventIconsAndFormatting tests that all event types have proper icons and formatting.
func TestEventIconsAndFormatting(t *testing.T) {
	eventTypes := []AuditEventType{
		AuditEventExecutionStarted,
		AuditEventExecutionCompleted,
		AuditEventExecutionFailed,
		AuditEventExecutionCancelled,
		AuditEventNodeStarted,
		AuditEventNodeCompleted,
		AuditEventNodeFailed,
		AuditEventNodeSkipped,
		AuditEventNodeRetried,
		AuditEventVariableSet,
		AuditEventError,
	}

	for _, eventType := range eventTypes {
		icon := getEventIcon(eventType)
		assert.NotEmpty(t, icon, "event type %s should have an icon", eventType)
	}
}

// TestVariableChangeDetection tests detection of variable value changes.
func TestVariableChangeDetection(t *testing.T) {
	exec, err := execution.NewExecution(
		types.WorkflowID("var-test"),
		"1.0.0",
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, exec.Start())

	ctx := exec.Context

	// Set initial value
	require.NoError(t, ctx.SetVariable("counter", 0))

	// Update multiple times
	for i := 1; i <= 5; i++ {
		require.NoError(t, ctx.SetVariable("counter", i))
	}

	require.NoError(t, exec.Complete(nil))

	trail, err := ReconstructAuditTrail(exec)
	require.NoError(t, err)

	// Should have 6 variable events (1 init + 5 updates)
	varEvents := trail.GetVariableChanges()
	assert.Equal(t, 6, len(varEvents))

	// Verify old/new values are tracked
	for i, event := range varEvents {
		assert.Equal(t, "counter", event.Details["variable_name"])
		if i == 0 {
			// First event: initialization
			assert.Nil(t, event.Details["old_value"])
			assert.Equal(t, i, event.Details["new_value"])
		} else {
			// Subsequent events: updates
			assert.Equal(t, i-1, event.Details["old_value"])
			assert.Equal(t, i, event.Details["new_value"])
		}
	}
}
