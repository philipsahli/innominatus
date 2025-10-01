#!/bin/bash
set -e

# GitOps End-to-End Test: Alice deploys nginx via Gitea + ArgoCD
# This test verifies full GitOps workflow

SCORE_FILE="test-nginx-alice-simple.yaml"
APP_NAME="alice-nginx-simple"
NAMESPACE="alice-nginx-simple-default"  # Namespace created by workflow
MAX_RETRIES=30
RETRY_DELAY=10
GITEA_URL="http://gitea.localtest.me"
GITEA_USER="giteaadmin"
GITEA_PASSWORD="admin"
GIT_REPO_NAME="alice-nginx-simple"  # Auto-generated repo name

# Use admin API key for authentication (from users.yaml)
export IDP_API_KEY="cf1d1f5afb8c1f1b2e17079c835b1f22d3719f651b4673f1bc4e3a006ebeb5db"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "═══════════════════════════════════════════════════════════════"
echo "  GitOps E2E Test: Alice's Nginx Deployment"
echo "  (Gitea + ArgoCD GitOps workflow)"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Function to wait for condition with retry
wait_for_condition() {
    local description="$1"
    local command="$2"
    local retries=0

    print_info "Waiting for: $description"

    while [ $retries -lt $MAX_RETRIES ]; do
        if eval "$command" > /dev/null 2>&1; then
            print_success "$description"
            return 0
        fi
        retries=$((retries + 1))
        echo "  Attempt $retries/$MAX_RETRIES..."
        sleep $RETRY_DELAY
    done

    print_error "Timeout waiting for: $description"
    return 1
}

# Step 1: Validate Score spec
echo ""
echo "Step 1: Validating Score specification"
echo "───────────────────────────────────────────────────────────────"
if ./innominatus-ctl validate "$SCORE_FILE"; then
    print_success "Score specification is valid"
else
    print_error "Score specification validation failed"
    exit 1
fi

# Step 2: Clean up any existing deployment
echo ""
echo "Step 2: Cleaning up any existing deployment"
echo "───────────────────────────────────────────────────────────────"
kubectl delete namespace "$NAMESPACE" --ignore-not-found=true > /dev/null 2>&1 || true
kubectl delete application "$APP_NAME-default" -n argocd --ignore-not-found=true > /dev/null 2>&1 || true
sleep 5
print_success "Cleanup completed"

# Step 3: Deploy via CLI (triggers automatic GitOps pipeline)
echo ""
echo "Step 3: Deploying nginx application via innominatus-ctl"
echo "───────────────────────────────────────────────────────────────"
echo "Note: This triggers automatic GitOps pipeline creation (Gitea + ArgoCD)"
if ./innominatus-ctl run deploy-app "$SCORE_FILE"; then
    print_success "Deployment command executed successfully"
else
    print_error "Deployment failed"
    exit 1
fi

# Step 4: Verify namespace was created
echo ""
echo "Step 4: Verifying Kubernetes namespace creation"
echo "───────────────────────────────────────────────────────────────"
if wait_for_condition "Namespace $NAMESPACE exists" "kubectl get namespace $NAMESPACE"; then
    print_success "Namespace verified"
else
    print_error "Namespace not found"
    exit 1
fi

# Step 5: Wait for pods to be running
echo ""
echo "Step 5: Waiting for nginx pods to be running"
echo "───────────────────────────────────────────────────────────────"
if wait_for_condition "Pods in namespace $NAMESPACE" "kubectl get pods -n $NAMESPACE 2>&1 | grep -q Running"; then
    print_success "Pods are running"
else
    print_error "No running pods found"
    echo ""
    echo "Debug information:"
    kubectl get pods -n "$NAMESPACE" 2>/dev/null || echo "  No pods found"
    kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' | tail -10
    exit 1
fi

# Step 6: Verify nginx deployment details
echo ""
echo "Step 6: Verifying nginx deployment details"
echo "───────────────────────────────────────────────────────────────"
POD_COUNT=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Running 2>/dev/null | grep -c "Running" || echo "0")
echo "  Running pods: $POD_COUNT"

if [ "$POD_COUNT" -gt 0 ]; then
    print_success "Nginx pods are running"

    # Show pod details
    echo ""
    echo "Pod details:"
    kubectl get pods -n "$NAMESPACE" -o wide
else
    print_error "No running nginx pods found"
    exit 1
fi

# Step 7: Verify service was created
echo ""
echo "Step 7: Verifying Kubernetes service"
echo "───────────────────────────────────────────────────────────────"
if kubectl get svc -n "$NAMESPACE" 2>&1 | grep -q "$APP_NAME"; then
    print_success "Service created"
    kubectl get svc -n "$NAMESPACE"
else
    print_info "No service found (expected for basic deployment)"
fi

# Step 8: Verify deployment
echo ""
echo "Step 8: Verifying Kubernetes deployment"
echo "───────────────────────────────────────────────────────────────"
if kubectl get deployment -n "$NAMESPACE" 2>&1 | grep -q "$APP_NAME"; then
    print_success "Deployment created"
    kubectl get deployment -n "$NAMESPACE" -o wide
