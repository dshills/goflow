# Feature Specification: GoFlow - Visual MCP Workflow Orchestrator

**Feature Branch**: `001-goflow-spec-review`
**Created**: 2025-11-05
**Status**: Draft
**Input**: User description: "review ./specs/goflow-specification.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Execute Simple Workflow (Priority: P1)

A developer wants to automate a multi-step task involving reading a file from one MCP server, transforming the data, and writing it to another location, without writing custom code or using LLM mediation for each step.

**Why this priority**: This is the core value proposition - enabling basic workflow composition. Without this, GoFlow has no utility. This represents the minimum viable product.

**Independent Test**: Can be fully tested by installing GoFlow, registering two MCP servers (e.g., filesystem server), creating a 3-node workflow (read → transform → write), executing it, and verifying the output file contains transformed data.

**Acceptance Scenarios**:

1. **Given** GoFlow is installed and two MCP servers are registered, **When** user creates a workflow with read, transform, and write nodes and executes it, **Then** the workflow completes successfully and produces correct output
2. **Given** a workflow is created, **When** execution fails at any node, **Then** user receives clear error message identifying the failing node and reason
3. **Given** a multi-step workflow, **When** user executes it, **Then** execution completes at least 10x faster than manual LLM-mediated orchestration
4. **Given** a workflow definition file, **When** user runs it from command line, **Then** workflow executes without requiring interactive TUI

---

### User Story 2 - Visual Workflow Building in TUI (Priority: P2)

A developer wants to create workflows through an interactive visual interface rather than writing YAML by hand, enabling faster experimentation and reducing syntax errors.

**Why this priority**: While workflows can be created via YAML files (P1), a visual builder significantly improves usability and reduces the learning curve. This is essential for broader adoption but not blocking for basic functionality.

**Independent Test**: Can be tested by launching the TUI, adding nodes to a canvas, connecting them with edges, configuring node parameters through a property panel, saving the workflow, and verifying the generated YAML is valid and executable.

**Acceptance Scenarios**:

1. **Given** TUI is launched, **When** user adds workflow nodes and connects them, **Then** visual representation updates in real-time with no lag (< 16ms frame time)
2. **Given** a node is selected, **When** user edits its properties, **Then** changes are validated immediately with clear error messages for invalid configurations
3. **Given** a workflow is created in TUI, **When** user saves it, **Then** a valid YAML file is generated that can be executed via CLI
4. **Given** user navigates the TUI, **When** using vim-style keybindings (h/j/k/l), **Then** all interface elements are accessible without mouse

---

### User Story 3 - Conditional Logic and Data Transformation (Priority: P3)

A developer wants to create workflows with conditional branching (if-then-else) and data transformations (JSONPath queries, template strings) without LLM intervention, enabling intelligent automation.

**Why this priority**: Conditional logic and transformations elevate GoFlow from simple linear orchestration to intelligent automation. However, users can still get value from linear workflows (P1) without this feature.

**Independent Test**: Can be tested by creating a workflow with a condition node that branches based on data (e.g., if file size > 1MB), executing it with different inputs, and verifying the correct branch executes each time.

**Acceptance Scenarios**:

1. **Given** a workflow with a condition node, **When** the condition evaluates to true, **Then** the true branch executes and false branch is skipped
2. **Given** a transform node with JSONPath expression, **When** executed with valid JSON input, **Then** output contains correctly extracted/transformed data
3. **Given** a transform node with invalid expression, **When** workflow is validated, **Then** validation fails with specific error about the expression
4. **Given** a workflow with template strings (${variable}), **When** executed, **Then** variables are correctly interpolated from workflow context

---

### User Story 4 - Monitor and Debug Workflow Execution (Priority: P4)

A developer wants to see real-time execution progress, inspect variable values at each step, and view detailed error context when workflows fail, enabling rapid debugging.

**Why this priority**: Observability is critical for production use but not required for basic workflow creation and execution. Users can run workflows and see final results before needing detailed execution traces.

**Independent Test**: Can be tested by executing a workflow in watch mode, observing real-time node execution status updates, inspecting variable values at each step through the TUI, and viewing complete execution logs including all inputs/outputs.

**Acceptance Scenarios**:

