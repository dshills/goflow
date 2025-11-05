package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/goflow/pkg/tui"
	"github.com/spf13/cobra"
)

// NewEditCommand creates the edit command
func NewEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [workflow-name]",
		Short: "Edit a workflow in the TUI",
		Long: `Launch the TUI (Terminal User Interface) to edit a workflow visually.

If a workflow name is provided, it will be loaded directly into the workflow builder.
If no workflow name is provided, the TUI will open in workflow explorer mode, allowing
you to browse and select a workflow to edit.

The TUI provides:
- Visual workflow builder with node and edge management
- Real-time validation
- Vim-style keyboard navigation (h/j/k/l)
- Context-sensitive help (press ?)

Examples:
  goflow edit                     # Launch TUI in explorer mode
  goflow edit my-workflow         # Edit specific workflow`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine workflow to edit (if provided)
			var workflowName string

			if len(args) > 0 {
				workflowName = args[0]

				// Construct workflow path
				workflowPath := filepath.Join(GetWorkflowsDir(), workflowName+".yaml")

				// Check if workflow exists
				if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
					return fmt.Errorf("workflow not found: %s\n\nLooked in: %s\n\nCreate it with: goflow init %s",
						workflowName, workflowPath, workflowName)
				}

				// Load workflow from file to verify it's valid
				_, err := LoadWorkflowFromFile(workflowPath)
				if err != nil {
					return fmt.Errorf("failed to load workflow: %w\n\nTip: Run 'goflow validate %s' for detailed error information",
						err, workflowName)
				}
			}

			// Initialize TUI application
			app, err := tui.NewApp()
			if err != nil {
				return fmt.Errorf("failed to initialize TUI: %w", err)
			}
			defer app.Close()

			// If a workflow was specified, configure the builder view
			if workflowName != "" {
				view, err := app.GetViewManager().GetView("builder")
				if err != nil {
					return fmt.Errorf("failed to get builder view: %w", err)
				}

				if builderView, ok := view.(*tui.WorkflowBuilderView); ok {
					builderView.SetWorkflow(workflowName)
				}

				// Switch to builder view
				if err := app.GetViewManager().SwitchTo("builder"); err != nil {
					return fmt.Errorf("failed to switch to builder view: %w", err)
				}
			} else {
				// Start in explorer view (already initialized by NewApp)
				// No additional configuration needed
			}

			// Run the TUI application
			if err := app.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}

			// Success message after TUI exits
			if workflowName != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nWorkflow '%s' editing session completed\n", workflowName)
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nTUI session completed")
			}

			return nil
		},
	}

	return cmd
}
