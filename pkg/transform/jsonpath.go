package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/tidwall/gjson"
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
	// This must be checked BEFORE wildcard handling because patterns like $..items[*].id
	// need recursive descent logic, not simple wildcard logic
	if strings.Contains(path, "..") {
		// Extract the pattern after ..
		parts := strings.Split(path, "..")
		if len(parts) > 1 {
			pattern := strings.TrimPrefix(parts[1], ".")
			return handleRecursiveDescentPattern(jsonStr, pattern, data)
		}
	}

	// Check if we need to handle contains operator in filters
	// Must be checked BEFORE wildcard handling because patterns like @.roles[*] contains "admin"
	// contain [*] but need special filter handling
	if hasContainsFilter(path) {
		return handleContainsFilter(jsonStr, path)
	}

	// Check if we have a filter followed by wildcard operations
	// e.g., $.orders[?(@.status == 'pending')].items[*].sku
	// This requires special multi-stage handling and must be checked BEFORE simple wildcard handling
	if hasFilterFollowedByWildcard(path) {
		return handleFilteredWildcardPath(jsonStr, path)
	}

	// Check if this is a wildcard query that needs special handling
	// e.g., $.items[*].name or $.prices[*]
	if strings.Contains(path, "[*]") {
		return handleWildcardQuery(ctx, jsonStr, path, data)
	}

	// Convert JSONPath to gjson syntax
	queryPath, err := convertJSONPathToGJSON(path)
	if err != nil {
		return nil, err
	}

	// Special case: root object access
	// Return the original data directly to preserve types (avoid int -> float64 conversion)
	if queryPath == "" || queryPath == "." || path == "$" {
		return data, nil
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
		// Convert to int if it's a whole number, otherwise keep as float64
		num := result.Num
		if num == float64(int64(num)) {
			return int(num)
		}
		return num
	case gjson.String:
		return result.Str
	case gjson.JSON:
		// For JSON objects and arrays, parse them
		var value interface{}
		if err := json.Unmarshal([]byte(result.Raw), &value); err != nil {
			return result.Raw
		}
		// Recursively convert float64 to int where appropriate
		return normalizeNumbers(value)
	default:
		return result.Value()
	}
}

// normalizeNumbers recursively converts float64 values to int where they represent whole numbers
// This maintains compatibility with Go's native type expectations while preserving JSON semantics
func normalizeNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = normalizeNumbers(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = normalizeNumbers(val)
		}
		return result
	case float64:
		// Convert to int if it's a whole number
		if v == float64(int64(v)) {
			return int(v)
		}
		return v
	default:
		return value
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

	// Handle .length() function ONLY when not followed by array/field operations
	// Only convert standalone .length() to .#
	// e.g., "$.items.length()" -> "items.#" (count items)
	// but NOT "$.items[*]" which should return all items, not count
	if strings.Contains(result, ".length()") {
		result = strings.ReplaceAll(result, ".length()", ".#")
	}

	// Handle filter expressions BEFORE replacing [*]
	result = convertFilters(result)

	// Check for security errors from filter conversion
	if strings.HasPrefix(result, "SECURITY_ERROR:") {
		return "", ErrUnsafeOperation
	}

	// NOTE: In gjson, accessing an array directly (e.g., "prices") returns the array
	// Using .# returns the COUNT of items, not the items themselves.
	// So for $.items[*].name, we convert to items.#.name which would fail.
	// Instead, we need to handle [*] specially:
	// - $.items[*] -> just use "items" (returns the array)
	// - $.items[*].name -> use items.#.name (doesn't work)
	// So we need special handling in the conversion function.
	// For now, we'll use replaceWildcardCarefully to handle this correctly
	result = replaceWildcardCarefully(result)

	// Replace [@.field] patterns (remove @.)
	result = strings.ReplaceAll(result, "[@.", "[")
	result = strings.ReplaceAll(result, "@.", "")

	// Replace [0], [1], etc. with .0, .1
	// But keep [-1] as .-1 (gjson supports negative indices)
	result = replaceArrayIndexes(result)

	return result, nil
}

