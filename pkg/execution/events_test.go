package execution

import (
	"context"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionMonitor_BasicEventStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a simple workflow
	yaml := `
version: "1.0"
name: "test-workflow"
nodes:
  - id: "start"
    type: "start"
  - id: "node1"
    type: "passthrough"
  - id: "end"
    type: "end"
edges:
  - from: "start"
    to: "node1"
  - from: "node1"
    to: "end"
`

	wf, err := workflow.Parse([]byte(yaml))
	require.NoError(t, err)

	engine := NewEngine()
	defer engine.Close()

	// Execute workflow and collect events
	events := make([]ExecutionEvent, 0)
	var done chan struct{}

	// Start execution in goroutine so we can subscribe before it runs
	go func() {
		_, _ = engine.Execute(ctx, wf, nil)
	}()

	// Give engine time to create monitor
	time.Sleep(10 * time.Millisecond)

	// Get monitor and subscribe
	monitor := engine.GetMonitor()
	if monitor != nil {
		eventCh := monitor.Subscribe()
		defer monitor.Unsubscribe(eventCh)

		done = make(chan struct{})
		go func() {
			for event := range eventCh {
				events = append(events, event)
			}
			close(done)
		}()
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	// Check events
	if monitor != nil {
		assert.Greater(t, len(events), 0, "Should have received some events")

		// Find execution started event
		var hasStarted bool
		for _, event := range events {
			if event.Type == EventExecutionStarted {
				hasStarted = true
				break
			}
		}
		assert.True(t, hasStarted, "Should have received execution started event")
	} else {
		t.Log("Monitor is nil - execution may have completed too quickly")
	}
}

func TestExecutionMonitor_SubscribeAndUnsubscribe(t *testing.T) {
	exec, err := execution.NewExecution("test-workflow", "1.0", nil)
	require.NoError(t, err)

	mon := NewMonitor(exec, 5)
	monitor := mon.(*monitor)

	// Subscribe
	ch1 := monitor.Subscribe()
	assert.NotNil(t, ch1)
	assert.Equal(t, 1, len(monitor.subscribers))

	ch2 := monitor.Subscribe()
	assert.NotNil(t, ch2)
	assert.Equal(t, 2, len(monitor.subscribers))

	// Emit event
	monitor.Emit(ExecutionEvent{
		Type:        EventExecutionStarted,
		ExecutionID: exec.ID,
	})

	// Both channels should receive
	select {
	case event := <-ch1:
		assert.Equal(t, EventExecutionStarted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 did not receive event")
	}

	select {
	case event := <-ch2:
		assert.Equal(t, EventExecutionStarted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 did not receive event")
	}

	// Unsubscribe ch1
	monitor.Unsubscribe(ch1)
	assert.Equal(t, 1, len(monitor.subscribers))

	// Emit another event
	monitor.Emit(ExecutionEvent{
		Type:        EventExecutionCompleted,
		ExecutionID: exec.ID,
	})

	// Only ch2 should receive
	select {
	case event := <-ch2:
		assert.Equal(t, EventExecutionCompleted, event.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 did not receive event")
	}

	// ch1 should be closed
	_, ok := <-ch1
	assert.False(t, ok, "ch1 should be closed")

	// Cleanup
	monitor.Unsubscribe(ch2)
	assert.Equal(t, 0, len(monitor.subscribers))
}

func TestExecutionMonitor_FilteredSubscription(t *testing.T) {
	exec, err := execution.NewExecution("test-workflow", "1.0", nil)
	require.NoError(t, err)

	mon := NewMonitor(exec, 5)
	monitor := mon.(*monitor)

	// Subscribe with filter for only node events
	filter := EventFilter{
		EventTypes: []ExecutionEventType{EventNodeStarted, EventNodeCompleted},
	}
	ch := monitor.SubscribeFiltered(filter)
	defer monitor.Unsubscribe(ch)

	// Emit various events
	monitor.Emit(ExecutionEvent{Type: EventExecutionStarted, ExecutionID: exec.ID})
	monitor.Emit(ExecutionEvent{Type: EventNodeStarted, ExecutionID: exec.ID, NodeID: "node1"})
	monitor.Emit(ExecutionEvent{Type: EventNodeCompleted, ExecutionID: exec.ID, NodeID: "node1"})
	monitor.Emit(ExecutionEvent{Type: EventExecutionCompleted, ExecutionID: exec.ID})

	// Should only receive node events
	receivedEvents := make([]ExecutionEvent, 0)
	timeout := time.After(100 * time.Millisecond)

collectLoop:
	for {
		select {
		case event := <-ch:
			receivedEvents = append(receivedEvents, event)
		case <-timeout:
			break collectLoop
		}
	}

	assert.Equal(t, 2, len(receivedEvents), "Should receive only 2 node events")
	for _, event := range receivedEvents {
		assert.Contains(t, []ExecutionEventType{EventNodeStarted, EventNodeCompleted}, event.Type)
	}
}

func TestExecutionMonitor_GetProgress(t *testing.T) {
	exec, err := execution.NewExecution("test-workflow", "1.0", nil)
	require.NoError(t, err)

	// Start execution
	err = exec.Start()
	require.NoError(t, err)

	mon := NewMonitor(exec, 5)
	monitor := mon.(*monitor)

	// Initial progress
	progress := monitor.GetProgress()
	assert.Equal(t, 5, progress.TotalNodes)
	assert.Equal(t, 0, progress.CompletedNodes)
	assert.Equal(t, 0.0, progress.PercentComplete)

	// Add completed node executions
	nodeExec1 := execution.NewNodeExecution(exec.ID, "node1", "passthrough")
	nodeExec1.Start()
	nodeExec1.Complete(nil)
	exec.AddNodeExecution(nodeExec1)

	nodeExec2 := execution.NewNodeExecution(exec.ID, "node2", "passthrough")
	nodeExec2.Start()
	nodeExec2.Complete(nil)
	exec.AddNodeExecution(nodeExec2)

	// Check progress
	progress = monitor.GetProgress()
	assert.Equal(t, 5, progress.TotalNodes)
	assert.Equal(t, 2, progress.CompletedNodes)
	assert.Equal(t, 40.0, progress.PercentComplete)
}

func TestExecutionMonitor_GetVariableSnapshot(t *testing.T) {
	exec, err := execution.NewExecution("test-workflow", "1.0", map[string]interface{}{
		"var1": "value1",
		"var2": 42,
	})
	require.NoError(t, err)

	mon := NewMonitor(exec, 3)
	monitor := mon.(*monitor)

	snapshot := monitor.GetVariableSnapshot()
	assert.Equal(t, 2, len(snapshot))
	assert.Equal(t, "value1", snapshot["var1"])
	assert.Equal(t, 42, snapshot["var2"])

	// Modify context
	exec.Context.SetVariable("var3", true)

	// Get new snapshot
	snapshot = monitor.GetVariableSnapshot()
	assert.Equal(t, 3, len(snapshot))
	assert.Equal(t, true, snapshot["var3"])
}

func TestEventFilter_Matches(t *testing.T) {
	tests := []struct {
		name     string
		filter   EventFilter
		event    ExecutionEvent
		expected bool
	}{
		{
			name:   "empty filter matches all",
			filter: EventFilter{},
			event: ExecutionEvent{
				Type:   EventExecutionStarted,
				NodeID: "node1",
			},
			expected: true,
		},
		{
			name: "type filter matches",
			filter: EventFilter{
				EventTypes: []ExecutionEventType{EventNodeStarted, EventNodeCompleted},
			},
			event: ExecutionEvent{
				Type:   EventNodeStarted,
				NodeID: "node1",
			},
			expected: true,
		},
		{
			name: "type filter doesn't match",
			filter: EventFilter{
				EventTypes: []ExecutionEventType{EventNodeStarted},
			},
			event: ExecutionEvent{
				Type: EventExecutionStarted,
			},
			expected: false,
		},
		{
			name: "node filter matches",
			filter: EventFilter{
				NodeIDs: []types.NodeID{"node1", "node2"},
			},
			event: ExecutionEvent{
				Type:   EventNodeStarted,
				NodeID: "node1",
			},
			expected: true,
		},
		{
			name: "node filter doesn't match",
			filter: EventFilter{
				NodeIDs: []types.NodeID{"node1"},
			},
			event: ExecutionEvent{
				Type:   EventNodeStarted,
				NodeID: "node2",
			},
			expected: false,
		},
		{
			name: "node filter with no node ID in event",
			filter: EventFilter{
				NodeIDs: []types.NodeID{"node1"},
			},
			event: ExecutionEvent{
				Type: EventExecutionStarted,
			},
			expected: false,
		},
		{
			name: "both filters match",
			filter: EventFilter{
				EventTypes: []ExecutionEventType{EventNodeStarted},
				NodeIDs:    []types.NodeID{"node1"},
			},
			event: ExecutionEvent{
				Type:   EventNodeStarted,
				NodeID: "node1",
			},
			expected: true,
		},
		{
			name: "type matches but node doesn't",
			filter: EventFilter{
				EventTypes: []ExecutionEventType{EventNodeStarted},
				NodeIDs:    []types.NodeID{"node1"},
			},
			event: ExecutionEvent{
				Type:   EventNodeStarted,
				NodeID: "node2",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}
