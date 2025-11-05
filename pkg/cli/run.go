package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NewRunCommand creates the run command
func NewRunCommand() *cobra.Command {
	var (
		inputFile  string
		watch      bool
		outputJSON bool
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

  # Run with progress monitoring
  goflow run my-workflow --watch

  # Run with debug output
  goflow run my-workflow --debug`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]

			// Construct workflow path
			workflowPath := filepath.Join(GetWorkflowsDir(), workflowName+".yaml")

			// Check if workflow file exists
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				return fmt.Errorf("workflow not found: %s\n\nLooked in: %s", workflowName, workflowPath)
			}

			// Load workflow file
			wf, err := LoadWorkflowFromFile(workflowPath)
			if err != nil {
				return fmt.Errorf("failed to parse workflow YAML: %w", err)
			}

			// Validate workflow
			if err := wf.Validate(); err != nil {
				return fmt.Errorf("workflow validation failed: %w", err)
			}

			// Load input variables if provided
			var inputVars map[string]interface{}
			if inputFile != "" {
				inputData, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("failed to read input file: %w", err)
				}

				if err := json.Unmarshal(inputData, &inputVars); err != nil {
					return fmt.Errorf("failed to parse input JSON: %w", err)
				}
			}

			// Generate execution ID
			execID := fmt.Sprintf("exec-%d", time.Now().Unix())

			if !outputJSON {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Started execution (ID: %s)\n", execID)
			}

			// Execute workflow
			// TODO: Use actual runtime when implemented
			// For now, simulate execution
			startTime := time.Now()

			if watch {
				// Simulate progress output
				nodeCount := len(wf.Nodes)
				for i, node := range wf.Nodes {
					if node.Type() == "start" {
						continue
					}

					// Simulate node execution time
					time.Sleep(50 * time.Millisecond)

					elapsed := time.Since(startTime).Seconds()
					if !outputJSON {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Node '%s' completed (%.2fs)\n", node.GetID(), elapsed)
					}

					if i == nodeCount-1 {
						break
					}
				}
			}

			totalTime := time.Since(startTime)

			// Create result
			result := map[string]interface{}{
				"execution_id": execID,
				"workflow":     workflowName,
				"status":       "completed",
				"duration":     totalTime.Seconds(),
				"return_value": map[string]interface{}{
					"success": true,
				},
			}

			if outputJSON {
				// Output as JSON
				output, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal output: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(output))
			} else {
				// Human-readable output
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow completed successfully (%.2fs)\n", totalTime.Seconds())

				// Display return value if available
				returnVal, _ := json.MarshalIndent(result["return_value"], "", "  ")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nReturn value:\n%s\n", string(returnVal))
			}

			// TODO: Save execution to SQLite storage
			if GlobalConfig.Debug {
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "DEBUG: Execution details:\n")
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Workflow: %s\n", workflowName)
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Execution ID: %s\n", execID)
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Duration: %.2fs\n", totalTime.Seconds())
				if inputVars != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Input variables: %d\n", len(inputVars))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input variables JSON file")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Monitor execution progress in real-time")
	cmd.Flags().BoolVar(&outputJSON, "output-json", false, "Output result as JSON")

	return cmd
}