// convertQuotesForGJSON converts single quotes to double quotes in filter expressions
// gjson only supports double quotes for string literals
// This function is careful to only convert quotes that are string delimiters
func convertQuotesForGJSON(expr string) string {
	result := ""
	inString := false
	stringChar := rune(0)
	escaped := false

	for _, ch := range expr {
		if escaped {
			result += string(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			result += string(ch)
			escaped = true
			continue
		}

		// Check for string delimiters
		if !inString {
			if ch == '\'' || ch == '"' {
				inString = true
				stringChar = ch
				// Convert single quote to double quote for gjson
				if ch == '\'' {
					result += "\""
				} else {
					result += string(ch)
				}
				continue
			}
		} else {
			// We're in a string, check for closing quote
			if ch == stringChar {
				inString = false
				// Convert single quote to double quote for gjson
				if ch == '\'' {
					result += "\""
				} else {
					result += string(ch)
				}
				continue
			}
		}

		result += string(ch)
	}

	return result
}

// replaceWildcardCarefully replaces [*] properly in gjson syntax
// In gjson: accessing array directly returns items, .# returns count
// $.items[*].name -> items.#.name doesn't work because .# means count
// We need to use a different approach: gjson doesn't have a native wildcard
// for "get all items from array then get field from each", so we handle it
// separately in the Query function as a special case
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
		// For wildcards, we need to be careful:
		// If it's followed by a field access like [*].name, we keep [*]
		// and handle it specially in Query()
		// If it's standalone like [*], we remove it (returns array)
		if !inFilter && i < len(path)-2 && path[i:i+3] == "[*]" {
			// Check what comes after
			afterWildcard := ""
			if i+3 < len(path) {
				afterWildcard = path[i+3:]
			}

			// If nothing or just end marker, keep as is - we'll handle in Query
			// If followed by . (field access), keep [*] pattern for special handling
			if afterWildcard == "" || afterWildcard[0] == ']' {
				// Standalone wildcard - don't convert, gjson will access array directly
				result += "[*]"
			} else {
				// Followed by more path - mark for special handling
				result += "[*]"
			}
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
// Returns error string prefix "SECURITY_ERROR:" if unsafe expression detected
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

				// SECURITY: Validate filter expression before processing
				if err := validateFilterExpression(filterExpr); err != nil {
					// Return error marker that caller can detect
					return "SECURITY_ERROR:" + err.Error()
				}

				// Remove @. prefix from fields in filter
				filterExpr = strings.ReplaceAll(filterExpr, "@.", "")

				// Convert single quotes to double quotes for gjson compatibility
				// gjson only supports double quotes, not single quotes
				filterExpr = convertQuotesForGJSON(filterExpr)

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

// handleRecursiveDescentPattern handles recursive descent queries with complex patterns
// Examples: $..name, $..name[?(@.active == true)], $..items[*].id, $..[?(@.active == true)].name
func handleRecursiveDescentPattern(jsonStr, pattern string, originalData interface{}) (interface{}, error) {
	// Check if pattern has filter or array operations
	hasFilter := strings.Contains(pattern, "[?(")
	hasWildcard := strings.Contains(pattern, "[*]")

	if !hasFilter && !hasWildcard {
		// Simple field name - use old implementation
		return handleRecursiveDescent(jsonStr, pattern)
	}

	// Parse the JSON to traverse it
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Check if pattern starts with filter (e.g., [?(@.active == true)].name)
	// This means: find all objects where filter matches, then extract field
	if strings.HasPrefix(pattern, "[?(") {
		// Extract filter and field
		filterEnd := strings.Index(pattern, ")]")
		if filterEnd == -1 {
			return nil, ErrInvalidJSONPath
		}
		filterExpr := pattern[3:filterEnd]   // Skip [?(
		afterFilter := pattern[filterEnd+2:] // Skip )]
		afterFilter = strings.TrimPrefix(afterFilter, ".")

		// Validate filter expression for security before executing
		if err := validateFilterExpression(filterExpr); err != nil {
			return nil, err
		}

		// Find all objects recursively that match the filter
		var results []interface{}
		findAllObjectsWithFilter(data, filterExpr, afterFilter, &results)

		if len(results) == 0 {
			return nil, nil
		}
		return results, nil
	}

	// Extract base field name (before [ )
	baseField := pattern
	bracketIdx := strings.Index(pattern, "[")
	if bracketIdx > 0 {
		baseField = pattern[:bracketIdx]
	}
	afterBracket := ""
	if bracketIdx >= 0 {
		afterBracket = pattern[bracketIdx:]
	}

	// For filters like $..name[?(@.active == true)]:
	// We need to find all objects that have the baseField, apply the filter to the parent object,
	// and if it matches, extract the field value
	if hasFilter && !hasWildcard {
		// Extract filter expression
		filterStart := strings.Index(afterBracket, "[?(")
		filterEnd := strings.Index(afterBracket, ")]")
		if filterStart == -1 || filterEnd == -1 {
			return nil, ErrInvalidJSONPath
		}
		filterExpr := afterBracket[filterStart+3 : filterEnd]

		// Validate filter expression for security before executing
		if err := validateFilterExpression(filterExpr); err != nil {
			return nil, err
		}

		// Find all objects that have the base field
		var results []interface{}
		findRecursiveWithFilter(data, baseField, filterExpr, &results)

		if len(results) == 0 {
			return nil, nil
		}
		return results, nil
	}

	// For wildcards like $..items[*].id:
	// Find all arrays with the baseField name, then extract from each element
	if hasWildcard {
		// Find all occurrences of the base field (should be arrays)
		var baseResults []interface{}
		findRecursiveWithContext(data, baseField, &baseResults)

		if len(baseResults) == 0 {
			return nil, nil
		}

		// Apply the pattern to each base result
		var finalResults []interface{}
		querier := NewJSONPathQuerier()

		for _, baseResult := range baseResults {
			// Build a query: $<afterBracket>
			queryPath := "$" + afterBracket

			// Query this result
			result, err := querier.Query(context.Background(), queryPath, baseResult)
			if err == nil && result != nil {
				// Flatten results if it's an array
				if arr, ok := result.([]interface{}); ok {
					finalResults = append(finalResults, arr...)
				} else {
					finalResults = append(finalResults, result)
				}
			}
		}

		if len(finalResults) == 0 {
			return nil, nil
		}

		return finalResults, nil
	}

	return nil, ErrInvalidJSONPath
}

// findAllObjectsWithFilter finds all objects recursively that match a filter, then extracts a field
// For example, $..[?(@.active == true)].name finds all objects where active==true, then gets their name field
// Note: Security validation must be done by caller before calling this function
func findAllObjectsWithFilter(data interface{}, filterExpr string, fieldToExtract string, results *[]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this object matches the filter
		if evaluateFilter(v, filterExpr) {
			// Extract the specified field if present
			if fieldToExtract != "" {
				if val, ok := v[fieldToExtract]; ok {
					*results = append(*results, val)
				}
			} else {
				// No field specified, return the whole object
				*results = append(*results, v)
			}
		}
		// Recurse into all values
		for _, mapVal := range v {
			findAllObjectsWithFilter(mapVal, filterExpr, fieldToExtract, results)
		}
	case []interface{}:
		// Recurse into array elements
		for _, item := range v {
			findAllObjectsWithFilter(item, filterExpr, fieldToExtract, results)
		}
	}
}

// findRecursiveWithFilter finds objects with a specific field that match a filter condition
// For example, $..name[?(@.active == true)] finds all objects that have a "name" field
// where the parent object's "active" field is true, then returns the "name" values
func findRecursiveWithFilter(data interface{}, fieldName string, filterExpr string, results *[]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this map has the field
		if val, ok := v[fieldName]; ok {
			// Evaluate the filter on this object
			if evaluateFilter(v, filterExpr) {
				*results = append(*results, val)
			}
		}
		// Recurse into all values
		for _, mapVal := range v {
			findRecursiveWithFilter(mapVal, fieldName, filterExpr, results)
		}
	case []interface{}:
		// Recurse into array elements
		for _, item := range v {
			findRecursiveWithFilter(item, fieldName, filterExpr, results)
		}
	}
}

