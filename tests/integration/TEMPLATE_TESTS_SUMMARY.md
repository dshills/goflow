# Workflow Template Instantiation - Integration Tests Summary

**Task**: T149 - Integration test for template instantiation
**Status**: Tests Written (FAILING - awaiting implementation)
**Date**: 2025-11-06
**File**: `tests/integration/template_test.go`

## Overview

Created comprehensive integration tests for workflow template instantiation following TDD principles. These tests define the expected behavior of the template system before implementation begins.

## Template System Requirements (Derived from Tests)

### Core Concept

**Workflow Templates** are parameterized workflow definitions that can be instantiated multiple times with different parameter values to create concrete workflows. This enables:

- Reusable workflow patterns
- Standardized workflow creation
- Parameter-driven workflow configuration
- Type-safe parameter substitution

### Template Structure

```go
type WorkflowTemplate struct {
    Name         string                 // Template identifier
    Description  string                 // Template purpose
    Version      string                 // Template version (semver)
    Parameters   []TemplateParameter    // Parameter definitions
    WorkflowSpec WorkflowSpec           // Workflow definition with placeholders
}
```

### Parameter System

#### Parameter Types

```go
type ParameterType string

const (
    ParameterTypeString  ParameterType = "string"
    ParameterTypeNumber  ParameterType = "number"
    ParameterTypeBoolean ParameterType = "boolean"
    ParameterTypeArray   ParameterType = "array"
)
```

#### Parameter Definition

```go
type TemplateParameter struct {
    Name        string                  // Parameter name (unique)
    Type        ParameterType           // Parameter type
    Required    bool                    // Whether parameter is required
    Default     interface{}             // Default value for optional params
    Description string                  // Parameter documentation
    Validation  *ParameterValidation    // Additional validation rules
}
```

#### Parameter Validation

```go
type ParameterValidation struct {
    Min       interface{} // Minimum value (numbers)
    Max       interface{} // Maximum value (numbers)
    MinLength int         // Minimum length (strings/arrays)
    MaxLength int         // Maximum length (strings/arrays)
    Pattern   string      // Regex pattern (strings)
}
```

### Placeholder Syntax

Parameters are referenced in workflow specs using `{{parameterName}}` syntax:

```yaml
nodes:
  - id: fetch_data
    type: mcp_tool
    config:
      server: fetch-server
      tool: fetch
      parameters:
        url: "https://{{serverHost}}:{{serverPort}}/{{apiPath}}"
        timeout: "{{timeout}}"
```

### Instantiation Function

```go
func InstantiateTemplate(
    ctx context.Context,
    template *WorkflowTemplate,
    params map[string]interface{}
) (*Workflow, error)
```

**Behavior**:
1. Validate template structure
2. Validate all required parameters provided
3. Validate parameter types match definitions
4. Apply parameter validation rules (min/max, pattern, etc.)
5. Apply default values for missing optional parameters
6. Substitute all `{{param}}` placeholders in workflow spec
7. Handle conditional node inclusion (nodes with `Condition` field)
8. Generate unique workflow ID
9. Return instantiated workflow ready for execution

### Error Types

```go
var (
    ErrMissingRequiredParameter = errors.New("required parameter not provided")
    ErrInvalidParameterType     = errors.New("parameter type mismatch")
    ErrInvalidTemplate          = errors.New("invalid template definition")
    ErrDuplicateParameterName   = errors.New("duplicate parameter name")
    ErrUndefinedParameter       = errors.New("parameter referenced but not defined")
)
```

## Test Scenarios Implemented

### 1. TestTemplateInstantiationWithAllParameters ✓

**Purpose**: Verify basic template instantiation with all parameters provided

**Tests**:
- Template with required and optional parameters
- Parameter substitution in node configurations
- Nested parameter substitution (parameters in nested maps)
- Generated workflow has correct structure

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "apiEndpoint", Type: ParameterTypeString, Required: true},
        {Name: "timeout", Type: ParameterTypeNumber, Required: false, Default: 30},
    },
    WorkflowSpec: WorkflowSpec{
        Nodes: []NodeSpec{
            {
                ID: "fetch_data",
                Config: map[string]interface{}{
                    "url": "{{apiEndpoint}}",
                    "timeout": "{{timeout}}",
                },
            },
        },
    },
}

