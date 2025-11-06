package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCredentialAddCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		validate    func(t *testing.T)
	}{
		{
			name: "add with environment variables",
			args: []string{"test-server", "--env", "API_KEY=secret123", "--env", "TOKEN=abc456"},
			validate: func(t *testing.T) {
				cred := GetCredential("test-server")
				if cred == nil {
					t.Fatal("credential not stored")
				}
				if len(cred.EnvVars) != 2 {
					t.Errorf("expected 2 env vars, got %d", len(cred.EnvVars))
				}
				if cred.EnvVars["API_KEY"] != "secret123" {
					t.Errorf("API_KEY = %s, want secret123", cred.EnvVars["API_KEY"])
				}
				if cred.EnvVars["TOKEN"] != "abc456" {
					t.Errorf("TOKEN = %s, want abc456", cred.EnvVars["TOKEN"])
				}
			},
		},
		{
			name: "add with credential reference",
			args: []string{"ref-server", "--credential-ref", "oauth-token"},
			validate: func(t *testing.T) {
				cred := GetCredential("ref-server")
				if cred == nil {
					t.Fatal("credential not stored")
				}
				if cred.CredentialRef != "oauth-token" {
					t.Errorf("CredentialRef = %s, want oauth-token", cred.CredentialRef)
				}
			},
		},
		{
			name: "add with mixed credentials",
			args: []string{"mixed-server", "--env", "DEBUG=true", "--credential-ref", "aws-profile"},
			validate: func(t *testing.T) {
				cred := GetCredential("mixed-server")
				if cred == nil {
					t.Fatal("credential not stored")
				}
				if len(cred.EnvVars) != 1 {
					t.Errorf("expected 1 env var, got %d", len(cred.EnvVars))
				}
				if cred.EnvVars["DEBUG"] != "true" {
					t.Errorf("DEBUG = %s, want true", cred.EnvVars["DEBUG"])
				}
				if cred.CredentialRef != "aws-profile" {
					t.Errorf("CredentialRef = %s, want aws-profile", cred.CredentialRef)
				}
			},
		},
		{
			name:        "error: invalid server ID",
			args:        []string{"invalid@server!", "--env", "KEY=value"},
			wantErr:     true,
			errContains: "invalid server ID",
		},
		{
			name:        "error: no credentials provided",
			args:        []string{"test-server"},
			wantErr:     true,
			errContains: "must provide at least one of --env or --credential-ref",
		},
		{
			name:        "error: invalid env var format (no equals)",
			args:        []string{"test-server", "--env", "INVALID"},
			wantErr:     true,
			errContains: "invalid environment variable format",
		},
		{
			name:        "error: invalid env var format (empty key)",
			args:        []string{"test-server", "--env", "=value"},
			wantErr:     true,
			errContains: "environment variable key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset credential store
			GlobalCredentialStore.mu.Lock()
			GlobalCredentialStore.credentials = make(map[string]*Credential)
			GlobalCredentialStore.mu.Unlock()

			cmd := NewCredentialCommand()
			// Prepend "add" subcommand to args
			args := append([]string{"add"}, tt.args...)
			cmd.SetArgs(args)

			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t)
				}
			}
		})
	}
}

func TestCredentialListCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		contains []string
	}{
		{
			name: "list empty credentials",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = make(map[string]*Credential)
				GlobalCredentialStore.mu.Unlock()
			},
			contains: []string{"No credentials stored"},
		},
		{
			name: "list with environment variables",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = map[string]*Credential{
					"test-server": {
						ServerID: "test-server",
						EnvVars: map[string]string{
							"API_KEY": "secret",
							"TOKEN":   "abc",
						},
					},
				}
				GlobalCredentialStore.mu.Unlock()
			},
			contains: []string{"test-server", "API_KEY, TOKEN", "Environment"},
		},
		{
			name: "list with credential reference",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = map[string]*Credential{
					"ref-server": {
						ServerID:      "ref-server",
						CredentialRef: "oauth-token",
					},
				}
				GlobalCredentialStore.mu.Unlock()
			},
			contains: []string{"ref-server", "oauth-token", "Reference"},
		},
		{
			name: "list with mixed credentials",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = map[string]*Credential{
					"mixed-server": {
						ServerID:      "mixed-server",
						EnvVars:       map[string]string{"DEBUG": "true"},
						CredentialRef: "aws-profile",
					},
				}
				GlobalCredentialStore.mu.Unlock()
			},
			contains: []string{"mixed-server", "DEBUG", "aws-profile", "Mixed"},
		},
		{
			name: "list with many env vars shows count",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = map[string]*Credential{
					"many-server": {
						ServerID: "many-server",
						EnvVars: map[string]string{
							"KEY1": "val1",
							"KEY2": "val2",
							"KEY3": "val3",
							"KEY4": "val4",
						},
					},
				}
				GlobalCredentialStore.mu.Unlock()
			},
			contains: []string{"many-server", "4 keys"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			cmd := NewCredentialCommand()
			cmd.SetArgs([]string{"list"})
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := stdout.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestCredentialRemoveCommand(t *testing.T) {
	tests := []struct {
		name        string
		setup       func()
		args        []string
		wantErr     bool
		errContains string
		validate    func(t *testing.T)
	}{
		{
			name: "remove existing credential",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = map[string]*Credential{
					"test-server": {
						ServerID: "test-server",
						EnvVars:  map[string]string{"KEY": "value"},
					},
				}
				GlobalCredentialStore.mu.Unlock()
			},
			args: []string{"test-server"},
			validate: func(t *testing.T) {
				cred := GetCredential("test-server")
				if cred != nil {
					t.Error("credential should be removed")
				}
			},
		},
		{
			name: "error: remove non-existent credential",
			setup: func() {
				GlobalCredentialStore.mu.Lock()
				GlobalCredentialStore.credentials = make(map[string]*Credential)
				GlobalCredentialStore.mu.Unlock()
			},
			args:        []string{"missing-server"},
			wantErr:     true,
			errContains: "no credentials found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			cmd := NewCredentialCommand()
			// Prepend "remove" subcommand to args
			args := append([]string{"remove"}, tt.args...)
			cmd.SetArgs(args)

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t)
				}
			}
		})
	}
}

func TestGetCredential(t *testing.T) {
	// Setup test data
	GlobalCredentialStore.mu.Lock()
	GlobalCredentialStore.credentials = map[string]*Credential{
		"test-server": {
			ServerID: "test-server",
			EnvVars: map[string]string{
				"API_KEY": "secret123",
			},
			CredentialRef: "oauth-token",
		},
	}
	GlobalCredentialStore.mu.Unlock()

	t.Run("get existing credential", func(t *testing.T) {
		cred := GetCredential("test-server")
		if cred == nil {
			t.Fatal("expected credential, got nil")
		}
		if cred.ServerID != "test-server" {
			t.Errorf("ServerID = %s, want test-server", cred.ServerID)
		}
		if cred.EnvVars["API_KEY"] != "secret123" {
			t.Errorf("API_KEY = %s, want secret123", cred.EnvVars["API_KEY"])
		}
		if cred.CredentialRef != "oauth-token" {
			t.Errorf("CredentialRef = %s, want oauth-token", cred.CredentialRef)
		}
	})

	t.Run("get non-existent credential", func(t *testing.T) {
		cred := GetCredential("missing-server")
		if cred != nil {
			t.Errorf("expected nil, got %+v", cred)
		}
	})

	t.Run("returned credential is a copy", func(t *testing.T) {
		cred := GetCredential("test-server")
		if cred == nil {
			t.Fatal("expected credential, got nil")
		}

		// Modify the returned credential
		cred.EnvVars["API_KEY"] = "modified"
		cred.CredentialRef = "modified"

		// Original should be unchanged
		original := GetCredential("test-server")
		if original.EnvVars["API_KEY"] == "modified" {
			t.Error("original credential was modified")
		}
		if original.CredentialRef == "modified" {
			t.Error("original credential ref was modified")
		}
	})
}

func TestIsValidServerID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"valid-server", true},
		{"valid_server", true},
		{"ValidServer123", true},
		{"server-123_test", true},
		{"", false},
		{"invalid server", false}, // space
		{"invalid@server", false}, // @ symbol
		{"invalid.server", false}, // dot
		{"invalid/server", false}, // slash
		{"invalid:server", false}, // colon
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidServerID(tt.input)
			if got != tt.want {
				t.Errorf("isValidServerID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
