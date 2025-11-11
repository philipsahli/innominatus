# PostgreSQL Provisioning Demo Guide

**For Tomorrow's Presentation**

## 🎯 Demo Objective

Demonstrate innominatus's **provider-based orchestration** with automatic PostgreSQL database provisioning from a Score specification.

---

## ⚡ Quick Start (5 minutes)

### 1. Build & Start Server

```bash
# Build everything
make build

# Start innominatus server (uses PostgreSQL: idp_orchestrator2)
./innominatus
```

**Server will start on:** http://localhost:8081

### 2. Run Automated Demo

```bash
# In another terminal
./scripts/demo-postgres-provisioning.sh
```

**This script demonstrates:**
- ✅ Provider auto-loading (database-team)
- ✅ Score spec submission
- ✅ Orchestration engine automatic resource detection
- ✅ Workflow execution (provision-postgres-mock)
- ✅ Resource state transitions
- ✅ Mock credential generation

**Expected output:** Resource becomes `active` with connection credentials in ~15-30 seconds

---

## 📋 Manual Demo Steps

### Step 1: Verify Server Health

```bash
curl http://localhost:8081/health
# Should return: {"status":"healthy"}
```

### Step 2: Check Loaded Providers

```bash
curl http://localhost:8081/api/providers | jq '.[] | {name, version, category, capabilities}'
```

**Look for:** `database-team` provider with capabilities: `["postgres", "postgresql", "postgres-mock"]`

### Step 3: Submit Score Spec with Postgres Resource

```bash
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer test" \
  --data-binary @tests/e2e/fixtures/postgres-mock-app.yaml
```

**Score Spec includes:**
```yaml
resources:
  database:
    type: postgres-mock  # Triggers database-team provider
    properties:
      db_name: demo_db
      size: small
      replicas: 2
```

### Step 4: Monitor Resource Provisioning

```bash
# Watch resource state changes
watch -n 2 'curl -s http://localhost:8081/api/resources | jq ".[] | {id, resource_type, state}"'
```

**State transitions:**
1. `requested` - Resource created from Score spec
2. `provisioning` - Orchestration engine picked up resource, workflow executing
3. `active` - Workflow completed, database provisioned

### Step 5: View Credentials

```bash
curl -s http://localhost:8081/api/resources | \
  jq '.[] | select(.resource_type=="postgres-mock") | .provider_metadata.outputs'
```

**Outputs include:**
- `connection_string` - Full PostgreSQL connection URL
- `username` - Database user
- `password` - Generated password
- `host`, `port`, `database_name` - Connection details

### Step 6: Check Workflow Execution

```bash
curl http://localhost:8081/api/workflows | \
  jq '.[] | select(.workflow_name=="provision-postgres-mock") | {id, status, steps: [.steps[] | {name, status}]}'
```

**Workflow steps executed:**
1. `create-mock-database` (policy) - Simulates database creation
2. `wait-for-ready` (policy) - Simulates readiness wait
3. `generate-credentials` (policy) - Generates mock credentials

---

## 🎨 Demo Talking Points

### 1. Provider Architecture

> "innominatus uses a **provider-based architecture**. The `database-team` provider is automatically loaded from the filesystem and registers its capabilities for handling `postgres` resources."

```bash
# Show provider manifest
cat providers/database-team/provider.yaml | head -20
```

### 2. Event-Driven Orchestration

> "When a Score spec requests a postgres resource, the **orchestration engine** automatically detects it (polls every 5 seconds), resolves the appropriate provider, and executes the provisioning workflow - no manual intervention needed."

```bash
# Show orchestration code (optional)
# internal/orchestration/engine.go:pollPendingResources()
```

### 3. Workflow Automation

> "The `provision-postgres-mock` workflow simulates database provisioning without requiring Kubernetes. For production, we have a real `provision-postgres` workflow that creates a Zalando PostgreSQL Operator CR."

```bash
# Show mock workflow
cat providers/database-team/workflows/provision-postgres-mock.yaml
```

### 4. CRUD Support

> "The provider supports full CRUD lifecycle: **create**, **update**, **delete** operations with separate workflows for each."

```bash
# Show available workflows
ls -1 providers/database-team/workflows/
# provision-postgres.yaml (CREATE)
# update-postgres.yaml (UPDATE)
# delete-postgres.yaml (DELETE)
```

### 5. Kubernetes Integration (if time permits)

> "The **critical fix** completed today: registered the kubernetes step executor, enabling workflows to create Kubernetes Custom Resources via `kubectl apply`."

```bash
# Show kubernetes executor code
# internal/workflow/executor.go:1552-1630
```

---

## 🚀 Advanced Demo (with Kubernetes)

**Requires:** Zalando PostgreSQL Operator installed

### Install Operator

```bash
kubectl apply -f https://raw.githubusercontent.com/zalando/postgres-operator/master/manifests/postgresql.yaml

# Verify operator running
kubectl get pods -n default | grep postgres-operator
```

### Deploy Real Postgres

