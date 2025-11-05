package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInitCommand creates the init command
func NewInitCommand() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "init <workflow-name>",
		Short: "Initialize a new workflow",
		Long: `Create a new workflow file with a basic template.

The workflow is created in ~/.goflow/workflows/<workflow-name>.yaml

Examples:
  goflow init my-workflow
  goflow init data-pipeline --description "ETL pipeline for customer data"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]

			// Construct workflow path
			workflowPath := filepath.Join(GetWorkflowsDir(), workflowName+".yaml")

			// Check if workflow already exists
			if _, err := os.Stat(workflowPath); err == nil {
				return fmt.Errorf("workflow already exists: %s", workflowName)
			}

			// Create default workflow template
			template := map[string]interface{}{
				"version":     "1.0",
				"name":        workflowName,
				"description": description,
				"metadata": map[string]interface{}{
					"author":  os.Getenv("USER"),
					"created": time.Now().Format(time.RFC3339),
					"tags":    []string{},
				},
				"variables": []map[string]interface{}{},
				"servers":   []map[string]interface{}{},
				"nodes": []map[string]interface{}{
					{
						"id":   "start",
						"type": "start",
					},
					{
						"id":   "end",
						"type": "end",
					},
				},
				"edges": []map[string]interface{}{
					{
						"from": "start",
						"to":   "end",
					},
				},
			}

			// Marshal to YAML
			data, err := yaml.Marshal(template)
			if err != nil {
				return fmt.Errorf("failed to create workflow template: %w", err)
			}

			// Write to file
			if err := os.WriteFile(workflowPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write workflow file: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow '%s' created successfully\n", workflowName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nWorkflow file: %s\n", workflowPath)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  1. Edit the workflow: $EDITOR %s\n", workflowPath)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  2. Validate: goflow validate %s\n", workflowName)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  3. Execute: goflow run %s\n", workflowName)

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Workflow description")

	return cmd
}
