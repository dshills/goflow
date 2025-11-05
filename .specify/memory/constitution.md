<!--
  Sync Impact Report
  ==================
  Version change: 1.0.0 → 1.0.1 (PATCH - clarifying pre-commit workflow requirements)

  Modified principles:
    - None (core principles unchanged)

  Added sections:
    - Pre-Commit Quality Gates subsection in Development Workflow

  Removed sections:
    - None

  Templates requiring updates:
    ✅ plan-template.md - Constitution Check section present, aligns with principles
    ✅ spec-template.md - User story format aligns with TDD and testability principles
    ✅ tasks-template.md - Task structure supports independent testing and DDD principles
    ✅ CLAUDE.md - No changes needed (references constitution for governance)

  Follow-up TODOs:
    - None - all fields populated
-->

# GoFlow Constitution

## Core Principles

### I. Domain-Driven Design (DDD)

GoFlow strictly follows Domain-Driven Design principles with clear aggregate boundaries. The system is organized around three core aggregates: Workflow, Execution, and MCP Server Registry. Each aggregate maintains its own invariants and must not be violated by any implementation. Cross-aggregate communication happens through references (IDs), never direct object access.

**Rationale**: DDD provides clear boundaries for complexity management, enables independent testing of aggregates, and ensures the codebase accurately reflects the problem domain. This is essential for a system orchestrating complex workflows where state management and consistency are critical.

### II. Test-First Development (NON-NEGOTIABLE)

All production code MUST be preceded by failing tests. Tests are written → User approved → Tests fail → Then implement. The Red-Green-Refactor cycle is strictly enforced. This applies to domain logic, execution engine, protocol integration, and TUI components. No feature branches merge without comprehensive test coverage.

**Rationale**: For a workflow orchestration system where execution correctness is paramount, test-first development ensures reliability from the ground up. Early test design catches design flaws before implementation and provides living documentation of expected behavior.

### III. Performance Consciousness

All features must meet defined performance targets: workflow validation < 100ms for < 100 nodes, execution startup < 500ms, node execution overhead < 10ms per node, parallel execution supporting 50+ branches, memory < 100MB base + 10MB per server, TUI responsiveness < 16ms frame time (60 FPS). Performance tests are required for core execution paths.

**Rationale**: GoFlow must not become a bottleneck in developer workflows. Users expect fast workflow editing, instant validation feedback, and minimal execution overhead. Poor performance would undermine the value proposition of workflow automation.

### IV. Security by Design

Security considerations are mandatory in every design decision: server credentials stored only in system keyring (never in workflow files), expression evaluation must be sandboxed (no arbitrary code execution), workflows must be shareable without exposing secrets, all user input must be validated, execution logs filtered for sensitive data, MCP server processes sandboxed with user permissions only.

**Rationale**: Workflow systems have access to sensitive resources (filesystems, databases, APIs). A security vulnerability could compromise entire development environments or production systems. Security cannot be retrofitted; it must be foundational.

### V. Observable and Debuggable

Every workflow execution MUST produce a complete audit trail with full execution trace (inputs/outputs for each node), variable state changes, MCP server communication logs, error context with stack traces, execution timing data, and resource usage metrics. The TUI must visualize execution state in real-time. Logs must be structured and parseable.

**Rationale**: When workflows fail (and they will), developers need comprehensive debugging information. Opaque execution makes troubleshooting impossible and erodes trust in the system. Observability enables rapid diagnosis and continuous improvement.

## Technical Constraints

**Language & Runtime**: Go 1.21+ exclusively. No embedded scripting languages for workflow expressions (use sandboxed expression evaluator). No CGO dependencies to maintain portability.

**Dependencies**: Minimize external dependencies. Core libraries only: goterm (TUI), yaml.v3 (parsing), gjson (JSON queries), expr-lang/expr (expressions), golang.org/x/sync (concurrency). All dependencies must be actively maintained and security-audited.

**Storage**: Local filesystem for workflow definitions (YAML), SQLite for execution history, system keyring for credentials. No cloud dependencies. All data must be portable.

**MCP Protocol**: Full compliance with MCP specification required. Support stdio, SSE, and HTTP transports. Maintain backward compatibility with MCP protocol versions through explicit versioning.

**Platform Support**: macOS, Linux, and Windows. TUI must work in all standard terminals. No GUI dependencies.

**Build & Distribution**: Single static binary distribution. No runtime dependencies beyond OS standard library. Binary size target < 50MB.

## Development Workflow

**Branch Strategy**: Feature branches from main (`###-feature-name` format). No direct commits to main. Feature branches deleted after merge.

**Specification Workflow**: All features start with `/speckit.specify` to create spec.md, followed by `/speckit.plan` for implementation planning, then `/speckit.tasks` for task breakdown. Implementation proceeds via `/speckit.implement`.

**Code Review**: All PRs require review focusing on: domain model integrity, test coverage, performance implications, security considerations, error handling completeness, documentation clarity. Constitution compliance is mandatory.

**Testing Gates**: All tests must pass before merge. Integration tests required for MCP protocol interactions. TUI interaction tests required for all views. Performance benchmarks must meet targets.

**Pre-Commit Quality Gates**: Before creating any commit, the following MUST be completed in order:
1. All linting errors resolved (run `golangci-lint run` and fix all issues)
2. All tests passing (run `go test ./...` with zero failures)
3. Automated code review via mcp-pr using OpenAI provider (run review on staged changes)
4. Address all critical issues identified in code review before committing

This ensures every commit in history represents reviewed, tested, quality code. Use git hooks or manual verification, but never commit code that hasn't passed all four gates.

**Documentation**: Public APIs require godoc comments. Complex domain logic requires inline explanatory comments. All user-facing features require updates to user guide and examples.

**Commit Hygiene**: Atomic commits with clear messages. Format: `type(scope): description` where type ∈ {feat, fix, refactor, test, docs, perf, security}. Reference issue/task numbers.

## Governance

This constitution supersedes all other development practices and conventions. All pull requests and code reviews MUST verify compliance with these principles. Any violation must be explicitly justified in the PR description with compelling rationale.

**Amendment Process**: Constitution changes require:
1. Proposal documented with rationale and impact analysis
2. Review by project maintainer (Darrell Hills)
3. Update to this document with version increment
4. Propagation to dependent templates (plan, spec, tasks)
5. Git commit with clear changelog

**Versioning Policy**:
- MAJOR: Backward incompatible principle removals or redefinitions
- MINOR: New principles added or materially expanded guidance
- PATCH: Clarifications, wording refinements, typo fixes

**Complexity Justification**: Any deviation from principles (e.g., adding non-sandboxed code execution, violating aggregate boundaries, skipping tests, missing performance targets) must be documented in the implementation plan's Complexity Tracking table with: (1) specific violation, (2) why needed, (3) why simpler alternatives were rejected.

**Compliance Reviews**: Conducted at:
- End of Phase 1 (Foundation) - verify domain model integrity
- End of Phase 2 (Execution Engine) - verify test coverage and performance
- End of Phase 3 (TUI) - verify usability and observability
- Before public release - comprehensive security and performance audit

**Runtime Guidance**: Development guidance and best practices are maintained in `/CLAUDE.md` at the repository root. This file provides context-specific instructions for AI-assisted development and should be referenced for implementation details not covered by constitutional principles.

**Version**: 1.0.1 | **Ratified**: 2025-11-05 | **Last Amended**: 2025-11-05
