package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/spf13/cobra"
)

// NewImportCommand creates the import command
func NewImportCommand() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "import <workflow-file>",
		Short: "Import a workflow from a file",
		Long: `Import a workflow from a YAML file and validate it.

This command:
- Loads the workflow file
- Validates workflow version compatibility
- Checks server references against registry
- Detects credential placeholders
- Validates workflow structure
- Saves to workflows directory

The imported workflow is saved in ~/.goflow/workflows/<workflow-name>.yaml

Examples:
  goflow import /path/to/workflow.yaml
  goflow import ./my-workflow.yaml --verbose`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowFile := args[0]

			// Check if workflow file exists
			if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
				return fmt.Errorf("workflow file not found: %s", workflowFile)
			}

			// Load server config and populate registry
			registry := mcpserver.NewRegistry()
			serverConfig, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load server config: %w", err)
			}

			// Populate registry with servers from config
			for _, entry := range serverConfig.Servers {
				// Convert string transport to TransportType
				transportType := mcpserver.TransportType(entry.Transport)
				if entry.Transport == "" {
					transportType = mcpserver.TransportStdio // Default to stdio
				}

				server, err := mcpserver.NewMCPServer(entry.ID, entry.Command, entry.Args, transportType)
				if err != nil {
					return fmt.Errorf("failed to create server %s: %w", entry.ID, err)
				}

				// Set name if provided
				if entry.Name != "" {
					server.Name = entry.Name
				}

				if err := registry.Register(server); err != nil {
					return fmt.Errorf("failed to register server %s: %w", entry.ID, err)
				}
			}

			// Import workflow using ImportWorkflow function
			wf, err := ImportWorkflow(workflowFile, registry)

			// Handle different error types
			if err != nil {
				// Check for MissingServerError
				var missingServerErr *MissingServerError
				if errors.As(err, &missingServerErr) {
					_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Workflow references missing servers")
					_, _ = fmt.Fprintln(cmd.OutOrStderr(), "\nMissing servers:")
					for _, serverID := range missingServerErr.MissingServers {
						_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  - %s\n", serverID)
					}
					_, _ = fmt.Fprintln(cmd.OutOrStderr(), "\nPlease register these servers before importing:")
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  goflow server add <server-id> <command> [args...]\n")
					return err
				}

				// Check for CredentialPlaceholderWarning
				var credentialWarning *CredentialPlaceholderWarning
				if errors.As(err, &credentialWarning) {
					// This is a warning, not a fatal error - continue with import
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "⚠ Workflow contains credential placeholders")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nServers with placeholders:")
					for _, serverID := range credentialWarning.ServersWithPlaceholders {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", serverID)
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nYou will need to configure credentials before execution.")
					// Continue with import despite warning
				} else {
					// Check for IncompatibleVersionError
					var versionErr *IncompatibleVersionError
					if errors.As(err, &versionErr) {
						_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Incompatible workflow version")
						if verbose {
							_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Workflow version: %s\n", versionErr.WorkflowVersion)
							_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Supported versions: %v\n", versionErr.SupportedVersions)
						}
						return err
					}

					// Other errors
					_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Failed to import workflow")
					if verbose {
						_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
					}
					return err
				}
			}

			// At this point, wf should not be nil
			if wf == nil {
				return fmt.Errorf("workflow import failed unexpectedly")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Workflow imported successfully")

			// Validate the workflow
			if err := wf.Validate(); err != nil {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Workflow validation failed")
				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
				}
				return err
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Workflow validation passed")

			// Save workflow to workflows directory
			workflowPath := filepath.Join(GetWorkflowsDir(), wf.Name+".yaml")

			// Check if workflow already exists
			if _, err := os.Stat(workflowPath); err == nil {
				return fmt.Errorf("workflow already exists: %s\n\nLocation: %s\nUse a different name or remove the existing workflow first", wf.Name, workflowPath)
			}

			// Copy the file to workflows directory
			sourceData, err := os.ReadFile(workflowFile)
			if err != nil {
				return fmt.Errorf("failed to read source workflow file: %w", err)
			}

			if err := os.WriteFile(workflowPath, sourceData, 0644); err != nil {
				return fmt.Errorf("failed to write workflow file: %w", err)
			}

			// Print success message
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Workflow '%s' imported successfully\n", wf.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Location: %s\n", workflowPath)

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  1. Edit the workflow: goflow edit %s\n", wf.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  2. Validate: goflow validate %s\n", wf.Name)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  3. Execute: goflow run %s\n", wf.Name)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed import information")

	return cmd
}
