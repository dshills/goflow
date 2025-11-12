package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	domainexec "github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
	"github.com/dshills/goflow/pkg/execution"
	"github.com/dshills/goflow/pkg/tui"
	"github.com/dshills/goflow/pkg/workflow"
	"github.com/dshills/goterm"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	var (
		inputFile    string
		watch        bool
		tuiMode      bool
		outputJSON   bool
		varFlags     []string // Inline variables (--var key=value)
		debugMode    bool
		outputFormat string
		timeout      int // Timeout in seconds
		fromStdin    bool
	)

	cmd := &cobra.Command{
		Use:   "run <workflow-name>",
		Short: "Execute a workflow",
		Long: `Execute a workflow with optional input variables.

The workflow is loaded from ~/.goflow/workflows/<workflow-name>.yaml

Examples:
  # Run workflow with default variables
  goflow run my-workflow

  # Run with input variables from JSON file
  goflow run my-workflow --input input.json

  # Run with inline progress monitoring
  goflow run my-workflow --watch

  # Run with full TUI monitoring
  goflow run my-workflow --tui

  # Run with debug output
  goflow run my-workflow --debug`,
		Args: func(cmd *cobra.Command, args []string) error {
			if fromStdin {
				return nil // No args required when reading from stdin
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var workflowName string
			var workflowPath string

			if fromStdin {
				workflowName = "stdin"
			} else {
				workflowName = args[0]
				// Construct workflow path
				workflowPath = filepath.Join(GetWorkflowsDir(), workflowName+".yaml")
			}

			var wf *workflow.Workflow
			var err error

			if fromStdin {
				// Read workflow from stdin (use cmd.InOrStdin for testability)
				wf, err = LoadWorkflowFromReader(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("failed to parse workflow from stdin: %w", err)
				}
			} else {
				// Check if workflow file exists
				if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
					return fmt.Errorf("workflow not found: %s\n\nLooked in: %s", workflowName, workflowPath)
				}

				// Load workflow file
				wf, err = LoadWorkflowFromFile(workflowPath)
				if err != nil {
					return fmt.Errorf("failed to parse workflow YAML: %w", err)
				}
			}

			// Validate workflow
			if err := wf.Validate(); err != nil {
				return fmt.Errorf("workflow validation failed: %w", err)
			}

			// Load input variables if provided
			inputVars := make(map[string]interface{})

			// Load from file if specified
			if inputFile != "" {
				inputData, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("failed to read input file: %w", err)
				}

				if err := json.Unmarshal(inputData, &inputVars); err != nil {
					return fmt.Errorf("failed to parse input JSON: %w", err)
				}
			}

			// Parse inline variables (--var key=value)
			for _, varFlag := range varFlags {
				parts := splitKeyValue(varFlag)
				if len(parts) != 2 {
					return fmt.Errorf("invalid variable format: %s (expected key=value)", varFlag)
				}
				inputVars[parts[0]] = parts[1]
			}

			// Validate required variables are provided
			for _, variable := range wf.Variables {
				if variable == nil {
					continue
				}
				if variable.Required {
					if _, exists := inputVars[variable.Name]; !exists {
						return fmt.Errorf("required variable missing: %s", variable.Name)
					}
				}
			}

			// Apply timeout if specified
			if timeout > 0 {
				// TODO: Implement timeout context when runtime is integrated
				_ = timeout
			}

			// Set debug mode from flag
			if debugMode {
				GlobalConfig.Debug = true
			}

			// Determine output format
			if outputFormat == "json" {
				outputJSON = true
			}

			// Create execution engine
			engine := execution.NewEngine()
			defer engine.Close()

			// Create context with cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Apply timeout if specified
			if timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
				defer cancel()
			}

			// Handle Ctrl+C for graceful cancellation
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				cancel()
			}()

			// Decide execution mode: TUI, watch (inline), or silent
			if tuiMode {
				// Launch TUI monitoring mode
				return runWithTUI(ctx, engine, wf, workflowName, inputVars)
			} else if watch {
				// Run with inline watch mode
				return runWithInlineWatch(ctx, cmd, engine, wf, workflowName, inputVars, outputJSON, debugMode)
			} else {
				// Run silently, only show result at end
				return runSilent(ctx, cmd, engine, wf, workflowName, inputVars, outputJSON, debugMode)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input variables JSON file")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Monitor execution progress with inline updates")
	cmd.Flags().BoolVar(&tuiMode, "tui", false, "Launch full TUI execution monitor")
	cmd.Flags().BoolVar(&outputJSON, "output-json", false, "Output result as JSON")
	cmd.Flags().StringArrayVar(&varFlags, "var", []string{}, "Set input variable (key=value), can be used multiple times")
	cmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format (json or text)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Execution timeout in seconds (0 = no timeout)")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read workflow definition from stdin")

	return cmd
}

