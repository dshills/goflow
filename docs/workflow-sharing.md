# Workflow Sharing Guide

## Overview

GoFlow workflows are designed to be safely shared between team members, published in workflow registries, or distributed as reusable templates. The workflow sharing system automatically strips sensitive credentials and provides clear guidance for setting up imported workflows.

This guide covers best practices for exporting, sharing, and importing GoFlow workflows while maintaining security and ease of use.

## Exporting Workflows

### Basic Export

Export a workflow with credentials stripped for safe sharing:

```bash
# Export to stdout (for piping or viewing)
goflow export my-workflow.yaml

# Export to a file
goflow export my-workflow.yaml --output shared-workflow.yaml

# Export from workflows directory
goflow export ~/.goflow/workflows/my-workflow.yaml -o exported.yaml
```

### What Gets Exported

When you export a workflow, GoFlow creates a sanitized copy that includes:

- **Workflow structure**: All nodes, edges, and execution flow
- **Non-sensitive configuration**: Server commands, transport settings
- **Variable definitions**: Workflow variables with types and defaults
- **Metadata**: Name, version, description, tags
- **Non-sensitive environment variables**: HOST, PORT, LOG_LEVEL, etc.

### What Gets Stripped

For security, the following sensitive information is automatically removed:

- **API keys and tokens**: Any environment variable containing KEY, TOKEN, SECRET
- **Passwords and credentials**: Variables with PASSWORD, PASSPHRASE, CREDENTIAL
- **Authentication tokens**: BEARER, AUTH, OAUTH patterns
- **Private keys**: Variables containing PRIVATE, CLIENT_SECRET
- **Database credentials**: DATABASE_URL, CONNECTION_STRING, DSN
- **Credential references**: Replaced with `<CREDENTIAL_REF_REQUIRED>` placeholder

### Export Warning Header

Exported workflows include a warning header:

```yaml
# CREDENTIAL WARNING: This workflow has been exported for sharing.
# Sensitive credentials have been removed and must be configured before use.
# Please set up credential references for servers marked with <CREDENTIAL_REF_REQUIRED>.

version: "1.0"
name: my-workflow
# ... rest of workflow
```

## Security Considerations

### What Credentials Are Removed

GoFlow's export system uses pattern matching to identify sensitive environment variables. The following patterns trigger credential stripping:

- KEY, SECRET, TOKEN
- PASSWORD, PASSPHRASE
- CREDENTIAL, AUTH, BEARER
- PRIVATE, CLIENT_SECRET
- DATABASE (when combined with URL)
- CONN (when combined with STRING)
- DSN (Data Source Name)
- OAUTH

### What Stays in Exported Files

Non-sensitive configuration is preserved for usability:

- **Server commands**: The command and arguments to start MCP servers
- **Transport type**: stdio, sse, or http
- **Non-sensitive env vars**: Configuration like HOST, PORT, SERVICE_NAME
- **Workflow logic**: All nodes, edges, conditions, and transformations
- **Variable definitions**: Workflow-scoped variables (not credential values)

### Best Practices for Sensitive Data

1. **Never commit credentials to workflow files**
   - Always use credential references or environment variables
   - Let GoFlow's export system handle sanitization

2. **Use descriptive server IDs**
   - Helps recipients understand what credentials are needed
   - Example: `github-api`, `slack-notifier`, `database-prod`

3. **Document required credentials**
   - Add comments in the workflow description
   - Create a README with credential setup instructions

4. **Version your workflows**
   - Use semantic versioning (1.0.0, 1.1.0)
   - Document breaking changes in credential requirements

5. **Test exported workflows**
   - Export, import, and validate before sharing
   - Ensure all placeholders are clear and actionable

## Importing Workflows

### Basic Import

Import a shared workflow from a file:

```bash
# Import a workflow
goflow import /path/to/workflow.yaml

# Import with verbose output
goflow import ./shared-workflow.yaml --verbose
```

### Import Process

When you import a workflow, GoFlow performs several validation steps:

1. **File validation**: Checks that the workflow file exists and is readable
2. **Version compatibility**: Ensures the workflow version is supported
3. **Server validation**: Checks that referenced MCP servers are registered
4. **Credential detection**: Identifies missing credentials and placeholders
5. **Workflow validation**: Validates structure, nodes, and edges
6. **Installation**: Copies to `~/.goflow/workflows/<workflow-name>.yaml`

### Import Success

A successful import displays:

