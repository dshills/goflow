package cli

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	// Version is the current version of GoFlow
	Version = "1.0.0"
)

// Config holds the global configuration for GoFlow CLI
type Config struct {
	ConfigDir string
	Debug     bool
}

// GlobalConfig is the shared configuration instance
var GlobalConfig = &Config{}

// NewRootCommand creates the root cobra command for GoFlow
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goflow",
		Short: "GoFlow - Workflow orchestration for MCP servers",
		Long: `GoFlow is a visual workflow orchestration system for Model Context Protocol (MCP) servers.
It enables developers to chain multiple MCP tools into sophisticated, reusable workflows
with conditional logic, data transformation, and parallel execution.`,
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize configuration: %w", err)
			}

			// Setup logging
			if GlobalConfig.Debug {
				log.SetOutput(os.Stderr)
				log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
			} else {
				log.SetOutput(io.Discard)
			}

			return nil
		},
	}

	// Persistent flags (available to all subcommands)
	cmd.PersistentFlags().BoolVar(&GlobalConfig.Debug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().StringVar(&GlobalConfig.ConfigDir, "config-dir", "", "Configuration directory (default: ~/.goflow)")

	// Add subcommands
	cmd.AddCommand(NewServerCommand())
	cmd.AddCommand(NewCredentialCommand())
	cmd.AddCommand(NewValidateCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewEditCommand())
	cmd.AddCommand(NewExecutionsCommand())
	cmd.AddCommand(NewExecutionCommand())
	cmd.AddCommand(NewLogsCommand())
	cmd.AddCommand(NewExportCommand())
	cmd.AddCommand(NewImportCommand())

	return cmd
}

// initConfig initializes the GoFlow configuration directory and files
func initConfig() error {
	// Determine config directory
	// Environment variable always takes priority (for testing)
	if envDir := os.Getenv("GOFLOW_CONFIG_DIR"); envDir != "" {
		GlobalConfig.ConfigDir = envDir
	} else if GlobalConfig.ConfigDir == "" {
		// Use default ~/.goflow
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		GlobalConfig.ConfigDir = filepath.Join(homeDir, ".goflow")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(GlobalConfig.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{"workflows", "executions"}
	for _, dir := range dirs {
		dirPath := filepath.Join(GlobalConfig.ConfigDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Load or create config file
	configFile := filepath.Join(GlobalConfig.ConfigDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config
		defaultConfig := map[string]interface{}{
			"version": "1.0",
		}
		data, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %w", err)
		}
		if err := os.WriteFile(configFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}
	}

	return nil
}

// GetConfigDir returns the configuration directory path
// Priority order: 1) GOFLOW_CONFIG_DIR env var (for testing), 2) GlobalConfig.ConfigDir, 3) ~/.goflow
func GetConfigDir() string {
	// Check environment variable first (for testing)
	if envDir := os.Getenv("GOFLOW_CONFIG_DIR"); envDir != "" {
		return envDir
	}
	if GlobalConfig.ConfigDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if home dir cannot be determined
			return ".goflow"
		}
		return filepath.Join(homeDir, ".goflow")
	}
	return GlobalConfig.ConfigDir
}

// GetWorkflowsDir returns the workflows directory path
func GetWorkflowsDir() string {
	return filepath.Join(GetConfigDir(), "workflows")
}

// GetServersConfigPath returns the path to the servers configuration file
func GetServersConfigPath() string {
	return filepath.Join(GetConfigDir(), "servers.yaml")
}

// Execute runs the root command
func Execute() error {
	return NewRootCommand().Execute()
}