params := map[string]interface{}{
    "apiEndpoint": "https://api.example.com/v1/data",
    "timeout": 60,
}

workflow, err := InstantiateTemplate(ctx, template, params)
// Expects: workflow with substituted values
```

### 2. TestTemplateInstantiationMissingRequiredParameters ✓

**Purpose**: Ensure required parameters are enforced

**Tests**:
- Missing required parameter returns ErrMissingRequiredParameter
- Error message identifies missing parameter
- Partial parameter provision fails

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "required1", Type: ParameterTypeString, Required: true},
        {Name: "required2", Type: ParameterTypeNumber, Required: true},
    },
}

params := map[string]interface{}{
    "required1": "value1",
    // missing required2
}

_, err := InstantiateTemplate(ctx, template, params)
// Expects: ErrMissingRequiredParameter
```

### 3. TestTemplateInstantiationWithDefaults ✓

**Purpose**: Verify default values are applied for optional parameters

**Tests**:
- String default values
- Number default values
- Boolean default values
- Array default values
- Defaults applied when parameter not provided
- Provided values override defaults

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "optionalString", Type: ParameterTypeString, Required: false, Default: "default-value"},
        {Name: "optionalNumber", Type: ParameterTypeNumber, Required: false, Default: 42},
    },
}

params := map[string]interface{}{} // No optional params provided

workflow, err := InstantiateTemplate(ctx, template, params)
// Expects: workflow with default values substituted
```

### 4. TestTemplateParameterTypeValidation ✓

**Purpose**: Ensure parameter type safety

**Tests**:
- String parameters accept strings only
- Number parameters accept int and float64
- Boolean parameters accept bool only
- Array parameters accept []interface{} only
- Type mismatches return ErrInvalidParameterType

**Test Matrix**:
| Parameter Type | Valid Values | Invalid Values |
|---------------|-------------|----------------|
| String | "hello" | 123, true |
| Number | 42, 42.5 | "not a number", true |
| Boolean | true, false | "true", 1 |
| Array | []interface{}{...} | "not array", 42 |

### 5. TestTemplateNestedParameterSubstitution ✓

**Purpose**: Verify deep parameter substitution in nested structures

**Tests**:
- Parameters in deeply nested maps
- Multiple parameters in same string value
- Parameters in array elements
- Parameters in nested object properties

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "serverHost", Type: ParameterTypeString, Required: true},
        {Name: "serverPort", Type: ParameterTypeNumber, Required: true},
    },
    WorkflowSpec: WorkflowSpec{
        Nodes: []NodeSpec{
            {
                Config: map[string]interface{}{
                    "parameters": map[string]interface{}{
                        "url": "https://{{serverHost}}:{{serverPort}}/api",
                        "headers": map[string]interface{}{
                            "X-Server": "{{serverHost}}",
                        },
                    },
                },
            },
        },
    },
}
// Expects: all nested {{param}} replaced correctly
```

### 6. TestTemplateValidationBeforeInstantiation ✓

**Purpose**: Ensure template validation before instantiation

**Tests**:
- Template with empty name fails
- Template with empty version fails
- Template with duplicate parameter names fails
- Template with invalid parameter types fails
- Template with undefined parameters in spec fails

**Validation Rules**:
1. Template must have non-empty name
2. Template must have valid version
3. Parameter names must be unique
4. Parameter types must be valid
5. All `{{param}}` in spec must be defined in Parameters
6. WorkflowSpec must be valid

### 7. TestTemplateMultipleInstantiations ✓

**Purpose**: Verify same template can create multiple distinct workflows

**Tests**:
- Multiple instantiations produce unique workflow IDs
- Each instantiation has correct parameter substitutions
- Instantiations are independent (changes don't affect others)
- Template can be reused indefinitely

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "instanceName", Type: ParameterTypeString, Required: true},
    },
}

wf1, _ := InstantiateTemplate(ctx, template, map[string]interface{}{
    "instanceName": "instance-1",
})

wf2, _ := InstantiateTemplate(ctx, template, map[string]interface{}{
    "instanceName": "instance-2",
})

