# Workflow Sharing Guide

## Overview

GoFlow workflows are designed to be safely shared between team members, published in workflow registries, or distributed as reusable templates. The workflow sharing system automatically strips sensitive credentials and provides clear guidance for setting up imported workflows.

This guide covers best practices for exporting, sharing, and importing GoFlow workflows while maintaining security and ease of use.

## Benefits of Workflow Sharing

- **Reusability**: Share common patterns across teams and projects
- **Collaboration**: Build workflows together with version control
- **Security**: Automatic credential stripping prevents accidental exposure
- **Portability**: Workflows work across different environments with proper credential setup
- **Templates**: Parameterized workflows enable customization without editing
- **Knowledge Sharing**: Document and distribute best practices

## Security Considerations

### The Security Model

GoFlow follows a strict separation between workflow logic and credentials:

1. **Workflows define structure**: Nodes, edges, transformations, and logic
2. **Credentials are stored separately**: System keyring or secure credential store
3. **Export strips credentials**: Automatic sanitization prevents leaks
4. **Import requires setup**: Recipients must configure their own credentials

This ensures that shared workflows are safe to commit to version control, post in public repositories, or share via email/Slack.

### What Gets Stripped on Export

GoFlow automatically removes sensitive information matching these patterns:

#### Sensitive Environment Variable Patterns
- `KEY`, `SECRET`, `TOKEN`
- `PASSWORD`, `PASSPHRASE`
- `CREDENTIAL`, `AUTH`, `BEARER`
- `PRIVATE`, `CLIENT_SECRET`
- `DATABASE_URL`, `CONNECTION_STRING`, `DSN`
- `OAUTH` (any OAuth-related variables)

#### Credential References
- `credential_ref` fields are replaced with `<CREDENTIAL_REF_REQUIRED>` placeholder
- This signals to recipients that credentials must be configured

#### Example: Before Export
```yaml
servers:
  - id: github-api
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "ghp_YOUR_TOKEN_HERE"  # Sensitive - will be stripped
      GITHUB_ORG: myorg                          # Non-sensitive - preserved
      API_BASE_URL: https://api.github.com       # Non-sensitive - preserved
    credential_ref: keyring://github-token
```

#### Example: After Export
```yaml
# CREDENTIAL WARNING: This workflow has been exported for sharing.
# Sensitive credentials have been removed and must be configured before use.
# Please set up credential references for servers marked with <CREDENTIAL_REF_REQUIRED>.

servers:
  - id: github-api
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_ORG: myorg
      API_BASE_URL: https://api.github.com
    credential_ref: "<CREDENTIAL_REF_REQUIRED>"
```

### What Stays in Exported Files

Non-sensitive configuration is preserved to ensure workflows remain functional:

- **Server commands**: The command and arguments to start MCP servers
- **Transport type**: stdio, sse, or http
- **Non-sensitive env vars**: HOST, PORT, SERVICE_NAME, LOG_LEVEL, etc.
- **Workflow logic**: All nodes, edges, conditions, and transformations
- **Variable definitions**: Workflow-scoped variables (not credential values)
- **Metadata**: Name, version, description, tags, author

### Best Practices for Safe Sharing

1. **Always use credential references**
   ```yaml
   servers:
     - id: api-server
       credential_ref: keyring://api-credentials
       # NOT: env: { API_KEY: "hardcoded-secret" }
   ```

2. **Never commit credentials to version control**
   - Add `.env` files to `.gitignore`
   - Use GoFlow's credential management commands
   - Let export system handle sanitization

3. **Review exported workflows before sharing**
   ```bash
   # Export and review
   goflow export my-workflow -o /tmp/review.yaml
   cat /tmp/review.yaml | grep -i "secret\|token\|password"

   # If clean, share it
   cp /tmp/review.yaml shared-workflows/
   ```

4. **Document required credentials**
   ```yaml
   # In workflow description or separate README
   description: |
     This workflow requires the following credentials:
     - github-api: GITHUB_TOKEN (personal access token with repo scope)
     - slack-notifier: SLACK_WEBHOOK_URL (incoming webhook URL)
   ```

5. **Use descriptive server IDs**
   - Good: `github-api`, `prod-database`, `slack-alerts`
   - Bad: `server1`, `api`, `db`

## Exporting Workflows