```bash
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  --data-binary @tests/e2e/fixtures/postgres-real-app.yaml
```

**What happens:**
1. Orchestration engine resolves to `provision-postgres` workflow
2. **Step 1 (kubernetes):** Creates PostgreSQL CR via `kubectl apply`
3. **Step 2 (policy):** Polls CR status until `Running` (max 10 min)
4. **Step 3 (policy):** Extracts credentials from Kubernetes secret
5. Resource becomes `active` with real credentials

### Verify PostgreSQL CR

```bash
# Check CR created
kubectl get postgresql -A

# View CR details
kubectl get postgresql ecommerce-team-ecommerce-db -n ecommerce -o yaml

# Check secret created by operator
kubectl get secret -n ecommerce | grep postgres
```

### Test Connection

```bash
# Get credentials
CONNECTION_STRING=$(curl -s http://localhost:8081/api/resources | \
  jq -r '.[] | select(.resource_type=="postgres") | .provider_metadata.outputs.connection_string')

# Connect from K8s pod
kubectl run -it --rm psql-client --image=postgres:15 --restart=Never -- \
  psql "$CONNECTION_STRING" -c "SELECT version();"
```

---

## 🛠️ Troubleshooting

### Server won't start

```bash
# Check PostgreSQL database
make db-status

# Create database if needed
./scripts/setup-postgres.sh

# Check logs
./innominatus 2>&1 | grep ERROR
```

### Resource stuck in 'provisioning'

```bash
# Get resource ID
RESOURCE_ID=$(curl -s http://localhost:8081/api/resources | jq -r '.[] | select(.state=="provisioning") | .id' | head -1)

# Check workflow execution
curl http://localhost:8081/api/resources/$RESOURCE_ID | jq '.workflow_execution_id'

# Get workflow logs
WORKFLOW_ID=...
curl http://localhost:8081/api/workflows/$WORKFLOW_ID | jq '.error_message, .steps[] | {name, status, error_message}'
```

### Provider not loaded

```bash
# Check admin-config.yaml
cat admin-config.yaml | grep -A 10 database-team

# Check provider manifest
cat providers/database-team/provider.yaml

# Restart server to reload providers
pkill innominatus
./innominatus
```

---

## 📊 Demo Success Metrics

### Must Have (Core Demo)
- ✅ Server starts without errors
- ✅ database-team provider loads
- ✅ postgres-mock resource provisions
- ✅ Resource state: requested → active
- ✅ Credentials generated in outputs

### Nice to Have (Extended Demo)
- ✅ Real postgres with Kubernetes
- ✅ PostgreSQL CR created and Running
- ✅ Connection test succeeds
- ✅ Web UI visualization

---

## 🎬 Demo Script Timeline

| Time | Step | Duration |
|------|------|----------|
| 0:00 | Intro: Provider architecture overview | 2 min |
| 0:02 | Show `database-team` provider loaded | 1 min |
| 0:03 | Submit Score spec with postgres-mock | 1 min |
| 0:04 | Monitor orchestration engine detecting resource | 2 min |
| 0:06 | Show workflow execution progress | 2 min |
| 0:08 | Resource becomes active, show credentials | 2 min |
| 0:10 | (Optional) Real postgres with K8s | 5 min |
| 0:15 | Q&A | 5 min |

**Total: 10-20 minutes**

---

## 📝 Key Files for Demo

### Show in IDE/Editor
- `providers/database-team/provider.yaml` - Provider manifest
- `providers/database-team/workflows/provision-postgres-mock.yaml` - Mock workflow
- `providers/database-team/workflows/provision-postgres.yaml` - Real K8s workflow
- `tests/e2e/fixtures/postgres-mock-app.yaml` - Example Score spec
- `internal/workflow/executor.go` (lines 1552-1630) - Kubernetes executor (today's fix)

### API Endpoints to Demo
- `GET /api/providers` - Show loaded providers
- `POST /api/specs` - Submit Score spec
- `GET /api/resources` - Monitor resource state
- `GET /api/workflows` - View workflow executions
- `GET /health` - Server health

---

## ✅ Pre-Demo Checklist

- [ ] PostgreSQL database running (`make db-status`)
- [ ] innominatus server built (`make build`)
- [ ] database-team provider loads (check startup logs)
- [ ] Test fixtures exist (`tests/e2e/fixtures/*.yaml`)
- [ ] Demo script tested (`./scripts/demo-postgres-provisioning.sh`)
- [ ] (Optional) Zalando operator installed for K8s demo
- [ ] Browser ready for Web UI: http://localhost:8081

---

## 💡 Backup Plan (if demo fails)

1. **Show existing test**: Point to comprehensive test plan in `tests/e2e/POSTGRES_TEST_PLAN.md`
2. **Walk through code**: Show kubernetes executor implementation
3. **Provider manifest**: Explain CRUD workflows and capabilities
4. **Mock workflow**: Show bash script simulation
5. **Architecture diagram**: Draw orchestration flow on whiteboard

---

**Good luck with tomorrow's demo! 🚀**
