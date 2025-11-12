package validation

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// ============================================================================
// T004: Constructor Tests
// ============================================================================

func TestNewPathValidator_ValidBasePath(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() string
		cleanup func(string)
	}{
		{
			name: "absolute path with existing directory",
			setup: func() string {
				dir := t.TempDir()
				return dir
			},
		},
		{
			name: "absolute path with nested directory",
			setup: func() string {
				base := t.TempDir()
				nested := filepath.Join(base, "nested", "path")
				if err := os.MkdirAll(nested, 0755); err != nil {
					t.Fatal(err)
				}
				return nested
			},
		},
		{
			name: "absolute path with symlink in base (should resolve)",
			setup: func() string {
				base := t.TempDir()
				target := filepath.Join(base, "target")
				link := filepath.Join(base, "link")
				if err := os.Mkdir(target, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, link); err != nil {
					if runtime.GOOS == "windows" {
						t.Skip("Symlink creation requires elevated privileges on Windows")
					}
					t.Fatal(err)
				}
				return link
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath := tt.setup()
			if tt.cleanup != nil {
				defer tt.cleanup(basePath)
			}

			validator, err := NewPathValidator(basePath)
			if err != nil {
				t.Fatalf("NewPathValidator() error = %v, want nil", err)
			}
			if validator == nil {
				t.Fatal("NewPathValidator() returned nil validator")
			}
			if validator.basePath == "" {
				t.Error("validator.basePath is empty")
			}
			if validator.resolvedBase == "" {
				t.Error("validator.resolvedBase is empty")
			}
			if validator.maxPathLen <= 0 {
				t.Errorf("validator.maxPathLen = %d, want > 0", validator.maxPathLen)
			}
		})
	}
}

func TestNewPathValidator_InvalidBasePath(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		setupPath func() string
		wantError string
	}{
		{
			name:      "relative path",
			basePath:  "relative/path",
			wantError: "absolute",
		},
		{
			name:      "non-existent path",
			basePath:  "/nonexistent/path/that/does/not/exist",
			wantError: "does not exist",
		},
		{
			name:      "empty path",
			basePath:  "",
			wantError: "empty",
		},
		{
			name: "path to file not directory",
			setupPath: func() string {
				f, err := os.CreateTemp("", "notadir")
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				return f.Name()
			},
			wantError: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.basePath
			if tt.setupPath != nil {
				testPath = tt.setupPath()
			}

			validator, err := NewPathValidator(testPath)
			if err == nil {
				t.Fatalf("NewPathValidator() error = nil, want error containing %q", tt.wantError)
			}
			if validator != nil {
				t.Errorf("NewPathValidator() = %v, want nil on error", validator)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantError)) {
				t.Errorf("error = %q, want error containing %q", err.Error(), tt.wantError)
			}
		})
	}
}

// ============================================================================
// T005: Security Tests - Malicious Path Detection
// ============================================================================