The `goflow export` command prepares workflows for sharing by automatically stripping credentials.

**Command Syntax**:
```bash
goflow export <workflow-name> [--output <file>] [--verbose]
```

- `<workflow-name>`: Name of the workflow (not file path) as stored in GoFlow
- `--output, -o`: Output file path (optional, defaults to stdout)
- `--verbose, -v`: Show detailed export information

### Basic Export Commands

```bash
# Export to stdout (for piping or viewing)
goflow export my-workflow

# Export to a file
goflow export my-workflow -o shared-workflow.yaml
goflow export my-workflow --output ~/Desktop/workflow.yaml

# Export with verbose information
goflow export my-workflow -o workflow.yaml --verbose
```

### Export Process

When you export a workflow:

1. **Load workflow**: Read from `~/.goflow/workflows/<name>.yaml`
2. **Validate structure**: Ensure workflow is well-formed (optional)
3. **Strip credentials**: Remove sensitive environment variables
4. **Replace credential references**: Use `<CREDENTIAL_REF_REQUIRED>` placeholder
5. **Add warning header**: Include credential setup instructions
6. **Output YAML**: Write to file or stdout

### Export Output Example

```bash
$ goflow export data-pipeline -o shared.yaml --verbose

✓ Exported workflow 'data-pipeline' successfully
  - Removed credentials from 2 server configuration(s)
  - Output: shared.yaml

⚠ Warning: Workflow contains credential references. Recipients must configure:
  - filesystem-server
  - database-server

Workflow details:
  Name: data-pipeline
  Version: 1.0
  Nodes: 8
  Servers: 2
```

### Exporting for Different Scenarios

#### For Team Collaboration (Version Control)
```bash
# Export to team workflows directory
goflow export data-pipeline -o team-workflows/data-pipeline.yaml

# Commit to version control
cd team-workflows
git add data-pipeline.yaml
git commit -m "Add data pipeline workflow"
git push
```

#### For Public Sharing (GitHub/GitLab)
```bash
# Export to public repository
goflow export api-integration -o workflows/api-integration.yaml

# Create README with setup instructions
cat > workflows/README.md << EOF
# API Integration Workflow

## Required Credentials
- \`api-server\`: API_KEY (get from https://example.com/api/keys)

## Setup
1. Register server: \`goflow server add api-server...\`
2. Add credentials: \`goflow credential add api-server --key API_KEY\`
3. Import workflow: \`goflow import api-integration.yaml\`
EOF

git add workflows/
git commit -m "Add API integration workflow with setup docs"
```

#### For One-off Sharing (Email/Slack)
```bash
# Export and share
goflow export quick-task -o /tmp/quick-task.yaml

# Email/Slack the file with instructions
echo "Import with: goflow import quick-task.yaml"
```

## Importing Workflows

The `goflow import` command loads shared workflows and helps configure required servers.

**Command Syntax**:
```bash
goflow import <file-path> [--name <workflow-name>] [--verbose] [--no-interact]
```

- `<file-path>`: Path to the workflow YAML file to import
- `--name, -n`: Override workflow name (optional, uses name from file if not specified)
- `--verbose, -v`: Show detailed import information
- `--no-interact`: Skip interactive prompts (for automation/CI)

### Basic Import Commands

```bash
# Import a workflow
goflow import /path/to/workflow.yaml

# Import with custom name
goflow import workflow.yaml --name my-custom-name

# Import with verbose output
goflow import workflow.yaml --verbose

# Non-interactive import (skip server setup prompts)
goflow import workflow.yaml --no-interact
```

### Import Process

When you import a workflow, GoFlow performs these steps:

1. **File validation**: Verify file exists and is readable
2. **Parse YAML**: Load and validate workflow structure
3. **Version check**: Ensure workflow version is compatible
4. **Server validation**: Check that referenced servers are registered
5. **Credential detection**: Identify missing credentials and placeholders
6. **Workflow validation**: Validate nodes, edges, and structure
7. **Installation**: Copy to `~/.goflow/workflows/<name>.yaml`

### Interactive Server Setup

If servers are missing, GoFlow can configure them interactively:

