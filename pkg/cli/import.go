package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/spf13/cobra"
)

// NewImportCommand creates the import command
func NewImportCommand() *cobra.Command {
	var (
		verbose    bool
		name       string
		noInteract bool
	)

	cmd := &cobra.Command{
		Use:   "import <workflow-file>",
		Short: "Import a workflow from a file",
		Long: `Import a workflow from a YAML file and validate it.

This command:
- Loads the workflow file
- Validates workflow version compatibility
- Checks server references against registry
- Interactively configures missing servers (if any)
- Detects credential placeholders
- Validates workflow structure
- Saves to workflows directory

The imported workflow is saved in ~/.goflow/workflows/<workflow-name>.yaml

Examples:
  goflow import /path/to/workflow.yaml
  goflow import ./my-workflow.yaml --verbose
  goflow import shared-workflow.yaml --name my-workflow
  goflow import workflow.yaml --no-interact  # Skip interactive prompts`,
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
					// Handle missing servers interactively if allowed
					if !noInteract {
						if err := handleMissingServers(cmd, missingServerErr.MissingServers, serverConfig); err != nil {
							return err
						}
						// Reload registry with newly added servers
						for _, serverID := range missingServerErr.MissingServers {
							if entry, exists := serverConfig.Servers[serverID]; exists {
								transportType := mcpserver.TransportType(entry.Transport)
								if entry.Transport == "" {
									transportType = mcpserver.TransportStdio
								}
								server, err := mcpserver.NewMCPServer(entry.ID, entry.Command, entry.Args, transportType)
								if err != nil {
									return fmt.Errorf("failed to create server %s: %w", entry.ID, err)
								}
								if entry.Name != "" {
									server.Name = entry.Name
								}
								if err := registry.Register(server); err != nil {
									return fmt.Errorf("failed to register server %s: %w", entry.ID, err)
								}
							}
						}
						// Retry import after adding servers
						wf, err = ImportWorkflow(workflowFile, registry)
						if err != nil {
							// Check again for credential warnings (expected)
							var credentialWarning *CredentialPlaceholderWarning
							if !errors.As(err, &credentialWarning) {
								return fmt.Errorf("import failed after configuring servers: %w", err)
							}
							// Continue with credential warning
						}
					} else {
						_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Workflow references missing servers")
						_, _ = fmt.Fprintln(cmd.OutOrStderr(), "\nMissing servers:")
						for _, serverID := range missingServerErr.MissingServers {
							_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  - %s\n", serverID)
						}
						_, _ = fmt.Fprintln(cmd.OutOrStderr(), "\nPlease register these servers before importing:")
						_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  goflow server add <server-id> <command> [args...]\n")
						return err
					}
				}

				// Check for CredentialPlaceholderWarning
				var credentialWarning *CredentialPlaceholderWarning
				if errors.As(err, &credentialWarning) {
					// This is a warning, not a fatal error - continue with import
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n⚠  Workflow contains credential placeholders")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nServers with placeholders:")
					for _, serverID := range credentialWarning.ServersWithPlaceholders {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", serverID)
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nYou will need to configure credentials before execution.")
					// Continue with import despite warning
				} else {
					// Check for IncompatibleVersionError
					var missingErr *MissingServerError
					var versionErr *IncompatibleVersionError
					if errors.As(err, &missingErr) {
						// This is a MissingServerError - already handled above
					} else if errors.As(err, &versionErr) {
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

			// Override workflow name if specified
			workflowName := wf.Name
			if name != "" {
				workflowName = name
			}

			// Validate the workflow
			if err := wf.Validate(); err != nil {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Workflow validation failed")
				if verbose {
					_, _ = fmt.Fprintf(cmd.OutOrStderr(), "  Error: %v\n", err)
				}
				return err
			}

			// Save workflow to workflows directory
			workflowPath := filepath.Join(GetWorkflowsDir(), workflowName+".yaml")

			// Check if workflow already exists
			if _, err := os.Stat(workflowPath); err == nil {
				return fmt.Errorf("workflow already exists: %s\n\nLocation: %s\nUse --name flag with a different name or remove the existing workflow first", workflowName, workflowPath)
			}

			// Copy the file to workflows directory
			sourceData, err := os.ReadFile(workflowFile)
			if err != nil {
				return fmt.Errorf("failed to read source workflow file: %w", err)
			}

			if err := os.WriteFile(workflowPath, sourceData, 0644); err != nil {
				return fmt.Errorf("failed to write workflow file: %w", err)
			}

			// Count newly added servers
			newServersCount := 0
			var serversWithCredentials []string
			for _, sc := range wf.ServerConfigs {
				// Track servers with credential refs
				if sc.CredentialRef != "" {
					serversWithCredentials = append(serversWithCredentials, sc.ID)
				}
			}

			// Print success message
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Imported workflow '%s' successfully\n", workflowName)
			if newServersCount > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - Added %d new server configuration(s)\n", newServersCount)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - Workflow saved to: %s\n", workflowPath)

			// Show next steps if there are credentials to configure
			if len(serversWithCredentials) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n⚠  Required setup:")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  1. Configure credentials:")
				for _, serverID := range serversWithCredentials {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "     goflow credential add %s --key <credential-key>\n", serverID)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n  2. Test servers:")
				for _, serverID := range serversWithCredentials {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "     goflow server test %s\n", serverID)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n  3. Run workflow:\n")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "     goflow run %s\n", workflowName)
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  1. Validate: goflow validate %s\n", workflowName)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  2. Run: goflow run %s\n", workflowName)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed import information")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Override workflow name")
	cmd.Flags().BoolVar(&noInteract, "no-interact", false, "Skip interactive prompts for missing servers")

	return cmd
}

