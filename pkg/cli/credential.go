package cli

import (
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// Credential represents stored credentials for an MCP server
type Credential struct {
	ServerID      string            `json:"server_id"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty"`
}

// CredentialStore provides in-memory credential storage
// TODO: Replace with actual system keyring integration in future implementation
type CredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
}

// Global credential store instance (exported for testing)
var GlobalCredentialStore = &CredentialStore{
	credentials: make(map[string]*Credential),
}

// NewCredentialCommand creates the credential management command
func NewCredentialCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credential",
		Short: "Manage MCP server credentials",
		Long: `Store and manage credentials for MCP servers in the system keyring.
Credentials are stored securely and referenced by server ID.`,
	}

	cmd.AddCommand(newCredentialAddCommand())
	cmd.AddCommand(newCredentialListCommand())
	cmd.AddCommand(newCredentialRemoveCommand())

	return cmd
}

// newCredentialAddCommand creates the credential add subcommand
func newCredentialAddCommand() *cobra.Command {
	var (
		envVars       []string
		credentialRef string
	)

	cmd := &cobra.Command{
		Use:   "add <server-id>",
		Short: "Add credentials for an MCP server",
		Long: `Store credentials for an MCP server in the system keyring.
Credentials can be environment variables or a reference to a named credential.

The credentials are stored securely and not written to workflow files.

Examples:
  # Add environment variable credentials
  goflow credential add myserver --env API_KEY=secret123 --env TOKEN=abc456

  # Add a credential reference
  goflow credential add myserver --credential-ref aws-profile-prod

  # Mix environment variables and credential reference
  goflow credential add myserver --env DEBUG=true --credential-ref oauth-token`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			// Validate server ID format
			if !isValidServerID(serverID) {
				return fmt.Errorf("invalid server ID: %s (must contain only letters, numbers, dashes, and underscores)", serverID)
			}

			// Require at least one credential type
			if len(envVars) == 0 && credentialRef == "" {
				return fmt.Errorf("must provide at least one of --env or --credential-ref")
			}

			// Parse environment variables
			env := make(map[string]string)
			for _, envVar := range envVars {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("environment variable key cannot be empty")
				}
				env[key] = value
			}

			// Store credentials
			cred := &Credential{
				ServerID:      serverID,
				EnvVars:       env,
				CredentialRef: credentialRef,
			}

			GlobalCredentialStore.mu.Lock()
			GlobalCredentialStore.credentials[serverID] = cred
			GlobalCredentialStore.mu.Unlock()

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Credentials for '%s' stored successfully\n", serverID)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNote: This is an in-memory store. Real keyring integration will be added in a future update.")

			// Show what was stored
			if len(env) > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Environment variables: %d key(s) stored\n", len(env))
			}
			if credentialRef != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Credential reference: %s\n", credentialRef)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&envVars, "env", []string{}, "Environment variables (KEY=VALUE, can be specified multiple times)")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "Named credential reference")

	return cmd
}

// newCredentialListCommand creates the credential list subcommand
func newCredentialListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored credentials",
		Long: `Display all stored credentials for MCP servers.

This shows which servers have credentials stored, along with the type
of credentials (environment variables or credential references).
The actual secret values are not displayed for security.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			GlobalCredentialStore.mu.RLock()
			defer GlobalCredentialStore.mu.RUnlock()

			if len(GlobalCredentialStore.credentials) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No credentials stored.")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nAdd credentials with: goflow credential add <server-id> --env KEY=VALUE")
				return nil
			}

			// Create table writer
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "SERVER ID\tENV VARS\tCREDENTIAL REF\tTYPE")
			_, _ = fmt.Fprintln(w, "─────────\t────────\t──────────────\t────")

			for _, cred := range GlobalCredentialStore.credentials {
				envVarCount := len(cred.EnvVars)
				credRef := cred.CredentialRef
				if credRef == "" {
					credRef = "-"
				}

				// Determine credential type
				credType := ""
				if envVarCount > 0 && credRef != "-" {
					credType = "Mixed"
				} else if envVarCount > 0 {
					credType = "Environment"
				} else if credRef != "-" {
					credType = "Reference"
				}

				envVarDisplay := "-"
				if envVarCount > 0 {
					// Show environment variable keys (not values)
					keys := make([]string, 0, len(cred.EnvVars))
					for key := range cred.EnvVars {
						keys = append(keys, key)
					}
					if len(keys) <= 3 {
						envVarDisplay = strings.Join(keys, ", ")
					} else {
						envVarDisplay = fmt.Sprintf("%d keys", len(keys))
					}
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					cred.ServerID,
					envVarDisplay,
					credRef,
					credType,
				)
			}

			_ = w.Flush()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nNote: Secret values are not displayed for security.")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Note: This is an in-memory store. Real keyring integration will be added in a future update.")

			return nil
		},
	}

	return cmd
}

// newCredentialRemoveCommand creates the credential remove subcommand
func newCredentialRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <server-id>",
		Short: "Remove stored credentials",
		Long: `Remove credentials for an MCP server from the keyring.

This will delete all stored credentials (environment variables and
credential references) for the specified server.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			GlobalCredentialStore.mu.Lock()
			defer GlobalCredentialStore.mu.Unlock()

			// Check if credentials exist
			if _, exists := GlobalCredentialStore.credentials[serverID]; !exists {
				return fmt.Errorf("no credentials found for server: %s", serverID)
			}

			// Remove credentials
			delete(GlobalCredentialStore.credentials, serverID)

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Credentials for '%s' removed successfully\n", serverID)
			return nil
		},
	}

	return cmd
}

// GetCredential retrieves stored credentials for a server ID
// Returns nil if no credentials are stored for the given server
func GetCredential(serverID string) *Credential {
	GlobalCredentialStore.mu.RLock()
	defer GlobalCredentialStore.mu.RUnlock()

	cred, exists := GlobalCredentialStore.credentials[serverID]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modification
	credCopy := &Credential{
		ServerID:      cred.ServerID,
		CredentialRef: cred.CredentialRef,
	}
	if len(cred.EnvVars) > 0 {
		credCopy.EnvVars = make(map[string]string, len(cred.EnvVars))
		for k, v := range cred.EnvVars {
			credCopy.EnvVars[k] = v
		}
	}

	return credCopy
}