```bash
$ goflow import team-workflow.yaml

⚠  Missing server configurations detected:
  - github-api
  - slack-notifier

Would you like to configure these servers now? (y/n): y

--- Configuring server: github-api ---
Command (e.g., node, python, npx): npx
Args (space-separated, or press Enter to skip): -y @modelcontextprotocol/server-github
Transport (stdio/sse/http) [default: stdio]: stdio
Friendly name [default: github-api]: GitHub API
Description (optional): GitHub integration server

✓ Server 'github-api' configured

--- Configuring server: slack-notifier ---
Command (e.g., node, python, npx): npx
Args (space-separated, or press Enter to skip): -y @modelcontextprotocol/server-slack
Transport (stdio/sse/http) [default: stdio]: sse
Friendly name [default: slack-notifier]: Slack Notifications
Description (optional): Slack notification integration

✓ Server 'slack-notifier' configured

✓ Saved 2 server configuration(s)

Note: Credentials must be configured separately using:
  goflow credential add <server-id> --key <credential-key>
```

### Handling Missing Servers (Non-interactive)

```bash
$ goflow import workflow.yaml --no-interact

✗ Workflow references missing servers

Missing servers:
  - github-api
  - slack-notifier

Please register these servers before importing:
  goflow server add <server-id> <command> [args...]
```

**Resolution**:
```bash
# Register servers manually
goflow server add github-api npx -y @modelcontextprotocol/server-github
goflow server add slack-notifier npx -y @modelcontextprotocol/server-slack --transport sse

# Retry import
goflow import workflow.yaml
```

### Secure Automation Patterns

For automated workflows (CI/CD, scripts), use these secure patterns:

**Environment Variable Pattern** (Recommended):
```bash
# Store secrets in CI/CD secret manager (GitHub Secrets, GitLab Variables, etc.)
# Inject as environment variables, then use stdin for credential input

# In GitHub Actions:
- name: Setup credentials
  env:
    API_KEY: ${{ secrets.API_KEY }}
  run: |
    echo "$API_KEY" | goflow credential add api-server --key API_KEY --stdin

# In shell script with environment variables:
#!/bin/bash
# Expects API_KEY environment variable to be set
echo "$API_KEY" | goflow credential add api-server --key API_KEY --stdin
```

**File-Based Pattern** (For secure file storage):
```bash
# Read from secure file (with restricted permissions)
# Assumes secret file is mounted/injected by CI system
cat /run/secrets/api-key | goflow credential add api-server --key API_KEY --stdin

# Or for multiple credentials:
while IFS='=' read -r key value; do
  echo "$value" | goflow credential add api-server --key "$key" --stdin
done < /run/secrets/credentials.env
```

**Docker/Kubernetes Pattern**:
```bash
# Kubernetes secret mounted as file
cat /var/secrets/api-key | goflow credential add api-server --key API_KEY --stdin

# Docker secret
cat /run/secrets/api_key | goflow credential add api-server --key API_KEY --stdin
```

**⚠️ Security Note**: Never use `--value` flag in automation. Always use stdin or environment variables that are injected securely by your CI/CD system.

### Credential Setup After Import

When importing workflows with credential placeholders:

```bash
$ goflow import workflow.yaml

⚠  Workflow contains credential placeholders

Servers with placeholders:
  - github-api
  - database-server

You will need to configure credentials before execution.

✓ Imported workflow 'data-pipeline' successfully
  - Workflow saved to: /Users/yourname/.goflow/workflows/data-pipeline.yaml

⚠  Required setup:
  1. Configure credentials:
     goflow credential add github-api --key GITHUB_TOKEN
     goflow credential add database-server --key DB_PASSWORD

  2. Test servers:
     goflow server test github-api
     goflow server test database-server

  3. Run workflow:
     goflow run data-pipeline
```

### Successful Import

```bash
$ goflow import simple-workflow.yaml

✓ Imported workflow 'simple-workflow' successfully
  - Workflow saved to: /Users/yourname/.goflow/workflows/simple-workflow.yaml

Next steps:
  1. Validate: goflow validate simple-workflow
  2. Run: goflow run simple-workflow
```

## Using Built-in Templates

GoFlow includes several built-in workflow templates for common patterns. Templates provide parameterized workflows that can be customized for your needs.

### Available Templates

#### 1. ETL Pipeline Template

Extract-Transform-Load workflow for data processing:

