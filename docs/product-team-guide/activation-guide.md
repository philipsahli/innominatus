# Product Team Provider Activation Guide

**Audience:** Platform Teams
**Status:** ✅ Available
**Last Updated:** 2025-10-29

---

## Overview

This guide explains how to **activate product team provider capabilities** in your innominatus deployment. Product team providers enable internal teams to offer services (databases, storage, secrets) to application developers through automated workflows.

**What You Get:**
- Git-based provider loading from remote repositories
- Multi-provider architecture with demo providers included
- Golden path workflows for developer onboarding
- PostgreSQL Operator integration for database provisioning
- Vault Secrets Operator (VSO) for secret synchronization
- Complete demo environment for testing

---

## Prerequisites

Before activation, ensure:

✅ **innominatus server** installed and running
✅ **Docker Desktop with Kubernetes** enabled (for demo environment)
✅ **Database connection** working (PostgreSQL or SQLite)
✅ **kubectl** access to Kubernetes cluster (for demo)
✅ **Admin permissions** for server configuration

---

## Activation Options

### Option 1: Demo Environment (Recommended for Testing)

The fastest way to explore provider capabilities:

```bash
# Install complete demo environment
./innominatus-ctl demo-time

# This automatically installs:
# - Gitea (Git repository)
# - Vault (Secret management)
# - MinIO (Object storage)
# - PostgreSQL Operator (Database provisioning)
# - Vault Secrets Operator (Secret synchronization)
# - ArgoCD (GitOps deployment)
# - 4 demo providers (database, storage, vault, container teams)

# Check status
./innominatus-ctl demo-status

# Expected output:
# ✅ Gitea: http://gitea.localtest.me
# ✅ Vault: http://vault.localtest.me
# ✅ MinIO: http://minio.localtest.me
# ✅ ArgoCD: http://argocd.localtest.me
# ✅ PostgreSQL Operator: Running
# ✅ Vault Secrets Operator: Running
```

**Demo Providers Location:** `providers/`
- `database-team/` - PostgreSQL via operator
- `storage-team/` - MinIO buckets
- `vault-team/` - Secret management
- `container-team/` - Container registry

**Try the onboarding workflow:**
```bash
./innominatus-ctl run onboard-dev-team examples/dev-team-app.yaml
```

### Option 2: Local Provider Setup (Production)

Set up provider structure for your organization:

#### Step 1: Create Provider Directory Structure

```bash
# Create provider directories
mkdir -p providers/database-team/workflows
mkdir -p providers/storage-team/workflows
mkdir -p providers/vault-team/workflows

# Directory structure:
# providers/
# ├── database-team/
# │   ├── provider.yaml
# │   └── workflows/
# │       ├── create-postgres.yaml
# │       └── delete-postgres.yaml
# ├── storage-team/
# │   ├── provider.yaml
# │   └── workflows/
# │       ├── create-bucket.yaml
# │       └── delete-bucket.yaml
# └── vault-team/
#     ├── provider.yaml
#     └── workflows/
#         ├── create-secrets.yaml
#         └── sync-secrets.yaml
```

#### Step 2: Create Provider Metadata

**Example:** `providers/database-team/provider.yaml`

```yaml
name: database-team
description: PostgreSQL database provisioning via operator
version: 1.0.0
owner: database-team@company.com
workflows_dir: workflows
supported_resources:
  - postgres
  - postgresql
tags:
  - database
  - postgres
  - operator
```

#### Step 3: Create Provider Workflows

**Example:** `providers/database-team/workflows/create-postgres.yaml`

```yaml
name: create-postgres
description: Create PostgreSQL database via operator
steps:
  - name: create-postgres-cr
    type: kubernetes
    config:
      manifest: |
        apiVersion: postgresql.cnpg.io/v1
        kind: Cluster
        metadata:
          name: ${APP_NAME}-db
          namespace: ${NAMESPACE}
        spec:
          instances: 1
          storage:
            size: 1Gi

  - name: wait-for-ready
    type: shell
    config:
      command: |
        kubectl wait --for=condition=Ready \
          cluster/${APP_NAME}-db \
          -n ${NAMESPACE} \
          --timeout=300s

  - name: get-credentials
    type: shell
    config:
      command: |
        kubectl get secret ${APP_NAME}-db-app \
          -n ${NAMESPACE} \
          -o json
```

#### Step 4: Start Server with Providers

