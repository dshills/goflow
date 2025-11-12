// Package contracts defines API contracts for the 002-pr-review-remediation feature.
//
// This file documents contract changes (security enhancements) for the test server.
// Implementation will be in internal/testutil/testserver/
package contracts

import (
	"time"
)

// ServerConfig configures the test server security policies.
//
// NEW TYPE: Configuration for secure file operations
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
// NEW FUNCTION: Provides sensible security defaults
//
// Defaults:
//   - AllowedDirectory: os.TempDir()
//   - MaxFileSize: 10MB
//   - LogSecurityEvents: true
//   - LogFilePath: "" (stderr)
//   - ReadTimeout: 5 seconds
//   - WriteTimeout: 5 seconds
//
// Example:
//
//	config := DefaultConfig()
//	config.AllowedDirectory = "/var/app/data" // Override if needed
//	server := NewServer(config)
func DefaultConfig() *ServerConfig

// LoadConfig loads configuration from environment variables and config files.
//
// NEW FUNCTION: Configuration loading with precedence
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
//
// Example:
//
//	config := LoadConfig()
func LoadConfig() *ServerConfig

// Server represents the MCP test server.
//
// ENHANCED: Now validates all file operations
type Server struct {
	config    *ServerConfig
	validator *validation.PathValidator // NEW: Path validator for security
	// ... other fields ...
}

// NewServer creates a new test server.
//
// ENHANCED: Now accepts ServerConfig and initializes path validator
//
// Returns error if:
//   - config.AllowedDirectory is not absolute
//   - config.AllowedDirectory does not exist
//   - Cannot create path validator
//
// Example:
//
//	config := DefaultConfig()
//	config.AllowedDirectory = "/var/app/data"
//	server, err := NewServer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewServer(config *ServerConfig) (*Server, error)

// ENHANCED: handleReadFile now validates paths before reading
//
// BEFORE (vulnerable):
//
//	func handleReadFile(path string) (string, error) {
//	    content, err := os.ReadFile(path) // SECURITY HOLE: No validation
//	    return string(content), err
//	}
//
// AFTER (secure):
//
//	func (s *Server) handleReadFile(path string) (string, error) {
//	    validPath, err := s.validator.Validate(path)
//	    if err != nil {
//	        s.logSecurityViolation("read", path, err)
//	        return "", fmt.Errorf("invalid file path: %w", err)
//	    }
//	    content, err := os.ReadFile(validPath)
//	    if err != nil {
//	        return "", fmt.Errorf("read file: %w", err)
//	    }
//	    return string(content), nil
//	}

// ENHANCED: handleWriteFile now validates paths before writing
//
// BEFORE (vulnerable):
//
//	func handleWriteFile(path, content string) error {
//	    return os.WriteFile(path, []byte(content), 0644) // SECURITY HOLE
//	}
//
// AFTER (secure):
//
//	func (s *Server) handleWriteFile(path, content string) error {
//	    validPath, err := s.validator.Validate(path)
//	    if err != nil {
//	        s.logSecurityViolation("write", path, err)
//	        return fmt.Errorf("invalid file path: %w", err)
//	    }
//	    if len(content) > int(s.config.MaxFileSize) {
//	        return fmt.Errorf("file size exceeds limit: %d > %d",
//	            len(content), s.config.MaxFileSize)
//	    }
//	    return os.WriteFile(validPath, []byte(content), 0644)
//	}

// logSecurityViolation logs a security policy violation.
//
// NEW METHOD: Security event logging
//
// Logs include:
//   - Operation attempted (read/write)
//   - User-provided path
//   - Validation error
//   - Timestamp
//   - Client context (if available)
//
// Format: "SECURITY [testserver] Rejected {operation}: input={path} error={err}"
//
// Example log:
//
//	2025-11-12T10:15:23Z SECURITY [testserver] Rejected file read:
//	  User Input: "../../etc/passwd"
//	  Reason: "path escapes allowed directory"
func (s *Server) logSecurityViolation(operation, path string, err error)

// Start starts the test server.
//
// ENHANCED: Logs configuration at startup
//
// Startup log includes:
//   - Allowed directory
//   - Max file size
//   - Security logging status
//
// Example:
//
//	server.Start()
//	// Logs: "Test server started: allowed_dir=/tmp max_size=10MB security_log=true"
func (s *Server) Start() error

// Stop stops the test server gracefully.
func (s *Server) Stop() error