```
✓ Workflow imported successfully
✓ Workflow validation passed

✓ Workflow 'data-pipeline' imported successfully
  Location: /Users/yourname/.goflow/workflows/data-pipeline.yaml

Next steps:
  1. Edit the workflow: goflow edit data-pipeline
  2. Validate: goflow validate data-pipeline
  3. Execute: goflow run data-pipeline
```

### Handling Missing Servers

If the workflow references servers not in your registry:

```
✗ Workflow references missing servers

Missing servers:
  - github-api
  - slack-notifier

Please register these servers before importing:
  goflow server add <server-id> <command> [args...]
```

**Resolution**: Register the missing servers first, then re-import:

```bash
# Register the required servers
goflow server add github-api github-mcp-server
goflow server add slack-notifier slack-mcp-server --transport sse

# Retry the import
goflow import workflow.yaml
```

### Handling Credential Placeholders

If the workflow contains credential placeholders:

```
⚠ Workflow contains credential placeholders

Servers with placeholders:
  - github-api
  - database-prod

You will need to configure credentials before execution.
```

**Note**: This is a warning, not an error. The workflow is still imported successfully, but you must configure credentials before running it.

## Credential Management

### Understanding Credential Storage

GoFlow stores credentials securely in the system keyring (or in-memory during development). Credentials are:

- **Never written to workflow files**: Export strips them automatically
- **Referenced by server ID**: Each server has its own credential set
- **Type-safe**: Support environment variables and named credential references
- **Isolated**: Each server's credentials are independent

### Adding Credentials After Import

Once you've imported a workflow, configure credentials for servers that need them:

```bash
# Add environment variable credentials
goflow credential add github-api \
  --env GITHUB_TOKEN=ghp_yourtokenhere \
  --env GITHUB_ORG=myorg

# Add a credential reference (for system-managed credentials)
goflow credential add database-prod \
  --credential-ref aws-rds-prod-credentials

# Mix environment variables and references
goflow credential add api-server \
  --env API_BASE_URL=https://api.example.com \
  --env DEBUG=false \
  --credential-ref oauth-token-prod
```

### Listing Stored Credentials

View which servers have credentials configured:

```bash
goflow credential list
```

Output:

```
SERVER ID       ENV VARS         CREDENTIAL REF           TYPE
─────────       ────────         ──────────────           ────
github-api      GITHUB_TOKEN     -                        Environment
database-prod   -                aws-rds-prod-credentials Reference
api-server      API_BASE_URL     oauth-token-prod         Mixed

Note: Secret values are not displayed for security.
```

### Removing Credentials

Remove credentials when no longer needed:

```bash
goflow credential remove github-api
```

### Security Model

GoFlow's credential system provides security through:

1. **Separation of concerns**: Workflows define logic, credentials are stored separately
2. **Keyring integration**: Future releases will use OS keyring (Keychain, Windows Credential Manager, etc.)
3. **In-memory only**: Current implementation uses secure in-memory storage (not persisted to disk)
4. **No export**: Credentials are never included in exported workflows
5. **User-scoped**: Credentials are local to your machine

## Distribution Strategies

### Version Control

**Best for**: Team collaboration, change tracking, code review

```bash
# In your repository
git clone https://github.com/yourorg/workflows.git
cd workflows

# Import team workflows
goflow import ./team/data-pipeline.yaml
goflow import ./team/notification-workflow.yaml

# Set up credentials (not in version control!)
goflow credential add github-api --env GITHUB_TOKEN=$GITHUB_TOKEN
```

**Tips**:
- Create a `.gitignore` for credential files
- Include a `CREDENTIALS_REQUIRED.md` documenting what credentials are needed
- Use workflow templates for common patterns
- Tag releases with semantic versions

### Workflow Registries

**Best for**: Public sharing, community workflows, reusable templates

Coming in future releases: GoFlow will support workflow registries for discovering and installing community workflows.

Planned features:
- `goflow registry search <keyword>`
- `goflow registry install <workflow-name>`
- `goflow registry publish <workflow-file>`
- Version compatibility checking
- Dependency resolution

### Direct File Sharing

**Best for**: One-off sharing, quick demos, prototypes

```bash
# Share via email, Slack, file sharing
# Recipient imports directly
goflow import ~/Downloads/workflow.yaml
```

### Workflow Templates

**Best for**: Parameterized workflows, organizational standards, reusable patterns

Use GoFlow's template system for workflows with customizable parameters:

```bash
# Create a template from a workflow
# (See docs/template-system.md for details)

# Instantiate a template
goflow template instantiate my-template \
  --param api_endpoint=https://api.example.com \
  --param retry_count=3 \
  --output customized-workflow.yaml
```

