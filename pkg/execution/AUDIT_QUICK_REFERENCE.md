# Audit Trail Quick Reference

## Basic Usage

### Reconstruct Audit Trail
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

// Access summary
fmt.Printf("Nodes: %d, Errors: %d, Duration: %s\n",
    trail.NodeCount, trail.ErrorCount, trail.Duration)
```

### Filter Events
```go
// By event type
errorFilter := execution.AuditTrailFilter{
    EventTypes: []execution.AuditEventType{
        execution.AuditEventNodeFailed,
        execution.AuditEventExecutionFailed,
    },
}
errorTrail := trail.FilterEvents(errorFilter)

// By node
nodeFilter := execution.AuditTrailFilter{
    NodeID: types.NodeID("my-node"),
}
nodeTrail := trail.FilterEvents(nodeFilter)

// By time range
startTime := time.Now().Add(-1 * time.Hour)
timeFilter := execution.AuditTrailFilter{
    StartTime: &startTime,
}
recentTrail := trail.FilterEvents(timeFilter)

// Exclude variables
noVarFilter := execution.AuditTrailFilter{
    IncludeVariableChanges: false,
}
coreTrail := trail.FilterEvents(noVarFilter)

// Combine filters
complexFilter := execution.AuditTrailFilter{
    EventTypes:             []execution.AuditEventType{execution.AuditEventNodeFailed},
    NodeID:                 types.NodeID("node-1"),
    IncludeVariableChanges: false,
}
```

### Query Events
```go
// By type
startedEvents := trail.GetEventsByType(execution.AuditEventNodeStarted)

// For specific node
nodeEvents := trail.GetEventsForNode(types.NodeID("node-1"))

// All errors
errorEvents := trail.GetErrorEvents()

// All variable changes
varChanges := trail.GetVariableChanges()
```

### Output Formats
```go
// Human-readable text
output := trail.FormatHumanReadable()
fmt.Println(output)

// Pretty JSON
prettyJSON, _ := trail.ExportJSON()
os.WriteFile("audit.json", prettyJSON, 0644)

// Compact JSON
compactJSON, _ := trail.ExportCompactJSON()
```

## Event Types

```go
// Execution events
AuditEventExecutionStarted
AuditEventExecutionCompleted
AuditEventExecutionFailed
AuditEventExecutionCancelled

// Node events
AuditEventNodeStarted
AuditEventNodeCompleted
AuditEventNodeFailed
AuditEventNodeSkipped
AuditEventNodeRetried

// Data events
AuditEventVariableSet

// Error events
AuditEventError
```

## Common Patterns

### Debug Failed Execution
```go
trail, _ := execution.ReconstructAuditTrail(exec)
errorEvents := trail.GetErrorEvents()

for _, event := range errorEvents {
    fmt.Printf("Error in %s at %s:\n", event.NodeID, event.Timestamp)
    fmt.Printf("  %s\n", event.Message)
    fmt.Printf("  Type: %v\n", event.Details["error_type"])
    fmt.Printf("  Message: %v\n", event.Details["error_message"])
}
```

### Analyze Performance
```go
completions := trail.GetEventsByType(execution.AuditEventNodeCompleted)

for _, event := range completions {
    if event.Duration != nil {
        fmt.Printf("%s: %s\n", event.NodeID, *event.Duration)
    }
}
```

### Track Variable Changes
```go
varChanges := trail.GetVariableChanges()

for _, change := range varChanges {
    name := change.Details["variable_name"]
    old := change.Details["old_value"]
    new := change.Details["new_value"]
    fmt.Printf("%s: %v → %v\n", name, old, new)
}
```

### Identify Retry Issues
```go
retryEvents := trail.GetEventsByType(execution.AuditEventNodeRetried)

retryStats := make(map[types.NodeID]int)
for _, event := range retryEvents {
    retryStats[event.NodeID]++
}

for nodeID, count := range retryStats {
    if count > 2 {
        fmt.Printf("WARNING: %s had %d retries\n", nodeID, count)
    }
}
```

### Generate Compliance Report
```go
trail, _ := execution.ReconstructAuditTrail(exec)

// Text report
report := trail.FormatHumanReadable()
os.WriteFile("audit-report.txt", []byte(report), 0644)

// JSON archive
jsonData, _ := trail.ExportJSON()
os.WriteFile("audit-data.json", jsonData, 0644)
```

## Event Structure

```go
type AuditEvent struct {
    Timestamp       time.Time              // When it occurred
    Type            AuditEventType         // Event category
    NodeID          types.NodeID           // Related node
    NodeType        string                 // Node type (mcp_tool, transform, etc.)
    NodeExecutionID types.NodeExecutionID  // Execution instance
    Message         string                 // Human-readable description
    Details         map[string]interface{} // Structured data
    Duration        *time.Duration         // Operation duration
}
```

## AuditTrail Structure

```go
type AuditTrail struct {
    ExecutionID         types.ExecutionID   // Execution ID
    WorkflowID          types.WorkflowID    // Workflow ID
    WorkflowVersion     string              // Workflow version
    Status              execution.Status    // Final status
    StartedAt           time.Time           // Start time
    CompletedAt         time.Time           // End time
    Duration            time.Duration       // Total time
    Events              []AuditEvent        // Chronological events
    NodeCount           int                 // Nodes executed
    ErrorCount          int                 // Errors encountered
    VariableChangeCount int                 // Variable changes
    RetryCount          int                 // Retry attempts
    ReturnValue         interface{}         // Final output
}
```

## Filter Options

```go
type AuditTrailFilter struct {
    EventTypes             []AuditEventType // Event type filter
    NodeID                 types.NodeID     // Node filter
    IncludeVariableChanges bool             // Include variables
    StartTime              *time.Time       // Start time
    EndTime                *time.Time       // End time
}
```

## Tips

- **Filter early**: Apply filters to reduce memory usage
- **Use appropriate format**: Text for debugging, JSON for storage
- **Cache completed executions**: Audit trails are immutable for completed executions
- **Exclude variables**: For performance-critical queries with many variable changes
- **Time-based filtering**: Focus on specific execution windows for debugging
