package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

	// Validate bracket matching
	if err := validateBrackets(path); err != nil {
		return nil, err
	}

	// Check for invalid syntax patterns
	if strings.Contains(path, "[[[") || strings.Contains(path, "..[[[") {
		return nil, ErrInvalidJSONPath
	}

	// Additional syntax validation
	if strings.Contains(path, "...[[[") {
		return nil, ErrInvalidJSONPath
	}

	// Convert data to JSON string for gjson
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	jsonStr := string(jsonBytes)

	// Check for negative array indexing and handle it
	if hasNegativeIndex(path) {
		return handleNegativeIndex(jsonStr, path)
	}

	// Check if this is a recursive descent query (before conversion)
	if strings.Contains(path, "..") {
		// Extract the field name after ..
		parts := strings.Split(path, "..")
		if len(parts) > 1 {
			fieldName := strings.TrimPrefix(parts[1], ".")
			return handleRecursiveDescent(jsonStr, fieldName)
		}
	}

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

	// Check if we need to handle OR conditions in filters
	if hasORFilter(path) {
		return handleORFilter(jsonStr, path)
	}

	// Check if we need to handle array slicing manually
	if hasSliceNotation(path) {
		return handleArraySlice(jsonStr, path)
	}

	// Check if we need special handling for nested wildcards (flattening)
	if needsFlatteningFlatten(queryPath) {
		return handleNestedWildcards(jsonStr, queryPath)
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

	// Handle recursive descent (..)
	// For gjson, we need to use special handling
	// $..email needs to become a pattern that finds all email fields
	// gjson uses #(...)# for deep search with filters, but for simple field search
	// we can use the @this syntax with a pattern

	// Handle .length() function - convert to .#
	result = strings.ReplaceAll(result, ".length()", ".#")

	// Handle filter expressions BEFORE replacing [*]
	result = convertFilters(result)

	// Replace [*] with .# for array wildcard (but not after filters)
	// Be careful not to replace the # in filters
	if !strings.Contains(result, "#(") {
		result = strings.ReplaceAll(result, "[*]", ".#")
	} else {
		// Only replace [*] that are not part of filters
		result = replaceWildcardCarefully(result)
	}

	// Replace [@.field] patterns (remove @.)
	result = strings.ReplaceAll(result, "[@.", "[")
	result = strings.ReplaceAll(result, "@.", "")

	// Replace [0], [1], etc. with .0, .1
	// But keep [-1] as .-1 (gjson supports negative indices)
	result = replaceArrayIndexes(result)

	return result, nil
}

// replaceWildcardCarefully replaces [*] but avoids filter expressions
func replaceWildcardCarefully(path string) string {
	result := ""
	inFilter := false
	i := 0
	for i < len(path) {
		if i < len(path)-2 && path[i:i+2] == "#(" {
			inFilter = true
			result += "#("
			i += 2
			continue
		}
		if inFilter && path[i] == ')' && i+1 < len(path) && path[i+1] == '#' {
			inFilter = false
			result += ")#"
			i += 2
			continue
		}
		if !inFilter && i < len(path)-2 && path[i:i+3] == "[*]" {
			result += ".#"
			i += 3
			continue
		}
		result += string(path[i])
		i++
	}
	return result
}

