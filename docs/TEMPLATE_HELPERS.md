# Template Helper Functions Reference

GoFlow's template engine provides a comprehensive set of helper functions for data transformation and formatting. All helpers use the `${function(args)}` syntax within template strings.

## Overview

The template renderer supports the following categories of helper functions:

- **Text Helpers**: Transform and manipulate strings
- **Collection Helpers**: Work with arrays and collections
- **Formatting Helpers**: Format numbers and dates
- **Conditional Helpers**: Implement conditional logic

## Text Helpers

### upper(text)

Convert text to uppercase.

**Arguments:**
- `text` (string): The text to convert

**Returns:** String in uppercase

**Example:**
```
${upper(name)}
Input: "john doe"
Output: "JOHN DOE"
```

### lower(text)

Convert text to lowercase.

**Arguments:**
- `text` (string): The text to convert

**Returns:** String in lowercase

**Example:**
```
${lower(title)}
Input: "HELLO WORLD"
Output: "hello world"
```

### capitalize(text)

Capitalize the first letter of the text.

**Arguments:**
- `text` (string): The text to capitalize

**Returns:** String with first character uppercase

**Example:**
```
${capitalize(word)}
Input: "hello"
Output: "Hello"
```

**Note:** Only the first character is uppercased; the rest remains unchanged.

### trim(text)

Remove leading and trailing whitespace.

**Arguments:**
- `text` (string): The text to trim

**Returns:** Trimmed string

**Example:**
```
${trim(text)}
Input: "  hello world  "
Output: "hello world"
```

## Collection Helpers

### length(collection)

Get the length of an array, string, or map.

**Arguments:**
- `collection` (array/string/map): The collection to measure

**Returns:** Integer length

**Errors:**
- Returns `ErrTypeMismatch` if argument is not an array, string, or map

**Example:**
```
${length(items)}
Input: ["apple", "banana", "cherry"]
Output: 3
```

### join(array, separator)

Join array elements into a string with a separator.

**Arguments:**
- `array` (array): The array to join
- `separator` (string): The separator between elements

**Returns:** Joined string

**Errors:**
- Returns `ErrTypeMismatch` if first argument is not an array

**Example:**
```
${join(tags, ', ')}
Input: ["go", "workflow", "orchestration"]
Output: "go, workflow, orchestration"
```

## Formatting Helpers

### formatNumber(value, precision)

Format a number with specified decimal places.

**Arguments:**
- `value` (number): The number to format
- `precision` (integer): Number of decimal places

**Returns:** Formatted number string

**Errors:**
- Returns `ErrTypeMismatch` if value is not numeric
- Returns `ErrTypeMismatch` if precision is not an integer

**Example:**
```
${formatNumber(price, 2)}
Input: 1234.5, 2
Output: "1234.50"

${formatNumber(price, 0)}
Input: 42.7, 0
Output: "43"
```

### formatDate(timestamp, layout)

Format a date/time string using Go's time layout format.

**Arguments:**
- `timestamp` (string): RFC3339-formatted timestamp
- `layout` (string): Go time layout format string

**Returns:** Formatted date string

**Errors:**
- Returns error if timestamp is not valid RFC3339 format
- Returns error if layout is invalid

**Common Layouts:**
- `"2006-01-02"` - Date only (YYYY-MM-DD)
- `"15:04:05"` - Time only (HH:MM:SS)
- `"2006-01-02 15:04:05"` - Date and time
- `"Mon, 02 Jan 2006"` - Day, date, and year

**Example:**
```
${formatDate(timestamp, '2006-01-02')}
Input: "2024-12-01T10:30:00Z"
Output: "2024-12-01"

${formatDate(timestamp, '15:04:05')}
Input: "2024-12-01T10:30:45Z"
Output: "10:30:45"
```

## Conditional Helpers

### if(condition, trueValue, falseValue)

Return one of two values based on a boolean condition.

**Arguments:**
- `condition` (boolean): The condition to evaluate
- `trueValue` (any): Value returned if condition is true
- `falseValue` (any): Value returned if condition is false

**Returns:** String representation of either trueValue or falseValue

**Errors:**
- Returns `ErrTypeMismatch` if condition is not boolean

**Example:**
```
${if(active, 'Active', 'Inactive')}
Input: true, "Active", "Inactive"
Output: "Active"

${if(active, 'Active', 'Inactive')}
Input: false, "Active", "Inactive"
Output: "Inactive"
```

### default(value, fallback)

Return a fallback value if the first value is nil or empty.

**Arguments:**
- `value` (any): The primary value
- `fallback` (any): Fallback value if primary is missing

**Returns:** Either value or fallback

**Example:**
```
${default(role, 'User')}
Input: nil, "User"
Output: "User"

${default(role, 'User')}
Input: "Admin", "User"
Output: "Admin"
```

**Note:** Works with missing variables (lenient mode) - if a variable is undefined, it's treated as nil.

## Usage Patterns

### Basic Template Rendering

