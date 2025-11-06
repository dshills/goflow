# Audit Trail Reconstruction

The audit trail system provides comprehensive execution history reconstruction, enabling detailed debugging, compliance auditing, and execution analysis without requiring real-time event streaming.

## Overview

The audit trail reconstructs a complete, chronological view of everything that happened during workflow execution by analyzing execution data stored in the database. This is particularly useful for:

- **Debugging**: Understanding what happened in failed or problematic executions
- **Compliance**: Maintaining detailed records of all workflow operations
- **Analysis**: Identifying performance bottlenecks and execution patterns
- **Reporting**: Generating human-readable execution summaries

## Core Components

### AuditEvent

Represents a single event in the execution timeline:

```go
type AuditEvent struct {
    Timestamp       time.Time              // When the event occurred
    Type            AuditEventType         // Event category
    NodeID          types.NodeID           // Related node (if applicable)
    NodeType        string                 // Type of node
    NodeExecutionID types.NodeExecutionID  // Specific execution instance
    Message         string                 // Human-readable description
    Details         map[string]interface{} // Structured event data
    Duration        *time.Duration         // Operation duration (for completions)
}
```

### AuditEventType

Events are categorized by type:

**Execution Events:**
- `AuditEventExecutionStarted` - Execution initialization
- `AuditEventExecutionCompleted` - Successful completion
- `AuditEventExecutionFailed` - Execution failure
- `AuditEventExecutionCancelled` - User cancellation

**Node Events:**
- `AuditEventNodeStarted` - Node began execution
- `AuditEventNodeCompleted` - Node finished successfully
- `AuditEventNodeFailed` - Node encountered an error
- `AuditEventNodeSkipped` - Node was skipped (conditional)
- `AuditEventNodeRetried` - Node retry attempt

**Data Events:**
- `AuditEventVariableSet` - Variable created or updated

**Error Events:**
- `AuditEventError` - General error event

### AuditTrail

The complete reconstructed execution history:

```go
type AuditTrail struct {
    ExecutionID         types.ExecutionID   // Execution identifier
    WorkflowID          types.WorkflowID    // Workflow identifier
    WorkflowVersion     string              // Workflow version
    Status              execution.Status    // Final status
    StartedAt           time.Time           // Start timestamp
    CompletedAt         time.Time           // Completion timestamp
    Duration            time.Duration       // Total execution time
    Events              []AuditEvent        // Chronological events
    NodeCount           int                 // Total nodes executed
    ErrorCount          int                 // Total errors encountered
    VariableChangeCount int                 // Total variable changes
    RetryCount          int                 // Total retry attempts
    ReturnValue         interface{}         // Final output
}
```

## Usage

### Basic Reconstruction

```go
// Load execution from repository
exec, err := repo.Load(executionID)
if err != nil {
    return err
}

// Reconstruct audit trail
trail, err := execution.ReconstructAuditTrail(exec)
if err != nil {
    return err
}

// Access summary information
fmt.Printf("Nodes executed: %d\n", trail.NodeCount)
fmt.Printf("Errors: %d\n", trail.ErrorCount)
fmt.Printf("Duration: %s\n", trail.Duration)
```

### Filtering Events

Filter events by type, node, or time range:

```go
// Filter to only error events
errorFilter := execution.AuditTrailFilter{
    EventTypes: []execution.AuditEventType{
        execution.AuditEventNodeFailed,
        execution.AuditEventExecutionFailed,
    },
}
errorTrail := trail.FilterEvents(errorFilter)

// Filter to specific node
nodeFilter := execution.AuditTrailFilter{
    NodeID: types.NodeID("problematic-node"),
}
nodeTrail := trail.FilterEvents(nodeFilter)

// Filter by time range
startTime := time.Now().Add(-1 * time.Hour)
endTime := time.Now()
timeFilter := execution.AuditTrailFilter{
    StartTime: &startTime,
    EndTime:   &endTime,
}
recentTrail := trail.FilterEvents(timeFilter)

// Exclude variable changes
noVarFilter := execution.AuditTrailFilter{
    IncludeVariableChanges: false,
}
coreTrail := trail.FilterEvents(noVarFilter)
```

### Querying Events

Retrieve specific event types:

```go
// Get all error events
errorEvents := trail.GetErrorEvents()
for _, event := range errorEvents {
    fmt.Printf("Error at %s: %s\n", event.Timestamp, event.Message)
    if event.Details != nil {
        fmt.Printf("  Type: %v\n", event.Details["error_type"])
        fmt.Printf("  Message: %v\n", event.Details["error_message"])
    }
}

// Get events for a specific node
nodeEvents := trail.GetEventsForNode(types.NodeID("data-transform"))
fmt.Printf("Node executed %d times\n", len(nodeEvents))

// Get variable changes
varChanges := trail.GetVariableChanges()
for _, change := range varChanges {
    varName := change.Details["variable_name"]
    oldValue := change.Details["old_value"]
    newValue := change.Details["new_value"]
    fmt.Printf("%s: %v → %v\n", varName, oldValue, newValue)
}

// Get events by type
startedEvents := trail.GetEventsByType(execution.AuditEventNodeStarted)
fmt.Printf("%d nodes started\n", len(startedEvents))
```