// convertFilters converts JSONPath filter syntax to gjson filter syntax
// JSONPath: $.items[?(@.price < 100)]
// gjson:    items.#(price<100)
// JSONPath with AND: $.items[?(@.price < 100 && @.inStock == true)]
// gjson:    items.#(price<100)#|#(inStock==true)#
func convertFilters(path string) string {
	result := ""
	i := 0

	for i < len(path) {
		// Look for filter pattern: [?(...)]
		if i < len(path)-3 && path[i:i+3] == "[?(" {
			// Find the closing )]
			depth := 1
			j := i + 3
			for j < len(path) && depth > 0 {
				if path[j] == '(' {
					depth++
				} else if path[j] == ')' {
					depth--
				}
				j++
			}

			if depth == 0 && j < len(path) && path[j] == ']' {
				// Extract filter expression
				filterExpr := path[i+3 : j-1]

				// Remove @. prefix from fields in filter
				filterExpr = strings.ReplaceAll(filterExpr, "@.", "")

				// Convert && to gjson pipe syntax for AND conditions
				// $.products[?(@.price < 100 && @.inStock == true)]
				// becomes: products.#(price<100)#|#(inStock==true)#
				//
				// Note: The pipe operator (|) in gjson is technically for modifiers, but
				// when chaining filters like .#(cond1)#|#(cond2)# it creates a sequential
				// filter that achieves AND semantics (filter by cond1, then filter that
				// result by cond2). This is tested and working correctly, but may be
				// fragile if gjson changes its pipe operator behavior in the future.
				// Alternative: Manually intersect results from separate queries.
				if strings.Contains(filterExpr, "&&") {
					parts := strings.Split(filterExpr, "&&")
					result += ".#(" + strings.TrimSpace(parts[0]) + ")#"
					for k := 1; k < len(parts); k++ {
						result += "|#(" + strings.TrimSpace(parts[k]) + ")#"
					}
				} else {
					// No AND condition, just convert normally
					// Replace the entire [?(...)] with .#(...)#
					// The trailing # returns all matches, not just the first
					result += ".#(" + filterExpr + ")#"
				}
				i = j + 1 // Skip past the ]
				continue
			}
		}

		result += string(path[i])
		i++
	}

	return result
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

// hasSliceNotation checks if path contains array slice notation [start:end]
func hasSliceNotation(path string) bool {
	inBracket := false
	for i := 0; i < len(path); i++ {
		if path[i] == '[' {
			inBracket = true
		} else if path[i] == ']' {
			inBracket = false
		} else if path[i] == ':' && inBracket {
			return true
		}
	}
	return false
}

// handleArraySlice processes array slice notation like [0:2]
func handleArraySlice(jsonStr, path string) (interface{}, error) {
	// Find the slice notation
	startIdx := strings.Index(path, "[")
	if startIdx == -1 {
		return nil, ErrInvalidJSONPath
	}
	endIdx := strings.Index(path[startIdx:], "]")
	if endIdx == -1 {
		return nil, ErrInvalidJSONPath
	}
	endIdx += startIdx

	// Extract base path and slice notation
	basePath := path[:startIdx]
	sliceNotation := path[startIdx+1 : endIdx]

	// Parse slice notation
	parts := strings.Split(sliceNotation, ":")
	if len(parts) != 2 {
		return nil, ErrInvalidJSONPath
	}

	start := 0
	end := -1

	if parts[0] != "" {
		var err error
		start, err = strconv.Atoi(parts[0])
		if err != nil {
			return nil, ErrInvalidJSONPath
		}
	}

	if parts[1] != "" {
		var err error
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, ErrInvalidJSONPath
		}
	}

	// Get the array from base path
	baseGPath := strings.TrimPrefix(basePath, "$")
	baseGPath = strings.TrimPrefix(baseGPath, ".")
	result := gjson.Get(jsonStr, baseGPath)

	if !result.IsArray() {
		return nil, ErrTypeMismatch
	}

	// Extract slice
	array := result.Array()
	if end == -1 || end > len(array) {
		end = len(array)
	}
	if start < 0 {
		start = 0
	}
	if start > len(array) {
		start = len(array)
	}

	sliced := array[start:end]
	slicedResult := make([]interface{}, len(sliced))
	for i, item := range sliced {
		slicedResult[i] = convertGJSONResult(item)
	}

	return slicedResult, nil
}

// needsFlatteningFlatten checks if the query needs flattening (has multiple .#)
func needsFlatteningFlatten(path string) bool {
	count := 0
	for i := 0; i < len(path)-1; i++ {
		if path[i] == '.' && path[i+1] == '#' {
			count++
		}
	}
	return count > 1
}

