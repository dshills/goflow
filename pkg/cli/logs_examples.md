# GoFlow Logs Command - Usage Examples

## Basic Usage

### View Execution Logs

Display all logs for a completed execution:

```bash
$ goflow logs exec-1730822400

Execution Logs: exec-1730822400
Workflow: payment-processing (version 1.0.0)
Status: completed

Summary: 12 events, 4 nodes

12:34:01.123  +0.000s  ▶ Execution started for workflow 'payment-processing' version 1.0.0
12:34:01.234  +0.111s  ▶ Node 'validate-payment' started execution
           Node: validate-payment (transform)
12:34:02.456  +1.333s  ✓ Node 'validate-payment' completed successfully (1.222s)
12:34:02.457  +1.334s  ▶ Node 'charge-card' started execution
           Node: charge-card (mcp_tool)
12:34:03.123  +2.000s  ✓ Node 'charge-card' completed successfully (0.666s)
12:34:03.234  +2.111s  ▶ Node 'send-receipt' started execution
           Node: send-receipt (mcp_tool)
12:34:04.567  +3.444s  ✓ Node 'send-receipt' completed successfully (1.333s)
12:34:04.678  +3.555s  ✓ Execution completed successfully (3.555s)

Completed in 3.555s
```

## Filtering Examples

### Show Only Errors

Perfect for debugging failed executions:

```bash
$ goflow logs exec-1730822401 --type error

Execution Logs: exec-1730822401
Workflow: data-pipeline (version 2.1.0)
Status: failed

Summary: 3 events, 2 nodes, 1 errors

12:45:01.123  +0.000s  ▶ Execution started for workflow 'data-pipeline' version 2.1.0
12:45:05.456  +4.333s  ✗ Node 'validate-schema' failed
           Node: validate-schema (transform)
           Error: JSON schema validation failed: missing required field 'email'
12:45:05.567  +4.444s  ✗ Execution failed (4.444s)
```

### Show Info Events Only

Focus on execution flow without details:

```bash
$ goflow logs exec-1730822402 --type info

Execution Logs: exec-1730822402
Workflow: batch-processor (version 1.5.0)
Status: completed

Summary: 8 events, 5 nodes

12:50:00.000  +0.000s  ▶ Execution started
12:50:01.234  +1.234s  ▶ Node 'fetch-data' started
12:50:03.456  +3.456s  ✓ Node 'fetch-data' completed (2.222s)
12:50:03.567  +3.567s  ▶ Node 'transform-data' started
12:50:05.678  +5.678s  ✓ Node 'transform-data' completed (2.111s)
12:50:05.789  +5.789s  ▶ Node 'save-results' started
12:50:07.890  +7.890s  ✓ Node 'save-results' completed (2.101s)
12:50:08.000  +8.000s  ✓ Execution completed (8.000s)

Completed in 8s
```

### Multiple Filter Types

Combine error and warning events:

```bash
$ goflow logs exec-1730822403 --type error,warning

Execution Logs: exec-1730822403
Workflow: retry-example (version 1.0.0)
Status: completed

Summary: 5 events, 3 nodes, 2 errors

13:00:01.000  +1.000s  ✗ Node 'flaky-api-call' failed
           Node: flaky-api-call (mcp_tool)
           Error: Connection timeout: server did not respond
13:00:02.000  +2.000s  ↻ Node 'flaky-api-call' retry attempt 1
13:00:05.000  +5.000s  ✗ Node 'flaky-api-call' failed
           Node: flaky-api-call (mcp_tool)
           Error: Connection timeout: server did not respond
13:00:06.000  +6.000s  ↻ Node 'flaky-api-call' retry attempt 2
13:00:09.000  +9.000s  ✓ Node 'flaky-api-call' completed successfully (3.000s)
```

## Tail Mode Examples

### Show Last 10 Events

Quick summary of recent activity:

