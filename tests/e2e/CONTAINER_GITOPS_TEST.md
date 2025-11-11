# Container GitOps Integration Test

## Overview

Comprehensive end-to-end integration test for the **container-team provider** that validates the complete GitOps deployment flow for containerized applications.

## Test File

`tests/e2e/container_gitops_test.go`

## What It Tests

### Complete GitOps Workflow

The test validates the entire lifecycle from Score specification submission to running Pod:

```
┌─────────────────────────────────────────────────────────┐
│ 1. Submit Score Spec (nginx:alpine)                    │
│    - type: container                                    │
│    - properties: namespace, image, ports, resources    │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Database - Resource Created                         │
│    - state: requested                                   │
│    - workflow_execution_id: NULL                        │
│    - resource_type: container                           │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Orchestration Engine - Automatic Detection          │
│    - Polls every 5 seconds                              │
│    - Finds pending resource                             │
│    - Triggers workflow execution                        │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Resolver - Provider Matching                        │
│    - container → container-team provider                │
│    - Selects: provision-container workflow              │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Workflow Execution (5 Steps)                        │
│                                                         │
│    Step 1: create-namespace                             │
│    ├─ Creates Kubernetes namespace                     │
│    ├─ Labels: app, team, managed-by                    │
│    └─ Resource quotas applied                           │
│                                                         │
│    Step 2: create-git-repo                              │
│    ├─ Calls Gitea API                                   │
│    ├─ Creates repo in organization                     │
│    ├─ Auto-initializes with README                     │
│    └─ Returns: repo_url, clone_url                      │
│                                                         │
│    Step 3: generate-manifests                           │
│    ├─ Clones Git repository                            │
│    ├─ Generates Deployment manifest                    │
│    ├─ Generates Service manifest                       │
│    ├─ Commits to main branch                           │
│    └─ Returns: commit_sha, manifest_path               │
│                                                         │
│    Step 4: create-argocd-app                            │
│    ├─ Creates ArgoCD Application CR                    │
│    ├─ Points to Git repo manifests/                    │
│    ├─ Enables auto-sync & self-heal                    │
│    └─ Sets sync policy: automated                       │
│                                                         │
│    Step 5: wait-for-sync                                │
│    ├─ Polls ArgoCD app status (every 10s)             │
│    ├─ Waits for sync status: Synced                    │
│    ├─ Waits for health status: Healthy                 │
│    └─ Timeout: 10 minutes                               │
│                                                         │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Verification - Infrastructure Created               │
│    ✓ Git repository exists in Gitea                    │
│    ✓ Kubernetes manifests in repo (deployment.yaml)    │
│    ✓ Namespace exists in Kubernetes                    │
│    ✓ ArgoCD application exists                         │
│    ✓ Pod is Running and Ready                          │
│    ✓ Service exists                                    │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│ 7. Final State - Resource Active                       │
│    - state: active                                      │
│    - workflow_execution_id: set                         │
│    - Outputs collected and returned                     │
└─────────────────────────────────────────────────────────┘
```

## Test Assertions

### 1. Score Spec Processing
- ✅ Application created in database
- ✅ Resource instance created with state='requested'
- ✅ Resource type correctly identified as 'container'

### 2. Orchestration Engine
- ✅ Pending resource detected automatically
- ✅ Resource transitions from 'requested' to 'provisioning'
- ✅ Workflow execution initiated

### 3. Provider Resolution
- ✅ Resolver matches 'container' → container-team provider
- ✅ Correct workflow selected: provision-container

### 4. Gitea Integration
- ✅ Git repository created in specified organization
- ✅ Repository accessible via API
- ✅ Kubernetes manifests exist in repo:
  - `manifests/deployment.yaml`
  - `manifests/service.yaml`

### 5. Kubernetes Integration
- ✅ Namespace created with correct labels
- ✅ Namespace accessible via kubectl
- ✅ Resource quotas applied

### 6. ArgoCD Integration
- ✅ ArgoCD Application CR created
- ✅ Application points to correct Git repository
- ✅ Sync policy set to automated
- ✅ Application reaches Synced status
- ✅ Application reaches Healthy status

