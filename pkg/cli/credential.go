package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"
	"unicode/utf8"

	"github.com/dshills/goflow/pkg/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxCredentialSize = 1 << 20 // 1MB limit for all credential inputs

// isOnlyWhitespace checks if a byte slice contains only Unicode whitespace characters
// without allocating strings. Returns true if empty or whitespace-only.
func isOnlyWhitespace(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 is treated as non-whitespace
			return false
		}
		if !unicode.IsSpace(r) {
			return false
		}
		i += size
	}
	return true
}

// NewCredentialCommand creates the credential management command
func NewCredentialCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credential",
		Short: "Manage server credentials",
		Long: `Manage credentials for MCP servers securely in the system keyring.
Credentials are stored in your system's native credential store (Keychain on macOS,
Credential Manager on Windows, Secret Service on Linux) and never in plain text files.`,
	}

	cmd.AddCommand(newCredentialAddCommand())
	cmd.AddCommand(newCredentialListCommand())

	return cmd
}

// newCredentialAddCommand creates the credential add subcommand
func newCredentialAddCommand() *cobra.Command {
	var (
		key      string
		value    string
		useStdin bool
	)

	cmd := &cobra.Command{
		Use:   "add <server-id>",
		Short: "Add a credential for a server",
		Long: `Add a credential for an MCP server. The credential is stored securely in your
system keyring and referenced by the server configuration.

Examples:
  # Add credential with interactive password prompt (recommended for local use)
  goflow credential add api-server --key api-key

  # Add credential from stdin (recommended for automation/CI/CD)
  printf '%s' "$API_KEY" | goflow credential add api-server --key api-key --stdin
  cat /run/secrets/api-key | goflow credential add api-server --key api-key --stdin

  # Add credential with value in command (NOT recommended - visible in shell history)
  goflow credential add db-server --key password --value secret123

Security:
  - Credentials are stored in your system keyring (never in plain text)
  - Use interactive prompt for local use (avoids shell history)
  - Use --stdin for automation (avoids process list exposure, max 1MB)
  - Avoid --value flag (visible in shell history and process list)
  - Credential values are never displayed by GoFlow commands

Note:
  - All input methods have a 1MB maximum credential size limit
  - --stdin reads until EOF and preserves leading/trailing spaces
  - Only trailing CR/LF characters are removed; other whitespace is preserved
  - Use printf '%s' to avoid adding a trailing newline
  - To send EOF: Ctrl-D on Unix/Linux/macOS, Ctrl-Z then Enter on Windows
  - Whitespace-only credentials are rejected (includes all Unicode whitespace)
  - Input buffers are zeroed after reading (best-effort; Go strings are immutable)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			// Validate key is provided
			if key == "" {
				return fmt.Errorf("credential key is required (use --key flag)")
			}

			// Load servers config to verify server exists
			config, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load servers config: %w", err)
			}

			if _, exists := config.Servers[serverID]; !exists {
				return fmt.Errorf("server not found: %s\nRegister the server first with: goflow server add %s <command> [args...]", serverID, serverID)
			}

			// Create credential store
			credStore := storage.NewKeyringCredentialStore()

			// Construct credential key (server-id:key-name)
			credentialKey := fmt.Sprintf("%s:%s", serverID, key)

			// Check if credential already exists
			_, err = credStore.Get(credentialKey)
			if err == nil {
				// Credential exists - confirm overwrite
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Warning: Credential '%s' for server '%s' already exists.\n", key, serverID)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), "Overwrite? [y/N]: ")

				var response string
				_, _ = fmt.Fscanln(os.Stdin, &response)
				response = strings.ToLower(strings.TrimSpace(response))

				if response != "y" && response != "yes" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			// Get credential value
			var credValue string
			if useStdin {
				// Read from stdin (for automation/CI/CD)
				// Limit stdin reading to prevent memory exhaustion
				limitedReader := io.LimitReader(cmd.InOrStdin(), maxCredentialSize+1)
				inputBytes, err := io.ReadAll(limitedReader)

				// Ensure buffer is zeroed on all exit paths
				defer func() {
					for i := range inputBytes {
						inputBytes[i] = 0
					}
				}()

				if err != nil {
					return fmt.Errorf("failed to read from stdin: %w", err)
				}

				// Check if credential exceeded size limit
				if len(inputBytes) > maxCredentialSize {
					return fmt.Errorf("credential value exceeds maximum size of %d bytes - if you need larger credentials, consider using a secret file reference", maxCredentialSize)
				}

				// Trim only trailing newline characters using bytes (preserve intentional spaces)
				trimmed := bytes.TrimRight(inputBytes, "\r\n")

				// Validate non-empty and not whitespace-only using Unicode-aware checks
				// This avoids creating temporary strings that can't be zeroed
				if len(trimmed) == 0 {
					return fmt.Errorf("credential value cannot be empty")
				}
				if isOnlyWhitespace(trimmed) {
					return fmt.Errorf("credential cannot contain only whitespace characters")
				}

				// Convert to string only at the last moment for keyring storage
				credValue = string(trimmed)
			} else if value != "" {
				// Value provided via flag (warn about security)
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "Warning: Using --value flag exposes credential in shell history.")
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "Consider using interactive prompt (omit --value) or --stdin for better security.")

				// Validate size
				if len(value) > maxCredentialSize {
					return fmt.Errorf("credential value exceeds maximum size of %d bytes", maxCredentialSize)
				}

				// Validate not whitespace-only
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("credential cannot contain only whitespace characters")
				}

				credValue = value
			} else {
				// Prompt for value securely (no echo)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Enter value for '%s': ", key)

				// Read password without echo
				passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				_, _ = fmt.Fprintln(cmd.OutOrStdout()) // New line after hidden input

				// Zero password bytes on all exit paths
				defer func() {
					for i := range passwordBytes {
						passwordBytes[i] = 0
					}
				}()

				if err != nil {
					return fmt.Errorf("failed to read credential value: %w", err)
				}

				// Validate size
				if len(passwordBytes) > maxCredentialSize {
					return fmt.Errorf("credential value exceeds maximum size of %d bytes", maxCredentialSize)
				}

				credValue = string(passwordBytes)

				// Validate non-empty and not whitespace-only
				if credValue == "" {
					return fmt.Errorf("credential value cannot be empty")
				}
				if strings.TrimSpace(credValue) == "" {
					return fmt.Errorf("credential cannot contain only whitespace characters")
				}
			}

			// Store credential in keyring
			if err := credStore.Set(credentialKey, credValue); err != nil {
				return fmt.Errorf("failed to store credential: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Credential '%s' added for server '%s'\n", key, serverID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Credential key name (required, e.g., 'api-key', 'password', 'token')")
	cmd.Flags().StringVarP(&value, "value", "v", "", "Credential value (optional - will prompt securely if omitted)")
	cmd.Flags().BoolVar(&useStdin, "stdin", false, "Read credential value from stdin (recommended for automation/CI/CD)")

	// Mark key as required
	_ = cmd.MarkFlagRequired("key")

	// Make --stdin and --value mutually exclusive
	cmd.MarkFlagsMutuallyExclusive("stdin", "value")

	return cmd
}

// newCredentialListCommand creates the credential list subcommand
func newCredentialListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [server-id]",
		Short: "List configured credentials",
		Long: `List all configured credentials or credentials for a specific server.
Shows only credential key names, never the actual values.

Examples:
  # List all credentials
  goflow credential list

  # List credentials for a specific server
  goflow credential list api-server

Security:
  - Only credential key names are displayed
  - Actual credential values are never shown
  - Use 'goflow server show <server-id>' to see which credentials a server references`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var filterServerID string
			if len(args) > 0 {
				filterServerID = args[0]
			}

			// Create credential store
			credStore := storage.NewKeyringCredentialStore()

			// Get all credential keys
			keys, err := credStore.List()
			if err != nil {
				return fmt.Errorf("failed to list credentials: %w", err)
			}

			// Filter out the internal index key
			credentialKeys := make([]string, 0, len(keys))
			for _, k := range keys {
				if k != "__goflow_index__" {
					credentialKeys = append(credentialKeys, k)
				}
			}

			if len(credentialKeys) == 0 {
				if filterServerID != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No credentials configured for server '%s'.\n", filterServerID)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nAdd a credential with: goflow credential add %s --key <name>\n", filterServerID)
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No credentials configured.")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nAdd a credential with: goflow credential add <server-id> --key <name>")
				}
				return nil
			}

			// Parse credential keys (format: "server-id:key-name")
			type credentialEntry struct {
				serverID string
				keyName  string
			}

			entries := make([]credentialEntry, 0)
			for _, fullKey := range credentialKeys {
				parts := strings.SplitN(fullKey, ":", 2)
				if len(parts) == 2 {
					serverID := parts[0]
					keyName := parts[1]

					// Apply filter if specified
					if filterServerID != "" && serverID != filterServerID {
						continue
					}

					entries = append(entries, credentialEntry{
						serverID: serverID,
						keyName:  keyName,
					})
				}
			}

			if len(entries) == 0 {
				if filterServerID != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No credentials configured for server '%s'.\n", filterServerID)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nAdd a credential with: goflow credential add %s --key <name>\n", filterServerID)
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No credentials configured.")
				}
				return nil
			}

			// Sort entries by server ID, then key name
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].serverID != entries[j].serverID {
					return entries[i].serverID < entries[j].serverID
				}
				return entries[i].keyName < entries[j].keyName
			})

			// Display results
			if filterServerID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Credentials for '%s':\n", filterServerID)
				for _, entry := range entries {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s (set)\n", entry.keyName)
				}
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Configured Credentials:")

				// Group by server ID
				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "\nSERVER ID\tCREDENTIAL KEY\tSTATUS")
				_, _ = fmt.Fprintln(w, "─────────\t──────────────\t──────")

				for _, entry := range entries {
					_, _ = fmt.Fprintf(w, "%s\t%s\t(set)\n", entry.serverID, entry.keyName)
				}

				_ = w.Flush()
			}

			return nil
		},
	}

	return cmd
}