// splitKeyValue splits a string like "key=value" into ["key", "value"]
func splitKeyValue(s string) []string {
	idx := -1
	for i, ch := range s {
		if ch == '=' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// runWithTUI launches the full TUI execution monitor.
func runWithTUI(ctx context.Context, engine *execution.Engine, wf *workflow.Workflow, workflowName string, inputs map[string]interface{}) error {
	// Create a goroutine to run the execution
	var exec *domainexec.Execution
	var execErr error
	execDone := make(chan struct{})

	go func() {
		exec, execErr = engine.Execute(ctx, wf, inputs)
		close(execDone)
	}()

	// Initialize TUI screen
	screen, err := goterm.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}
	defer screen.Close()

	// Wait for execution to start and get monitor
	time.Sleep(100 * time.Millisecond)
	monitor := engine.GetMonitor()
	if monitor == nil {
		return fmt.Errorf("failed to get execution monitor")
	}

	// Create a temporary execution for initial display
	if exec == nil {
		exec, _ = domainexec.NewExecution(types.WorkflowID(wf.ID), wf.Version, inputs)
	}

	// Create execution monitor view
	monitorView := tui.NewExecutionMonitor(exec, wf, screen)
	monitorView.SetEventMonitor(monitor)
	defer monitorView.Close()

	// TUI event loop with periodic refresh
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Initial render
	monitorView.Render()

	for {
		select {
		case <-execDone:
			// Execution finished, show final state for a moment then exit
			monitorView.Render()
			time.Sleep(2 * time.Second)

			if execErr != nil {
				return execErr
			}
			return nil

		case <-ctx.Done():
			return fmt.Errorf("execution cancelled")

		case <-ticker.C:
			// Periodic refresh
			monitorView.Render()
		}
	}
}

// runWithInlineWatch runs execution with inline progress updates.
func runWithInlineWatch(ctx context.Context, cmd *cobra.Command, engine *execution.Engine, wf *workflow.Workflow, workflowName string, inputs map[string]interface{}, outputJSON, debugMode bool) error {
	// Start execution in background
	var exec *domainexec.Execution
	var execErr error
	execDone := make(chan struct{})

	go func() {
		exec, execErr = engine.Execute(ctx, wf, inputs)
		close(execDone)
	}()

	// Wait for execution to start and get monitor
	time.Sleep(100 * time.Millisecond)
	monitor := engine.GetMonitor()
	if monitor == nil {
		return fmt.Errorf("failed to get execution monitor")
	}

	// Subscribe to execution events
	eventChan := monitor.Subscribe()
	defer monitor.Unsubscribe(eventChan)

	// Check if stdout is a terminal for ANSI codes
	isTerm := term.IsTerminal(int(os.Stdout.Fd()))

	if !outputJSON {
		fmt.Fprintf(cmd.OutOrStdout(), "Executing: %s\n", workflowName)
		fmt.Fprintf(cmd.OutOrStdout(), "[Press Ctrl+C to cancel]\n\n")
	}

	// Track state for display
	state := &watchState{
		startTime: time.Now(),
		nodeCount: len(wf.Nodes),
	}

	// Event processing loop
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-execDone:
			// Show final status
			if !outputJSON {
				displayFinalResult(cmd, exec, execErr, state, debugMode)
			} else {
				displayJSONResult(cmd, exec, execErr)
			}
			return execErr

		case event := <-eventChan:
			if !outputJSON {
				handleInlineEvent(cmd, event, state, isTerm)
			}

		case <-ticker.C:
			// Periodic progress update
			if !outputJSON && isTerm {
				progress := monitor.GetProgress()
				displayInlineProgress(cmd, progress, state)
			}

		case <-ctx.Done():
			return fmt.Errorf("execution cancelled")
		}
	}
}

// runSilent runs execution without progress updates, only showing final result.
func runSilent(ctx context.Context, cmd *cobra.Command, engine *execution.Engine, wf *workflow.Workflow, workflowName string, inputs map[string]interface{}, outputJSON, debugMode bool) error {
	if !outputJSON {
		fmt.Fprintf(cmd.OutOrStdout(), "Executing workflow: %s\n", workflowName)
	}

	// Execute workflow
	exec, err := engine.Execute(ctx, wf, inputs)

	// Display result
	if !outputJSON {
		displayFinalResult(cmd, exec, err, &watchState{startTime: time.Now()}, debugMode)
	} else {
		displayJSONResult(cmd, exec, err)
	}

	return err
}