```bash
# View template parameters
goflow template show etl-pipeline

# Instantiate with parameters
goflow template instantiate etl-pipeline \
  --param source_path=/data/input.json \
  --param destination_path=/data/output.json \
  --param transform_expression='$.items[*].{name,price}' \
  --param batch_size=100 \
  --param validate_output=true \
  --output my-etl-workflow.yaml

# Import and run
goflow import my-etl-workflow.yaml
goflow run my-etl-workflow
```

**Parameters**:
- `source_path` (string, required): Path to source data file
- `destination_path` (string, required): Output file path
- `transform_expression` (string, optional): JSONPath transformation
- `batch_size` (number, optional): Records per batch (default: 100)
- `validate_output` (boolean, optional): Validate before writing (default: true)
- `error_handling` (string, optional): fail_fast, continue, or retry

**Use Cases**:
- Processing log files
- Converting data formats
- Data migration pipelines
- Batch data transformations

#### 2. API Workflow Template

HTTP API integration with error handling:

```bash
# Instantiate API workflow
goflow template instantiate api-workflow \
  --param api_endpoint=https://api.example.com/users \
  --param api_method=POST \
  --param request_body='{"name":"John","email":"john@example.com"}' \
  --param retry_count=5 \
  --param timeout_seconds=60 \
  --output user-api-workflow.yaml
```

**Parameters**:
- `api_endpoint` (string, required): Target API endpoint URL
- `api_method` (string, optional): HTTP method (default: GET)
- `request_body` (string, optional): JSON request body
- `retry_count` (number, optional): Retry attempts (default: 3)
- `timeout_seconds` (number, optional): Request timeout (default: 30)

**Use Cases**:
- REST API integration
- Webhook processing
- Third-party service integration
- API testing workflows

#### 3. Multi-Server Workflow Template

Coordinate multiple MCP servers:

```bash
# Instantiate multi-server workflow
goflow template instantiate multi-server-workflow \
  --param input_source=/data/records.json \
  --param processing_mode=parallel \
  --param notification_enabled=true \
  --param recipients='["admin@example.com","dev@example.com"]' \
  --output data-processor.yaml
```

**Parameters**:
- `input_source` (string, required): Data source identifier
- `processing_mode` (string, optional): sequential or parallel (default: sequential)
- `notification_enabled` (boolean, optional): Enable notifications (default: true)
- `recipients` (array, optional): Email recipients for notifications

**Use Cases**:
- Complex data pipelines
- Multi-service orchestration
- Event-driven workflows
- Notification systems

### Creating Custom Templates

You can create your own templates from existing workflows:

```yaml
# my-template.yaml
name: custom-api-template
description: Custom API integration template
version: "1.0"

parameters:
  - name: endpoint
    type: string
    required: true
    description: API endpoint URL
    validation:
      pattern: "^https?://.+"

  - name: timeout
    type: number
    required: false
    default: 30
    validation:
      min: 1
      max: 300

workflow_spec:
  nodes:
    - id: start
      type: start

    - id: api_call
      type: mcp_tool
      config:
        server: http-server
        tool: fetch
        parameters:
          url: "{{endpoint}}"
          timeout: "{{timeout}}"

    - id: end
      type: end

  edges:
    - from: start
      to: api_call
    - from: api_call
      to: end
```

See [Template System Documentation](template-system.md) for complete template creation guide.

## Credential Management

### Adding Credentials

After importing a workflow, configure credentials for servers that need them:

```bash
# Recommended: Interactive prompt (secure - not in shell history)
goflow credential add github-api --key GITHUB_TOKEN
# Prompts: Enter value for 'GITHUB_TOKEN': [hidden input]

# Add multiple credentials for one server
goflow credential add database-server --key DB_HOST
# Prompts: Enter value for 'DB_HOST': [hidden input]

goflow credential add database-server --key DB_PORT
# Prompts: Enter value for 'DB_PORT': [hidden input]

goflow credential add database-server --key DB_PASSWORD
# Prompts: Enter value for 'DB_PASSWORD': [hidden input]
```

**⚠️ WARNING**: Avoid using `--value` flag as it exposes secrets in shell history and process tables:
```bash
# INSECURE - visible in shell history:
goflow credential add api-server --key API_KEY --value sk-abc123  # DON'T DO THIS

# Use interactive prompt instead:
goflow credential add api-server --key API_KEY  # Secure - prompts for value
```