### 7. Pod Rollout
- ✅ Pod created in correct namespace
- ✅ Pod reaches Running phase
- ✅ All containers ready
- ✅ Pod labeled correctly (app=<app-name>)

### 8. Service Creation
- ✅ Service created by ArgoCD
- ✅ Service type matches specification
- ✅ Service ports configured correctly

### 9. Final Resource State
- ✅ Resource state transitions to 'active'
- ✅ Resource linked to workflow execution
- ✅ No error messages

### 10. Outputs
- ✅ All deployment outputs collected:
  - app_name
  - namespace
  - repo_url
  - clone_url
  - argocd_app
  - resource_state
  - service_name
  - service_port

## Running the Test

### Prerequisites

1. **Kubernetes Cluster**
   ```bash
   kubectl cluster-info
   ```

2. **Gitea** (running at http://gitea.localtest.me)
   ```bash
   # Via demo-time
   ./innominatus-ctl demo-time

   # Or manually
   helm install gitea gitea-charts/gitea
   ```

3. **ArgoCD** (installed in 'argocd' namespace)
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   ```

4. **Gitea API Token**
   ```bash
   # Generate at: http://gitea.localtest.me/user/settings/applications
   export GITEA_TOKEN="your-token-here"
   ```

5. **Gitea Organization**
   ```bash
   curl -X POST "http://gitea.localtest.me/api/v1/orgs" \
     -H "Authorization: token $GITEA_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"username": "platform"}'
   ```

### Run Commands

```bash
# Using Makefile
make test-e2e-gitops

# Using script
./scripts/run-e2e-tests.sh

# Direct Go test
export RUN_E2E_TESTS=true
export GITEA_TOKEN="your-token"
go test -v -tags e2e -timeout 30m ./tests/e2e/... -run TestContainerGitOpsTestSuite

# Run with verbose output
go test -v -tags e2e -timeout 30m ./tests/e2e/... -run TestContainerGitOpsTestSuite 2>&1 | tee test.log
```

## Test Duration

**Expected duration:** 3-5 minutes

**Breakdown:**
- Database setup: 5-10 seconds
- Provider loading: 2-3 seconds
- Score spec submission: < 1 second
- Orchestration engine detection: 5-15 seconds
- Workflow execution:
  - Namespace creation: 2-5 seconds
  - Git repo creation: 3-5 seconds
  - Manifest generation: 5-10 seconds
  - ArgoCD app creation: 2-5 seconds
  - Wait for sync: 30-120 seconds
- Pod rollout: 20-60 seconds
- Verification: 5-10 seconds
- Cleanup: 10-20 seconds

**Timeout:** 30 minutes (to handle slow environments)

## Expected Output

```
========================================
Starting nginx GitOps deployment test
========================================

📝 Step 1: Creating Score specification for nginx container
✓ Score spec created: nginx-test-1730380000

💾 Step 2: Submitting Score spec to database
✓ Application created in database: nginx-test-1730380000

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
✓ Git repository exists: platform/nginx-test-1730380000

🔍 Step 8: Verifying Kubernetes manifests in Git repository
✓ Kubernetes manifests found in repo

🔍 Step 9: Verifying Kubernetes namespace created
✓ Namespace exists: e2e-test-1730380000

🔍 Step 10: Verifying ArgoCD application created
✓ ArgoCD application exists: nginx-test-1730380000

⏳ Step 11: Waiting for Pod rollout (3 minutes timeout)
Pod nginx-test-1730380000-7d4b8c9f-abc12 status: Pending
Pod nginx-test-1730380000-7d4b8c9f-abc12 status: Running
✓ Pod nginx-test-1730380000-7d4b8c9f-abc12 is running and ready

🔍 Step 12: Verifying final resource state
✓ Resource state: active

📊 Step 13: Collecting deployment outputs
✓ Deployment outputs:
  - app_name: nginx-test-1730380000
  - namespace: e2e-test-1730380000
  - repo_url: http://gitea.localtest.me/platform/nginx-test-1730380000
  - clone_url: http://gitea.localtest.me/platform/nginx-test-1730380000.git
  - argocd_app: nginx-test-1730380000
  - resource_state: active
  - resource_id: 42
  - workflow_execution_id: 123
  - service_name: nginx-test-1730380000
  - service_type: ClusterIP
  - service_port: 80

========================================
✅ Nginx GitOps deployment test PASSED
========================================

🧹 Cleaning up test resources...
✓ Deleted namespace: e2e-test-1730380000
✓ Deleted Git repository: platform/nginx-test-1730380000
⚠️  Manual cleanup required: Delete ArgoCD application 'nginx-test-1730380000' in namespace 'argocd'
✓ Cleanup completed
```

## Cleanup

**Automatic:**
- ✅ Kubernetes namespace (cascade deletes all resources)
- ✅ Git repository in Gitea

**Manual (if test fails):**
```bash
# List test resources
kubectl get ns | grep e2e-test

# Delete namespace
kubectl delete ns e2e-test-1730380000

# Delete ArgoCD app
argocd app delete nginx-test-1730380000 --cascade

# Delete Git repo
curl -X DELETE "http://gitea.localtest.me/api/v1/repos/platform/nginx-test-1730380000" \
  -H "Authorization: token $GITEA_TOKEN"
```

## Troubleshooting

### Test Skipped

**Problem:** Test skipped with "GITEA_TOKEN environment variable not set"

**Solution:**
```bash
export GITEA_TOKEN="your-token-here"
export RUN_E2E_TESTS=true
```

### Workflow Timeout

**Problem:** Workflow execution times out after 5 minutes

**Possible causes:**
1. Gitea not accessible from cluster
2. ArgoCD not installed or unhealthy
3. GITEA_TOKEN invalid

**Debug:**
```bash
# Check Gitea connectivity
curl -H "Authorization: token $GITEA_TOKEN" http://gitea.localtest.me/api/v1/user

# Check ArgoCD
kubectl get pods -n argocd

# Check workflow logs
kubectl logs -n innominatus-system deployment/innominatus | grep orchestration
```

### Pod Not Ready

**Problem:** Pod never reaches Ready state

**Possible causes:**
1. Image pull error
2. ArgoCD sync failed
3. Insufficient resources

**Debug:**
```bash
# Check ArgoCD app
argocd app get nginx-test-1730380000

# Check pod status
kubectl get pods -n e2e-test-1730380000

# Check pod events
kubectl describe pod -n e2e-test-1730380000 <pod-name>

# Check pod logs
kubectl logs -n e2e-test-1730380000 <pod-name>
```

### Manifest Not Found

**Problem:** Kubernetes manifests not found in Git repository

**Possible causes:**
1. Git credentials invalid
2. Manifest generation step failed
3. Git push failed

**Debug:**
```bash
# Check repo contents
curl -H "Authorization: token $GITEA_TOKEN" \
  "http://gitea.localtest.me/api/v1/repos/platform/nginx-test-123/contents/manifests"

# Clone repo manually
git clone http://gitea.localtest.me/platform/nginx-test-123.git
cd nginx-test-123
ls -la manifests/
```

## Future Enhancements

- [ ] Add test for multi-container applications
- [ ] Add test for container with database dependency
- [ ] Add test for ConfigMap/Secret injection
- [ ] Add test for Ingress creation
- [ ] Add test for resource updates (scale up/down)
- [ ] Add test for resource deletion (cleanup workflow)
- [ ] Add test for failure scenarios (Gitea down, ArgoCD unhealthy)
- [ ] Add test for concurrent deployments
- [ ] Add performance benchmarks

## Related Documentation

- [Container Team Provider](../../providers/container-team/README.md)
- [Orchestration Architecture](../../docs/ORCHESTRATION_ARCHITECTURE.md)
- [E2E Test Guide](README.md)
- [Troubleshooting](../../docs/TROUBLESHOOTING.md)
