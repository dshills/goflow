package execution

import (
	"sync"
	"testing"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// TestExecutionContextCreation tests creating ExecutionContext with variable store
func TestExecutionContextCreation(t *testing.T) {
	tests := []struct {
		name           string
		initialVars    map[string]interface{}
		wantErr        bool
		validateFields func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name:        "empty context creation",
			initialVars: map[string]interface{}{},
			wantErr:     false,
			validateFields: func(t *testing.T, ctx *execution.ExecutionContext) {
				if ctx.Variables == nil {
					t.Error("Variables map should be initialized")
				}
				if len(ctx.Variables) != 0 {
					t.Errorf("Variables length = %d, want 0", len(ctx.Variables))
				}
			},
		},
		{
			name: "context with initial variables",
			initialVars: map[string]interface{}{
				"input1": "value1",
				"input2": 42,
				"input3": true,
			},
			wantErr: false,
			validateFields: func(t *testing.T, ctx *execution.ExecutionContext) {
				if len(ctx.Variables) != 3 {
					t.Errorf("Variables length = %d, want 3", len(ctx.Variables))
				}
				if ctx.Variables["input1"] != "value1" {
					t.Errorf("Variables[input1] = %v, want value1", ctx.Variables["input1"])
				}
				if ctx.Variables["input2"] != 42 {
					t.Errorf("Variables[input2] = %v, want 42", ctx.Variables["input2"])
				}
				if ctx.Variables["input3"] != true {
					t.Errorf("Variables[input3] = %v, want true", ctx.Variables["input3"])
				}
			},
		},
		{
			name:        "context with nil initial variables",
			initialVars: nil,
			wantErr:     false,
			validateFields: func(t *testing.T, ctx *execution.ExecutionContext) {
				if ctx.Variables == nil {
					t.Error("Variables map should be initialized even with nil input")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(tt.initialVars)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecutionContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFields != nil {
				tt.validateFields(t, ctx)
			}
		})
	}
}

// TestVariableGetSet tests variable get/set operations
func TestVariableGetSet(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.ExecutionContext) error
		validate func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name: "set and get string variable",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.SetVariable("key1", "value1")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				val, exists := ctx.GetVariable("key1")
				if !exists {
					t.Error("Variable key1 should exist")
				}
				if val != "value1" {
					t.Errorf("GetVariable(key1) = %v, want value1", val)
				}
			},
		},
		{
			name: "set and get numeric variable",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.SetVariable("count", 42)
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				val, exists := ctx.GetVariable("count")
				if !exists {
					t.Error("Variable count should exist")
				}
				if val != 42 {
					t.Errorf("GetVariable(count) = %v, want 42", val)
				}
			},
		},
		{
			name: "set and get complex object",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.SetVariable("data", map[string]interface{}{
					"name": "test",
					"age":  30,
				})
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				val, exists := ctx.GetVariable("data")
				if !exists {
					t.Error("Variable data should exist")
				}
				data, ok := val.(map[string]interface{})
				if !ok {
					t.Error("Variable should be map[string]interface{}")
				}
				if data["name"] != "test" {
					t.Errorf("data[name] = %v, want test", data["name"])
				}
			},
		},
		{
			name: "get non-existent variable",
			setup: func(ctx *execution.ExecutionContext) error {
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				val, exists := ctx.GetVariable("nonexistent")
				if exists {
					t.Error("Variable nonexistent should not exist")
				}
				if val != nil {
					t.Errorf("GetVariable(nonexistent) = %v, want nil", val)
				}
			},
		},
		{
			name: "update existing variable",
			setup: func(ctx *execution.ExecutionContext) error {
				if err := ctx.SetVariable("key1", "original"); err != nil {
					return err
				}
				return ctx.SetVariable("key1", "updated")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				val, exists := ctx.GetVariable("key1")
				if !exists {
					t.Error("Variable key1 should exist")
				}
				if val != "updated" {
					t.Errorf("GetVariable(key1) = %v, want updated", val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(nil)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			if err := tt.setup(ctx); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, ctx)
		})
	}
}