1. **Given** a workflow is executing, **When** user views execution monitor, **Then** current node is highlighted and progress percentage is displayed
2. **Given** execution fails, **When** user inspects the failed node, **Then** full error context is shown including inputs, outputs, stack trace, and MCP server logs
3. **Given** a completed execution, **When** user views execution history, **Then** complete audit trail is available showing all variable changes and node I/O
4. **Given** a long-running workflow, **When** user monitors execution, **Then** performance metrics (time per node, resource usage) are displayed in real-time

---

### User Story 5 - Share and Reuse Workflows (Priority: P5)

A developer wants to export workflows as shareable YAML files without embedded secrets, import workflows created by others, and use workflow templates for common patterns, enabling team collaboration and knowledge sharing.

**Why this priority**: Workflow sharing amplifies value but requires workflows to be useful first (P1-P4). This enables team collaboration and community building but is not blocking for individual use.

**Independent Test**: Can be tested by exporting a workflow that uses MCP servers with credentials, sharing the YAML file with another user, having them import it, configure their own server credentials via keyring, and successfully execute the workflow.

**Acceptance Scenarios**:

1. **Given** a workflow with MCP server credentials, **When** user exports it, **Then** credentials are excluded and server references use IDs only
2. **Given** an exported workflow file, **When** another user imports it, **Then** workflow structure is preserved and user is prompted to configure required MCP servers
3. **Given** a workflow template is available, **When** user creates new workflow from template, **Then** template structure is copied with placeholder values for customization
4. **Given** user shares workflow via git, **When** teammate pulls and runs it, **Then** workflow executes correctly after teammate configures their local MCP servers

---

### User Story 6 - Parallel Execution and Loops (Priority: P6)

A developer wants to execute independent workflow branches concurrently for performance and iterate over collections (e.g., process each file in a directory), enabling advanced automation patterns.

**Why this priority**: Advanced control flow is valuable for complex automation but most workflows can be expressed with linear and conditional logic (P1, P3). This is optimization and advanced functionality.

**Independent Test**: Can be tested by creating a workflow with a parallel node containing two independent branches, executing it, and verifying both branches run concurrently and complete faster than sequential execution. Loop testing involves iterating over an array and verifying all items are processed.

**Acceptance Scenarios**:

1. **Given** a workflow with parallel branches, **When** executed, **Then** all branches run concurrently and execution time is comparable to the slowest branch (not sum of all)
2. **Given** a parallel execution fails in one branch, **When** error handling is configured, **Then** other branches can continue or all branches stop based on configuration
3. **Given** a loop node iterating over a collection, **When** executed, **Then** loop body executes once per item and results are collected
4. **Given** a loop with break condition, **When** condition is met, **Then** loop terminates early and partial results are available

---

### Edge Cases

- What happens when an MCP server becomes unreachable during workflow execution?
- How does the system handle workflows with circular dependencies in the graph?
- What happens when a transformation produces data that doesn't match the expected type for the next node?
- How does the system handle workflows that exceed resource limits (memory, execution time)?
- What happens when a user tries to execute a workflow with missing or invalid MCP server configurations?
- How does the system handle concurrent executions of the same workflow with different inputs?
- What happens when a workflow is modified while an execution is in progress?
- How does the system handle special characters or very large data values in variable interpolation?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to register MCP servers by providing server ID, command, and arguments
- **FR-002**: System MUST validate workflow definitions against schema before execution, catching circular dependencies, missing nodes, and invalid connections
- **FR-003**: System MUST execute workflows following topological order of nodes, respecting edge dependencies
- **FR-004**: System MUST support MCP protocol stdio transport for server communication
- **FR-005**: System MUST store MCP server credentials in system keyring, never in workflow files
- **FR-006**: System MUST provide workflow variable store for passing data between nodes with scoped visibility
- **FR-007**: System MUST support workflow definition in YAML format with versioning
- **FR-008**: System MUST generate execution logs with complete audit trail (node inputs, outputs, errors, timing)
- **FR-009**: System MUST validate workflow definitions in under 100ms for workflows with fewer than 100 nodes
- **FR-010**: System MUST support command-line execution of workflows without interactive UI
- **FR-011**: System MUST provide terminal user interface (TUI) for visual workflow building
- **FR-012**: System MUST support node types: start, end, MCP tool call, transform, condition
- **FR-013**: System MUST evaluate data transformation expressions (JSONPath, template strings) in sandboxed environment
- **FR-014**: System MUST handle execution errors gracefully with detailed error context
- **FR-015**: System MUST support workflow import/export preserving structure without secrets
- **FR-016**: System MUST provide real-time execution status updates during workflow runs
- **FR-017**: System MUST support workflow templates with parameterization
- **FR-018**: System MUST persist execution history in local storage
- **FR-019**: System MUST support concurrent execution of multiple workflow instances
- **FR-020**: System MUST provide retry policies for node failures
- **FR-021**: System MUST support parallel node type for concurrent branch execution
- **FR-022**: System MUST support loop node type for collection iteration
- **FR-023**: System MUST support SSE and HTTP transports for MCP protocol
- **FR-024**: System MUST provide vim-style keyboard navigation in TUI
- **FR-025**: System MUST render TUI at 60 FPS (< 16ms frame time)

