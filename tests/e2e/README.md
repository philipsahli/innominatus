# End-to-End Integration Tests

This directory contains comprehensive integration tests for innominatus that validate complete workflows against real infrastructure.

## Test Suites

### 1. Container GitOps Test (`container_gitops_test.go`)

**Purpose:** Validates the complete GitOps deployment flow for containerized applications using the container-team provider.

**What it tests:**
1. Score spec submission (nginx container)
2. Orchestration engine automatic resource detection
3. Provider resolution (container → container-team)
4. Git repository creation in Gitea
5. Kubernetes manifest generation and commit
6. ArgoCD application provisioning
7. Pod rollout and health verification
8. Complete deployment output collection

**Flow diagram:**
```
Score Spec (nginx)
    → Resource Created (state: requested)
    → Orchestration Engine Detects
    → Resolver: container → container-team provider
    → Workflow Execution:
        Step 1: Create K8s Namespace
        Step 2: Create Git Repo in Gitea
        Step 3: Generate & Commit Manifests
        Step 4: Create ArgoCD Application
        Step 5: Wait for Sync & Healthy Status
    → Pod Rollout Complete
    → Resource State: active
    → Outputs Collected
```

## Prerequisites

### Required Services

1. **PostgreSQL Database**
   - Used for innominatus state storage
   - Automatically provisioned by test suite using testcontainers

2. **Kubernetes Cluster**
   - Docker Desktop Kubernetes (recommended)
   - Minikube
   - Kind
   - Any accessible K8s cluster
   - Must have `kubectl` configured with cluster access

3. **Gitea** (Git Server)
   ```bash
   # Quick setup using demo-time
   ./innominatus-ctl demo-time

   # Or manually:
   helm repo add gitea-charts https://dl.gitea.io/charts/
   helm install gitea gitea-charts/gitea \
     --namespace git-system \
     --create-namespace \
     --set service.http.type=LoadBalancer
   ```

   - Accessible at `http://gitea.localtest.me` (or set `GITEA_URL`)
   - Default credentials: `admin/admin`
   - Must have organization named `platform` (or set `GITEA_ORG`)

4. **ArgoCD** (GitOps Controller)
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   ```

   - Installed in `argocd` namespace
   - Accessible at `http://argocd.localtest.me` (via demo-time)
   - Get password: `kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d`

### Environment Variables

**Required:**
```bash
# Gitea API token (generate from Gitea UI: Settings → Applications → Generate New Token)
export GITEA_TOKEN="your-gitea-api-token-here"

# Enable E2E tests
export RUN_E2E_TESTS=true
```

**Optional:**
```bash
# Gitea URL (default: http://gitea.localtest.me)
export GITEA_URL="http://gitea.localtest.me"

# Gitea organization (default: platform)
export GITEA_ORG="platform"

# Kubernetes config (default: ~/.kube/config)
export KUBECONFIG="$HOME/.kube/config"
```

## Quick Start

### 1. Install Demo Environment

The easiest way to get all prerequisites is to use the demo installer:

```bash
# Install all required services (Gitea, ArgoCD, etc.)
./innominatus-ctl demo-time

# Check health
./innominatus-ctl demo-status

# Get Gitea admin password (default: admin/admin)
echo "Gitea: http://gitea.localtest.me (admin/admin)"
```

### 2. Generate Gitea API Token

```bash
# 1. Login to Gitea at http://gitea.localtest.me
# 2. Go to Settings → Applications
# 3. Generate New Token (name: "e2e-tests", scopes: all)
# 4. Copy the token

export GITEA_TOKEN="your-token-here"
```

### 3. Create Gitea Organization

```bash
# Via Gitea UI:
# 1. Click "+" in top right
# 2. Click "New Organization"
# 3. Name: "platform"

# Or via API:
curl -X POST "http://gitea.localtest.me/api/v1/orgs" \
  -H "Authorization: token $GITEA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username": "platform"}'
```

### 4. Run Tests

```bash
# Run all E2E tests
export RUN_E2E_TESTS=true
go test -v -tags e2e ./tests/e2e/...

# Run specific test
go test -v -tags e2e -run TestContainerGitOpsTestSuite ./tests/e2e/

# Run with verbose output
go test -v -tags e2e ./tests/e2e/ 2>&1 | tee test-output.log
```

## Test Output

The test provides detailed step-by-step output:

```
========================================
Starting nginx GitOps deployment test
========================================

📝 Step 1: Creating Score specification for nginx container
✓ Score spec created: nginx-test-1234567890

💾 Step 2: Submitting Score spec to database
✓ Application created in database: nginx-test-1234567890

📦 Step 3: Creating resource instances from Score spec
✓ Resource created: app (type: container, state: requested)

🔄 Step 4: Running orchestration engine to detect pending resource
⏳ Waiting for orchestration engine to process resource (30s timeout)...

🔍 Step 5: Verifying resource state transition
Current resource state: provisioning

⏳ Step 6: Waiting for workflow execution to complete
Workflow status: running (step 1/5)
Workflow status: running (step 2/5)
Workflow status: running (step 3/5)
Workflow status: running (step 4/5)
Workflow status: running (step 5/5)
✓ Workflow completed successfully

🔍 Step 7: Verifying Git repository created in Gitea
✓ Git repository exists: platform/nginx-test-1234567890

🔍 Step 8: Verifying Kubernetes manifests in Git repository
✓ Kubernetes manifests found in repo

🔍 Step 9: Verifying Kubernetes namespace created
✓ Namespace exists: e2e-test-1234567890

🔍 Step 10: Verifying ArgoCD application created
✓ ArgoCD application exists: nginx-test-1234567890

⏳ Step 11: Waiting for Pod rollout (3 minutes timeout)
Pod nginx-test-1234567890-abc123 status: Pending
Pod nginx-test-1234567890-abc123 status: Running
✓ Pod nginx-test-1234567890-abc123 is running and ready

🔍 Step 12: Verifying final resource state
✓ Resource state: active

📊 Step 13: Collecting deployment outputs
✓ Deployment outputs:
  - app_name: nginx-test-1234567890
  - namespace: e2e-test-1234567890
  - repo_url: http://gitea.localtest.me/platform/nginx-test-1234567890
  - clone_url: http://gitea.localtest.me/platform/nginx-test-1234567890.git
  - argocd_app: nginx-test-1234567890
  - resource_state: active
  - resource_id: 42
  - workflow_execution_id: 123
  - service_name: nginx-test-1234567890
  - service_type: ClusterIP
  - service_port: 80

========================================
✅ Nginx GitOps deployment test PASSED
========================================
```