### Human-Readable Output

Generate formatted text reports:

```go
// Format as human-readable text
output := trail.FormatHumanReadable()
fmt.Println(output)
```

Example output:

```
═══════════════════════════════════════════════════════════════
Execution Audit Trail: exec-abc123
═══════════════════════════════════════════════════════════════

Workflow:     user-onboarding (version 1.2.0)
Status:       completed
Started:      2025-11-05T10:30:00Z
Completed:    2025-11-05T10:30:05Z
Duration:     5.234s

Nodes Executed:    5
Errors:            0
Variable Changes:  8
Retries:           0

───────────────────────────────────────────────────────────────
Event Timeline
───────────────────────────────────────────────────────────────

[10:30:00.000] +0s ▶ Execution started for workflow 'user-onboarding' version 1.2.0

[10:30:00.100] +100ms → Node 'fetch-user-data' started execution
        Node: fetch-user-data (mcp_tool)

[10:30:01.234] +1.234s ✓ Node 'fetch-user-data' completed successfully
        Node: fetch-user-data (mcp_tool)
        Duration: 1.134s

[10:30:01.250] +1.250s ≔ Variable 'user_email' initialized

[10:30:01.300] +1.300s → Node 'send-welcome-email' started execution
        Node: send-welcome-email (mcp_tool)

[10:30:02.500] +2.500s ✓ Node 'send-welcome-email' completed successfully
        Node: send-welcome-email (mcp_tool)
        Duration: 1.200s

[10:30:05.234] +5.234s ✓ Execution completed successfully

═══════════════════════════════════════════════════════════════
```

### JSON Export

Export for programmatic processing or storage:

```go
// Pretty JSON (indented, for human viewing)
prettyJSON, err := trail.ExportJSON()
if err != nil {
    return err
}
os.WriteFile("audit-trail.json", prettyJSON, 0644)

// Compact JSON (for transmission or storage)
compactJSON, err := trail.ExportCompactJSON()
if err != nil {
    return err
}

// Parse back from JSON
var loadedTrail execution.AuditTrail
err = json.Unmarshal(compactJSON, &loadedTrail)
```

JSON structure:

```json
{
  "execution_id": "exec-abc123",
  "workflow_id": "user-onboarding",
  "workflow_version": "1.2.0",
  "status": "completed",
  "started_at": "2025-11-05T10:30:00Z",
  "completed_at": "2025-11-05T10:30:05Z",
  "duration": 5234000000,
  "events": [
    {
      "timestamp": "2025-11-05T10:30:00Z",
      "type": "execution_started",
      "message": "Execution started for workflow 'user-onboarding' version 1.2.0",
      "details": {
        "execution_id": "exec-abc123",
        "workflow_id": "user-onboarding",
        "workflow_version": "1.2.0"
      }
    },
    {
      "timestamp": "2025-11-05T10:30:00.100Z",
      "type": "node_started",
      "node_id": "fetch-user-data",
      "node_type": "mcp_tool",
      "node_execution_id": "node-exec-xyz789",
      "message": "Node 'fetch-user-data' started execution",
      "details": {
        "inputs": {
          "user_id": "user-123"
        }
      }
    }
  ],
  "node_count": 5,
  "error_count": 0,
  "variable_change_count": 8,
  "retry_count": 0,
  "return_value": {
    "status": "success"
  }
}
```

## Use Cases

### Debugging Failed Executions

```go
// Load failed execution
exec, _ := repo.Load(failedExecutionID)
trail, _ := execution.ReconstructAuditTrail(exec)

// Find what went wrong
errorEvents := trail.GetErrorEvents()
for _, event := range errorEvents {
    fmt.Printf("Error in %s at %s:\n", event.NodeID, event.Timestamp)
    fmt.Printf("  %s\n", event.Message)

    if errorCtx, ok := event.Details["error_context"].(map[string]interface{}); ok {
        fmt.Printf("  Context: %+v\n", errorCtx)
    }
}

// Check what variables were set before failure
varChanges := trail.GetVariableChanges()
fmt.Printf("\nVariable state before failure:\n")
for _, change := range varChanges {
    if change.Timestamp.Before(trail.CompletedAt) {
        fmt.Printf("  %s = %v\n", change.Details["variable_name"], change.Details["new_value"])
    }
}
```

### Performance Analysis

```go
trail, _ := execution.ReconstructAuditTrail(exec)

// Analyze node execution times
nodeCompletions := trail.GetEventsByType(execution.AuditEventNodeCompleted)
for _, event := range nodeCompletions {
    if event.Duration != nil {
        fmt.Printf("%s: %s\n", event.NodeID, *event.Duration)
    }
}

// Identify slow nodes
var slowNodes []string
for _, event := range nodeCompletions {
    if event.Duration != nil && *event.Duration > 1*time.Second {
        slowNodes = append(slowNodes, string(event.NodeID))
    }
}
fmt.Printf("Slow nodes (>1s): %v\n", slowNodes)
```

### Compliance Reporting