### Key Entities

- **Workflow**: A directed acyclic graph of nodes and edges representing an automation task. Contains metadata (name, version, author), variable definitions, server configurations, and execution flow. Workflows are shareable, versionable, and executable.

- **Node**: A single step in a workflow. Types include: Start (entry point), End (exit point), MCP Tool (calls MCP server tool), Transform (data transformation), Condition (branching logic), Parallel (concurrent execution), Loop (iteration). Each node has inputs, outputs, and configuration.

- **Edge**: A connection between two nodes defining execution flow. May include conditional expressions for branching. Determines data flow and execution order.

- **Execution**: A single run of a workflow with specific inputs. Tracks execution state, variable values, node execution results, errors, and timing. Creates complete audit trail for debugging and compliance.

- **MCP Server**: An external Model Context Protocol server providing tools. Registered with GoFlow by ID, connection configuration, and credentials (stored in keyring). Health status and available tools are tracked.

- **Variable**: Data storage within workflow scope for passing values between nodes. Has name, type, and value. Supports variable interpolation in expressions.

- **Execution Context**: Runtime state for workflow execution including current node, variable store, execution trace, and error state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create and execute a basic 3-node workflow (read-transform-write) in under 5 minutes from installation
- **SC-002**: Workflow execution completes 10x faster than equivalent manual LLM-mediated orchestration (measured on standard multi-step tasks)
- **SC-003**: 90% of workflow validations complete in under 100ms for workflows with fewer than 100 nodes
- **SC-004**: TUI maintains 60 FPS rendering (< 16ms frame time) during workflow editing with up to 50 nodes visible
- **SC-005**: Workflow execution reduces LLM token usage by 80% compared to manual orchestration for multi-step MCP operations
- **SC-006**: System supports 50+ concurrent parallel branches within a single workflow without degradation
- **SC-007**: 90% of users successfully complete their first workflow on first attempt using TUI
- **SC-008**: Execution startup overhead is under 500ms from command invocation to first node execution
- **SC-009**: Per-node execution overhead is under 10ms (time between nodes, excluding MCP tool execution time)
- **SC-010**: Memory usage stays under 100MB base plus 10MB per active MCP server connection
- **SC-011**: System achieves 500+ GitHub stars within 6 months of public launch
- **SC-012**: Zero critical security vulnerabilities in pre-launch security audit
- **SC-013**: Complete execution audit trail is available for 100% of workflow runs (success or failure)
- **SC-014**: Exported workflows can be imported and executed by other users with 100% structural fidelity (after credential configuration)
- **SC-015**: System handles workflows with 1000+ nodes without performance degradation

### Assumptions

- Users have Go 1.21+ runtime available for installation
- Users are familiar with command-line interfaces and terminal usage
- MCP servers users want to orchestrate are compliant with MCP protocol specification
- Users have appropriate permissions on their system to access keyring for credential storage
- Workflows will primarily contain 10-50 nodes (optimized for this range, but support up to 1000+)
- Users have basic understanding of JSON structure for data transformation expressions
- System keyring is available on user's platform (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- MCP servers respond within reasonable timeouts (configurable, default 30s per tool call)
- Users have network access if using remote MCP servers via SSE/HTTP transports
