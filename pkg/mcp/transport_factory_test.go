package mcp

import (
	"testing"
)

func TestCreateClient_Stdio(t *testing.T) {
	config := ServerConfig{
		ID:        "test-stdio",
		Command:   "echo",
		Args:      []string{"test"},
		Transport: "stdio",
	}

	client, err := createClient(config)
	if err != nil {
		t.Fatalf("createClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("createClient() returned nil client")
	}

	// Verify it's a StdioClient (type assertion)
	if _, ok := client.(*StdioClient); !ok {
		t.Errorf("createClient() returned wrong client type, want *StdioClient")
	}
}

func TestCreateClient_SSE(t *testing.T) {
	config := ServerConfig{
		ID:        "test-sse",
		Transport: "sse",
		URL:       "http://localhost:8080/sse",
		Headers: map[string]string{
			"Authorization": "Bearer test",
		},
	}

	client, err := createClient(config)
	if err != nil {
		t.Fatalf("createClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("createClient() returned nil client")
	}

	// Verify it's an SSEClient (type assertion)
	if _, ok := client.(*SSEClient); !ok {
		t.Errorf("createClient() returned wrong client type, want *SSEClient")
	}
}

func TestCreateClient_HTTP(t *testing.T) {
	config := ServerConfig{
		ID:        "test-http",
		Transport: "http",
		URL:       "https://api.example.com/rpc",
		Headers: map[string]string{
			"X-API-Key": "test-key",
		},
	}

	client, err := createClient(config)
	if err != nil {
		t.Fatalf("createClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("createClient() returned nil client")
	}

	// Verify it's an HTTPClient (type assertion)
	if _, ok := client.(*HTTPClient); !ok {
		t.Errorf("createClient() returned wrong client type, want *HTTPClient")
	}
}

func TestCreateClient_DefaultsToStdio(t *testing.T) {
	config := ServerConfig{
		ID:      "test-default",
		Command: "echo",
		Args:    []string{"test"},
		// Transport not specified - should default to stdio
	}

	client, err := createClient(config)
	if err != nil {
		t.Fatalf("createClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("createClient() returned nil client")
	}

	// Verify it defaults to StdioClient
	if _, ok := client.(*StdioClient); !ok {
		t.Errorf("createClient() with no transport should default to *StdioClient")
	}
}

func TestCreateClient_InvalidTransport(t *testing.T) {
	config := ServerConfig{
		ID:        "test-invalid",
		Transport: "websocket", // Invalid transport
		URL:       "ws://localhost:8080",
	}

	client, err := createClient(config)
	if err == nil {
		t.Fatal("createClient() expected error for invalid transport, got nil")
	}

	if client != nil {
		t.Errorf("createClient() should return nil client on error")
	}

	expectedErr := "unsupported transport type: websocket"
	if err.Error() != expectedErr {
		t.Errorf("createClient() error = %v, want %v", err.Error(), expectedErr)
	}
}

func TestCreateClient_SSEMissingURL(t *testing.T) {
	config := ServerConfig{
		ID:        "test-sse-no-url",
		Transport: "sse",
		// Missing URL
	}

	client, err := createClient(config)
	if err == nil {
		t.Fatal("createClient() expected error for SSE without URL, got nil")
	}

	// Note: Some client constructors may return a client object even on error
	// The important thing is that err is not nil
	_ = client
}

func TestCreateClient_HTTPMissingURL(t *testing.T) {
	config := ServerConfig{
		ID:        "test-http-no-url",
		Transport: "http",
		// Missing URL
	}

	client, err := createClient(config)
	if err == nil {
		t.Fatal("createClient() expected error for HTTP without URL, got nil")
	}

	// Note: Some client constructors may return a client object even on error
	// The important thing is that err is not nil
	_ = client
}

func TestCreateClient_AllTransportTypes(t *testing.T) {
	tests := []struct {
		name           string
		config         ServerConfig
		wantClientType string
		wantErr        bool
	}{
		{
			name: "stdio transport",
			config: ServerConfig{
				ID:        "stdio-test",
				Transport: "stdio",
				Command:   "python",
			},
			wantClientType: "*mcp.StdioClient",
			wantErr:        false,
		},
		{
			name: "sse transport",
			config: ServerConfig{
				ID:        "sse-test",
				Transport: "sse",
				URL:       "http://localhost:8080/sse",
			},
			wantClientType: "*mcp.SSEClient",
			wantErr:        false,
		},
		{
			name: "http transport",
			config: ServerConfig{
				ID:        "http-test",
				Transport: "http",
				URL:       "http://localhost:8080/rpc",
			},
			wantClientType: "*mcp.HTTPClient",
			wantErr:        false,
		},
		{
			name: "default to stdio",
			config: ServerConfig{
				ID:      "default-test",
				Command: "node",
			},
			wantClientType: "*mcp.StdioClient",
			wantErr:        false,
		},
		{
			name: "invalid transport",
			config: ServerConfig{
				ID:        "invalid-test",
				Transport: "grpc",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := createClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("createClient() expected error, got nil")
				}
				if client != nil {
					t.Errorf("createClient() should return nil client on error")
				}
				return
			}

			if err != nil {
				t.Errorf("createClient() unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Fatal("createClient() returned nil client")
			}

			// Verify client type matches expected
			clientType := ""
			switch client.(type) {
			case *StdioClient:
				clientType = "*mcp.StdioClient"
			case *SSEClient:
				clientType = "*mcp.SSEClient"
			case *HTTPClient:
				clientType = "*mcp.HTTPClient"
			}

			if clientType != tt.wantClientType {
				t.Errorf("createClient() client type = %v, want %v", clientType, tt.wantClientType)
			}
		})
	}
}