// TestVariableScoping tests variable scoping and isolation
func TestVariableScoping(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*execution.ExecutionContext, *execution.ExecutionContext)
		validate func(*testing.T, *execution.ExecutionContext, *execution.ExecutionContext)
	}{
		{
			name: "separate contexts are isolated",
			setup: func() (*execution.ExecutionContext, *execution.ExecutionContext) {
				ctx1, _ := execution.NewExecutionContext(map[string]interface{}{
					"shared": "ctx1-value",
				})
				ctx2, _ := execution.NewExecutionContext(map[string]interface{}{
					"shared": "ctx2-value",
				})
				return ctx1, ctx2
			},
			validate: func(t *testing.T, ctx1, ctx2 *execution.ExecutionContext) {
				val1, _ := ctx1.GetVariable("shared")
				val2, _ := ctx2.GetVariable("shared")

				if val1 == val2 {
					t.Error("Contexts should have isolated variable stores")
				}
				if val1 != "ctx1-value" {
					t.Errorf("ctx1 shared = %v, want ctx1-value", val1)
				}
				if val2 != "ctx2-value" {
					t.Errorf("ctx2 shared = %v, want ctx2-value", val2)
				}
			},
		},
		{
			name: "modifications in one context don't affect another",
			setup: func() (*execution.ExecutionContext, *execution.ExecutionContext) {
				ctx1, _ := execution.NewExecutionContext(map[string]interface{}{
					"counter": 0,
				})
				ctx2, _ := execution.NewExecutionContext(map[string]interface{}{
					"counter": 0,
				})
				ctx1.SetVariable("counter", 10)
				return ctx1, ctx2
			},
			validate: func(t *testing.T, ctx1, ctx2 *execution.ExecutionContext) {
				val1, _ := ctx1.GetVariable("counter")
				val2, _ := ctx2.GetVariable("counter")

				if val1 != 10 {
					t.Errorf("ctx1 counter = %v, want 10", val1)
				}
				if val2 != 0 {
					t.Errorf("ctx2 counter = %v, want 0", val2)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx1, ctx2 := tt.setup()
			tt.validate(t, ctx1, ctx2)
		})
	}
}

// TestContextSnapshots tests context snapshots for audit trail
func TestContextSnapshots(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.ExecutionContext) error
		validate func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name: "setting variable creates snapshot",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.SetVariable("key1", "value1")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) == 0 {
					t.Error("Setting variable should create snapshot")
				}
				if snapshots[0].VariableName != "key1" {
					t.Errorf("Snapshot variable name = %v, want key1", snapshots[0].VariableName)
				}
				if snapshots[0].NewValue != "value1" {
					t.Errorf("Snapshot new value = %v, want value1", snapshots[0].NewValue)
				}
			},
		},
		{
			name: "updating variable creates new snapshot with old value",
			setup: func(ctx *execution.ExecutionContext) error {
				if err := ctx.SetVariable("key1", "original"); err != nil {
					return err
				}
				return ctx.SetVariable("key1", "updated")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) != 2 {
					t.Errorf("Should have 2 snapshots, got %d", len(snapshots))
				}
				// First snapshot: nil -> "original"
				if snapshots[0].OldValue != nil {
					t.Errorf("First snapshot old value = %v, want nil", snapshots[0].OldValue)
				}
				if snapshots[0].NewValue != "original" {
					t.Errorf("First snapshot new value = %v, want original", snapshots[0].NewValue)
				}
				// Second snapshot: "original" -> "updated"
				if snapshots[1].OldValue != "original" {
					t.Errorf("Second snapshot old value = %v, want original", snapshots[1].OldValue)
				}
				if snapshots[1].NewValue != "updated" {
					t.Errorf("Second snapshot new value = %v, want updated", snapshots[1].NewValue)
				}
			},
		},
		{
			name: "snapshots are append-only",
			setup: func(ctx *execution.ExecutionContext) error {
				for i := 0; i < 5; i++ {
					if err := ctx.SetVariable("counter", i); err != nil {
						return err
					}
				}
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) != 5 {
					t.Errorf("Should have 5 snapshots, got %d", len(snapshots))
				}
				// Verify order is maintained
				for i := 0; i < 5; i++ {
					if snapshots[i].NewValue != i {
						t.Errorf("Snapshot[%d] new value = %v, want %d", i, snapshots[i].NewValue, i)
					}
				}
			},
		},
		{
			name: "snapshots record timestamp",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.SetVariable("key1", "value1")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) == 0 {
					t.Fatal("Expected at least one snapshot")
				}
				if snapshots[0].Timestamp.IsZero() {
					t.Error("Snapshot timestamp should be set")
				}
			},
		},
		{
			name: "snapshots record node execution ID",
			setup: func(ctx *execution.ExecutionContext) error {
				nodeExecID := types.NodeExecutionID("node-exec-123")
				return ctx.SetVariableWithNode("key1", "value1", nodeExecID)
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) == 0 {
					t.Fatal("Expected at least one snapshot")
				}
				if snapshots[0].NodeExecutionID != types.NodeExecutionID("node-exec-123") {
					t.Errorf("Snapshot node execution ID = %v, want node-exec-123", snapshots[0].NodeExecutionID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(nil)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			if err := tt.setup(ctx); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, ctx)
		})
	}
}