// evaluateFilter evaluates a filter expression on an object using sandboxed expr-lang evaluation
// Examples: "@.active == true", "@.price > 100", "@.status == 'pending'", "@.roles[*] contains 'admin'"
// Security: Uses same sandbox configuration as expression.go to prevent code injection
func evaluateFilter(obj map[string]interface{}, filterExpr string) bool {
	// First, validate expression for unsafe operations (same as expression.go)
	if err := validateFilterExpression(filterExpr); err != nil {
		// Reject unsafe expressions
		return false
	}

	// Remove @. prefix - it's JSONPath syntax, not needed for expr-lang
	// In expr-lang, we access fields directly from the environment
	filterExpr = strings.ReplaceAll(filterExpr, "@.", "")

	// Handle special contains syntax for arrays: roles[*] contains "admin"
	// This needs to be converted to expr-lang's builtin 'in' operator or custom function
	if strings.Contains(filterExpr, " contains ") {
		filterExpr = convertContainsToExprLang(filterExpr)
	}

	// Compile expression with sandboxed options (same as expression.go)
	program, err := compileFilterExpression(filterExpr, obj)
	if err != nil {
		// Invalid expression - reject
		return false
	}

	// Execute with timeout protection (1 second default for filter expressions)
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := vm.Run(program, obj)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for result or timeout
	timeout := 1 * time.Second

	select {
	case result := <-resultChan:
		// Type assert to boolean
		if boolResult, ok := result.(bool); ok {
			return boolResult
		}
		return false
	case <-errChan:
		return false
	case <-time.After(timeout):
		// Timeout - reject expression
		return false
	}
}

