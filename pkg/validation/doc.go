// Package validation provides security validation utilities for GoFlow.
//
// # Path Validation
//
// The primary purpose of this package is to prevent directory traversal attacks
// when processing user-provided file paths. This is critical for GoFlow's
// workflow storage and execution systems.
//
// # Security Guarantees
//
// The PathValidator provides defense-in-depth security with multiple validation layers:
//
//   - Lexical validation: Rejects absolute paths, ".." components, and Windows reserved names
//   - Path normalization: Cleans and normalizes paths to canonical form
//   - Symbolic link resolution: Resolves all symlinks to their real paths
//   - Containment verification: Ensures the final path is within the base directory
//
// These layers work together to prevent:
//
//   - Directory traversal attacks (../../etc/passwd)
//   - Absolute path injection (/etc/passwd)
//   - Symbolic link escapes (symlink pointing outside base directory)
//   - Windows reserved name exploitation (CON, PRN, AUX, etc.)
//   - Path length overflow attacks
//
// # Usage
//
// For repeated validations (recommended):
//
//	validator, err := validation.NewPathValidator("/var/app/workflows")
//	if err != nil {
//	    log.Fatalf("Failed to create validator: %v", err)
//	}
//
//	// Later, validate user input
//	safePath, err := validator.Validate(userInput)
//	if err != nil {
//	    return fmt.Errorf("invalid path: %w", err)
//	}
//
//	// Safe to use safePath for file operations
//	data, err := os.ReadFile(safePath)
//
// For one-off validations:
//
//	safePath, err := validation.ValidateSecurePath("/var/app/workflows", userInput)
//	if err != nil {
//	    return fmt.Errorf("invalid path: %w", err)
//	}
//
// # Performance
//
// The validator is optimized for production use:
//
//   - Average validation time: ~100μs
//   - P99 latency: <1ms
//   - Thread-safe for concurrent use
//   - Memory efficient (single base path resolution)
//
// # Monitoring
//
// The validator provides statistics for security monitoring:
//
//	validations, rejections := validator.Stats()
//	rejectionRate := float64(rejections) / float64(validations)
//	if rejectionRate > 0.1 {
//	    log.Warnf("High path rejection rate: %.2f%%", rejectionRate*100)
//	}
//
// # Thread Safety
//
// All types in this package are safe for concurrent use by multiple goroutines.
package validation
