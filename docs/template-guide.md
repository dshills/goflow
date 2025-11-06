# Workflow Template Creation Guide

This guide teaches you how to create your own workflow templates in GoFlow. Templates allow you to build reusable, parameterized workflows that can be instantiated with different values for different scenarios.

## Table of Contents

- [Introduction](#introduction)
- [When to Use Templates](#when-to-use-templates)
- [Template Structure](#template-structure)
- [Parameters](#parameters)
- [Parameter Substitution](#parameter-substitution)
- [Creating Templates](#creating-templates)
- [Built-in Templates](#built-in-templates)
- [Template Instantiation](#template-instantiation)
- [Advanced Topics](#advanced-topics)
- [Best Practices](#best-practices)
- [Reference](#reference)

## Introduction

### What Are Workflow Templates?

Workflow templates are reusable workflow definitions with parameterized values. Instead of hardcoding specific values like API endpoints, file paths, or configuration settings, you define parameters that can be provided when creating a workflow from the template.

Think of templates as workflow blueprints. One template can generate many concrete workflows, each customized with different parameter values.

### Benefits of Templates

- **Reusability**: Write once, instantiate many times with different parameters
- **Consistency**: Ensure workflows follow the same pattern and best practices
- **Maintainability**: Update the template once to affect all future instantiations
- **Sharing**: Share templates across teams or publish them for community use
- **Type Safety**: Parameter validation ensures correct values are provided

## When to Use Templates

Create a template when:

1. **You repeat similar workflows**: Multiple workflows with the same structure but different values (different API endpoints, file paths, etc.)
2. **You want to share workflows**: Templates make it easy for others to create workflows without understanding all the details
3. **You need flexibility**: One workflow pattern that needs to adapt to different environments (dev/staging/production)
4. **You value consistency**: Ensure all team members follow the same workflow patterns
5. **You need validation**: Ensure parameters meet specific constraints before execution

Examples of good template candidates:
- ETL pipelines with different data sources
- API integrations with different endpoints
- Data processing workflows with configurable batch sizes
- Multi-environment deployments (dev/test/prod)
- Standard operating procedures with variable inputs

## Template Structure

A workflow template has this YAML structure:

```yaml
name: template-name
description: What this template does
version: "1.0"

parameters:
  - name: param_name
    type: string
    required: true
    description: What this parameter controls
    validation:
      pattern: "regex pattern"

workflow_spec:
  nodes:
    - id: node1
      type: mcp_tool
      config:
        server: server-name
        tool: tool-name
        parameters:
          key: "{{param_name}}"

  edges:
    - from: node1
      to: node2
```

### Root Level Fields

- **name**: Unique identifier for the template (lowercase, hyphens)
- **description**: Brief explanation of what the template does
- **version**: Template version (use semantic versioning: "1.0", "2.1", etc.)
- **parameters**: List of parameters (optional - templates can have no parameters)
- **workflow_spec**: The workflow definition with parameter placeholders

## Parameters

Parameters are the heart of templates. They define what can be customized when instantiating a workflow.

### Parameter Definition

Each parameter has these fields:

```yaml
parameters:
  - name: api_endpoint           # Required: parameter identifier
    type: string                 # Required: string, number, boolean, array
    required: true               # Required: is this parameter mandatory?
    default: "https://api.com"   # Optional: default value
    description: "API URL"       # Optional: documentation
    validation:                  # Optional: constraints
      pattern: "^https://.+"     # Validation rules
```

### Parameter Types

#### String

Text values - most commonly used type.

```yaml
- name: file_path
  type: string
  required: true
  description: Path to the input file
  validation:
    min_length: 1
    max_length: 4096
    pattern: "^/.*\\.json$"  # Must be absolute path ending in .json
```

**Validation Options**:
- `min_length`: Minimum string length
- `max_length`: Maximum string length
- `pattern`: Regular expression the string must match

**Example Values**:
```go
params := map[string]interface{}{
    "file_path": "/data/input.json",
}
```

#### Number

Integer or floating-point values.

```yaml
- name: timeout_seconds
  type: number
  required: false
  default: 30
  description: Request timeout in seconds
  validation:
    min: 1
    max: 300
```

**Validation Options**:
- `min`: Minimum value (inclusive)
- `max`: Maximum value (inclusive)

**Example Values**:
```go
params := map[string]interface{}{
    "timeout_seconds": 60,        // int
    "retry_delay": 1.5,           // float64
    "batch_size": int64(1000),    // int64
}
```

#### Boolean

True/false values - useful for feature flags and conditional logic.

```yaml
- name: enable_logging
  type: boolean
  required: false
  default: true
  description: Enable detailed logging
```

**Note**: Booleans have no validation options.

**Example Values**:
```go
params := map[string]interface{}{
    "enable_logging": true,
}
```

#### Array

Lists of values - useful for multiple items of the same type.

```yaml
- name: email_recipients
  type: array
  required: false
  default: []
  description: Email addresses to notify
  validation:
    min_length: 1
    max_length: 10
```

**Validation Options**:
- `min_length`: Minimum array length
- `max_length`: Maximum array length

**Example Values**:
```go
params := map[string]interface{}{
    "email_recipients": []string{"user1@example.com", "user2@example.com"},
    "ports": []int{8080, 8081, 8082},
}
```

### Required vs Optional Parameters

**Required Parameters** must be provided at instantiation:

```yaml
- name: api_key
  type: string
  required: true  # Must be provided
  description: API authentication key
```

**Optional Parameters** can have defaults:

```yaml
- name: retry_count
  type: number
  required: false  # Can be omitted
  default: 3      # Will use this if not provided
  description: Number of retry attempts
```

**Best Practices**:
- Make parameters required only when there's no sensible default
- Always provide defaults for optional parameters
- Document why a parameter is required in the description

## Parameter Substitution

Parameters are substituted using `{{parameter_name}}` syntax.

### Basic Substitution

Replace a single value:

```yaml
nodes:
  - id: fetch_data
    type: mcp_tool
    config:
      server: http-server
      tool: fetch
      parameters:
        url: "{{api_endpoint}}"  # Replaced with parameter value
        timeout: "{{timeout}}"    # Can be any YAML value
```

### Multiple Parameters in One String

Combine multiple parameters:

```yaml
nodes:
  - id: log_message
    type: transform
    config:
      message: "Processing {{file_count}} files from {{source_dir}}"
```

With parameters:
```go
params := map[string]interface{}{
    "file_count": 42,
    "source_dir": "/data/input",
}
```

Result: `"Processing 42 files from /data/input"`

### Type Preservation

**Single Placeholder**: Type is preserved

```yaml
timeout: "{{timeout_seconds}}"  # With timeout_seconds=60
# Result: timeout: 60 (integer, not string)
```

**Multiple Placeholders**: Converted to string

```yaml
url: "https://{{host}}:{{port}}/api"  # With host="api.example.com", port=443
# Result: url: "https://api.example.com:443/api" (string)
```

### Nested Structures

Parameters work in nested YAML structures:

```yaml
nodes:
  - id: api_call
    type: mcp_tool
    config:
      server: http-server
      tool: request
      parameters:
        url: "{{api_url}}"
        method: "{{http_method}}"
        headers:
          Authorization: "Bearer {{api_token}}"
          X-Custom-Header: "{{custom_header}}"
        body:
          query: "{{search_query}}"
          limit: "{{result_limit}}"
```

### Parameter Substitution in Expressions

Use parameters in workflow expressions:

```yaml
nodes:
  - id: check_threshold
    type: condition
    config:
      condition: "${processed_count > {{threshold_value}}}"
```

Parameters are substituted first, then expressions are evaluated.

## Creating Templates

### From Scratch

Follow these steps to create a new template:

#### Step 1: Define Template Metadata

```yaml
name: my-workflow-template
description: Short description of what this template does
version: "1.0"
```

#### Step 2: Identify Parameters

List what should be customizable. Ask:
- What varies between different uses of this workflow?
- What configuration should users control?
- What environment-specific values exist?

```yaml
parameters:
  - name: input_path
    type: string
    required: true
    description: Path to input data file

  - name: output_path
    type: string
    required: true
    description: Path where results will be written

  - name: batch_size
    type: number
    required: false
    default: 100
    description: Number of records per batch
    validation:
      min: 1
      max: 1000
```

#### Step 3: Define Workflow Structure

Create the workflow with parameter placeholders:

```yaml
workflow_spec:
  nodes:
    - id: start
      type: start

    - id: read_input
      type: mcp_tool
      config:
        server: filesystem-server
        tool: read_file
        parameters:
          path: "{{input_path}}"
        output_variable: input_data

    - id: process_batches
      type: transform
      config:
        input: "${input_data}"
        expression: "jq([range(0; length; {{batch_size}})])"
        output_variable: batches

    - id: write_output
      type: mcp_tool
      config:
        server: filesystem-server
        tool: write_file
        parameters:
          path: "{{output_path}}"
          content: "${processed_data}"

    - id: end
      type: end
      config:
        return_value: "${processed_data}"

  edges:
    - from: start
      to: read_input
    - from: read_input
      to: process_batches
    - from: process_batches
      to: write_output
    - from: write_output
      to: end
```

#### Step 4: Add Validation

Add validation rules to ensure correct parameter values:

```yaml
parameters:
  - name: email_address
    type: string
    required: true
    validation:
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

  - name: port
    type: number
    required: true
    validation:
      min: 1
      max: 65535
```

### From Existing Workflows

Convert an existing workflow to a template:

#### Step 1: Identify Hardcoded Values

Find values that should be parameters:

```yaml
# Before: Hardcoded workflow
nodes:
  - id: fetch_data
    type: mcp_tool
    config:
      server: http-server
      tool: fetch
      parameters:
        url: "https://api.example.com/data"  # Hardcoded!
        timeout: 30                           # Hardcoded!
```

#### Step 2: Replace with Parameters

```yaml
# After: Template with parameters
parameters:
  - name: api_url
    type: string
    required: true
    description: API endpoint URL
    validation:
      pattern: "^https://.+"

  - name: timeout
    type: number
    required: false
    default: 30
    validation:
      min: 1
      max: 300

workflow_spec:
  nodes:
    - id: fetch_data
      type: mcp_tool
      config:
        server: http-server
        tool: fetch
        parameters:
          url: "{{api_url}}"      # Now a parameter
          timeout: "{{timeout}}"  # Now a parameter
```

#### Step 3: Add Template Metadata

```yaml
name: api-fetch-template
description: Fetches data from a configurable API endpoint
version: "1.0"
# ... parameters and workflow_spec follow
```

### Template File Organization

Organize template files:

```
workflows/
  templates/
    my-template.yaml          # Your custom templates
    api-integration.yaml
    data-pipeline.yaml
  instances/
    prod-api-workflow.yaml    # Workflows created from templates
    dev-pipeline.yaml
```

## Built-in Templates

GoFlow includes several built-in templates you can use as-is or as examples for your own templates.

### ETL Pipeline Template

Extract-Transform-Load workflow for data processing.

**File**: `internal/templates/etl-pipeline.yaml`

**Parameters**:
- `source_path` (string, required): Path to source data file
- `destination_path` (string, required): Path for processed output
- `transform_expression` (string, optional): JSONPath transformation (default: "$")
- `batch_size` (number, optional): Records per batch (default: 100)
- `validate_output` (boolean, optional): Enable output validation (default: true)
- `error_handling` (string, optional): Error strategy (default: "fail_fast")

**Features**:
- Three-phase pipeline: Extract, Transform, Load
- Batch processing support
- Optional output validation
- Error handling with multiple endpoints
- Detailed success metrics

**Example Usage**:
```go
params := map[string]interface{}{
    "source_path":      "/data/raw/users.json",
    "destination_path": "/data/processed/users.json",
    "batch_size":       50,
    "validate_output":  true,
}

workflow, err := workflow.InstantiateTemplate(ctx, etlTemplate, params)
```

### API Workflow Template

HTTP API integration with retry logic and error handling.

**File**: `internal/templates/api-workflow.yaml`

**Parameters**:
- `api_endpoint` (string, required): Target API URL (must be https://)
- `api_method` (string, optional): HTTP method (default: "GET")
- `request_body` (string, optional): JSON request body (default: "")
- `retry_count` (number, optional): Retry attempts (default: 3, max: 10)
- `timeout_seconds` (number, optional): Request timeout (default: 30, max: 300)

**Features**:
- Configurable HTTP method and request body
- Response status validation
- Separate success and error processing paths
- Error logging
- Detailed response transformation

**Example Usage**:
```go
params := map[string]interface{}{
    "api_endpoint":    "https://api.example.com/users",
    "api_method":      "POST",
    "request_body":    `{"name":"John","email":"john@example.com"}`,
    "retry_count":     5,
    "timeout_seconds": 60,
}

workflow, err := workflow.InstantiateTemplate(ctx, apiTemplate, params)
```

### Multi-Server Workflow Template

Coordinates multiple MCP servers (filesystem, database, notifications).

**File**: `internal/templates/multi-server.yaml`

**Parameters**:
- `input_source` (string, required): Data source identifier
- `processing_mode` (string, optional): "sequential" or "parallel" (default: "sequential")
- `notification_enabled` (boolean, optional): Enable notifications (default: true)
- `recipients` (array, optional): Email recipients (default: [], max: 10)

**Features**:
- Multi-server coordination
- Conditional notification logic
- Sequential or parallel processing
- Database integration
- Success metrics tracking

**Example Usage**:
```go
params := map[string]interface{}{
    "input_source":         "/data/sales.csv",
    "processing_mode":      "parallel",
    "notification_enabled": true,
    "recipients":           []string{"admin@example.com", "ops@example.com"},
}

workflow, err := workflow.InstantiateTemplate(ctx, multiServerTemplate, params)
```

## Template Instantiation

### Programmatic Instantiation

Use the `InstantiateTemplate` function:

```go
package main

import (
    "context"
    "github.com/dshills/goflow/pkg/workflow"
)

func main() {
    ctx := context.Background()

    // Load or define your template
    template := &workflow.WorkflowTemplate{
        Name:    "my-template",
        Version: "1.0",
        Parameters: []workflow.TemplateParameter{
            {
                Name:     "api_url",
                Type:     workflow.ParameterTypeString,
                Required: true,
            },
            {
                Name:     "timeout",
                Type:     workflow.ParameterTypeNumber,
                Required: false,
                Default:  30,
            },
        },
        WorkflowSpec: workflow.WorkflowSpec{
            // ... nodes and edges
        },
    }

    // Provide parameter values
    params := map[string]interface{}{
        "api_url": "https://api.example.com/v1",
        "timeout": 60,
    }

    // Instantiate the template
    wf, err := workflow.InstantiateTemplate(ctx, template, params)
    if err != nil {
        panic(err)
    }

    // wf is now a concrete workflow ready for execution
    // Execute it with your execution engine
}
```

### CLI Instantiation

Using the GoFlow CLI (when implemented):

```bash
# Create workflow from template
goflow create my-workflow --template api-workflow \
  --param api_endpoint="https://api.example.com" \
  --param timeout_seconds=60 \
  --param retry_count=5

# Using a parameter file
goflow create my-workflow --template etl-pipeline \
  --params-file params.json

# params.json:
{
  "source_path": "/data/input.json",
  "destination_path": "/data/output.json",
  "batch_size": 100,
  "validate_output": true
}
```

### Validation During Instantiation

The template system validates in this order:

1. **Template Structure**: Name, version, parameter definitions
2. **Parameter References**: All `{{param}}` placeholders exist
3. **Required Parameters**: All required parameters provided
4. **Type Validation**: Values match declared types
5. **Constraint Validation**: Values meet validation rules

**Example Error Messages**:

```
Missing required parameter: api_endpoint

Invalid parameter type: timeout expected type number, got string

Parameter validation failed for retry_count: value 15 is greater than maximum 10

Undefined parameter: api_endpont in node fetch_data field url
(Did you mean: api_endpoint?)
```

## Advanced Topics

### Conditional Node Inclusion

Include or exclude nodes based on boolean parameters:

```yaml
parameters:
  - name: enable_caching
    type: boolean
    required: false
    default: false
    description: Enable response caching

workflow_spec:
  nodes:
    - id: cache_check
      type: mcp_tool
      condition: "{{enable_caching}}"  # Only included if true
      config:
        server: cache-server
        tool: get_cached
        parameters:
          key: "{{cache_key}}"

    - id: fetch_fresh
      type: mcp_tool
      condition: "{{enable_caching}}"  # Paired with cache logic
      config:
        server: http-server
        tool: fetch
```

**How It Works**:
- When `enable_caching: true`, both nodes are included
- When `enable_caching: false`, both nodes are excluded
- Edges referencing excluded nodes are also excluded

**Best Practices**:
- Use for optional features (logging, caching, notifications)
- Keep conditional sections self-contained
- Document the behavior in parameter descriptions

### Array Parameters

Work with lists of values:

```yaml
parameters:
  - name: webhook_urls
    type: array
    required: true
    description: List of webhooks to notify
    validation:
      min_length: 1
      max_length: 5

workflow_spec:
  nodes:
    - id: notify_webhooks
      type: loop
      config:
        collection: "{{webhook_urls}}"
        item_variable: webhook_url
        body:
          - notify_single_webhook

    - id: notify_single_webhook
      type: mcp_tool
      config:
        server: http-server
        tool: post
        parameters:
          url: "${webhook_url}"
          body: "${notification_data}"
```

**Usage**:
```go
params := map[string]interface{}{
    "webhook_urls": []string{
        "https://webhook1.example.com/notify",
        "https://webhook2.example.com/notify",
        "https://webhook3.example.com/notify",
    },
}
```

### Complex Validations

Combine multiple validation rules:

```yaml
parameters:
  - name: username
    type: string
    required: true
    description: User login name
    validation:
      min_length: 3
      max_length: 20
      pattern: "^[a-z][a-z0-9_-]*$"  # Must start with letter, lowercase only

  - name: password
    type: string
    required: true
    description: User password
    validation:
      min_length: 8
      max_length: 128
      pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]+"
      # Requires: lowercase, uppercase, digit, special char

  - name: age
    type: number
    required: true
    description: User age
    validation:
      min: 18
      max: 120
```

### Template Versioning

Version your templates to support evolution:

```yaml
# Version 1.0
name: api-integration
version: "1.0"
parameters:
  - name: url
    type: string
    required: true

---

# Version 2.0 - Added timeout parameter
name: api-integration
version: "2.0"
parameters:
  - name: url
    type: string
    required: true
  - name: timeout
    type: number
    required: false
    default: 30  # Default for backward compatibility
```

**Best Practices**:
- Increment version when adding/removing parameters
- Provide defaults for new optional parameters
- Document breaking changes in description
- Keep old versions for backward compatibility

### Nested Configuration

Use parameters deep in nested structures:

```yaml
parameters:
  - name: db_host
    type: string
    required: true
  - name: db_port
    type: number
    required: false
    default: 5432
  - name: db_name
    type: string
    required: true
  - name: pool_size
    type: number
    required: false
    default: 10

workflow_spec:
  nodes:
    - id: connect_db
      type: mcp_tool
      config:
        server: database-server
        tool: connect
        parameters:
          connection:
            host: "{{db_host}}"
            port: "{{db_port}}"
            database: "{{db_name}}"
            options:
              pool_size: "{{pool_size}}"
              ssl_mode: "require"
              timeout: 30
```

## Best Practices

### Parameter Naming

**DO**:
- Use lowercase with underscores: `api_endpoint`, `retry_count`
- Be descriptive: `timeout_seconds` not `timeout`
- Use consistent naming: `source_path` and `destination_path`

**DON'T**:
- Use camelCase: `apiEndpoint` (inconsistent with YAML conventions)
- Use single letters: `x`, `n` (unclear purpose)
- Use inconsistent patterns: `srcPath` and `destination_path`

### Parameter Descriptions

Write clear, helpful descriptions:

**Good**:
```yaml
- name: batch_size
  type: number
  description: Number of records to process in each batch. Higher values use more memory but may be faster.
```

**Bad**:
```yaml
- name: batch_size
  type: number
  description: Batch size  # Too brief, not helpful
```

### Default Values

Provide sensible defaults:

**Good**:
```yaml
- name: timeout_seconds
  type: number
  required: false
  default: 30  # Reasonable default for most API calls
  description: Request timeout in seconds
```

**Bad**:
```yaml
- name: timeout_seconds
  type: number
  required: true  # Forcing users to always specify
  description: Request timeout
```

### Validation Rules

Add validation to catch errors early:

**Good**:
```yaml
- name: percentage
  type: number
  required: true
  description: Success threshold percentage
  validation:
    min: 0
    max: 100
```

**Good**:
```yaml
- name: email
  type: string
  required: true
  description: Notification email address
  validation:
    pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
```

### Template Documentation

Document your templates:

```yaml
# Customer Data Import Template
#
# This template imports customer data from CSV files, validates records,
# and loads them into the database. Failed records are written to an
# error file for manual review.
#
# Required Parameters:
#   - input_file: Path to CSV file with customer data
#   - error_file: Path where rejected records will be written
#
# Optional Parameters:
#   - batch_size: Records per batch (default: 100)
#   - skip_validation: Skip data validation (default: false)
#
# Example Usage:
#   goflow create import-customers --template customer-import \
#     --param input_file="/data/customers.csv" \
#     --param error_file="/data/errors.csv" \
#     --param batch_size=500
#
name: customer-import
description: Import and validate customer data from CSV files
version: "1.0"
# ... parameters and workflow_spec
```

### Error Handling in Templates

Include error handling paths:

```yaml
workflow_spec:
  nodes:
    - id: risky_operation
      type: mcp_tool
      config:
        server: external-server
        tool: risky_call
        parameters:
          url: "{{api_url}}"

    - id: check_success
      type: condition
      config:
        condition: "${risky_operation.status == 'success'}"

    - id: handle_error
      type: transform
      config:
        input: "${risky_operation.error}"
        expression: "jq({error: .message, details: .})"
        output_variable: error_info

    - id: success_end
      type: end
      config:
        return_value: "${risky_operation.result}"

    - id: error_end
      type: end
      config:
        return_value: "${error_info}"
        status: "error"

  edges:
    - from: risky_operation
      to: check_success
    - from: check_success
      to: success_end
      condition: "true"
    - from: check_success
      to: handle_error
      condition: "false"
    - from: handle_error
      to: error_end
```

### Testing Templates

Test templates with different parameter combinations:

```go
func TestTemplateInstantiation(t *testing.T) {
    ctx := context.Background()

    // Test with valid parameters
    t.Run("valid parameters", func(t *testing.T) {
        params := map[string]interface{}{
            "api_url": "https://api.example.com",
            "timeout": 60,
        }
        wf, err := workflow.InstantiateTemplate(ctx, template, params)
        assert.NoError(t, err)
        assert.NotNil(t, wf)
    })

    // Test with missing required parameter
    t.Run("missing required parameter", func(t *testing.T) {
        params := map[string]interface{}{
            "timeout": 60,
        }
        _, err := workflow.InstantiateTemplate(ctx, template, params)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "missing required parameter")
    })

    // Test with invalid type
    t.Run("invalid parameter type", func(t *testing.T) {
        params := map[string]interface{}{
            "api_url": "https://api.example.com",
            "timeout": "sixty",  // Should be number
        }
        _, err := workflow.InstantiateTemplate(ctx, template, params)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid parameter type")
    })

    // Test validation constraints
    t.Run("validation constraint violation", func(t *testing.T) {
        params := map[string]interface{}{
            "api_url": "https://api.example.com",
            "timeout": 500,  // Exceeds max of 300
        }
        _, err := workflow.InstantiateTemplate(ctx, template, params)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "greater than maximum")
    })
}
```

## Reference

### Parameter Types

| Type | Go Type | YAML Example | Description |
|------|---------|--------------|-------------|
| `string` | `string` | `"hello"` | Text values |
| `number` | `int`, `int64`, `float64` | `42`, `3.14` | Numeric values |
| `boolean` | `bool` | `true`, `false` | True/false values |
| `array` | `[]interface{}` | `[1, 2, 3]` | Lists of values |

### Validation Options

#### String Validation

```yaml
validation:
  min_length: 1       # Minimum string length (int)
  max_length: 100     # Maximum string length (int)
  pattern: "^[a-z]+$" # Regular expression (string)
```

#### Number Validation

```yaml
validation:
  min: 0      # Minimum value (number)
  max: 100    # Maximum value (number)
```

#### Array Validation

```yaml
validation:
  min_length: 1   # Minimum array length (int)
  max_length: 10  # Maximum array length (int)
```

### Common Regular Expression Patterns

```yaml
# Email address
pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

# URL (http or https)
pattern: "^https?://.+"

# IPv4 address
pattern: "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"

# Alphanumeric with underscores and hyphens
pattern: "^[a-zA-Z0-9_-]+$"

# ISO 8601 date (YYYY-MM-DD)
pattern: "^\\d{4}-\\d{2}-\\d{2}$"

# Semantic version (1.2.3)
pattern: "^\\d+\\.\\d+\\.\\d+$"

# Absolute file path (Unix)
pattern: "^/.*"

# Absolute file path (Windows)
pattern: "^[A-Za-z]:\\\\"
```

### Template Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `ErrInvalidTemplate` | Template structure is invalid | Check name, version, and parameter definitions |
| `ErrDuplicateParameterName` | Parameter name appears multiple times | Ensure unique parameter names |
| `ErrUndefinedParameter` | `{{param}}` references undefined parameter | Define the parameter or fix typo |
| `ErrMissingRequiredParameter` | Required parameter not provided | Provide the parameter or make it optional |
| `ErrInvalidParameterType` | Value doesn't match declared type | Provide correct type or update declaration |
| `ErrParameterValidation` | Value violates validation constraint | Provide valid value or adjust validation rules |

### Complete Template Example

```yaml
name: complete-example
description: Demonstrates all template features
version: "1.0"

parameters:
  # String with validation
  - name: api_endpoint
    type: string
    required: true
    description: API endpoint URL
    validation:
      pattern: "^https://.+"
      min_length: 10
      max_length: 200

  # Number with range
  - name: timeout_seconds
    type: number
    required: false
    default: 30
    description: Request timeout
    validation:
      min: 1
      max: 300

  # Boolean for feature flag
  - name: enable_caching
    type: boolean
    required: false
    default: false
    description: Enable response caching

  # Array with constraints
  - name: webhook_urls
    type: array
    required: false
    default: []
    description: Webhooks to notify on completion
    validation:
      max_length: 5

workflow_spec:
  nodes:
    - id: start
      type: start

    # Node with parameter substitution
    - id: fetch_data
      type: mcp_tool
      config:
        server: http-server
        tool: fetch
        parameters:
          url: "{{api_endpoint}}"
          timeout: "{{timeout_seconds}}"
        output_variable: response

    # Conditional node
    - id: cache_response
      type: mcp_tool
      condition: "{{enable_caching}}"
      config:
        server: cache-server
        tool: set
        parameters:
          key: "response_cache"
          value: "${response}"

    # Loop over array parameter
    - id: notify_webhooks
      type: loop
      config:
        collection: "{{webhook_urls}}"
        item_variable: webhook_url
        body:
          - send_webhook

    - id: send_webhook
      type: mcp_tool
      config:
        server: http-server
        tool: post
        parameters:
          url: "${webhook_url}"
          body: "${response}"

    - id: end
      type: end
      config:
        return_value: "${response}"

  edges:
    - from: start
      to: fetch_data
    - from: fetch_data
      to: cache_response
      condition: "{{enable_caching}}"
    - from: fetch_data
      to: notify_webhooks
      condition: "!{{enable_caching}}"
    - from: cache_response
      to: notify_webhooks
    - from: notify_webhooks
      to: end
```

## Additional Resources

- [Template System Documentation](template-system.md) - Technical details of the template implementation
- [Template Quick Reference](TEMPLATE_QUICK_REFERENCE.md) - Quick reference for template syntax
- [Template Helpers](TEMPLATE_HELPERS.md) - Helper functions and utilities
- [GoFlow Specification](../specs/goflow-specification.md) - Full workflow specification

## Getting Help

If you need help with templates:

1. Check the built-in templates in `internal/templates/` for examples
2. Review the template system documentation for technical details
3. Test your template with various parameter combinations
4. Use the validation errors to guide corrections

Template creation is an iterative process. Start simple, test thoroughly, and add complexity as needed.
