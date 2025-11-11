# innominatus Examples

This directory contains up-to-date examples demonstrating the current Score specification pattern for innominatus.

## Quick Start

**Deploy an application with a database:**
```bash
./innominatus-ctl deploy examples/score-ecommerce-backend-v1.yaml -w
```

**Add storage to an existing application:**
```bash
./innominatus-ctl deploy examples/score-ecommerce-backend-v2.yaml -w
```

The system automatically detects existing resources and provisions only new ones.

## Current Best Practices

### ✅ Use Score Specifications

Resources are declared in Score specs, not created via CLI commands:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app

containers:
  main:
    image: myapp:latest
    env:
      DATABASE_URL: ${resources.db.connection_string}
      S3_ENDPOINT: ${resources.storage.endpoint}

resources:
  db:
    type: postgres
    properties:  # Use 'properties', not 'params'
      version: "15"
      size: "medium"

  storage:
    type: s3
    properties:
      size: "standard"
      versioning: true
```

### ✅ Use Variable Substitution

Never hardcode credentials. Use `${resources.name.attribute}` syntax:

```yaml
containers:
  main:
    env:
      # ✅ CORRECT: Automatic credential injection
      DATABASE_URL: ${resources.db.connection_string}
      S3_ACCESS_KEY: ${resources.storage.access_key}

      # ❌ WRONG: Hardcoded credentials
      DATABASE_URL: "postgresql://user:pass@host:5432/db"
```

### ✅ Provider-Based Workflows

Workflows are owned by providers (teams), not defined as standalone files:

```
providers/
  database-team/
    provider.yaml
    workflows/
      provision-postgres.yaml
  storage-team/
    provider.yaml
    workflows/
      provision-s3.yaml
```

Resources in Score specs automatically trigger the correct provider workflow.

### ✅ Use 'properties' Field

Modern Score specs use `properties:`, not `params:`:

```yaml
resources:
  db:
    type: postgres
    properties:  # ✅ CORRECT
      version: "15"
    # params:    # ❌ DEPRECATED
```

## Available Examples

### Basic Score Specifications

- **score-spec.yaml** - Minimal Score spec with single database resource
- **score-spec-k8s.yaml** - Multiple resources (route, postgres, volume)
- **score-spec-web-app.yaml** - Web application with DNS and storage
- **score-test-graph.yaml** - Simple spec for testing graph relationships

### Advanced Examples

- **score-spec-with-product-metadata.yaml** - Product metadata for workflow resolution (postgres, redis, vault-space, route)
- **score-spec-with-vault.yaml** - Vault integration for secrets management
- **score-spec-with-gitea-workflow.yaml** - Basic web application template

### Incremental Resource Addition Pattern

These examples demonstrate adding resources to an existing application:

**Step 1: Deploy v1 (database only)**
```bash
./innominatus-ctl deploy examples/score-ecommerce-backend-v1.yaml -w
```

**Step 2: Deploy v2 (database + S3)**
```bash
./innominatus-ctl deploy examples/score-ecommerce-backend-v2.yaml -w
```

**Output:**
```
ℹ️  Detected existing: db (postgres) - Skipping
🆕 Detected new: storage (s3) - Provisioning
```

**Files:**
- **score-ecommerce-backend-v1.yaml** - Ecommerce app with database
- **score-ecommerce-backend-v2.yaml** - Same app with S3 added
- **score-order-service-v1.yaml** - Order service with database
- **score-order-service-v2.yaml** - Order service with S3 added

### Configuration Files

- **admin-config.yaml** - Provider configuration (Git sources, version pinning)
- **dev-team-app.yaml** - Complete dev team onboarding example
- **goldenpaths.yaml** - Golden path workflow definitions
- **platform.yaml** - Platform-level configuration

### Graph Examples

- **graphs/README.md** - Documentation for graph visualization and querying

## Deployment Commands

### Deploy Score Specification
```bash
./innominatus-ctl deploy score-spec.yaml -w
```

### Execute Golden Path Workflow
```bash
./innominatus-ctl list-goldenpaths
./innominatus-ctl run onboard-dev-team inputs.yaml
```

### Check Resource Status
```bash
./innominatus-ctl list-resources
./innominatus-ctl list-resources --type postgres
```

## Migration from Old Patterns

If you have old Score specs or standalone workflows, see the migration guide:

**[examples/archive/README.md](archive/README.md)**

### Key Changes

1. **Standalone workflows** → Provider workflows in `providers/<team>/workflows/`
2. **`params:` field** → `properties:` field
3. **Hardcoded credentials** → Variable substitution `${resources.name.attribute}`
4. **Embedded workflows** → Separate provider workflows
5. **CLI resource creation** → Resources declared in Score specs

## Resource Types and Providers

innominatus automatically resolves resource types to providers:

| Resource Type | Provider | Example |
|---------------|----------|---------|
| `postgres`, `postgresql` | database-team | PostgreSQL database |
| `s3`, `s3-bucket`, `object-storage` | storage-team | S3/MinIO bucket |
| `namespace`, `kubernetes-namespace` | container-team | Kubernetes namespace |
| `gitea-repo` | container-team | Gitea repository |
| `argocd-app` | container-team | ArgoCD application |
| `vault-space` | vault-team | Vault secrets namespace |
| `gitea-org` | identity-team | Gitea organization |
| `keycloak-group` | identity-team | Keycloak group |

See `providers/*/provider.yaml` for complete capability listings.

## Infrastructure as Code Benefits

Score specifications provide:

- **Declarative**: Define desired state, not imperative steps
- **Version controlled**: Store specs in Git
- **Idempotent**: Deploy same spec multiple times safely
- **GitOps ready**: Integrate with CI/CD pipelines
- **Incremental**: Add/remove resources without redeploy

## Need Help?

- **Full documentation**: See [CLAUDE.md](../CLAUDE.md)
- **Demo playbook**: See [DEMO_PLAYBOOK.md](../DEMO_PLAYBOOK.md)
- **Troubleshooting**: See [docs/TROUBLESHOOTING.md](../docs/TROUBLESHOOTING.md)
- **Provider examples**: See `providers/database-team/`, `providers/storage-team/`
- **Archived patterns**: See [archive/README.md](archive/README.md)

---

**Last Updated:** 2025-11-11
**Pattern:** Provider-based architecture with Score specification
