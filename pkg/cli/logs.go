package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	pkgexec "github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/storage"
	"github.com/spf13/cobra"
)

// ANSI color codes are defined in colors.go

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	var (
		follow       bool
		eventType    string
		tailCount    int
		noColor      bool
		showVariable bool
	)

	cmd := &cobra.Command{
		Use:   "logs <execution-id>",
		Short: "Display execution logs",
		Long: `Display execution logs in chronological order.

View historical logs for completed executions or follow real-time logs for running executions.

The logs command reconstructs the audit trail from stored execution data and displays
events with timestamps, types, and contextual information.

Examples:
  # View all logs for execution
  goflow logs exec-12345

  # Follow logs for running execution (real-time)
  goflow logs exec-12345 --follow

  # Show only errors
  goflow logs exec-12345 --type error

  # Show last 20 log entries
  goflow logs exec-12345 --tail 20

  # Combine filters
  goflow logs exec-12345 --type info --tail 50

Event Types:
  execution_started    - Execution began
  execution_completed  - Execution finished successfully
  execution_failed     - Execution failed with error
  execution_cancelled  - Execution was cancelled
  node_started         - Node began execution
  node_completed       - Node finished successfully
  node_failed          - Node failed with error
  node_skipped         - Node was skipped
  node_retried         - Node was retried after failure
  variable_set         - Variable was created or updated
  error                - Error occurred`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			executionID := types.ExecutionID(args[0])

			// Initialize storage
			repo, err := storage.NewSQLiteExecutionRepository()
			if err != nil {
				return fmt.Errorf("failed to initialize storage: %w", err)
			}
			defer func() { _ = repo.Close() }()

			// Load execution
			exec, err := repo.Load(executionID)
			if err != nil {
				return fmt.Errorf("failed to load execution: %w", err)
			}

			// Check if following is possible
			if follow && exec.Status.IsTerminal() {
				return fmt.Errorf("cannot follow completed execution (status: %s)\nUse without --follow to view historical logs", exec.Status)
			}

			// Reconstruct audit trail
			trail, err := pkgexec.ReconstructAuditTrail(exec)
			if err != nil {
				return fmt.Errorf("failed to reconstruct audit trail: %w", err)
			}

			// Apply filters
			filter := pkgexec.AuditTrailFilter{
				IncludeVariableChanges: showVariable,
			}

			// Parse event type filter
			if eventType != "" {
				filter.EventTypes = parseEventTypeFilter(eventType)
			}

			filteredTrail := trail.FilterEvents(filter)

			// Apply tail limit
			if tailCount > 0 && tailCount < len(filteredTrail.Events) {
				start := len(filteredTrail.Events) - tailCount
				filteredTrail.Events = filteredTrail.Events[start:]
			}

			// Display logs
			if follow {
				return displayFollowLogs(cmd, exec, filteredTrail, filter, noColor)
			}

			return displayHistoricalLogs(cmd, filteredTrail, noColor)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow logs for running execution (real-time)")
	cmd.Flags().StringVar(&eventType, "type", "", "Filter by event type (error, info, node_started, etc.)")
	cmd.Flags().IntVar(&tailCount, "tail", 0, "Show last N log entries (0 = show all)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	cmd.Flags().BoolVar(&showVariable, "show-variables", false, "Include variable change events")

	return cmd
}

// displayHistoricalLogs displays logs for completed executions
func displayHistoricalLogs(cmd *cobra.Command, trail *pkgexec.AuditTrail, noColor bool) error {
	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "Execution Logs: %s\n", trail.ExecutionID)
	fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s (version %s)\n", trail.WorkflowID, trail.WorkflowVersion)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", formatStatus(trail.Status, noColor))
	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Summary
	if trail.ErrorCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d events, %d nodes, %d errors\n",
			len(trail.Events), trail.NodeCount, trail.ErrorCount)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d events, %d nodes\n",
			len(trail.Events), trail.NodeCount)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Events
	for _, event := range trail.Events {
		displayEvent(cmd.OutOrStdout(), event, trail.StartedAt, noColor)
	}

	// Footer
	if !trail.CompletedAt.IsZero() {
		fmt.Fprintf(cmd.OutOrStdout(), "\nCompleted in %s\n", trail.Duration.Round(time.Millisecond))
	}

	return nil
}

