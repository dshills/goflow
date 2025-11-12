# Feature Specification: Code Review Remediation

**Feature Branch**: `002-pr-review-remediation`
**Created**: 2025-11-12
**Status**: Draft
**Input**: User description: "Address critical and high-priority issues from code review report"

## User Scenarios & Testing

### User Story 1 - Secure File Operations (Priority: P1)

Developers need the test server to safely handle file operations without exposing the system to directory traversal attacks or arbitrary file access vulnerabilities. When a malicious or misconfigured client attempts to access files outside intended boundaries, the system must reject these operations and log the security incident.

**Why this priority**: Critical security vulnerability that could allow unauthorized access to sensitive system files (credentials, SSH keys, configuration files). This is the highest risk issue in the codebase.

**Independent Test**: Can be fully tested by attempting file operations with various malicious paths (../../../etc/passwd, absolute paths to system files, symbolic link traversals) and verifying they are blocked with appropriate security errors.

**Acceptance Scenarios**:

1. **Given** a test server is running, **When** a client requests to read a file using a relative path with directory traversal (e.g., "../../etc/passwd"), **Then** the operation is rejected with a security error and the attempt is logged
2. **Given** a test server is running, **When** a client requests to read a file using an absolute path to a system file (e.g., "/etc/passwd"), **Then** the operation is rejected with a security error
3. **Given** a test server is running, **When** a client requests to read a file within the allowed directory, **Then** the operation succeeds normally
4. **Given** a test server is running, **When** a client attempts to write to a file outside the allowed directory, **Then** the operation is rejected with a security error

---

### User Story 2 - Workflow Execution Timeout Protection (Priority: P1)

System operators need workflows to complete or timeout within reasonable bounds to prevent resource exhaustion and system hangs. When a workflow contains nodes that block indefinitely, the execution should timeout gracefully with clear error messaging.

**Why this priority**: Critical operational issue that can cause system hangs, resource exhaustion, and inability to recover without manual intervention. Affects reliability and availability.

**Independent Test**: Can be fully tested by creating a workflow with a deliberately blocking node, executing it with various timeout configurations, and verifying the execution terminates with appropriate timeout errors.

**Acceptance Scenarios**:

1. **Given** a workflow with a node that blocks indefinitely, **When** the workflow is executed with a 30-second timeout, **Then** the execution terminates after 30 seconds with a timeout error
2. **Given** a workflow that completes normally in 10 seconds, **When** executed with a 60-second timeout, **Then** the workflow completes successfully without timeout
3. **Given** a workflow execution times out, **When** inspecting the execution logs, **Then** clear timeout information is provided including which node was executing when the timeout occurred

---

### User Story 3 - MCP Connection Pool API Consistency (Priority: P1)

Developers using the MCP connection pool need consistent API signatures across the codebase to ensure compilation succeeds and runtime behavior is predictable. When calling connection pool methods, the signatures must match between implementation and usage sites.

**Why this priority**: Critical compilation issue that prevents the code from building. This is a functional bug that blocks all development and deployment.

**Independent Test**: Can be fully tested by running `go build ./...` and verifying compilation succeeds, followed by integration tests that exercise connection pool acquisition and release patterns.

**Acceptance Scenarios**:

1. **Given** the connection pool implementation, **When** health check code calls Get() and Release(), **Then** the code compiles without type errors
2. **Given** a connection is acquired from the pool, **When** it is released back to the pool, **Then** the connection is properly returned and available for reuse
3. **Given** multiple concurrent requests for connections, **When** using the connection pool, **Then** all requests succeed with consistent behavior

---

### User Story 4 - Secure Expression Evaluation (Priority: P1)

Developers need JSONPath filter expressions to be evaluated safely without introducing injection vulnerabilities. When users provide filter expressions in JSONPath queries, these expressions must be parsed and executed using robust, security-tested libraries rather than hand-rolled evaluators.

**Why this priority**: Critical security issue with hand-rolled expression parser that could be exploited through injection attacks or logic bypasses. Affects data security and system integrity.

**Independent Test**: Can be fully tested by providing various malicious filter expressions and verifying they are handled safely, compared against behavior of the existing secure expression evaluator.

**Acceptance Scenarios**:

