package execution

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// AuditEventType categorizes different types of audit events.
type AuditEventType string

const (
	// AuditEventExecutionStarted indicates the execution was initialized and started
	AuditEventExecutionStarted AuditEventType = "execution_started"
	// AuditEventExecutionCompleted indicates the execution finished successfully
	AuditEventExecutionCompleted AuditEventType = "execution_completed"
	// AuditEventExecutionFailed indicates the execution encountered an error
	AuditEventExecutionFailed AuditEventType = "execution_failed"
	// AuditEventExecutionCancelled indicates the execution was cancelled
	AuditEventExecutionCancelled AuditEventType = "execution_cancelled"

	// AuditEventNodeStarted indicates a node began execution
	AuditEventNodeStarted AuditEventType = "node_started"
	// AuditEventNodeCompleted indicates a node finished successfully
	AuditEventNodeCompleted AuditEventType = "node_completed"
	// AuditEventNodeFailed indicates a node encountered an error
	AuditEventNodeFailed AuditEventType = "node_failed"
	// AuditEventNodeSkipped indicates a node was skipped (e.g., conditional branch)
	AuditEventNodeSkipped AuditEventType = "node_skipped"
	// AuditEventNodeRetried indicates a node was retried after a failure
	AuditEventNodeRetried AuditEventType = "node_retried"

	// AuditEventVariableSet indicates a variable was created or updated
	AuditEventVariableSet AuditEventType = "variable_set"

	// AuditEventError indicates an error occurred
	AuditEventError AuditEventType = "error"
)

// AuditEvent represents a single event in the execution audit trail.
// Each event captures what happened, when it happened, and relevant context.
type AuditEvent struct {
	// Timestamp records when this event occurred
	Timestamp time.Time `json:"timestamp"`

	// Type categorizes the event
	Type AuditEventType `json:"type"`

	// NodeID identifies the node this event relates to (empty if not node-specific)
	NodeID types.NodeID `json:"node_id,omitempty"`

	// NodeType identifies the type of node (e.g., "mcp_tool", "transform")
	NodeType string `json:"node_type,omitempty"`

	// ExecutionID identifies which node execution this relates to (for node events)
	NodeExecutionID types.NodeExecutionID `json:"node_execution_id,omitempty"`

	// Message provides a human-readable description of the event
	Message string `json:"message"`

	// Details contains structured data specific to this event type
	Details map[string]interface{} `json:"details,omitempty"`

	// Duration records how long an operation took (for completion events)
	Duration *time.Duration `json:"duration,omitempty"`
}

// AuditTrail represents the complete execution history reconstructed from execution data.
// It provides a chronological, human-readable view of everything that happened.
type AuditTrail struct {
	// ExecutionID identifies which execution this trail is for
	ExecutionID types.ExecutionID `json:"execution_id"`

	// WorkflowID identifies which workflow was executed
	WorkflowID types.WorkflowID `json:"workflow_id"`

	// WorkflowVersion captures the workflow version at execution time
	WorkflowVersion string `json:"workflow_version"`

	// Status is the final execution status
	Status execution.Status `json:"status"`

	// StartedAt is when the execution began
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the execution finished (nil if still running)
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// Duration is the total execution time
	Duration time.Duration `json:"duration"`

	// Events is the chronologically ordered list of all events
	Events []AuditEvent `json:"events"`

	// NodeCount is the total number of nodes executed
	NodeCount int `json:"node_count"`

	// ErrorCount is the total number of errors encountered
	ErrorCount int `json:"error_count"`

	// VariableChangeCount is the total number of variable changes
	VariableChangeCount int `json:"variable_change_count"`

	// RetryCount is the total number of retries across all nodes
	RetryCount int `json:"retry_count"`

	// ReturnValue is the final output (if execution completed successfully)
	ReturnValue interface{} `json:"return_value,omitempty"`
}

// AuditTrailFilter allows filtering audit events by type.
type AuditTrailFilter struct {
	// EventTypes filters to only include these event types (nil = no filter)
	EventTypes []AuditEventType

	// NodeID filters to only include events for this node (empty = no filter)
	NodeID types.NodeID

	// IncludeVariableChanges controls whether to include variable change events
	IncludeVariableChanges bool

	// StartTime filters events after this time (nil = no filter)
	StartTime *time.Time

	// EndTime filters events before this time (nil = no filter)
	EndTime *time.Time
}

