// Package contracts defines API contracts for the 002-pr-review-remediation feature.
//
// This file documents the contract (public API) for the validation package.
// Implementation will be in pkg/validation/
package contracts

import (
	"sync"
	"time"
)

// PathValidator validates user-provided file paths to prevent directory traversal attacks.
//
// It implements defense-in-depth security with multiple validation layers:
//   - Lexical validation (reject absolute paths, .., reserved names)
//   - Path normalization
//   - Symbolic link resolution
//   - Containment verification
//
// Thread-safe for concurrent use.
type PathValidator struct {
	basePath     string
	resolvedBase string
	maxPathLen   int
	validations  uint64
	rejections   uint64
	mu           sync.RWMutex
}

// NewPathValidator creates a new path validator for the given base directory.
//
// The base directory must be an absolute path and must exist. All validated
// paths will be restricted to this directory and its subdirectories.
//
// Returns error if:
//   - basePath is not absolute
//   - basePath does not exist
//   - basePath is not a directory
//   - Cannot resolve symbolic links in basePath
//
// Example:
//
//	validator, err := NewPathValidator("/var/app/data")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewPathValidator(basePath string) (*PathValidator, error)

// Validate validates that userPath is safe to access within the base directory.
//
// Returns the validated absolute path if safe, or error if the path:
//   - Is empty
//   - Escapes the base directory (via .., absolute paths, or symlinks)
//   - Contains Windows reserved names (CON, PRN, etc.)
//   - Exceeds maximum path length
//   - Cannot be resolved
//
// The returned path is guaranteed to be:
//   - Absolute
//   - Within the base directory (after symlink resolution)
//   - Safe to use with os.Open, os.ReadFile, os.WriteFile, etc.
//
// Performance: ~100μs average, <1ms p99
//
// Example:
//
//	validPath, err := validator.Validate("uploads/file.txt")
//	if err != nil {
//	    return fmt.Errorf("invalid path: %w", err)
//	}
//	// Safe to use validPath
//	content, _ := os.ReadFile(validPath)
func (v *PathValidator) Validate(userPath string) (string, error)

// Stats returns validation statistics for monitoring.
//
// Returns:
//   - validations: Total number of Validate() calls
//   - rejections: Number of rejected paths (validation errors)
//
// Thread-safe.
func (v *PathValidator) Stats() (validations, rejections uint64)

// ValidationError represents a path validation failure with context for logging.
type ValidationError struct {
	UserPath     string    // Original user input that was rejected
	Reason       string    // Human-readable reason for rejection
	ResolvedPath string    // Resolved path if resolution succeeded (may be empty)
	Timestamp    time.Time // When the validation error occurred
}

// Error implements the error interface.
//
// Format: "path validation failed: {Reason} (input: {UserPath})"
func (e *ValidationError) Error() string

// ValidateSecurePath is a convenience function that validates a path without
// creating a PathValidator instance.
//
// Equivalent to:
//
//	v, _ := NewPathValidator(basePath)
//	return v.Validate(userPath)
//
// Use this for one-off validations. For repeated validations, create a
// PathValidator instance to avoid re-resolving the base path.
//
// See PathValidator.Validate for parameter and return value documentation.
func ValidateSecurePath(basePath, userPath string) (string, error)
