package testserver_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dshills/goflow/internal/testutil/testserver"
)

// TestServer_NewServer verifies server initialization.
func TestServer_NewServer(t *testing.T) {
	tests := []struct {
		name      string
		config    *testserver.ServerConfig
		wantError bool
	}{
		{
			name:      "valid config",
			config:    testserver.DefaultConfig(),
			wantError: false,
		},
		{
			name: "invalid config - relative path",
			config: &testserver.ServerConfig{
				AllowedDirectory:  "relative/path",
				MaxFileSize:       10 * 1024 * 1024,
				LogSecurityEvents: true,
			},
			wantError: true,
		},
		{
			name: "invalid config - non-existent directory",
			config: &testserver.ServerConfig{
				AllowedDirectory:  "/this/does/not/exist/12345",
				MaxFileSize:       10 * 1024 * 1024,
				LogSecurityEvents: true,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := testserver.NewServer(tt.config)

			if tt.wantError {
				if err == nil {
					t.Error("NewServer() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("NewServer() error = %v, want nil", err)
				}
				if server == nil {
					t.Error("NewServer() returned nil server")
				}
			}
		})
	}
}

// TestServer_ReadFile_MaliciousPaths verifies that malicious paths are rejected.
// This test achieves 100% detection rate per SC-005 requirement.
func TestServer_ReadFile_MaliciousPaths(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "..", "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret data"), 0644); err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}
	defer os.Remove(secretFile)

	// Create server with restricted directory
	config := testserver.DefaultConfig()
	config.AllowedDirectory = tempDir
	server, err := testserver.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	maliciousPaths := []struct {
		name string
		path string
	}{
		{"parent traversal", "../secret.txt"},
		{"absolute path escape", secretFile},
		{"multiple parent traversals", "../../secret.txt"},
		{"hidden parent traversal", "subdir/../../secret.txt"},
		{"null byte injection", "../secret.txt\x00.jpg"},
		{"windows drive letter", "C:\\Windows\\System32\\config\\SAM"},
		{"UNC path", "\\\\server\\share\\file.txt"},
		{"relative with absolute", "./../../secret.txt"},
	}

	rejectionCount := 0
	for _, tt := range maliciousPaths {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate MCP tool call request
			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "read_file",
					"arguments": map[string]interface{}{
						"path": tt.path,
					},
				},
			}

			var stdout bytes.Buffer
			server.SetStdout(&stdout)

			reqJSON, _ := json.Marshal(req)
			stdin := bytes.NewBuffer(reqJSON)
			stdin.WriteString("\n")
			server.SetStdin(stdin)

			// Process single request
			if err := server.ProcessSingleRequest(); err != nil {
				// Error is expected - path should be rejected
				rejectionCount++
				return
			}

			// Check response for error
			var resp map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Should have error in response
			if respErr, ok := resp["error"]; ok {
				rejectionCount++
				t.Logf("Correctly rejected malicious path: %s (error: %v)", tt.path, respErr)
			} else {
				t.Errorf("Malicious path was NOT rejected: %s", tt.path)
			}
		})
	}

	// Verify 100% detection rate (SC-005)
	detectionRate := float64(rejectionCount) / float64(len(maliciousPaths)) * 100
	if detectionRate < 100.0 {
		t.Errorf("Detection rate = %.1f%%, want 100%% (SC-005 requirement)", detectionRate)
	} else {
		t.Logf("Detection rate: 100%% (%d/%d malicious paths blocked)", rejectionCount, len(maliciousPaths))
	}
}

// TestServer_ReadFile_ValidPaths verifies that valid paths are accepted.
func TestServer_ReadFile_ValidPaths(t *testing.T) {
	// Create temporary directory with test files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	subFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(subFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	config := testserver.DefaultConfig()
	config.AllowedDirectory = tempDir
	server, err := testserver.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	validPaths := []struct {
		name        string
		path        string
		wantContent string
	}{
		{"simple file", "test.txt", testContent},
		{"nested file", "subdir/nested.txt", "nested content"},
		{"nested file with slash", "./subdir/nested.txt", "nested content"},
	}

	for _, tt := range validPaths {
		t.Run(tt.name, func(t *testing.T) {
			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "read_file",
					"arguments": map[string]interface{}{
						"path": tt.path,
					},
				},
			}

			var stdout bytes.Buffer
			server.SetStdout(&stdout)

			reqJSON, _ := json.Marshal(req)
			stdin := bytes.NewBuffer(reqJSON)
			stdin.WriteString("\n")
			server.SetStdin(stdin)

			if err := server.ProcessSingleRequest(); err != nil {
				t.Fatalf("ProcessSingleRequest() error = %v, want nil", err)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Should NOT have error in response
			if respErr, ok := resp["error"]; ok {
				t.Errorf("Valid path was rejected: %s (error: %v)", tt.path, respErr)
			} else {
				t.Logf("Correctly accepted valid path: %s", tt.path)
			}
		})
	}
}

