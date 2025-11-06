# Expression Validation Examples

This document demonstrates the expression validation feature (T120) that validates expressions at workflow load time, not just at runtime.

## Overview

The workflow validator now checks:

1. **ConditionNode expressions**: Validates syntax, checks for undefined variables, detects unsafe operations
2. **TransformNode expressions**: Validates JSONPath syntax, template syntax, and variable references
3. **Security**: Prevents unsafe operations like file access, exec, network calls
4. **Variable references**: Ensures all referenced variables are defined in the workflow

## ConditionNode Validation

### Valid Examples

```go
// Simple comparison
workflow.AddVariable(&Variable{Name: "count", Type: "number"})
node := &ConditionNode{
    ID:        "check_count",
    Condition: "count > 10",
}
// ✓ Valid: count is defined, syntax is correct

// Multiple conditions
workflow.AddVariable(&Variable{Name: "price", Type: "number"})
workflow.AddVariable(&Variable{Name: "quantity", Type: "number"})
node := &ConditionNode{
    ID:        "check_order",
    Condition: "price > 100 && quantity < 50",
}
// ✓ Valid: all variables defined, syntax correct

// String operations
workflow.AddVariable(&Variable{Name: "email", Type: "string"})
node := &ConditionNode{
    ID:        "check_email",
    Condition: "email contains '@' && email contains '.'",
}
// ✓ Valid: email defined, contains operator supported
```

### Invalid Examples

```go
// Undefined variable
workflow.AddVariable(&Variable{Name: "count", Type: "number"})
node := &ConditionNode{
    ID:        "bad_check",
    Condition: "price > 100", // price not defined
}
// ✗ Error: "undefined variable in condition: price"

// Unsafe operation
workflow.AddVariable(&Variable{Name: "file", Type: "string"})
node := &ConditionNode{
    ID:        "bad_check",
    Condition: "os.ReadFile(file)", // forbidden operation
}
// ✗ Error: "unsafe operation detected: os."

// Syntax error
node := &ConditionNode{
    ID:        "bad_check",
    Condition: "count > (10", // unclosed parenthesis
}
// ✗ Error: "invalid condition expression: ..."
```

## TransformNode Validation

### JSONPath Validation

```go
// Valid JSONPath
workflow.AddVariable(&Variable{Name: "data", Type: "object"})
workflow.AddVariable(&Variable{Name: "result", Type: "string"})
node := &TransformNode{
    ID:             "extract_name",
    InputVariable:  "data",
    Expression:     "$.users[0].name",
    OutputVariable: "result",
}
// ✓ Valid: JSONPath syntax correct, variables defined

// Invalid JSONPath
node := &TransformNode{
    ID:             "bad_extract",
    InputVariable:  "data",
    Expression:     "$.users[0.name", // unclosed bracket
    OutputVariable: "result",
}
// ✗ Error: "invalid JSONPath expression: unclosed bracket"
```

### Template Validation

```go
// Valid template
workflow.AddVariable(&Variable{Name: "user", Type: "object"})
workflow.AddVariable(&Variable{Name: "greeting", Type: "string"})
node := &TransformNode{
    ID:             "create_greeting",
    InputVariable:  "user",
    Expression:     "Hello ${user.name}!",
    OutputVariable: "greeting",
}
// ✓ Valid: template syntax correct, user variable defined

// Invalid - undefined variable
workflow.AddVariable(&Variable{Name: "data", Type: "object"})
node := &TransformNode{
    ID:             "bad_template",
    InputVariable:  "data",
    Expression:     "Hello ${userName}!", // userName not defined
    OutputVariable: "greeting",
}
// ✗ Error: "undefined variable in template: userName"

// Invalid - unclosed brace
node := &TransformNode{
    ID:             "bad_template",
    InputVariable:  "user",
    Expression:     "Hello ${user.name", // missing }
    OutputVariable: "greeting",
}
// ✗ Error: "invalid template syntax: unclosed brace at position ..."
```

## Real-World Examples

### E-commerce Order Validation

