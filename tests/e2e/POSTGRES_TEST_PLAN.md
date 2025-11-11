# PostgreSQL Provisioning Test Plan

## ✅ CRITICAL FIX COMPLETED

**Fixed:** Kubernetes step executor registration in `internal/workflow/executor.go`

- **Line 1552-1630:** Registered `kubernetes` executor in `registerDefaultStepExecutors()`
- **Line 1882-1977:** Implemented helper methods:
  - `kubernetesCreateNamespace()` - Creates K8s namespaces
  - `kubernetesApply()` - Applies manifests via kubectl
  - `kubernetesDelete()` - Deletes resources
  - `kubernetesGet()` - Retrieves resource info
  - `renderTemplate()` - Go template rendering for manifests

**Impact:** PostgreSQL provision-postgres workflow will now execute kubernetes steps correctly.

---

## Quick Test: Manual Postgres Mock Provisioning

### Prerequisites
```bash
# Ensure PostgreSQL is running
make db-status

# Build innominatus
make build
```

###Minimal Test (No Kubernetes)

```bash
# 1. Start innominatus server
./innominatus

# 2. In another terminal, submit a Score spec with postgres-mock
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer test" \
  --data-binary @- <<EOF
apiVersion: score.dev/v1b1
metadata:
  name: test-postgres-mock-app
containers:
  main:
    image: nginx:latest
resources:
  db:
    type: postgres-mock
    properties:
      db_name: testdb
      namespace: test-ns
      team_id: test-team
      size: small
      replicas: 2
      version: "15"
EOF

# 3. Monitor resource provisioning
curl http://localhost:8081/api/resources | jq '.[] | select(.resource_type=="postgres-mock")'

# 4. Check workflow execution
curl http://localhost:8081/api/workflows | jq '.[] | select(.workflow_name=="provision-postgres-mock")'

# 5. Verify resource becomes active (wait ~10 seconds)
watch -n 2 'curl -s http://localhost:8081/api/resources | jq ".[] | select(.resource_type==\"postgres-mock\") | {state, outputs}"'
```

**Expected Result:**
- Resource state: `requested` → `provisioning` → `active`
- Outputs contain: `connection_string`, `username`, `password`, `database_name`

---

## Test Plan Phases

### Phase 1: Mock Testing (No Kubernetes) - 2 hours

**Goal:** Validate orchestration engine without K8s dependencies

#### Test 1.1: Provider Resolution
```bash
# Verify database-team provider is loaded
curl http://localhost:8081/api/providers | jq '.[] | select(.name=="database-team")'

# Expected capabilities: postgres, postgresql, postgres-mock
```

#### Test 1.2: Mock Workflow Execution
- Use example above
- Verify complete flow: Score spec → resource → workflow → outputs
- Check workflow logs: `GET /api/workflows/{id}/logs`

#### Test 1.3: State Transitions
- Monitor resource state changes via API
- Verify no orphaned resources (state=provisioning forever)

#### Test 1.4: Error Scenarios
- Submit invalid resource type → verify graceful failure
- Missing required parameters → verify error message

---

### Phase 2: Integration Testing (Kubernetes Required) - 4 hours

**Prerequisites:**
```bash
# Install Zalando PostgreSQL Operator
kubectl apply -f https://raw.githubusercontent.com/zalando/postgres-operator/master/manifests/postgresql.yaml

# Verify operator is running
kubectl get pods -n default | grep postgres-operator
```

#### Test 2.1: Real Postgres Provisioning
```bash
# Submit Score spec with type: postgres
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  --data-binary @providers/database-team/examples/score-postgres-app.yaml

# Monitor PostgreSQL CR creation
kubectl get postgresql -A --watch

# Expected: CR created, cluster becomes Running, resource becomes active
```

#### Test 2.2: Workflow Step Validation
```bash
# Get workflow execution details
WORKFLOW_ID=$(curl -s http://localhost:8081/api/workflows | jq -r '.[0].id')

# Check each step completed
curl http://localhost:8081/api/workflows/$WORKFLOW_ID | jq '.steps[] | {name, status, error_message}'

# Expected steps:
# 1. create-postgres-cluster (kubernetes) - SUCCESS
# 2. wait-for-database (policy) - SUCCESS (max 10 min)
# 3. get-credentials (policy) - SUCCESS
```

#### Test 2.3: Credentials Validation
```bash
# Get resource outputs
RESOURCE_ID=$(curl -s http://localhost:8081/api/resources | jq -r '.[] | select(.resource_type=="postgres") | .id')

curl http://localhost:8081/api/resources/$RESOURCE_ID | jq '.provider_metadata.outputs'

# Extract connection info
CONNECTION_STRING=$(curl -s http://localhost:8081/api/resources/$RESOURCE_ID | jq -r '.provider_metadata.outputs.connection_string')

# Test connection from within K8s
kubectl run -it --rm psql-test --image=postgres:15 --restart=Never -- \
  psql "$CONNECTION_STRING" -c "SELECT version();"
```

