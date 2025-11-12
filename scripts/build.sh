#!/bin/bash
# GoFlow Cross-Platform Build Script
# Builds GoFlow binaries for multiple platforms with optimization flags

set -e  # Exit on error
set -u  # Exit on undefined variable

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DIR="${BUILD_DIR:-${PROJECT_ROOT}/bin}"
OUTPUT_DIR="${OUTPUT_DIR:-${BUILD_DIR}/releases}"

# Build modes
BUILD_MODE="${BUILD_MODE:-release}"  # Options: release, dev

# Supported platforms
declare -a PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Print functions
print_header() {
    echo -e "\n${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}  GoFlow Cross-Platform Build Script                         ${BLUE}║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}\n"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_separator() {
    echo -e "${BLUE}────────────────────────────────────────────────────────────────${NC}"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build GoFlow binaries for multiple platforms with cross-compilation support.

OPTIONS:
    -h, --help              Show this help message
    -v, --version VERSION   Set version string (default: git describe or 'dev')
    -m, --mode MODE         Build mode: release or dev (default: release)
    -p, --platform PLATFORM Build for specific platform (e.g., linux/amd64)
    -a, --all               Build for all supported platforms (default)
    -c, --clean             Clean build directory before building
    --no-checksums          Skip checksum generation
    --no-compress           Skip binary compression (UPX)

PLATFORMS:
    linux/amd64             Linux x86_64
    linux/arm64             Linux ARM64
    darwin/amd64            macOS Intel
    darwin/arm64            macOS Apple Silicon
    windows/amd64           Windows x86_64

BUILD MODES:
    release                 Optimized build with stripped symbols (-s -w)
    dev                     Development build with debug symbols

EXAMPLES:
    # Build for all platforms (release mode)
    $0 --all

    # Build for specific platform
    $0 --platform linux/amd64

    # Development build
    $0 --mode dev --platform darwin/arm64

    # Clean build with custom version
    $0 --clean --version v1.0.0 --all

ENVIRONMENT VARIABLES:
    VERSION                 Override version string
    BUILD_MODE              Override build mode
    BUILD_DIR               Override build directory
    OUTPUT_DIR              Override output directory
    GIT_COMMIT              Override git commit hash

EOF
    exit 0
}

# Parse command line arguments
BUILD_ALL=false
BUILD_PLATFORM=""
CLEAN_BUILD=false
GENERATE_CHECKSUMS=true
COMPRESS_BINARIES=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -m|--mode)
            BUILD_MODE="$2"
            shift 2
            ;;
        -p|--platform)
            BUILD_PLATFORM="$2"
            shift 2
            ;;
        -a|--all)
            BUILD_ALL=true
            shift
            ;;
        -c|--clean)
            CLEAN_BUILD=true
            shift
            ;;
        --no-checksums)
            GENERATE_CHECKSUMS=false
            shift
            ;;
        --no-compress)
            COMPRESS_BINARIES=false
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate build mode
if [[ "$BUILD_MODE" != "release" && "$BUILD_MODE" != "dev" ]]; then
    print_error "Invalid build mode: $BUILD_MODE"
    print_info "Valid modes: release, dev"
    exit 1
fi

# Determine what to build
if [[ "$BUILD_PLATFORM" != "" ]]; then
    PLATFORMS=("$BUILD_PLATFORM")
elif [[ "$BUILD_ALL" == "false" ]]; then
    # Default: build for current platform
    CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    CURRENT_ARCH=$(uname -m)

    # Normalize architecture names
    case "$CURRENT_ARCH" in
        x86_64)
            CURRENT_ARCH="amd64"
            ;;
        aarch64)
            CURRENT_ARCH="arm64"
            ;;
        arm64)
            CURRENT_ARCH="arm64"
            ;;
    esac

    PLATFORMS=("${CURRENT_OS}/${CURRENT_ARCH}")
fi

# Clean build directory if requested
if [[ "$CLEAN_BUILD" == "true" ]]; then
    print_info "Cleaning build directories..."
    rm -rf "$BUILD_DIR" "$OUTPUT_DIR"
    print_success "Build directories cleaned"
fi

# Create output directories
mkdir -p "$BUILD_DIR"
mkdir -p "$OUTPUT_DIR"

# Build flags based on mode
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

if [[ "$BUILD_MODE" == "release" ]]; then
    # Release mode: strip debug symbols for smaller binaries
    LDFLAGS="${LDFLAGS} -s -w"
    BUILD_FLAGS="-trimpath"
else
    # Dev mode: keep debug symbols
    BUILD_FLAGS=""
fi

# Print build configuration
print_header
print_info "Build Configuration:"
print_separator
echo "  Version:        ${VERSION}"
echo "  Build Mode:     ${BUILD_MODE}"
echo "  Git Commit:     ${GIT_COMMIT}"
echo "  Build Time:     ${BUILD_TIME}"
echo "  Build Dir:      ${BUILD_DIR}"
echo "  Output Dir:     ${OUTPUT_DIR}"
echo "  Platforms:      ${#PLATFORMS[@]}"
print_separator