## Test Cleanup

The test automatically cleans up resources after completion:

- ✅ Kubernetes namespace (cascade deletes all resources)
- ✅ Git repository in Gitea
- ⚠️  ArgoCD application (manual cleanup recommended)

**Manual cleanup (if needed):**
```bash
# List test namespaces
kubectl get ns | grep e2e-test

# Delete specific test namespace
kubectl delete ns e2e-test-1234567890

# Delete ArgoCD application
argocd app delete nginx-test-1234567890 --cascade

# Delete Git repository via API
curl -X DELETE "http://gitea.localtest.me/api/v1/repos/platform/nginx-test-1234567890" \
  -H "Authorization: token $GITEA_TOKEN"
```

## Troubleshooting

### Test Skipped: "GITEA_TOKEN environment variable not set"

**Solution:**
```bash
export GITEA_TOKEN="your-gitea-token"
export RUN_E2E_TESTS=true
```

### Test Skipped: "Kubernetes config not found"

**Solution:**
```bash
# Check kubectl is configured
kubectl cluster-info

# Set KUBECONFIG if needed
export KUBECONFIG="$HOME/.kube/config"
```

### Workflow Execution Timed Out

**Possible causes:**
1. Gitea not accessible from innominatus server
2. ArgoCD not installed or not accessible
3. GITEA_TOKEN invalid or expired
4. Git repository creation failed

**Debug:**
```bash
# Check Gitea connectivity
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/user

# Check ArgoCD installation
kubectl get pods -n argocd

# Check orchestration engine logs
tail -f /var/log/innominatus/orchestration.log
```

### Pod Not Ready

**Possible causes:**
1. ArgoCD sync failed
2. Image pull error
3. Resource limits too low

**Debug:**
```bash
# Check ArgoCD app status
argocd app get nginx-test-123456

# Check pod status
kubectl get pods -n e2e-test-123456

# Check pod logs
kubectl logs -n e2e-test-123456 <pod-name>

# Describe pod for events
kubectl describe pod -n e2e-test-123456 <pod-name>
```

### Git Repository Not Created

**Possible causes:**
1. Gitea organization doesn't exist
2. GITEA_TOKEN lacks permissions
3. Network connectivity issues

**Debug:**
```bash
# Verify organization exists
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/orgs/platform

# Test repository creation manually
curl -X POST "http://gitea.localtest.me/api/v1/orgs/platform/repos" \
  -H "Authorization: token $GITEA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-repo", "auto_init": true}'
```

## CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Set up Kind cluster
        uses: helm/kind-action@v1.5.0

      - name: Install Gitea
        run: |
          helm repo add gitea-charts https://dl.gitea.io/charts/
          helm install gitea gitea-charts/gitea --wait

      - name: Install ArgoCD
        run: |
          kubectl create namespace argocd
          kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
          kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd

      - name: Run E2E Tests
        env:
          GITEA_TOKEN: ${{ secrets.GITEA_TOKEN }}
          RUN_E2E_TESTS: true
        run: |
          go test -v -tags e2e ./tests/e2e/...
```

## Writing New E2E Tests

### Template

```go
//go:build e2e
// +build e2e

package e2e

import (
	"testing"
	"github.com/stretchr/testify/suite"
)

type MyE2ETestSuite struct {
	suite.Suite
	// Add your test fixtures
}

func (s *MyE2ETestSuite) SetupSuite() {
	// Initialize test infrastructure
}

func (s *MyE2ETestSuite) TearDownSuite() {
	// Cleanup
}

func (s *MyE2ETestSuite) TestMyFeature() {
	// Test implementation
}

func TestMyE2ETestSuite(t *testing.T) {
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		t.Skip("E2E tests disabled")
	}
	suite.Run(t, new(MyE2ETestSuite))
}
```

## Test Coverage

- ✅ Container deployment (nginx)
- ✅ Git repository creation (Gitea)
- ✅ Kubernetes manifest generation
- ✅ ArgoCD application provisioning
- ✅ Pod rollout verification
- ✅ Resource state transitions
- ✅ Workflow execution
- ⏳ Database resource (postgres) - TODO
- ⏳ Multi-resource orchestration - TODO
- ⏳ Resource updates (CRUD) - TODO
- ⏳ Failure scenarios - TODO

## References

- [Container Team Provider](../../providers/container-team/README.md)
- [Orchestration Architecture](../../docs/ORCHESTRATION_ARCHITECTURE.md)
- [Troubleshooting Guide](../../docs/TROUBLESHOOTING.md)
