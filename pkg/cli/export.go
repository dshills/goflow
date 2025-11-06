package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/goflow/pkg/workflow"
	"github.com/spf13/cobra"
)

// NewExportCommand creates the export command
func NewExportCommand() *cobra.Command {
	var (
		outputFile string
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:   "export <workflow-name>",
		Short: "Export a workflow for sharing",
		Long: `Export a workflow with credentials stripped for safe sharing.

This command:
- Loads the workflow from the workflows directory
- Strips sensitive credentials and environment variables
- Replaces credential references with placeholders
- Outputs sanitized YAML to stdout or a file

The exported workflow is safe to share publicly, but recipients must
configure their own credentials before execution.

Sensitive environment variables detected by patterns (KEY, SECRET, TOKEN,
PASSWORD, etc.) are removed. Non-sensitive variables (HOST, PORT, etc.)
are preserved.

Examples:
  # Export to stdout
  goflow export my-workflow

  # Export to a file
  goflow export my-workflow -o shared-workflow.yaml
  goflow export my-workflow --output /path/to/workflow.yaml`,
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

			// Validate workflow before export
			if err := wf.Validate(); err != nil {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "⚠ Warning: Workflow validation failed")
				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "  Continuing with export...")
			}

			// Export workflow (strips credentials)
			yamlBytes, err := workflow.Export(wf)
			if err != nil {
				return fmt.Errorf("failed to export workflow: %w", err)
			}

			// Write to output file or stdout
			if outputFile != "" {
				// Write to file
				if err := os.WriteFile(outputFile, yamlBytes, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}

				// Success message
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Exported workflow '%s' successfully\n", workflowName)

				// Count servers with credentials
				credentialCount := 0
				serversWithCredentials := []string{}
				for _, sc := range wf.ServerConfigs {
					if sc.CredentialRef != "" || hasCredentialEnvVars(sc.Env) {
						credentialCount++
						serversWithCredentials = append(serversWithCredentials, sc.ID)
					}
				}

				if credentialCount > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - Removed credentials from %d server configuration(s)\n", credentialCount)
				}

				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - Output: %s\n", outputFile)

				// Show warning if credentials were present
				if credentialCount > 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n⚠ Warning: Workflow contains credential references. Recipients must configure:")
					for _, serverID := range serversWithCredentials {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", serverID)
					}
				}

				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nWorkflow details:\n")
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Name: %s\n", wf.Name)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Version: %s\n", wf.Version)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Nodes: %d\n", len(wf.Nodes))
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Servers: %d\n", len(wf.ServerConfigs))
				}
			} else {
				// Write to stdout
				_, _ = fmt.Fprint(cmd.OutOrStdout(), string(yamlBytes))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed export information")

	return cmd
}

// hasCredentialEnvVars checks if a server's environment variables contain sensitive credentials
func hasCredentialEnvVars(env map[string]string) bool {
	if len(env) == 0 {
		return false
	}

	sensitivePatterns := []string{
		"KEY", "SECRET", "TOKEN", "PASSWORD", "PASSPHRASE",
		"CREDENTIAL", "AUTH", "BEARER", "PRIVATE", "CLIENT_SECRET",
	}

	for key := range env {
		upperKey := strings.ToUpper(key)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(upperKey, pattern) {
				return true
			}
		}

		// Check for database URLs and connection strings
		if strings.Contains(upperKey, "DATABASE") && strings.Contains(upperKey, "URL") {
			return true
		}
		if strings.Contains(upperKey, "CONN") && strings.Contains(upperKey, "STRING") {
			return true
		}
		if strings.Contains(upperKey, "DSN") {
			return true
		}
		if strings.Contains(upperKey, "OAUTH") {
			return true
		}
	}

	return false
}