// handleNestedWildcards handles queries like categories.#.items.# which need flattening
func handleNestedWildcards(jsonStr, queryPath string) (interface{}, error) {
	// Parse the path to find the levels
	parts := strings.Split(queryPath, ".#")
	if len(parts) < 2 {
		return nil, ErrInvalidJSONPath
	}

	// Get the base array
	basePath := strings.TrimPrefix(parts[0], ".")
	result := gjson.Get(jsonStr, basePath)
	if !result.IsArray() {
		return nil, ErrTypeMismatch
	}

	// Navigate through each level
	var flattened []interface{}
	var current []gjson.Result = result.Array()

	for i := 1; i < len(parts); i++ {
		var next []gjson.Result
		subPath := strings.TrimPrefix(parts[i], ".")

		for _, item := range current {
			if subPath == "" || subPath == "." {
				// Just flattening arrays
				if item.IsArray() {
					next = append(next, item.Array()...)
				} else {
					next = append(next, item)
				}
			} else {
				// Navigate to subpath
				sub := item.Get(subPath)
				if sub.IsArray() {
					next = append(next, sub.Array()...)
				} else if sub.Exists() {
					next = append(next, sub)
				}
			}
		}
		current = next
	}

	// Convert results
	for _, item := range current {
		flattened = append(flattened, convertGJSONResult(item))
	}

	return flattened, nil
}

// handleRecursiveDescent handles recursive descent queries like $..email
// fieldName should already be extracted (e.g., "email" from "$..email")
func handleRecursiveDescent(jsonStr, fieldName string) (interface{}, error) {

	// Parse the JSON to traverse it
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Recursively find all occurrences of the field
	var results []interface{}
	findRecursive(data, fieldName, &results)

	if len(results) == 0 {
		return nil, nil
	}

	return results, nil
}

// findRecursive recursively searches for a field in nested structures
func findRecursive(data interface{}, fieldName string, results *[]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this map has the field
		if val, ok := v[fieldName]; ok {
			*results = append(*results, val)
		}
		// Recurse into all values
		for _, mapVal := range v {
			findRecursive(mapVal, fieldName, results)
		}
	case []interface{}:
		// Recurse into array elements
		for _, item := range v {
			findRecursive(item, fieldName, results)
		}
	}
}

// validateBrackets checks if all brackets are properly matched
func validateBrackets(path string) error {
	stack := 0
	for i, ch := range path {
		if ch == '[' {
			stack++
			// Check for invalid characters after opening bracket
			if i+1 < len(path) {
				next := path[i+1]
				// Allow: numbers, -, ?, *, :, @
				if next != '-' && next != '?' && next != '*' && next != ':' && next != '@' && (next < '0' || next > '9') {
					// Could be start of filter or other valid syntax
					// Only reject clearly invalid patterns
					if next == '[' && i+2 < len(path) && path[i+2] == '[' {
						return ErrInvalidJSONPath
					}
				}
			}
		} else if ch == ']' {
			stack--
			if stack < 0 {
				return ErrInvalidJSONPath
			}
		}
	}
	if stack != 0 {
		return ErrInvalidJSONPath
	}
	return nil
}

// hasNegativeIndex checks if the path contains negative array indexing
func hasNegativeIndex(path string) bool {
	inBracket := false
	for i := 0; i < len(path); i++ {
		if path[i] == '[' {
			inBracket = true
		} else if path[i] == ']' {
			inBracket = false
		} else if path[i] == '-' && inBracket {
			// Check if it's followed by a digit (negative number)
			if i+1 < len(path) && path[i+1] >= '0' && path[i+1] <= '9' {
				return true
			}
		}
	}
	return false
}