// Expects: wf1.ID != wf2.ID
// Expects: wf1 has "instance-1", wf2 has "instance-2"
```

### 8. TestTemplateWithConditionalSections ✓

**Purpose**: Support conditional node inclusion based on parameters

**Tests**:
- Nodes with `Condition: "{{boolParam}}"` included when param is true
- Nodes excluded when condition param is false
- Multiple conditional sections work independently
- Conditional nodes don't break edge validation

**Example**:
```go
template := &WorkflowTemplate{
    Parameters: []TemplateParameter{
        {Name: "enableLogging", Type: ParameterTypeBoolean, Default: false},
    },
    WorkflowSpec: WorkflowSpec{
        Nodes: []NodeSpec{
            {
                ID: "log_start",
                Type: "transform",
                Condition: "{{enableLogging}}", // Only include if true
            },
        },
    },
}

// When enableLogging=true: log_start node included
// When enableLogging=false: log_start node excluded from result
```

### 9. TestTemplateWithParameterSchema ✓

**Purpose**: Validate parameters against extended constraints

**Tests**:
- Number min/max validation
- String regex pattern validation
- Array min/max length validation
- Constraint violations return appropriate errors

**Validation Types**:

**Number Constraints**:
```go
{
    Name: "port",
    Type: ParameterTypeNumber,
    Validation: &ParameterValidation{
        Min: 1,
        Max: 65535,
    },
}
// Rejects: 0, 70000
// Accepts: 8080, 443
```

**String Pattern**:
```go
{
    Name: "email",
    Type: ParameterTypeString,
    Validation: &ParameterValidation{
        Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
    },
}
// Rejects: "not-an-email"
// Accepts: "user@example.com"
```

**Array Length**:
```go
{
    Name: "tags",
    Type: ParameterTypeArray,
    Validation: &ParameterValidation{
        MinLength: 1,
        MaxLength: 5,
    },
}
// Rejects: [], ["t1","t2","t3","t4","t5","t6"]
// Accepts: ["tag1"], ["t1","t2","t3"]
```

## Implementation Requirements

### 1. New Types in pkg/workflow/template.go

```go
// WorkflowTemplate - parameterized workflow definition
type WorkflowTemplate struct {
    Name         string
    Description  string
    Version      string
    Parameters   []TemplateParameter
    WorkflowSpec WorkflowSpec
}

// TemplateParameter - parameter definition
type TemplateParameter struct {
    Name        string
    Type        ParameterType
    Required    bool
    Default     interface{}
    Description string
    Validation  *ParameterValidation
}

// ParameterType - allowed parameter types
type ParameterType string

const (
    ParameterTypeString  ParameterType = "string"
    ParameterTypeNumber  ParameterType = "number"
    ParameterTypeBoolean ParameterType = "boolean"
    ParameterTypeArray   ParameterType = "array"
)

// ParameterValidation - extended validation rules
type ParameterValidation struct {
    Min       interface{}
    Max       interface{}
    MinLength int
    MaxLength int
    Pattern   string
}

// WorkflowSpec - intermediate workflow definition with placeholders
type WorkflowSpec struct {
    Nodes []NodeSpec
    Edges []EdgeSpec
}

// NodeSpec - node definition with placeholders
type NodeSpec struct {
    ID        string
    Type      string
    Condition string                 // Optional: "{{boolParam}}"
    Config    map[string]interface{} // May contain {{param}} placeholders
}

// EdgeSpec - edge definition
type EdgeSpec struct {
    From string
    To   string
    When string
}
```

### 2. Core Function

```go
// InstantiateTemplate creates a concrete workflow from a template
func InstantiateTemplate(
    ctx context.Context,
    template *WorkflowTemplate,
    params map[string]interface{}
) (*Workflow, error) {
    // 1. Validate template structure
    // 2. Validate all required parameters provided
    // 3. Validate parameter types
    // 4. Apply validation rules (min/max, pattern, etc.)
    // 5. Apply defaults for missing optional parameters
    // 6. Substitute all {{param}} placeholders
    // 7. Handle conditional nodes
    // 8. Build Workflow from WorkflowSpec
    // 9. Return workflow
}
```

### 3. Helper Functions Needed

```go
// validateTemplate validates template structure
func validateTemplate(template *WorkflowTemplate) error

// validateParameters validates provided parameters against template
func validateParameters(
    template *WorkflowTemplate,
    params map[string]interface{}
) error

