# Template Helper Functions - Quick Reference

## All Available Helpers (12 total)

### Text Transformation
| Function | Syntax | Example | Output |
|----------|--------|---------|--------|
| upper | `${upper(text)}` | `${upper("hello")}` | `"HELLO"` |
| lower | `${lower(text)}` | `${lower("HELLO")}` | `"hello"` |
| capitalize | `${capitalize(text)}` | `${capitalize("hello")}` | `"Hello"` |
| trim | `${trim(text)}` | `${trim("  hello  ")}` | `"hello"` |

### Collections
| Function | Syntax | Example | Output |
|----------|--------|---------|--------|
| length | `${length(arr)}` | `${length([1,2,3])}` | `3` |
| join | `${join(arr, sep)}` | `${join([1,2,3], ",")}` | `"1,2,3"` |

### Formatting
| Function | Syntax | Example | Output |
|----------|--------|---------|--------|
| formatNumber | `${formatNumber(num, decimals)}` | `${formatNumber(99.5, 2)}` | `"99.50"` |
| formatDate | `${formatDate(timestamp, layout)}` | `${formatDate("2024-01-15T10:30:00Z", "2006-01-02")}` | `"2024-01-15"` |

### Conditional
| Function | Syntax | Example | Output |
|----------|--------|---------|--------|
| if | `${if(cond, true, false)}` | `${if(true, "yes", "no")}` | `"yes"` |
| default | `${default(val, fallback)}` | `${default(null, "N/A")}` | `"N/A"` |

---

## Common Patterns

### Greeting Message
```
Hello ${capitalize(first_name)} ${upper(last_name)}!
```

### Price Display
```
Total: $${formatNumber(price, 2)}
```

### Date Display
```
Order Date: ${formatDate(created_at, "2006-01-02")}
Time: ${formatDate(created_at, "15:04:05")}
```

### List Formatting
```
Tags: ${join(tags, ", ")}
```

### Conditional Status
```
Status: ${if(is_active, "Active", "Inactive")}
```

### Fallback Values
```
Department: ${default(dept_name, "Unassigned")}
```

### Combined
```
Order #${order_id}: ${join(items, ", ")} - $${formatNumber(total, 2)} - ${if(shipped, "Shipped", "Processing")}
```

---

## Function Nesting

Functions can be nested for complex transformations:

```
${upper(trim(name))}                          # Trim then uppercase
${capitalize(lower(title))}                   # Lowercase then capitalize
${join(tags, " | ")}                          # Join with separator
${if(active, upper(status), "INACTIVE")}     # Conditional transformation
${default(upper(trim(category)), "OTHER")}   # Default with transformation
```

---

## Field Access

Access nested objects with dot notation:

```
${user.name}                    # Simple field
${user.address.city}            # Nested fields
${user.settings.theme.color}    # Deep nesting
```

Can combine with functions:

```
${upper(user.name)}                          # Transform field
${capitalize(user.first_name)}               # Transform nested field
${join(product.tags, ", ")}                  # Array field
${formatNumber(order.total, 2)}              # Numeric field
```

---

## Usage in Code

```go
renderer := transform.NewTemplateRenderer()

// Simple template
result, err := renderer.Render(
    context.Background(),
    "Hello ${upper(name)}!",
    map[string]interface{}{"name": "world"},
)
// Output: "Hello WORLD!"

// Configuration options
renderer.SetStrictMode(true)        // Fail on missing variables
renderer.SetDefaultValue("N/A")     // Fallback for missing values

// Complex template
template := `Order #${id}:
Items: ${join(items, ", ")}
Total: $${formatNumber(total, 2)}
Status: ${if(shipped, "Shipped", "Processing")}
Eta: ${formatDate(eta, "2006-01-02")}`

result, err := renderer.Render(context.Background(), template, orderData)
```

---

## Error Handling

| Error | Cause | Example |
|-------|-------|---------|
| `ErrNilContext` | Nil context passed | `Render(ctx, template, nil)` |
| `ErrInvalidTemplate` | Bad syntax | `"${unclosed"` or `"${}"` |
| `ErrUnknownFunction` | Unknown function | `"${unknown()}"` |
| `ErrTypeMismatch` | Wrong type for function | `${length(42)}` (number not array) |
| `ErrUndefinedVariable` | Missing var in strict mode | `${missing}` (with SetStrictMode(true)) |

---

## Escaping

Escape literal dollar signs with backslash:

```
\${literal}    → ${literal}
Price: \$99    → Price: $99
```

---

## Performance Tips

1. **Reuse renderers** - Create once, use many times
2. **Lenient mode** - Faster than strict mode for missing variables
3. **Pre-compile** - Cache frequently-used templates
4. **Goroutine-safe** - Render() is safe for concurrent calls
5. **Config once** - Set options before use, don't change during rendering

---

## Strict vs Lenient Mode

### Lenient Mode (Default)
```go
renderer := transform.NewTemplateRenderer()
// Missing variables return empty string or default value

result, err := renderer.Render(ctx, "${name}", map[string]interface{}{})
// result = "", err = nil
```

### Strict Mode
```go
renderer := transform.NewTemplateRenderer()
renderer.SetStrictMode(true)

_, err := renderer.Render(ctx, "${name}", map[string]interface{}{})
// err = ErrUndefinedVariable
```

### Default Values
```go
renderer := transform.NewTemplateRenderer()
renderer.SetDefaultValue("N/A")

result, _ := renderer.Render(ctx, "${missing}", map[string]interface{}{})
// result = "N/A"
```

---

## Real Examples

### Email Template
```
Dear ${capitalize(user.firstName)},

Your order #${order.id} has been placed!
Items: ${length(order.items)}
Total: $${formatNumber(order.total, 2)}
Estimated Delivery: ${formatDate(order.eta, "Monday, 2 Jan 2006")}

Status: ${if(order.confirmed, "Confirmed", "Pending")}

Thank you,
${upper(company.name)}
```

### API URL
```
https://api.example.com/v${version}/users/${user_id}/orders/${order_id}?limit=${limit}
```

### Log Message
```
[${upper(level)}] ${formatDate(timestamp, "2006-01-02 15:04:05")} - ${service}: ${message} (${duration}ms)
```

### CSV Export
```
${id},${capitalize(name)},${lower(email)},${formatNumber(salary, 2)},${if(active, "Yes", "No")}
```

---

## See Also

- **Full Reference:** `/docs/TEMPLATE_HELPERS.md`
- **Implementation:** `/pkg/transform/template.go`
- **Tests:** `/tests/integration/transform_template_test.go`
- **Package Tests:** `/pkg/transform/template_test.go`
