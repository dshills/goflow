# MCP Transport Configuration

GoFlow supports three transport types for connecting to MCP servers: stdio, Server-Sent Events (SSE), and HTTP JSON-RPC. This document explains how to configure each transport type in your workflow definitions.

## Transport Types Overview

### stdio (Default)
Traditional subprocess-based communication using standard input/output streams. Best for local MCP servers running as child processes.

**Use cases:**
- Local Python/Node.js MCP servers
- Development and testing
- Servers that don't require network communication

### SSE (Server-Sent Events)
Unidirectional event streaming where the server pushes messages to the client. Client sends requests via HTTP POST.

**Use cases:**
- Real-time event streaming from servers
- Long-running connections with server updates
- Services that need to push notifications

### HTTP (JSON-RPC)
Synchronous request-response protocol using HTTP POST with JSON-RPC payloads.

**Use cases:**
- Stateless MCP servers
- Load-balanced server deployments
- RESTful API-style services

## Configuration Examples

### stdio Transport

**Minimal configuration (uses stdio by default):**
```yaml
servers:
  - id: my-server
    command: python
    args:
      - -m
      - my_mcp_server
```

**Explicit stdio configuration:**
```yaml
servers:
  - id: my-server
    name: My Local Server
    transport: stdio
    command: python
    args:
      - -m
      - my_mcp_server
    env:
      PYTHONPATH: /opt/mcp-servers
      DEBUG: "true"
```

**Required fields:**
- `id`: Server identifier
- `command`: Executable command

**Optional fields:**
- `args`: Command arguments
- `env`: Environment variables
- `name`: Human-readable server name

**Validation rules:**
- ✅ Command must be specified
- ❌ URL must not be specified
- ❌ Headers must not be specified

### SSE Transport

```yaml
servers:
  - id: sse-server
    name: SSE MCP Server
    transport: sse
    url: https://api.example.com/mcp/sse
    headers:
      Authorization: Bearer ${env.MCP_TOKEN}
      X-Client-ID: goflow-client
```