```bash
# Server automatically loads providers from ./providers/ directory
./innominatus

# Or build and run:
make build
./innominatus

# Expected output:
# Loading providers from: ./providers
# ✅ Loaded provider: database-team (3 workflows)
# ✅ Loaded provider: storage-team (2 workflows)
# ✅ Loaded provider: vault-team (2 workflows)
# Server listening on :8081
```

### Option 3: Git-Based Provider Loading

Load providers from remote Git repositories:

#### Step 1: Prepare Git Repository

```bash
# Structure in your Git repository:
# providers/
# ├── database-team/
# │   ├── provider.yaml
# │   └── workflows/
# └── storage-team/
#     ├── provider.yaml
#     └── workflows/

git add providers/
git commit -m "Add product team providers"
git push origin main
```

#### Step 2: Register Provider via Admin API

```bash
# Register database-team provider from Git
curl -X POST http://localhost:8081/api/admin/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "database-team",
    "git_url": "https://github.com/yourorg/providers",
    "path": "providers/database-team",
    "ref": "main"
  }'

# Response:
# {
#   "message": "Provider registered successfully",
#   "provider": {
#     "name": "database-team",
#     "version": "1.0.0",
#     "workflows": 3
#   }
# }
```

#### Step 3: Verify Provider Registration

```bash
# List registered providers
curl http://localhost:8081/api/providers \
  -H "Authorization: Bearer $API_TOKEN" | jq

# Expected output:
# [
#   {
#     "name": "database-team",
#     "description": "PostgreSQL database provisioning",
#     "version": "1.0.0",
#     "workflows": ["create-postgres", "delete-postgres"],
#     "source": "git"
#   }
# ]
```

#### Step 4: Update Provider from Git

```bash
# Provider automatically reloads on server restart
# Or use API to reload:
curl -X POST http://localhost:8081/api/admin/providers/database-team/reload \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Onboarding Product Teams

### Process

1. **Product team creates provider structure**
   - Location: `providers/{team-name}/`
   - Files: `provider.yaml` + `workflows/` directory
   - Follow demo provider examples

2. **Product team develops workflows**
   - Create workflow YAML files in `workflows/` subdirectory
   - Test locally with `innominatus-ctl run`
   - Use demo environment for integration testing

3. **Platform team reviews**
   - Review provider metadata (`provider.yaml`)
   - Check workflow security (no hardcoded secrets)
   - Test workflows in staging environment
   - Verify resource cleanup

4. **Deployment options**
   - **Option A (Git):** Register provider via admin API with Git URL
   - **Option B (Local):** Add provider to `providers/` directory and restart server
   - **Option C (Both):** Use both for development/production separation

### Review Checklist

When reviewing product team providers:

- [ ] Provider structure: `providers/{team-name}/provider.yaml` + `workflows/`
- [ ] Valid YAML structure in all files
- [ ] Provider metadata complete (name, version, owner, description)
- [ ] Owner contact information provided
- [ ] Workflows use supported step types (kubernetes, shell, terraform, ansible)
- [ ] No hardcoded secrets (use Vault or Kubernetes secrets)
- [ ] Resource names use variables (`${APP_NAME}`, `${NAMESPACE}`)
- [ ] Cleanup/deletion workflows provided
- [ ] Tested in demo or staging environment
- [ ] Documentation for app developers included

---

## Monitoring

### Health Endpoints

Monitor innominatus server health:

```bash
# Health check
curl http://localhost:8081/health

# Readiness probe
curl http://localhost:8081/ready

# Prometheus metrics
curl http://localhost:8081/metrics
```

### Key Metrics

Monitor these after activation:

```promql
# Total workflow executions
sum(innominatus_workflows_total)

# Workflow success rate
sum(innominatus_workflows_success_total) / sum(innominatus_workflows_total)

# Provider workflow duration
histogram_quantile(0.95, innominatus_provider_workflow_duration_seconds)

# Demo environment health
sum(innominatus_demo_components_healthy) / sum(innominatus_demo_components_total)
```

### Logs to Watch

```bash
# Provider loading
tail -f innominatus.log | grep "provider"

# Workflow execution
tail -f innominatus.log | grep "workflow"

# Demo environment status
./innominatus-ctl demo-status

# PostgreSQL Operator
kubectl logs -n postgres-operator -l app=postgres-operator

# Vault Secrets Operator
kubectl logs -n vault-secrets-operator -l app=vault-secrets-operator
```

### Alerts to Set Up

```yaml
# Prometheus alerts
groups:
  - name: innominatus_providers
    rules:
      - alert: ProviderWorkflowFailureRate
        expr: sum by (provider) (rate(innominatus_provider_workflows_failed_total[5m])) > 0.1
        annotations:
          summary: "Provider {{ $labels.provider }} workflow failure rate >10%"

      - alert: PostgresOperatorDown
        expr: up{job="postgres-operator"} == 0
        annotations:
          summary: "PostgreSQL Operator is down - database provisioning unavailable"

      - alert: VaultSecretsOperatorDown
        expr: up{job="vault-secrets-operator"} == 0
        annotations:
          summary: "Vault Secrets Operator is down - secret sync failing"
