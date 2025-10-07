#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${GREEN}innominatus Cleanup Script${NC}"
echo "================================"
echo ""

# Function to print status
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Parse command line arguments
CLEAN_BUILD=false
CLEAN_DB=false
CLEAN_TEMP=false
CLEAN_DEPS=false
CLEAN_ALL=false
CLEAN_DEMO=false

show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Cleanup script for innominatus project

OPTIONS:
    --build         Clean build artifacts (binaries, web-ui build output)
    --db            Clean database (truncate tables)
    --temp          Clean temporary files (terraform workspaces, logs)
    --deps          Clean dependency caches (go mod cache, node_modules)
    --demo          Clean demo environment (calls demo-nuke)
    --all           Clean everything (all of the above)
    -h, --help      Show this help message

EXAMPLES:
    $0 --build              # Clean only build artifacts
    $0 --build --temp       # Clean builds and temp files
    $0 --all                # Complete cleanup
    $0 --demo               # Clean demo environment only

EOF
}

# Parse arguments
if [ $# -eq 0 ]; then
    show_help
    exit 0
fi

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            CLEAN_BUILD=true
            shift
            ;;
        --db)
            CLEAN_DB=true
            shift
            ;;
        --temp)
            CLEAN_TEMP=true
            shift
            ;;
        --deps)
            CLEAN_DEPS=true
            shift
            ;;
        --demo)
            CLEAN_DEMO=true
            shift
            ;;
        --all)
            CLEAN_ALL=true
            CLEAN_BUILD=true
            CLEAN_DB=true
            CLEAN_TEMP=true
            CLEAN_DEPS=true
            CLEAN_DEMO=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Clean build artifacts
if [ "$CLEAN_BUILD" = true ]; then
    echo ""
    echo "Cleaning build artifacts..."

    # Remove Go binaries
    if [ -f "innominatus" ]; then
        rm innominatus
        print_status "Removed innominatus binary"
    fi

    if [ -f "innominatus-ctl" ]; then
        rm innominatus-ctl
        print_status "Removed innominatus-ctl binary"
    fi

    # Clean GoReleaser dist
    if [ -d "dist" ]; then
        rm -rf dist
        print_status "Removed dist/ directory"
    fi

    # Clean web-ui build output
    if [ -d "web-ui/out" ]; then
        rm -rf web-ui/out
        print_status "Removed web-ui/out/ directory"
    fi

    if [ -d "web-ui/.next" ]; then
        rm -rf web-ui/.next
        print_status "Removed web-ui/.next/ directory"
    fi
fi

# Clean database
if [ "$CLEAN_DB" = true ]; then
    echo ""
    echo "Cleaning database..."

    DB_HOST="${DB_HOST:-localhost}"
    DB_PORT="${DB_PORT:-5432}"
    DB_NAME="${DB_NAME:-idp_orchestrator}"
    DB_USER="${DB_USER:-postgres}"

    print_warning "Truncating tables in database: $DB_NAME"

    # Check if psql is available
    if command -v psql &> /dev/null; then
        # Test database connection first
        if PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
            # Truncate tables
            PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
                TRUNCATE TABLE workflows CASCADE;
                TRUNCATE TABLE workflow_steps CASCADE;
                TRUNCATE TABLE deployments CASCADE;
            " && print_status "Database tables truncated"
        else
            print_warning "Database cleanup skipped (not accessible at $DB_HOST:$DB_PORT)"
        fi
    else
        print_warning "psql not found - skipping database cleanup"
    fi
fi

# Clean temporary files
if [ "$CLEAN_TEMP" = true ]; then
    echo ""
    echo "Cleaning temporary files..."

    # Clean terraform workspaces
    if [ -d "terraform" ]; then
        find terraform -type d -name ".terraform" -exec rm -rf {} + 2>/dev/null || true
        find terraform -type f -name "terraform.tfstate*" -delete 2>/dev/null || true
        find terraform -type f -name ".terraform.lock.hcl" -delete 2>/dev/null || true
        find terraform -type f -name "infra_provisioned.txt" -delete 2>/dev/null || true
        print_status "Cleaned terraform artifacts"
    fi

    # Clean log files
    find . -type f -name "*.log" -delete 2>/dev/null || true
    print_status "Removed log files"

    # Clean temporary score files
    find . -type f -name "test-*.yaml" -delete 2>/dev/null || true
    print_status "Removed temporary test files"
fi

# Clean dependency caches
if [ "$CLEAN_DEPS" = true ]; then
    echo ""
    echo "Cleaning dependency caches..."

    # Clean Go module cache (optional - only removes downloaded dependencies)
    if command -v go &> /dev/null; then
        print_warning "Running go clean -modcache (this may take a moment)..."
        go clean -modcache 2>/dev/null || print_warning "Could not clean Go module cache"
    fi

    # Clean node_modules
    if [ -d "web-ui/node_modules" ]; then
        rm -rf web-ui/node_modules
        print_status "Removed web-ui/node_modules"
    fi

    # Clean npm cache
    if [ -f "web-ui/package-lock.json" ]; then
        cd web-ui
        npm cache clean --force 2>/dev/null || true
        cd ..
        print_status "Cleaned npm cache"
    fi
fi

# Clean demo environment
if [ "$CLEAN_DEMO" = true ]; then
    echo ""
    echo "Cleaning demo environment..."

    if [ -f "innominatus-ctl" ]; then
        ./innominatus-ctl demo-nuke
        print_status "Demo environment cleaned"
    else
        print_warning "innominatus-ctl not found - build it first to clean demo environment"
        print_warning "Run: go build -o innominatus-ctl cmd/cli/main.go"
    fi
fi

echo ""
echo -e "${GREEN}Cleanup completed!${NC}"

if [ "$CLEAN_ALL" = true ]; then
    echo ""
    echo "Complete cleanup performed. To rebuild:"
    echo "  1. go build -o innominatus cmd/server/main.go"
    echo "  2. go build -o innominatus-ctl cmd/cli/main.go"
    echo "  3. ./scripts/build-web-ui.sh"
fi