**Required fields:**
- `id`: Server identifier
- `transport`: Must be "sse"
- `url`: SSE endpoint URL (must start with http:// or https://)

**Optional fields:**
- `headers`: HTTP headers for authentication and custom metadata
- `name`: Human-readable server name

**Validation rules:**
- ✅ URL must be specified
- ✅ URL must start with http:// or https://
- ❌ Command must not be specified
- ❌ Args must not be specified

### HTTP Transport

```yaml
servers:
  - id: http-server
    name: HTTP MCP Server
    transport: http
    url: https://api.example.com/mcp/rpc
    headers:
      Authorization: Bearer ${env.MCP_TOKEN}
      Content-Type: application/json
      X-API-Key: key123
```

**Required fields:**
- `id`: Server identifier
- `transport`: Must be "http"
- `url`: HTTP endpoint URL (must start with http:// or https://)

**Optional fields:**
- `headers`: HTTP headers for authentication and custom metadata
- `name`: Human-readable server name

**Validation rules:**
- ✅ URL must be specified
- ✅ URL must start with http:// or https://
- ❌ Command must not be specified
- ❌ Args must not be specified

## Backward Compatibility

Existing workflows without a `transport` field will automatically use stdio transport. This ensures complete backward compatibility:

**Before (still works):**
```yaml
servers:
  - id: legacy-server
    command: python
    args:
      - -m
      - server
```

**Equivalent to:**
```yaml
servers:
  - id: legacy-server
    transport: stdio
    command: python
    args:
      - -m
      - server
```

## Common Patterns

### Mixed Transport Workflow

You can use multiple transport types in a single workflow:

```yaml
servers:
  # Local data processing
  - id: local-processor
    command: python
    args: ["-m", "data_processor"]

  # Remote API service
  - id: api-service
    transport: http
    url: https://api.example.com/mcp

  # Real-time event stream
  - id: event-stream
    transport: sse
    url: https://events.example.com/sse

nodes:
  - id: process_local
    type: mcp_tool
    server: local-processor
    tool: process

  - id: call_api
    type: mcp_tool
    server: api-service
    tool: analyze

  - id: subscribe_events
    type: mcp_tool
    server: event-stream
    tool: listen
```

### Authentication Headers

Use environment variable interpolation for secure credential management:

```yaml
servers:
  - id: secure-server
    transport: http
    url: https://api.example.com/mcp
    headers:
      Authorization: Bearer ${env.MCP_AUTH_TOKEN}
      X-API-Key: ${env.MCP_API_KEY}
```

Set credentials via environment variables:
```bash
export MCP_AUTH_TOKEN="your-token-here"
export MCP_API_KEY="your-api-key"
goflow run workflow.yaml
```

## Validation Errors

Common validation errors and how to fix them:

### "command is required for stdio transport"
**Problem:** Using stdio transport without specifying a command.
```yaml
# ❌ Invalid
servers:
  - id: my-server
    transport: stdio
```

**Solution:** Add the command field:
```yaml
# ✅ Valid
servers:
  - id: my-server
    transport: stdio
    command: python
```

### "URL is required for sse/http transport"
**Problem:** Using SSE or HTTP transport without a URL.
```yaml
# ❌ Invalid
servers:
  - id: my-server
    transport: http
```

**Solution:** Add the URL field:
```yaml
# ✅ Valid
servers:
  - id: my-server
    transport: http
    url: https://api.example.com/mcp
```

### "URL must start with http:// or https://"
**Problem:** Invalid URL scheme.
```yaml
# ❌ Invalid
servers:
  - id: my-server
    transport: http
    url: ftp://server.com
```

**Solution:** Use http:// or https://:
```yaml
# ✅ Valid
servers:
  - id: my-server
    transport: http
    url: https://server.com
```

### "command should not be specified for sse/http transport"
**Problem:** Mixing stdio and network transport configurations.
```yaml
# ❌ Invalid
servers:
  - id: my-server
    transport: http
    url: https://api.example.com
    command: python  # Not allowed with http transport
```

**Solution:** Remove command and args for network transports:
```yaml
# ✅ Valid
servers:
  - id: my-server
    transport: http
    url: https://api.example.com
```

## Implementation Details

### Client Creation

GoFlow automatically creates the appropriate client based on the transport type:

- **stdio**: Creates `mcp.StdioClient` with subprocess management
- **sse**: Creates `mcp.SSEClient` with event stream handling
- **http**: Creates `mcp.HTTPClient` with synchronous request/response

### Connection Pooling

All transports support connection pooling with:
- Maximum 10 connections per server
- Automatic reconnection on failure
- Idle connection cleanup after 5 minutes

### Error Handling

Transport-specific errors are wrapped with context:
- Connection errors: Server unreachable, network issues
- Authentication errors: Invalid credentials, expired tokens
- Protocol errors: Invalid JSON-RPC, malformed responses

## Migration Guide

### From stdio-only to Multi-Transport

**Step 1:** Identify remote servers that should use HTTP/SSE:
```yaml
# Before: All stdio
servers:
  - id: local-server
    command: python
  - id: remote-server  # Should be HTTP
    command: curl
```

**Step 2:** Convert remote servers to HTTP/SSE transport:
```yaml
# After: Mixed transports
servers:
  - id: local-server
    transport: stdio
    command: python

  - id: remote-server
    transport: http
    url: https://api.example.com/mcp
```

**Step 3:** Update credentials from command args to headers:
```yaml
# Before: Credentials in command
servers:
  - id: api
    command: curl
    args: ["-H", "Authorization: Bearer token"]

# After: Credentials in headers
servers:
  - id: api
    transport: http
    url: https://api.example.com
    headers:
      Authorization: Bearer ${env.API_TOKEN}
```

## Testing

Validate your transport configuration:

```bash
# Validate workflow
goflow validate workflow.yaml

# Test server connections
goflow server test my-server

# Run workflow with verbose transport logging
goflow run --debug workflow.yaml
```

## Performance Considerations

### stdio Transport
- **Pros:** Low latency, direct process communication
- **Cons:** Limited to local servers, process overhead

### SSE Transport
- **Pros:** Real-time updates, persistent connections
- **Cons:** Server must support SSE, more complex error handling

### HTTP Transport
- **Pros:** Stateless, load balancer friendly, simple protocol
- **Cons:** Higher latency per request, no server push

Choose the transport that best matches your server architecture and performance requirements.
