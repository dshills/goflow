# Tasks: GoFlow - Visual MCP Workflow Orchestrator

**Input**: Design documents from `/specs/001-goflow-spec-review/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/, research.md, quickstart.md

**Tests**: This project follows test-first development (constitutional requirement). Tests MUST be written before implementation and MUST fail before writing code.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Repository root** for Go CLI project
- **pkg/** for domain logic and public packages
- **cmd/goflow/** for CLI entry point
- **tests/** for all test types (unit, integration, TUI)
- **internal/** for private utilities

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic Go module structure

- [X] T001 Initialize Go module with go.mod (go mod init github.com/dshills/goflow)
- [X] T002 Create project directory structure per plan.md (cmd/, pkg/, tests/, internal/)
- [X] T003 [P] Configure golangci-lint with .golangci.yml
- [X] T004 [P] Add go.mod dependencies: goterm, yaml.v3, gjson, expr, errgroup, modernc.org/sqlite
- [X] T005 [P] Create .gitignore for Go project (vendor/, *.test, coverage.txt, goflow binary)
- [X] T006 [P] Setup pre-commit hook script in .git/hooks/pre-commit (lint, test, mcp-pr review)
- [X] T007 [P] Create internal/testutil/fixtures/ directory with sample workflow YAML files
- [X] T008 [P] Create internal/testutil/mocks/ directory for mock MCP servers

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure and domain types that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests for Foundational Phase

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T009 [P] Unit test for Workflow entity validation in tests/unit/workflow/workflow_test.go
- [X] T010 [P] Unit test for Node value objects in tests/unit/workflow/node_test.go
- [X] T011 [P] Unit test for Edge validation in tests/unit/workflow/edge_test.go
- [X] T012 [P] Unit test for Variable validation in tests/unit/workflow/variable_test.go
- [X] T013 [P] Unit test for Execution entity in tests/unit/execution/execution_test.go
- [X] T014 [P] Unit test for ExecutionContext in tests/unit/execution/context_test.go
- [X] T015 [P] Unit test for MCPServer entity in tests/unit/mcpserver/server_test.go
- [X] T016 [P] Unit test for expression evaluation in tests/unit/transform/expression_test.go
- [X] T017 [P] Unit test for JSONPath queries in tests/unit/transform/jsonpath_test.go
- [X] T018 [P] Unit test for template interpolation in tests/unit/transform/template_test.go

### Workflow Aggregate Implementation

- [X] T019 [P] Create WorkflowID, NodeID, EdgeID type aliases in pkg/workflow/types.go
- [X] T020 [P] Create WorkflowMetadata struct in pkg/workflow/workflow.go
- [X] T021 Implement Workflow root entity with invariants in pkg/workflow/workflow.go (depends on T019, T020)
- [X] T022 [P] Implement StartNode value object in pkg/workflow/node.go
- [X] T023 [P] Implement EndNode value object in pkg/workflow/node.go
- [X] T024 [P] Implement MCPToolNode value object in pkg/workflow/node.go
- [X] T025 [P] Implement TransformNode value object in pkg/workflow/node.go
- [X] T026 [P] Implement ConditionNode value object in pkg/workflow/node.go
- [X] T027 [P] Implement ParallelNode value object in pkg/workflow/node.go
- [X] T028 [P] Implement LoopNode value object in pkg/workflow/node.go
- [X] T029 Implement Edge value object in pkg/workflow/edge.go
- [X] T030 Implement Variable value object in pkg/workflow/variable.go
- [X] T031 Implement ServerConfig value object in pkg/workflow/server_config.go
- [X] T032 Implement Workflow.Validate() with all invariant checks in pkg/workflow/workflow.go (depends on T021-T031)
- [X] T033 Implement Workflow.AddNode() with validation in pkg/workflow/workflow.go (depends on T032)
- [X] T034 Implement Workflow.AddEdge() with validation in pkg/workflow/workflow.go (depends on T032)
- [X] T035 Implement WorkflowRepository interface in pkg/workflow/repository.go

### Execution Aggregate Implementation

- [X] T036 [P] Create ExecutionID, NodeExecutionID type aliases in pkg/execution/types.go
- [X] T037 Implement Execution root entity in pkg/execution/execution.go
- [X] T038 Implement ExecutionContext with variable store in pkg/execution/context.go
- [X] T039 Implement NodeExecution value object in pkg/execution/node_execution.go
- [X] T040 Implement VariableSnapshot value object in pkg/execution/variable_snapshot.go
- [X] T041 Implement ExecutionRepository interface in pkg/execution/repository.go

### MCP Server Registry Aggregate Implementation

- [X] T042 [P] Create ServerID, ToolID type aliases in pkg/mcpserver/types.go
- [X] T043 Implement MCPServer root entity in pkg/mcpserver/server.go
- [X] T044 Implement Tool value object in pkg/mcpserver/tool.go
- [X] T045 Implement ToolSchema value object in pkg/mcpserver/tool_schema.go
- [X] T046 Implement Connection value object in pkg/mcpserver/connection.go
- [X] T047 Implement ServerRepository interface in pkg/mcpserver/repository.go

### Transformation Engine Implementation

- [X] T048 [P] Implement JSONPath query evaluator using gjson in pkg/transform/jsonpath.go
- [X] T049 [P] Implement expression evaluator using expr-lang in pkg/transform/expression.go
- [X] T050 [P] Implement template string interpolation in pkg/transform/template.go
- [X] T051 Create unified Transform() function in pkg/transform/transform.go (depends on T048-T050)

### Storage Layer Implementation

- [X] T052 Implement FilesystemWorkflowRepository (YAML) in pkg/storage/filesystem.go
- [X] T053 Implement SQLiteExecutionRepository in pkg/storage/sqlite.go
- [X] T054 Implement KeyringCredentialStore (OS keyring) in pkg/storage/keyring.go
- [X] T055 Create SQLite schema migrations in pkg/storage/migrations.go (executions, node_executions, variable_snapshots tables)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Create and Execute Simple Workflow (Priority: P1) 🎯 MVP

**Goal**: Enable users to create a 3-node workflow (read → transform → write) via YAML and execute it from CLI without TUI

**Independent Test**: Install GoFlow, register filesystem MCP server, create YAML workflow with read/transform/write nodes, execute via `goflow run`, verify output file has transformed data

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T056 [P] [US1] Integration test for YAML workflow parsing in tests/integration/workflow_parse_test.go
- [X] T057 [P] [US1] Integration test for workflow validation in tests/integration/workflow_validation_test.go
- [X] T058 [P] [US1] Integration test for MCP protocol stdio transport in tests/integration/mcp_stdio_test.go
- [X] T059 [P] [US1] Integration test for read-transform-write workflow execution in tests/integration/workflow_execution_test.go
- [X] T060 [P] [US1] Unit test for CLI run command in tests/unit/cli/run_test.go
- [X] T061 [P] [US1] Unit test for CLI server command in tests/unit/cli/server_test.go

### Workflow Parsing Implementation

- [X] T062 [US1] Implement YAML workflow parser in pkg/workflow/parser.go
- [X] T063 [US1] Implement YAML workflow serializer (ToYAML) in pkg/workflow/parser.go
- [X] T064 [US1] Add schema validation against workflow-schema-v1.json in pkg/workflow/validator.go
- [X] T065 [US1] Implement topological sort for workflow validation in pkg/workflow/validator.go

### MCP Protocol Client Implementation

- [X] T066 [P] [US1] Implement MCP stdio transport in pkg/mcpserver/client.go
- [X] T067 [P] [US1] Implement MCP tool discovery via introspection in pkg/mcpserver/client.go
- [X] T068 [P] [US1] Implement MCP tool invocation in pkg/mcpserver/client.go
- [X] T069 [US1] Implement connection pooling in pkg/mcpserver/connection_pool.go (depends on T066-T068)
- [X] T070 [US1] Implement health check pings in pkg/mcpserver/health.go

### Workflow Execution Runtime

- [X] T071 [US1] Implement workflow runtime engine in pkg/execution/runtime.go
- [X] T072 [US1] Implement MCPToolNode executor in pkg/execution/node_executor.go
- [X] T073 [US1] Implement TransformNode executor in pkg/execution/node_executor.go
- [X] T074 [US1] Implement StartNode and EndNode executors in pkg/execution/node_executor.go
- [X] T075 [US1] Implement error context collection in pkg/execution/error.go
- [X] T076 [US1] Implement execution logging to SQLite in pkg/execution/logger.go

### CLI Commands Implementation

- [X] T077 [P] [US1] Implement root CLI command structure in pkg/cli/root.go
- [X] T078 [P] [US1] Implement `goflow server add` command in pkg/cli/server.go
- [X] T079 [P] [US1] Implement `goflow server list` command in pkg/cli/server.go
- [X] T080 [P] [US1] Implement `goflow server test` command in pkg/cli/server.go
- [X] T081 [P] [US1] Implement `goflow validate` command in pkg/cli/validate.go
- [X] T082 [US1] Implement `goflow run` command in pkg/cli/run.go (depends on T071-T076)
- [X] T083 [US1] Create CLI entry point main.go in cmd/goflow/main.go (depends on T077-T082)

### Documentation and Examples

- [X] T084 [P] [US1] Create example workflow YAML for read-transform-write in examples/simple-pipeline.yaml
- [X] T085 [P] [US1] Verify quickstart.md tutorial works end-to-end

**Checkpoint**: ✅ User Story 1 COMPLETE - users can create and execute workflows via CLI

---

## Phase 4: User Story 2 - Visual Workflow Building in TUI (Priority: P2)

**Goal**: Enable users to create workflows through interactive TUI with visual node editor, real-time validation, and vim-style keybindings

**Independent Test**: Launch TUI, add nodes to canvas, connect edges, configure node parameters, save workflow, verify generated YAML is valid and executable

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T086 [P] [US2] TUI component test for workflow explorer view in tests/tui/explorer_test.go
- [X] T087 [P] [US2] TUI component test for workflow builder view in tests/tui/builder_test.go
- [X] T088 [P] [US2] TUI interaction test for node addition in tests/tui/node_operations_test.go
- [X] T089 [P] [US2] TUI interaction test for edge creation in tests/tui/edge_operations_test.go
- [X] T090 [P] [US2] TUI interaction test for vim keybindings in tests/tui/keyboard_test.go

### TUI Foundation

- [X] T091 [US2] Create TUI application root using goterm in pkg/tui/app.go
- [X] T092 [P] [US2] Implement reusable TUI components (buttons, panels, modals) in pkg/tui/components/
- [X] T093 [P] [US2] Implement vim-style keyboard handler in pkg/tui/keyboard.go
- [X] T094 [P] [US2] Implement view switching system in pkg/tui/views.go

### Workflow Explorer View

- [X] T095 [US2] Implement workflow list view in pkg/tui/workflow_explorer.go
- [X] T096 [US2] Implement workflow creation dialog in pkg/tui/workflow_explorer.go
- [X] T097 [US2] Implement workflow deletion confirmation in pkg/tui/workflow_explorer.go
- [X] T098 [US2] Implement workflow rename functionality in pkg/tui/workflow_explorer.go

### Workflow Builder View

- [X] T099 [US2] Implement canvas rendering for workflow graph in pkg/tui/workflow_builder.go
- [X] T100 [US2] Implement node palette (add node menu) in pkg/tui/workflow_builder.go
- [X] T101 [US2] Implement node selection and highlighting in pkg/tui/workflow_builder.go
- [X] T102 [US2] Implement edge creation mode in pkg/tui/workflow_builder.go
- [X] T103 [US2] Implement node property panel in pkg/tui/workflow_builder.go
- [X] T104 [US2] Implement real-time validation display in pkg/tui/workflow_builder.go
- [X] T105 [US2] Implement workflow save/load in pkg/tui/workflow_builder.go

### CLI Integration

- [X] T106 [US2] Implement `goflow edit` command to launch TUI in pkg/cli/edit.go
- [X] T107 [US2] Implement `goflow init` command to create workflow template in pkg/cli/init.go

### Performance Optimization

- [X] T108 [US2] Implement incremental canvas rendering (<16ms frame time) in pkg/tui/renderer.go
- [X] T109 [US2] Add performance profiling for TUI operations in pkg/tui/profiler.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work - users can create workflows via TUI or YAML and execute them

---

## Phase 5: User Story 3 - Conditional Logic and Data Transformation (Priority: P3)

**Goal**: Enable workflows with conditional branching (if-then-else) and advanced data transformations (JSONPath, templates, expressions)

**Independent Test**: Create workflow with condition node that branches based on data (e.g., file size > 1MB), execute with different inputs, verify correct branch executes

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T110 [P] [US3] Integration test for condition node evaluation in tests/integration/condition_test.go
- [X] T111 [P] [US3] Integration test for transform node with JSONPath in tests/integration/transform_jsonpath_test.go
- [X] T112 [P] [US3] Integration test for transform node with templates in tests/integration/transform_template_test.go
- [X] T113 [P] [US3] Unit test for expression parser and validator in tests/unit/transform/parser_test.go

### Conditional Execution Implementation

- [X] T114 [US3] Implement ConditionNode executor with branching logic in pkg/execution/node_executor.go
- [X] T115 [US3] Implement conditional edge evaluation in pkg/execution/runtime.go
- [X] T116 [US3] Add support for boolean expressions in transform engine in pkg/transform/expression.go

### Enhanced Transformation Support

- [X] T117 [P] [US3] Add advanced JSONPath operators (filters, recursive descent) in pkg/transform/jsonpath.go
- [X] T118 [P] [US3] Add template helper functions (upper, lower, format) in pkg/transform/template.go
- [X] T119 [P] [US3] Add type conversion utilities in pkg/transform/type_conversion.go
- [X] T120 [US3] Implement expression validation at workflow load time in pkg/workflow/validator.go (depends on T117-T119)

### TUI Integration

- [X] T121 [US3] Add condition node to TUI node palette in pkg/tui/workflow_builder.go
- [X] T122 [US3] Implement condition expression editor in TUI property panel in pkg/tui/workflow_builder.go
- [X] T123 [US3] Add conditional edge labels ("true"/"false") in TUI in pkg/tui/workflow_builder.go
- [X] T124 [US3] Implement transform expression validator in TUI in pkg/tui/validation.go

### Documentation and Examples

- [X] T125 [P] [US3] Create example workflow with conditional logic in examples/conditional-workflow.yaml
- [X] T126 [P] [US3] Create example workflow with transformations in examples/data-transformation.yaml

**Checkpoint**: ✅ User Story 3 COMPLETE - workflows support branching and transformations

---

## Phase 6: User Story 4 - Monitor and Debug Workflow Execution (Priority: P4)

**Goal**: Enable real-time execution monitoring, variable inspection, detailed error context, and execution history viewing

**Independent Test**: Execute workflow in watch mode, observe real-time status updates, inspect variables at each step, view complete execution logs

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T127 [P] [US4] Integration test for execution monitoring in tests/integration/execution_monitor_test.go
- [X] T128 [P] [US4] Unit test for execution history queries in tests/unit/execution/history_test.go
- [X] T129 [P] [US4] TUI test for execution monitor view in tests/tui/execution_monitor_test.go

### Execution Monitoring Implementation

- [X] T130 [US4] Implement real-time execution event stream in pkg/execution/events.go
- [X] T131 [US4] Implement execution progress tracking in pkg/execution/progress.go
- [X] T132 [US4] Implement variable snapshot recording in pkg/execution/snapshot.go
- [X] T133 [US4] Enhance error context with stack traces and MCP logs in pkg/execution/error.go

### Execution History and Querying

- [X] T134 [US4] Implement execution history queries in pkg/storage/sqlite.go (list, filter by status, search)
- [X] T135 [US4] Implement execution detail retrieval in pkg/storage/sqlite.go
- [X] T136 [US4] Implement audit trail reconstruction in pkg/execution/audit.go

### TUI Execution Monitor View

- [X] T137 [US4] Implement execution monitor view in pkg/tui/execution_monitor.go
- [X] T138 [US4] Implement real-time execution visualization (highlighted nodes, progress) in pkg/tui/execution_monitor.go
- [X] T139 [US4] Implement variable inspector panel in pkg/tui/execution_monitor.go
- [X] T140 [US4] Implement error detail view in pkg/tui/execution_monitor.go
- [X] T141 [US4] Implement execution log viewer in pkg/tui/execution_monitor.go
- [X] T142 [US4] Implement performance metrics display (time per node, memory) in pkg/tui/execution_monitor.go

### CLI Integration

- [X] T143 [P] [US4] Implement `goflow executions` command to list execution history in pkg/cli/executions.go
- [X] T144 [P] [US4] Implement `goflow execution <id>` command to view details in pkg/cli/executions.go
- [X] T145 [P] [US4] Implement `goflow logs <id>` command to view logs in pkg/cli/logs.go
- [X] T146 [US4] Add `--watch` flag to `goflow run` for real-time monitoring in pkg/cli/run.go

**Checkpoint**: ✅ User Story 4 COMPLETE - users can monitor and debug workflow executions

---

## Phase 7: User Story 5 - Share and Reuse Workflows (Priority: P5)

**Goal**: Enable workflow export without secrets, import from others, workflow templates, and git-friendly sharing

**Independent Test**: Export workflow with credentials, share YAML file, import on different machine, configure credentials via keyring, execute successfully

### Tests for User Story 5

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T147 [P] [US5] Integration test for workflow export (credential stripping) in tests/integration/workflow_export_test.go
- [X] T148 [P] [US5] Integration test for workflow import in tests/integration/workflow_import_test.go
- [X] T149 [P] [US5] Integration test for template instantiation in tests/integration/template_test.go

### Workflow Export/Import Implementation

- [X] T150 [US5] Implement credential detection and removal in workflow export in pkg/workflow/export.go
- [X] T151 [US5] Implement server reference validation in workflow import in pkg/workflow/import.go
- [X] T152 [US5] Implement missing server detection and user prompts in pkg/workflow/import.go

### Workflow Template System

- [X] T153 [US5] Define template format with parameter placeholders in pkg/workflow/template.go
- [X] T154 [US5] Implement template instantiation with parameter substitution in pkg/workflow/template.go
- [X] T155 [US5] Create built-in template library in internal/templates/

### CLI Integration

- [X] T156 [P] [US5] Implement `goflow export` command in pkg/cli/export.go
- [X] T157 [P] [US5] Implement `goflow import` command in pkg/cli/import.go
- [X] T158 [P] [US5] Implement `goflow credential add` command in pkg/cli/credential.go
- [X] T159 [P] [US5] Implement `goflow credential list` command in pkg/cli/credential.go

### Built-in Templates

- [X] T160 [P] [US5] Create ETL pipeline template in internal/templates/etl-pipeline.yaml
- [X] T161 [P] [US5] Create API integration template in internal/templates/api-workflow.yaml
- [X] T162 [P] [US5] Create multi-server template in internal/templates/multi-server.yaml

### Documentation

- [X] T163 [US5] Document workflow sharing best practices in docs/workflow-sharing.md
- [X] T164 [US5] Document template creation guide in docs/template-guide.md

**Checkpoint**: ✅ User Story 5 COMPLETE - users can share and reuse workflows across teams

---

## Phase 8: User Story 6 - Parallel Execution and Loops (Priority: P6)

**Goal**: Enable concurrent branch execution for performance and collection iteration (for-each loops)

**Independent Test**: Create workflow with parallel node containing two branches, execute, verify both run concurrently. Create loop iterating over array, verify all items processed.

### Tests for User Story 6

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T165 [P] [US6] Integration test for parallel execution in tests/integration/parallel_test.go
- [X] T166 [P] [US6] Integration test for loop execution in tests/integration/loop_test.go
- [X] T167 [P] [US6] Performance test for 50+ parallel branches in tests/integration/parallel_performance_test.go

### Parallel Execution Implementation

- [ ] T168 [US6] Implement ParallelNode executor using errgroup in pkg/execution/node_executor.go
- [ ] T169 [US6] Implement merge strategies (wait_all, wait_any, wait_first) in pkg/execution/parallel.go
- [ ] T170 [US6] Implement parallel branch context isolation in pkg/execution/context.go
- [ ] T171 [US6] Implement parallel error handling and propagation in pkg/execution/error.go

### Loop Implementation

- [ ] T172 [US6] Implement LoopNode executor in pkg/execution/node_executor.go
- [ ] T173 [US6] Implement break condition evaluation in pkg/execution/loop.go
- [ ] T174 [US6] Implement loop result collection in pkg/execution/loop.go
- [ ] T175 [US6] Implement loop variable scoping (item variable per iteration) in pkg/execution/context.go

### TUI Integration

- [ ] T176 [P] [US6] Add parallel node to TUI node palette in pkg/tui/workflow_builder.go
- [ ] T177 [P] [US6] Add loop node to TUI node palette in pkg/tui/workflow_builder.go
- [ ] T178 [US6] Implement parallel branch editor in TUI property panel in pkg/tui/workflow_builder.go
- [ ] T179 [US6] Implement loop configuration editor in TUI property panel in pkg/tui/workflow_builder.go
- [ ] T180 [US6] Implement parallel execution visualization in execution monitor in pkg/tui/execution_monitor.go

### Documentation and Examples

- [ ] T181 [P] [US6] Create example workflow with parallel branches in examples/parallel-batch.yaml
- [ ] T182 [P] [US6] Create example workflow with loop in examples/loop-processing.yaml

**Checkpoint**: At this point, ALL user stories (1-6) should work - full feature set complete

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories, documentation, security hardening

### Additional Protocol Support

- [ ] T183 [P] Implement MCP SSE transport in pkg/mcpserver/client.go
- [ ] T184 [P] Implement MCP HTTP+JSON-RPC transport in pkg/mcpserver/client.go
- [ ] T185 Add transport selection in server configuration in pkg/workflow/server_config.go

### Retry Policies

- [ ] T186 Implement retry policy configuration in workflow schema in pkg/workflow/node.go
- [ ] T187 Implement exponential backoff retry in pkg/execution/retry.go
- [ ] T188 Implement retry on specific error types in pkg/execution/retry.go

### Performance Optimization

- [ ] T189 [P] Implement workflow caching (skip unchanged nodes) in pkg/execution/cache.go
- [ ] T190 [P] Implement connection pre-warming for frequently used servers in pkg/mcpserver/connection_pool.go
- [ ] T191 Add performance benchmarks in tests/benchmark/

### Security Hardening

- [ ] T192 [P] Add input validation for all user-supplied data in pkg/workflow/validator.go
- [ ] T193 [P] Add expression injection attack tests in tests/security/expression_test.go
- [ ] T194 [P] Add credential leak detection in exports in pkg/workflow/export.go
- [ ] T195 Run security audit with gosec in CI/CD

### TUI Server Registry View

- [ ] T196 Implement server registry view in pkg/tui/server_registry.go
- [ ] T197 Implement server add dialog in pkg/tui/server_registry.go
- [ ] T198 Implement server health status display in pkg/tui/server_registry.go
- [ ] T199 Implement server tool schema viewer in pkg/tui/server_registry.go

### Documentation

- [ ] T200 [P] Create comprehensive README.md with installation and quickstart
- [ ] T201 [P] Create docs/nodes.md documenting all node types
- [ ] T202 [P] Create docs/expressions.md documenting transformation syntax
- [ ] T203 [P] Create docs/patterns.md with advanced workflow patterns
- [ ] T204 [P] Create docs/mcp-servers.md guide for server development
- [ ] T205 [P] Create CONTRIBUTING.md with development guidelines
- [ ] T206 Verify quickstart.md tutorial works end-to-end

### Build and Release

- [ ] T207 Create GitHub Actions workflow for CI/CD in .github/workflows/ci.yml
- [ ] T208 Add cross-compilation build script in scripts/build.sh
- [ ] T209 Create release process documentation in docs/release-process.md
- [ ] T210 Verify binary size < 50MB target

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - MVP functionality
- **User Story 2 (Phase 4)**: Depends on Foundational and User Story 1 (reuses workflow model)
- **User Story 3 (Phase 5)**: Depends on Foundational and User Story 1 (extends execution engine)
- **User Story 4 (Phase 6)**: Depends on Foundational and User Story 1 (adds monitoring)
- **User Story 5 (Phase 7)**: Depends on Foundational and User Story 1 (workflow I/O)
- **User Story 6 (Phase 8)**: Depends on Foundational and User Story 1 (advanced execution)
- **Polish (Phase 9)**: Depends on all desired user stories being complete

### User Story Dependencies

All user stories depend on Phase 2 (Foundational) but are otherwise independent:

- **User Story 1 (P1)**: No dependencies on other stories - can start immediately after Foundational
- **User Story 2 (P2)**: Can start after Foundational - integrates with US1 for workflow model
- **User Story 3 (P3)**: Can start after Foundational - extends US1 execution engine
- **User Story 4 (P4)**: Can start after Foundational - adds monitoring to US1 execution
- **User Story 5 (P5)**: Can start after Foundational - adds I/O to US1 workflows
- **User Story 6 (P6)**: Can start after Foundational - adds advanced control flow to US1

**Recommended Order**: P1 → P2 → P3 → P4 → P5 → P6 (prioritized)

**Parallel Option**: After Foundational phase, User Stories 2-6 can be worked on in parallel by different developers (all depend on US1 model)

### Within Each User Story

1. Tests MUST be written and FAIL before implementation (test-first requirement)
2. Models → Services → CLI/TUI → Integration
3. Story complete before moving to next priority

### Parallel Opportunities

- **Setup Phase**: All tasks marked [P] (T003-T008) can run in parallel
- **Foundational Phase**: All tests (T009-T018) can run in parallel
  - All value objects within each aggregate (T022-T028 nodes, T048-T050 transform) can run in parallel
- **User Story 1**: Tests (T056-T061), MCP client components (T066-T068), CLI commands (T077-T081) can run in parallel
- **User Story 2**: Tests (T086-T090), TUI components (T092-T094) can run in parallel
- **Each User Story**: Tests always run in parallel, independent components run in parallel

---

## Parallel Example: Foundational Phase

Launch all Foundational tests together:

```bash
# All unit tests can run in parallel (different files):
Task T009: "Unit test for Workflow entity validation in tests/unit/workflow/workflow_test.go"
Task T010: "Unit test for Node value objects in tests/unit/workflow/node_test.go"
Task T011: "Unit test for Edge validation in tests/unit/workflow/edge_test.go"
Task T012: "Unit test for Variable validation in tests/unit/workflow/variable_test.go"
Task T013: "Unit test for Execution entity in tests/unit/execution/execution_test.go"
Task T014: "Unit test for ExecutionContext in tests/unit/execution/context_test.go"
Task T015: "Unit test for MCPServer entity in tests/unit/mcpserver/server_test.go"
Task T016: "Unit test for expression evaluation in tests/unit/transform/expression_test.go"
Task T017: "Unit test for JSONPath queries in tests/unit/transform/jsonpath_test.go"
Task T018: "Unit test for template interpolation in tests/unit/transform/template_test.go"
```

Launch all Node value objects together:

```bash
Task T022: "Implement StartNode value object in pkg/workflow/node.go"
Task T023: "Implement EndNode value object in pkg/workflow/node.go"
Task T024: "Implement MCPToolNode value object in pkg/workflow/node.go"
Task T025: "Implement TransformNode value object in pkg/workflow/node.go"
Task T026: "Implement ConditionNode value object in pkg/workflow/node.go"
Task T027: "Implement ParallelNode value object in pkg/workflow/node.go"
Task T028: "Implement LoopNode value object in pkg/workflow/node.go"
```

---

## Parallel Example: User Story 1

Launch all User Story 1 tests together:

```bash
Task T056: "Integration test for YAML workflow parsing in tests/integration/workflow_parse_test.go"
Task T057: "Integration test for workflow validation in tests/integration/workflow_validation_test.go"
Task T058: "Integration test for MCP protocol stdio transport in tests/integration/mcp_stdio_test.go"
Task T059: "Integration test for read-transform-write workflow execution in tests/integration/workflow_execution_test.go"
Task T060: "Unit test for CLI run command in tests/unit/cli/run_test.go"
Task T061: "Unit test for CLI server command in tests/unit/cli/server_test.go"
```

Launch all MCP client components together:

```bash
Task T066: "Implement MCP stdio transport in pkg/mcpserver/client.go"
Task T067: "Implement MCP tool discovery via introspection in pkg/mcpserver/client.go"
Task T068: "Implement MCP tool invocation in pkg/mcpserver/client.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

