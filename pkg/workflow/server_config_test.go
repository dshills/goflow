package workflow

import (
	"testing"
)

func TestServerConfig_GetTransport(t *testing.T) {
	tests := []struct {
		name      string
		config    ServerConfig
		wantValue string
	}{
		{
			name: "defaults to stdio when not specified",
			config: ServerConfig{
				ID:      "test-server",
				Command: "python",
			},
			wantValue: "stdio",
		},
		{
			name: "returns specified transport",
			config: ServerConfig{
				ID:        "test-server",
				Transport: "http",
				URL:       "http://localhost:8080",
			},
			wantValue: "http",
		},
		{
			name: "returns sse transport",
			config: ServerConfig{
				ID:        "test-server",
				Transport: "sse",
				URL:       "http://localhost:8080/events",
			},
			wantValue: "sse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetTransport()
			if got != tt.wantValue {
				t.Errorf("GetTransport() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
		errMsg  string
	}{
		// Valid configurations
		{
			name: "valid stdio config",
			config: ServerConfig{
				ID:      "stdio-server",
				Command: "python",
				Args:    []string{"-m", "mcp_server"},
			},
			wantErr: false,
		},
		{
			name: "valid stdio config with explicit transport",
			config: ServerConfig{
				ID:        "stdio-server",
				Command:   "python",
				Args:      []string{"-m", "mcp_server"},
				Transport: "stdio",
			},
			wantErr: false,
		},
		{
			name: "valid sse config",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "http://localhost:8080/events",
			},
			wantErr: false,
		},
		{
			name: "valid sse config with https",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "https://api.example.com/sse",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
		{
			name: "valid http config",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "http://localhost:8080/rpc",
			},
			wantErr: false,
		},
		{
			name: "valid http config with headers",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "https://api.example.com/rpc",
				Headers: map[string]string{
					"Authorization": "Bearer token",
					"X-API-Key":     "key123",
				},
			},
			wantErr: false,
		},

		// Invalid configurations - general
		{
			name: "missing server ID",
			config: ServerConfig{
				Command: "python",
			},
			wantErr: true,
			errMsg:  "empty server ID",
		},
		{
			name: "invalid transport type",
			config: ServerConfig{
				ID:        "test-server",
				Transport: "websocket",
			},
			wantErr: true,
			errMsg:  "invalid transport type: websocket",
		},

		// Invalid stdio configurations
		{
			name: "stdio missing command",
			config: ServerConfig{
				ID:        "stdio-server",
				Transport: "stdio",
			},
			wantErr: true,
			errMsg:  "command is required for stdio transport",
		},
		{
			name: "stdio with URL specified",
			config: ServerConfig{
				ID:        "stdio-server",
				Transport: "stdio",
				Command:   "python",
				URL:       "http://localhost:8080",
			},
			wantErr: true,
			errMsg:  "URL should not be specified for stdio transport",
		},

		// Invalid SSE configurations
		{
			name: "sse missing URL",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
			},
			wantErr: true,
			errMsg:  "URL is required for sse transport",
		},
		{
			name: "sse with invalid URL scheme",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "ftp://localhost:8080",
			},
			wantErr: true,
			errMsg:  "URL must start with http:// or https://",
		},
		{
			name: "sse with command specified",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "http://localhost:8080",
				Command:   "python",
			},
			wantErr: true,
			errMsg:  "command should not be specified for sse transport",
		},
		{
			name: "sse with args specified",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "http://localhost:8080",
				Args:      []string{"--debug"},
			},
			wantErr: true,
			errMsg:  "args should not be specified for sse transport",
		},

		// Invalid HTTP configurations
		{
			name: "http missing URL",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
			},
			wantErr: true,
			errMsg:  "URL is required for http transport",
		},
		{
			name: "http with invalid URL scheme",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "ws://localhost:8080",
			},
			wantErr: true,
			errMsg:  "URL must start with http:// or https://",
		},
		{
			name: "http with command specified",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "http://localhost:8080",
				Command:   "python",
			},
			wantErr: true,
			errMsg:  "command should not be specified for http transport",
		},
		{
			name: "http with args specified",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "http://localhost:8080",
				Args:      []string{"--debug"},
			},
			wantErr: true,
			errMsg:  "args should not be specified for http transport",
		},

		// Backward compatibility - default to stdio
		{
			name: "backward compatible config without transport",
			config: ServerConfig{
				ID:      "legacy-server",
				Command: "python",
				Args:    []string{"-m", "server"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestServerConfig_MarshalUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
	}{
		{
			name: "stdio config",
			config: ServerConfig{
				ID:      "stdio-server",
				Command: "python",
				Args:    []string{"-m", "server"},
				Env: map[string]string{
					"PATH": "/usr/bin",
				},
			},
		},
		{
			name: "sse config with headers",
			config: ServerConfig{
				ID:        "sse-server",
				Transport: "sse",
				URL:       "https://api.example.com/sse",
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
			},
		},
		{
			name: "http config",
			config: ServerConfig{
				ID:        "http-server",
				Transport: "http",
				URL:       "https://api.example.com/rpc",
				Headers: map[string]string{
					"X-API-Key": "key123",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := tt.config.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error: %v", err)
			}

			// Unmarshal back
			var got ServerConfig
			if err := got.UnmarshalJSON(data); err != nil {
				t.Fatalf("UnmarshalJSON() error: %v", err)
			}

			// Compare (basic field check)
			if got.ID != tt.config.ID {
				t.Errorf("ID mismatch: got %v, want %v", got.ID, tt.config.ID)
			}
			if got.Transport != tt.config.Transport {
				t.Errorf("Transport mismatch: got %v, want %v", got.Transport, tt.config.Transport)
			}
			if got.URL != tt.config.URL {
				t.Errorf("URL mismatch: got %v, want %v", got.URL, tt.config.URL)
			}
		})
	}
}

// Helper function to check if a string contains a substring
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