// ReconstructAuditTrail rebuilds the complete audit trail from an execution.
// This creates a chronological event log from execution data stored in the database.
func ReconstructAuditTrail(exec *execution.Execution) (*AuditTrail, error) {
	if exec == nil {
		return nil, fmt.Errorf("execution cannot be nil")
	}

	trail := &AuditTrail{
		ExecutionID:     exec.ID,
		WorkflowID:      exec.WorkflowID,
		WorkflowVersion: exec.WorkflowVersion,
		Status:          exec.Status,
		StartedAt:       exec.StartedAt,
		CompletedAt:     exec.CompletedAt,
		Duration:        exec.Duration(),
		Events:          make([]AuditEvent, 0),
		ReturnValue:     exec.ReturnValue,
	}

	// Add execution started event
	trail.Events = append(trail.Events, AuditEvent{
		Timestamp: exec.StartedAt,
		Type:      AuditEventExecutionStarted,
		Message:   fmt.Sprintf("Execution started for workflow '%s' version %s", exec.WorkflowID, exec.WorkflowVersion),
		Details: map[string]interface{}{
			"execution_id":     exec.ID.String(),
			"workflow_id":      string(exec.WorkflowID),
			"workflow_version": exec.WorkflowVersion,
		},
	})

	// Add variable initialization events from context history
	if exec.Context != nil {
		variableHistory := exec.Context.GetVariableHistory()
		for _, snapshot := range variableHistory {
			trail.Events = append(trail.Events, createVariableChangeEvent(snapshot))
			trail.VariableChangeCount++
		}
	}

	// Add node execution events
	for _, nodeExec := range exec.NodeExecutions {
		nodeEvents := createNodeExecutionEvents(nodeExec)
		trail.Events = append(trail.Events, nodeEvents...)
		trail.NodeCount++

		if nodeExec.Error != nil {
			trail.ErrorCount++
		}
		if nodeExec.RetryCount > 0 {
			trail.RetryCount += nodeExec.RetryCount
		}
	}

	// Add execution completion event
	if exec.Status.IsTerminal() {
		trail.Events = append(trail.Events, createExecutionCompletionEvent(exec))
	}

	// Sort all events chronologically
	sort.Slice(trail.Events, func(i, j int) bool {
		return trail.Events[i].Timestamp.Before(trail.Events[j].Timestamp)
	})

	return trail, nil
}

// FilterEvents returns a new audit trail with only events matching the filter criteria.
func (at *AuditTrail) FilterEvents(filter AuditTrailFilter) *AuditTrail {
	filtered := &AuditTrail{
		ExecutionID:     at.ExecutionID,
		WorkflowID:      at.WorkflowID,
		WorkflowVersion: at.WorkflowVersion,
		Status:          at.Status,
		StartedAt:       at.StartedAt,
		CompletedAt:     at.CompletedAt,
		Duration:        at.Duration,
		Events:          make([]AuditEvent, 0),
		ReturnValue:     at.ReturnValue,
	}

	for _, event := range at.Events {
		// Filter by event type
		if len(filter.EventTypes) > 0 {
			found := false
			for _, t := range filter.EventTypes {
				if event.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by node ID
		if filter.NodeID != "" && event.NodeID != filter.NodeID {
			continue
		}

		// Filter variable changes
		if !filter.IncludeVariableChanges && event.Type == AuditEventVariableSet {
			continue
		}

		// Filter by time range
		if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
			continue
		}

		filtered.Events = append(filtered.Events, event)
	}

	// Recalculate counts based on filtered events
	filtered.recalculateCounts()

	return filtered
}

// recalculateCounts updates the summary counts based on current events.
func (at *AuditTrail) recalculateCounts() {
	at.NodeCount = 0
	at.ErrorCount = 0
	at.VariableChangeCount = 0
	at.RetryCount = 0

	nodesSeen := make(map[types.NodeExecutionID]bool)

	for _, event := range at.Events {
		switch event.Type {
		case AuditEventNodeStarted:
			if event.NodeExecutionID != "" && !nodesSeen[event.NodeExecutionID] {
				at.NodeCount++
				nodesSeen[event.NodeExecutionID] = true
			}
		case AuditEventNodeFailed, AuditEventError:
			at.ErrorCount++
		case AuditEventVariableSet:
			at.VariableChangeCount++
		case AuditEventNodeRetried:
			at.RetryCount++
		}
	}
}

// FormatHumanReadable returns a human-readable text representation of the audit trail.
func (at *AuditTrail) FormatHumanReadable() string {
	var sb strings.Builder

	// Header
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Execution Audit Trail: %s\n", at.ExecutionID))
	sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	// Metadata
	sb.WriteString(fmt.Sprintf("Workflow:     %s (version %s)\n", at.WorkflowID, at.WorkflowVersion))
	sb.WriteString(fmt.Sprintf("Status:       %s\n", at.Status))
	sb.WriteString(fmt.Sprintf("Started:      %s\n", at.StartedAt.Format(time.RFC3339)))
	if !at.CompletedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("Completed:    %s\n", at.CompletedAt.Format(time.RFC3339)))
	}
	sb.WriteString(fmt.Sprintf("Duration:     %s\n", at.Duration))
	sb.WriteString("\n")

	// Summary
	sb.WriteString(fmt.Sprintf("Nodes Executed:    %d\n", at.NodeCount))
	sb.WriteString(fmt.Sprintf("Errors:            %d\n", at.ErrorCount))
	sb.WriteString(fmt.Sprintf("Variable Changes:  %d\n", at.VariableChangeCount))
	sb.WriteString(fmt.Sprintf("Retries:           %d\n", at.RetryCount))
	sb.WriteString("\n")

	// Events
	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("Event Timeline\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n\n")

	for i, event := range at.Events {
		// Calculate time offset from execution start
		offset := event.Timestamp.Sub(at.StartedAt)

		// Format timestamp and offset
		timestamp := event.Timestamp.Format("15:04:05.000")
		offsetStr := fmt.Sprintf("+%s", offset.Round(time.Millisecond))

		// Event type indicator
		icon := getEventIcon(event.Type)

		// Format based on event type
		sb.WriteString(fmt.Sprintf("[%s] %s %s %s\n", timestamp, offsetStr, icon, event.Message))

		// Add node context if present
		if event.NodeID != "" {
			sb.WriteString(fmt.Sprintf("        Node: %s (%s)\n", event.NodeID, event.NodeType))
		}

		// Add duration if present
		if event.Duration != nil {
			sb.WriteString(fmt.Sprintf("        Duration: %s\n", event.Duration.Round(time.Millisecond)))
		}

		// Add important details
		if len(event.Details) > 0 {
			// Filter out verbose details for human-readable format
			importantDetails := filterImportantDetails(event.Details)
			if len(importantDetails) > 0 {
				sb.WriteString("        Details:\n")
				for key, value := range importantDetails {
					sb.WriteString(fmt.Sprintf("          - %s: %v\n", key, formatAuditValue(value)))
				}
			}
		}

		// Add blank line between events for readability
		if i < len(at.Events)-1 {
			sb.WriteString("\n")
		}
	}

	// Footer
	sb.WriteString("\n═══════════════════════════════════════════════════════════════\n")

	return sb.String()
}

