# Type Assertion Refactoring Checklist (Phase 12: T077-T082)

**Goal**: Replace type assertions with direct typing where types are known at compile time

**Completion Date**: 2025-11-12
**Status**: Analysis Complete

---

## Analysis Summary

### Type Assertion Categories

1. **Error Type Checking** - Can use `errors.As()` or `errors.Is()` instead of type assertions
2. **Interface{} to Concrete Types** - Runtime checks needed (keep as-is, but add safety)
3. **Type Switches** - Already idiomatic, but can add default cases
4. **Boolean/String Conversions** - Runtime checks needed (keep as-is)
5. **Map Access Patterns** - Can use generics for type-safe operations

### Files with Type Assertions

#### pkg/execution/

1. **error_integration_example.go:75**
   - Pattern: `if e, ok := baseErr.(*execution.ExecutionError); ok`
   - Status: ✅ KEEP - This is an example file demonstrating error handling patterns
   - Action: None (example code)

2. **retry.go:196-209**
   - Pattern: Multiple type assertions for error classification
   ```go
   if execErr, ok := err.(*execution.ExecutionError); ok {
       return execErr.Type
   }
   if _, ok := err.(*MCPToolError); ok {
       return execution.ErrorTypeConnection
   }
   ```
   - Status: 🔄 REFACTOR - Use `errors.As()` for safer error type checking
   - Action: Replace with `errors.As()` pattern
   - Reason: More idiomatic and handles wrapped errors correctly

3. **loop.go:170**
   - Pattern: `broken, ok := result.(bool)`
   - Status: ✅ KEEP - Runtime type check necessary for dynamic evaluation
   - Action: Add test coverage for non-bool results
   - Reason: Result type is determined at runtime from expression evaluation

4. **loop.go:205**
   - Pattern: `switch v := collection.(type)`
   - Status: ✅ KEEP - Type switch is idiomatic
   - Action: Ensure all expected types are covered in switch
   - Reason: Handles multiple collection types at runtime

5. **node_executor.go:235**
   - Pattern: `if m, ok := current.(map[string]interface{}); ok`
   - Status: ✅ KEEP - Runtime type check for nested field access
   - Action: Add test coverage for non-map types
   - Reason: Variable structure determined at runtime

6. **audit.go:594**
   - Pattern: `switch v := value.(type)`
   - Status: ✅ KEEP - Type switch for value serialization
   - Action: Ensure default case exists
   - Reason: Handles multiple value types for audit logging

7. **snapshot_test.go:167,176**
   - Pattern: Type assertions in test code
   - Status: ✅ KEEP - Test code validating runtime behavior
   - Action: None (test code)

8. **events_test.go:132,196,240,274**
   - Pattern: `monitor := mon.(*monitor)` in tests
   - Status: 🔄 IMPROVE - Test code can be more explicit
   - Action: Use type-safe constructor or test helpers
   - Reason: Tests should be explicit about types expected

#### pkg/workflow/

1. **workflow.go:325,444,639**
   - Pattern: Multiple `switch n := node.(type)` for node type handling
   - Status: ✅ KEEP - Type switches are idiomatic for polymorphic behavior
   - Action: Ensure all node types covered, add default case
   - Reason: Handles different node types polymorphically

2. **workflow.go:627**
   - Pattern: `if loopNode, ok := node.(*LoopNode); ok`
   - Status: ✅ KEEP - Specific type check for loop validation
   - Action: Consider visitor pattern for node operations
   - Reason: Specific behavior needed for loop nodes

3. **parser.go:378**
   - Pattern: `switch n := node.(type)`
   - Status: ✅ KEEP - Type switch for parsing different node types
   - Action: Ensure all node types covered
   - Reason: Parser needs to handle different node types

4. **variable.go:81,86,93,97,101**
   - Pattern: Multiple type assertions for variable validation
   ```go
   if _, ok := v.DefaultValue.(string); !ok {
       return fmt.Errorf("...")
   }
   ```
   - Status: 🔄 REFACTOR - Use type-safe validation helpers
   - Action: Create generic `validateType[T any](value interface{}) error` helper
   - Reason: Reduces duplication, improves compile-time safety

5. **template.go:211,214,221,269,309,327,414,530,555,607**
   - Pattern: Multiple type checks for template value types
   - Status: 🔄 REFACTOR - Use type-safe value converters
   - Action: Create generic conversion helpers with better error messages
   - Reason: Heavy use of type assertions, can benefit from abstraction

#### pkg/transform/

1. **expression.go:109**
   - Pattern: `boolResult, ok := result.(bool)`
   - Status: ✅ KEEP - Runtime type check for expression result
   - Action: Add comprehensive test coverage
   - Reason: Expression result type determined at runtime

2. **expression.go:166-167,179**
   - Pattern: Multiple assertions in function parameters
   ```go
   str, ok1 := params[0].(string)
   substr, ok2 := params[1].(string)
   ```
   - Status: 🔄 REFACTOR - Use type-safe parameter extraction
   - Action: Create `extractParam[T any](params []interface{}, index int) (T, error)`
   - Reason: Repeated pattern, can be abstracted for safety

