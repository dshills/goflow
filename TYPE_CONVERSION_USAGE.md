# Type Conversion Utilities - Usage Guide

## Overview

The type conversion utilities in `pkg/transform/type_conversion.go` provide safe, comprehensive conversion between Go types for use in GoFlow workflow transformations and conditional logic.

## Quick Reference

### String Conversions
```go
package main

import (
    "github.com/dshills/goflow/pkg/transform"
)

func main() {
    // Convert to string
    s, _ := transform.ToString(42)           // "42"
    s, _ := transform.ToString(3.14)         // "3.14"
    s, _ := transform.ToString(true)         // "true"
    s, _ := transform.ToString(nil)          // ""

    // Parse from string
    i, _ := transform.ParseInt("42")         // 42
    i, _ := transform.ParseInt("0xFF")       // 255 (hex)
    i, _ := transform.ParseInt("0o77")       // 63 (octal)
    i, _ := transform.ParseInt("0b1010")     // 10 (binary)

    f, _ := transform.ParseFloat("3.14")     // 3.14
    f, _ := transform.ParseFloat("1.23e4")   // 12300 (scientific)

    b, _ := transform.ParseBool("true")      // true
    b, _ := transform.ParseBool("yes")       // true
    b, _ := transform.ParseBool("no")        // false
}
```

### Numeric Conversions
```go
// Convert any type to int64
i, err := transform.ToInt(42)              // from int
i, err := transform.ToInt("100")           // from string
i, err := transform.ToInt(3.14)            // from float (truncates)
i, err := transform.ToInt(true)            // from bool (true=1, false=0)

// Convert any type to float64
f, err := transform.ToFloat(42)            // from int
f, err := transform.ToFloat("3.14")        // from string
f, err := transform.ToFloat(true)          // from bool (true=1, false=0)
```

### Collection Conversions
```go
// Convert to array
arr, err := transform.ToArray([]int{1, 2, 3})           // from slice
arr, err := transform.ToArray([3]int{1, 2, 3})          // from array
arr, err := transform.ToArray(`[1,2,3]`)                // from JSON string
arr, err := transform.ToArray("hello")                  // wraps string

// Convert to map
m, err := transform.ToMap(map[string]int{"a": 1})       // from map
m, err := transform.ToMap(`{"a":1,"b":2}`)              // from JSON string
m, err := transform.ToMap(map[string]string{"x": "y"})  // from other map types
```

### Type Checking
```go
// Check type of value
if transform.IsNumeric(value) {
    // Handle numeric types: int, uint, float, json.Number
}

if transform.IsString(value) {
    // Handle string types
}

if transform.IsArray(value) {
    // Handle arrays, slices, and JSON array strings
}

if transform.IsMap(value) {
    // Handle maps and JSON object strings
}

if transform.IsBool(value) {
    // Handle boolean types
}

if transform.IsNil(value) {
    // Handle nil values
}

// Get type name as string
typeName := transform.GetType(value)  // "string", "int", "float64", etc.
```

## Practical Examples

### Example 1: Safe Type Conversion in Conditional
```go
// In a workflow condition that requires numeric comparison
func evaluateCondition(left, right interface{}) (bool, error) {
    // Check if both values are numeric
    if !transform.IsNumeric(left) || !transform.IsNumeric(right) {
        return false, fmt.Errorf("condition requires numeric values")
    }

    // Convert to float for comparison
    leftVal, _ := transform.ToFloat(left)
    rightVal, _ := transform.ToFloat(right)

    return leftVal > rightVal, nil
}
```

### Example 2: JSON Response Handling
```go
// Processing API response with mixed types
func processAPIResponse(data string) error {
    // Parse JSON response
    m, err := transform.ToMap(data)
    if err != nil {
        return fmt.Errorf("invalid JSON response: %w", err)
    }

    // Extract and convert fields
    if age, exists := m["age"]; exists && transform.IsNumeric(age) {
        userAge, _ := transform.ToInt(age)
        // Use userAge...
    }

    // Handle array field
    if items, exists := m["items"]; exists && transform.IsArray(items) {
        itemArray, _ := transform.ToArray(items)
        // Process itemArray...
    }

    return nil
}
```