// ExportJSON returns the audit trail as JSON.
func (at *AuditTrail) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(at, "", "  ")
}

// ExportCompactJSON returns the audit trail as compact JSON (no indentation).
func (at *AuditTrail) ExportCompactJSON() ([]byte, error) {
	return json.Marshal(at)
}

// GetEventsByType returns all events of a specific type.
func (at *AuditTrail) GetEventsByType(eventType AuditEventType) []AuditEvent {
	var events []AuditEvent
	for _, event := range at.Events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}
	return events
}

// GetEventsForNode returns all events related to a specific node.
func (at *AuditTrail) GetEventsForNode(nodeID types.NodeID) []AuditEvent {
	var events []AuditEvent
	for _, event := range at.Events {
		if event.NodeID == nodeID {
			events = append(events, event)
		}
	}
	return events
}

// GetErrorEvents returns all error events.
func (at *AuditTrail) GetErrorEvents() []AuditEvent {
	var events []AuditEvent
	for _, event := range at.Events {
		if event.Type == AuditEventError || event.Type == AuditEventNodeFailed || event.Type == AuditEventExecutionFailed {
			events = append(events, event)
		}
	}
	return events
}

// GetVariableChanges returns all variable change events.
func (at *AuditTrail) GetVariableChanges() []AuditEvent {
	return at.GetEventsByType(AuditEventVariableSet)
}

// Helper functions

func createVariableChangeEvent(snapshot execution.VariableSnapshot) AuditEvent {
	event := AuditEvent{
		Timestamp:       snapshot.Timestamp,
		Type:            AuditEventVariableSet,
		NodeExecutionID: snapshot.NodeExecutionID,
		Details: map[string]interface{}{
			"variable_name": snapshot.VariableName,
			"old_value":     snapshot.OldValue,
			"new_value":     snapshot.NewValue,
		},
	}

	// Create message
	if snapshot.OldValue == nil {
		event.Message = fmt.Sprintf("Variable '%s' initialized", snapshot.VariableName)
	} else {
		event.Message = fmt.Sprintf("Variable '%s' updated", snapshot.VariableName)
	}

	return event
}