// TestConcurrentAccess tests concurrent access patterns to context
func TestConcurrentAccess(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.ExecutionContext, *sync.WaitGroup)
		validate func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name: "concurrent writes to different variables",
			setup: func(ctx *execution.ExecutionContext, wg *sync.WaitGroup) {
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						key := string(rune('a' + n))
						ctx.SetVariable(key, n)
					}(i)
				}
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				if len(ctx.Variables) != 10 {
					t.Errorf("Variables length = %d, want 10", len(ctx.Variables))
				}
				for i := 0; i < 10; i++ {
					key := string(rune('a' + i))
					val, exists := ctx.GetVariable(key)
					if !exists {
						t.Errorf("Variable %s should exist", key)
					}
					if val != i {
						t.Errorf("Variable %s = %v, want %d", key, val, i)
					}
				}
			},
		},
		{
			name: "concurrent writes to same variable",
			setup: func(ctx *execution.ExecutionContext, wg *sync.WaitGroup) {
				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						ctx.SetVariable("counter", n)
					}(i)
				}
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				// Should not panic, final value is non-deterministic
				val, exists := ctx.GetVariable("counter")
				if !exists {
					t.Error("Variable counter should exist")
				}
				if val == nil {
					t.Error("Variable counter should have a value")
				}
				// Should have 100 snapshots (one per write)
				snapshots := ctx.GetVariableHistory()
				if len(snapshots) != 100 {
					t.Errorf("Should have 100 snapshots, got %d", len(snapshots))
				}
			},
		},
		{
			name: "concurrent reads and writes",
			setup: func(ctx *execution.ExecutionContext, wg *sync.WaitGroup) {
				// Pre-populate
				ctx.SetVariable("shared", 0)

				// Writers
				for i := 0; i < 50; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						ctx.SetVariable("shared", n)
					}(i)
				}

				// Readers
				for i := 0; i < 50; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						ctx.GetVariable("shared")
					}()
				}
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				// Should not panic, value should exist
				val, exists := ctx.GetVariable("shared")
				if !exists {
					t.Error("Variable shared should exist")
				}
				if val == nil {
					t.Error("Variable shared should have a value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(nil)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			var wg sync.WaitGroup
			tt.setup(ctx, &wg)
			wg.Wait()

			tt.validate(t, ctx)
		})
	}
}

// TestExecutionTrace tests execution trace recording
func TestExecutionTrace(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.ExecutionContext) error
		validate func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name: "record trace entry",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.RecordTrace(types.NodeID("node-1"), "started")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				trace := ctx.GetExecutionTrace()
				if len(trace) != 1 {
					t.Errorf("Trace length = %d, want 1", len(trace))
				}
				if trace[0].NodeID != types.NodeID("node-1") {
					t.Errorf("Trace NodeID = %v, want node-1", trace[0].NodeID)
				}
				if trace[0].Event != "started" {
					t.Errorf("Trace Event = %v, want started", trace[0].Event)
				}
			},
		},
		{
			name: "trace maintains chronological order",
			setup: func(ctx *execution.ExecutionContext) error {
				nodes := []types.NodeID{"start", "node-1", "node-2", "end"}
				for _, nodeID := range nodes {
					if err := ctx.RecordTrace(nodeID, "executed"); err != nil {
						return err
					}
					time.Sleep(1 * time.Millisecond) // Ensure different timestamps
				}
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				trace := ctx.GetExecutionTrace()
				if len(trace) != 4 {
					t.Errorf("Trace length = %d, want 4", len(trace))
				}
				expected := []types.NodeID{"start", "node-1", "node-2", "end"}
				for i, entry := range trace {
					if entry.NodeID != expected[i] {
						t.Errorf("Trace[%d].NodeID = %v, want %v", i, entry.NodeID, expected[i])
					}
				}
			},
		},
		{
			name: "trace entries have timestamps",
			setup: func(ctx *execution.ExecutionContext) error {
				return ctx.RecordTrace(types.NodeID("node-1"), "executed")
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				trace := ctx.GetExecutionTrace()
				if len(trace) == 0 {
					t.Fatal("Expected at least one trace entry")
				}
				if trace[0].Timestamp.IsZero() {
					t.Error("Trace entry timestamp should be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(nil)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			if err := tt.setup(ctx); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, ctx)
		})
	}
}

// TestCurrentNodeTracking tests tracking the current node being executed
func TestCurrentNodeTracking(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*execution.ExecutionContext) error
		validate func(*testing.T, *execution.ExecutionContext)
	}{
		{
			name: "initial current node is nil",
			setup: func(ctx *execution.ExecutionContext) error {
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				if ctx.CurrentNodeID != nil {
					t.Errorf("CurrentNodeID = %v, want nil", ctx.CurrentNodeID)
				}
			},
		},
		{
			name: "set current node",
			setup: func(ctx *execution.ExecutionContext) error {
				nodeID := types.NodeID("node-1")
				ctx.SetCurrentNode(&nodeID)
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				if ctx.CurrentNodeID == nil {
					t.Fatal("CurrentNodeID should not be nil")
				}
				if *ctx.CurrentNodeID != types.NodeID("node-1") {
					t.Errorf("CurrentNodeID = %v, want node-1", *ctx.CurrentNodeID)
				}
			},
		},
		{
			name: "clear current node",
			setup: func(ctx *execution.ExecutionContext) error {
				nodeID := types.NodeID("node-1")
				ctx.SetCurrentNode(&nodeID)
				ctx.SetCurrentNode(nil)
				return nil
			},
			validate: func(t *testing.T, ctx *execution.ExecutionContext) {
				if ctx.CurrentNodeID != nil {
					t.Errorf("CurrentNodeID = %v, want nil", ctx.CurrentNodeID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := execution.NewExecutionContext(nil)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			if err := tt.setup(ctx); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			tt.validate(t, ctx)
		})
	}
}
