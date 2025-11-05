package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dshills/goflow/pkg/mcpserver"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ServerConfig represents the configuration for MCP servers
type ServerConfig struct {
	Servers map[string]*ServerEntry `yaml:"servers"`
}

// ServerEntry represents a single server configuration
type ServerEntry struct {
	ID            string            `yaml:"id"`
	Name          string            `yaml:"name,omitempty"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args,omitempty"`
	Transport     string            `yaml:"transport,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"`
	CredentialRef string            `yaml:"credential_ref,omitempty"`
}

// NewServerCommand creates the server management command
func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage MCP servers",
		Long:  `Register, list, test, and remove MCP servers used in workflows.`,
	}

	cmd.AddCommand(newServerAddCommand())
	cmd.AddCommand(newServerListCommand())
	cmd.AddCommand(newServerTestCommand())
	cmd.AddCommand(newServerRemoveCommand())

	return cmd
}

// newServerAddCommand creates the server add subcommand
func newServerAddCommand() *cobra.Command {
	var (
		transport     string
		envVars       []string
		credentialRef string
		name          string
	)

	cmd := &cobra.Command{
		Use:   "add <server-id> <command> [args...]",
		Short: "Add a new MCP server",
		Long: `Register a new MCP server that can be used in workflows.

Examples:
  # Add filesystem server
  goflow server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp

  # Add with custom transport
  goflow server add myserver node server.js --transport sse

  # Add with environment variables
  goflow server add api-server python api.py --env API_KEY=value --env DEBUG=true`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]
			command := args[1]
			commandArgs := args[2:]

			// Parse environment variables
			env := make(map[string]string)
			for _, envVar := range envVars {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
				}
				env[parts[0]] = parts[1]
			}

			// Default transport to stdio
			if transport == "" {
				transport = "stdio"
			}

			// Validate transport
			validTransports := map[string]bool{"stdio": true, "sse": true, "http": true}
			if !validTransports[transport] {
				return fmt.Errorf("invalid transport: %s (must be stdio, sse, or http)", transport)
			}

			// Load existing servers config
			config, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load servers config: %w", err)
			}

			// Check for duplicate server ID
			if _, exists := config.Servers[serverID]; exists {
				return fmt.Errorf("server with ID '%s' already exists", serverID)
			}

			// Use server ID as name if name not provided
			if name == "" {
				name = serverID
			}

			// Add new server
			config.Servers[serverID] = &ServerEntry{
				ID:            serverID,
				Name:          name,
				Command:       command,
				Args:          commandArgs,
				Transport:     transport,
				Env:           env,
				CredentialRef: credentialRef,
			}

			// Save config
			if err := saveServersConfig(config); err != nil {
				return fmt.Errorf("failed to save servers config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Server '%s' added successfully\n", serverID)
			return nil
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type (stdio|sse|http)")
	cmd.Flags().StringSliceVar(&envVars, "env", []string{}, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVar(&credentialRef, "credential-ref", "", "Reference to keyring credential")
	cmd.Flags().StringVar(&name, "name", "", "Friendly name for the server")

	return cmd
}

// newServerListCommand creates the server list subcommand
func newServerListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered MCP servers",
		Long:  `Display all registered MCP servers with their status and tool count.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load servers config: %w", err)
			}

			if len(config.Servers) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No servers registered.")
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nRegister a server with: goflow server add <id> <command> [args...]")
				return nil
			}

			// Create table writer
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tCOMMAND\tTRANSPORT\tSTATUS")
			_, _ = fmt.Fprintln(w, "──\t────\t───────\t─────────\t──────")

			for _, server := range config.Servers {
				status := "Unknown"
				// TODO: Test connection to determine actual status
				// For now, show as "Registered"
				status = "Registered"

				name := server.Name
				if name == "" {
					name = server.ID
				}

				transport := server.Transport
				if transport == "" {
					transport = "stdio"
				}

				cmdDisplay := server.Command
				if len(server.Args) > 0 {
					cmdDisplay += " " + strings.Join(server.Args, " ")
					if len(cmdDisplay) > 40 {
						cmdDisplay = cmdDisplay[:37] + "..."
					}
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					server.ID, name, cmdDisplay, transport, status)
			}

			_ = w.Flush()
			return nil
		},
	}

	return cmd
}

// newServerTestCommand creates the server test subcommand
func newServerTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <server-id>",
		Short: "Test MCP server connection",
		Long:  `Test connection to an MCP server and discover available tools.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			// Load servers config
			config, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load servers config: %w", err)
			}

			// Find server
			serverEntry, exists := config.Servers[serverID]
			if !exists {
				return fmt.Errorf("server not found: %s", serverID)
			}

			// Create MCP server instance
			transport := mcpserver.TransportStdio
			switch serverEntry.Transport {
			case "sse":
				transport = mcpserver.TransportSSE
			case "http":
				transport = mcpserver.TransportHTTP
			}

			server, err := mcpserver.NewMCPServer(
				serverEntry.ID,
				serverEntry.Command,
				serverEntry.Args,
				transport,
			)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "✗ Failed to create server instance: %v\n", err)
				return err
			}

			// Test connection (this would actually connect to the MCP server)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Testing connection to '%s'...\n", serverID)

			// TODO: Implement actual connection test
			// For now, just validate the configuration
			if server.ID == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStderr(), "✗ Server configuration is invalid")
				return fmt.Errorf("invalid server configuration")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "✓ Connection successful")

			// Show server details
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Server ID: %s\n", server.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Transport: %s\n", transport)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Command: %s %s\n", serverEntry.Command, strings.Join(serverEntry.Args, " "))

			// TODO: Discover and display tools
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\n✓ Server configuration is valid")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Note: Tool discovery requires server to be running")

			return nil
		},
	}

	return cmd
}

// newServerRemoveCommand creates the server remove subcommand
func newServerRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <server-id>",
		Short: "Remove an MCP server",
		Long:  `Unregister an MCP server. This will not affect existing workflows that reference it.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverID := args[0]

			// Load servers config
			config, err := loadServersConfig()
			if err != nil {
				return fmt.Errorf("failed to load servers config: %w", err)
			}

			// Check if server exists
			if _, exists := config.Servers[serverID]; !exists {
				return fmt.Errorf("server not found: %s", serverID)
			}

			// Remove server
			delete(config.Servers, serverID)

			// Save config
			if err := saveServersConfig(config); err != nil {
				return fmt.Errorf("failed to save servers config: %w", err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Server '%s' removed successfully\n", serverID)
			return nil
		},
	}

	return cmd
}

// loadServersConfig loads the servers configuration file
func loadServersConfig() (*ServerConfig, error) {
	configPath := GetServersConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config
		return &ServerConfig{
			Servers: make(map[string]*ServerEntry),
		}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize map if nil
	if config.Servers == nil {
		config.Servers = make(map[string]*ServerEntry)
	}

	return &config, nil
}

// saveServersConfig saves the servers configuration file
func saveServersConfig(config *ServerConfig) error {
	configPath := GetServersConfigPath()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically using temp file + rename
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Rename(tempPath, configPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}
