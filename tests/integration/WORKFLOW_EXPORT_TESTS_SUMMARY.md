# Workflow Export Integration Tests Summary

## Overview

Created comprehensive integration tests for workflow export functionality with credential stripping in `/Users/dshills/Development/projects/goflow/tests/integration/workflow_export_test.go`.

These tests follow **TDD (Test-Driven Development)** principles and are designed to **FAIL initially** until the export implementation is completed.

## Test Coverage

### 1. Credential Stripping Tests

#### TestExport_WorkflowWithInlineCredentials
- **Purpose**: Verify inline credentials in server config environment variables are stripped
- **Tested Patterns**: API_KEY, API_SECRET, API_TOKEN, PASSWORD
- **Expected Behavior**:
  - Sensitive env vars removed from export
  - Non-sensitive env vars (HOST, PORT) preserved
  - Placeholder comments/instructions added to YAML

#### TestExport_VariousCredentialPatterns
- **Purpose**: Comprehensive test of credential pattern detection
- **Covers 19 different patterns**:
  - **Should Strip**: api_key, secret_key, access_token, password, auth_token, bearer_token, client_secret, private_key, credential, passphrase
  - **Should Preserve**: host, port, service_name, log_level, timeout, max_retries, endpoint, region, namespace
- **Pattern Matching**: Tests both exact matches and case variations