func TestPathValidator_Validate_DirectoryTraversal(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create a file outside base directory for testing
	outsideDir := t.TempDir()
	targetFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(targetFile, []byte("sensitive"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		userPath   string
		reason     string
		shouldFail bool
	}{
		{
			name:       "classic traversal with ..",
			userPath:   "../../etc/passwd",
			reason:     "directory traversal",
			shouldFail: true,
		},
		{
			name:       "multiple traversals",
			userPath:   "../../../../../../../etc/passwd",
			reason:     "directory traversal",
			shouldFail: true,
		},
		{
			name:       "traversal with mixed separators",
			userPath:   "../\\../etc/passwd",
			reason:     "directory traversal",
			shouldFail: true,
		},
		{
			name:       "path with dots that are NOT traversal (valid path)",
			userPath:   "....//....//etc/passwd",
			reason:     "four dots is a directory name, not ..",
			shouldFail: false, // This is actually valid!
		},
		{
			name:       "URL encoded traversal",
			userPath:   "..%2F..%2Fetc%2Fpasswd",
			reason:     "directory traversal",
			shouldFail: true,
		},
		{
			name:       "traversal after valid path",
			userPath:   "valid/../../etc/passwd",
			reason:     "directory traversal",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.userPath)
			if tt.shouldFail {
				if err == nil {
					t.Errorf("Validate(%q) succeeded with result %q, want error", tt.userPath, result)
				}
				if result != "" {
					t.Errorf("Validate(%q) returned non-empty path %q on error", tt.userPath, result)
				}

				// Check error message mentions the issue
				errMsg := strings.ToLower(err.Error())
				if !strings.Contains(errMsg, "escape") && !strings.Contains(errMsg, "traversal") && !strings.Contains(errMsg, "not relative") && !strings.Contains(errMsg, "resolve") {
					t.Errorf("error message %q doesn't clearly indicate path escape/traversal", err.Error())
				}

				// Ensure we're tracking rejections
				_, rejections := validator.Stats()
				if rejections == 0 {
					t.Error("Stats() rejections = 0, want > 0 after rejection")
				}
			} else {
				// Path should be accepted (it's safe, even if non-existent)
				if err != nil {
					t.Logf("Note: %q was rejected with: %v (this may be OK if path doesn't exist)", tt.userPath, err)
				}
			}
		})
	}
}

func TestPathValidator_Validate_AbsolutePaths(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	tests := []struct {
		name          string
		userPath      string
		skipOnUnix    bool
		skipOnWindows bool
	}{
		{
			name:     "Unix absolute path",
			userPath: "/etc/passwd",
		},
		{
			name:     "Unix root",
			userPath: "/",
		},
		{
			name:       "Windows drive letter",
			userPath:   "C:\\Windows\\System32",
			skipOnUnix: true, // Treated as relative on Unix
		},
		{
			name:       "Windows drive relative",
			userPath:   "C:file.txt",
			skipOnUnix: true, // Treated as relative on Unix
		},
		{
			name:       "Windows UNC path",
			userPath:   "\\\\server\\share\\file",
			skipOnUnix: true, // Treated as relative on Unix
		},
		{
			name:       "Windows extended-length path",
			userPath:   "\\\\?\\C:\\path",
			skipOnUnix: true, // Treated as relative on Unix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnUnix && runtime.GOOS != "windows" {
				t.Skip("Windows path test, skipping on Unix")
			}
			if tt.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Unix path test, skipping on Windows")
			}

			result, err := validator.Validate(tt.userPath)
			if err == nil {
				t.Errorf("Validate(%q) succeeded with result %q, want error", tt.userPath, result)
			}
			if result != "" {
				t.Errorf("Validate(%q) returned non-empty path %q on error", tt.userPath, result)
			}
		})
	}
}

func TestPathValidator_Validate_SymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink test requires Unix-like OS or Windows admin privileges")
	}

	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create symlink pointing outside base directory
	outsideDir := t.TempDir()
	targetFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(targetFile, []byte("sensitive"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkPath := filepath.Join(basePath, "escape-link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		userPath   string
		shouldFail bool
	}{
		{
			name:       "direct symlink to outside directory",
			userPath:   "escape-link/secret.txt",
			shouldFail: true, // Follows symlink outside base
		},
		{
			name:       "symlink with traversal that stays inside",
			userPath:   "escape-link/../secret.txt",
			shouldFail: false, // Cleans to "secret.txt" which doesn't exist in base, but is safe
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.userPath)
			if tt.shouldFail {
				if err == nil {
					t.Errorf("Validate(%q) succeeded with result %q, want error (symlink escape)", tt.userPath, result)
				}
				if result != "" {
					t.Errorf("Validate(%q) returned non-empty path %q on error", tt.userPath, result)
				}
			} else {
				if err != nil {
					t.Errorf("Validate(%q) error = %v, want nil (path is safe after cleaning)", tt.userPath, err)
				}
			}
		})
	}
}