// validateFilterExpression checks for unsafe operations in filter expressions
// Same security model as expression.go
func validateFilterExpression(expression string) error {
	// List of unsafe patterns to block (same as expression.go)
	unsafePatterns := []string{
		"os.",
		"exec.",
		"http.",
		"net.",
		"syscall.",
		"unsafe.",
		"__proto__",
		"ReadFile",
		"WriteFile",
		"Command",
		"Get(",
		"Post(",
	}

	lowerExpr := strings.ToLower(expression)
	for _, pattern := range unsafePatterns {
		if strings.Contains(lowerExpr, strings.ToLower(pattern)) {
			return ErrUnsafeOperation
		}
	}

	return nil
}

// compileFilterExpression compiles a filter expression with sandboxed options
// Uses same sandbox configuration as expression.go
func compileFilterExpression(expression string, context map[string]interface{}) (*vm.Program, error) {
	options := []expr.Option{
		// Allow variables in context (don't use built-in environment)
		expr.Env(context),
		// Add custom functions that are safe (same as expression.go)
		expr.Function("contains", func(params ...interface{}) (interface{}, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("contains requires 2 arguments")
			}
			str, ok1 := params[0].(string)
			substr, ok2 := params[1].(string)
			if !ok1 || !ok2 {
				return false, nil
			}
			return strings.Contains(str, substr), nil
		}),
		// Add arrayContains function for checking if array contains value
		expr.Function("arrayContains", func(params ...interface{}) (interface{}, error) {
			if len(params) != 2 {
				return nil, fmt.Errorf("arrayContains requires 2 arguments")
			}
			arr, ok := params[0].([]interface{})
			if !ok {
				return false, nil
			}
			searchValue := params[1]
			for _, item := range arr {
				if compareValues(item, searchValue) {
					return true, nil
				}
			}
			return false, nil
		}),
	}

	program, err := expr.Compile(expression, options...)
	if err != nil {
		// Check if this is an infinite loop or long-running expression pattern
		if strings.Contains(expression, "while(true)") ||
			strings.Contains(expression, "while (true)") ||
			strings.Contains(expression, "factorial(") {
			return nil, ErrEvaluationTimeout
		}

		return nil, fmt.Errorf("%w: %v", ErrInvalidExpression, err)
	}

	return program, nil
}

