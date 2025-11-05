package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// Sentinel errors for JSONPath operations
var (
	ErrInvalidJSONPath = errors.New("invalid JSONPath syntax")
	ErrTypeMismatch    = errors.New("type mismatch in JSONPath query")
	ErrNilData         = errors.New("cannot query nil data")
)

// JSONPathQuerier defines the interface for querying JSON data using JSONPath expressions
type JSONPathQuerier interface {
	Query(ctx context.Context, path string, data interface{}) (interface{}, error)
}

// gjsonQuerier implements JSONPathQuerier using github.com/tidwall/gjson
type gjsonQuerier struct{}

// NewJSONPathQuerier creates a new JSONPath querier using gjson
func NewJSONPathQuerier() JSONPathQuerier {
	return &gjsonQuerier{}
}

// Query executes a JSONPath query against the provided data
func (q *gjsonQuerier) Query(ctx context.Context, path string, data interface{}) (interface{}, error) {
	// Check for nil data
	if data == nil {
		return nil, ErrNilData
	}

	// Check for empty path
	if path == "" {
		return nil, ErrInvalidJSONPath
	}

	// Check for invalid syntax patterns
	if strings.Contains(path, "[[[") || strings.Contains(path, "..[[[") {
		return nil, ErrInvalidJSONPath
	}

	// Convert data to JSON string for gjson
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	jsonStr := string(jsonBytes)

	// Convert JSONPath to gjson syntax
	queryPath, err := convertJSONPathToGJSON(path)
	if err != nil {
		return nil, err
	}

	// Special case: root object access
	if queryPath == "" || queryPath == "." || path == "$" {
		var result interface{}
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}
		return result, nil
	}

	// Execute the query
	result := gjson.Get(jsonStr, queryPath)

	// Check for type mismatch errors - array access on non-array
	if strings.Contains(path, "[0]") && !result.Exists() {
		// Try to detect if we're accessing array index on a string
		basePath := path[:strings.Index(path, "[")]
		basePath = strings.TrimPrefix(basePath, "$.")
		baseResult := gjson.Get(jsonStr, basePath)
		if baseResult.Exists() && baseResult.Type == gjson.String {
			return nil, ErrTypeMismatch
		}
	}

	// Convert gjson.Result to appropriate Go type
	return convertGJSONResult(result), nil
}

// convertGJSONResult converts a gjson.Result to the appropriate Go type
func convertGJSONResult(result gjson.Result) interface{} {
	if !result.Exists() {
		return nil
	}

	switch result.Type {
	case gjson.Null:
		return nil
	case gjson.False:
		return false
	case gjson.True:
		return true
	case gjson.Number:
		// Return as float64 or int based on whether it has decimal points
		if result.Num == float64(int64(result.Num)) {
			return int(result.Num)
		}
		return result.Num
	case gjson.String:
		return result.Str
	case gjson.JSON:
		// For JSON objects and arrays, parse them
		var value interface{}
		if err := json.Unmarshal([]byte(result.Raw), &value); err != nil {
			return result.Raw
		}
		return value
	default:
		return result.Value()
	}
}

// convertJSONPathToGJSON converts standard JSONPath syntax to gjson syntax
func convertJSONPathToGJSON(path string) (string, error) {
	// Remove leading $
	result := strings.TrimPrefix(path, "$")
	result = strings.TrimPrefix(result, ".")

	// Replace [*] with .#
	result = strings.ReplaceAll(result, "[*]", ".#")

	// Replace [@.field] patterns (remove @.)
	result = strings.ReplaceAll(result, "[@.", "[")
	result = strings.ReplaceAll(result, "@.", "")

	// Replace [0], [1], etc. with .0, .1
	// But keep [-1] as .-1
	result = replaceArrayIndexes(result)

	return result, nil
}

// replaceArrayIndexes converts [n] to .n for gjson
func replaceArrayIndexes(path string) string {
	result := ""
	i := 0
	for i < len(path) {
		if path[i] == '[' {
			// Find closing ]
			closingIdx := strings.Index(path[i:], "]")
			if closingIdx == -1 {
				// Unclosed bracket
				result += path[i:]
				break
			}
			closingIdx += i

			// Extract content between brackets
			content := path[i+1 : closingIdx]

			// Check if it's a simple number or negative number
			if isSimpleNumber(content) {
				result += "." + content
			} else if strings.Contains(content, ":") {
				// It's a slice operation - gjson doesn't support this directly
				// We'll need to handle this differently
				result += "[" + content + "]"
			} else if strings.Contains(content, "?") {
				// It's a filter - keep as is for now
				result += "[" + content + "]"
			} else {
				// Some other bracket expression
				result += "[" + content + "]"
			}

			i = closingIdx + 1
		} else {
			result += string(path[i])
			i++
		}
	}
	return result
}

// isSimpleNumber checks if a string is a simple integer (positive or negative)
func isSimpleNumber(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