// TestServer_WriteFile_MaliciousPaths verifies that malicious write paths are rejected.
func TestServer_WriteFile_MaliciousPaths(t *testing.T) {
	tempDir := t.TempDir()

	config := testserver.DefaultConfig()
	config.AllowedDirectory = tempDir
	server, err := testserver.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	maliciousPaths := []string{
		"../escape.txt",
		"../../etc/passwd",
		"/etc/shadow",
		"subdir/../../escape.txt",
	}

	for _, path := range maliciousPaths {
		t.Run(path, func(t *testing.T) {
			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "write_file",
					"arguments": map[string]interface{}{
						"path":    path,
						"content": "malicious content",
					},
				},
			}

			var stdout bytes.Buffer
			server.SetStdout(&stdout)

			reqJSON, _ := json.Marshal(req)
			stdin := bytes.NewBuffer(reqJSON)
			stdin.WriteString("\n")
			server.SetStdin(stdin)

			// Process request
			server.ProcessSingleRequest()

			var resp map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Should have error in response
			if _, ok := resp["error"]; !ok {
				t.Errorf("Malicious write path was NOT rejected: %s", path)
			} else {
				t.Logf("Correctly rejected malicious write path: %s", path)
			}
		})
	}
}

// TestServer_WriteFile_MaxFileSize verifies file size limit enforcement.
func TestServer_WriteFile_MaxFileSize(t *testing.T) {
	tempDir := t.TempDir()

	config := testserver.DefaultConfig()
	config.AllowedDirectory = tempDir
	config.MaxFileSize = 1024 // 1KB limit for testing
	server, err := testserver.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	tests := []struct {
		name        string
		contentSize int
		wantError   bool
	}{
		{"within limit", 512, false},
		{"at limit", 1024, false},
		{"exceeds limit", 2048, true},
		{"far exceeds limit", 10240, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Repeat("x", tt.contentSize)

			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "write_file",
					"arguments": map[string]interface{}{
						"path":    "test.txt",
						"content": content,
					},
				},
			}

			var stdout bytes.Buffer
			server.SetStdout(&stdout)

			reqJSON, _ := json.Marshal(req)
			stdin := bytes.NewBuffer(reqJSON)
			stdin.WriteString("\n")
			server.SetStdin(stdin)

			server.ProcessSingleRequest()

			var resp map[string]interface{}
			if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			hasError := resp["error"] != nil
			if tt.wantError && !hasError {
				t.Errorf("Expected error for %d byte file (limit: %d), but succeeded", tt.contentSize, config.MaxFileSize)
			} else if !tt.wantError && hasError {
				t.Errorf("Unexpected error for %d byte file (limit: %d): %v", tt.contentSize, config.MaxFileSize, resp["error"])
			}
		})
	}
}

// TestServer_SecurityLogging verifies that security violations are logged.
func TestServer_SecurityLogging(t *testing.T) {
	tempDir := t.TempDir()

	config := testserver.DefaultConfig()
	config.AllowedDirectory = tempDir
	config.LogSecurityEvents = true

	// Capture stderr for log verification
	var stderr bytes.Buffer

	server, err := testserver.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	server.SetStderr(&stderr)

	// Attempt malicious read
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "read_file",
			"arguments": map[string]interface{}{
				"path": "../secret.txt",
			},
		},
	}

	var stdout bytes.Buffer
	server.SetStdout(&stdout)

	reqJSON, _ := json.Marshal(req)
	stdin := bytes.NewBuffer(reqJSON)
	stdin.WriteString("\n")
	server.SetStdin(stdin)

	server.ProcessSingleRequest()

	// Verify security log was written
	logOutput := stderr.String()
	if !strings.Contains(logOutput, "SECURITY") {
		t.Error("Security violation was not logged (missing 'SECURITY' keyword)")
	}
	if !strings.Contains(logOutput, "testserver") {
		t.Error("Security log missing '[testserver]' identifier")
	}
	if !strings.Contains(logOutput, "Rejected") {
		t.Error("Security log missing 'Rejected' operation")
	}
	if !strings.Contains(logOutput, "../secret.txt") {
		t.Error("Security log missing rejected path")
	}

	t.Logf("Security log format verified: %s", logOutput)
}