**Goal**: Deliver working CLI workflow orchestration in 4 weeks

1. Complete Phase 1: Setup (1 day)
2. Complete Phase 2: Foundational (2 weeks - most complex phase)
3. Complete Phase 3: User Story 1 (1.5 weeks)
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

**Result**: Users can create and execute workflows via CLI and YAML

### Incremental Delivery (All User Stories)

**Goal**: Deliver full feature set in 20 weeks (per research.md timeline)

1. Weeks 1-3: Setup + Foundational → Foundation ready
2. Weeks 4-5: User Story 1 → Test independently → Deploy MVP
3. Weeks 6-8: User Story 2 (TUI) → Test independently → Deploy
4. Weeks 9-11: User Story 3 (Conditionals) → Test independently → Deploy
5. Weeks 12-14: User Story 4 (Monitoring) → Test independently → Deploy
6. Weeks 15-17: User Story 5 (Sharing) → Test independently → Deploy
7. Weeks 18-19: User Story 6 (Parallel/Loops) → Test independently → Deploy
8. Week 20: Polish phase → Security audit → Final release

**Result**: Each story adds value without breaking previous stories

### Parallel Team Strategy

With 3+ developers:

1. Team completes Setup + Foundational together (Weeks 1-3)
2. Once Foundational is done:
   - Developer A: User Story 1 (CLI execution) - Weeks 4-5
   - Developer B: Can start User Story 2 (TUI) - Weeks 4-8 (depends on US1 model)
   - Developer C: Can start User Story 3 (Conditionals) - Weeks 4-8 (depends on US1 execution)