### Listing Credentials

```bash
# List all credentials
goflow credential list

# Output:
# Configured Credentials:
#
# SERVER ID        CREDENTIAL KEY   STATUS
# ─────────        ──────────────   ──────
# github-api       GITHUB_TOKEN     (set)
# database-server  DB_HOST          (set)
# database-server  DB_PORT          (set)
# database-server  DB_PASSWORD      (set)

# List credentials for specific server
goflow credential list github-api

# Output:
# Credentials for 'github-api':
#   - GITHUB_TOKEN (set)
```

### Credential Security

Credentials are stored securely using OS-native keyrings:

1. **System Keyring Storage** (Current Implementation)
   - macOS: Keychain (encrypted, system-level security)
   - Windows: Credential Manager (Windows Credential Store)
   - Linux: Secret Service API (GNOME Keyring, KWallet)
   - Credentials persist across sessions
   - Protected by OS security mechanisms

2. **Never Exported**: The following are automatically stripped from exported workflows:
   - All environment variables matching sensitive patterns (KEY, SECRET, TOKEN, PASSWORD, etc.)
   - `credential_ref` field values (replaced with placeholder)
   - Any field containing credential data

3. **Access Control**: Only accessible by GoFlow and your user account (OS-enforced)

### Security Best Practices

```bash
# ✓ RECOMMENDED: Interactive prompt (most secure - for local use)
goflow credential add api-server --key API_KEY
# Prompts for value with hidden input - not stored in shell history

# ✗ AVOID: Never pass secrets as command-line arguments
goflow credential add api-server --key API_KEY --value secret123  # INSECURE
```

### CI/CD Automation (Non-Interactive)

For automation and CI/CD pipelines, use environment variables:

**GitHub Actions Example**:
```yaml
# .github/workflows/deploy.yml
- name: Configure workflow credentials
  env:
    API_KEY: ${{ secrets.API_KEY }}  # GitHub secret injected as env var
    DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
  run: |
    # Use stdin to avoid exposing secrets in process list
    echo "$API_KEY" | goflow credential add api-server --key API_KEY --stdin
    echo "$DB_PASSWORD" | goflow credential add db-server --key DB_PASSWORD --stdin
```

**GitLab CI Example**:
```yaml
# .gitlab-ci.yml
deploy:
  variables:
    API_KEY: $CI_API_KEY  # GitLab secret variable
  script:
    - echo "$API_KEY" | goflow credential add api-server --key API_KEY --stdin
```

**CI/CD Best Practices**:
- ✓ Use CI provider's secret manager (GitHub Secrets, GitLab Variables, etc.)
- ✓ Inject secrets via environment variables in CI workflows
- ✓ Use stdin for non-interactive credential input: `echo "$SECRET" | goflow credential add --stdin`
- ✗ Never pass secrets as CLI arguments (`--value` flag)
- ✗ Never commit credential files to version control
- ✗ Never hardcode secrets in CI configuration files
- Note: File permission commands like `chmod 600` apply to Linux/macOS only

## Collaboration Workflows

### Team Workflow Repository

**Structure**:
```
team-workflows/
├── README.md                   # Setup instructions
├── CREDENTIALS_REQUIRED.md     # Credential documentation
├── workflows/
│   ├── data-pipeline.yaml
│   ├── api-integration.yaml
│   └── notification-system.yaml
├── templates/
│   ├── etl-template.yaml
│   └── api-template.yaml
└── docs/
    ├── data-pipeline.md
    └── api-integration.md
```

**Setup README**:
```markdown
# Team Workflows

## Quick Start

1. Clone repository:
   ```bash
   git clone https://github.com/yourorg/workflows.git
   cd workflows
   ```

2. Import workflows:
   ```bash
   goflow import workflows/data-pipeline.yaml
   goflow import workflows/api-integration.yaml
   ```

3. Configure credentials (see CREDENTIALS_REQUIRED.md)

4. Validate and run:
   ```bash
   goflow validate data-pipeline
   goflow run data-pipeline
   ```

## Adding New Workflows

1. Create workflow locally
2. Test thoroughly
3. Export: `goflow export my-workflow -o workflows/my-workflow.yaml`
4. Document credentials in CREDENTIALS_REQUIRED.md
5. Create PR for review
```

