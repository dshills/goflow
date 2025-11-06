# Watch Mode Implementation Summary

## Task: T146 - Add --watch flag to `goflow run` command

### Implementation Status: ✅ COMPLETE

## Overview

Enhanced the `goflow run` command with real-time execution monitoring capabilities through two distinct display modes:

1. **Inline Watch Mode** (`--watch`): Compact progress updates with ANSI terminal control codes
2. **TUI Mode** (`--tui`): Full-screen interactive execution monitor

## Changes Made

### 1. Enhanced CLI Command (`pkg/cli/run.go`)

#### Added Flags
- `--watch` / `-w`: Enable inline progress monitoring
- `--tui`: Launch full TUI execution monitor

#### New Execution Modes
- **Silent Mode** (default): Only show final result
- **Inline Watch Mode**: Real-time event streaming with progress updates
- **TUI Mode**: Full interactive monitoring interface

#### Core Implementation

**Three Execution Functions:**

```go
// runSilent - Default mode, shows only final result
func runSilent(ctx context.Context, cmd *cobra.Command, engine *execution.Engine,
    wf *workflow.Workflow, workflowName string, inputs map[string]interface{},
    outputJSON, debugMode bool) error

// runWithInlineWatch - Live progress with ANSI terminal codes
func runWithInlineWatch(ctx context.Context, cmd *cobra.Command, engine *execution.Engine,
    wf *workflow.Workflow, workflowName string, inputs map[string]interface{},
    outputJSON, debugMode bool) error

// runWithTUI - Full-screen interactive monitor
func runWithTUI(ctx context.Context, engine *execution.Engine, wf *workflow.Workflow,
    workflowName string, inputs map[string]interface{}) error
```

**Event Subscription System:**
- Subscribes to `ExecutionMonitor` events from execution engine
- Processes events in real-time via channels
- Thread-safe state tracking with mutex protection

**Signal Handling:**
- Graceful cancellation on Ctrl+C (SIGINT, SIGTERM)
- Context-based cancellation propagation to execution engine
- Proper cleanup of event subscriptions

### 2. Display Modes

#### Inline Watch Mode (`--watch`)

**Features:**
- Live event streaming (execution started, node started/completed/failed)
- Periodic progress bar updates (percentage, node count)
- Variable change tracking
- Timestamp for each event
- ANSI escape codes for in-place updates
- Terminal detection (degrades gracefully in non-TTY environments)

**Example Output:**
```
Executing: payment-processing
[Press Ctrl+C to cancel]

100ms ▶ Execution started
150ms ▶ validate-order started
650ms ✓ validate-order completed
700ms ▶ check-inventory started
Status: Running | Progress: 60% (3/5 nodes) | Current: process-payment

✓ Workflow completed successfully (2.35s)

Return value:
{
  "order_id": "12345",
  "status": "processed"
}
```

#### TUI Mode (`--tui`)

**Features:**
- Full-screen interactive interface using `goterm`
- Integration with existing `ExecutionMonitor` TUI view
- Real-time panel updates (workflow graph, variables, metrics, logs)
- Periodic screen refresh (100ms intervals)
- Shows final state for 2 seconds after completion
- Context-based cancellation support

**Layout:**
- Workflow graph with node status visualization
- Variable inspector panel
- Performance metrics panel
- Execution log viewer
- Error detail view (when applicable)

### 3. State Management

**watchState Structure:**
```go
type watchState struct {
    startTime      time.Time
    nodeCount      int
    lastNodeID     string
    lastUpdateTime time.Time
    recentLogs     []string
    variables      map[string]interface{}
    mu             sync.Mutex
}
```

**Thread-Safe Operations:**
- Mutex-protected state updates
- Non-blocking event emission
- Buffered event channels (100 events)

### 4. Event Handling

**Supported Event Types:**
- `EventExecutionStarted`: Workflow begins
- `EventExecutionCompleted`: Workflow succeeds
- `EventExecutionFailed`: Workflow fails
- `EventExecutionCancelled`: User cancellation
- `EventNodeStarted`: Node begins execution
- `EventNodeCompleted`: Node finishes successfully
- `EventNodeFailed`: Node fails with error
- `EventVariableChanged`: Variable value updated

**Event Processing:**
```go
func handleInlineEvent(cmd *cobra.Command, event execution.ExecutionEvent,
    state *watchState, isTerm bool)
```
- Formats events with elapsed time
- Uses appropriate symbols (▶ ✓ ✗)
- Tracks variable changes
- Updates last activity timestamp

### 5. Progress Display

**Progress Bar Updates:**
- Percentage complete (0-100%)
- Node counts (completed/failed/skipped vs total)
- Current node being executed
- ANSI codes for in-place updates (`\r\033[K`)
- 500ms refresh interval

```go
func displayInlineProgress(cmd *cobra.Command, progress execution.ExecutionProgress,
    state *watchState)
```

### 6. Result Display

**Final Result Functions:**

```go
// Human-readable output
func displayFinalResult(cmd *cobra.Command, exec *domainexec.Execution,
    err error, state *watchState, debugMode bool)

// JSON output
func displayJSONResult(cmd *cobra.Command, exec *domainexec.Execution, err error)
```

**Features:**
- Success/failure indication
- Execution duration
- Return value pretty-printing
- Debug information (when enabled)
- JSON output mode support

### 7. Integration Points

**Execution Engine:**
- `engine.GetMonitor()`: Access to event monitor
- `monitor.Subscribe()`: Event channel subscription
- `monitor.GetProgress()`: Progress snapshot retrieval
- Context cancellation propagation