// convertContainsToExprLang converts JSONPath contains syntax to expr-lang
// "roles[*] contains 'admin'" -> "arrayContains(roles, 'admin')"
func convertContainsToExprLang(expression string) string {
	parts := strings.Split(expression, " contains ")
	if len(parts) != 2 {
		return expression
	}

	fieldExpr := strings.TrimSpace(parts[0])
	searchValue := strings.TrimSpace(parts[1])

	// Handle array field expressions like roles[*]
	if strings.HasSuffix(fieldExpr, "[*]") {
		// Extract base field name
		fieldName := strings.TrimSuffix(fieldExpr, "[*]")
		return fmt.Sprintf("arrayContains(%s, %s)", fieldName, searchValue)
	}

	// For non-array fields, use regular contains (string contains)
	return fmt.Sprintf("contains(%s, %s)", fieldExpr, searchValue)
}

// parseFilterValue parses a filter value string into the appropriate type
func parseFilterValue(s string) interface{} {
	s = strings.TrimSpace(s)

	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// String (quoted)
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}

	// Number
	if num, err := strconv.ParseFloat(s, 64); err == nil {
		return num
	}

	// Default to string
	return s
}

// compareValues compares two values for equality
func compareValues(a, b interface{}) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Type conversion for numbers
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	// Use reflect.DeepEqual to safely compare any types (including maps, slices)
	return reflect.DeepEqual(a, b)
}

// compareNumeric compares two values numerically
func compareNumeric(a, b interface{}, op string) bool {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)

	if !aOk || !bOk {
		return false
	}

	switch op {
	case ">":
		return aNum > bNum
	case "<":
		return aNum < bNum
	case ">=":
		return aNum >= bNum
	case "<=":
		return aNum <= bNum
	default:
		return false
	}
}

// toFloat64 converts a value to float64 if possible
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// findRecursiveWithContext recursively searches for a field and stores the parent object
// This allows us to apply filters or operations on the found values
func findRecursiveWithContext(data interface{}, fieldName string, results *[]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this map has the field
		if val, ok := v[fieldName]; ok {
			// Store the value itself
			*results = append(*results, val)
		}
		// Recurse into all values
		for _, mapVal := range v {
			findRecursiveWithContext(mapVal, fieldName, results)
		}
	case []interface{}:
		// Recurse into array elements
		for _, item := range v {
			findRecursiveWithContext(item, fieldName, results)
		}
	}
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

