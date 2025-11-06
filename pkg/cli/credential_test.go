package cli

import (
	"testing"
)

func TestIsOnlyWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: true,
		},
		{
			name:     "ASCII space only",
			input:    []byte("   "),
			expected: true,
		},
		{
			name:     "ASCII tab only",
			input:    []byte("\t\t"),
			expected: true,
		},
		{
			name:     "ASCII newline only",
			input:    []byte("\n"),
			expected: true,
		},
		{
			name:     "mixed ASCII whitespace",
			input:    []byte(" \t\n\r"),
			expected: true,
		},
		{
			name:     "Unicode whitespace (non-breaking space U+00A0)",
			input:    []byte("\u00A0\u00A0"),
			expected: true,
		},
		{
			name:     "Unicode whitespace (em space U+2003)",
			input:    []byte("\u2003"),
			expected: true,
		},
		{
			name:     "mixed ASCII and Unicode whitespace",
			input:    []byte(" \t\u00A0\u2003\n"),
			expected: true,
		},
		{
			name:     "single non-whitespace character",
			input:    []byte("a"),
			expected: false,
		},
		{
			name:     "non-whitespace with surrounding spaces",
			input:    []byte("  a  "),
			expected: false,
		},
		{
			name:     "typical password",
			input:    []byte("MyS3cr3tP@ssw0rd"),
			expected: false,
		},
		{
			name:     "API key format",
			input:    []byte("sk-1234567890abcdef"),
			expected: false,
		},
		{
			name:     "binary credential (invalid UTF-8)",
			input:    []byte{0xFF, 0xFE, 0xFD},
			expected: false,
		},
		{
			name:     "binary with valid bytes mixed",
			input:    []byte{0x41, 0xFF, 0x42}, // A, invalid, B
			expected: false,
		},
		{
			name:     "valid U+FFFD replacement character (3 bytes: 0xEF 0xBF 0xBD)",
			input:    []byte{0xEF, 0xBF, 0xBD}, // This is VALID UTF-8 for U+FFFD
			expected: false,                    // U+FFFD is not a whitespace character
		},
		{
			name:     "credential containing U+FFFD",
			input:    []byte("secret\uFFFDkey"),
			expected: false, // Contains non-whitespace characters
		},
		{
			name:     "invalid UTF-8 followed by whitespace",
			input:    []byte{0xFF, 0x20}, // invalid byte, then space
			expected: false,              // Invalid UTF-8 should trigger false
		},
		{
			name:     "whitespace followed by invalid UTF-8",
			input:    []byte{0x20, 0xFF}, // space, then invalid byte
			expected: false,              // Invalid UTF-8 should trigger false
		},
		{
			name:     "password with special characters",
			input:    []byte("P@ssw0rd!#$%"),
			expected: false,
		},
		{
			name:     "base64-like string",
			input:    []byte("YWJjZGVmZ2hpamtsbW5vcA=="),
			expected: false,
		},
		{
			name:     "JWT-like string",
			input:    []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"),
			expected: false,
		},
		{
			name:     "single space",
			input:    []byte(" "),
			expected: true,
		},
		{
			name:     "multiple newlines",
			input:    []byte("\n\n\n"),
			expected: true,
		},
		{
			name:     "CR LF sequence",
			input:    []byte("\r\n"),
			expected: true,
		},
		{
			name:     "zero-width space (U+200B)",
			input:    []byte("\u200B"),
			expected: false, // U+200B is not in Unicode White Space property
		},
		{
			name:     "credential with zero byte (binary)",
			input:    []byte{0x00, 0x41}, // null byte, then 'A'
			expected: false,              // Null is not whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOnlyWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("isOnlyWhitespace(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsOnlyWhitespace_UTF8ReplacementCharacter specifically tests that valid U+FFFD
// characters are correctly distinguished from invalid UTF-8 sequences
func TestIsOnlyWhitespace_UTF8ReplacementCharacter(t *testing.T) {
	// Valid U+FFFD in UTF-8 is 3 bytes: 0xEF 0xBF 0xBD
	// DecodeRune should return (U+FFFD, 3) for valid encoding
	validUFFFD := []byte{0xEF, 0xBF, 0xBD}
	result := isOnlyWhitespace(validUFFFD)

	// U+FFFD is NOT a whitespace character, so should return false
	if result != false {
		t.Errorf("Valid U+FFFD should not be considered whitespace, got %v", result)
	}

	// Invalid UTF-8 that produces RuneError with size=1
	// DecodeRune returns (RuneError, 1) for invalid sequences
	invalidUTF8 := []byte{0xFF}
	result = isOnlyWhitespace(invalidUTF8)

	// Invalid UTF-8 should return false (treated as non-whitespace)
	if result != false {
		t.Errorf("Invalid UTF-8 should return false, got %v", result)
	}

	// Verify the implementation correctly distinguishes between:
	// 1. Valid U+FFFD (size=3) - treated according to its Unicode properties (not whitespace)
	// 2. Invalid UTF-8 (size=1) - treated as non-whitespace (allows binary credentials)
}
