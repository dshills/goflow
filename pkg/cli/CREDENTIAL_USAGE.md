# GoFlow Credential Management

This document describes the credential management commands for storing and managing MCP server credentials securely.

## Overview

The `goflow credential` commands allow you to securely store credentials for MCP servers. Credentials are stored separately from workflow definitions, ensuring workflows can be shared without exposing secrets.

**Current Implementation**: In-memory credential store (credentials are lost when the process exits)
**Future Implementation**: System keyring integration for persistent, secure credential storage

## Commands

### credential add

Add credentials for an MCP server.

#### Usage

```bash
goflow credential add <server-id> [flags]
```

#### Flags

- `--env KEY=VALUE` - Add environment variable (can be specified multiple times)
- `--credential-ref NAME` - Reference to a named credential
- `--help` - Show help for this command

#### Examples

**Add environment variable credentials:**
```bash
goflow credential add myserver --env API_KEY=secret123 --env TOKEN=abc456
```

**Add a credential reference:**
```bash
goflow credential add myserver --credential-ref aws-profile-prod
```

**Mix environment variables and credential reference:**
```bash
goflow credential add myserver --env DEBUG=true --credential-ref oauth-token
```

#### Validation

- Server ID must contain only letters, numbers, dashes, and underscores
- At least one of `--env` or `--credential-ref` must be provided
- Environment variables must follow `KEY=VALUE` format
- Environment variable keys cannot be empty

### credential list

List all stored credentials (shows server IDs and credential types, but not actual secret values).

#### Usage

```bash
goflow credential list
```

#### Output Format

The command displays a table with:
- **SERVER ID**: The server identifier
- **ENV VARS**: Environment variable keys (or count if many)
- **CREDENTIAL REF**: Named credential reference
- **TYPE**: Credential type (Environment, Reference, or Mixed)

#### Example Output

```
SERVER ID      ENV VARS         CREDENTIAL REF    TYPE
─────────      ────────         ──────────────    ────
test-server    API_KEY, TOKEN   -                 Environment
my-api         -                oauth-token       Reference
mixed-server   DEBUG            aws-profile       Mixed
many-server    4 keys           -                 Environment
```

**Note**: Secret values are never displayed for security reasons.

### credential remove

Remove stored credentials for a server.

#### Usage

```bash
goflow credential remove <server-id>
```

#### Example

```bash
goflow credential remove myserver
```

This removes all credentials (environment variables and credential references) for the specified server.

## Integration with Server Configuration

Credentials work in conjunction with the `goflow server` commands:

1. **Register a server:**
   ```bash
   goflow server add myserver npx @modelcontextprotocol/server-example
   ```

2. **Add credentials for the server:**
   ```bash
   goflow credential add myserver --env API_KEY=secret123
   ```

3. **The workflow execution will automatically use the stored credentials**

## Security Model

### Current Implementation (In-Memory)

- Credentials are stored in memory during the process lifetime
- Credentials are lost when the process exits
- This is suitable for testing and development

### Future Implementation (Keyring)

- Credentials will be stored in the system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- Credentials persist across sessions
- OS-level encryption and access control
- Credentials never written to workflow files
- Workflows remain shareable without secrets

## Best Practices

1. **Never commit credentials to version control**
   - Credentials are stored separately from workflow YAML files
   - Workflow files can be safely committed

2. **Use credential references for shared credentials**
   - Use `--credential-ref` for credentials managed by external systems (AWS profiles, OAuth tokens, etc.)

3. **Minimize credential scope**
   - Only add credentials for servers that require them
   - Use read-only credentials when possible

4. **Rotate credentials regularly**
   - Remove old credentials with `credential remove`
   - Add updated credentials with `credential add`

## Error Handling

### Common Errors

**Invalid server ID:**
```
Error: invalid server ID: invalid@server! (must contain only letters, numbers, dashes, and underscores)
```

**No credentials provided:**
```
Error: must provide at least one of --env or --credential-ref
```

**Invalid environment variable format:**
```
Error: invalid environment variable format: INVALID (expected KEY=VALUE)
```

**Server not found on remove:**
```
Error: no credentials found for server: missing-server
```

## Programmatic Access

The credential store can be accessed programmatically within the CLI package:

```go
import "github.com/dshills/goflow/pkg/cli"

// Get credentials for a server
cred := cli.GetCredential("myserver")
if cred != nil {
    // Access environment variables
    for key, value := range cred.EnvVars {
        fmt.Printf("%s=%s\n", key, value)
    }

    // Access credential reference
    if cred.CredentialRef != "" {
        fmt.Printf("Credential Ref: %s\n", cred.CredentialRef)
    }
}
```

**Note**: `GetCredential` returns a copy of the credential to prevent external modification.

## Future Enhancements

1. **System Keyring Integration**
   - Persistent storage using OS keyring
   - Per-user credential isolation
   - OS-level encryption

2. **Credential Encryption**
   - Optional encryption at rest
   - Key derivation from user password

3. **Credential Validation**
   - Test credentials before storage
   - Verify server accessibility with provided credentials

4. **Credential Templates**
   - Pre-configured credential templates for common services
   - OAuth flow integration

5. **Credential Audit Log**
   - Track credential access and modifications
   - Security event logging