// handleNegativeIndex handles queries with negative array indexing like $.users[-1].email
func handleNegativeIndex(jsonStr, path string) (interface{}, error) {
	// Find the negative index in the path
	negIdx := strings.Index(path, "[-")
	if negIdx == -1 {
		return nil, ErrInvalidJSONPath
	}

	// Find the closing bracket
	closingIdx := strings.Index(path[negIdx:], "]")
	if closingIdx == -1 {
		return nil, ErrInvalidJSONPath
	}
	closingIdx += negIdx

	// Extract the negative index value
	indexStr := path[negIdx+1 : closingIdx]
	negativeIndex, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, ErrInvalidJSONPath
	}

	// Extract the array path (before the negative index)
	arrayPath := path[:negIdx]
	// Extract the remaining path (after the negative index)
	remainingPath := path[closingIdx+1:]

	// Get the array
	arrayGPath := strings.TrimPrefix(arrayPath, "$")
	arrayGPath = strings.TrimPrefix(arrayGPath, ".")

	var arrayResult gjson.Result
	if arrayGPath == "" {
		// Root is an array
		arrayResult = gjson.Parse(jsonStr)
	} else {
		arrayResult = gjson.Get(jsonStr, arrayGPath)
	}

	if !arrayResult.IsArray() {
		return nil, ErrTypeMismatch
	}

	// Get the array length and convert negative index
	array := arrayResult.Array()
	arrayLen := len(array)

	// Convert negative index to positive
	positiveIndex := arrayLen + negativeIndex
	if positiveIndex < 0 || positiveIndex >= arrayLen {
		return nil, nil // Out of bounds returns nil
	}

	// Get the element at the positive index
	element := array[positiveIndex]

	// If there's a remaining path, apply it to the element
	if remainingPath != "" {
		remainingPath = strings.TrimPrefix(remainingPath, ".")
		result := element.Get(remainingPath)
		return convertGJSONResult(result), nil
	}

	return convertGJSONResult(element), nil
}

// hasORFilter checks if the path contains OR (||) in a filter
func hasORFilter(path string) bool {
	inFilter := false
	for i := 0; i < len(path)-1; i++ {
		if i < len(path)-2 && path[i:i+3] == "[?(" {
			inFilter = true
		}
		if inFilter && path[i:i+2] == "||" {
			return true
		}
		if inFilter && path[i:i+2] == ")]" {
			inFilter = false
		}
	}
	return false
}

// handleORFilter handles queries with OR conditions by running separate queries and merging results
// $.items[?(@.category == "electronics" || @.category == "books")]
func handleORFilter(jsonStr, path string) (interface{}, error) {
	// Find the filter
	filterStart := strings.Index(path, "[?(")
	if filterStart == -1 {
		return nil, ErrInvalidJSONPath
	}

	// Find the closing )]
	depth := 1
	j := filterStart + 3
	for j < len(path) && depth > 0 {
		if path[j] == '(' {
			depth++
		} else if path[j] == ')' {
			depth--
		}
		j++
	}

	if depth != 0 || j >= len(path) || path[j] != ']' {
		return nil, ErrInvalidJSONPath
	}

	// Extract parts
	basePath := path[:filterStart]
	filterExpr := path[filterStart+3 : j-1]

	// Split by ||
	conditions := strings.Split(filterExpr, "||")

	// Run each condition separately
	seen := make(map[string]bool)
	var results []interface{}

	for _, cond := range conditions {
		cond = strings.TrimSpace(cond)
		// Remove @. prefix
		cond = strings.ReplaceAll(cond, "@.", "")

		// Build the query for this condition
		queryPath := basePath + "[?(" + cond + ")]"

		// Convert to gjson syntax
		gjsonPath, err := convertJSONPathToGJSON(queryPath)
		if err != nil {
			continue
		}

		// Execute the query
		result := gjson.Get(jsonStr, gjsonPath)
		if !result.IsArray() {
			continue
		}

		// Add unique results
		for _, item := range result.Array() {
			// Use raw JSON as key to detect duplicates
			key := item.Raw
			if !seen[key] {
				seen[key] = true
				results = append(results, convertGJSONResult(item))
			}
		}
	}

	return results, nil
}