// substituteParameters replaces {{param}} with values
func substituteParameters(
    spec WorkflowSpec,
    params map[string]interface{}
) (WorkflowSpec, error)

// applyDefaults applies default values to params
func applyDefaults(
    parameters []TemplateParameter,
    params map[string]interface{}
) map[string]interface{}

// buildWorkflow converts WorkflowSpec to Workflow
func buildWorkflow(spec WorkflowSpec, name string) (*Workflow, error)
```

### 4. Error Definitions

```go
var (
    ErrMissingRequiredParameter = errors.New("required parameter not provided")
    ErrInvalidParameterType     = errors.New("parameter type mismatch")
    ErrInvalidTemplate          = errors.New("invalid template definition")
    ErrDuplicateParameterName   = errors.New("duplicate parameter name")
    ErrUndefinedParameter       = errors.New("parameter referenced but not defined")
    ErrParameterValidationFailed = errors.New("parameter validation failed")
)
```

## Usage Examples

### Example 1: API Integration Template

```go
template := &WorkflowTemplate{
    Name: "api-integration",
    Description: "Template for REST API integration",
    Version: "1.0.0",
    Parameters: []TemplateParameter{
        {
            Name: "apiBaseURL",
            Type: ParameterTypeString,
            Required: true,
            Description: "Base URL for the API",
        },
        {
            Name: "apiKey",
            Type: ParameterTypeString,
            Required: true,
            Description: "API authentication key",
        },
        {
            Name: "timeout",
            Type: ParameterTypeNumber,
            Required: false,
            Default: 30,
            Description: "Request timeout in seconds",
        },
    },
    WorkflowSpec: WorkflowSpec{
        Nodes: []NodeSpec{
            {
                ID: "fetch_data",
                Type: "mcp_tool",
                Config: map[string]interface{}{
                    "server": "fetch",
                    "tool": "fetch",
                    "parameters": map[string]interface{}{
                        "url": "{{apiBaseURL}}/data",
                        "headers": map[string]interface{}{
                            "Authorization": "Bearer {{apiKey}}",
                        },
                        "timeout": "{{timeout}}",
                    },
                },
            },
        },
    },
}

// Use template
params := map[string]interface{}{
    "apiBaseURL": "https://api.example.com/v1",
    "apiKey": "secret-key",
    "timeout": 60,
}