```go
workflow, _ := NewWorkflow("order-check", "Check order validity")

// Define variables
workflow.AddVariable(&Variable{Name: "price", Type: "number"})
workflow.AddVariable(&Variable{Name: "quantity", Type: "number"})
workflow.AddVariable(&Variable{Name: "inStock", Type: "boolean"})
workflow.AddVariable(&Variable{Name: "discountCode", Type: "string"})

// Condition: Check if order qualifies for express shipping
condition := &ConditionNode{
    ID: "check_express_eligible",
    Condition: "price > 100 && quantity < 50 && inStock == true",
}
// ✓ Validates at workflow load time

// Transform: Apply discount code
transform := &TransformNode{
    ID:             "apply_discount",
    InputVariable:  "discountCode",
    Expression:     "${upper(discountCode)}",
    OutputVariable: "normalizedCode",
}
// ✓ Validates template syntax
```

### User Profile Processing

```go
workflow, _ := NewWorkflow("profile", "Process user profile")

workflow.AddVariable(&Variable{Name: "userData", Type: "object"})
workflow.AddVariable(&Variable{Name: "email", Type: "string"})
workflow.AddVariable(&Variable{Name: "age", Type: "number"})

// Extract email from user data
extractEmail := &TransformNode{
    ID:             "get_email",
    InputVariable:  "userData",
    Expression:     "$.profile.email",
    OutputVariable: "email",
}

// Check email validity
checkEmail := &ConditionNode{
    ID:        "validate_email",
    Condition: "email contains '@' && email contains '.'",
}

// Check age requirement
checkAge := &ConditionNode{
    ID:        "check_age",
    Condition: "(age >= 18 && age <= 65) || verified == true",
}
// ✗ This would fail validation: "verified" is not defined

// Fixed version
workflow.AddVariable(&Variable{Name: "verified", Type: "boolean"})
// ✓ Now validates correctly
```

## Security Examples

### Blocked Operations

```go
// All of these will fail validation at load time:

// File system access
Condition: "os.ReadFile('/etc/passwd')"
// ✗ Error: "unsafe operation detected: os."

// Command execution
Condition: "exec.Command('rm -rf /')"
// ✗ Error: "unsafe operation detected: exec."

// Network access
Condition: "http.Get('http://evil.com')"
// ✗ Error: "unsafe operation detected: http."

// Syscalls
Condition: "syscall.Kill(pid, 9)"
// ✗ Error: "unsafe operation detected: syscall."
```

## Validation Timing

Expression validation happens during:

1. **Workflow.Validate()** - Called explicitly
2. **Workflow loading** - Before execution begins
3. **Parser.Parse()** - When loading from YAML (if Validate is called)

```go
// Example validation flow
yamlContent := []byte(`...`)
workflow, err := Parse(yamlContent)
if err != nil {
    // Parse error
}

// Explicit validation
if err := workflow.Validate(); err != nil {
    // Validation catches expression errors here!
    // - Undefined variables
    // - Invalid syntax
    // - Unsafe operations
}

// Only execute if validation passes
executor := NewExecutor(workflow)
result, err := executor.Execute(ctx, inputs)
```

## Error Messages

The validator provides clear, actionable error messages:

```
node trans1: undefined variable in template: userName
node cond1: invalid condition expression: unsafe operation detected: os.
node trans2: invalid JSONPath expression: unclosed bracket in JSONPath
node cond2: undefined variable in condition: price
```

## Best Practices

1. **Always validate workflows before execution**
   ```go
   if err := workflow.Validate(); err != nil {
       log.Fatalf("Workflow validation failed: %v", err)
   }
   ```

2. **Define all variables upfront**
   ```go
   workflow.AddVariable(&Variable{Name: "input", Type: "object"})
   workflow.AddVariable(&Variable{Name: "result", Type: "any"})
   ```

3. **Test expressions in isolation**
   ```go
   node := &ConditionNode{Condition: expr}
   err := workflow.validateConditionExpression(node)
   ```

4. **Use meaningful variable names**
   ```go
   // Good
   Variable{Name: "customerEmail", Type: "string"}

   // Avoid
   Variable{Name: "x", Type: "any"}
   ```

5. **Document complex expressions**
   ```go
   // Check if user is eligible for premium features
   // - Age between 18-65 OR already verified
   // - Has valid email
   Condition: "(age >= 18 && age <= 65 || verified == true) && email contains '@'"
   ```
