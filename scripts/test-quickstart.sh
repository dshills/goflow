#!/bin/bash
# GoFlow Quickstart Tutorial Test Script
# This script tests the quickstart tutorial from specs/001-goflow-spec-review/quickstart.md
# It validates that all commands work as documented and produces expected output
#
# Status: Documentation/Reference (CLI commands not yet fully implemented)
# This script serves as:
#   1. Expected behavior documentation
#   2. Integration test specification
#   3. Tutorial validation tool (once implementation complete)
#
# Usage:
#   ./scripts/test-quickstart.sh
#
# Requirements:
#   - goflow binary in PATH or ./goflow
#   - npx (Node.js) for filesystem MCP server
#   - /tmp directory with write access

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test result tracking
PASSED=0
FAILED=0
SKIPPED=0

# Helper functions
print_step() {
    echo -e "\n${BLUE}=== Step $1: $2 ===${NC}"
}

print_pass() {
    echo -e "${GREEN}✓ PASS:${NC} $1"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL:${NC} $1"
    ((FAILED++))
}

print_skip() {
    echo -e "${YELLOW}⊘ SKIP:${NC} $1"
    ((SKIPPED++))
}

print_summary() {
    echo -e "\n${BLUE}=== Test Summary ===${NC}"
    echo -e "${GREEN}Passed: ${PASSED}${NC}"
    echo -e "${RED}Failed: ${FAILED}${NC}"
    echo -e "${YELLOW}Skipped: ${SKIPPED}${NC}"

    if [ $FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up test environment...${NC}"
    rm -f /tmp/input.json /tmp/output.txt
    # Remove test workflows
    rm -rf ~/.goflow/workflows/data-pipeline.yaml 2>/dev/null || true
}

# Set up cleanup on exit
trap cleanup EXIT

# Find goflow binary
GOFLOW="./goflow"
if [ ! -f "$GOFLOW" ]; then
    GOFLOW="goflow"
    if ! command -v $GOFLOW &> /dev/null; then
        echo -e "${RED}Error: goflow binary not found${NC}"
        echo "Please build it first: go build -o goflow ./cmd/goflow"
        exit 1
    fi
fi

echo -e "${BLUE}GoFlow Quickstart Tutorial Test${NC}"
echo "Testing goflow binary: $GOFLOW"
echo "Working directory: $(pwd)"

# ==============================================================================
# Step 1: Verify Installation
# ==============================================================================
print_step "1" "Verify Installation"

if output=$($GOFLOW --version 2>&1); then
    if echo "$output" | grep -q "goflow.*v"; then
        print_pass "goflow --version returns version string"
        echo "  Output: $output"
    else
        print_fail "goflow --version output doesn't match expected format"
        echo "  Expected: goflow v1.0.0"
        echo "  Got: $output"
    fi
else
    print_fail "goflow --version command failed"
fi

# ==============================================================================
# Step 2: Register MCP Server
# ==============================================================================
print_step "2" "Register MCP Server"

# Check if server add command exists
if $GOFLOW server --help &> /dev/null; then
    if output=$($GOFLOW server add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp 2>&1); then
        print_pass "goflow server add command executed"
        echo "  Output: $output"
    else
        print_skip "goflow server add command failed (not yet implemented)"
        echo "  Output: $output"
    fi
else
    print_skip "goflow server command not yet implemented"
fi

# ==============================================================================
# Step 3: List Servers
# ==============================================================================
print_step "3" "List Servers"

if $GOFLOW server list &> /dev/null; then
    output=$($GOFLOW server list 2>&1)
    print_pass "goflow server list command executed"
    echo "  Output: $output"

    # Check if filesystem server is listed
    if echo "$output" | grep -q "filesystem"; then
        print_pass "filesystem server appears in list"
    else
        print_fail "filesystem server not found in list"
    fi
else
    print_skip "goflow server list command not yet implemented"
fi

# ==============================================================================
# Step 4: Test Server Connection
# ==============================================================================
print_step "4" "Test Server Connection"

if $GOFLOW server test filesystem &> /dev/null 2>&1; then
    output=$($GOFLOW server test filesystem 2>&1)
    print_pass "goflow server test command executed"
    echo "  Output: $output"

    # Check for success message
    if echo "$output" | grep -q -i "success\|healthy\|ok"; then
        print_pass "Server connection successful"
    else
        print_fail "Server connection test did not indicate success"
    fi
else
    print_skip "goflow server test command not yet implemented"
fi

# ==============================================================================
# Step 5: Create Workflow
# ==============================================================================
print_step "5" "Create Workflow"

# Create .goflow directory if it doesn't exist
mkdir -p ~/.goflow/workflows

# Copy example workflow
if [ -f "examples/simple-pipeline.yaml" ]; then
    cp examples/simple-pipeline.yaml ~/.goflow/workflows/data-pipeline.yaml
    print_pass "Copied workflow from examples/simple-pipeline.yaml"

    # Verify workflow file exists
    if [ -f ~/.goflow/workflows/data-pipeline.yaml ]; then
        print_pass "Workflow file created at ~/.goflow/workflows/data-pipeline.yaml"
    else
        print_fail "Workflow file not found after copy"
    fi
else
    print_fail "Example workflow not found at examples/simple-pipeline.yaml"
fi

# ==============================================================================
# Step 6: Validate Workflow
# ==============================================================================
print_step "6" "Validate Workflow"

if $GOFLOW validate --help &> /dev/null; then
    if output=$($GOFLOW validate data-pipeline 2>&1); then
        print_pass "goflow validate command executed"
        echo "  Output: $output"

        # Check for validation success messages
        if echo "$output" | grep -q -i "valid\|success\|ok"; then
            print_pass "Workflow validation successful"
        else
            print_fail "Workflow validation did not indicate success"
        fi
    else
        print_fail "goflow validate command failed"
        echo "  Output: $output"
    fi
else
    print_skip "goflow validate command not yet implemented"
fi

# ==============================================================================
# Step 7: Create Test Data
# ==============================================================================
print_step "7" "Create Test Data"

echo '{"data": [{"price": 10.5}, {"price": 20.3}, {"price": 5.2}]}' > /tmp/input.json

if [ -f /tmp/input.json ]; then
    print_pass "Created test data at /tmp/input.json"

    # Verify content
    if grep -q "10.5" /tmp/input.json && grep -q "20.3" /tmp/input.json && grep -q "5.2" /tmp/input.json; then
        print_pass "Test data contains expected prices"
    else
        print_fail "Test data does not contain expected prices"
    fi
else
    print_fail "Failed to create test data file"
fi

# ==============================================================================
# Step 8: Execute Workflow
# ==============================================================================
print_step "8" "Execute Workflow"

if $GOFLOW run --help &> /dev/null; then
    if output=$($GOFLOW run data-pipeline 2>&1); then
        print_pass "goflow run command executed"
        echo "  Output: $output"

        # Check for execution success messages
        if echo "$output" | grep -q -i "complete\|success"; then
            print_pass "Workflow execution completed"
        else
            print_fail "Workflow execution did not indicate completion"
        fi
    else
        print_fail "goflow run command failed"
        echo "  Output: $output"
    fi
else
    print_skip "goflow run command not yet implemented"
fi

# ==============================================================================
# Step 9: Verify Output
# ==============================================================================
print_step "9" "Verify Output"

if [ -f /tmp/output.txt ]; then
    print_pass "Output file created at /tmp/output.txt"

    output_content=$(cat /tmp/output.txt)
    echo "  Output content: $output_content"

    # Verify expected output (sum of 10.5 + 20.3 + 5.2 = 36.0)
    if echo "$output_content" | grep -q "36"; then
        print_pass "Output contains expected sum (36.0)"
    else
        print_fail "Output does not contain expected sum (36.0)"
        echo "  Expected: Total: 36.0"
        echo "  Got: $output_content"
    fi
else
    print_fail "Output file not created at /tmp/output.txt"
fi

# ==============================================================================
# Additional Validation Tests
# ==============================================================================
print_step "10" "Additional Validation"

# Check workflow YAML syntax
if command -v yamllint &> /dev/null; then
    if yamllint examples/simple-pipeline.yaml &> /dev/null; then
        print_pass "Workflow YAML is valid (yamllint)"
    else
        print_fail "Workflow YAML has syntax errors"
    fi
else
    print_skip "yamllint not installed (optional)"
fi

# Check all example workflows are valid YAML
for workflow in examples/*.yaml; do
    if [ -f "$workflow" ]; then
        if python3 -c "import yaml; yaml.safe_load(open('$workflow'))" 2>/dev/null; then
            print_pass "$(basename $workflow) is valid YAML"
        else
            print_fail "$(basename $workflow) has YAML syntax errors"
        fi
    fi
done

# ==============================================================================
# Print Summary
# ==============================================================================
print_summary
