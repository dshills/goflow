# MCP Protocol Client

This package provides a complete JSON-RPC 2.0 client implementation for the Model Context Protocol (MCP) over stdio transport.

## Architecture

The package follows Domain-Driven Design principles with clear separation of concerns:

```
pkg/mcp/
├── types.go              # Client interface and configuration types
├── jsonrpc.go            # JSON-RPC 2.0 protocol types and helpers
├── stdio_client.go       # StdioClient implementation
├── connection_pool.go    # Connection pooling and reuse
└── health.go             # Health monitoring and periodic checks
```

## Components

### StdioClient

Implements the `Client` interface using stdio transport (spawns subprocess and communicates via stdin/stdout).

**Key Features:**
- Spawns MCP server as subprocess with command/args
- Manages stdin/stdout/stderr pipes
- Handles JSON-RPC 2.0 request/response correlation
- Background goroutine for reading responses
- Proper cleanup on close
- Context support for timeouts and cancellation

**Connection Lifecycle:**
1. `NewStdioClient(config)` - Create client
2. `Connect(ctx)` - Spawn process, setup pipes, send initialize
3. `ListTools(ctx)` - Discover available tools
4. `CallTool(ctx, name, params)` - Invoke tools
5. `Ping(ctx)` - Health check
6. `Close()` - Terminate process and cleanup

### Connection Pool

Manages multiple concurrent connections to MCP servers with pooling and reuse.

**Features:**
- Per-server connection pools (max 10 connections per server)
- Automatic connection reuse for idle connections
- Connection health tracking
- Automatic cleanup of idle connections (5 minute timeout)
- Thread-safe Get/Release operations
- Graceful shutdown of all connections

**Usage:**
```go
pool := mcp.NewConnectionPool()
pool.RegisterServer(config)

// Get connection from pool (creates if needed)
client, err := pool.Get(ctx, "server-id")
defer pool.Release("server-id", client)

// Use client
tools, err := client.ListTools(ctx)
```

### Health Monitor

Periodic health checking of registered MCP servers.

**Features:**
- Background health checks every 30 seconds
- Configurable health check timeout (5 seconds)
- Failed check tracking (unhealthy after 3 failures)
- Per-server health status with timestamps
- Manual health status updates
- Thread-safe concurrent checks

**Usage:**
```go
monitor := mcp.NewHealthMonitor(pool)
monitor.RegisterServer("server-id")

// Get current health status
health, ok := monitor.GetHealth("server-id")
if ok && health.IsHealthy {
    // Server is healthy
}

// Perform immediate check
err := monitor.CheckNow(ctx, "server-id")
```

## JSON-RPC 2.0 Protocol

The client implements JSON-RPC 2.0 with these methods:

### initialize
Sent automatically during `Connect()`. Establishes protocol version and capabilities.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "goflow",
      "version": "0.1.0"
    }
  }
}
```

### tools/list
Discovers available tools from the server.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": [
      {
        "name": "echo",
        "description": "Echoes back the input",
        "inputSchema": {
          "type": "object",
          "properties": {
            "message": {"type": "string"}
          },
          "required": ["message"]
        }
      }
    ]
  }
}
```

### tools/call
Invokes a tool with parameters.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "tools/call",
  "params": {
    "name": "echo",
    "arguments": {
      "message": "Hello, MCP!"
    }
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Hello, MCP!"
      }
    ]
  }
}
```

### ping
Health check to verify server responsiveness.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "method": "ping",
  "params": {}
}
```

## Error Handling

The implementation handles four types of errors:

1. **Connection Errors** - Failed to start process, pipe creation failures
2. **Protocol Errors** - Invalid JSON, malformed JSON-RPC responses
3. **Server Errors** - JSON-RPC error responses from the server
4. **Timeout Errors** - Context deadline exceeded, no response received

All errors include context and are wrapped for easy error chain inspection.

## Concurrency

The implementation is fully thread-safe:

- **StdioClient**: Mutex protects pending requests map and connection state
- **ConnectionPool**: RWMutex for pool operations, per-connection mutexes
- **HealthMonitor**: RWMutex for health status map
- **Background goroutines**: Scanner reads responses, cleanup timers manage idle connections

## Testing

The package includes comprehensive tests:

- **Unit tests**: JSON-RPC marshaling, ID comparison, scanner behavior
- **Integration tests**: Full client lifecycle with mock MCP server
- **Connection pool tests**: Concurrent access, reuse, cleanup
- **Health monitor tests**: Periodic checks, failure tracking

Run tests:
```bash
go test ./pkg/mcp -v
```

## Performance Characteristics

- **Connection overhead**: < 50ms to spawn subprocess and initialize
- **Request latency**: < 10ms for simple requests (excluding server processing)
- **Memory per connection**: ~1MB (subprocess overhead + buffers)
- **Max connections**: 10 per server (configurable via MaxConnectionsPerServer)
- **Idle cleanup**: 5 minutes (configurable via ConnectionIdleTimeout)

## Implementation Notes

### Request ID Management
Request IDs are string-formatted sequential integers to avoid type mismatch issues. JSON unmarshaling converts numeric IDs to float64, which doesn't compare equal to uint64 in Go's type system.

### Lock Strategy
Locks are released before I/O operations to prevent deadlocks. The `Connect` method releases its lock before calling `initialize` to avoid holding the lock during the request/response cycle.

### Scanner Blocking
`bufio.Scanner` blocks until EOF or a newline. The background `readResponses` goroutine runs continuously, routing responses to waiting requests via channels.

### Cleanup
The `Close` method kills the subprocess and closes all pipes. Pending requests receive closed channel signals. The connection pool's cleanup goroutine runs every minute to close idle connections.

## Future Enhancements

Potential improvements for future iterations:

1. **SSE Transport** - Add Server-Sent Events transport alongside stdio
2. **HTTP Transport** - Add HTTP JSON-RPC transport
3. **Reconnection** - Automatic reconnection on connection failures
4. **Retry Logic** - Configurable retry with exponential backoff
5. **Metrics** - Request latency, error rates, connection pool stats
6. **Tracing** - OpenTelemetry spans for observability