# Verify dependencies
print_info "Verifying dependencies..."
go mod download
go mod verify
print_success "Dependencies verified"

# Build for each platform
BUILD_COUNT=0
FAILED_COUNT=0

for PLATFORM in "${PLATFORMS[@]}"; do
    # Parse platform string
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"

    # Binary name
    BINARY_NAME="goflow-${GOOS}-${GOARCH}"
    if [[ "$GOOS" == "windows" ]]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi

    OUTPUT_PATH="${OUTPUT_DIR}/${BINARY_NAME}"

    print_separator
    print_info "Building for ${GOOS}/${GOARCH}..."

    # Build command
    export GOOS GOARCH
    if go build ${BUILD_FLAGS} -ldflags="${LDFLAGS}" -o "${OUTPUT_PATH}" ./cmd/goflow; then
        # Get binary size
        BINARY_SIZE=$(stat -f%z "${OUTPUT_PATH}" 2>/dev/null || stat -c%s "${OUTPUT_PATH}" 2>/dev/null)
        BINARY_SIZE_MB=$((BINARY_SIZE / 1024 / 1024))

        # Check size limit (50MB)
        if [[ $BINARY_SIZE_MB -gt 50 ]]; then
            print_warning "Binary size ${BINARY_SIZE_MB}MB exceeds 50MB target"
        fi

        print_success "Built ${BINARY_NAME} (${BINARY_SIZE_MB}MB)"

        # Compress with UPX if requested and available
        if [[ "$COMPRESS_BINARIES" == "true" ]] && command -v upx &> /dev/null; then
            print_info "Compressing with UPX..."
            if upx --best --lzma "${OUTPUT_PATH}" 2>/dev/null; then
                COMPRESSED_SIZE=$(stat -f%z "${OUTPUT_PATH}" 2>/dev/null || stat -c%s "${OUTPUT_PATH}" 2>/dev/null)
                COMPRESSED_SIZE_MB=$((COMPRESSED_SIZE / 1024 / 1024))
                print_success "Compressed to ${COMPRESSED_SIZE_MB}MB"
            else
                print_warning "UPX compression failed"
            fi
        fi

        ((BUILD_COUNT++))
    else
        print_error "Build failed for ${GOOS}/${GOARCH}"
        ((FAILED_COUNT++))
    fi
done

# Generate checksums
if [[ "$GENERATE_CHECKSUMS" == "true" ]] && [[ $BUILD_COUNT -gt 0 ]]; then
    print_separator
    print_info "Generating checksums..."

    CHECKSUM_FILE="${OUTPUT_DIR}/checksums.txt"
    rm -f "$CHECKSUM_FILE"

    cd "$OUTPUT_DIR"

    # Generate SHA256 checksums
    for BINARY in goflow-*; do
        if [[ -f "$BINARY" ]]; then
            if command -v sha256sum &> /dev/null; then
                sha256sum "$BINARY" >> checksums.txt
            elif command -v shasum &> /dev/null; then
                shasum -a 256 "$BINARY" >> checksums.txt
            fi
        fi
    done

    if [[ -f checksums.txt ]]; then
        print_success "Checksums generated: ${CHECKSUM_FILE}"
    fi

    cd "$PROJECT_ROOT"
fi

# Build summary
print_separator
echo ""
print_header
print_info "Build Summary:"
print_separator
echo "  Total Platforms:    ${#PLATFORMS[@]}"
echo "  Successful Builds:  ${BUILD_COUNT}"
echo "  Failed Builds:      ${FAILED_COUNT}"
echo "  Output Directory:   ${OUTPUT_DIR}"
print_separator

# List built binaries
if [[ $BUILD_COUNT -gt 0 ]]; then
    echo ""
    print_info "Built Binaries:"
    print_separator

    for BINARY in "$OUTPUT_DIR"/goflow-*; do
        if [[ -f "$BINARY" && "$BINARY" != *checksums.txt ]]; then
            BINARY_NAME=$(basename "$BINARY")
            BINARY_SIZE=$(stat -f%z "$BINARY" 2>/dev/null || stat -c%s "$BINARY" 2>/dev/null)
            BINARY_SIZE_MB=$(echo "scale=2; $BINARY_SIZE / 1024 / 1024" | bc)
            printf "  %-30s %8s MB\n" "$BINARY_NAME" "$BINARY_SIZE_MB"
        fi
    done

    print_separator
fi

# Exit with appropriate code
if [[ $FAILED_COUNT -gt 0 ]]; then
    echo ""
    print_error "Build completed with ${FAILED_COUNT} failure(s)"
    exit 1
else
    echo ""
    print_success "All builds completed successfully! 🚀"
    exit 0
fi