```

---

## Troubleshooting

### Issue: Providers Not Loading

**Symptom:** Server starts but providers aren't listed

**Debug Steps:**

1. **Check provider directory structure:**
   ```bash
   ls -la providers/
   # Should show: database-team/, storage-team/, etc.
   ```

2. **Verify provider.yaml files:**
   ```bash
   find providers/ -name "provider.yaml" -exec cat {} \;
   ```

3. **Check server logs:**
   ```bash
   tail -f innominatus.log | grep -i "provider"
   # Look for: "Loaded provider: database-team"
   ```

4. **List providers via API:**
   ```bash
   curl http://localhost:8081/api/providers | jq
   ```

### Issue: Demo Environment Fails to Install

**Symptom:** `demo-time` command fails or components unhealthy

**Debug Steps:**

1. **Check Kubernetes is running:**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

2. **Check namespace:**
   ```bash
   kubectl get ns | grep demo
   ```

3. **Check pod status:**
   ```bash
   kubectl get pods -n demo
   kubectl get pods -n postgres-operator
   kubectl get pods -n vault-secrets-operator
   ```

4. **Check specific component:**
   ```bash
   ./innominatus-ctl demo-status

   # Check logs for specific component:
   kubectl logs -n demo -l app=gitea
   kubectl logs -n demo -l app=vault
   kubectl logs -n demo -l app=minio
   ```

5. **Reinstall demo:**
   ```bash
   ./innominatus-ctl demo-nuke
   ./innominatus-ctl demo-time
   ```

### Issue: PostgreSQL Operator Database Not Provisioning

**Symptom:** Workflow creates PostgreSQL CRD but database doesn't start

**Debug Steps:**

1. **Check PostgreSQL Operator is running:**
   ```bash
   kubectl get pods -n postgres-operator
   ```

2. **Check PostgreSQL cluster status:**
   ```bash
   kubectl get postgresql.cnpg.io -A
   kubectl describe postgresql.cnpg.io <db-name> -n <namespace>
   ```

3. **Check operator logs:**
   ```bash
   kubectl logs -n postgres-operator -l app=postgres-operator --tail=100
   ```

4. **Check PVC creation:**
   ```bash
   kubectl get pvc -n <namespace>
   ```

### Issue: Vault Secrets Not Syncing to Kubernetes

**Symptom:** VSO ExternalSecret created but secret not appearing in namespace

**Debug Steps:**

1. **Check VSO is running:**
   ```bash
   kubectl get pods -n vault-secrets-operator
   ```

2. **Check ExternalSecret status:**
   ```bash
   kubectl get externalsecrets -A
   kubectl describe externalsecret <name> -n <namespace>
   ```

3. **Check VSO logs:**
   ```bash
   kubectl logs -n vault-secrets-operator -l app=vault-secrets-operator --tail=100
   ```

4. **Verify secret exists in Vault:**
   ```bash
   # Get vault token (demo)
   export VAULT_TOKEN=root
   export VAULT_ADDR=http://vault.localtest.me

   # Check secret path
   vault kv get secret/data/<app-name>
   ```

### Issue: Workflow Execution Fails

**Symptom:** Workflow starts but fails with error

**Debug Steps:**

1. **Check workflow logs:**
   ```bash
   ./innominatus-ctl workflow logs <workflow-id>
   ```

2. **Check workflow detail:**
   ```bash
   ./innominatus-ctl workflow detail <workflow-id>
   ```

3. **Verify variables are set:**
   ```bash
   # Check if APP_NAME, NAMESPACE, etc. are provided
   # Review workflow YAML for required variables
   ```

4. **Test workflow step manually:**
   ```bash
   # For kubernetes steps:
   kubectl apply -f <manifest>

   # For shell steps:
   # Run the command directly to see error output
   ```

---

## Security Best Practices

### Provider Review

**Before deploying provider to production:**

1. **Code review provider workflows**
   - No hardcoded credentials
   - Use Vault or Kubernetes secrets
   - Validate all input parameters
   - Implement error handling

2. **Test in isolated environment**
   - Use demo environment first
   - Test with non-production clusters
   - Verify cleanup/deletion workflows

3. **Restrict access**
   - Limit who can register providers
   - Require admin approval for Git-based providers
   - Monitor provider registration via API logs

### Secret Management

**Best practices for secrets in workflows:**

```yaml
# ❌ BAD - Hardcoded secret
steps:
  - name: deploy
    config:
      password: "mysecretpassword"