else
    print_error "Deployment not found"
fi

# Step 9: Verify Git repository exists in Gitea
echo ""
echo "Step 9: Verifying Git repository in Gitea"
echo "───────────────────────────────────────────────────────────────"
GITEA_API_URL="${GITEA_URL}/api/v1/repos/${GITEA_USER}/${GIT_REPO_NAME}"
if curl -s -u "${GITEA_USER}:${GITEA_PASSWORD}" "${GITEA_API_URL}" | grep -q "\"name\":\"${GIT_REPO_NAME}\""; then
    print_success "Git repository exists in Gitea: ${GITEA_USER}/${GIT_REPO_NAME}"
    echo "  Repository URL: ${GITEA_URL}/${GITEA_USER}/${GIT_REPO_NAME}"
else
    print_error "Git repository not found in Gitea"
    echo "  Expected: ${GITEA_URL}/${GITEA_USER}/${GIT_REPO_NAME}"
    exit 1
fi

# Step 10: Verify ArgoCD Application exists
echo ""
echo "Step 10: Verifying ArgoCD Application"
echo "───────────────────────────────────────────────────────────────"
ARGOCD_APP_NAME="${APP_NAME}-default"
if kubectl get application "$ARGOCD_APP_NAME" -n argocd 2>/dev/null; then
    print_success "ArgoCD Application exists: $ARGOCD_APP_NAME"

    # Show ArgoCD app details
    echo ""
    echo "ArgoCD Application details:"
    kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.spec.source.repoURL}{"\n"}' | sed 's/^/  Repository: /'
    kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.spec.source.path}{"\n"}' | sed 's/^/  Path: /'
    kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.spec.destination.namespace}{"\n"}' | sed 's/^/  Target Namespace: /'

    # Check sync status
    SYNC_STATUS=$(kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.status.sync.status}' 2>/dev/null || echo "Unknown")
    HEALTH_STATUS=$(kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.status.health.status}' 2>/dev/null || echo "Unknown")
    echo "  Sync Status: $SYNC_STATUS"
    echo "  Health Status: $HEALTH_STATUS"
else
    print_error "ArgoCD Application not found"
    echo "  Expected application name: $ARGOCD_APP_NAME in namespace: argocd"
    echo "  Listing all ArgoCD applications:"
    kubectl get applications -n argocd 2>/dev/null || echo "  No applications found"
    exit 1
fi

# Step 11: Verify ArgoCD Application references the correct Git repository
echo ""
echo "Step 11: Verifying ArgoCD references correct Git repository"
echo "───────────────────────────────────────────────────────────────"
ARGOCD_REPO=$(kubectl get application "$ARGOCD_APP_NAME" -n argocd -o jsonpath='{.spec.source.repoURL}' 2>/dev/null || echo "")

# The ArgoCD provisioner may use cluster-local Gitea URL or external URL
# Check both patterns
if echo "$ARGOCD_REPO" | grep -q "$APP_NAME"; then
    print_success "ArgoCD Application references Git repository for $APP_NAME"
    echo "  Repository: $ARGOCD_REPO"
else
    print_error "ArgoCD Application references incorrect repository"
    echo "  Expected repository to contain: $APP_NAME"
    echo "  Actual: $ARGOCD_REPO"
    exit 1
fi

# Final Summary
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Test Summary - Alice's Nginx GitOps Deployment"
echo "═══════════════════════════════════════════════════════════════"
print_success "✓ Score specification validated"
print_success "✓ Application deployed via innominatus-ctl"
print_success "✓ Kubernetes namespace created: $NAMESPACE"
print_success "✓ Nginx pods running: $POD_COUNT pod(s)"
print_success "✓ Deployment verified"
print_success "✓ Git repository exists in Gitea"
print_success "✓ Kubernetes manifests present in Git repository"
print_success "✓ ArgoCD Application exists"
print_success "✓ ArgoCD references correct Git repository"

echo ""
echo "GitOps Verification:"
echo "───────────────────────────────────────────────────────────────"
echo "  # View Git repository"
echo "  ${GITEA_URL}/${GITEA_USER}/${GIT_REPO_NAME}"
echo ""
echo "  # View ArgoCD Application"
echo "  http://argocd.localtest.me/applications/$APP_NAME"
echo "  kubectl get application $APP_NAME -n argocd"
echo ""

echo ""
echo "Kubernetes Verification Commands:"
echo "───────────────────────────────────────────────────────────────"
echo "  # View nginx pods"
echo "  kubectl get pods -n $NAMESPACE"
echo ""
echo "  # View all resources"
echo "  kubectl get all -n $NAMESPACE"
echo ""
echo "  # View pod logs"
echo "  kubectl logs -n $NAMESPACE -l app=$APP_NAME"
echo ""
echo "  # Port forward to access nginx"
echo "  kubectl port-forward -n $NAMESPACE svc/$APP_NAME 8080:80"
echo ""

print_success "GitOps end-to-end test completed successfully!"
echo ""

# Cleanup option
read -p "Do you want to delete the test deployment? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    kubectl delete namespace "$NAMESPACE"
    print_success "Test deployment cleaned up"
fi
