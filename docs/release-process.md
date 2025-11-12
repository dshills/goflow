# GoFlow Release Process

**Version**: 1.0
**Last Updated**: 2025-11-11
**Maintainers**: GoFlow Core Team

This document describes the complete release process for GoFlow, including versioning, testing, building, and distribution.

## Table of Contents

- [Versioning Strategy](#versioning-strategy)
- [Release Types](#release-types)
- [Pre-Release Checklist](#pre-release-checklist)
- [Release Steps](#release-steps)
- [Post-Release Tasks](#post-release-tasks)
- [Hotfix Process](#hotfix-process)
- [Rollback Procedure](#rollback-procedure)

## Versioning Strategy

GoFlow follows [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILDMETADATA]
```

### Version Components

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible new features
- **PATCH**: Backwards-compatible bug fixes
- **PRERELEASE**: Optional pre-release identifier (alpha, beta, rc)
- **BUILDMETADATA**: Optional build metadata

### Version Examples

```
1.0.0           # Major release
1.1.0           # Minor release (new features)
1.1.1           # Patch release (bug fixes)
1.2.0-alpha.1   # Alpha pre-release
1.2.0-beta.2    # Beta pre-release
1.2.0-rc.1      # Release candidate
```

### Version Increment Rules

| Change Type | Example | Version Change |
|-------------|---------|----------------|
| Breaking API change | Remove/rename command | MAJOR (1.0.0 → 2.0.0) |
| Workflow format change | New required field | MAJOR (1.0.0 → 2.0.0) |
| New feature | Add new node type | MINOR (1.0.0 → 1.1.0) |
| New CLI command | Add `goflow debug` | MINOR (1.0.0 → 1.1.0) |
| Bug fix | Fix execution error | PATCH (1.0.0 → 1.0.1) |
| Security fix | Fix credential leak | PATCH (1.0.0 → 1.0.1) |
| Documentation | Update README | No version change |
| Refactoring | Internal code cleanup | No version change |

## Release Types

### 1. Major Release (X.0.0)

**When**: Breaking changes, major feature additions, significant architectural changes

**Timeline**: 3-6 months

**Process**:
- Extended beta period (4+ weeks)
- Migration guide required
- Deprecation warnings in previous minor version
- User communication campaign

### 2. Minor Release (X.Y.0)

**When**: New features, backwards-compatible changes

**Timeline**: 4-8 weeks

**Process**:
- Beta period (2-3 weeks)
- Feature documentation required
- Optional user announcement

### 3. Patch Release (X.Y.Z)

**When**: Bug fixes, security patches

**Timeline**: As needed (typically 1-2 weeks)

**Process**:
- Minimal testing (affected areas only)
- Fast-track for security issues
- Optional announcement for security fixes

### 4. Pre-Release Versions

**Alpha**: Feature incomplete, unstable, internal testing only

```
1.2.0-alpha.1, 1.2.0-alpha.2, ...
```

**Beta**: Feature complete, stable enough for user testing

```
1.2.0-beta.1, 1.2.0-beta.2, ...
```

**Release Candidate**: Production-ready, final testing

```
1.2.0-rc.1, 1.2.0-rc.2, ...
```

## Pre-Release Checklist

Complete this checklist before starting the release process:

### Code Quality

- [ ] All tests passing on `main` branch
- [ ] No critical or high-severity security issues (gosec, govulncheck)
- [ ] Code coverage meets target (>85% for core packages)
- [ ] All linting checks pass (golangci-lint)
- [ ] No known critical bugs in issue tracker

### Documentation

- [ ] CHANGELOG.md updated with all changes since last release
- [ ] README.md reflects current version features
- [ ] API documentation up to date
- [ ] Migration guide prepared (for major/minor releases)
- [ ] Example workflows tested and updated

### Testing

- [ ] Unit tests passing (all platforms)
- [ ] Integration tests passing
- [ ] Manual testing completed (see Testing Checklist below)
- [ ] Performance benchmarks run (no regressions)
- [ ] Security audit completed (for major releases)

### Build Verification

- [ ] Cross-compilation successful for all platforms
- [ ] Binary sizes within limits (<50MB)
- [ ] Binaries tested on target platforms
- [ ] No unexpected dependencies added

### Release Materials

- [ ] Release notes drafted
- [ ] GitHub release description prepared
- [ ] Social media announcements drafted (optional)
- [ ] Blog post prepared (for major/minor releases)

## Release Steps

### Step 1: Prepare Release Branch

```bash
# Create release branch from main
git checkout main
git pull origin main
git checkout -b release/v1.2.0

# Update version in relevant files
# - cmd/goflow/version.go
# - README.md
# - Documentation files

git add .
git commit -m "chore: prepare release v1.2.0"
git push origin release/v1.2.0
```

### Step 2: Update CHANGELOG

Edit `CHANGELOG.md` to document all changes:

```markdown
## [1.2.0] - 2025-11-11

### Added
- New parallel execution node type (#123)
- Support for JSONPath expressions in transformations (#145)
- CLI command `goflow debug` for workflow debugging (#167)

### Changed
- Improved error messages for validation failures (#134)
- Updated TUI navigation shortcuts (#156)

### Fixed
- Fixed race condition in execution engine (#178)
- Resolved credential keyring access on Windows (#189)

### Security
- Fixed potential path traversal in file operations (#201)
- Updated vulnerable dependency golang.org/x/crypto (#203)

### Deprecated
- `goflow run --legacy-mode` will be removed in v2.0.0

### Removed
- Dropped support for Go 1.20 (now requires 1.21+)
```

Commit the changelog:

```bash
git add CHANGELOG.md
git commit -m "docs: update changelog for v1.2.0"
git push origin release/v1.2.0
```

### Step 3: Run Full Test Suite

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run security scans
make security

# Run benchmarks
make bench

# Verify cross-compilation
./scripts/build.sh --all
```

### Step 4: Create Release Tag

```bash
# Ensure you're on the release branch
git checkout release/v1.2.0

# Create annotated tag
git tag -a v1.2.0 -m "Release version 1.2.0"

# Verify tag
git tag -v v1.2.0

# Push tag to remote
git push origin v1.2.0
```

### Step 5: Build Release Binaries

```bash
# Clean build with version tagging
VERSION=v1.2.0 ./scripts/build.sh --clean --all

# Verify all binaries built successfully
ls -lh bin/releases/

# Test binaries on target platforms
# - macOS (Intel): bin/releases/goflow-darwin-amd64
# - macOS (Apple Silicon): bin/releases/goflow-darwin-arm64
# - Linux (x86_64): bin/releases/goflow-linux-amd64
# - Linux (ARM64): bin/releases/goflow-linux-arm64
# - Windows: bin/releases/goflow-windows-amd64.exe
```

### Step 6: Create GitHub Release

Using GitHub CLI:

```bash
# Create draft release
gh release create v1.2.0 \
  --draft \
  --title "GoFlow v1.2.0" \
  --notes-file release-notes.md \
  bin/releases/goflow-* \
  bin/releases/checksums.txt

# Review draft release on GitHub
# Once verified, publish the release
gh release edit v1.2.0 --draft=false
```

Using GitHub Web Interface:

1. Go to https://github.com/dshills/goflow/releases/new
2. Select tag: `v1.2.0`
3. Release title: `GoFlow v1.2.0`
4. Description: Paste release notes from `release-notes.md`
5. Attach binaries:
   - `goflow-darwin-amd64`
   - `goflow-darwin-arm64`
   - `goflow-linux-amd64`
   - `goflow-linux-arm64`
   - `goflow-windows-amd64.exe`
   - `checksums.txt`
6. Click "Publish release"

### Step 7: Merge Release Branch

```bash
# Merge release branch back to main
git checkout main
git merge --no-ff release/v1.2.0 -m "Merge release v1.2.0"
git push origin main

# Merge to develop branch
git checkout develop
git merge --no-ff release/v1.2.0 -m "Merge release v1.2.0"
git push origin develop

# Delete release branch
git branch -d release/v1.2.0
git push origin --delete release/v1.2.0
```

## Post-Release Tasks

### Immediate Tasks (Within 1 Day)

- [ ] Verify release appears on GitHub Releases page
- [ ] Test download links for all platforms
- [ ] Verify checksums match binaries
- [ ] Update website/documentation with new version
- [ ] Announce release on social media/blog (optional)
- [ ] Close GitHub milestone for this release
- [ ] Update issue tracker labels (if applicable)

### Follow-Up Tasks (Within 1 Week)

- [ ] Monitor issue tracker for bug reports
- [ ] Check analytics for download metrics
- [ ] Gather user feedback
- [ ] Plan next release cycle
- [ ] Update project roadmap

### Homebrew Formula Update (macOS)

If you have a Homebrew formula:

```bash
# Update formula with new version and checksums
# Formula location: homebrew-goflow/Formula/goflow.rb

brew bump-formula-pr --url=https://github.com/dshills/goflow/archive/v1.2.0.tar.gz goflow
```

### Docker Image Release (Optional)

If you publish Docker images:

```bash
# Build and tag Docker image
docker build -t dshills/goflow:1.2.0 -t dshills/goflow:latest .

# Push to Docker Hub
docker push dshills/goflow:1.2.0
docker push dshills/goflow:latest
```

## Hotfix Process

For critical bugs that need immediate release:

### Step 1: Create Hotfix Branch

```bash
# Branch from the latest release tag
git checkout -b hotfix/v1.2.1 v1.2.0

# Fix the critical bug
# ... make changes ...

git add .
git commit -m "fix: critical bug in execution engine"
```

### Step 2: Test Hotfix

```bash
# Run focused tests on affected areas
go test ./pkg/execution/... -v

# Build and test binary
go build -o goflow ./cmd/goflow
./goflow run test-workflow
```

### Step 3: Release Hotfix

```bash
# Update version and changelog
# Create tag
git tag -a v1.2.1 -m "Hotfix release 1.2.1"
git push origin v1.2.1

# Build and release
VERSION=v1.2.1 ./scripts/build.sh --all
gh release create v1.2.1 --title "GoFlow v1.2.1 (Hotfix)" --notes "..." bin/releases/*

# Merge back to main and develop
git checkout main
git merge --no-ff hotfix/v1.2.1
git push origin main

git checkout develop
git merge --no-ff hotfix/v1.2.1
git push origin develop

git branch -d hotfix/v1.2.1
```

## Rollback Procedure

If a critical issue is discovered after release:

### Option 1: Immediate Hotfix

- Release new patch version with fix (preferred)
- Follow hotfix process above

### Option 2: Release Deprecation

- Mark release as deprecated on GitHub
- Update release notes with warning
- Point users to previous stable version
- Issue hotfix as soon as possible

### Option 3: Release Deletion (Extreme Cases Only)

```bash
# Delete GitHub release
gh release delete v1.2.0 --yes

# Delete git tag locally and remotely
git tag -d v1.2.0
git push origin :refs/tags/v1.2.0

# Communicate to users immediately
```

**WARNING**: Only delete a release if:
- No users have downloaded it yet
- It contains a critical security vulnerability
- Data loss or corruption is possible

## Testing Checklist

### Manual Testing for Each Release

#### Core Functionality
- [ ] Initialize new workflow: `goflow init test-workflow`
- [ ] Validate workflow: `goflow validate test-workflow`
- [ ] Execute workflow: `goflow run test-workflow`
- [ ] View execution history: `goflow executions`
- [ ] View execution details: `goflow execution <id>`
- [ ] View logs: `goflow logs <id>`
- [ ] Export workflow: `goflow export test-workflow`
- [ ] Import workflow: `goflow import exported.yaml`

#### Server Management
- [ ] Add MCP server: `goflow server add ...`
- [ ] List servers: `goflow server list`
- [ ] Test server: `goflow server test <server-id>`
- [ ] Remove server: `goflow server remove <server-id>`

#### TUI Functionality
- [ ] Launch TUI: `goflow edit test-workflow`
- [ ] Navigate workflow explorer
- [ ] Create new node in builder
- [ ] Connect nodes with edges
- [ ] Validate workflow in TUI
- [ ] Execute workflow from TUI
- [ ] View execution monitor

#### Platform-Specific Testing
- [ ] macOS: Test binary on Intel and Apple Silicon
- [ ] Linux: Test binary on Ubuntu/Debian and CentOS/RHEL
- [ ] Windows: Test binary on Windows 10/11

#### Edge Cases
- [ ] Large workflows (100+ nodes)
- [ ] Parallel execution (10+ concurrent branches)
- [ ] Long-running workflows (10+ minutes)
- [ ] Error handling (invalid workflows, connection failures)
- [ ] Credential management (add/list/remove)

## Version Bump Script

Use this helper script to automate version bumping:

```bash
#!/bin/bash
# scripts/bump-version.sh

OLD_VERSION=$1
NEW_VERSION=$2

if [[ -z "$OLD_VERSION" ]] || [[ -z "$NEW_VERSION" ]]; then
    echo "Usage: $0 <old-version> <new-version>"
    echo "Example: $0 1.1.0 1.2.0"
    exit 1
fi

# Update version in code
sed -i '' "s/Version = \"$OLD_VERSION\"/Version = \"$NEW_VERSION\"/" cmd/goflow/version.go

# Update README
sed -i '' "s/goflow v$OLD_VERSION/goflow v$NEW_VERSION/" README.md

# Update documentation
find docs -type f -name "*.md" -exec sed -i '' "s/v$OLD_VERSION/v$NEW_VERSION/" {} +

echo "✓ Version bumped from $OLD_VERSION to $NEW_VERSION"
echo "✓ Review changes and update CHANGELOG.md manually"
```

## Release Notes Template

Use this template for release notes:

```markdown
## GoFlow v1.2.0

**Release Date**: 2025-11-11
**Download**: [GitHub Releases](https://github.com/dshills/goflow/releases/tag/v1.2.0)

### Highlights

- 🚀 New parallel execution node type for concurrent workflow branches
- 🔍 Enhanced JSONPath support in data transformations
- 🐛 Fixed critical race condition in execution engine
- 🔒 Security improvements in credential handling

### What's New

#### Parallel Execution Node

Execute multiple workflow branches concurrently for improved performance:

```yaml
nodes:
  - id: "parallel_task"
    type: "parallel"
    branches:
      - ["task_a", "task_b"]
      - ["task_c", "task_d"]
```

See [documentation](docs/nodes.md#parallel-node) for details.

#### Enhanced JSONPath Expressions

More powerful data transformations with improved JSONPath support:

```yaml
expression: "$.users[?(@.age > 18)].email"
```

### Breaking Changes

**None** - This release is fully backwards-compatible with v1.1.x

### Upgrade Instructions

Download the appropriate binary for your platform:

**macOS**:
```bash
curl -L https://github.com/dshills/goflow/releases/download/v1.2.0/goflow-darwin-arm64 -o goflow
chmod +x goflow
sudo mv goflow /usr/local/bin/
```

**Linux**:
```bash
curl -L https://github.com/dshills/goflow/releases/download/v1.2.0/goflow-linux-amd64 -o goflow
chmod +x goflow
sudo mv goflow /usr/local/bin/
```

**Windows**: Download `goflow-windows-amd64.exe` and add to PATH

### Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete list of changes.

### Contributors

Thank you to all contributors who made this release possible!

- @contributor1
- @contributor2
- @contributor3

### Checksums

Verify your download with SHA256 checksums:

```
<checksums from checksums.txt>
```

---

**Questions?** Open an issue on [GitHub](https://github.com/dshills/goflow/issues)
**Documentation**: https://github.com/dshills/goflow/tree/main/docs
```

## Support and Resources

- **Issue Tracker**: https://github.com/dshills/goflow/issues
- **Discussions**: https://github.com/dshills/goflow/discussions
- **Documentation**: https://github.com/dshills/goflow/tree/main/docs
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md)

## Questions and Feedback

For questions about the release process, contact the GoFlow maintainers or open a discussion on GitHub.
