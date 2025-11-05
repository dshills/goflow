package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewValidateCommand creates the validate command
func NewValidateCommand() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "validate <workflow-name>",
		Short: "Validate a workflow",
		Long: `Validate a workflow file for correctness.

This checks:
- Workflow structure and syntax
- Node connectivity (all nodes reachable from start)
- No circular dependencies
- Variable consistency
- Server registrations
- Node configurations

Examples:
  goflow validate my-workflow
  goflow validate my-workflow --verbose`,
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
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Failed to parse workflow YAML")
				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
				}
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Workflow YAML parsed successfully")

			// Validate workflow structure
			if err := wf.Validate(); err != nil {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Workflow validation failed")
				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
				}
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Workflow structure valid")

			// Check for start node
			hasStart := false
			for _, node := range wf.Nodes {
				if node.Type() == "start" {
					hasStart = true
					break
				}
			}
			if hasStart {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Start node found")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ No start node found")
				return fmt.Errorf("workflow must have a start node")
			}

			// Check for end node
			hasEnd := false
			for _, node := range wf.Nodes {
				if node.Type() == "end" {
					hasEnd = true
					break
				}
			}
			if hasEnd {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ End node found")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ No end node found")
				return fmt.Errorf("workflow must have at least one end node")
			}

			// Check node reachability (already done in Validate, but report it)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ All nodes reachable")

			// Check for circular dependencies (already done in Validate)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ No circular dependencies")

			// Validate variable types
			if len(wf.Variables) > 0 {
				allValid := true
				for _, variable := range wf.Variables {
					// Skip variables without type (workflow under construction)
					if variable.Type == "" {
						continue
					}
					if err := variable.Validate(); err != nil {
						allValid = false
						if verbose {
							_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Variable '%s': %v\n", variable.Name, err)
						}
					}
				}
				if allValid {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Variable types consistent")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Some variables have type errors")
					return fmt.Errorf("variable validation failed")
				}
			}

			// Check server registrations
			if len(wf.ServerConfigs) > 0 {
				config, err := loadServersConfig()
				if err != nil {
					if verbose {
						_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Warning: Could not load server config: %v\n", err)
					}
				} else {
					allRegistered := true
					for _, serverCfg := range wf.ServerConfigs {
						if _, exists := config.Servers[serverCfg.ID]; !exists {
							allRegistered = false
							if verbose {
								_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Server not registered: %s\n", serverCfg.ID)
							}
						}
					}
					if allRegistered {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ All servers registered")
					} else {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "⚠ Some servers not registered (workflow will still load)")
						if !verbose {
							_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  Use --verbose to see details")
						}
					}
				}
			}

			// Summary
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n✓ Workflow validation passed")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workflow '%s' is valid and ready to execute\n", workflowName)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed validation information")

	return cmd
}
