package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/storage"
	"github.com/spf13/cobra"
)

// ExecutionsListFlags holds the flags for the executions list command
type ExecutionsListFlags struct {
	Limit    int
	Offset   int
	Workflow string
	Status   string
	Since    string
}

// ExecutionDetailFlags holds the flags for the execution detail command
type ExecutionDetailFlags struct {
	JSON bool
}

// NewExecutionsCommand creates the main executions command
func NewExecutionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "executions",
		Short: "List execution history",
		Long:  `List execution history with pagination and filtering options.`,
		RunE:  runExecutionsList,
	}

	flags := &ExecutionsListFlags{}
	cmd.Flags().IntVar(&flags.Limit, "limit", 20, "Maximum number of executions to display")
	cmd.Flags().IntVar(&flags.Offset, "offset", 0, "Number of executions to skip")
	cmd.Flags().StringVar(&flags.Workflow, "workflow", "", "Filter by workflow name")
	cmd.Flags().StringVar(&flags.Status, "status", "", "Filter by status (pending, running, completed, failed, cancelled)")
	cmd.Flags().StringVar(&flags.Since, "since", "", "Filter by date (e.g., 7d, 24h, 2025-01-05)")

	// Store flags in command context for access in RunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmd.Context())
		return nil
	}

	return cmd
}

