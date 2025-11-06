# Workflow Template System

## Overview

The GoFlow template system provides a powerful way to create reusable, parameterized workflow definitions. Templates allow you to define workflows with placeholders that are substituted with actual values at instantiation time.

## Core Components

### Template Definition (`WorkflowTemplate`)

A workflow template consists of:

- **Name**: Unique identifier for the template
- **Version**: Template version (semantic versioning recommended)
- **Description**: Human-readable description of the template's purpose
- **Parameters**: List of parameters that can be customized
- **WorkflowSpec**: The workflow definition with parameter placeholders

### Parameter Types

The template system supports four parameter types:

1. **String** (`ParameterTypeString`): Text values
2. **Number** (`ParameterTypeNumber`): Integer or float values
3. **Boolean** (`ParameterTypeBoolean`): true/false values
4. **Array** (`ParameterTypeArray`): Lists of values

### Parameter Definition

Each parameter can specify:

- **Name**: Parameter identifier
- **Type**: One of the four supported types
- **Required**: Whether the parameter must be provided
- **Default**: Default value if not provided (optional parameters only)
- **Description**: Documentation for the parameter
- **Validation**: Constraints on the parameter value

### Parameter Validation

Parameters support comprehensive validation:

- **Numbers**: Min/max range constraints
- **Strings**: Regex pattern matching, min/max length
- **Arrays**: Min/max length constraints

## Template Syntax

Templates use `{{paramName}}` syntax for parameter placeholders:

```yaml
# Simple substitution
url: "{{apiEndpoint}}"

# Multiple parameters in one string
url: "https://{{serverHost}}:{{serverPort}}/{{apiPath}}"

# Nested structures
parameters:
  url: "{{apiURL}}"
  headers:
    X-Server: "{{serverHost}}"
```

## Conditional Node Inclusion

Nodes can be conditionally included based on boolean parameters:

```yaml
nodes:
  - id: log_node
    type: transform
    condition: "{{enableLogging}}"  # Only included if enableLogging=true
    config:
      message: "Workflow started"
```

## Usage Example

```go
package main

import (
    "context"
    "github.com/dshills/goflow/pkg/workflow"
)

func main() {
    // Define a template
    template := &workflow.WorkflowTemplate{
        Name:        "api-integration",
        Version:     "1.0.0",
        Description: "Template for API integrations",
        Parameters: []workflow.TemplateParameter{
            {
                Name:        "apiEndpoint",
                Type:        workflow.ParameterTypeString,
                Required:    true,
                Description: "API endpoint URL",
            },
            {
                Name:     "timeout",
                Type:     workflow.ParameterTypeNumber,
                Required: false,
                Default:  30,
                Validation: &workflow.ParameterValidation{
                    Min: 1,
                    Max: 300,
                },
            },
        },
        WorkflowSpec: workflow.WorkflowSpec{
            Nodes: []workflow.NodeSpec{
                {
                    ID:   "fetch",
                    Type: "mcp_tool",
                    Config: map[string]interface{}{
                        "server": "http-server",
                        "tool":   "fetch",
                        "parameters": map[string]interface{}{
                            "url":     "{{apiEndpoint}}",
                            "timeout": "{{timeout}}",
                        },
                    },
                },
            },
        },
    }

    // Instantiate the template with parameters
    params := map[string]interface{}{
        "apiEndpoint": "https://api.example.com/v1/data",
        "timeout":     60,
    }

    wf, err := workflow.InstantiateTemplate(context.Background(), template, params)
    if err != nil {
        panic(err)
    }

    // wf is now a concrete workflow ready for execution
}
```

## Validation Flow

Template instantiation performs the following validations in order:

1. **Template Structure Validation**
   - Template name and version are required
   - No duplicate parameter names
   - All parameter types are valid

2. **Parameter Reference Validation**
   - All `{{param}}` placeholders reference defined parameters
   - Catches typos and undefined references early

3. **Required Parameter Validation**
   - All required parameters must be provided
   - Missing required parameters result in clear error messages

4. **Parameter Type Validation**
   - Provided values must match declared parameter types
   - Type mismatches are caught before instantiation

5. **Parameter Constraint Validation**
   - Numeric ranges (min/max)
   - String patterns (regex)
   - Array/string length constraints

## Error Handling

The template system provides specific errors for different failure scenarios:

- `ErrInvalidTemplate`: Template structure is invalid
- `ErrDuplicateParameterName`: Parameter name appears multiple times
- `ErrUndefinedParameter`: Placeholder references undefined parameter
- `ErrMissingRequiredParameter`: Required parameter not provided
- `ErrInvalidParameterType`: Value doesn't match declared type
- `ErrParameterValidation`: Value violates validation constraints

## Design Decisions

### Generic Node Wrappers

To preserve parameter types through substitution, the template system uses generic node wrappers:

- `GenericMCPToolNode`: Preserves parameter types in MCP tool nodes
- `GenericTransformNode`: Supports arbitrary config fields
- `GenericConditionNode`: Maintains config alongside condition

These wrappers implement the `Node` interface and preserve the original config structure, allowing type-safe parameter substitution.

### Type-Preserving Substitution

When substituting parameters:

- Single placeholder strings (`"{{param}}"`) preserve the original parameter type
- Multiple placeholders in one string convert all values to strings for concatenation
- Nested structures are recursively processed

Example:
```go
// Single placeholder - type preserved
"{{timeout}}" with timeout=60 → 60 (int)

// Multiple placeholders - string conversion
"https://{{host}}:{{port}}" with host="api.example.com", port=443
→ "https://api.example.com:443" (string)
```

### Early Validation

Parameter reference validation occurs before checking for missing required parameters. This ensures that typos in parameter names are caught even when the required parameter is missing.

## Multiple Instantiations

Templates can be instantiated multiple times with different parameters. Each instantiation creates a new workflow with a unique ID, allowing the same template to be used for multiple scenarios.

```go
// First instance
wf1, _ := workflow.InstantiateTemplate(ctx, template, map[string]interface{}{
    "apiEndpoint": "https://api1.example.com",
})

// Second instance with different parameters
wf2, _ := workflow.InstantiateTemplate(ctx, template, map[string]interface{}{
    "apiEndpoint": "https://api2.example.com",
})

// wf1.ID != wf2.ID - each is a distinct workflow
```

## Future Enhancements (T155 - Built-in Template Library)

The template system is designed to support a built-in template library:

- Template discovery and listing
- Template categories (API integration, data processing, etc.)
- Template validation on load
- Template registry for sharing and reuse

Implementation placeholder in `internal/templates/` for future expansion.