**Credentials Documentation** (CREDENTIALS_REQUIRED.md):
```markdown
# Required Credentials

## data-pipeline workflow

### filesystem-server
- **STORAGE_ACCESS_KEY**: S3/storage access key
- **Where to get**: AWS Console > IAM > Access Keys
- **Setup**: `goflow credential add filesystem-server --key STORAGE_ACCESS_KEY`

### database-server
- **DB_PASSWORD**: PostgreSQL password
- **Where to get**: Database admin
- **Setup**: `goflow credential add database-server --key DB_PASSWORD`

## api-integration workflow

### api-server
- **API_KEY**: Third-party API key
- **Where to get**: https://example.com/api/keys
- **Setup**: `goflow credential add api-server --key API_KEY`
- **Scopes required**: read, write
```

### Version Control Best Practices

#### .gitignore
```gitignore
# Credential files
*.credentials
*.secrets
.env
.env.local

# Local workflow copies with credentials
*-local.yaml
*-dev.yaml

# GoFlow local data (if committed by accident)
.goflow/
```

#### Git Workflow
```bash
# 1. Create and test workflow locally
goflow create my-workflow
goflow edit my-workflow
goflow validate my-workflow
goflow run my-workflow

# 2. Export for sharing
goflow export my-workflow -o workflows/my-workflow.yaml

# 3. Review exported file (ensure no credentials)
cat workflows/my-workflow.yaml | grep -i "secret\|password\|token"

# 4. Commit and push
git add workflows/my-workflow.yaml
git commit -m "Add my-workflow: description of what it does"
git push origin feature/my-workflow

# 5. Team members pull and import
git pull origin main
goflow import workflows/my-workflow.yaml
```

### Sharing Across Environments

#### Development → Staging → Production

```bash
# Development
goflow export data-pipeline -o data-pipeline-dev.yaml

# Staging (import and reconfigure for staging environment)
goflow import data-pipeline-dev.yaml --name data-pipeline-staging
goflow credential add filesystem-server --key STORAGE_ACCESS_KEY
# (use staging credentials)

# Production (import and reconfigure for production)
goflow import data-pipeline-dev.yaml --name data-pipeline-prod
goflow credential add filesystem-server --key STORAGE_ACCESS_KEY
# (use production credentials)
```

**Environment-Specific Variables**:
```yaml
# Use workflow variables for environment-specific configuration
variables:
  - name: environment
    type: string
    default: "development"

  - name: api_base_url
    type: string
    default: "https://dev-api.example.com"

  - name: log_level
    type: string
    default: "debug"
```

### Code Review for Workflows

**Review Checklist**:
- [ ] No hardcoded credentials
- [ ] All server configurations use credential references
- [ ] Workflow validates successfully
- [ ] Documentation includes credential setup
- [ ] Sensitive environment variables are not present
- [ ] Workflow tested in clean environment
- [ ] Error handling is appropriate
- [ ] Naming conventions followed

## Common Patterns and Examples

### Example 1: Data Pipeline

**Scenario**: Extract data from API, transform, load to database

```bash
# Export the workflow
goflow export data-pipeline -o workflows/data-pipeline.yaml

# Team member imports
goflow import workflows/data-pipeline.yaml

# Configure credentials
goflow credential add api-server --key API_KEY
goflow credential add database-server --key DB_PASSWORD

# Run
goflow run data-pipeline
```

### Example 2: Notification Workflow

**Scenario**: Monitor system and send Slack notifications

```bash
# Create workflow
goflow create system-monitor

# Export for team
goflow export system-monitor -o workflows/system-monitor.yaml

# Document in README
echo "Requires: Slack webhook URL" >> README.md

# Team imports and configures
goflow import workflows/system-monitor.yaml
goflow credential add slack-notifier --key SLACK_WEBHOOK_URL
```

### Example 3: Multi-Environment Deployment

**Scenario**: Deploy application to dev, staging, prod

```bash
# Create deployment workflow
goflow create app-deployment

# Export base workflow
goflow export app-deployment -o workflows/app-deployment.yaml

# Each environment imports with different name
# Dev
goflow import workflows/app-deployment.yaml --name app-deployment-dev
goflow credential add deploy-server --key DEPLOY_KEY
# (use dev deploy key)

# Staging
goflow import workflows/app-deployment.yaml --name app-deployment-staging
goflow credential add deploy-server --key DEPLOY_KEY
# (use staging deploy key)

# Production
goflow import workflows/app-deployment.yaml --name app-deployment-prod
goflow credential add deploy-server --key DEPLOY_KEY
# (use production deploy key)
```