See [Template System Documentation](template-system.md) for creating and using workflow templates.

## Version Compatibility

### Workflow Versions

GoFlow workflows use semantic versioning:

```yaml
version: "1.0"
```

### Compatibility Checks

During import, GoFlow validates version compatibility:

- **Supported versions**: 1.0 (current)
- **Future versions**: May introduce breaking changes in 2.0+
- **Backward compatibility**: GoFlow strives to maintain compatibility within major versions

### Version Mismatch Errors

If you import a workflow with an incompatible version:

```
✗ Incompatible workflow version
  Workflow version: 2.0
  Supported versions: [1.0]
```

**Resolution**:
- Update GoFlow to a compatible version: `go install github.com/dshills/goflow@latest`
- Or migrate the workflow to the supported version (may require manual changes)

### Migration Between Versions

When GoFlow releases a new major version, migration guides will be provided:

- Breaking changes documentation
- Automated migration tools (when possible)
- Side-by-side version support during transition periods

## Troubleshooting

### Common Import Errors

#### Error: Workflow file not found

```
✗ workflow file not found: workflow.yaml
```

**Cause**: File path is incorrect or file doesn't exist

**Fix**: Verify the file path and try again
```bash
ls -la /path/to/workflow.yaml
goflow import /absolute/path/to/workflow.yaml
```

#### Error: Workflow already exists

```
✗ workflow already exists: my-workflow

Location: /Users/yourname/.goflow/workflows/my-workflow.yaml
Use a different name or remove the existing workflow first
```

**Fix**: Remove the existing workflow or rename it
```bash
# Remove the existing workflow
rm ~/.goflow/workflows/my-workflow.yaml

# Or rename it
mv ~/.goflow/workflows/my-workflow.yaml ~/.goflow/workflows/my-workflow-old.yaml

# Then retry import
goflow import workflow.yaml
```

#### Error: Failed to load server config

**Cause**: Server configuration file is corrupted or missing

**Fix**: Check or recreate your server configuration
```bash
# Check if servers.yaml exists
cat ~/.goflow/servers.yaml

# Re-register servers if needed
goflow server add myserver command [args...]
```

### Common Execution Errors

#### Error: Missing credentials

When running a workflow without configured credentials:

```
✗ Execution failed: server 'github-api' requires credentials
```

**Fix**: Add credentials for the required server
```bash
goflow credential add github-api --env GITHUB_TOKEN=your_token
```

#### Error: Server not found

```
✗ Execution failed: server 'unknown-server' not found in registry
```

**Fix**: Ensure the server is registered
```bash
# List registered servers
goflow server list

# Register the missing server
goflow server add unknown-server command [args...]
```

### Validation Issues

#### Server Validation Failures

If imported workflow references invalid servers:

```bash
# Test server connectivity
goflow server test github-api

# Check server configuration
goflow server list --verbose
```

#### Workflow Structure Errors

If validation fails after import:

```
✗ Workflow validation failed
  Error: node 'process-data' references undefined variable 'missing_var'
```

**Fix**: Edit the workflow to fix structural issues
```bash
goflow edit my-workflow
# Fix the issue in the editor
goflow validate my-workflow
```

### Debug Mode

Enable debug mode for detailed troubleshooting:

```bash
goflow import workflow.yaml --debug --verbose
```

This shows:
- Full error stack traces
- Server registration details
- Credential detection information
- Validation step-by-step output

## See Also

- [Template System Documentation](template-system.md) - Creating reusable, parameterized workflows
- [Template Quick Reference](TEMPLATE_QUICK_REFERENCE.md) - Template syntax and helpers
- [CLI Commands Implementation](CLI_COMMANDS_IMPLEMENTATION.md) - Comprehensive CLI reference
- [Template Helpers](TEMPLATE_HELPERS.md) - Template function reference

## Quick Reference

### Export Commands

```bash
# Export to stdout
goflow export workflow.yaml

# Export to file
goflow export workflow.yaml -o shared.yaml
```

### Import Commands

```bash
# Import workflow
goflow import workflow.yaml

# Import with verbose output
goflow import workflow.yaml --verbose
```

### Credential Commands

```bash
# Add credentials
goflow credential add server-id --env KEY=value

# List credentials
goflow credential list

# Remove credentials
goflow credential remove server-id
```

### Server Commands

```bash
# Register server
goflow server add server-id command [args...]

# List servers
goflow server list

# Test server connectivity
goflow server test server-id
```