// displayFollowLogs displays real-time logs for running executions
func displayFollowLogs(cmd *cobra.Command, exec *execution.Execution, trail *pkgexec.AuditTrail, filter pkgexec.AuditTrailFilter, noColor bool) error {
	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "Execution Logs: %s (following...)\n", trail.ExecutionID)
	fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s\n", trail.WorkflowID)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", formatStatus(trail.Status, noColor))
	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Display historical events first
	for _, event := range trail.Events {
		displayEvent(cmd.OutOrStdout(), event, trail.StartedAt, noColor)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Conditionally use color codes
	yellow := ""
	cyan := ""
	gray := ""
	reset := ""
	if !noColor {
		yellow = colorYellow
		cyan = colorCyan
		gray = colorGray
		reset = colorReset
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(cmd.OutOrStderr(), "\n%sReceived interrupt signal, stopping...%s\n",
			yellow, reset)
		cancel()
	}()

	// Create monitor for real-time events
	// Note: In real implementation, this would connect to a running execution
	// For now, we'll poll the storage for updates
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastEventCount := len(trail.Events)
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s[waiting for more events...]%s\n", gray, reset)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Reload execution to check for updates
			repo, err := storage.NewSQLiteExecutionRepository()
			if err != nil {
				return fmt.Errorf("failed to reconnect to storage: %w", err)
			}

			updatedExec, err := repo.Load(exec.ID)
			_ = repo.Close()
			if err != nil {
				return fmt.Errorf("failed to reload execution: %w", err)
			}

			// Reconstruct updated trail
			updatedTrail, err := pkgexec.ReconstructAuditTrail(updatedExec)
			if err != nil {
				return fmt.Errorf("failed to reconstruct trail: %w", err)
			}

			// Apply filters
			filteredUpdated := updatedTrail.FilterEvents(filter)

			// Display new events
			if len(filteredUpdated.Events) > lastEventCount {
				for i := lastEventCount; i < len(filteredUpdated.Events); i++ {
					displayEvent(cmd.OutOrStdout(), filteredUpdated.Events[i], updatedTrail.StartedAt, noColor)
				}
				lastEventCount = len(filteredUpdated.Events)
			}

			// Check if execution completed
			if updatedExec.Status.IsTerminal() {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sExecution completed with status: %s%s\n",
					cyan, formatStatus(updatedExec.Status, noColor), reset)
				if !updatedExec.CompletedAt.IsZero() {
					duration := updatedExec.CompletedAt.Sub(updatedExec.StartedAt)
					fmt.Fprintf(cmd.OutOrStdout(), "Total duration: %s\n", duration.Round(time.Millisecond))
				}
				return nil
			}
		}
	}
}

// displayEvent formats and displays a single audit event
func displayEvent(w io.Writer, event pkgexec.AuditEvent, startTime time.Time, noColor bool) {
	// Format timestamp with millisecond precision
	timestamp := event.Timestamp.Format("15:04:05.000")

	// Calculate offset from start
	offset := event.Timestamp.Sub(startTime)
	var offsetStr string
	if offset >= 0 {
		offsetStr = fmt.Sprintf("+%.3fs", offset.Seconds())
	} else {
		offsetStr = fmt.Sprintf("%.3fs", offset.Seconds())
	}

	// Get icon and color for event type
	icon := getEventIcon(event.Type)
	color := getEventColor(event.Type, noColor)

	// Conditionally use color codes
	reset := ""
	gray := ""
	red := ""
	if !noColor {
		reset = colorReset
		gray = colorGray
		red = colorRed
	}

	// Format main message
	fmt.Fprintf(w, "%s  %8s  %s%s %s%s",
		timestamp, offsetStr, color, icon, event.Message, reset)

	// Add duration if present
	if event.Duration != nil {
		fmt.Fprintf(w, " %s(%.3fs)%s",
			gray, event.Duration.Seconds(), reset)
	}

	fmt.Fprintf(w, "\n")

	// Add node context if present (indented)
	if event.NodeID != "" && event.NodeType != "" {
		fmt.Fprintf(w, "           %sNode: %s (%s)%s\n",
			gray, event.NodeID, event.NodeType, reset)
	}

	// Display important error details
	if event.Type == pkgexec.AuditEventNodeFailed || event.Type == pkgexec.AuditEventExecutionFailed || event.Type == pkgexec.AuditEventError {
		if errorMsg, ok := event.Details["error_message"].(string); ok {
			fmt.Fprintf(w, "           %sError: %s%s\n",
				red, errorMsg, reset)
		}
	}
}