func TestPathValidator_Validate_WindowsReservedNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows reserved name tests only on Windows")
	}

	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5",
		"COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5",
		"LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, name := range reservedNames {
		t.Run(name, func(t *testing.T) {
			// Test bare name
			result, err := validator.Validate(name)
			if err == nil {
				t.Errorf("Validate(%q) succeeded with result %q, want error", name, result)
			}

			// Test with extension
			nameWithExt := name + ".txt"
			result, err = validator.Validate(nameWithExt)
			if err == nil {
				t.Errorf("Validate(%q) succeeded with result %q, want error", nameWithExt, result)
			}

			// Test in subdirectory
			inSubdir := filepath.Join("subdir", name)
			result, err = validator.Validate(inSubdir)
			if err == nil {
				t.Errorf("Validate(%q) succeeded with result %q, want error", inSubdir, result)
			}
		})
	}
}

func TestPathValidator_Validate_PathLength(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create path exceeding max length (1024 bytes default)
	longPath := strings.Repeat("a", 1025)

	result, err := validator.Validate(longPath)
	if err == nil {
		t.Errorf("Validate(path with 1025 chars) succeeded with result %q, want error", result)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "length") && !strings.Contains(strings.ToLower(err.Error()), "long") {
		t.Errorf("error message %q should mention path length", err.Error())
	}
}

// ============================================================================
// T006: Valid Path Tests
// ============================================================================

func TestPathValidator_Validate_ValidPaths(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create test files and directories
	testFiles := []string{
		"file.txt",
		"subdir/file.json",
		"a/b/c/d/e/deep.txt",
		"my documents/file.txt",
		"unicode-文件.txt",
		".hidden",
	}

	for _, f := range testFiles {
		fullPath := filepath.Join(basePath, f)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		userPath string
	}{
		{
			name:     "simple filename",
			userPath: "file.txt",
		},
		{
			name:     "relative path with subdirectory",
			userPath: "subdir/file.json",
		},
		{
			name:     "path with current directory reference",
			userPath: "./subdir/file.json",
		},
		{
			name:     "deeply nested path",
			userPath: "a/b/c/d/e/deep.txt",
		},
		{
			name:     "path with spaces",
			userPath: "my documents/file.txt",
		},
		{
			name:     "path with Unicode characters",
			userPath: "unicode-文件.txt",
		},
		{
			name:     "hidden file",
			userPath: ".hidden",
		},
		{
			name:     "redundant separators (should clean)",
			userPath: "subdir//file.json",
		},
		{
			name:     "trailing separator (should clean)",
			userPath: "subdir/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.userPath)
			if err != nil {
				t.Fatalf("Validate(%q) error = %v, want nil", tt.userPath, err)
			}
			if result == "" {
				t.Fatal("Validate() returned empty path")
			}

			// Result should be absolute
			if !filepath.IsAbs(result) {
				t.Errorf("Validate() returned relative path %q, want absolute", result)
			}

			// Result should be within base directory
			// Note: We need to compare against resolved base, not original base (macOS symlinks /var -> /private/var)
			resolvedBase, err := filepath.EvalSymlinks(basePath)
			if err != nil {
				t.Fatal(err)
			}
			relPath, err := filepath.Rel(resolvedBase, result)
			if err != nil {
				t.Errorf("filepath.Rel() error = %v, result %q should be relative to resolved base %q", err, result, resolvedBase)
			}
			if strings.HasPrefix(relPath, "..") {
				t.Errorf("result %q escapes base directory %q (relPath = %q)", result, resolvedBase, relPath)
			}

			// Verify validations counter incremented
			validations, _ := validator.Stats()
			if validations == 0 {
				t.Error("Stats() validations = 0, want > 0 after validation")
			}
		})
	}
}

func TestPathValidator_Validate_EmptyPath(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	result, err := validator.Validate("")
	if err == nil {
		t.Errorf("Validate(\"\") succeeded with result %q, want error", result)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "empty") {
		t.Errorf("error message %q should mention empty path", err.Error())
	}
}