# ✅ GOOD - Reference from Vault
steps:
  - name: get-secret
    type: shell
    config:
      command: |
        vault kv get -field=password secret/${APP_NAME}

# ✅ GOOD - Reference from Kubernetes secret
steps:
  - name: deploy
    config:
      password_from_secret:
        name: ${APP_NAME}-credentials
        key: password
```

### RBAC Considerations

**Production deployment:**

1. **Enable OIDC authentication**
   ```bash
   export OIDC_ENABLED=true
   export OIDC_ISSUER="https://your-idp.com"
   ./innominatus
   ```

2. **Configure team-based access**
   - Admin users can register providers
   - Product teams manage their own providers
   - App developers use providers via workflows

3. **API token management**
   - Rotate API tokens regularly
   - Use short-lived tokens for CI/CD
   - Monitor token usage via logs

---

## Performance Tuning

### Workflow Execution

**Optimize workflow performance:**

1. **Use efficient kubectl commands**
   ```yaml
   # ❌ Slow - Multiple kubectl calls
   - kubectl create namespace ${NS}
   - kubectl create secret ...
   - kubectl apply -f ...

   # ✅ Fast - Single kubectl apply with multiple resources
   - kubectl apply -f - <<EOF
     apiVersion: v1
     kind: Namespace
     ...
     ---
     apiVersion: v1
     kind: Secret
     ...
     EOF
   ```

2. **Parallel step execution (future enhancement)**
   - Currently steps run sequentially
   - Plan for parallel execution in future release

3. **Resource limits**
   - Set appropriate timeout values
   - Monitor workflow duration
   - Optimize long-running workflows

### Demo Environment

**Performance considerations:**

- Demo environment requires ~4GB RAM
- PostgreSQL Operator adds ~500MB
- Vault Secrets Operator adds ~200MB
- Each database instance adds ~1GB

**For production:**
- Use external PostgreSQL (not operator)
- Use external Vault cluster
- Deploy to production Kubernetes cluster

---

## Next Steps

After successful activation:

1. **✅ Core Features Active:** Provider loading, Git integration, demo environment
2. **➡️ Onboard Product Teams:** Work with internal teams to create providers
3. **➡️ Production Deployment:** Deploy to production Kubernetes cluster
4. **➡️ OIDC Integration:** Configure authentication for your organization
5. **➡️ Monitoring Setup:** Configure Prometheus alerts and dashboards

**Future Enhancements:**
- Custom provisioner SDK
- Multi-tenant RBAC
- Runtime policy enforcement
- Workflow templates

---

## Getting Help

**Platform Team Resources:**
- **Product Team Guide:** [README.md](README.md) - Overview and getting started
- **Product Workflows:** [product-workflows.md](product-workflows.md) - Detailed workflow development guide
- **Kubernetes Deployment:** [../platform-team-guide/kubernetes-deployment.md](../platform-team-guide/kubernetes-deployment.md)
- **Main Documentation:** [CLAUDE.md](../../CLAUDE.md) - Core system documentation

**Demo Providers (Examples):**
- `providers/database-team/` - PostgreSQL via operator
- `providers/storage-team/` - MinIO object storage
- `providers/vault-team/` - Secret management with VSO
- `providers/container-team/` - Container registry management

**Support:**
- GitHub Issues: https://github.com/philipsahli/innominatus/issues
- Check demo status: `./innominatus-ctl demo-status`
- View logs: `tail -f innominatus.log`

---

## Success Metrics

Track these to measure activation success:

**Week 1:**
- [ ] Demo environment installed and healthy
- [ ] 2+ product teams exploring demo providers
- [ ] First custom provider created
- [ ] Golden path workflow tested successfully

**Month 1:**
- [ ] 3+ product team providers deployed
- [ ] 10+ workflows running successfully
- [ ] <5% workflow failure rate
- [ ] Product teams self-servicing via CLI

**Quarter 1:**
- [ ] 5+ product teams active
- [ ] Provider catalog established
- [ ] OIDC authentication configured
- [ ] Production deployment complete
- [ ] Product team satisfaction >85%

---

**Questions?** Contact platform team or open a GitHub issue.

**Ready to get started?** Run `./innominatus-ctl demo-time` to explore the demo environment! 🚀