1. **Given** a JSONPath query with a filter expression, **When** the expression is evaluated, **Then** it is processed using a security-tested library with proper sandboxing
2. **Given** a malicious filter expression designed to exploit parsing ambiguities, **When** the expression is evaluated, **Then** it is rejected or handled safely without causing injection vulnerabilities
3. **Given** legitimate filter expressions from existing workflows, **When** migrating to the secure evaluator, **Then** all expressions continue to work correctly with equivalent results

---

### User Story 5 - Terminal Input Handling (Priority: P1)

Users interacting with the TUI need keyboard input to be read reliably across all platforms without compilation errors. The system must use platform-compatible methods for non-blocking terminal input that work on Linux, macOS, and Windows.

**Why this priority**: Critical compilation issue that prevents the TUI from building on any platform. Blocks all interactive use cases.

**Independent Test**: Can be fully tested by compiling the TUI on each target platform and running interactive sessions with various keyboard inputs.

**Acceptance Scenarios**:

1. **Given** the TUI is built on macOS, **When** compilation occurs, **Then** the build succeeds without errors related to SetReadDeadline
2. **Given** the TUI is running, **When** a user presses keyboard keys, **Then** the input is received and processed without blocking indefinitely
3. **Given** the TUI is idle waiting for input, **When** a timeout period elapses, **Then** the input routine can be cancelled gracefully

---

### User Story 6 - Keyboard Binding Type Safety (Priority: P1)

Developers defining keyboard bindings need type-safe APIs that compile correctly and provide predictable behavior. When registering global keyboard bindings, the system must use consistent types that match the underlying data structures.

**Why this priority**: Critical compilation issue preventing the keyboard handling system from building. Blocks all TUI functionality.

**Independent Test**: Can be fully tested by running `go build ./pkg/tui/...` and verifying compilation succeeds, followed by TUI tests that exercise keyboard bindings.

**Acceptance Scenarios**:

1. **Given** keyboard binding definitions, **When** code calls GetAllBindings(), **Then** the code compiles without type mismatch errors
2. **Given** global keyboard bindings are registered, **When** accessing all bindings, **Then** global bindings are included with correct type handling
3. **Given** mode-specific bindings are registered, **When** switching between modes, **Then** appropriate bindings are active and accessible

---

### User Story 7 - Enhanced Error Context (Priority: P2)

Developers debugging issues need comprehensive error information that includes context about what operation was being attempted when the error occurred. When errors are wrapped, they should preserve the original error while adding relevant contextual information.

**Why this priority**: High-priority code quality issue affecting debuggability and maintainability. Speeds up issue resolution significantly but not a blocking functional issue.

**Independent Test**: Can be fully tested by triggering various error conditions and verifying error messages include contextual information like operation type, resource identifiers, and parameter values.

**Acceptance Scenarios**:

1. **Given** a workflow execution fails, **When** inspecting the error, **Then** the error message includes the workflow ID, node ID, and operation being performed
2. **Given** an MCP connection fails, **When** the error is logged, **Then** it includes the server ID, connection parameters, and underlying network error
3. **Given** a validation error occurs, **When** the user sees the error, **Then** it clearly indicates which validation rule failed and what value caused the failure

---

### User Story 8 - Connection Pool Cleanup (Priority: P2)

System operators need proper resource cleanup when connections are no longer needed to prevent connection leaks and resource exhaustion. When the connection pool shuts down or connections become stale, all resources must be properly closed and released.

**Why this priority**: High-priority reliability issue that can lead to resource leaks over time but doesn't immediately break functionality.

**Independent Test**: Can be fully tested by monitoring connection counts, file descriptors, and goroutines during pool lifecycle operations and verifying no leaks occur.

**Acceptance Scenarios**:

1. **Given** a connection pool with active connections, **When** the pool is shut down, **Then** all connections are closed and resources are freed
2. **Given** a connection becomes stale, **When** health checks detect the staleness, **Then** the connection is removed from the pool and closed
3. **Given** connections are acquired and released repeatedly, **When** monitoring system resources, **Then** no resource leaks are detected over extended operation

---

### User Story 9 - Consistent Nil/Error Checks (Priority: P2)