```go
// Generate audit report for compliance
trail, _ := execution.ReconstructAuditTrail(exec)

// Create report document
report := fmt.Sprintf(`
Execution Audit Report
======================

Execution ID: %s
Workflow: %s v%s
Status: %s
Started: %s
Completed: %s
Duration: %s

Operations Performed:
- Nodes Executed: %d
- Variable Changes: %d
- Errors: %d
- Retries: %d

Detailed Timeline:
%s
`,
    trail.ExecutionID,
    trail.WorkflowID,
    trail.WorkflowVersion,
    trail.Status,
    trail.StartedAt,
    trail.CompletedAt,
    trail.Duration,
    trail.NodeCount,
    trail.VariableChangeCount,
    trail.ErrorCount,
    trail.RetryCount,
    trail.FormatHumanReadable(),
)

// Save report
os.WriteFile("audit-report.txt", []byte(report), 0644)

// Also export JSON for archival
jsonData, _ := trail.ExportJSON()
os.WriteFile("audit-data.json", jsonData, 0644)
```

### Retry Analysis

```go
trail, _ := execution.ReconstructAuditTrail(exec)

// Find nodes that required retries
retryEvents := trail.GetEventsByType(execution.AuditEventNodeRetried)

retryStats := make(map[types.NodeID]int)
for _, event := range retryEvents {
    retryStats[event.NodeID]++
}

fmt.Printf("Retry statistics:\n")
for nodeID, count := range retryStats {
    fmt.Printf("  %s: %d retries\n", nodeID, count)
}

// Identify problematic nodes (>2 retries)
for nodeID, count := range retryStats {
    if count > 2 {
        fmt.Printf("WARNING: %s required %d retries\n", nodeID, count)

        // Get all events for this node
        nodeEvents := trail.GetEventsForNode(nodeID)
        for _, event := range nodeEvents {
            fmt.Printf("  [%s] %s\n", event.Type, event.Message)
        }
    }
}
```

## Integration with Storage

The audit trail system integrates seamlessly with the execution repository:

```go
// Repository method to load execution with full details
func (s *Service) GetExecutionAuditTrail(executionID types.ExecutionID) (*execution.AuditTrail, error) {
    // Load execution from database
    exec, err := s.repo.Load(executionID)
    if err != nil {
        return nil, fmt.Errorf("failed to load execution: %w", err)
    }

    // Reconstruct audit trail
    trail, err := execution.ReconstructAuditTrail(exec)
    if err != nil {
        return nil, fmt.Errorf("failed to reconstruct audit trail: %w", err)
    }

    return trail, nil
}
```

## Performance Considerations

- **Reconstruction Cost**: Audit trail reconstruction is O(n) where n is the number of events (node executions + variable changes). For typical workflows (<100 nodes), this is <10ms.

- **Memory Usage**: Each event is ~200-500 bytes. A 100-node execution with 50 variable changes = ~30KB in memory.

- **Filtering**: Filtering is done in-memory after reconstruction. For very large executions (>1000 nodes), consider filtering at the query level instead.

- **Export Size**: JSON export is roughly 2-3x the in-memory size due to formatting. Use compact JSON for transmission.

## Best Practices

1. **Filter Early**: When you only need specific events, use filters to reduce memory and processing:
   ```go
   // Instead of reconstructing everything and filtering later
   trail, _ := execution.ReconstructAuditTrail(exec)
   errorEvents := trail.GetErrorEvents()

   // Reconstruct once, then filter
   trail, _ := execution.ReconstructAuditTrail(exec)
   filtered := trail.FilterEvents(errorFilter)
   ```

2. **Use Appropriate Format**:
   - Human-readable for debugging and manual inspection
   - JSON for storage, transmission, and programmatic processing

3. **Cache When Appropriate**: For completed executions, the audit trail is immutable. Cache it to avoid repeated reconstruction.

4. **Variable Changes**: For workflows with many variable updates, consider excluding them for performance-critical queries:
   ```go
   filter := execution.AuditTrailFilter{
       IncludeVariableChanges: false,
   }
   ```

5. **Time-based Queries**: When debugging time-sensitive issues, use time filters to focus on specific periods:
   ```go
   // Only events in the critical window
   startTime := problemStartTime.Add(-30 * time.Second)
   endTime := problemStartTime.Add(30 * time.Second)
   filter := execution.AuditTrailFilter{
       StartTime: &startTime,
       EndTime:   &endTime,
   }
   ```

## Event Icons

The human-readable format uses Unicode icons for visual clarity:

- ▶ Execution started
- ✓ Successful completion
- ✗ Failure
- ⊗ Cancellation
- → Node started
- ⊘ Node skipped
- ↻ Retry attempt
- ≔ Variable assignment
- ⚠ Error/Warning

## See Also

- `pkg/domain/execution/execution.go` - Execution domain model
- `pkg/domain/execution/node_execution.go` - Node execution records
- `pkg/domain/execution/variable_snapshot.go` - Variable change tracking
- `pkg/storage/sqlite.go` - Execution storage implementation
- `pkg/execution/error.go` - Enhanced error context
