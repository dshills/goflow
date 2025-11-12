// Package testserver provides an MCP test server for development and testing.
package testserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ServerConfig configures the test server security policies.
//
// This configuration enables secure file operations with whitelist-based access.
type ServerConfig struct {
	// File operation security
	AllowedDirectory string // Base directory for file operations (must be absolute)
	MaxFileSize      int64  // Maximum file size for read/write in bytes

	// Logging
	LogSecurityEvents bool   // Whether to log security violations
	LogFilePath       string // Path to security audit log (empty = stderr)

	// Performance
	ReadTimeout  time.Duration // Timeout for file read operations
	WriteTimeout time.Duration // Timeout for file write operations
}

// DefaultConfig returns a secure default configuration.
//
// Defaults:
//   - AllowedDirectory: os.TempDir()
//   - MaxFileSize: 10MB
//   - LogSecurityEvents: true
//   - LogFilePath: "" (stderr)
//   - ReadTimeout: 5 seconds
//   - WriteTimeout: 5 seconds
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		AllowedDirectory:  os.TempDir(),
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		LogSecurityEvents: true,
		LogFilePath:       "",
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
}

// LoadConfig loads configuration from environment variables and config files.
//
// Precedence (highest to lowest):
//  1. Environment variables (GOFLOW_TESTSERVER_*)
//  2. Config file (.goflow/testserver.yaml if exists)
//  3. Defaults (DefaultConfig())
//
// Environment variables:
//   - GOFLOW_TESTSERVER_ALLOWED_DIR: Override allowed directory
//   - GOFLOW_TESTSERVER_MAX_FILE_SIZE: Override max file size (bytes)
//   - GOFLOW_TESTSERVER_LOG_SECURITY: Override security logging (true/false)
func LoadConfig() *ServerConfig {
	config := DefaultConfig()

	// Override with environment variables if present
	if allowedDir := os.Getenv("GOFLOW_TESTSERVER_ALLOWED_DIR"); allowedDir != "" {
		config.AllowedDirectory = allowedDir
	}

	if maxSizeStr := os.Getenv("GOFLOW_TESTSERVER_MAX_FILE_SIZE"); maxSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil && maxSize > 0 {
			config.MaxFileSize = maxSize
		}
		// If parsing fails or value is negative, keep the default
	}

	if logSecurityStr := os.Getenv("GOFLOW_TESTSERVER_LOG_SECURITY"); logSecurityStr != "" {
		// Parse boolean (case-insensitive)
		logSecurityStr = strings.ToLower(strings.TrimSpace(logSecurityStr))
		switch logSecurityStr {
		case "true", "1", "yes":
			config.LogSecurityEvents = true
		case "false", "0", "no":
			config.LogSecurityEvents = false
		}
		// If parsing fails, keep the default
	}

	return config
}

// Validate checks if the configuration is valid.
//
// Returns error if:
//   - AllowedDirectory is not an absolute path
//   - AllowedDirectory does not exist
//   - AllowedDirectory is not a directory
//   - MaxFileSize is not positive
func (c *ServerConfig) Validate() error {
	// Validate AllowedDirectory is not empty
	if c.AllowedDirectory == "" {
		return fmt.Errorf("allowed directory cannot be empty")
	}

	// Validate AllowedDirectory is absolute
	if !filepath.IsAbs(c.AllowedDirectory) {
		return fmt.Errorf("allowed directory must be an absolute path: %s", c.AllowedDirectory)
	}

	// Validate AllowedDirectory exists
	info, err := os.Stat(c.AllowedDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("allowed directory does not exist: %s", c.AllowedDirectory)
		}
		return fmt.Errorf("cannot access allowed directory: %w", err)
	}

	// Validate AllowedDirectory is a directory
	if !info.IsDir() {
		return fmt.Errorf("allowed directory is not a directory: %s", c.AllowedDirectory)
	}

	// Validate MaxFileSize is positive
	if c.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive, got %d", c.MaxFileSize)
	}

	return nil
}
