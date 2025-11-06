# Export Implementation Requirements

## Quick Reference

**File to Create**: `/Users/dshills/Development/projects/goflow/pkg/workflow/export.go`

**Test File**: `/Users/dshills/Development/projects/goflow/tests/integration/workflow_export_test.go` (701 lines, 11 test functions)

## Required Public API

```go
package workflow

// Export exports a workflow to YAML bytes with credentials stripped.
// Returns error if workflow is nil.
func Export(workflow *Workflow) ([]byte, error)

// ExportFile exports a workflow to a YAML file with credentials stripped.
// Returns error if workflow is nil or file cannot be written.
func ExportFile(workflow *Workflow, filePath string) error
```

## Core Implementation Logic

### 1. Credential Detection Patterns (Case-Insensitive)

**MUST Strip** if env var key contains:
```go
var sensitivePatterns = []string{
    "KEY",          // API_KEY, SECRET_KEY, ACCESS_KEY
    "SECRET",       // CLIENT_SECRET, API_SECRET
    "TOKEN",        // AUTH_TOKEN, BEARER_TOKEN, OAUTH_TOKEN
    "PASSWORD",     // PASSWORD, DB_PASSWORD
    "CREDENTIAL",   // CREDENTIAL, CREDENTIALS
    "PRIVATE",      // PRIVATE_KEY
    "PASSPHRASE",   // PASSPHRASE
    "AUTH",         // (when part of credential-related key)
    "BEARER",       // BEARER_TOKEN
    "OAUTH",        // OAUTH_TOKEN, OAUTH_SECRET
}

// Special cases
var sensitiveURLPatterns = []string{
    "DATABASE_URL",
    "DB_URL",
    "CONNECTION_STRING",
    "CONN_STR",
}
```

**MUST Preserve** (examples):
```go
var nonSensitivePatterns = []string{
    "HOST", "PORT", "ENDPOINT", "REGION", "NAMESPACE",
    "LOG_LEVEL", "LOG_FORMAT", "DEBUG", "VERBOSE",
    "TIMEOUT", "MAX_RETRIES", "POOL_SIZE", "BUFFER_SIZE",
    "SERVICE_NAME", "APP_NAME", "VERSION",
}
```

### 2. Credential Reference Handling

```go
// Replace credential references with placeholder
if serverConfig.CredentialRef != "" {
    serverConfig.CredentialRef = "<CREDENTIAL_REF_REQUIRED>"
}
```

### 3. Export Algorithm

```go
func Export(workflow *Workflow) ([]byte, error) {
    // 1. Validate input
    if workflow == nil {
        return nil, errors.New("workflow cannot be nil")
    }

    // 2. Deep copy workflow to avoid modifying original
    exportWf := deepCopy(workflow)

    // 3. Strip credentials from all server configs
    for i, serverConfig := range exportWf.ServerConfigs {
        exportWf.ServerConfigs[i] = stripCredentials(serverConfig)
    }

    // 4. Convert to YAML using existing ToYAML function
    yamlBytes, err := ToYAML(exportWf)
    if err != nil {
        return nil, fmt.Errorf("failed to convert to YAML: %w", err)
    }

    // 5. Add credential placeholder comments
    yamlBytes = addCredentialComments(yamlBytes)

    return yamlBytes, nil
}
```

### 4. Helper Functions

```go
// isSensitiveEnvVar checks if environment variable key is sensitive
func isSensitiveEnvVar(key string) bool {
    upperKey := strings.ToUpper(key)

    // Check against sensitive patterns
    for _, pattern := range sensitivePatterns {
        if strings.Contains(upperKey, pattern) {
            return true
        }
    }

    // Check exact matches for URL patterns
    for _, pattern := range sensitiveURLPatterns {
        if upperKey == pattern {
            return true
        }
    }

    return false
}

// stripCredentials removes sensitive data from server config
func stripCredentials(config *ServerConfig) *ServerConfig {
    // Create new config (don't modify original)
    stripped := &ServerConfig{
        ID:        config.ID,
        Name:      config.Name,
        Command:   config.Command,
        Args:      make([]string, len(config.Args)),
        Transport: config.Transport,
        Env:       make(map[string]string),
    }

    copy(stripped.Args, config.Args)

    // Filter environment variables
    for key, value := range config.Env {
        if !isSensitiveEnvVar(key) {
            stripped.Env[key] = value
        }
    }

    // Replace credential reference with placeholder
    if config.CredentialRef != "" {
        stripped.CredentialRef = "<CREDENTIAL_REF_REQUIRED>"
    }

    return stripped
}

// deepCopy creates a deep copy of workflow for safe mutation
func deepCopy(wf *Workflow) *Workflow {
    // Implementation: Marshal to JSON and back, or manual field-by-field copy
    // Must copy: Variables, ServerConfigs, Nodes, Edges
}

// addCredentialComments injects helpful comments into YAML
func addCredentialComments(yamlBytes []byte) []byte {
    // Add comment above servers section if credentials were stripped
    // Example: "# CREDENTIALS REMOVED: Configure before use"
}
```

## Test Coverage Requirements

### Must Pass All 11 Tests:

1. ✅ **TestExport_WorkflowWithInlineCredentials** - Strip API keys, secrets, tokens, passwords
2. ✅ **TestExport_WorkflowWithCredentialReferences** - Replace keyring refs with placeholders
3. ✅ **TestExport_WorkflowWithNoCredentials** - Preserve non-credential workflows
4. ✅ **TestExport_ValidYAML** - Produce valid YAML output
5. ✅ **TestExport_WorkflowStructurePreserved** - Keep nodes, edges, variables intact
6. ✅ **TestExport_NonSensitiveDataPreserved** - Preserve config like HOST, PORT
7. ✅ **TestExport_RoundTripImport** - Re-import exported YAML successfully
8. ✅ **TestExport_VariousCredentialPatterns** - Detect 10+ credential patterns
9. ✅ **TestExport_ToFile** - Write to filesystem
10. ✅ **TestExport_NilWorkflow** - Error handling for nil input
11. ✅ **TestExport_EmptyWorkflow** - Handle minimal valid workflows

