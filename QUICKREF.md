# innominatus Quick Reference

## Build & Run

```bash
make install              # Install dependencies
make build                # Build server + CLI + web UI
make dev                  # Start server + web UI (http://localhost:8081 & :3000)
make test                 # Run all tests

./innominatus             # Start server (http://localhost:8081)
./innominatus-ctl --help  # CLI help
```

## CLI Commands (31 total)

```bash
# Specs
innominatus-ctl submit score.yaml                    # Deploy Score spec
innominatus-ctl list                                 # List applications
innominatus-ctl get <name>                           # Get app details
innominatus-ctl delete <name>                        # Delete application

# Workflows
innominatus-ctl run <workflow> [inputs.yaml]         # Execute golden path
innominatus-ctl list-goldenpaths                     # Available workflows
innominatus-ctl workflow list                        # List executions
innominatus-ctl workflow detail <id>                 # Execution details
innominatus-ctl workflow logs <id> [--step <name>]   # View logs

# Resources
innominatus-ctl list-resources [--type postgres]     # List resources
innominatus-ctl resource detail <id>                 # Resource details

# Providers
innominatus-ctl list-providers                       # Available providers
innominatus-ctl provider detail <name>               # Provider info

# Demo Environment
innominatus-ctl demo-time                            # Install demo services
innominatus-ctl demo-status                          # Check health
innominatus-ctl demo-nuke                            # Remove demo

# Local Commands (no auth)
innominatus-ctl validate workflow.yaml               # Validate workflow
innominatus-ctl analyze workflow.yaml                # Analyze workflow
```

## API Endpoints

```bash
# Specs
POST   /api/specs                                    # Submit Score spec
GET    /api/specs                                    # List specs
GET    /api/specs/{name}                             # Get spec
DELETE /api/specs/{name}                             # Delete spec

# Workflows
POST   /api/workflows/execute                        # Execute workflow
GET    /api/workflows                                # List executions
GET    /api/workflows/{id}                           # Get execution
GET    /api/workflows/{id}/logs                      # Stream logs (SSE)
GET    /api/workflows/{id}/graph                     # Get workflow graph

# Resources
GET    /api/resources                                # List resources
GET    /api/resources/{id}                           # Get resource
GET    /api/resources/{id}/graph                     # Get resource graph

# Providers
GET    /api/providers                                # List providers
GET    /api/providers/{name}                         # Get provider details
GET    /api/providers/{name}/workflows               # Provider workflows
POST   /api/admin/providers/reload                   # Reload providers (admin)

# System
GET    /health                                       # Health check
GET    /ready                                        # Readiness probe
GET    /metrics                                      # Prometheus metrics
GET    /swagger-user                                 # API docs (user)
GET    /swagger-admin                                # API docs (admin)
```

## Authentication

```bash
# Development (file-based)
export AUTH_TYPE=file
# Uses users.yaml

# Production (OIDC)
export OIDC_ENABLED=true
export OIDC_ISSUER="https://keycloak.example.com/realms/prod"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="secret"

# API Keys (generate via Web UI Profile page)
curl -H "Authorization: Bearer <api-key>" http://localhost:8081/api/specs
```

## Provider Operations

```bash
# Provider manifest (provider.yaml)
apiVersion: v1
kind: Provider
metadata:
  name: my-provider
capabilities:
  resourceTypes: [postgres, mysql]  # Auto-provisioning triggers
workflows:
  - name: provision-postgres
    category: provisioner             # Auto-triggered
  - name: onboard-team
    category: goldenpath              # Manual

# Register provider (admin-config.yaml)
providers:
  - source: git
    url: https://github.com/org/provider
    ref: v1.0.0

# Reload providers
curl -X POST http://localhost:8081/api/admin/providers/reload \
  -H "Authorization: Bearer <admin-token>"
```

## Workflow Execution Patterns

```bash
# Automatic resource provisioning (via Score spec)
cat > score.yaml <<EOF
resources:
  db:
    type: postgres  # → database-team provider → provision-postgres workflow
    properties:
      version: "15"
EOF

innominatus-ctl submit score.yaml

# Manual golden path execution
cat > inputs.yaml <<EOF
team_name: platform
github_org: my-org
EOF

innominatus-ctl run onboard-dev-team inputs.yaml
```

## Environment Variables (Key)

```bash
# Server
SERVER_PORT=8081
SERVER_HOST=0.0.0.0

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=innominatus
DB_USER=postgres
DB_PASSWORD=postgres

# Auth
AUTH_TYPE=file                     # or "oidc"
SESSION_SECRET=changeme
OIDC_ENABLED=false
OIDC_ISSUER=""
OIDC_CLIENT_ID=""
OIDC_CLIENT_SECRET=""

# Workflow
WORKFLOW_TIMEOUT=3600
TERRAFORM_ENABLED=true
ANSIBLE_ENABLED=true
KUBECTL_ENABLED=true

# Observability
LOG_LEVEL=info                     # debug, info, warn, error
ENABLE_METRICS=true
ENABLE_TRACING=false
```

## Common Debugging

```bash
# Server logs
./innominatus  # Watch for startup errors

# Database connection
psql -h localhost -U postgres -d innominatus -c "\dt"

# Workflow logs
innominatus-ctl workflow logs <execution-id>
innominatus-ctl workflow logs <execution-id> --step terraform-apply

# Provider resolution
innominatus-ctl list-providers
innominatus-ctl provider detail database-team

# Resource state
innominatus-ctl list-resources --type postgres
innominatus-ctl resource detail <resource-id>

# Graph visualization
curl http://localhost:8081/api/workflows/{id}/graph | jq

# Health checks
curl http://localhost:8081/health
curl http://localhost:8081/ready
```

## Demo Services

```bash
# Installed by demo-time
Gitea:    http://gitea.localtest.me     (admin/admin)
ArgoCD:   http://argocd.localtest.me    (admin/argocd123)
Vault:    http://vault.localtest.me     (root)
Minio:    http://minio.localtest.me     (minioadmin/minioadmin)
Keycloak: http://keycloak.localtest.me  (admin/admin)
Grafana:  http://grafana.localtest.me   (admin/admin)

# Prerequisites: Docker Desktop with Kubernetes enabled
```

## Testing

```bash
make test             # All tests
make test-unit        # Go unit tests
make test-e2e         # Go E2E tests (no K8s)
make test-ui          # Playwright (web UI)
make coverage         # Coverage report

# Individual tests
go test ./internal/...
cd web-ui && npx playwright test

# Verification scripts
node verification/test-gitea-keycloak-oauth.mjs
```

## Release

```bash
git tag v0.1.0
git push origin v0.1.0
# GitHub Actions → Binaries, Docker images, GitHub release
```

---

**Full docs:** See CLAUDE.md, DIGEST.md, docs/
