#!/bin/bash
# Octo Installation Script
# Installs Octo Docker management CLI

set -euo pipefail

# Configuration
REPO="bsisduck/octo"
BINARY_NAME="octo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Icons
ICON_SUCCESS="✓"
ICON_ERROR="✗"
ICON_INFO="→"

# Print functions
info() { echo -e "${BLUE}${ICON_INFO}${NC} $1"; }
success() { echo -e "${GREEN}${ICON_SUCCESS}${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
error() { echo -e "${RED}${ICON_ERROR}${NC} $1" >&2; }

# Detect OS and architecture
detect_platform() {
    local os arch

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    case "$os" in
        darwin|linux)
            ;;
        mingw*|msys*|cygwin*)
            os="windows"
            ;;
        *)
            error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Get latest version from GitHub
get_latest_version() {
    local version

    if command -v curl > /dev/null 2>&1; then
        version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null |
            grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
    elif command -v wget > /dev/null 2>&1; then
        version=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null |
            grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
    fi

    # Remove 'v' prefix if present
    version="${version#v}"
    version="${version#V}"

    echo "$version"
}

# Download binary
download_binary() {
    local version="$1"
    local platform="$2"
    local url
    local tmp_file

    # Construct download URL
    local binary_name="${BINARY_NAME}-${platform}"
    [[ "$platform" == *"windows"* ]] && binary_name="${binary_name}.exe"

    url="https://github.com/${REPO}/releases/download/v${version}/${binary_name}"
    tmp_file=$(mktemp)

    info "Downloading Octo v${version} for ${platform}..."

    if command -v curl > /dev/null 2>&1; then
        if ! curl -fsSL "$url" -o "$tmp_file" 2>/dev/null; then
            error "Download failed from: $url"
            rm -f "$tmp_file"
            return 1
        fi
    elif command -v wget > /dev/null 2>&1; then
        if ! wget -q "$url" -O "$tmp_file" 2>/dev/null; then
            error "Download failed from: $url"
            rm -f "$tmp_file"
            return 1
        fi
    else
        error "curl or wget is required for installation"
        return 1
    fi

    echo "$tmp_file"
}

# Build from source
build_from_source() {
    info "Building from source..."

    if ! command -v go > /dev/null 2>&1; then
        error "Go is required to build from source"
        error "Install Go from https://golang.org/dl/"
        exit 1
    fi

    local tmp_dir
    tmp_dir=$(mktemp -d)

    info "Cloning repository..."
    if ! git clone --depth 1 "https://github.com/${REPO}.git" "$tmp_dir" 2>/dev/null; then
        error "Failed to clone repository"
        rm -rf "$tmp_dir"
        exit 1
    fi

    cd "$tmp_dir"

    info "Building..."
    if ! make build 2>/dev/null; then
        error "Build failed"
        rm -rf "$tmp_dir"
        exit 1
    fi

    echo "$tmp_dir/bin/${BINARY_NAME}"
}

# Install binary
install_binary() {
    local binary_path="$1"
    local requires_sudo=false

    # Check if we need sudo
    if [[ ! -w "$INSTALL_DIR" ]]; then
        requires_sudo=true
    fi

    info "Installing to ${INSTALL_DIR}..."

    if [[ "$requires_sudo" == "true" ]]; then
        sudo mkdir -p "$INSTALL_DIR"
        sudo cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

        # Create 'oc' alias
        sudo ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/oc"
    else
        mkdir -p "$INSTALL_DIR"
        cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

        # Create 'oc' alias
        ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/oc"
    fi
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" > /dev/null 2>&1; then
        local version
        version=$("$BINARY_NAME" version 2>/dev/null | head -1 || echo "unknown")
        success "Octo installed successfully!"
        echo ""
        echo "  $version"
        echo ""
        echo "  Run 'octo' to start the interactive menu"
        echo "  Run 'octo --help' for usage information"
        echo "  'oc' is available as a short alias"
        return 0
    else
        warn "Octo was installed but may not be in your PATH"
        warn "Add ${INSTALL_DIR} to your PATH or use the full path"
        return 1
    fi
}

# Check Docker availability
check_docker() {
    if command -v docker > /dev/null 2>&1; then
        if docker info > /dev/null 2>&1; then
            success "Docker is available and running"
        else
            warn "Docker is installed but not running"
            warn "Start Docker before using Octo"
        fi
    else
        warn "Docker is not installed"
        warn "Install Docker from https://docs.docker.com/get-docker/"
    fi
}

# Main installation flow
main() {
    echo ""
    echo -e "${BLUE}   ___       _"
    echo "  / _ \  ___| |_ ___"
    echo " | | | |/ __| __/ _ \\"
    echo " | |_| | (__| || (_) |"
    echo -e "  \___/ \___|\__\___/${NC}"
    echo ""
    echo "  Docker Container Management CLI"
    echo ""

    local platform
    platform=$(detect_platform)
    info "Detected platform: $platform"

    local version
    version=$(get_latest_version)

    local binary_path=""

    if [[ -n "$version" ]]; then
        info "Latest version: v${version}"
        binary_path=$(download_binary "$version" "$platform") || binary_path=""
    fi

    # Fall back to building from source if download fails
    if [[ -z "$binary_path" || ! -f "$binary_path" ]]; then
        warn "Could not download pre-built binary"
        info "Attempting to build from source..."
        binary_path=$(build_from_source)
    fi

    if [[ -z "$binary_path" || ! -f "$binary_path" ]]; then
        error "Installation failed"
        exit 1
    fi

    install_binary "$binary_path"

    # Cleanup
    rm -f "$binary_path" 2>/dev/null || true

    echo ""
    verify_installation
    echo ""
    check_docker
    echo ""
}

# Handle arguments
case "${1:-}" in
    --help|-h)
        echo "Octo Installation Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version, -v  Show version to install"
        echo ""
        echo "Environment variables:"
        echo "  INSTALL_DIR    Installation directory (default: /usr/local/bin)"
        exit 0
        ;;
    --version|-v)
        version=$(get_latest_version)
        if [[ -n "$version" ]]; then
            echo "v${version}"
        else
            echo "Could not determine latest version"
            exit 1
        fi
        exit 0
        ;;
    *)
        main
        ;;
esac