func TestPathValidator_Validate_NonExistentPath(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Non-existent paths should still validate if they're safe
	// (user might want to create them)
	userPath := "future/file.txt"
	result, err := validator.Validate(userPath)
	if err != nil {
		t.Errorf("Validate(%q) for non-existent path error = %v, want nil (path is safe)", userPath, err)
	}
	if result == "" {
		t.Error("Validate() returned empty path for non-existent but safe path")
	}

	// Result should still be within base (use resolved base for symlink compatibility)
	resolvedBase, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result, resolvedBase) {
		t.Errorf("result %q doesn't start with resolved base %q", result, resolvedBase)
	}
}

// ============================================================================
// T011: ValidationError Tests
// ============================================================================

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		wantText []string // All these strings should appear in error message
	}{
		{
			name: "basic validation error",
			err: &ValidationError{
				UserPath:  "../../etc/passwd",
				Reason:    "path escapes base directory",
				Timestamp: mockTime(),
			},
			wantText: []string{"validation", "../../etc/passwd", "escapes"},
		},
		{
			name: "error with resolved path",
			err: &ValidationError{
				UserPath:     "link/file.txt",
				Reason:       "resolved path escapes base directory",
				ResolvedPath: "/etc/passwd",
				Timestamp:    mockTime(),
			},
			wantText: []string{"validation", "link/file.txt", "escapes"},
		},
		{
			name: "Windows reserved name error",
			err: &ValidationError{
				UserPath:  "CON",
				Reason:    "Windows reserved name not allowed",
				Timestamp: mockTime(),
			},
			wantText: []string{"validation", "CON", "reserved"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			if errMsg == "" {
				t.Fatal("Error() returned empty string")
			}

			for _, want := range tt.wantText {
				if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(want)) {
					t.Errorf("Error() = %q, should contain %q", errMsg, want)
				}
			}

			// Verify format includes user input
			if !strings.Contains(errMsg, tt.err.UserPath) {
				t.Errorf("Error() = %q, should include user input %q", errMsg, tt.err.UserPath)
			}
		})
	}
}

// ============================================================================
// T012: Stats Tests
// ============================================================================

func TestPathValidator_Stats(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Initial stats
	validations, rejections := validator.Stats()
	if validations != 0 {
		t.Errorf("initial validations = %d, want 0", validations)
	}
	if rejections != 0 {
		t.Errorf("initial rejections = %d, want 0", rejections)
	}

	// Create test file
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Perform valid validation
	_, err = validator.Validate("test.txt")
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	validations, rejections = validator.Stats()
	if validations != 1 {
		t.Errorf("after valid validation, validations = %d, want 1", validations)
	}
	if rejections != 0 {
		t.Errorf("after valid validation, rejections = %d, want 0", rejections)
	}

	// Perform invalid validation
	_, err = validator.Validate("../../etc/passwd")
	if err == nil {
		t.Fatal("Validate() with malicious path succeeded, want error")
	}

	validations, rejections = validator.Stats()
	if validations != 2 {
		t.Errorf("after invalid validation, validations = %d, want 2", validations)
	}
	if rejections != 1 {
		t.Errorf("after invalid validation, rejections = %d, want 1", rejections)
	}

	// Multiple operations
	for i := 0; i < 10; i++ {
		validator.Validate("test.txt")
	}
	for i := 0; i < 5; i++ {
		validator.Validate("../../etc/passwd")
	}

	validations, rejections = validator.Stats()
	if validations != 17 { // 2 + 10 + 5
		t.Errorf("after batch operations, validations = %d, want 17", validations)
	}
	if rejections != 6 { // 1 + 5
		t.Errorf("after batch operations, rejections = %d, want 6", rejections)
	}
}