```bash
$ goflow logs exec-1730822404 --tail 10

Execution Logs: exec-1730822404
Workflow: long-workflow (version 1.0.0)
Status: completed

Summary: 10 events, 50 nodes

[... showing last 10 events ...]
13:10:55.678  +295.678s  ✓ Node 'step-48' completed (2.100s)
13:10:55.789  +295.789s  ▶ Node 'step-49' started
13:10:57.890  +297.890s  ✓ Node 'step-49' completed (2.101s)
13:10:58.000  +298.000s  ▶ Node 'step-50' started
13:10:59.123  +299.123s  ✓ Node 'step-50' completed (1.123s)
13:10:59.234  +299.234s  ✓ Execution completed (299.234s)

Completed in 4m59s
```

### Combine Tail with Filter

Last 20 errors:

```bash
$ goflow logs exec-1730822405 --type error --tail 20
```

## Follow Mode (Real-time)

### Follow Running Execution

Watch logs as they happen:

```bash
$ goflow logs exec-running-123 --follow

Execution Logs: exec-running-123 (following...)
Workflow: realtime-processing
Status: running

13:20:00.000  +0.000s  ▶ Execution started for workflow 'realtime-processing'
13:20:01.123  +1.123s  ▶ Node 'initialize' started
13:20:02.234  +2.234s  ✓ Node 'initialize' completed (1.111s)
13:20:02.345  +2.345s  ▶ Node 'process-batch-1' started

[waiting for more events...]

13:20:05.678  +5.678s  ✓ Node 'process-batch-1' completed (3.333s)
13:20:05.789  +5.789s  ▶ Node 'process-batch-2' started

[waiting for more events...]

^C
Received interrupt signal, stopping...
```

### Follow with Filter

Watch only errors in real-time:

```bash
$ goflow logs exec-running-456 --follow --type error

Execution Logs: exec-running-456 (following...)
Workflow: monitoring-pipeline
Status: running

[waiting for more events...]

13:25:15.123  +15.123s  ✗ Node 'validate-input' failed
           Node: validate-input (transform)
           Error: Invalid format: expected JSON, got plain text

[waiting for more events...]

Execution completed with status: failed
Total duration: 15.234s
```

### Auto-completion Detection

Follow mode automatically stops when execution completes:

```bash
$ goflow logs exec-running-789 --follow

[... events stream ...]

Execution completed with status: completed
Total duration: 2m34s
```

## Color and Formatting

### Disable Colors

For logging to files or non-color terminals:

```bash
$ goflow logs exec-123 --no-color > execution.log
```

### With Colors (default)

Terminal output with ANSI colors:
- 🟢 Green: Successful completions
- 🔴 Red: Errors and failures
- 🔵 Blue: Started/running events
- 🟡 Yellow: Warnings, skips, retries
- 🔵 Cyan: Variable changes
- ⚫ Gray: Context information

## Variable Tracking

### Show Variable Changes

Include variable set/update events:

```bash
$ goflow logs exec-1730822406 --show-variables

Execution Logs: exec-1730822406
Workflow: variable-demo (version 1.0.0)
Status: completed

Summary: 15 events, 4 nodes

13:30:00.000  +0.000s  ▶ Execution started
13:30:00.100  +0.100s  ≔ Variable 'input_data' initialized
13:30:00.200  +0.200s  ≔ Variable 'config' initialized
13:30:01.000  +1.000s  ▶ Node 'process' started
13:30:02.000  +2.000s  ≔ Variable 'processed_data' updated
13:30:03.000  +3.000s  ✓ Node 'process' completed (2.000s)
13:30:03.100  +3.100s  ≔ Variable 'output' updated
13:30:03.200  +3.200s  ✓ Execution completed (3.200s)

Completed in 3.2s
```

## Error Analysis Workflows

### Debug Failed Execution

Step 1: See what went wrong:
```bash
$ goflow logs exec-failed --type error
```

Step 2: Get full context:
```bash
$ goflow logs exec-failed --tail 50
```

