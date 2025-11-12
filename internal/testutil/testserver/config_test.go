package testserver_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dshills/goflow/internal/testutil/testserver"
)

// TestDefaultConfig verifies that DefaultConfig returns sensible defaults.
func TestDefaultConfig(t *testing.T) {
	config := testserver.DefaultConfig()

	// Verify secure defaults are set
	if config.AllowedDirectory == "" {
		t.Error("DefaultConfig() AllowedDirectory is empty, want os.TempDir()")
	}

	if !filepath.IsAbs(config.AllowedDirectory) {
		t.Errorf("DefaultConfig() AllowedDirectory = %q, want absolute path", config.AllowedDirectory)
	}

	// Check that temp dir exists
	if _, err := os.Stat(config.AllowedDirectory); err != nil {
		t.Errorf("DefaultConfig() AllowedDirectory %q does not exist: %v", config.AllowedDirectory, err)
	}

	// Verify max file size is 10MB
	expectedMaxSize := int64(10 * 1024 * 1024)
	if config.MaxFileSize != expectedMaxSize {
		t.Errorf("DefaultConfig() MaxFileSize = %d, want %d", config.MaxFileSize, expectedMaxSize)
	}

	// Verify security logging is enabled by default
	if !config.LogSecurityEvents {
		t.Error("DefaultConfig() LogSecurityEvents = false, want true")
	}

	// Verify log file path is stderr (empty string)
	if config.LogFilePath != "" {
		t.Errorf("DefaultConfig() LogFilePath = %q, want empty string (stderr)", config.LogFilePath)
	}

	// Verify timeouts are set
	if config.ReadTimeout != 5*time.Second {
		t.Errorf("DefaultConfig() ReadTimeout = %v, want 5s", config.ReadTimeout)
	}

	if config.WriteTimeout != 5*time.Second {
		t.Errorf("DefaultConfig() WriteTimeout = %v, want 5s", config.WriteTimeout)
	}
}

// TestLoadConfig_Defaults verifies LoadConfig returns defaults when no overrides exist.
func TestLoadConfig_Defaults(t *testing.T) {
	// Clear any environment variables that might affect the test
	oldAllowedDir := os.Getenv("GOFLOW_TESTSERVER_ALLOWED_DIR")
	oldMaxSize := os.Getenv("GOFLOW_TESTSERVER_MAX_FILE_SIZE")
	oldLogSecurity := os.Getenv("GOFLOW_TESTSERVER_LOG_SECURITY")

	os.Unsetenv("GOFLOW_TESTSERVER_ALLOWED_DIR")
	os.Unsetenv("GOFLOW_TESTSERVER_MAX_FILE_SIZE")
	os.Unsetenv("GOFLOW_TESTSERVER_LOG_SECURITY")

	defer func() {
		// Restore environment variables
		if oldAllowedDir != "" {
			os.Setenv("GOFLOW_TESTSERVER_ALLOWED_DIR", oldAllowedDir)
		}
		if oldMaxSize != "" {
			os.Setenv("GOFLOW_TESTSERVER_MAX_FILE_SIZE", oldMaxSize)
		}
		if oldLogSecurity != "" {
			os.Setenv("GOFLOW_TESTSERVER_LOG_SECURITY", oldLogSecurity)
		}
	}()

	config := testserver.LoadConfig()

	// Should match defaults
	defaultConfig := testserver.DefaultConfig()
	if config.AllowedDirectory != defaultConfig.AllowedDirectory {
		t.Errorf("LoadConfig() AllowedDirectory = %q, want %q", config.AllowedDirectory, defaultConfig.AllowedDirectory)
	}

	if config.MaxFileSize != defaultConfig.MaxFileSize {
		t.Errorf("LoadConfig() MaxFileSize = %d, want %d", config.MaxFileSize, defaultConfig.MaxFileSize)
	}

	if config.LogSecurityEvents != defaultConfig.LogSecurityEvents {
		t.Errorf("LoadConfig() LogSecurityEvents = %v, want %v", config.LogSecurityEvents, defaultConfig.LogSecurityEvents)
	}
}

