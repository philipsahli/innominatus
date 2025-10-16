#!/bin/bash

#
# innominatus Environment Setup & Validation Script
#
# This script validates your development environment and installs dependencies.
#
# Usage:
#   ./setup.sh           # Full setup and validation
#   ./setup.sh --check   # Only check prerequisites (no installation)
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Flags
CHECK_ONLY=false
if [[ "$1" == "--check" ]]; then
  CHECK_ONLY=true
fi

# ============================================================
# Helper Functions
# ============================================================

print_header() {
  echo ""
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
  echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
  echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
  echo -e "${RED}✗${NC} $1"
}

print_info() {
  echo -e "${BLUE}ℹ${NC} $1"
}

check_command() {
  local cmd=$1
  local name=$2
  local required=$3

  if command -v "$cmd" &> /dev/null; then
    local version=$($cmd --version 2>&1 | head -n1)
    print_success "$name is installed: $version"
    return 0
  else
    if [[ "$required" == "true" ]]; then
      print_error "$name is NOT installed (required)"
      return 1
    else
      print_warning "$name is NOT installed (optional)"
      return 0
    fi
  fi
}

# ============================================================
# Header
# ============================================================

clear
echo ""
echo -e "${BLUE}╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                                                   ║${NC}"
echo -e "${BLUE}║          innominatus Environment Setup            ║${NC}"
echo -e "${BLUE}║                                                   ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════╝${NC}"
echo ""

if [[ "$CHECK_ONLY" == "true" ]]; then
  print_info "Running in CHECK ONLY mode (no installations)"
  echo ""
fi

# ============================================================
# 1. Check Prerequisites
# ============================================================

print_header "1. Checking Prerequisites"

# Track if all required tools are present
ALL_REQUIRED_PRESENT=true

# Go (required)
if ! check_command "go" "Go" "true"; then
  ALL_REQUIRED_PRESENT=false
  print_error "Install Go from: https://go.dev/dl/"
  print_error "Required version: 1.24 or higher"
fi

# Node.js (required)
if ! check_command "node" "Node.js" "true"; then
  ALL_REQUIRED_PRESENT=false
  print_error "Install Node.js from: https://nodejs.org/"
  print_error "Required version: 18 or higher"
fi

# npm (required)
if ! check_command "npm" "npm" "true"; then
  ALL_REQUIRED_PRESENT=false
  print_error "npm should be installed with Node.js"
fi

# kubectl (optional for demo)
check_command "kubectl" "kubectl" "false"

# helm (optional for demo)
check_command "helm" "Helm" "false"

# docker (optional for demo)
check_command "docker" "Docker" "false"

# psql (optional for database checks)
check_command "psql" "PostgreSQL Client (psql)" "false"

echo ""

if [[ "$ALL_REQUIRED_PRESENT" == "false" ]]; then
  print_error "Missing required prerequisites. Please install them and re-run setup.sh"
  exit 1
fi

print_success "All required prerequisites are installed!"

# ============================================================
# 2. Check Go Version
# ============================================================

print_header "2. Validating Go Version"

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_GO_VERSION="1.24"

print_info "Installed Go version: $GO_VERSION"
print_info "Required Go version: $REQUIRED_GO_VERSION or higher"

# Simple version comparison (works for 1.x versions)
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

if [[ "$GO_MAJOR" -eq 1 && "$GO_MINOR" -ge 24 ]] || [[ "$GO_MAJOR" -gt 1 ]]; then
  print_success "Go version meets requirements"
else
  print_error "Go version too old. Please upgrade to 1.24 or higher"
  exit 1
fi

echo ""

# ============================================================
# 3. Check Database Connection
# ============================================================

print_header "3. Validating PostgreSQL Connection"

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-idp_orchestrator}
DB_USER=${DB_USER:-orchestrator_user}
DB_PASSWORD=${DB_PASSWORD:-}

print_info "Database Host: $DB_HOST:$DB_PORT"
print_info "Database Name: $DB_NAME"
print_info "Database User: $DB_USER"

if command -v psql &> /dev/null; then
  if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
    print_success "PostgreSQL connection successful"
  else
    print_warning "Could not connect to PostgreSQL"
    print_warning "This is OK if you're using a different database or running server standalone"
    print_info "Set DB_HOST, DB_PORT, DB_USER, DB_PASSWORD env vars if needed"
  fi
else
  print_warning "psql not installed, skipping database connection check"
  print_info "Database connection will be validated when server starts"
fi

echo ""

# ============================================================
# 4. Install Go Dependencies
# ============================================================

if [[ "$CHECK_ONLY" == "false" ]]; then
  print_header "4. Installing Go Dependencies"

  print_info "Running: go mod download"
  if go mod download; then
    print_success "Go dependencies installed"
  else
    print_error "Failed to download Go dependencies"
    exit 1
  fi

  print_info "Running: go mod tidy"
  if go mod tidy; then
    print_success "Go modules tidied"
  else
    print_warning "go mod tidy had issues (this may be OK)"
  fi

  echo ""
fi

# ============================================================
# 5. Install Node.js Dependencies (web-ui)
# ============================================================

if [[ "$CHECK_ONLY" == "false" ]]; then
  print_header "5. Installing Node.js Dependencies (web-ui)"

  if [[ -d "web-ui" ]]; then
    print_info "Running: cd web-ui && npm install"

    cd web-ui

    if npm install; then
      print_success "Node.js dependencies installed"
    else
      print_error "Failed to install Node.js dependencies"
      exit 1
    fi

    cd ..
  else
    print_warning "web-ui directory not found, skipping npm install"
  fi

  echo ""