## Critical Security Requirements

### NEVER Export:
- ❌ Inline API keys (API_KEY, ACCESS_KEY)
- ❌ Secrets (SECRET, CLIENT_SECRET)
- ❌ Tokens (AUTH_TOKEN, BEARER_TOKEN, OAUTH_TOKEN)
- ❌ Passwords (PASSWORD, DB_PASSWORD)
- ❌ Private keys (PRIVATE_KEY)
- ❌ Database URLs with embedded credentials
- ❌ Keyring paths (replace with placeholder)

### ALWAYS Preserve:
- ✅ Server commands and arguments
- ✅ Transport type (stdio, sse, http)
- ✅ Non-credential env vars (HOST, PORT, LOG_LEVEL)
- ✅ All workflow nodes and their configurations
- ✅ All workflow edges and connections
- ✅ All workflow variables
- ✅ Workflow metadata

## Implementation Checklist

- [ ] Create `pkg/workflow/export.go`
- [ ] Implement `isSensitiveEnvVar()` with pattern matching
- [ ] Implement `stripCredentials()` for ServerConfig
- [ ] Implement `deepCopy()` for safe workflow cloning
- [ ] Implement `Export()` main function
- [ ] Implement `ExportFile()` file writer
- [ ] Implement `addCredentialComments()` for YAML comments
- [ ] Run tests: `go test ./tests/integration -run TestExport -v`
- [ ] Verify all 11 tests pass
- [ ] Manual security review - no credential leakage
- [ ] Test with real workflows containing test credentials
- [ ] Update documentation with export examples

## Validation Commands

```bash
# Run all export tests
go test ./tests/integration -run TestExport -v

# Run specific test during development
go test ./tests/integration -run TestExport_WorkflowWithInlineCredentials -v

# Check test count
go test ./tests/integration -run TestExport -list=.

# Verify no credentials in output (manual test)
go test ./tests/integration -run TestExport_WorkflowWithInlineCredentials -v 2>&1 | grep -i "api_key"
# Should NOT appear in exported YAML
```

## Edge Cases to Handle

1. **Nil Workflow**: Return error "workflow cannot be nil"
2. **Empty Env Map**: Don't fail, just skip credential stripping
3. **No ServerConfigs**: Export workflow as-is
4. **Mixed Case Keys**: Use case-insensitive matching (API_Key, api_key, API_KEY)
5. **Special Characters in Values**: Properly escape YAML
6. **Large Workflows**: Handle efficiently (don't load all in memory if huge)
7. **File Write Errors**: Proper error messages with file path

## Performance Considerations

- **Deep Copy**: Use efficient method (consider `encoding/gob` or manual copy)
- **Pattern Matching**: Compile patterns once, not per key
- **Memory**: Don't hold multiple copies unnecessarily
- **File I/O**: Use buffered writer for large workflows

## Success Criteria

Export implementation is complete when:
1. ✅ All 11 tests pass without modification
2. ✅ No credentials visible in exported YAML (manual verification)
3. ✅ Exported YAML re-imports successfully
4. ✅ Non-sensitive config preserved accurately
5. ✅ Error handling works for edge cases
6. ✅ Performance acceptable (<100ms for typical workflow)
7. ✅ Code review confirms no security issues

## Example Test Run (Expected After Implementation)

```bash
$ go test ./tests/integration -run TestExport -v

=== RUN   TestExport_WorkflowWithInlineCredentials
--- PASS: TestExport_WorkflowWithInlineCredentials (0.01s)
=== RUN   TestExport_WorkflowWithCredentialReferences
--- PASS: TestExport_WorkflowWithCredentialReferences (0.00s)
=== RUN   TestExport_WorkflowWithNoCredentials
--- PASS: TestExport_WorkflowWithNoCredentials (0.00s)
=== RUN   TestExport_ValidYAML
--- PASS: TestExport_ValidYAML (0.01s)
=== RUN   TestExport_WorkflowStructurePreserved
--- PASS: TestExport_WorkflowStructurePreserved (0.01s)
=== RUN   TestExport_NonSensitiveDataPreserved
--- PASS: TestExport_NonSensitiveDataPreserved (0.00s)
=== RUN   TestExport_RoundTripImport
--- PASS: TestExport_RoundTripImport (0.01s)
=== RUN   TestExport_VariousCredentialPatterns
--- PASS: TestExport_VariousCredentialPatterns (0.02s)
=== RUN   TestExport_ToFile
--- PASS: TestExport_ToFile (0.00s)
=== RUN   TestExport_NilWorkflow
--- PASS: TestExport_NilWorkflow (0.00s)
=== RUN   TestExport_EmptyWorkflow
--- PASS: TestExport_EmptyWorkflow (0.00s)
PASS
ok      github.com/dshills/goflow/tests/integration    0.108s
```

## Related Files for Reference

- **Parser**: `/Users/dshills/Development/projects/goflow/pkg/workflow/parser.go` - Has `ToYAML()` function
- **Server Config**: `/Users/dshills/Development/projects/goflow/pkg/workflow/server_config.go` - ServerConfig struct
- **Workflow**: `/Users/dshills/Development/projects/goflow/pkg/workflow/workflow.go` - Workflow struct
- **Test Fixture**: `/Users/dshills/Development/projects/goflow/internal/testutil/fixtures/simple-workflow.yaml`
