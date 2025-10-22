#!/usr/bin/env bash
#
# innominatus installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/philipsahli/innominatus/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/philipsahli/innominatus/main/install.sh | bash -s -- --version v0.1.0
#   curl -fsSL https://raw.githubusercontent.com/philipsahli/innominatus/main/install.sh | bash -s -- --component cli
#

set -e

# Configuration
REPO="philipsahli/innominatus"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"
COMPONENT="both"  # server, cli, or both

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --component)
      COMPONENT="$2"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    --help)
      echo "innominatus installer"
      echo ""
      echo "Usage:"
      echo "  $0 [options]"
      echo ""
      echo "Options:"
      echo "  --version <version>      Install specific version (default: latest)"
      echo "  --component <component>  Install server, cli, or both (default: both)"
      echo "  --install-dir <dir>      Installation directory (default: /usr/local/bin)"
      echo "  --help                   Show this help message"
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      exit 1
      ;;
  esac
done

# Helper functions
log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Linux*)     echo "Linux";;
    Darwin*)    echo "Darwin";;
    MINGW*|MSYS*|CYGWIN*) echo "Windows";;
    *)          echo "unknown";;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "x86_64";;
    aarch64|arm64) echo "arm64";;
    armv7l)       echo "armv7";;
    i386|i686)    echo "i386";;
    *)            echo "unknown";;
  esac
}

# Get latest release version from GitHub
get_latest_version() {
  local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
  local version

  if command -v curl >/dev/null 2>&1; then
    version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  elif command -v wget >/dev/null 2>&1; then
    version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  else
    log_error "Neither curl nor wget is available. Please install one of them."
    exit 1
  fi

  echo "$version"
}

# Download and install binary
install_binary() {
  local component=$1
  local os=$2
  local arch=$3
  local version=$4

  local binary_name
  local archive_name
  local archive_prefix

  if [ "$component" = "server" ]; then
    binary_name="innominatus"
    archive_prefix="innominatus-server"
  else
    binary_name="innominatus-ctl"
    archive_prefix="innominatus-ctl"
  fi

  # Construct archive name based on goreleaser naming convention
  # Format: innominatus-server_<version>_<os>_<arch>.tar.gz or innominatus-ctl_<version>_<os>_<arch>.tar.gz
  archive_name="${archive_prefix}_${version#v}_${os}_${arch}.tar.gz"

  local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

  log_info "Downloading ${component} from ${download_url}..."

  # Create temporary directory
  local tmp_dir
  tmp_dir=$(mktemp -d)
  trap "rm -rf ${tmp_dir}" EXIT

  # Download archive
  if command -v curl >/dev/null 2>&1; then
    if ! curl -fsSL -o "${tmp_dir}/${archive_name}" "$download_url"; then
      log_error "Failed to download ${component}. URL: ${download_url}"
      return 1
    fi
  elif command -v wget >/dev/null 2>&1; then
    if ! wget -qO "${tmp_dir}/${archive_name}" "$download_url"; then
      log_error "Failed to download ${component}. URL: ${download_url}"
      return 1
    fi
  fi

  # Extract archive
  log_info "Extracting ${archive_name}..."
  tar -xzf "${tmp_dir}/${archive_name}" -C "${tmp_dir}"

  # Install binary
  log_info "Installing ${binary_name} to ${INSTALL_DIR}..."

  if [ -w "$INSTALL_DIR" ]; then
    install -m 755 "${tmp_dir}/${binary_name}" "${INSTALL_DIR}/${binary_name}"
  else
    log_warn "Requires sudo to install to ${INSTALL_DIR}"
    sudo install -m 755 "${tmp_dir}/${binary_name}" "${INSTALL_DIR}/${binary_name}"
  fi

  log_info "${GREEN}✓${NC} ${binary_name} installed successfully!"
}

# Main installation logic
main() {
  log_info "innominatus installer"
  echo ""

  # Detect OS and architecture
  local os
  local arch
  os=$(detect_os)
  arch=$(detect_arch)

  if [ "$os" = "unknown" ] || [ "$arch" = "unknown" ]; then
    log_error "Unsupported OS (${os}) or architecture (${arch})"
    exit 1
  fi

  log_info "Detected OS: ${os}"
  log_info "Detected architecture: ${arch}"

  # Get version
  if [ "$VERSION" = "latest" ]; then
    log_info "Fetching latest release version..."
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
      log_error "Failed to fetch latest version"
      exit 1
    fi
  fi

  log_info "Installing version: ${VERSION}"

  # Verify installation directory exists
  if [ ! -d "$INSTALL_DIR" ]; then
    log_error "Installation directory ${INSTALL_DIR} does not exist"
    exit 1
  fi

  # Install components
  case "$COMPONENT" in
    server)
      install_binary "server" "$os" "$arch" "$VERSION"
      ;;
    cli)
      install_binary "cli" "$os" "$arch" "$VERSION"
      ;;
    both)
      install_binary "server" "$os" "$arch" "$VERSION"
      install_binary "cli" "$os" "$arch" "$VERSION"
      ;;
    *)
      log_error "Invalid component: ${COMPONENT}. Must be 'server', 'cli', or 'both'"
      exit 1
      ;;
  esac

  echo ""
  log_info "${GREEN}Installation complete!${NC}"
  echo ""

  # Show next steps
  if [ "$COMPONENT" = "server" ] || [ "$COMPONENT" = "both" ]; then
    echo "Start the server:"
    echo "  innominatus"
    echo ""
  fi

  if [ "$COMPONENT" = "cli" ] || [ "$COMPONENT" = "both" ]; then
    echo "Use the CLI:"
    echo "  innominatus-ctl --help"
    echo "  innominatus-ctl list"
    echo ""
    echo "Set up shell completion (optional):"
    echo "  innominatus-ctl completion bash > /etc/bash_completion.d/innominatus-ctl"
    echo "  source /etc/bash_completion.d/innominatus-ctl"
    echo ""
  fi

  echo "Documentation: https://github.com/${REPO}"
}

main