#### TestExport_WorkflowWithCredentialReferences
- **Purpose**: Verify credential references (keyring://) are replaced with placeholders
- **Expected Behavior**:
  - credential_ref field replaced with `<CREDENTIAL_REF_REQUIRED>`
  - Original keyring reference not exposed in export
  - Placeholder indicates manual configuration needed

### 2. Workflow Structure Preservation Tests

#### TestExport_WorkflowStructurePreserved
- **Purpose**: Ensure workflow logic and structure remain intact after export
- **Validates**:
  - All node types preserved (start, mcp_tool, transform, end)
  - Variable definitions maintained
  - Edge connections preserved
  - Node configurations (non-sensitive) intact
  - Correct node count and types

#### TestExport_NonSensitiveDataPreserved
- **Purpose**: Verify non-credential configuration is preserved accurately
- **Tests**:
  - Server command and args preserved
  - Transport type preserved
  - Non-sensitive env vars (REDIS_HOST, SERVICE_NAME, LOG_LEVEL) maintained
  - Mix of sensitive/non-sensitive vars correctly filtered

### 3. YAML Validity Tests

#### TestExport_ValidYAML
- **Purpose**: Ensure exported YAML is syntactically valid
- **Validates**:
  - YAML parses without errors
  - Required fields present (version, name, nodes, edges)
  - No malformed YAML indicators
  - Non-empty output

#### TestExport_RoundTripImport
- **Purpose**: Verify exported YAML can be successfully re-imported
- **Workflow**:
  1. Create original workflow with nodes, variables, edges
  2. Export to YAML
  3. Re-import from exported YAML
  4. Verify structure matches (name, node count, variable count, edge count)
  5. Validate reimported workflow passes validation
- **Critical for**: Ensuring export doesn't corrupt workflow definition

### 4. Edge Cases and Error Handling

#### TestExport_WorkflowWithNoCredentials
- **Purpose**: Workflows without credentials export unchanged
- **Validates**: Non-credential workflows are not modified unnecessarily

#### TestExport_NilWorkflow
- **Purpose**: Proper error handling for nil workflow
- **Expected**: Returns error with message "workflow cannot be nil"

#### TestExport_EmptyWorkflow
- **Purpose**: Minimal valid workflow exports successfully
- **Tests**: Workflow with only start → end nodes produces valid YAML

#### TestExport_ToFile
- **Purpose**: Export directly to file system
- **Validates**:
  - File created at specified path
  - File contains valid YAML
  - File content matches expected workflow

## Implementation Requirements

Based on the tests, the following functions must be implemented in `pkg/workflow/`:

### Core Export Functions

```go
// Export exports a workflow to YAML bytes with credentials stripped
func Export(workflow *Workflow) ([]byte, error)

// ExportFile exports a workflow to a YAML file with credentials stripped
func ExportFile(workflow *Workflow, filePath string) error
```

### Credential Detection Logic

The implementation must detect and strip the following sensitive patterns in environment variables:

**Sensitive Patterns (case-insensitive)**:
- `*KEY` - API_KEY, SECRET_KEY, ACCESS_KEY
- `*SECRET*` - CLIENT_SECRET, API_SECRET
- `*TOKEN*` - AUTH_TOKEN, BEARER_TOKEN, OAUTH_TOKEN, ACCESS_TOKEN
- `*PASSWORD*` - PASSWORD, DB_PASSWORD
- `*CREDENTIAL*` - CREDENTIAL, CREDENTIALS
- `*AUTH*` - (when combined with other patterns)
- `*PRIVATE*` - PRIVATE_KEY, PRIVATE_TOKEN
- Database URLs with embedded credentials: `DATABASE_URL`, `DB_URL`, `*_CONNECTION_STRING`

**Non-Sensitive Patterns (should preserve)**:
- Configuration: HOST, PORT, ENDPOINT, REGION, NAMESPACE
- Logging: LOG_LEVEL, LOG_FORMAT, DEBUG
- Performance: TIMEOUT, MAX_RETRIES, POOL_SIZE
- Service info: SERVICE_NAME, APP_NAME, VERSION

### Credential Reference Handling

```go
// Replace credential references with placeholders
if serverConfig.CredentialRef != "" {
    serverConfig.CredentialRef = "<CREDENTIAL_REF_REQUIRED>"
}
```

### YAML Comment Injection

Add helpful comments to exported YAML to guide users:

```yaml
servers:
  - id: "api-server"
    # CREDENTIALS REMOVED: Configure credential_ref or environment variables before use
    env:
      HOST: "api.example.com"
      PORT: "8080"
```

### Export Process Flow

1. **Clone Workflow**: Deep copy to avoid modifying original
2. **Strip Credentials**: Iterate through all ServerConfigs
   - Remove sensitive env vars based on pattern matching
   - Replace credential references with placeholders
   - Preserve all non-sensitive configuration
3. **Convert to YAML**: Use existing `ToYAML()` function as base
4. **Add Comments**: Inject helpful placeholder comments
5. **Return/Write**: Return bytes or write to file

### Helper Functions Needed

```go
// isSensitiveEnvVar determines if an env var key contains credentials
func isSensitiveEnvVar(key string) bool

// stripCredentialsFromServerConfig removes sensitive data from server config
func stripCredentialsFromServerConfig(config *ServerConfig) *ServerConfig

// addCredentialComments injects helpful comments into YAML
func addCredentialComments(yamlBytes []byte) []byte
```

## Test Execution

### Running the Tests

```bash
# Run all export tests (will fail until implementation complete)
go test ./tests/integration -run TestExport -v

# Run specific test
go test ./tests/integration -run TestExport_WorkflowWithInlineCredentials -v

# Run with verbose output to see failure details
go test ./tests/integration -run TestExport -v -count=1
```

### Expected Initial Behavior

All tests will **FAIL** with errors like:
```
undefined: workflow.Export
undefined: workflow.ExportFile
```

This is intentional and follows TDD methodology.

### Definition of Done

Tests will **PASS** when:
1. `workflow.Export()` function exists and correctly strips credentials
2. `workflow.ExportFile()` function exists and writes to filesystem
3. All credential patterns are detected and removed
4. Non-sensitive data is preserved
5. Exported YAML is valid and re-importable
6. Proper error handling for edge cases

## Security Considerations

### Critical Requirements

1. **Never Export Secrets**: Inline credentials, API keys, tokens, passwords must be completely removed
2. **Credential References**: Replace keyring references with placeholders, not expose paths
3. **Pattern Matching**: Must be comprehensive to catch all credential variations
4. **Default to Safe**: If uncertain whether a value is sensitive, err on side of caution
5. **Audit Trail**: Consider logging what was stripped (without logging the actual values)

### Credential Patterns to Detect

The implementation must use case-insensitive matching for:
- Exact matches: `PASSWORD`, `API_KEY`, `SECRET`, `TOKEN`
- Substring matches: `*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, `*_CREDENTIAL`
- URL patterns: Database connection strings with embedded credentials
- File paths: Private key file paths, certificate paths with keys

### Safe Sharing Guarantee

After export, the YAML file should be:
- Safe to commit to version control
- Safe to share with team members
- Safe to use in documentation
- Safe to include in tutorials/examples

## Related Files

- **Implementation Target**: `/Users/dshills/Development/projects/goflow/pkg/workflow/export.go` (to be created)
- **Existing Parser**: `/Users/dshills/Development/projects/goflow/pkg/workflow/parser.go` (has `ToYAML()` function to extend)
- **Server Config**: `/Users/dshills/Development/projects/goflow/pkg/workflow/server_config.go` (credential_ref field defined here)
- **Test Fixtures**: `/Users/dshills/Development/projects/goflow/internal/testutil/fixtures/` (simple-workflow.yaml available)

## Test Statistics

- **Total Tests**: 12 comprehensive test functions
- **Test Scenarios**: 19+ credential pattern variations
- **Lines of Test Code**: ~690 lines
- **Coverage Areas**:
  - Credential stripping
  - Structure preservation
  - YAML validity
  - Round-trip compatibility
  - Error handling
  - File I/O

## Next Steps for Implementation

1. **Create export.go**: New file in `pkg/workflow/`
2. **Implement isSensitiveEnvVar()**: Pattern matching function
3. **Implement stripCredentialsFromServerConfig()**: Server config sanitizer
4. **Implement Export()**: Main export function
5. **Implement ExportFile()**: File writer wrapper
6. **Add credential placeholder comments**: YAML comment injection
7. **Run tests**: Iterate until all tests pass
8. **Security review**: Verify no credential leakage paths
9. **Integration testing**: Test with real workflows containing actual (test) credentials
10. **Documentation**: Add examples of export usage to README

## Example Usage (Once Implemented)

```go
// Load workflow with credentials
wf, _ := workflow.ParseFile("my-workflow.yaml")

// Export with credentials stripped
exportedYAML, _ := workflow.Export(wf)

// Save to shareable file
workflow.ExportFile(wf, "my-workflow-shareable.yaml")

// Original workflow unchanged, exported version has credentials removed
```

## Success Criteria

✅ All 12 test functions pass
✅ No credentials in exported YAML
✅ Non-sensitive config preserved
✅ Exported YAML is valid and re-importable
✅ Proper error handling for edge cases
✅ File export works correctly
✅ Security review confirms no credential leakage

## Notes

- Tests use `t.TempDir()` for file operations (automatic cleanup)
- Tests are independent and can run in any order
- Each test has clear documentation of expected behavior
- Tests validate both positive cases (correct stripping) and negative cases (preservation)
- Round-trip test ensures no data corruption during export/import cycle