// handleMissingServers prompts the user to configure missing servers interactively
func handleMissingServers(cmd *cobra.Command, missingServers []string, serverConfig *ServerConfig) error {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n⚠  Missing server configurations detected:")
	for _, serverID := range missingServers {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", serverID)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nWould you like to configure these servers now? (y/n)")

	reader := bufio.NewReader(cmd.InOrStdin())
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		return fmt.Errorf("workflow import cancelled - missing server configurations")
	}

	// Configure each missing server
	for _, serverID := range missingServers {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n--- Configuring server: %s ---\n", serverID)

		// Prompt for command
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Command (e.g., node, python, npx): ")
		command, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read command: %w", err)
		}
		command = strings.TrimSpace(command)
		if command == "" {
			return fmt.Errorf("command cannot be empty")
		}

		// Prompt for args
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Args (space-separated, or press Enter to skip): ")
		argsInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read args: %w", err)
		}
		argsInput = strings.TrimSpace(argsInput)
		var args []string
		if argsInput != "" {
			args = strings.Fields(argsInput)
		}

		// Prompt for transport
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Transport (stdio/sse/http) [default: stdio]: ")
		transport, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read transport: %w", err)
		}
		transport = strings.TrimSpace(strings.ToLower(transport))
		if transport == "" {
			transport = "stdio"
		}
		// Validate transport
		if transport != "stdio" && transport != "sse" && transport != "http" {
			return fmt.Errorf("invalid transport: %s (must be stdio, sse, or http)", transport)
		}

		// Prompt for optional name
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Friendly name [default: %s]: ", serverID)
		serverName, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read name: %w", err)
		}
		serverName = strings.TrimSpace(serverName)
		if serverName == "" {
			serverName = serverID
		}

		// Prompt for optional description
		_, _ = fmt.Fprint(cmd.OutOrStdout(), "Description (optional): ")
		description, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read description: %w", err)
		}
		description = strings.TrimSpace(description)

		// Create server entry
		serverEntry := &ServerEntry{
			ID:          serverID,
			Name:        serverName,
			Description: description,
			Command:     command,
			Args:        args,
			Transport:   transport,
			Env:         make(map[string]string),
		}

		// Add to config
		serverConfig.Servers[serverID] = serverEntry

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Server '%s' configured\n", serverID)
	}

	// Save server config
	if err := saveServersConfig(serverConfig); err != nil {
		return fmt.Errorf("failed to save server configuration: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Saved %d server configuration(s)\n", len(missingServers))
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNote: Credentials must be configured separately using:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  goflow credential add <server-id> --key <credential-key>")

	return nil
}