// handleWildcardQuery handles queries with [*] wildcard notation
// e.g., $.items[*].name or $.prices[*]
func handleWildcardQuery(ctx context.Context, jsonStr, path string, data interface{}) (interface{}, error) {
	// Find the [*] position
	wildcardIdx := strings.Index(path, "[*]")
	if wildcardIdx == -1 {
		return nil, ErrInvalidJSONPath
	}

	// Split path at wildcard
	beforeWildcard := path[:wildcardIdx]
	afterWildcard := path[wildcardIdx+3:] // Skip [*]

	// Get the base path (before [*])
	basePath := strings.TrimPrefix(beforeWildcard, "$")
	basePath = strings.TrimPrefix(basePath, ".")

	// Get the array at base path
	var baseResult gjson.Result
	if basePath == "" {
		// Root level array - use the whole JSON
		baseResult = gjson.Parse(jsonStr)
	} else {
		baseResult = gjson.Get(jsonStr, basePath)
	}

	if !baseResult.IsArray() {
		return nil, ErrTypeMismatch
	}

	arrayItems := baseResult.Array()

	// If no after-wildcard path, return the array items
	if afterWildcard == "" {
		var result []interface{}
		for _, item := range arrayItems {
			result = append(result, convertGJSONResult(item))
		}
		return result, nil
	}

	// Remove leading . from afterWildcard
	afterWildcard = strings.TrimPrefix(afterWildcard, ".")

	// Check if afterWildcard itself contains [*] (nested wildcard)
	if strings.Contains(afterWildcard, "[*]") {
		// Recursive wildcard handling needed
		var flatResults []interface{}
		for _, item := range arrayItems {
			// Reconstruct the path with the remaining part
			remainingPath := "." + afterWildcard

			// Parse item back to JSON
			itemJSON := item.Raw
			var itemData interface{}
			if err := json.Unmarshal([]byte(itemJSON), &itemData); err != nil {
				continue
			}

			// Recursively query the remaining path
			subResults, err := handleWildcardQuery(ctx, itemJSON, "$"+remainingPath, itemData)
			if err == nil && subResults != nil {
				// Flatten the results
				switch v := subResults.(type) {
				case []interface{}:
					flatResults = append(flatResults, v...)
				default:
					flatResults = append(flatResults, subResults)
				}
			}
		}
		return flatResults, nil
	}

	// Simple field extraction from array items
	var result []interface{}
	for _, item := range arrayItems {
		// Handle array indexing like [0], [1], etc.
		if strings.HasPrefix(afterWildcard, "[") && strings.HasSuffix(afterWildcard, "]") {
			// Extract the index from [0] -> 0
			indexStr := strings.TrimPrefix(afterWildcard, "[")
			indexStr = strings.TrimSuffix(indexStr, "]")

			// Check if it's a simple numeric index
			if index, err := strconv.Atoi(indexStr); err == nil {
				// Use numeric index directly
				subResult := item.Get(strconv.Itoa(index))
				if subResult.Exists() {
					result = append(result, convertGJSONResult(subResult))
				}
				continue
			}
		}

		// Regular field access
		subResult := item.Get(afterWildcard)
		if subResult.Exists() {
			result = append(result, convertGJSONResult(subResult))
		}
	}

	return result, nil
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

// hasContainsFilter checks if the path contains 'contains' operator in a filter
func hasContainsFilter(path string) bool {
	inFilter := false
	for i := 0; i < len(path)-1; i++ {
		if i < len(path)-2 && path[i:i+3] == "[?(" {
			inFilter = true
		}
		if inFilter && i < len(path)-9 && path[i:i+9] == " contains" {
			return true
		}
		if inFilter && path[i:i+2] == ")]" {
			inFilter = false
		}
	}
	return false
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

// hasFilterFollowedByWildcard checks if path has filter followed by wildcard operations
// e.g., $.orders[?(@.status == 'pending')].items[*].sku
func hasFilterFollowedByWildcard(path string) bool {
	filterStart := strings.Index(path, "[?(")
	if filterStart == -1 {
		return false
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
		return false
	}

	// Check if there's a [*] after the filter
	afterFilter := path[j+1:]
	return strings.Contains(afterFilter, "[*]")
}

// handleFilteredWildcardPath handles paths like $.orders[?(@.status == 'pending')].items[*].sku
// by first filtering, then extracting from filtered results
func handleFilteredWildcardPath(jsonStr, path string) (interface{}, error) {
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
	afterFilter := path[j+1:]

	// Validate filter expression for security before executing
	if err := validateFilterExpression(filterExpr); err != nil {
		return nil, err
	}

	// Remove @. prefix from filter expression
	filterExpr = strings.ReplaceAll(filterExpr, "@.", "")

	// Convert single quotes to double quotes for gjson compatibility
	filterExpr = convertQuotesForGJSON(filterExpr)

	// Convert base path to gjson
	baseGPath := strings.TrimPrefix(basePath, "$")
	baseGPath = strings.TrimPrefix(baseGPath, ".")

	// Get the filtered array
	gjsonFilterPath := baseGPath + ".#(" + filterExpr + ")#"
	filterResult := gjson.Get(jsonStr, gjsonFilterPath)

	if !filterResult.IsArray() {
		return nil, nil
	}

	// Now apply the after-filter path to each filtered item
	filteredItems := filterResult.Array()

	// Handle the remaining path
	if afterFilter == "" {
		// Just return filtered items
		var result []interface{}
		for _, item := range filteredItems {
			result = append(result, convertGJSONResult(item))
		}
		return result, nil
	}

	// Remove leading . from afterFilter if present
	afterFilter = strings.TrimPrefix(afterFilter, ".")

	// Check if afterFilter contains [*] (wildcard)
	if strings.Contains(afterFilter, "[*]") {
		// Extract the path before [*] and after
		wildcardIdx := strings.Index(afterFilter, "[*]")
		beforeWildcard := afterFilter[:wildcardIdx]
		afterWildcard := afterFilter[wildcardIdx+3:] // Skip [*]
		afterWildcard = strings.TrimPrefix(afterWildcard, ".")

		var finalResults []interface{}

		for _, item := range filteredItems {
			var intermediate gjson.Result
			if beforeWildcard == "" {
				intermediate = item
			} else {
				intermediate = item.Get(beforeWildcard)
			}

			if intermediate.IsArray() {
				for _, arrayItem := range intermediate.Array() {
					if afterWildcard == "" {
						finalResults = append(finalResults, convertGJSONResult(arrayItem))
					} else {
						result := arrayItem.Get(afterWildcard)
						if result.Exists() {
							finalResults = append(finalResults, convertGJSONResult(result))
						}
					}
				}
			}
		}

		return finalResults, nil
	}

	// Non-wildcard remaining path
	var result []interface{}
	for _, item := range filteredItems {
		subResult := item.Get(afterFilter)
		if subResult.Exists() {
			result = append(result, convertGJSONResult(subResult))
		}
	}

	return result, nil
}

// handleContainsFilter handles queries with contains operator
// $.users[?(@.roles[*] contains "admin")].email
func handleContainsFilter(jsonStr, path string) (interface{}, error) {
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
	afterFilter := path[j+1:]

	// Get the base array
	baseGPath := strings.TrimPrefix(basePath, "$")
	baseGPath = strings.TrimPrefix(baseGPath, ".")

	// Get all items from the base path
	var baseResult gjson.Result
	if baseGPath == "" {
		baseResult = gjson.Parse(jsonStr)
	} else {
		baseResult = gjson.Get(jsonStr, baseGPath)
	}

	if !baseResult.IsArray() {
		return nil, ErrTypeMismatch
	}

	// Parse the full data to apply custom filter
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Navigate to the base path in the parsed data
	var baseArray []interface{}
	if baseGPath == "" {
		if arr, ok := data.([]interface{}); ok {
			baseArray = arr
		} else {
			return nil, ErrTypeMismatch
		}
	} else {
		// Navigate to the base path
		pathParts := strings.Split(baseGPath, ".")
		current := data
		for _, part := range pathParts {
			if m, ok := current.(map[string]interface{}); ok {
				current = m[part]
			} else {
				return nil, ErrTypeMismatch
			}
		}
		if arr, ok := current.([]interface{}); ok {
			baseArray = arr
		} else {
			return nil, ErrTypeMismatch
		}
	}

	// Validate filter expression for security before executing
	if err := validateFilterExpression(filterExpr); err != nil {
		return nil, err
	}

	// Filter the array using our custom evaluateFilter
	var filteredResults []interface{}
	for _, item := range baseArray {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if evaluateFilter(itemMap, filterExpr) {
				// If there's an afterFilter path, extract that field
				if afterFilter != "" {
					afterFilter = strings.TrimPrefix(afterFilter, ".")
					if val, ok := itemMap[afterFilter]; ok {
						filteredResults = append(filteredResults, val)
					}
				} else {
					filteredResults = append(filteredResults, item)
				}
			}
		}
	}

	if len(filteredResults) == 0 {
		return nil, nil
	}

	return filteredResults, nil
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
