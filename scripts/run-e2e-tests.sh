#!/bin/bash
# Run End-to-End Integration Tests
#
# This script runs comprehensive E2E tests for innominatus against real infrastructure.
# It validates the complete GitOps deployment flow including Gitea, ArgoCD, and Kubernetes.
#
# Prerequisites:
# - Kubernetes cluster with kubectl access
# - Gitea running at http://gitea.localtest.me (or set GITEA_URL)
# - ArgoCD installed in 'argocd' namespace
# - GITEA_TOKEN environment variable set
#
# Quick setup: ./innominatus-ctl demo-time

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Innominatus E2E Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"

    # Check Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}✗ Go not found${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Go installed:${NC} $(go version)"

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}✗ kubectl not found${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ kubectl installed:${NC} $(kubectl version --client --short 2>/dev/null || kubectl version --client)"

    # Check Kubernetes cluster access
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}✗ Kubernetes cluster not accessible${NC}"
        echo -e "${YELLOW}  Make sure kubectl is configured with a valid cluster${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Kubernetes cluster accessible${NC}"

    # Check GITEA_TOKEN
    if [ -z "$GITEA_TOKEN" ]; then
        echo -e "${RED}✗ GITEA_TOKEN environment variable not set${NC}"
        echo -e "${YELLOW}  Generate token at: http://gitea.localtest.me/user/settings/applications${NC}"
        echo -e "${YELLOW}  Then run: export GITEA_TOKEN='your-token-here'${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ GITEA_TOKEN set${NC}"

    # Check Gitea accessibility
    GITEA_URL="${GITEA_URL:-http://gitea.localtest.me}"
    if curl -s -f -H "Authorization: token $GITEA_TOKEN" "$GITEA_URL/api/v1/user" > /dev/null 2>&1; then
        GITEA_USER=$(curl -s -H "Authorization: token $GITEA_TOKEN" "$GITEA_URL/api/v1/user" | grep -o '"username":"[^"]*"' | cut -d'"' -f4)
        echo -e "${GREEN}✓ Gitea accessible:${NC} $GITEA_URL (user: $GITEA_USER)"
    else
        echo -e "${YELLOW}⚠  Gitea not accessible at $GITEA_URL${NC}"
        echo -e "${YELLOW}  Tests will fail if Gitea is not running${NC}"
        echo -e "${YELLOW}  Quick setup: ./innominatus-ctl demo-time${NC}"
    fi

    # Check ArgoCD namespace
    if kubectl get namespace argocd &> /dev/null; then
        echo -e "${GREEN}✓ ArgoCD namespace exists${NC}"
    else
        echo -e "${YELLOW}⚠  ArgoCD namespace not found${NC}"
        echo -e "${YELLOW}  Tests will fail if ArgoCD is not installed${NC}"
        echo -e "${YELLOW}  Install: kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml${NC}"
    fi

    # Check Gitea organization exists
    GITEA_ORG="${GITEA_ORG:-platform}"
    if curl -s -f -H "Authorization: token $GITEA_TOKEN" "$GITEA_URL/api/v1/orgs/$GITEA_ORG" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Gitea organization exists:${NC} $GITEA_ORG"
    else
        echo -e "${YELLOW}⚠  Gitea organization '$GITEA_ORG' not found${NC}"
        echo -e "${YELLOW}  Create via UI or API:${NC}"
        echo -e "${YELLOW}  curl -X POST '$GITEA_URL/api/v1/orgs' \\${NC}"
        echo -e "${YELLOW}    -H 'Authorization: token \$GITEA_TOKEN' \\${NC}"
        echo -e "${YELLOW}    -H 'Content-Type: application/json' \\${NC}"
        echo -e "${YELLOW}    -d '{\"username\": \"$GITEA_ORG\"}'${NC}"
    fi

    echo ""
}

# Run tests
run_tests() {
    echo -e "${BLUE}Running E2E tests...${NC}"
    echo ""

    # Set environment variables
    export RUN_E2E_TESTS=true
    export GITEA_URL="${GITEA_URL:-http://gitea.localtest.me}"
    export GITEA_ORG="${GITEA_ORG:-platform}"

    # Change to project root
    cd "$(dirname "$0")/.."

    # Run tests with verbose output
    if go test -v -tags e2e -timeout 30m ./tests/e2e/... 2>&1 | tee e2e-test-output.log; then
        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}✅ E2E Tests PASSED${NC}"
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}Test output saved to: e2e-test-output.log${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}❌ E2E Tests FAILED${NC}"
        echo -e "${RED}========================================${NC}"
        echo -e "${RED}Test output saved to: e2e-test-output.log${NC}"
        echo ""
        echo -e "${YELLOW}Troubleshooting:${NC}"
        echo -e "${YELLOW}1. Check orchestration engine logs${NC}"
        echo -e "${YELLOW}2. Verify Gitea is accessible: curl -H 'Authorization: token \$GITEA_TOKEN' \$GITEA_URL/api/v1/user${NC}"
        echo -e "${YELLOW}3. Check ArgoCD is running: kubectl get pods -n argocd${NC}"
        echo -e "${YELLOW}4. Review test output: cat e2e-test-output.log${NC}"
        echo ""
        return 1
    fi
}

# Show help
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Run End-to-End integration tests for innominatus.

OPTIONS:
    -h, --help          Show this help message
    -s, --skip-checks   Skip prerequisite checks
    -t, --test NAME     Run specific test (e.g., TestContainerGitOpsTestSuite)
    -v, --verbose       Extra verbose output

ENVIRONMENT VARIABLES:
    GITEA_TOKEN         Gitea API token (required)
    GITEA_URL           Gitea base URL (default: http://gitea.localtest.me)
    GITEA_ORG           Gitea organization (default: platform)
    KUBECONFIG          Kubernetes config path (default: ~/.kube/config)
    RUN_E2E_TESTS       Enable E2E tests (automatically set to true)

EXAMPLES:
    # Run all E2E tests
    $0

    # Run specific test
    $0 --test TestContainerGitOpsTestSuite

    # Skip prerequisite checks
    $0 --skip-checks

SETUP:
    # Quick demo environment setup
    ./innominatus-ctl demo-time

    # Generate Gitea token
    # 1. Login to http://gitea.localtest.me (admin/admin)
    # 2. Go to Settings → Applications
    # 3. Generate New Token
    # 4. export GITEA_TOKEN='your-token-here'

    # Create Gitea organization
    curl -X POST "http://gitea.localtest.me/api/v1/orgs" \\
      -H "Authorization: token \$GITEA_TOKEN" \\
      -H "Content-Type: application/json" \\
      -d '{"username": "platform"}'

For more information, see: tests/e2e/README.md
EOF
}

# Parse command line arguments
SKIP_CHECKS=false
TEST_NAME=""
VERBOSE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -s|--skip-checks)
            SKIP_CHECKS=true
            shift
            ;;
        -t|--test)
            TEST_NAME="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
done

# Run prerequisite checks unless skipped
if [ "$SKIP_CHECKS" = false ]; then
    check_prerequisites
fi

# Run tests
if [ -n "$TEST_NAME" ]; then
    echo -e "${BLUE}Running specific test: ${TEST_NAME}${NC}"
    export RUN_E2E_TESTS=true
    cd "$(dirname "$0")/.."
    go test $VERBOSE -tags e2e -timeout 30m -run "$TEST_NAME" ./tests/e2e/...
else
    run_tests
fi