#### Test 2.4: Size Tiers
```bash
# Test small (5Gi, 100m-500m CPU)
# Test medium (20Gi, 500m-2000m CPU)
# Test large (100Gi, 2000m-4000m CPU)

# Verify PostgreSQL CR resource specs
kubectl get postgresql test-team-testdb -o yaml | yq '.spec.resources'
```

#### Test 2.5: Failure Scenarios
- **Missing namespace:** Submit with namespace="nonexistent" → verify error
- **Operator not running:** Stop operator → verify timeout
- **Invalid version:** Submit version="999" → verify validation error

---

### Phase 3: CRUD Lifecycle - 3 hours

#### Test 3.1: Create → Update → Delete
```bash
# 1. CREATE (size=small, replicas=2)
curl -X POST http://localhost:8081/api/specs --data-binary @create-spec.yaml

# Wait for active state

# 2. UPDATE (size=medium, replicas=3)
RESOURCE_ID=$(...)
curl -X PATCH http://localhost:8081/api/resources/$RESOURCE_ID \
  -H "Content-Type: application/json" \
  -d '{"desired_operation": "update", "configuration": {"size": "medium", "replicas": 3}}'

# Verify: update-postgres workflow executes, pods scale 2→3

# 3. DELETE
curl -X DELETE http://localhost:8081/api/applications/test-app

# Verify: delete-postgres workflow executes, CR removed
```

---

## Test Fixtures

### Example Score Specs

**File:** `tests/e2e/fixtures/postgres-mock-app.yaml`
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: test-postgres-mock
containers:
  app:
    image: nginx:latest
resources:
  database:
    type: postgres-mock
    properties:
      db_name: myapp
      namespace: test-namespace
      team_id: platform-team
      size: small
      version: "15"
```

**File:** `tests/e2e/fixtures/postgres-real-app.yaml`
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: ecommerce-backend
containers:
  backend:
    image: ecommerce-api:latest
    env:
      DATABASE_URL: ${resources.database.connection_string}
      DB_USER: ${resources.database.username}
      DB_PASSWORD: ${resources.database.password}
resources:
  database:
    type: postgres
    properties:
      db_name: ecommerce_db
      namespace: ecommerce
      team_id: ecommerce-team
      size: medium
      replicas: 3
      version: "15"
```

---

## Success Criteria

### Minimum Viable
- ✅ Kubernetes executor registered and compiles
- ✅ Server builds successfully
- ⏳ Mock postgres workflow executes (manual test)
- ⏳ Resource state: requested → active
- ⏳ Mock credentials generated

### Production Ready
- Real postgres workflow executes with Zalando operator
- PostgreSQL CR created and becomes Running
- Credentials extracted from K8s secret
- Connection test succeeds
- CRUD operations validated

---

## Troubleshooting

### Resource stuck in 'provisioning'
```bash
# Check workflow execution
RESOURCE_ID=...
curl http://localhost:8081/api/resources/$RESOURCE_ID | jq '.workflow_execution_id'

WORKFLOW_ID=...
curl http://localhost:8081/api/workflows/$WORKFLOW_ID | jq '.status, .error_message'

# Check logs
curl http://localhost:8081/api/workflows/$WORKFLOW_ID/logs
```

### Workflow fails at step 1 (create-postgres-cluster)
```bash
# Verify kubectl works
kubectl get nodes

# Verify namespace exists
kubectl get namespace test-namespace

# Check RBAC permissions
kubectl auth can-i create postgresql --as=system:serviceaccount:default:default
```

### Step 2 times out (wait-for-database)
```bash
# Check PostgreSQL CR status
kubectl get postgresql -A

# Describe CR for events
kubectl describe postgresql test-team-testdb -n test-namespace

# Check operator logs
kubectl logs -n default -l name=postgres-operator
```

---

## Next Steps

1. **Run manual mock test** (10 minutes) - Validates orchestration engine works
2. **Install Zalando operator locally** (15 minutes) - Enables real postgres testing
3. **Run integration test** (20 minutes) - Full E2E validation
4. **Implement automated tests** (4 hours) - CI/CD integration
5. **Document edge cases** (2 hours) - Production readiness

---

## Key Files Modified

- `internal/workflow/executor.go` - Added kubernetes executor (lines 1552-1630, 1882-1977)
- `tests/e2e/postgres_mock_test.go` - Test template (needs API updates)
- `tests/e2e/POSTGRES_TEST_PLAN.md` - This file

## Provider Files

- `providers/database-team/provider.yaml` - Provider manifest
- `providers/database-team/workflows/provision-postgres.yaml` - Real provisioner
- `providers/database-team/workflows/provision-postgres-mock.yaml` - Mock provisioner
- `providers/database-team/workflows/update-postgres.yaml` - Update workflow
- `providers/database-team/workflows/delete-postgres.yaml` - Delete workflow
- `providers/database-team/examples/score-postgres-app.yaml` - Example Score spec