### Example 3: User Input Validation
```go
// Validating and converting user input
func parseUserInput(inputStr string) (interface{}, error) {
    // Try to determine type and convert

    // Check if it's a boolean
    if b, err := transform.ParseBool(inputStr); err == nil {
        return b, nil
    }

    // Check if it's an integer
    if i, err := transform.ParseInt(inputStr); err == nil {
        return i, nil
    }

    // Check if it's a float
    if f, err := transform.ParseFloat(inputStr); err == nil {
        return f, nil
    }

    // Otherwise treat as string
    return inputStr, nil
}
```

### Example 4: Type-Safe Data Pipeline
```go
// Processing data between workflow nodes
func transformValue(value interface{}, targetType string) (interface{}, error) {
    switch targetType {
    case "int":
        return transform.ToInt(value)
    case "float":
        return transform.ToFloat(value)
    case "string":
        return transform.ToString(value)
    case "bool":
        if b, ok := value.(bool); ok {
            return b, nil
        }
        return transform.ParseBool(transform.ToString(value))
    case "array":
        return transform.ToArray(value)
    case "map", "object":
        return transform.ToMap(value)
    default:
        return nil, fmt.Errorf("unknown target type: %s", targetType)
    }
}
```

### Example 5: Error-Aware Processing
```go
// Robust error handling with type conversion
func processWithFallback(value interface{}, fallback int64) int64 {
    // Try conversion with error handling
    result, err := transform.ToInt(value)
    if err != nil {
        // Log error and use fallback
        log.Printf("conversion failed: %v, using fallback: %d", err, fallback)
        return fallback
    }
    return result
}
```

## Error Handling

All conversion functions return an error that wraps `ErrTypeMismatch`:

```go
value, err := transform.ToInt("not-a-number")
if err != nil {
    // Check error type
    if errors.Is(err, transform.ErrTypeMismatch) {
        // Handle type mismatch specifically
    }
}
```

## Performance Considerations

- **String conversions**: O(1) to O(n) depending on type and size
- **Numeric conversions**: O(1) with overflow detection
- **Array conversions**: O(n) where n is array length
- **Map conversions**: O(n) where n is map size
- **Type checking**: O(1) using type assertion and reflection

### Optimization Tips

1. **Cache type information** if checking same values multiple times
2. **Pre-validate** in hot paths using type checks before conversion
3. **Batch conversions** when possible
4. **Use IsX functions** instead of GetType() comparisons for better performance

## Supported Types

### Input Types
- `nil` (all functions handle)
- Integers: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- Floats: `float32`, `float64`
- Booleans: `bool`
- Strings: `string`
- Bytes: `[]byte`
- JSON: `json.Number`
- Collections: slices, arrays, maps
- JSON strings for collections

### Output Types
- `string`
- `int64`
- `float64`
- `bool`
- `[]interface{}`
- `map[string]interface{}`

## Integration with GoFlow

### In Expression Evaluator
```go
// Type coercion in boolean expressions
evaluator := transform.NewExpressionEvaluator()
// Automatically uses ToInt, ToFloat, etc. for type conversions
```

### In Template Renderer
```go
// Type conversion in template functions
renderer := transform.NewTemplateRenderer()
// Automatically uses ToString, ToInt, etc. in template processing
```

### In Workflow Execution
```go
// Data transformation between nodes
execution.TransformValue = func(val interface{}, targetType string) (interface{}, error) {
    // Uses ToInt, ToFloat, ToArray, ToMap, etc.
}
```

## Testing

All functions are thoroughly tested with 200+ test cases covering:
- Normal cases for each type
- Edge cases (overflow, underflow, max/min values)
- Error cases (invalid conversions, parsing failures)
- JSON parsing and marshaling
- Roundtrip conversions (value → string → value)

Run tests with:
```bash
go test -v -run TestToString ./pkg/transform
go test -v -run "Test(ToString|ParseInt|ParseFloat)" ./pkg/transform
```

## See Also

- `pkg/transform/expression.go` - Expression evaluation using type conversions
- `pkg/transform/template.go` - Template rendering with type conversions
- `pkg/transform/jsonpath.go` - JSONPath queries returning various types
- `pkg/transform/errors.go` - Error definitions including `ErrTypeMismatch`