// watchState tracks state for inline watch display.
type watchState struct {
	startTime      time.Time
	nodeCount      int
	lastNodeID     string
	lastUpdateTime time.Time
	recentLogs     []string
	variables      map[string]interface{}
	mu             sync.Mutex
}

// handleInlineEvent processes execution events for inline display.
func handleInlineEvent(cmd *cobra.Command, event execution.ExecutionEvent, state *watchState, isTerm bool) {
	state.mu.Lock()
	defer state.mu.Unlock()

	elapsed := time.Since(state.startTime)

	switch event.Type {
	case execution.EventExecutionStarted:
		timestamp := elapsed.Truncate(time.Millisecond)
		fmt.Fprintf(cmd.OutOrStdout(), "%s ▶ Execution started\n", timestamp)

	case execution.EventNodeStarted:
		timestamp := elapsed.Truncate(time.Millisecond)
		state.lastNodeID = string(event.NodeID)
		fmt.Fprintf(cmd.OutOrStdout(), "%s ▶ %s started\n", timestamp, event.NodeID)

	case execution.EventNodeCompleted:
		timestamp := elapsed.Truncate(time.Millisecond)
		fmt.Fprintf(cmd.OutOrStdout(), "%s ✓ %s completed\n", timestamp, event.NodeID)

	case execution.EventNodeFailed:
		timestamp := elapsed.Truncate(time.Millisecond)
		fmt.Fprintf(cmd.OutOrStdout(), "%s ✗ %s failed: %v\n", timestamp, event.NodeID, event.Error)

	case execution.EventVariableChanged:
		state.variables = event.Variables
	}

	state.lastUpdateTime = time.Now()
}

// displayInlineProgress shows current execution progress in place (using ANSI codes).
func displayInlineProgress(cmd *cobra.Command, progress execution.ExecutionProgress, state *watchState) {
	state.mu.Lock()
	defer state.mu.Unlock()

	// Use ANSI escape codes to update in place
	// Move cursor up and clear line
	fmt.Fprintf(cmd.OutOrStdout(), "\r\033[K")

	status := fmt.Sprintf("Status: Running | Progress: %.0f%% (%d/%d nodes)",
		progress.PercentComplete,
		progress.CompletedNodes+progress.FailedNodes+progress.SkippedNodes,
		progress.TotalNodes)

	if progress.CurrentNode != "" {
		status += fmt.Sprintf(" | Current: %s", progress.CurrentNode)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s", status)
}

// displayFinalResult shows the final execution result.
func displayFinalResult(cmd *cobra.Command, exec *domainexec.Execution, err error, state *watchState, debugMode bool) {
	fmt.Fprintln(cmd.OutOrStdout())

	duration := time.Since(state.startTime)

	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "✗ Workflow failed (%.2fs)\n", duration.Seconds())
		fmt.Fprintf(cmd.OutOrStdout(), "Error: %v\n", err)
	} else if exec != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow completed successfully (%.2fs)\n", duration.Seconds())

		if exec.ReturnValue != nil {
			fmt.Fprintln(cmd.OutOrStdout(), "\nReturn value:")
			returnJSON, _ := json.MarshalIndent(exec.ReturnValue, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(returnJSON))
		}

		if debugMode {
			fmt.Fprintf(cmd.OutOrStderr(), "\nDEBUG: Execution details:\n")
			fmt.Fprintf(cmd.OutOrStderr(), "  Execution ID: %s\n", exec.ID)
			fmt.Fprintf(cmd.OutOrStderr(), "  Duration: %.2fs\n", duration.Seconds())
			fmt.Fprintf(cmd.OutOrStderr(), "  Nodes executed: %d\n", len(exec.NodeExecutions))
		}
	}
}

// displayJSONResult outputs execution result as JSON.
func displayJSONResult(cmd *cobra.Command, exec *domainexec.Execution, err error) {
	result := map[string]interface{}{}

	if exec != nil {
		result["execution_id"] = exec.ID.String()
		result["status"] = string(exec.Status)
		result["started_at"] = exec.StartedAt
		result["completed_at"] = exec.CompletedAt
		result["return_value"] = exec.ReturnValue

		if !exec.CompletedAt.IsZero() {
			duration := exec.CompletedAt.Sub(exec.StartedAt)
			result["duration"] = duration.Seconds()
		}
	}

	if err != nil {
		result["error"] = err.Error()
	}

	// FR-022: Check marshal error before using output
	output, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		// If marshaling fails (e.g., channel, func in return_value),
		// create a fallback error response
		result["marshal_error"] = marshalErr.Error()
		result["return_value"] = fmt.Sprintf("<unmarshalable: %T>", exec.ReturnValue)
		// Try again with the error info
		output, _ = json.MarshalIndent(result, "", "  ")
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(output))
}
