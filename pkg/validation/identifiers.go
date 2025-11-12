package validation

// IsValidIdentifierChar checks if a character is valid for identifiers
// (alphanumeric, hyphen, or underscore).
//
// This function is used to validate server IDs, workflow names, and other
// user-provided identifiers in GoFlow. It enforces a consistent naming
// convention across the application.
//
// Valid characters:
//   - Lowercase letters: a-z
//   - Uppercase letters: A-Z
//   - Digits: 0-9
//   - Hyphen: -
//   - Underscore: _
//
// Example usage:
//
//	func isValidServerID(id string) bool {
//	    if id == "" {
//	        return false
//	    }
//	    for _, ch := range id {
//	        if !validation.IsValidIdentifierChar(ch) {
//	            return false
//	        }
//	    }
//	    return true
//	}
func IsValidIdentifierChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' || ch == '_'
}