fi

# ============================================================
# 6. Build Binaries
# ============================================================

if [[ "$CHECK_ONLY" == "false" ]]; then
  print_header "6. Building Binaries"

  # Build server
  print_info "Building server: go build -o innominatus cmd/server/main.go"
  if go build -o innominatus cmd/server/main.go; then
    print_success "Server binary built: ./innominatus"
  else
    print_error "Failed to build server binary"
    exit 1
  fi

  # Build CLI
  print_info "Building CLI: go build -o innominatus-ctl cmd/cli/main.go"
  if go build -o innominatus-ctl cmd/cli/main.go; then
    print_success "CLI binary built: ./innominatus-ctl"
  else
    print_error "Failed to build CLI binary"
    exit 1
  fi

  # Build web-ui
  if [[ -d "web-ui" ]]; then
    print_info "Building web-ui: cd web-ui && npm run build"

    cd web-ui

    if npm run build; then
      print_success "Web UI built: web-ui/out/"
    else
      print_error "Failed to build web-ui"
      exit 1
    fi

    cd ..
  fi

  echo ""
fi

# ============================================================
# 7. Kubernetes Check (for demo environment)
# ============================================================

print_header "7. Kubernetes Environment Check (Optional)"

if command -v kubectl &> /dev/null; then
  if kubectl cluster-info &> /dev/null; then
    CONTEXT=$(kubectl config current-context)
    print_success "Kubernetes cluster is accessible"
    print_info "Current context: $CONTEXT"

    if [[ "$CONTEXT" == "docker-desktop" ]]; then
      print_success "Docker Desktop Kubernetes detected (good for demo-time)"
    fi
  else
    print_warning "kubectl is installed but no cluster is accessible"
    print_info "This is OK if you're not using the demo environment"
  fi
else
  print_warning "kubectl not installed"
  print_info "Install kubectl if you want to use demo-time feature"
fi

echo ""

# ============================================================
# 8. Health Checks
# ============================================================

print_header "8. Running Health Checks"

# Check if server is running
if curl -s http://localhost:8081/health &> /dev/null; then
  print_success "innominatus server is running at http://localhost:8081"
  print_info "Server health: OK"

  # Check Web UI
  if curl -s http://localhost:8081/ &> /dev/null; then
    print_success "Web UI is accessible at http://localhost:8081"
  fi

  # Check API
  if curl -s http://localhost:8081/api/specs &> /dev/null; then
    print_success "API is accessible at http://localhost:8081/api/"
  fi
else
  print_warning "innominatus server is NOT running"
  print_info "Start server with: ./innominatus"
fi

echo ""

# ============================================================
# 9. Summary Dashboard
# ============================================================

print_header "9. Setup Summary"

echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                                                   ║${NC}"
echo -e "${GREEN}║              Setup Complete! 🎉                    ║${NC}"
echo -e "${GREEN}║                                                   ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${BLUE}Quick Start Commands:${NC}"
echo ""
echo -e "  ${GREEN}1. Start Server:${NC}"
echo -e "     ./innominatus"
echo ""
echo -e "  ${GREEN}2. Access Web UI:${NC}"
echo -e "     http://localhost:8081"
echo ""
echo -e "  ${GREEN}3. Deploy with CLI:${NC}"
echo -e "     ./innominatus-ctl run deploy-app score-spec.yaml"
echo ""
echo -e "  ${GREEN}4. Install Demo Environment:${NC}"
echo -e "     ./innominatus-ctl demo-time"
echo ""
echo -e "  ${GREEN}5. Run Tests:${NC}"
echo -e "     go test ./..."
echo ""
echo -e "  ${GREEN}6. Run Verification:${NC}"
echo -e "     node verification/examples/test-verification.mjs"
echo ""

echo -e "${BLUE}Service URLs (when server is running):${NC}"
echo ""
echo -e "  • Web UI:            http://localhost:8081"
echo -e "  • API (User):        http://localhost:8081/api/"
echo -e "  • Swagger (User):    http://localhost:8081/swagger-user"
echo -e "  • Swagger (Admin):   http://localhost:8081/swagger-admin"
echo -e "  • Health:            http://localhost:8081/health"
echo -e "  • Readiness:         http://localhost:8081/ready"
echo -e "  • Metrics:           http://localhost:8081/metrics"
echo ""

echo -e "${BLUE}Documentation:${NC}"
echo ""
echo -e "  • Development Guide: CLAUDE.md"
echo -e "  • Quick Context:     DIGEST.md"
echo -e "  • User Guide:        README.md"
echo -e "  • Verification:      verification/README.md"
echo ""

echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo -e "  1. Read DIGEST.md for quick project context"
echo -e "  2. Read CLAUDE.md for development principles (SOLID, KISS, YAGNI)"
echo -e "  3. Start server: ./innominatus"
echo -e "  4. Open Web UI: http://localhost:8081"
echo -e "  5. Try demo environment: ./innominatus-ctl demo-time"
echo ""

if [[ "$CHECK_ONLY" == "true" ]]; then
  echo -e "${YELLOW}NOTE: This was a check-only run. Run ./setup.sh without --check to install dependencies.${NC}"
  echo ""
fi

echo -e "${GREEN}Happy coding! 🚀${NC}"
echo ""