func createNodeExecutionEvents(nodeExec *execution.NodeExecution) []AuditEvent {
	events := make([]AuditEvent, 0, 4) // Start, possible retries, completion

	// Add retry events if any
	for i := 0; i < nodeExec.RetryCount; i++ {
		events = append(events, AuditEvent{
			Timestamp:       nodeExec.StartedAt.Add(-time.Duration(nodeExec.RetryCount-i) * time.Second), // Estimate retry times
			Type:            AuditEventNodeRetried,
			NodeID:          nodeExec.NodeID,
			NodeType:        nodeExec.NodeType,
			NodeExecutionID: nodeExec.ID,
			Message:         fmt.Sprintf("Node '%s' retry attempt %d", nodeExec.NodeID, i+1),
			Details: map[string]interface{}{
				"retry_count": i + 1,
			},
		})
	}

	// Add node started event
	events = append(events, AuditEvent{
		Timestamp:       nodeExec.StartedAt,
		Type:            AuditEventNodeStarted,
		NodeID:          nodeExec.NodeID,
		NodeType:        nodeExec.NodeType,
		NodeExecutionID: nodeExec.ID,
		Message:         fmt.Sprintf("Node '%s' started execution", nodeExec.NodeID),
		Details: map[string]interface{}{
			"inputs": nodeExec.Inputs,
		},
	})

	// Add completion event based on status
	if !nodeExec.CompletedAt.IsZero() {
		duration := nodeExec.Duration()
		completionEvent := AuditEvent{
			Timestamp:       nodeExec.CompletedAt,
			NodeID:          nodeExec.NodeID,
			NodeType:        nodeExec.NodeType,
			NodeExecutionID: nodeExec.ID,
			Duration:        &duration,
		}

		switch nodeExec.Status {
		case execution.NodeStatusCompleted:
			completionEvent.Type = AuditEventNodeCompleted
			completionEvent.Message = fmt.Sprintf("Node '%s' completed successfully", nodeExec.NodeID)
			completionEvent.Details = map[string]interface{}{
				"outputs": nodeExec.Outputs,
			}

		case execution.NodeStatusFailed:
			completionEvent.Type = AuditEventNodeFailed
			completionEvent.Message = fmt.Sprintf("Node '%s' failed", nodeExec.NodeID)
			if nodeExec.Error != nil {
				completionEvent.Details = map[string]interface{}{
					"error_type":    string(nodeExec.Error.Type),
					"error_message": nodeExec.Error.Message,
					"error_context": nodeExec.Error.Context,
				}
			}

		case execution.NodeStatusSkipped:
			completionEvent.Type = AuditEventNodeSkipped
			completionEvent.Message = fmt.Sprintf("Node '%s' was skipped", nodeExec.NodeID)
		}

		events = append(events, completionEvent)
	}

	return events
}

func createExecutionCompletionEvent(exec *execution.Execution) AuditEvent {
	event := AuditEvent{
		Timestamp: exec.CompletedAt,
	}

	duration := exec.Duration()
	event.Duration = &duration

	switch exec.Status {
	case execution.StatusCompleted:
		event.Type = AuditEventExecutionCompleted
		event.Message = "Execution completed successfully"
		if exec.ReturnValue != nil {
			event.Details = map[string]interface{}{
				"return_value": exec.ReturnValue,
			}
		}

	case execution.StatusFailed:
		event.Type = AuditEventExecutionFailed
		event.Message = "Execution failed"
		if exec.Error != nil {
			event.Details = map[string]interface{}{
				"error_type":    string(exec.Error.Type),
				"error_message": exec.Error.Message,
				"error_node_id": string(exec.Error.NodeID),
				"error_context": exec.Error.Context,
			}
		}

	case execution.StatusCancelled:
		event.Type = AuditEventExecutionCancelled
		event.Message = "Execution was cancelled"
	}

	return event
}

func getEventIcon(eventType AuditEventType) string {
	switch eventType {
	case AuditEventExecutionStarted:
		return "▶"
	case AuditEventExecutionCompleted:
		return "✓"
	case AuditEventExecutionFailed:
		return "✗"
	case AuditEventExecutionCancelled:
		return "⊗"
	case AuditEventNodeStarted:
		return "→"
	case AuditEventNodeCompleted:
		return "✓"
	case AuditEventNodeFailed:
		return "✗"
	case AuditEventNodeSkipped:
		return "⊘"
	case AuditEventNodeRetried:
		return "↻"
	case AuditEventVariableSet:
		return "≔"
	case AuditEventError:
		return "⚠"
	default:
		return "•"
	}
}

func filterImportantDetails(details map[string]interface{}) map[string]interface{} {
	important := make(map[string]interface{})

	// Only include high-value details in human-readable format
	importantKeys := []string{
		"error_type", "error_message",
		"retry_count",
		"variable_name",
		"workflow_id", "execution_id",
	}

	for _, key := range importantKeys {
		if value, exists := details[key]; exists {
			important[key] = value
		}
	}

	return important
}

func formatAuditValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		if len(v) > 100 {
			return v[:100] + "..."
		}
		return v
	case map[string]interface{}:
		return fmt.Sprintf("{%d fields}", len(v))
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	default:
		str := fmt.Sprintf("%v", v)
		if len(str) > 100 {
			return str[:100] + "..."
		}
		return str
	}
}