workflow, err := InstantiateTemplate(ctx, template, params)
```

### Example 2: ETL Pipeline Template

```go
template := &WorkflowTemplate{
    Name: "etl-pipeline",
    Description: "Extract, Transform, Load pipeline",
    Version: "1.0.0",
    Parameters: []TemplateParameter{
        {
            Name: "sourceDB",
            Type: ParameterTypeString,
            Required: true,
        },
        {
            Name: "targetDB",
            Type: ParameterTypeString,
            Required: true,
        },
        {
            Name: "enableValidation",
            Type: ParameterTypeBoolean,
            Required: false,
            Default: true,
        },
    },
    WorkflowSpec: WorkflowSpec{
        Nodes: []NodeSpec{
            {
                ID: "extract",
                Type: "mcp_tool",
                Config: map[string]interface{}{
                    "server": "sqlite",
                    "tool": "query",
                    "database": "{{sourceDB}}",
                },
            },
            {
                ID: "validate",
                Type: "transform",
                Condition: "{{enableValidation}}", // Conditional node
                Config: map[string]interface{}{
                    "validator": "schema-validator",
                },
            },
            {
                ID: "load",
                Type: "mcp_tool",
                Config: map[string]interface{}{
                    "server": "sqlite",
                    "tool": "execute",
                    "database": "{{targetDB}}",
                },
            },
        },
    },
}
```

## Test Execution Results

**Current Status**: ❌ All tests FAILING (expected)

**Compilation Errors**:
```
tests/integration/template_test.go:15:24: undefined: workflow.WorkflowTemplate
tests/integration/template_test.go:19:26: undefined: workflow.TemplateParameter
tests/integration/template_test.go:22:27: undefined: workflow.ParameterTypeString
tests/integration/template_test.go:47:26: undefined: workflow.WorkflowSpec
tests/integration/template_test.go:83:22: undefined: workflow.InstantiateTemplate
```

**Expected**: These errors confirm tests are written before implementation (TDD)

## Next Steps

1. **Implement Core Types** (T153):
   - Create `pkg/workflow/template.go`
   - Define all template-related types
   - Define error types

2. **Implement InstantiateTemplate Function** (T154):
   - Parameter validation
   - Type checking
   - Default value application
   - Placeholder substitution
   - Conditional node handling
   - Workflow construction

3. **Run Tests**:
   ```bash
   go test ./tests/integration -run TestTemplate -v
   ```

4. **Iterate Until All Tests Pass**:
   - Fix compilation errors
   - Implement missing functionality
   - Debug failing assertions
   - Ensure all 9 test scenarios pass

## Coverage

**Test Count**: 9 comprehensive integration tests
**Test Scenarios**: 27+ individual test cases
**Lines of Test Code**: 850+

**Areas Covered**:
- ✅ Basic instantiation with all parameters
- ✅ Required parameter enforcement
- ✅ Default value application
- ✅ Type validation (4 types × multiple cases)
- ✅ Nested parameter substitution
- ✅ Template validation
- ✅ Multiple instantiations
- ✅ Conditional sections
- ✅ Parameter schema validation (min/max, pattern, length)

**Areas NOT Covered** (for future tests):
- Template inheritance/extension
- Template versioning and migration
- Template catalog/registry
- Template validation with MCP server schemas
- Performance testing with large templates
- Concurrent instantiation stress testing

## Design Decisions

### 1. Placeholder Syntax: `{{param}}` vs `${param}`

**Chosen**: `{{param}}`
**Rationale**:
- Distinct from string template syntax `${param}` (used by transform nodes)
- More visible in YAML files
- Common in template systems (Handlebars, Go templates)
- Less likely to conflict with shell variable syntax

### 2. Parameter Type System

**Chosen**: Limited set (string, number, boolean, array)
**Rationale**:
- Covers 95% of use cases
- Easy to validate
- Maps cleanly to JSON/YAML types
- Extensible for future object/map types

### 3. Conditional Node Inclusion

**Chosen**: `Condition` field on NodeSpec
**Rationale**:
- Declarative approach
- No need for if/else logic in template
- Boolean parameters naturally enable/disable features
- Simpler than template sections or loops

### 4. Validation Strategy

**Chosen**: Validate early, fail fast
**Rationale**:
- Catch errors before workflow execution
- Clear error messages at instantiation time
- Prevents runtime failures from bad parameters
- Template validation separate from parameter validation

### 5. Workflow ID Generation

**Chosen**: Generate new UUID for each instantiation
**Rationale**:
- Each instantiation is a distinct workflow
- Enables multiple instances from same template
- Simplifies workflow tracking and history
- No collision concerns

## Compatibility Notes

### String Template Rendering (Existing)

The **string template system** (`pkg/transform/template.go`) uses `${param}` syntax for rendering string values within workflow execution. This is **separate and complementary** to workflow template instantiation.

**String Templates**: Runtime value substitution within node execution
**Workflow Templates**: Pre-execution workflow structure creation

**Example**:
```yaml
# Workflow Template (instantiation time)
parameters:
  - name: apiURL
    type: string

nodes:
  - id: fetch
    config:
      url: "{{apiURL}}"  # Substituted at instantiation

# String Template (execution time)
nodes:
  - id: transform
    config:
      expression: "Hello ${user.name}"  # Substituted at runtime
```

Both systems can coexist:
```yaml
# Combined example
parameters:
  - name: baseURL
    type: string

nodes:
  - id: fetch
    config:
      url: "{{baseURL}}/users"  # Template substitution
  - id: transform
    expression: "User: ${response.name}"  # Runtime substitution
```

## References

- GoFlow Specification: `/specs/goflow-specification.md` (Section: Workflow Templates)
- Task Definition: `/specs/001-goflow-spec-review/tasks.md` (T149)
- Related Tasks: T147 (export), T148 (import), T153-155 (implementation)
- String Template Tests: `/tests/integration/transform_template_test.go`

---

**Status**: Ready for implementation (T153-T154)
**Blocked By**: None (tests define the contract)
**Blocks**: T155 (built-in templates), T156-157 (export/import CLI)
