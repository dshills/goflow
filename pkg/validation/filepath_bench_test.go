package validation

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// T014: Benchmark Tests
// ============================================================================

func BenchmarkPathValidator_Validate_Valid(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create test file
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Validate("test.txt")
	}
}

func BenchmarkPathValidator_Validate_Malicious(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Validate("../../etc/passwd")
	}
}

func BenchmarkPathValidator_Validate_DeepPath(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create deep directory structure
	deepPath := filepath.Join("a", "b", "c", "d", "e", "f", "g", "h", "file.txt")
	fullPath := filepath.Join(basePath, deepPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Validate(deepPath)
	}
}

func BenchmarkPathValidator_Validate_NonExistent(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Validate("nonexistent/file.txt")
	}
}

func BenchmarkPathValidator_Validate_Mixed(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	// Create test file
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		b.Fatal(err)
	}

	testPaths := []string{
		"test.txt",           // valid
		"../../etc/passwd",   // malicious
		"nonexistent.txt",    // non-existent but safe
		"/absolute/path.txt", // absolute (rejected)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Validate(testPaths[i%len(testPaths)])
	}
}

func BenchmarkValidateSecurePath(b *testing.B) {
	basePath := b.TempDir()

	// Create test file
	testFile := filepath.Join(basePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateSecurePath(basePath, "test.txt")
	}
}

func BenchmarkNewPathValidator(b *testing.B) {
	basePath := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewPathValidator(basePath)
	}
}

func BenchmarkPathValidator_Stats(b *testing.B) {
	basePath := b.TempDir()
	validator, err := NewPathValidator(basePath)
	if err != nil {
		b.Fatalf("NewPathValidator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.Stats()
	}
}