// NewExecutionCommand creates the execution detail command
func NewExecutionCommand() *cobra.Command {
	flags := &ExecutionDetailFlags{}

	cmd := &cobra.Command{
		Use:   "execution <id>",
		Short: "Display detailed execution information",
		Long:  `Display detailed information about a specific execution including node executions and variables.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExecutionDetail(cmd, args[0], flags)
		},
	}

	cmd.Flags().BoolVar(&flags.JSON, "json", false, "Output execution details as JSON")

	return cmd
}

// runExecutionsList handles the executions list command
func runExecutionsList(cmd *cobra.Command, args []string) error {
	// Parse flags
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	workflowName, _ := cmd.Flags().GetString("workflow")
	statusStr, _ := cmd.Flags().GetString("status")
	sinceStr, _ := cmd.Flags().GetString("since")

	// Create repository
	repo, err := storage.NewSQLiteExecutionRepository()
	if err != nil {
		return fmt.Errorf("failed to create execution repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	// Build list options
	options := execution.ListOptions{
		Limit:  limit,
		Offset: offset,
	}

	// Apply workflow filter
	if workflowName != "" {
		workflowID := types.WorkflowID(workflowName)
		options.WorkflowID = &workflowID
	}

	// Apply status filter
	if statusStr != "" {
		status := execution.Status(statusStr)
		// Validate status
		validStatuses := []execution.Status{
			execution.StatusPending,
			execution.StatusRunning,
			execution.StatusCompleted,
			execution.StatusFailed,
			execution.StatusCancelled,
		}
		valid := false
		for _, vs := range validStatuses {
			if status == vs {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status: %s (valid: pending, running, completed, failed, cancelled)", statusStr)
		}
		options.Status = &status
	}

	// Apply date filter
	if sinceStr != "" {
		startedAfter, err := parseSinceFlag(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		options.StartedAfter = &startedAfter
	}

	// Query executions
	result, err := repo.List(options)
	if err != nil {
		return fmt.Errorf("failed to list executions: %w", err)
	}

	// Display results
	if len(result.Executions) == 0 {
		fmt.Println("No executions found.")
		return nil
	}

	printExecutionsTable(result)

	// Show pagination info
	if result.TotalCount > len(result.Executions) {
		showing := offset + len(result.Executions)
		fmt.Printf("\nShowing %d-%d of %d total executions\n", offset+1, showing, result.TotalCount)
	}

	return nil
}

// runExecutionDetail handles the execution detail command
func runExecutionDetail(cmd *cobra.Command, executionID string, flags *ExecutionDetailFlags) error {
	// Create repository
	repo, err := storage.NewSQLiteExecutionRepository()
	if err != nil {
		return fmt.Errorf("failed to create execution repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	// Load execution
	exec, err := repo.Load(types.ExecutionID(executionID))
	if err != nil {
		return fmt.Errorf("failed to load execution: %w", err)
	}

	// Output as JSON if requested
	if flags.JSON {
		return printExecutionJSON(exec)
	}

	// Otherwise print formatted output
	printExecutionDetail(exec)
	return nil
}

// printExecutionsTable displays executions in a formatted table
func printExecutionsTable(result *execution.ListResult) {
	// Print header
	fmt.Printf("%-20s %-25s %-12s %-10s %s\n",
		"ID", "Workflow", "Status", "Duration", "Started")
	fmt.Println(strings.Repeat("-", 90))

	// Print each execution
	for _, exec := range result.Executions {
		id := truncateString(string(exec.ID), 18)
		workflow := truncateString(string(exec.WorkflowID), 23)
		status := colorizeStatus(string(exec.Status))
		duration := formatDuration(exec)
		started := exec.StartedAt.Format("2006-01-02 15:04")

		fmt.Printf("%-20s %-25s %-22s %-10s %s\n",
			id, workflow, status, duration, started)
	}
}

// printExecutionDetail displays detailed execution information
func printExecutionDetail(exec *execution.Execution) {
	// Header section
	fmt.Printf("Execution: %s\n", colorCyan+string(exec.ID)+colorReset)
	fmt.Printf("Workflow: %s (v%s)\n", exec.WorkflowID, exec.WorkflowVersion)
	fmt.Printf("Status: %s\n", colorizeStatus(string(exec.Status)))
	fmt.Printf("Started: %s\n", exec.StartedAt.Format("2006-01-02 15:04:05"))
	if !exec.CompletedAt.IsZero() {
		fmt.Printf("Completed: %s\n", exec.CompletedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Duration: %s\n", formatDurationValue(exec.Duration()))
	}
	fmt.Println()

	// Error section (if failed)
	if exec.Error != nil {
		fmt.Printf("%sError:%s\n", colorRed, colorReset)
		fmt.Printf("  Type: %s\n", exec.Error.Type)
		fmt.Printf("  Message: %s\n", exec.Error.Message)
		if exec.Error.NodeID != "" {
			fmt.Printf("  Node: %s\n", exec.Error.NodeID)
		}
		if len(exec.Error.Context) > 0 {
			fmt.Printf("  Context:\n")
			for k, v := range exec.Error.Context {
				fmt.Printf("    %s: %v\n", k, v)
			}
		}
		fmt.Println()
	}

	// Node executions section
	if len(exec.NodeExecutions) > 0 {
		fmt.Println("Node Executions:")
		for _, ne := range exec.NodeExecutions {
			symbol := getNodeSymbol(ne.Status)
			nodeType := truncateString(ne.NodeType, 12)
			duration := formatDurationValue(ne.Duration())

			fmt.Printf("  %s %-15s (%-12s) %-6s\n",
				symbol,
				truncateString(string(ne.NodeID), 15),
				nodeType,
				duration)

			// Show node error if present
			if ne.Error != nil {
				fmt.Printf("      %sError: %s%s\n", colorRed, ne.Error.Message, colorReset)
			}
		}
		fmt.Println()
	}

	// Variables section
	if exec.Context != nil && len(exec.Context.Variables) > 0 {
		fmt.Println("Variables:")
		for name, value := range exec.Context.Variables {
			valueStr := formatValue(value)
			fmt.Printf("  %s: %s\n", name, valueStr)
		}
		fmt.Println()
	}

	// Return value section
	if exec.ReturnValue != nil {
		fmt.Println("Return Value:")
		valueStr := formatValue(exec.ReturnValue)
		fmt.Printf("  %s\n", valueStr)
	}
}

// printExecutionJSON outputs execution as JSON
func printExecutionJSON(exec *execution.Execution) error {
	// Create a simplified structure for JSON output
	output := map[string]interface{}{
		"id":               exec.ID,
		"workflow_id":      exec.WorkflowID,
		"workflow_version": exec.WorkflowVersion,
		"status":           exec.Status,
		"started_at":       exec.StartedAt,
		"completed_at":     exec.CompletedAt,
		"duration_ms":      exec.Duration().Milliseconds(),
	}

	if exec.Error != nil {
		output["error"] = map[string]interface{}{
			"type":    exec.Error.Type,
			"message": exec.Error.Message,
			"node_id": exec.Error.NodeID,
			"context": exec.Error.Context,
		}
	}

	if len(exec.NodeExecutions) > 0 {
		nodeExecs := make([]map[string]interface{}, len(exec.NodeExecutions))
		for i, ne := range exec.NodeExecutions {
			nodeExecs[i] = map[string]interface{}{
				"id":           ne.ID,
				"node_id":      ne.NodeID,
				"node_type":    ne.NodeType,
				"status":       ne.Status,
				"started_at":   ne.StartedAt,
				"completed_at": ne.CompletedAt,
				"duration_ms":  ne.Duration().Milliseconds(),
				"inputs":       ne.Inputs,
				"outputs":      ne.Outputs,
				"retry_count":  ne.RetryCount,
			}
			if ne.Error != nil {
				nodeExecs[i]["error"] = map[string]interface{}{
					"type":    ne.Error.Type,
					"message": ne.Error.Message,
					"context": ne.Error.Context,
				}
			}
		}
		output["node_executions"] = nodeExecs
	}

	if exec.Context != nil && len(exec.Context.Variables) > 0 {
		output["variables"] = exec.Context.Variables
	}

	if exec.ReturnValue != nil {
		output["return_value"] = exec.ReturnValue
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// Helper functions

// parseSinceFlag parses the --since flag into a time.Time
// Supports formats: "7d" (7 days), "24h" (24 hours), "2025-01-05" (date)
func parseSinceFlag(since string) (time.Time, error) {
	now := time.Now()

	// Try parsing as duration (e.g., "7d", "24h")
	if strings.HasSuffix(since, "d") {
		days := since[:len(since)-1]
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err == nil {
			return now.AddDate(0, 0, -d), nil
		}
	}
	if strings.HasSuffix(since, "h") {
		hours := since[:len(since)-1]
		var h int
		if _, err := fmt.Sscanf(hours, "%d", &h); err == nil {
			return now.Add(-time.Duration(h) * time.Hour), nil
		}
	}

	// Try parsing as date (e.g., "2025-01-05")
	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, since); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format (use: 7d, 24h, or 2025-01-05)")
}

// colorizeStatus returns a colored status string
func colorizeStatus(status string) string {
	switch execution.Status(status) {
	case execution.StatusCompleted:
		return colorGreen + status + colorReset
	case execution.StatusFailed:
		return colorRed + status + colorReset
	case execution.StatusRunning:
		return colorYellow + status + colorReset
	case execution.StatusPending:
		return colorGray + status + colorReset
	case execution.StatusCancelled:
		return colorGray + status + colorReset
	default:
		return status
	}
}

// getNodeSymbol returns a status symbol for a node execution
func getNodeSymbol(status execution.NodeStatus) string {
	switch status {
	case execution.NodeStatusCompleted:
		return colorGreen + "✓" + colorReset
	case execution.NodeStatusFailed:
		return colorRed + "✗" + colorReset
	case execution.NodeStatusRunning:
		return colorYellow + "●" + colorReset
	case execution.NodeStatusSkipped:
		return colorGray + "○" + colorReset
	case execution.NodeStatusPending:
		return colorGray + "○" + colorReset
	default:
		return " "
	}
}

// formatDuration returns formatted duration for an execution
func formatDuration(exec *execution.Execution) string {
	if exec.CompletedAt.IsZero() {
		return "-"
	}
	return formatDurationValue(exec.Duration())
}

// formatDurationValue formats a duration value
func formatDurationValue(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}

	// Handle basic types
	switch val := v.(type) {
	case string:
		if len(val) > 100 {
			return fmt.Sprintf("%q...", val[:97])
		}
		return fmt.Sprintf("%q", val)
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", val)
	}

	// Handle complex types - marshal to JSON
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	str := string(data)
	if len(str) > 100 {
		return str[:97] + "..."
	}
	return str
}
