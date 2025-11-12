# Contributing to GoFlow

Thank you for your interest in contributing to GoFlow! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, constructive, and collaborative. We're building tools to help developers, and we welcome diverse perspectives and experiences.

## How to Contribute

### Reporting Bugs

1. **Search existing issues** to avoid duplicates
2. **Use the bug report template** if available
3. **Include reproduction steps**:
   - GoFlow version (`goflow version`)
   - Operating system and version
   - Go version (`go version`)
   - Minimal workflow YAML that reproduces the issue
   - Expected vs actual behavior
   - Error messages and logs

### Suggesting Features

1. **Check existing discussions** and feature requests
2. **Describe the use case**: What problem does this solve?
3. **Provide examples**: How would it work in practice?
4. **Consider alternatives**: Are there other ways to achieve this?

### Submitting Pull Requests

#### Before You Start

1. **Open an issue first** for significant changes
2. **Check the roadmap** in README.md to avoid duplicate effort
3. **Read CLAUDE.md** for architecture and design principles

#### Development Setup

```bash
# Clone repository
git clone https://github.com/dshills/goflow.git
cd goflow

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o goflow ./cmd/goflow

# Run linter (if available)
golangci-lint run
```

#### Pull Request Process

1. **Create a feature branch**: `git checkout -b feature/your-feature-name`

2. **Follow code style**:
   - Run `go fmt ./...`
   - Follow Go best practices and idioms
   - Add comments for exported functions and types
   - Keep functions focused and testable

3. **Write tests**:
   - Unit tests for business logic (target: >85% coverage)
   - Integration tests for MCP protocol interactions
   - Examples in test files for complex features

4. **Update documentation**:
   - Add/update doc comments
   - Update README.md if adding user-facing features
   - Update CLAUDE.md for architecture changes
   - Add examples to `examples/` directory

5. **Commit messages**:
   ```
   type(scope): brief description

   Longer explanation if needed.

   Fixes #issue-number
   ```

   Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

6. **Run checks before pushing**:
   ```bash
   go fmt ./...
   go vet ./...
   go test ./...
   go test -race ./...
   ```

7. **Create pull request**:
   - Reference related issues
   - Describe changes and motivation
   - Include test results
   - Add screenshots/examples for UI changes

#### Review Process

- Maintainers will review within 1-2 weeks
- Address feedback with additional commits
- Once approved, maintainers will merge

## Development Guidelines

### Architecture Principles

GoFlow follows Domain-Driven Design (DDD):

1. **Aggregates**: Workflow, Execution, MCP Server Registry
2. **Entities**: Maintain identity across lifecycle
3. **Value Objects**: Immutable, compared by value
4. **Repositories**: Persistence abstraction
5. **Services**: Cross-aggregate operations

See [CLAUDE.md](CLAUDE.md) for detailed architecture.

### Security Guidelines

1. **Never commit credentials**: Use system keyring
2. **Validate all inputs**: Especially workflow YAML and user expressions
3. **Sanitize file paths**: Use `pkg/validation` path validator
4. **Sandbox expressions**: Use `expr-lang` with restricted environment
5. **Log security events**: Failed auth, rejected paths, suspicious patterns

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

### Testing Guidelines

1. **Unit Tests** (`*_test.go` alongside source):
   - Test business logic in isolation
   - Use table-driven tests
   - Mock external dependencies

2. **Integration Tests** (`tests/integration/`):
   - Test MCP protocol interactions
   - Use test MCP server (`internal/testutil/testserver`)
   - Test complete workflows end-to-end

3. **TUI Tests** (`tests/tui/`):
   - Test user interactions
   - Use goterm's testing facilities
   - Verify screen output

4. **Test Naming**:
   ```go
   func TestFeatureName_Scenario(t *testing.T)
   func TestFeatureName_Scenario_ExpectedBehavior(t *testing.T)
   ```

### Performance Guidelines

Target performance metrics (see CLAUDE.md):
- Workflow validation: < 100ms for < 100 nodes
- Execution startup: < 500ms
- Node execution overhead: < 10ms per node
- Memory: < 100MB base + 10MB per MCP server

Use benchmarks for performance-critical code:
```go
func BenchmarkFeatureName(b *testing.B) {
    // benchmark code
}
```

## Project Structure

```
goflow/
├── cmd/goflow/          # CLI entry point
├── pkg/                 # Public packages
│   ├── workflow/        # Workflow aggregate
│   ├── execution/       # Execution aggregate
│   ├── mcpserver/       # MCP server registry
│   ├── mcp/             # MCP protocol client
│   ├── transform/       # Data transformation
│   ├── storage/         # Persistence
│   ├── cli/             # CLI commands
│   └── tui/             # Terminal UI
├── internal/            # Private packages
├── tests/               # Integration and TUI tests
├── examples/            # Example workflows
├── specs/               # Feature specifications
└── docs/                # Additional documentation
```

## Getting Help

- **Documentation**: [CLAUDE.md](CLAUDE.md)
- **Quickstart**: [quickstart.md](specs/001-goflow-spec-review/quickstart.md)
- **Discussions**: https://github.com/dshills/goflow/discussions
- **Issues**: https://github.com/dshills/goflow/issues

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