Developers need consistent error and nil checking patterns throughout the codebase to prevent panics and undefined behavior. When code paths can potentially return nil or errors, these conditions must be checked before using the values.

**Why this priority**: High-priority code quality issue that prevents runtime panics. Important for stability but existing issues may be in rarely-exercised code paths.

**Independent Test**: Can be fully tested by creating test cases that trigger all identified code paths with nil values or error conditions and verifying proper handling without panics.

**Acceptance Scenarios**:

1. **Given** a function that can return nil, **When** the result is used, **Then** it is checked for nil before dereferencing
2. **Given** a function that returns an error, **When** it is called, **Then** the error is checked before using the return value
3. **Given** edge cases that produce nil or error results, **When** the code executes, **Then** no panics occur and errors are handled gracefully

---

### User Story 10 - Type Safety Improvements (Priority: P2)

Developers need type-safe interfaces that eliminate unnecessary type assertions and potential runtime panics. When working with strongly-typed interfaces, the system should leverage compile-time type checking rather than runtime assertions.

**Why this priority**: High-priority code quality issue improving type safety and preventing runtime panics. Affects code maintainability and reliability.

**Independent Test**: Can be fully tested by removing type assertions, running compilation to verify type safety, and executing comprehensive test suites to verify behavior.

**Acceptance Scenarios**:

1. **Given** code using type assertions on known types, **When** refactored to use direct typing, **Then** compilation succeeds with stronger type guarantees
2. **Given** refactored type-safe code, **When** running all tests, **Then** behavior remains identical to previous implementation
3. **Given** interfaces with known implementing types, **When** used in type-safe manner, **Then** no runtime type assertion panics are possible

---

### Edge Cases

- What happens when a file operation validation check encounters a symbolic link that points outside the allowed directory?
- How does the system handle workflows that timeout while a node is holding locks or other resources?
- What occurs when connection pool cleanup is triggered while connections are actively being used?
- How does error wrapping behave with deeply nested error chains (10+ levels)?
- What happens when keyboard input handling encounters platform-specific terminal modes or configurations?
- How does the system handle concurrent access to the connection pool during shutdown?
- What occurs when JSONPath filter expressions reference undefined properties or use malformed syntax?
- How does type assertion refactoring handle interfaces with multiple possible implementations?
- What happens when timeout contexts are cancelled while multiple goroutines are executing?
- How does the system behave when file paths contain Unicode characters, null bytes, or other special characters?

## Requirements

### Functional Requirements

- **FR-001**: Test server MUST validate all file operation paths against an allowed directory whitelist before performing any file system operations
- **FR-002**: Test server MUST reject file operations that use relative paths with directory traversal sequences (../)
- **FR-003**: Test server MUST reject file operations that use absolute paths outside the allowed directory
- **FR-004**: Test server MUST log all rejected file operation attempts with relevant security context
- **FR-005**: Workflow execution runtime MUST accept configurable timeout durations for workflow executions
- **FR-006**: Workflow execution MUST terminate and return timeout errors when the configured timeout is exceeded
- **FR-007**: Timeout errors MUST include context about which node was executing when the timeout occurred
- **FR-008**: Connection pool MUST provide consistent API signatures for Get() and Release() methods across all usage sites
- **FR-009**: Connection pool Get() method MUST accept context and server identifier parameters that match caller expectations
- **FR-010**: Connection pool Release() method MUST accept parameters that match how callers invoke the method
- **FR-011**: JSONPath filter evaluation MUST use a security-tested expression evaluation library rather than hand-rolled parsers
- **FR-012**: JSONPath filter evaluation MUST provide sandboxed execution that prevents injection attacks
- **FR-013**: Terminal input handling MUST use platform-compatible methods that compile on all target platforms (Linux, macOS, Windows)
- **FR-014**: Terminal input reading MUST support non-blocking operation with cancellation support
- **FR-015**: Keyboard binding registration MUST use type-safe APIs that compile without type mismatch errors
- **FR-016**: Keyboard binding system MUST handle global and mode-specific bindings with consistent types
- **FR-017**: Error messages MUST include contextual information about the operation being performed when the error occurred
- **FR-018**: Error wrapping MUST preserve original error information while adding context
- **FR-019**: Connection pool MUST properly close and release all connections during shutdown
- **FR-020**: Connection pool MUST remove and close stale connections when detected by health checks
- **FR-021**: All code paths MUST check for nil values before dereferencing pointers
- **FR-022**: All code paths MUST check for errors before using return values from error-returning functions
- **FR-023**: Code MUST use direct typing rather than type assertions when types are known at compile time
- **FR-024**: Refactored type-safe code MUST maintain behavioral compatibility with previous implementations