// getEventIcon returns an icon for the event type
func getEventIcon(eventType pkgexec.AuditEventType) string {
	switch eventType {
	case pkgexec.AuditEventExecutionStarted:
		return "▶"
	case pkgexec.AuditEventExecutionCompleted:
		return "✓"
	case pkgexec.AuditEventExecutionFailed:
		return "✗"
	case pkgexec.AuditEventExecutionCancelled:
		return "⊗"
	case pkgexec.AuditEventNodeStarted:
		return "▶"
	case pkgexec.AuditEventNodeCompleted:
		return "✓"
	case pkgexec.AuditEventNodeFailed:
		return "✗"
	case pkgexec.AuditEventNodeSkipped:
		return "⊘"
	case pkgexec.AuditEventNodeRetried:
		return "↻"
	case pkgexec.AuditEventVariableSet:
		return "≔"
	case pkgexec.AuditEventError:
		return "⚠"
	default:
		return "◆"
	}
}

// getEventColor returns the color code for an event type
func getEventColor(eventType pkgexec.AuditEventType, noColor bool) string {
	if noColor {
		return ""
	}

	switch eventType {
	case pkgexec.AuditEventExecutionStarted, pkgexec.AuditEventNodeStarted:
		return colorBlue
	case pkgexec.AuditEventExecutionCompleted, pkgexec.AuditEventNodeCompleted:
		return colorGreen
	case pkgexec.AuditEventExecutionFailed, pkgexec.AuditEventNodeFailed, pkgexec.AuditEventError:
		return colorRed
	case pkgexec.AuditEventExecutionCancelled, pkgexec.AuditEventNodeSkipped:
		return colorYellow
	case pkgexec.AuditEventNodeRetried:
		return colorYellow
	case pkgexec.AuditEventVariableSet:
		return colorCyan
	default:
		return colorGray
	}
}

// formatStatus formats execution status with color
func formatStatus(status execution.Status, noColor bool) string {
	statusStr := string(status)
	if noColor {
		return statusStr
	}

	switch status {
	case execution.StatusCompleted:
		return colorGreen + statusStr + colorReset
	case execution.StatusFailed:
		return colorRed + statusStr + colorReset
	case execution.StatusCancelled:
		return colorYellow + statusStr + colorReset
	case execution.StatusRunning:
		return colorBlue + statusStr + colorReset
	default:
		return statusStr
	}
}

// parseEventTypeFilter parses comma-separated event type filter string
func parseEventTypeFilter(filter string) []pkgexec.AuditEventType {
	if filter == "" {
		return nil
	}

	parts := strings.Split(filter, ",")
	eventTypes := make([]pkgexec.AuditEventType, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Support shorthand filters
		switch strings.ToLower(part) {
		case "error", "errors":
			eventTypes = append(eventTypes,
				pkgexec.AuditEventError,
				pkgexec.AuditEventNodeFailed,
				pkgexec.AuditEventExecutionFailed)
		case "info":
			eventTypes = append(eventTypes,
				pkgexec.AuditEventExecutionStarted,
				pkgexec.AuditEventExecutionCompleted,
				pkgexec.AuditEventNodeStarted,
				pkgexec.AuditEventNodeCompleted)
		case "warning", "warnings":
			eventTypes = append(eventTypes,
				pkgexec.AuditEventNodeRetried,
				pkgexec.AuditEventNodeSkipped)
		default:
			// Try exact event type match
			eventTypes = append(eventTypes, pkgexec.AuditEventType(part))
		}
	}

	return eventTypes
}