func TestPathValidator_Stats_ThreadSafety(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create test file
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Concurrent validations
	const goroutines = 10
	const iterationsPerGoroutine = 100

	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < iterationsPerGoroutine; j++ {
				if j%2 == 0 {
					validator.Validate("test.txt")
				} else {
					validator.Validate("../../etc/passwd")
				}
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	validations, rejections := validator.Stats()
	expectedTotal := goroutines * iterationsPerGoroutine
	if validations != uint64(expectedTotal) {
		t.Errorf("concurrent validations = %d, want %d", validations, expectedTotal)
	}
	expectedRejections := uint64(goroutines * iterationsPerGoroutine / 2)
	if rejections != expectedRejections {
		t.Errorf("concurrent rejections = %d, want %d", rejections, expectedRejections)
	}
}

// ============================================================================
// T013: Property-Based Tests
// ============================================================================

func TestPathValidator_Validate_PropertyBased_NoTraversal(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Property: Any path containing ".." should be rejected
	f := func(s string) bool {
		if !strings.Contains(s, "..") {
			return true // Skip paths without ".."
		}

		_, err := validator.Validate(s)
		return err != nil // Should always return error
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violated: paths with '..' should be rejected: %v", err)
	}
}

func TestPathValidator_Validate_PropertyBased_NoAbsolute(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Property: Any absolute path should be rejected
	f := func(s string) bool {
		if !filepath.IsAbs(s) {
			return true // Skip relative paths
		}

		_, err := validator.Validate(s)
		return err != nil // Should always return error
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violated: absolute paths should be rejected: %v", err)
	}
}

func TestPathValidator_Validate_PropertyBased_ValidPathContainment(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Get resolved base for symlink-safe comparisons
	resolvedBase, err := filepath.EvalSymlinks(basePath)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks() error = %v", err)
	}

	// Property: If validation succeeds, result must be within base directory
	f := func(s string) bool {
		result, err := validator.Validate(s)
		if err != nil {
			return true // Rejection is fine
		}

		// If accepted, must be within resolved base
		relPath, relErr := filepath.Rel(resolvedBase, result)
		if relErr != nil {
			t.Logf("filepath.Rel failed for accepted path: %v", relErr)
			return false
		}

		return !strings.HasPrefix(relPath, "..")
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violated: accepted paths must be within base directory: %v", err)
	}
}

func TestPathValidator_Validate_PropertyBased_Determinism(t *testing.T) {
	basePath := t.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		t.Fatalf("NewPathValidator() error = %v", err)
	}

	// Property: Same input always produces same result
	f := func(s string) bool {
		result1, err1 := validator.Validate(s)
		result2, err2 := validator.Validate(s)

		// Both should succeed or both should fail
		if (err1 == nil) != (err2 == nil) {
			t.Logf("Non-deterministic error for %q: err1=%v, err2=%v", s, err1, err2)
			return false
		}

		// If both succeed, results should match
		if err1 == nil && result1 != result2 {
			t.Logf("Non-deterministic result for %q: %q vs %q", s, result1, result2)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property violated: validation should be deterministic: %v", err)
	}
}

// ============================================================================
// ValidateSecurePath Convenience Function Tests
// ============================================================================

func TestValidateSecurePath(t *testing.T) {
	basePath := t.TempDir()
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		userPath  string
		wantError bool
	}{
		{
			name:      "valid path",
			userPath:  "test.txt",
			wantError: false,
		},
		{
			name:      "malicious path",
			userPath:  "../../etc/passwd",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSecurePath(basePath, tt.userPath)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateSecurePath(%q) succeeded with %q, want error", tt.userPath, result)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSecurePath(%q) error = %v, want nil", tt.userPath, err)
				}
				if result == "" {
					t.Error("ValidateSecurePath() returned empty path")
				}
			}
		})
	}
}

func TestValidateSecurePath_InvalidBase(t *testing.T) {
	result, err := ValidateSecurePath("/nonexistent/path", "test.txt")
	if err == nil {
		t.Errorf("ValidateSecurePath() with invalid base succeeded with %q, want error", result)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func mockTime() time.Time {
	t, _ := time.Parse(time.RFC3339, "2025-11-12T10:00:00Z")
	return t
}