## Troubleshooting

### Common Import Errors

#### Error: Workflow file not found

```bash
✗ workflow file not found: workflow.yaml
```

**Cause**: File path is incorrect or file doesn't exist

**Fix**:
```bash
# Verify file exists
ls -la /path/to/workflow.yaml

# Use absolute path
goflow import /absolute/path/to/workflow.yaml

# Or use relative path from current directory
cd /path/to
goflow import ./workflow.yaml
```

#### Error: Workflow already exists

```bash
✗ workflow already exists: my-workflow

Location: /Users/yourname/.goflow/workflows/my-workflow.yaml
Use --name flag with a different name or remove the existing workflow first
```

**Fix Option 1**: Use different name
```bash
goflow import workflow.yaml --name my-workflow-v2
```

**Fix Option 2**: Remove existing workflow
```bash
rm ~/.goflow/workflows/my-workflow.yaml
goflow import workflow.yaml
```

**Fix Option 3**: Backup and replace
```bash
mv ~/.goflow/workflows/my-workflow.yaml ~/.goflow/workflows/my-workflow-backup.yaml
goflow import workflow.yaml
```

#### Error: Missing server configurations

```bash
✗ Workflow references missing servers

Missing servers:
  - github-api
  - database-server

Please register these servers before importing:
  goflow server add <server-id> <command> [args...]
```

**Fix**: Register missing servers
```bash
# Register each server
goflow server add github-api npx -y @modelcontextprotocol/server-github
goflow server add database-server database-mcp-server --transport stdio

# Verify registration
goflow server list

# Retry import
goflow import workflow.yaml
```

#### Error: Version incompatibility

```bash
✗ Incompatible workflow version
  Workflow version: 2.0
  Supported versions: [1.0]
```

**Fix**: Update GoFlow or migrate workflow
```bash
# Option 1: Update GoFlow
go install github.com/dshills/goflow@latest

# Option 2: Check for migration guide
# Visit: https://github.com/dshills/goflow/wiki/Migration-Guides
```

#### Error: Invalid YAML syntax

```bash
✗ Failed to parse workflow YAML: yaml: line 15: mapping values are not allowed in this context
```

**Fix**: Validate and fix YAML syntax
```bash
# Use YAML linter
yamllint workflow.yaml

# Common issues:
# - Incorrect indentation (must use spaces, not tabs)
# - Missing quotes around special characters
# - Invalid escape sequences
```

### Common Execution Errors

#### Error: Missing credentials

```bash
✗ Execution failed: server 'github-api' requires credentials
```

**Fix**: Add required credentials
```bash
# Check what credentials are needed
goflow server show github-api

# Add credentials
goflow credential add github-api --key GITHUB_TOKEN

# Verify
goflow credential list github-api

# Retry execution
goflow run my-workflow
```

#### Error: Server connection failed

```bash
✗ Execution failed: failed to connect to server 'api-server'
```

**Fix**: Test server connectivity
```bash
# Test server
goflow server test api-server

# Check server configuration
goflow server list --verbose

# If server command is wrong, update it
goflow server remove api-server
goflow server add api-server correct-command [args...]
```

#### Error: Server not found

```bash
✗ Execution failed: server 'unknown-server' not found in registry
```

**Fix**: Register the server
```bash
# List registered servers
goflow server list

# Register missing server
goflow server add unknown-server command [args...]

# Or check workflow for typo
goflow edit my-workflow
# (verify server ID matches registered servers)
```

### Validation Issues

#### Circular dependency detected

```bash
✗ Workflow validation failed: circular dependency detected: node1 -> node2 -> node3 -> node1
```

**Fix**: Remove circular edges
```bash
# Edit workflow
goflow edit my-workflow

# Remove edge that creates cycle
# Or redesign workflow to eliminate loop

# Validate
goflow validate my-workflow
```

#### Undefined variable reference

```bash
✗ Workflow validation failed: node 'transform_data' references undefined variable 'missing_var'
```