3. **expression.go:220**
   - Pattern: `switch v := val.(type)` for value formatting
   - Status: ✅ KEEP - Type switch is idiomatic
   - Action: Ensure default case covers all types
   - Reason: Handles multiple value types for formatting

4. **jsonpath.go:185,777,807,1062,1081,1123**
   - Pattern: Multiple type switches for JSON value handling
   - Status: ✅ KEEP - Type switches are idiomatic for JSON processing
   - Action: Ensure complete type coverage, add default cases
   - Reason: JSON values can be any type

5. **jsonpath.go:755,874,1594**
   - Pattern: Type assertions for specific operations
   - Status: ✅ KEEP - Runtime checks for JSON operations
   - Action: Add comprehensive test coverage
   - Reason: JSON structure determined at runtime

6. **jsonpath.go:926-927,938**
   - Pattern: Parameter type assertions in functions
   - Status: 🔄 REFACTOR - Use type-safe parameter extraction
   - Action: Share with expression.go parameter helper
   - Reason: Same pattern as expression.go

---

## Refactoring Priorities

### Priority 1: High Impact (Immediate Refactoring)

1. **retry.go error handling** (T079)
   - Replace type assertions with `errors.As()`
   - Improves error handling for wrapped errors
   - Estimated time: 30 minutes

2. **variable.go type validation** (T080)
   - Create generic `validateType[T any]()` helper
   - Reduces duplication, improves safety
   - Estimated time: 45 minutes

3. **template.go type conversions** (T080)
   - Create generic type conversion helpers
   - Reduces code duplication significantly
   - Estimated time: 1 hour

### Priority 2: Medium Impact (Good to Have)

4. **expression.go + jsonpath.go parameter extraction** (T081)
   - Create shared `extractParam[T any]()` helper
   - Improves consistency across transform package
   - Estimated time: 45 minutes

5. **events_test.go test helpers** (T079)
   - Create type-safe test fixtures
   - Improves test clarity
   - Estimated time: 20 minutes

### Priority 3: Low Impact (Keep As-Is)

6. **Type switches** - All packages
   - Already idiomatic Go
   - Action: Ensure default cases exist
   - Estimated time: 15 minutes (verification only)

7. **Runtime type checks** - All packages
   - Necessary for dynamic evaluation
   - Action: Ensure test coverage
   - Estimated time: Covered by T078

---

## Refactoring Patterns

### Pattern 1: Error Type Checking with errors.As()

**Before**:
```go
if execErr, ok := err.(*execution.ExecutionError); ok {
    return execErr.Type
}
```

**After**:
```go
var execErr *execution.ExecutionError
if errors.As(err, &execErr) {
    return execErr.Type
}
```

### Pattern 2: Generic Type Validation

**Before**:
```go
if _, ok := v.DefaultValue.(string); !ok {
    return fmt.Errorf("expected string, got %T", v.DefaultValue)
}
```

**After**:
```go
func validateType[T any](value interface{}, typeName string) (T, error) {
    if v, ok := value.(T); ok {
        return v, nil
    }
    var zero T
    return zero, fmt.Errorf("expected %s, got %T", typeName, value)
}

if _, err := validateType[string](v.DefaultValue, "string"); err != nil {
    return err
}
```

### Pattern 3: Generic Parameter Extraction

**Before**:
```go
str, ok1 := params[0].(string)
substr, ok2 := params[1].(string)
if !ok1 || !ok2 {
    return "", false
}
```

**After**:
```go
func extractParam[T any](params []interface{}, index int, name string) (T, error) {
    if index >= len(params) {
        var zero T
        return zero, fmt.Errorf("parameter %d (%s) not provided", index, name)
    }
    if v, ok := params[index].(T); ok {
        return v, nil
    }
    var zero T
    return zero, fmt.Errorf("parameter %d (%s) must be %T, got %T",
        index, name, zero, params[index])
}

str, err := extractParam[string](params, 0, "string")
substr, err := extractParam[string](params, 1, "substring")
```

---

## Success Criteria (from tasks.md)

- [x] FR-023: Use direct typing instead of type assertions when types are known
- [ ] FR-024: Maintain behavioral compatibility
- [ ] SC-003: Compilation must succeed
- [ ] SC-004: Zero test regressions

---

## Implementation Order

1. **T077**: ✅ Complete - This checklist
2. **T078**: Write tests for new helper functions
3. **T079**: Refactor pkg/execution/retry.go + events_test.go
4. **T080**: Refactor pkg/workflow/variable.go + template.go
5. **T081**: Refactor pkg/transform/expression.go + jsonpath.go
6. **T082**: Run full test suite and verify

---

## Notes

- Most type assertions in the codebase are actually **necessary and idiomatic**
- Focus refactoring on **repeated patterns** that can benefit from abstraction
- Use Go 1.21+ **generics** for type-safe helpers
- Maintain **backward compatibility** - don't change public APIs
- **Type switches** are idiomatic Go - keep them, just ensure completeness
- Add **comprehensive tests** before refactoring (T078)