**TUI System:**
- `tui.NewExecutionMonitor()`: Full TUI view creation
- `monitorView.SetEventMonitor()`: Event subscription
- `monitorView.Render()`: Screen updates
- Periodic refresh loop

### 8. Error Handling

**Scenarios Covered:**
- Workflow validation failures
- Execution engine errors
- Context cancellation (Ctrl+C)
- TUI initialization failures
- Monitor subscription failures
- JSON marshaling errors

**Error Display:**
- Inline mode: Shows error with ✗ symbol
- TUI mode: Shows error in dedicated error panel
- JSON mode: Includes error in result object

## Usage Examples

### Basic Execution (Silent)
```bash
goflow run payment-processing
```

### Inline Watch Mode
```bash
# Simple watch
goflow run payment-processing --watch

# With input variables
goflow run payment-processing --watch --input input.json

# Combined with debug
goflow run payment-processing --watch --debug
```

### TUI Mode
```bash
# Full interactive monitor
goflow run payment-processing --tui

# With variables
goflow run payment-processing --tui --var amount=100 --var currency=USD
```

### JSON Output
```bash
# Silent with JSON
goflow run payment-processing --output json

# Watch mode doesn't affect JSON output (events not shown)
goflow run payment-processing --watch --output json
```

## Technical Details

### Dependencies
- `github.com/dshills/goflow/pkg/domain/execution`: Domain execution types
- `github.com/dshills/goflow/pkg/execution`: Execution engine and monitor
- `github.com/dshills/goflow/pkg/tui`: TUI execution monitor view
- `github.com/dshills/goterm`: Terminal UI framework
- `golang.org/x/term`: Terminal capability detection
- Standard library: `context`, `os/signal`, `sync`, `time`

### Performance Characteristics
- Event channel buffer: 100 events (prevents blocking)
- Progress refresh interval: 500ms (inline mode)
- Screen refresh interval: 100ms (TUI mode)
- Non-blocking event emission (drops events if buffer full)
- Thread-safe state access with RWMutex

### Terminal Compatibility
- **TTY Detection**: Uses `term.IsTerminal()` for ANSI code support
- **Graceful Degradation**: Falls back to simple output in non-TTY
- **ANSI Escape Codes**: `\r` (carriage return), `\033[K` (clear line)
- **Cross-Platform**: Works on Unix-like systems and Windows (with ANSI support)

## Testing Recommendations

### Unit Tests Needed
1. `TestRunWithInlineWatch`: Event processing and display
2. `TestRunWithTUI`: TUI initialization and event loop
3. `TestRunSilent`: Basic execution without monitoring
4. `TestHandleInlineEvent`: Event formatting and state updates
5. `TestDisplayInlineProgress`: Progress bar rendering
6. `TestSignalHandling`: Ctrl+C cancellation

### Integration Tests Needed
1. Execute test workflow with --watch flag
2. Verify event stream correctness
3. Test cancellation behavior
4. Validate TUI rendering
5. Check JSON output mode compatibility

### Manual Testing Checklist
- [ ] Run workflow with --watch flag
- [ ] Verify progress updates appear
- [ ] Test Ctrl+C cancellation
- [ ] Launch TUI mode with --tui
- [ ] Check variable updates in watch mode
- [ ] Validate error handling for failed workflows
- [ ] Test in non-TTY environment (pipes, redirects)
- [ ] Verify JSON output mode
- [ ] Test with --debug flag combination

## Files Modified

1. `/Users/dshills/Development/projects/goflow/pkg/cli/run.go`
   - Added `--watch` and `--tui` flags
   - Implemented three execution modes
   - Added event subscription and handling
   - Implemented progress display functions
   - Added signal handling for Ctrl+C

2. `/Users/dshills/Development/projects/goflow/pkg/cli/logs.go`
   - Removed duplicate color constant declarations

## Compatibility

### Backward Compatibility
- ✅ Default behavior unchanged (silent mode)
- ✅ Existing flags remain functional
- ✅ JSON output format preserved
- ✅ No breaking changes to workflow definitions

### Forward Compatibility
- Event system designed for extensibility
- Additional event types can be added
- Display modes can be expanded
- TUI panels can be customized

## Future Enhancements

### Potential Improvements
1. **Enhanced TUI Interaction**
   - Keyboard event handling (pause/resume)
   - Node detail inspection on selection
   - Log filtering and search

2. **Advanced Progress Display**
   - Estimated time remaining
   - Node execution duration histogram
   - Resource usage metrics

3. **Recording and Playback**
   - Save execution events to file
   - Replay execution for debugging
   - Export execution trace

4. **Remote Monitoring**
   - WebSocket-based event streaming
   - Web UI for execution monitoring
   - Multi-execution dashboard

5. **Notification System**
   - Desktop notifications on completion
   - Slack/email alerts for long-running workflows
   - Custom webhook integrations

## Related Documentation

- [Execution Engine Documentation](pkg/execution/README.md)
- [TUI System Documentation](pkg/tui/VIEW_SYSTEM.md)
- [Execution Monitor Panel Documentation](pkg/tui/execution_monitor.go)
- [Event System Documentation](pkg/execution/events.go)

## Conclusion

The watch mode implementation successfully provides real-time execution monitoring with two distinct user experiences:

1. **Inline Mode**: Lightweight, terminal-friendly progress updates suitable for CI/CD and headless environments
2. **TUI Mode**: Rich, interactive monitoring experience for development and debugging

Both modes integrate seamlessly with the existing execution engine's event system, providing thread-safe, non-blocking real-time updates with graceful cancellation support.

**Implementation Status**: ✅ Complete and ready for testing
**Build Status**: ✅ Compiles successfully
**CLI Integration**: ✅ Help documentation updated