**Fix**: Add missing variable or fix reference
```bash
# Edit workflow
goflow edit my-workflow

# Add variable definition:
# variables:
#   - name: missing_var
#     type: string
#     default: "value"

# Or fix variable reference in node

# Validate
goflow validate my-workflow
```

#### Node type not supported

```bash
✗ Workflow validation failed: unknown node type 'custom_node'
```

**Fix**: Use supported node type
```bash
# Supported node types:
# - start
# - end
# - mcp_tool
# - transform
# - condition
# - loop
# - parallel
# - passthrough

# Edit workflow and change node type
goflow edit my-workflow
```

### Debug Mode

Enable detailed troubleshooting output:

```bash
# Import with debug output
goflow import workflow.yaml --verbose --debug

# Export with debug output
goflow export workflow.yaml -o output.yaml --verbose

# Run with debug output
goflow run workflow.yaml --debug --verbose
```

Debug output includes:
- Full error stack traces
- Server registration details
- Credential detection information
- Validation step-by-step output
- YAML parsing details
- File system operations

### Getting Help

```bash
# Command help
goflow import --help
goflow export --help
goflow credential --help

# Check GoFlow version
goflow version

# View configuration
goflow config show

# Server diagnostics
goflow server test <server-id> --verbose
```

## Version Compatibility

### Workflow Versions

GoFlow workflows use semantic versioning:

```yaml
version: "1.0"
```

Current supported version: **1.0**

### Compatibility Matrix

| Workflow Version | GoFlow Version | Status      |
|-----------------|----------------|-------------|
| 1.0             | 0.1.0+         | ✓ Supported |
| 2.0             | Future         | Planned     |

### Version Mismatch Handling

If you import a workflow with an incompatible version:

```bash
✗ Incompatible workflow version
  Workflow version: 2.0
  Supported versions: [1.0]
```

**Resolution**:
1. Update GoFlow: `go install github.com/dshills/goflow@latest`
2. Check migration guides: https://github.com/dshills/goflow/wiki/Migration
3. Contact workflow author for compatible version

### Backward Compatibility

GoFlow strives to maintain backward compatibility:
- **Minor versions** (1.0 → 1.1): Fully backward compatible
- **Major versions** (1.x → 2.x): May introduce breaking changes
- **Migration period**: Old versions supported for 6 months after new major release

## See Also

- [Template System Documentation](template-system.md) - Creating reusable, parameterized workflows
- [Template Quick Reference](TEMPLATE_QUICK_REFERENCE.md) - Template syntax and helpers
- [CLI Commands Implementation](CLI_COMMANDS_IMPLEMENTATION.md) - Comprehensive CLI reference
- [Template Guide](template-guide.md) - In-depth template creation guide

## Quick Reference

### Export Commands
```bash
# Export to stdout
goflow export workflow

# Export to file
goflow export workflow -o shared.yaml

# Export with verbose output
goflow export workflow -o shared.yaml --verbose
```

### Import Commands
```bash
# Import workflow
goflow import workflow.yaml

# Import with custom name
goflow import workflow.yaml --name custom-name

# Import with verbose output
goflow import workflow.yaml --verbose

# Non-interactive import
goflow import workflow.yaml --no-interact
```

### Credential Commands
```bash
# Add credential (interactive prompt - RECOMMENDED)
goflow credential add server-id --key KEY_NAME

# List all credentials
goflow credential list

# List credentials for specific server
goflow credential list server-id
```

### Server Commands
```bash
# Register server
goflow server add server-id command [args...]

# Register with transport
goflow server add server-id command [args...] --transport sse

# List servers
goflow server list

# Show server details
goflow server show server-id

# Test server connectivity
goflow server test server-id

# Remove server
goflow server remove server-id
```

### Template Commands
```bash
# Show available templates
goflow template list

# Show template parameters
goflow template show etl-pipeline

# Instantiate template
goflow template instantiate etl-pipeline \
  --param source_path=/data/input.json \
  --param destination_path=/data/output.json \
  --output my-workflow.yaml
```

### Workflow Commands
```bash
# Create workflow
goflow create workflow-name

# Edit workflow
goflow edit workflow-name

# Validate workflow
goflow validate workflow-name

# Run workflow
goflow run workflow-name

# List workflows
goflow list
```
