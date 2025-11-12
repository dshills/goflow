package validation

import "testing"

func TestIsValidIdentifierChar(t *testing.T) {
	tests := []struct {
		name string
		ch   rune
		want bool
	}{
		// Valid characters
		{"lowercase a", 'a', true},
		{"lowercase z", 'z', true},
		{"uppercase A", 'A', true},
		{"uppercase Z", 'Z', true},
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"hyphen", '-', true},
		{"underscore", '_', true},

		// Invalid characters
		{"space", ' ', false},
		{"dot", '.', false},
		{"slash", '/', false},
		{"backslash", '\\', false},
		{"colon", ':', false},
		{"semicolon", ';', false},
		{"asterisk", '*', false},
		{"question mark", '?', false},
		{"exclamation", '!', false},
		{"at sign", '@', false},
		{"hash", '#', false},
		{"dollar", '$', false},
		{"percent", '%', false},
		{"caret", '^', false},
		{"ampersand", '&', false},
		{"parenthesis", '(', false},
		{"bracket", '[', false},
		{"brace", '{', false},
		{"less than", '<', false},
		{"greater than", '>', false},
		{"pipe", '|', false},
		{"backtick", '`', false},
		{"tilde", '~', false},
		{"newline", '\n', false},
		{"tab", '\t', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidIdentifierChar(tt.ch); got != tt.want {
				t.Errorf("IsValidIdentifierChar(%q) = %v, want %v", tt.ch, got, tt.want)
			}
		})
	}
}

// TestIdentifierValidation demonstrates how to use IsValidIdentifierChar
// to validate complete identifiers
func TestIdentifierValidation(t *testing.T) {
	isValidIdentifier := func(id string) bool {
		if id == "" {
			return false
		}
		for _, ch := range id {
			if !IsValidIdentifierChar(ch) {
				return false
			}
		}
		return true
	}

	tests := []struct {
		name       string
		identifier string
		want       bool
	}{
		// Valid identifiers
		{"simple lowercase", "test", true},
		{"simple uppercase", "TEST", true},
		{"mixed case", "TestCase", true},
		{"with digits", "test123", true},
		{"with hyphens", "test-case", true},
		{"with underscores", "test_case", true},
		{"complex valid", "my-server_123", true},

		// Invalid identifiers
		{"empty string", "", false},
		{"with space", "test case", false},
		{"with dot", "test.case", false},
		{"with slash", "test/case", false},
		{"with special char", "test@case", false},
		{"path traversal", "../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIdentifier(tt.identifier); got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.identifier, got, tt.want)
			}
		})
	}
}
