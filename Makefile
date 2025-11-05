# GoFlow Makefile
# Workflow orchestration system for MCP servers

.PHONY: all build install test clean fmt lint help examples run-tests check deps

# Binary names
BINARY_NAME=goflow
INSTALL_PATH=$(GOPATH)/bin

# Directories
BIN_DIR=./bin
CMD_DIR=./cmd
PKG_DIR=./pkg
TEST_DIR=./tests

# Build variables
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Default target
all: clean deps fmt build test

## help: Display this help message
help:
	@echo "GoFlow - Visual MCP Workflow Orchestrator"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'
	@echo ""

## build: Build the main goflow binary
build:
	@echo "Building goflow..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)/goflow/main.go
	@echo "✓ Built $(BIN_DIR)/$(BINARY_NAME)"

## build-all: Build all binaries including examples
build-all: build examples
	@echo "✓ All binaries built"

## examples: Build example programs to ./bin directory
examples:
	@echo "Building examples..."
	@mkdir -p $(BIN_DIR)
	@for dir in $(CMD_DIR)/*/; do \
		if [ "$$(basename $$dir)" != "goflow" ]; then \
			binary=$$(basename $$dir); \
			echo "  Building $$binary..."; \
			$(GOBUILD) -o $(BIN_DIR)/$$binary $$dir/*.go 2>/dev/null || true; \
		fi \
	done
	@echo "✓ Examples built in $(BIN_DIR)/"

## install: Install goflow binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## test-short: Run tests without integration tests
test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short -race ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	$(GOTEST) -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "✓ Coverage report: coverage/coverage.html"

## test-integration: Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race $(TEST_DIR)/integration/...

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race $(TEST_DIR)/unit/...

## run-tests: Alias for test (matches user request)
run-tests: test

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## check: Run tests and linter
check: test lint
	@echo "✓ All checks passed"

## fmt: Format code with gofmt
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -w -s $(PKG_DIR) $(CMD_DIR) $(TEST_DIR)
	@echo "✓ Code formatted"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run; \
		echo "✓ Linting complete"; \
	else \
		echo "⚠ golangci-lint not installed. Run: brew install golangci-lint"; \
	fi

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@echo "✓ Dependencies ready"

## tidy: Tidy and verify dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "✓ Dependencies tidied"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf coverage
	@rm -f *.test
	@rm -f *.out
	$(GOCLEAN)
	@echo "✓ Cleaned"

## dev: Quick build and test for development
dev: fmt build test-short
	@echo "✓ Development build complete"

## release: Build release binaries for multiple platforms
release:
	@echo "Building release binaries..."
	@mkdir -p $(BIN_DIR)/release
	@# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/goflow/main.go
	@# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/goflow/main.go
	@# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/goflow/main.go
	@# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/goflow/main.go
	@# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/goflow/main.go
	@echo "✓ Release binaries built in $(BIN_DIR)/release/"

## validate: Validate example workflows
validate: build
	@echo "Validating example workflows..."
	@for file in examples/*.yaml; do \
		echo "  Validating $$file..."; \
		$(BIN_DIR)/$(BINARY_NAME) validate $$file || exit 1; \
	done
	@echo "✓ All workflows valid"

## server-list: List registered MCP servers
server-list: build
	@$(BIN_DIR)/$(BINARY_NAME) server list

## version: Show version information
version:
	@echo "GoFlow version: $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"

## info: Show project information
info:
	@echo "Project: GoFlow - Visual MCP Workflow Orchestrator"
	@echo "Version: $(VERSION)"
	@echo "Go version: $$(go version)"
	@echo "Build directory: $(BIN_DIR)"
	@echo "Install path: $(INSTALL_PATH)"