```go
renderer := transform.NewTemplateRenderer()
context := map[string]interface{}{
    "name": "john doe",
    "email": "JOHN@EXAMPLE.COM",
}

result, err := renderer.Render(
    context.Background(),
    "User: ${upper(name)}, Email: ${lower(email)}",
    context,
)
// Output: "User: JOHN DOE, Email: john@example.com"
```

### Nested Function Calls

Functions can be nested to combine transformations:

```
${upper(trim(name))}         // Trim then uppercase
${if(active, upper(status), 'INACTIVE')}  // Conditional with transformation
${capitalize(default(title, 'unknown'))}   // Default with capitalization
```

### Field Access with Helpers

Helper functions work with nested field access:

```
${upper(user.name)}           // Access nested field then uppercase
${join(order.items, ', ')}    // Access array field then join
${formatNumber(product.price, 2)}  // Access nested number field
```

### Real-World Example

Email notification template:

```
Dear ${capitalize(user.firstName)},

Your order ${order.id} has been ${lower(order.status)}.
Order Total: $${formatNumber(order.total, 2)}
Items: ${length(order.items)}
Tags: ${join(order.tags, ', ')}

Status: ${if(order.shipped, 'Shipped', 'Processing')}
Estimated Delivery: ${formatDate(order.eta, '2006-01-02')}

Best regards,
${upper(company.name)}
```

## Strict vs Lenient Mode

The template renderer supports two modes for handling missing variables:

### Lenient Mode (Default)

Missing variables return empty string:

```go
renderer := transform.NewTemplateRenderer()
// No strict mode set - defaults to lenient

context := map[string]interface{}{
    "name": "Alice",
}

result, _ := renderer.Render(
    context.Background(),
    "User: ${name}, Role: ${role}",  // role missing
    context,
)
// Output: "User: Alice, Role: "
```

### Strict Mode

Missing variables return an error:

```go
renderer := transform.NewTemplateRenderer()
renderer.SetStrictMode(true)

context := map[string]interface{}{
    "name": "Alice",
}

_, err := renderer.Render(
    context.Background(),
    "User: ${name}, Role: ${role}",  // role missing
    context,
)
// Returns ErrUndefinedVariable error
```

### Default Values

Set a default value for all missing variables in lenient mode:

```go
renderer := transform.NewTemplateRenderer()
renderer.SetDefaultValue("N/A")

context := map[string]interface{}{
    "name": "Alice",
}

result, _ := renderer.Render(
    context.Background(),
    "Name: ${name}, Department: ${dept}",
    context,
)
// Output: "Name: Alice, Department: N/A"
```

## Error Handling

The template renderer can return the following errors:

| Error | Cause | Example |
|-------|-------|---------|
| `ErrNilContext` | Context map is nil | Passing nil as context |
| `ErrInvalidTemplate` | Unclosed braces or empty variables | `"Hello ${name"` or `"Hello ${}"`  |
| `ErrUnknownFunction` | Function doesn't exist | `"${notAFunction(x)}"` |
| `ErrTypeMismatch` | Wrong type for function | `${length(42)}` (number, not array) |
| `ErrUndefinedVariable` | Variable missing in strict mode | Missing variable with strict=true |

## Thread Safety

The `TemplateRenderer` is **not goroutine-safe** for concurrent configuration:

- Configuration methods (`SetStrictMode`, `SetDefaultValue`) must not be called concurrently with `Render`
- Safe usage pattern: Configure the renderer once, then use it from multiple goroutines
- Alternative: Create separate renderer instances per goroutine

```go
// Safe: Configure once, use many times
renderer := transform.NewTemplateRenderer()
renderer.SetStrictMode(true)

// Use renderer concurrently from multiple goroutines
for i := 0; i < 10; i++ {
    go func() {
        result, _ := renderer.Render(ctx, template, context)
    }()
}
```

## Performance Characteristics

Template rendering performance is linear in template size:

- Variable interpolation: O(template length)
- Function evaluation: O(argument count) for most functions
- Collection operations: O(n) for arrays (join, length)
- Date formatting: Constant time (after parsing)

For production use, consider:

- Pre-compiling frequently-used templates
- Caching renderer instances
- Using lenient mode for better error tolerance

## Implementation Details

Helper functions are implemented in `/pkg/transform/template.go`:

- **Text helpers**: Use `strings` package
- **Collection helpers**: Use `reflect` package for generic handling
- **Formatting helpers**: Use `strconv` and `time` packages
- **Conditional helpers**: Simple boolean logic

The engine uses a custom tokenizer (not regex-based) for performance:

- Processes template character-by-character
- Respects escape sequences (`\$`)
- Handles nested function calls
- Supports intelligent argument parsing (respects quoted strings)

## Testing

Comprehensive test coverage in `tests/integration/transform_template_test.go`:

- **TestTemplateVariableInterpolation**: Basic variable substitution
- **TestTemplateHelperFunctions**: All helper function scenarios
- **TestTemplateStrictMode**: Strict vs lenient behavior
- **TestTemplateErrorHandling**: Error cases and edge cases
- **TestTemplateRealWorldScenarios**: Practical use cases
- **TestTemplateConcurrency**: Thread safety verification
- **TestTemplateWithContextDeadline**: Context deadline handling

All tests pass with 100% helper function coverage.