3. Stories complete and integrate independently

---

## Task Count Summary

- **Phase 1 (Setup)**: 8 tasks
- **Phase 2 (Foundational)**: 46 tasks (10 tests + 36 implementation)
- **Phase 3 (User Story 1)**: 30 tasks (6 tests + 24 implementation)
- **Phase 4 (User Story 2)**: 24 tasks (5 tests + 19 implementation)
- **Phase 5 (User Story 3)**: 17 tasks (4 tests + 13 implementation)
- **Phase 6 (User Story 4)**: 20 tasks (3 tests + 17 implementation)
- **Phase 7 (User Story 5)**: 18 tasks (3 tests + 15 implementation)
- **Phase 8 (User Story 6)**: 18 tasks (3 tests + 15 implementation)
- **Phase 9 (Polish)**: 28 tasks

**Total**: 210 tasks

**Parallel Opportunities**: 80+ tasks can run in parallel within phases

**MVP Task Count**: 84 tasks (Phases 1-3)

---

## Notes

- **[P] tasks**: Can run in parallel (different files, no dependencies)
- **[Story] label**: Maps task to specific user story for traceability
- **Test-First**: Constitutional requirement - tests MUST be written before implementation
- **Each user story should be independently completable and testable**
- **Verify tests fail before implementing** (Red-Green-Refactor)
- **Commit after each task or logical group**
- **Stop at any checkpoint to validate story independently**
- **Performance targets**: < 100ms validation, < 500ms startup, < 10ms node overhead, < 16ms TUI frame time
- **Security**: Keyring-only credentials, sandboxed expressions, no arbitrary code execution
- **Avoid**: Vague tasks, same file conflicts, cross-story dependencies that break independence