// TestLoadConfig_EnvironmentVariables verifies environment variable precedence.
func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		envVars   map[string]string
		wantDir   string
		wantSize  int64
		wantLog   bool
		wantError bool
	}{
		{
			name: "override allowed directory",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_ALLOWED_DIR": tempDir,
			},
			wantDir:  tempDir,
			wantSize: 10 * 1024 * 1024, // default
			wantLog:  true,             // default
		},
		{
			name: "override max file size",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_MAX_FILE_SIZE": "5242880", // 5MB
			},
			wantDir:  os.TempDir(), // default
			wantSize: 5242880,
			wantLog:  true, // default
		},
		{
			name: "disable security logging",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_LOG_SECURITY": "false",
			},
			wantDir:  os.TempDir(),     // default
			wantSize: 10 * 1024 * 1024, // default
			wantLog:  false,
		},
		{
			name: "enable security logging explicitly",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_LOG_SECURITY": "true",
			},
			wantDir:  os.TempDir(),     // default
			wantSize: 10 * 1024 * 1024, // default
			wantLog:  true,
		},
		{
			name: "all overrides",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_ALLOWED_DIR":   tempDir,
				"GOFLOW_TESTSERVER_MAX_FILE_SIZE": "1048576", // 1MB
				"GOFLOW_TESTSERVER_LOG_SECURITY":  "false",
			},
			wantDir:  tempDir,
			wantSize: 1048576,
			wantLog:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up environment variables after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			config := testserver.LoadConfig()

			if config.AllowedDirectory != tt.wantDir {
				t.Errorf("LoadConfig() AllowedDirectory = %q, want %q", config.AllowedDirectory, tt.wantDir)
			}

			if config.MaxFileSize != tt.wantSize {
				t.Errorf("LoadConfig() MaxFileSize = %d, want %d", config.MaxFileSize, tt.wantSize)
			}

			if config.LogSecurityEvents != tt.wantLog {
				t.Errorf("LoadConfig() LogSecurityEvents = %v, want %v", config.LogSecurityEvents, tt.wantLog)
			}
		})
	}
}

// TestLoadConfig_InvalidEnvironmentVariables verifies error handling for invalid env vars.
func TestLoadConfig_InvalidEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name: "invalid max file size - not a number",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_MAX_FILE_SIZE": "invalid",
			},
		},
		{
			name: "invalid max file size - negative",
			envVars: map[string]string{
				"GOFLOW_TESTSERVER_MAX_FILE_SIZE": "-1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up environment variables after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// LoadConfig should not panic and should use defaults for invalid values
			config := testserver.LoadConfig()

			// Should fall back to default max file size
			if _, exists := tt.envVars["GOFLOW_TESTSERVER_MAX_FILE_SIZE"]; exists {
				expectedDefault := int64(10 * 1024 * 1024)
				if config.MaxFileSize != expectedDefault {
					t.Errorf("LoadConfig() with invalid MAX_FILE_SIZE should use default %d, got %d", expectedDefault, config.MaxFileSize)
				}
			}
		})
	}
}

// TestServerConfig_Validation verifies configuration validation logic.
func TestServerConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    *testserver.ServerConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid config",
			config:    testserver.DefaultConfig(),
			wantError: false,
		},
		{
			name: "relative path not allowed",
			config: &testserver.ServerConfig{
				AllowedDirectory:  "relative/path",
				MaxFileSize:       10 * 1024 * 1024,
				LogSecurityEvents: true,
			},
			wantError: true,
			errorMsg:  "absolute path",
		},
		{
			name: "non-existent directory",
			config: &testserver.ServerConfig{
				AllowedDirectory:  "/this/path/does/not/exist/12345",
				MaxFileSize:       10 * 1024 * 1024,
				LogSecurityEvents: true,
			},
			wantError: true,
			errorMsg:  "does not exist",
		},
		{
			name: "negative max file size",
			config: &testserver.ServerConfig{
				AllowedDirectory:  os.TempDir(),
				MaxFileSize:       -1,
				LogSecurityEvents: true,
			},
			wantError: true,
			errorMsg:  "positive",
		},
		{
			name: "zero max file size",
			config: &testserver.ServerConfig{
				AllowedDirectory:  os.TempDir(),
				MaxFileSize:       0,
				LogSecurityEvents: true,
			},
			wantError: true,
			errorMsg:  "positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				if err == nil {
					t.Errorf("ServerConfig.Validate() error = nil, want error containing %q", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ServerConfig.Validate() error = %q, want error containing %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ServerConfig.Validate() error = %v, want nil", err)
				}
			}
		})
	}
}

// contains checks if a string contains a substring (case-insensitive helper).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