Step 3: Examine specific node:
```bash
$ goflow logs exec-failed | grep "validate-input"
```

### Monitor Production Execution

Watch critical workflow:
```bash
$ goflow logs exec-prod-001 --follow --type error,warning
```

Alert on completion:
```bash
$ goflow logs exec-prod-001 --follow && echo "Execution completed!" | mail -s "Alert" admin@example.com
```

## Performance Analysis

### Identify Slow Nodes

Look at completion events with durations:

```bash
$ goflow logs exec-perf-test --type node_completed

13:40:01.234  +1.234s  ✓ Node 'fast-step' completed (0.234s)
13:40:05.678  +5.678s  ✓ Node 'slow-step' completed (4.444s)
13:40:06.890  +6.890s  ✓ Node 'medium-step' completed (1.212s)
```

### Timeline Analysis

Full execution timeline:

```bash
$ goflow logs exec-timeline --type node_started,node_completed

13:45:00.000  +0.000s  ▶ Node 'step-1' started
13:45:01.000  +1.000s  ✓ Node 'step-1' completed (1.000s)
13:45:01.100  +1.100s  ▶ Node 'step-2' started
13:45:02.500  +2.500s  ✓ Node 'step-2' completed (1.400s)
13:45:02.600  +2.600s  ▶ Node 'step-3' started
13:45:04.000  +4.000s  ✓ Node 'step-3' completed (1.400s)
```

## Integration Examples

### Pipe to grep

Find specific events:

```bash
$ goflow logs exec-123 --no-color | grep "payment"
```

### Save to File

Keep execution history:

```bash
$ goflow logs exec-123 --no-color > logs/exec-123.log
```

### Watch Multiple Executions

Monitor several executions:

```bash
# Terminal 1
$ goflow logs exec-001 --follow

# Terminal 2
$ goflow logs exec-002 --follow

# Terminal 3
$ goflow logs exec-003 --follow
```

### Automation Scripts

Check execution status:

```bash
#!/bin/bash
EXEC_ID=$1
goflow logs $EXEC_ID --tail 1 --type execution_completed,execution_failed --no-color
if [ $? -eq 0 ]; then
    echo "Execution succeeded"
else
    echo "Execution failed"
fi
```

## Common Patterns

### Quick Status Check
```bash
$ goflow logs exec-123 --tail 1
```

### Error Investigation
```bash
$ goflow logs exec-failed --type error
```

### Performance Review
```bash
$ goflow logs exec-123 --type node_completed
```

### Full Audit Trail
```bash
$ goflow logs exec-123 --show-variables > audit.log
```

### Live Monitoring
```bash
$ goflow logs exec-running --follow --type error,warning
```

## Tips and Best Practices

1. **Use `--tail` for large executions** to avoid overwhelming output
2. **Combine filters** to focus on relevant events
3. **Use `--no-color`** when piping to files or other commands
4. **Follow mode is non-blocking** - you can monitor while execution runs
5. **Ctrl+C gracefully stops follow mode** without killing execution
6. **Variable changes are hidden by default** - use `--show-variables` when debugging state
7. **Error shorthand includes all error types** - comprehensive error view
8. **Timestamps show millisecond precision** - useful for performance analysis

## Error Messages

### Execution Not Found
```bash
$ goflow logs exec-nonexistent
Error: failed to load execution: execution not found: exec-nonexistent
```

### Cannot Follow Completed Execution
```bash
$ goflow logs exec-completed --follow
Error: cannot follow completed execution (status: completed)
Use without --follow to view historical logs
```

### Storage Connection Error
```bash
$ goflow logs exec-123
Error: failed to initialize storage: failed to open database: unable to open database file
```

## Summary

The `goflow logs` command provides comprehensive execution observability with:
- Historical log viewing
- Real-time log streaming
- Flexible filtering
- Colorized output
- Performance analysis
- Error debugging

Use it to monitor, debug, and analyze workflow executions throughout their lifecycle.