### Key Entities

- **File Operation Request**: Represents a client request to read or write files, including the requested path, operation type, and client identifier
- **File Path Validator**: Validates file paths against security policies, including whitelist checking, directory traversal detection, and symbolic link resolution
- **Workflow Execution Context**: Contains workflow execution state including timeout configuration, current node, execution start time, and cancellation signals
- **Connection Pool Entry**: Represents a pooled MCP connection including the client instance, server metadata, health status, and lifecycle state
- **JSONPath Filter Expression**: Represents a user-provided filter expression to be evaluated against JSON data, including the expression string, parsed AST, and evaluation context
- **Terminal Input Handler**: Manages platform-specific terminal input operations including non-blocking reads, timeout handling, and cancellation
- **Keyboard Binding**: Represents a key sequence mapped to an action, including the key combination, mode specificity, and handler function
- **Error Context**: Contains contextual information about an error including operation type, resource identifiers, parameter values, and timestamp

## Success Criteria

### Measurable Outcomes

- **SC-001**: All 8 critical issues identified in the code review are resolved and verified through tests
- **SC-002**: All 45 high-priority issues are addressed or explicitly documented as accepted risks
- **SC-003**: Codebase compiles successfully on all target platforms (Linux, macOS, Windows) with `go build ./...`
- **SC-004**: All existing tests continue to pass after remediation changes
- **SC-005**: Security tests successfully block malicious file path attempts with 100% detection rate
- **SC-006**: Workflow timeouts trigger reliably within 5% of configured timeout duration
- **SC-007**: Connection pool API changes result in zero compilation errors across the codebase
- **SC-008**: Zero runtime panics occur from nil dereferences or missing error checks in affected code paths
- **SC-009**: Performance tests show no degradation (< 5% slowdown) after remediation changes
- **SC-010**: Code coverage for remediated areas reaches at least 80%

## Assumptions

- The test server is intended only for development/testing use and will have a clearly defined allowed directory (defaults to a temporary directory or explicitly configured path)
- Workflow timeout configuration will be provided either in the workflow definition or as a system-level default (assumed default: 5 minutes)
- The connection pool is used in multi-threaded contexts and must be thread-safe
- JSONPath filter expressions are used in user-facing workflows and are considered untrusted input
- The TUI must work on terminal emulators commonly used by developers (iTerm2, Terminal.app, GNOME Terminal, Windows Terminal, etc.)
- Keyboard bindings follow standard conventions (Vim-style keybindings as documented in CLAUDE.md)
- Error context should be detailed enough for debugging but not expose sensitive information in user-facing messages
- Connection pool cleanup should be graceful but force-close connections if graceful shutdown takes longer than a reasonable timeout (assumed: 30 seconds)
- Type safety improvements should not change external API contracts
- The existing test suite provides reasonable coverage of affected code paths

## Dependencies

- Security-tested expression evaluation library (expr-lang is already in use for expression.go)
- Platform-compatible terminal input library (goterm is already in use, may need enhancement or alternative library consultation)
- Testing infrastructure to verify file path validation (likely standard library filepath package enhancements)
- Integration test infrastructure for MCP connection pool testing
- Performance benchmarking tools to verify no regressions after changes

## Constraints

- Changes must maintain backward compatibility with existing workflow definitions
- Performance impact must be minimal (< 5% overhead on affected operations)
- Security fixes must not break legitimate use cases
- All changes must follow Go idioms and best practices
- Changes must compile and pass tests on Go 1.21+
- Remediation work should not introduce new dependencies unless absolutely necessary
- File path validation must work correctly on both Unix-like systems (Linux, macOS) and Windows with different path separators and conventions
- Timeout implementation must not interfere with proper cleanup of resources held by workflow nodes
- Error context must not expose sensitive information (credentials, internal paths, etc.)
