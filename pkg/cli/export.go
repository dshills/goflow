package cli

import (
	"fmt"
	"os"

	"github.com/dshills/goflow/pkg/workflow"
	"github.com/spf13/cobra"
)

// NewExportCommand creates the export command
func NewExportCommand() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "export <workflow-file>",
		Short: "Export a workflow with credentials stripped for sharing",
		Long: `Export a workflow to YAML format with sensitive credentials removed.

This command:
- Strips sensitive environment variables from server configurations
- Replaces credential references with placeholders
- Adds warning comments about required credentials
- Outputs to stdout or a file

The exported workflow can be safely shared without exposing secrets.
Recipients must configure their own credentials before running it.

Examples:
  # Export to stdout
  goflow export my-workflow.yaml

  # Export to a file
  goflow export my-workflow.yaml --output shared-workflow.yaml

  # Export from workflows directory
  goflow export ~/.goflow/workflows/my-workflow.yaml -o exported.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowPath := args[0]

			// Check if workflow file exists
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				return fmt.Errorf("workflow file not found: %s", workflowPath)
			}

			// Load workflow from file
			wf, err := LoadWorkflowFromFile(workflowPath)
			if err != nil {
				return fmt.Errorf("failed to load workflow: %w", err)
			}

			// Export workflow with credentials stripped
			exportedYAML, err := workflow.Export(wf)
			if err != nil {
				return fmt.Errorf("failed to export workflow: %w", err)
			}

			// Write to output
			if outputPath != "" {
				// Write to file
				if err := os.WriteFile(outputPath, exportedYAML, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow exported successfully to: %s\n", outputPath)
				fmt.Fprintln(cmd.OutOrStdout(), "  Credentials have been stripped for safe sharing")
			} else {
				// Write to stdout
				fmt.Fprint(cmd.OutOrStdout(), string(exportedYAML))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: stdout)")

	return cmd
}
