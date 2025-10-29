# innominatus - Score-based Platform Orchestration

Score-based platform orchestration component for enterprise Internal Developer Platform (IDP) ecosystems. Provides centralized execution of multi-step workflows with database persistence and RESTful API integration.

## Quick Start

**Build (using Makefile):**
```bash
make install          # Install dependencies
make build            # Build all components
make test             # Run all tests
```

**Or build directly:**
```bash
go build -o innominatus cmd/server/main.go          # Server
go build -o innominatus-ctl cmd/cli/main.go         # CLI
./scripts/build-web-ui.sh                            # Web UI
```

**Run:**
```bash
make dev              # Start server + web UI (http://localhost:8081 & http://localhost:3000)
# Or separately:
./innominatus                                        # Start server only
./innominatus-ctl list                               # List applications
./innominatus-ctl run deploy-app score-spec.yaml    # Deploy via golden path
```

**Development:**
```bash
make dev              # Start both server + web UI
# Or separately:
go run cmd/server/main.go                            # Dev server
cd web-ui && npm run dev                             # Dev web UI (http://localhost:3000)
```

**Testing:**
```bash
make test             # Run all local tests
make test-unit        # Go unit tests only
make test-e2e         # Go E2E tests (no K8s)
make test-ui          # Web UI Playwright tests
make coverage         # Generate coverage report
make help             # Show all available commands
```

## Core Features

### Deployment Options

**1. Direct API (Simple Deployments)**
```bash
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @score-spec.yaml
```

**2. Golden Paths (Production)**
```bash
./innominatus-ctl run deploy-app score-spec.yaml
./innominatus-ctl run ephemeral-env
./innominatus-ctl list-goldenpaths
```

### CLI (Cobra Framework)

**Built with:** `github.com/spf13/cobra` (same framework as kubectl, docker, helm)

**Key Features:**
- **31 commands** organized hierarchically (e.g., `workflow detail`, `workflow logs`)
- **Shell completion** for bash, zsh, fish, powershell
- **Auto-generated help** with `--help` flag for all commands
- **Global flags**: `--server`, `--details`, `--skip-validation`

**Setup shell completion:**
```bash
innominatus-ctl completion bash > /etc/bash_completion.d/innominatus-ctl
source /etc/bash_completion.d/innominatus-ctl
```

**Examples:**
```bash
innominatus-ctl --help                           # All commands
innominatus-ctl workflow logs <id> --step init   # Hierarchical subcommand
innominatus-ctl list-resources --type postgres   # Filtered listing
```

**Authentication:** Auto-authenticates for server commands, skips for local commands (validate, analyze, demo-*).

### Workflow Capabilities

- **Terraform**: Infrastructure provisioning (init, plan, apply, destroy)
- **Ansible**: Configuration management
- **Kubernetes**: Application deployment (namespace, manifest, kubectl apply)

Example workflow step:
```yaml
- name: provision-storage
  type: terraform
  config:
    operation: apply
    working_dir: ./terraform/minio
    variables:
      bucket_name: my-app-storage
```

### Authentication

**File-based (Development):**
```bash
./innominatus  # Uses users.yaml
```

**OIDC (Production):**
```bash
export OIDC_ENABLED=true
export OIDC_ISSUER="https://keycloak.company.com/realms/production"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="your-secret"
./innominatus
```

Users can generate API keys via Web UI Profile page for CLI/API access.

## Provider Architecture

### Three-Layer System

**1. Core (Go Engine)**
- Workflow execution engine
- Step executors: terraform, kubernetes, ansible, policy, gitea-repo, argocd-app
- API server and authentication
- Database persistence

**2. Built-in Provider (Filesystem)**
- Standard workflows shipped with innominatus
- Located in `providers/builtin/`
- Example: postgres-cluster, redis-cache, deploy-database

**3. Extension Providers (Git Repositories)**
- Custom workflows from product/platform teams
- Loaded dynamically from Git with version pinning
- Configured in `admin-config.yaml`
- Supports hot-reload for rapid iteration

### Workflow Types

**Provisioners (Single-Resource)**
- Create individual resources (database, namespace, bucket, repository)
- Simple, composable building blocks
- Example: `postgres-cluster.yaml`, `gitea-repo.yaml`

**Golden Paths (Multi-Resource Orchestration)**
- Combine multiple provisioners into end-to-end flows
- Opinionated "happy path" for common scenarios
- Example: `onboard-dev-team.yaml` (creates namespace + repo + ArgoCD app)

### Creating a Provider

**1. Directory Structure:**
```
my-provider/
├── provider.yaml          # Manifest
└── workflows/
    ├── postgres.yaml      # Provisioner workflow
    └── onboard-team.yaml  # Golden path workflow
```

**2. Provider Manifest (provider.yaml):**
```yaml
apiVersion: v1
kind: Provider
metadata:
  name: my-provider
  version: 1.0.0
  category: infrastructure  # or "service"
  description: Custom workflows for my team

compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0

workflows:
  - name: postgres-cluster
    file: ./workflows/postgres.yaml
    description: PostgreSQL cluster provisioner
    category: provisioner
    tags: [database, postgres]

  - name: onboard-team
    file: ./workflows/onboard-team.yaml
    description: Complete team onboarding
    category: goldenpath
    tags: [onboarding, team]
```

**3. Register Provider (admin-config.yaml):**
```yaml
providers:
  - source: git
    url: https://github.com/my-org/my-provider
    ref: v1.0.0  # Tag, branch, or commit
```

**4. Use Workflows:**
```bash
./innominatus-ctl list-goldenpaths           # List available
./innominatus-ctl run onboard-team inputs.yaml
```

### Provider Categories

**Infrastructure Providers (Platform Teams)**
- AWS, Azure, GCP resources
- Kubernetes primitives
- Storage, databases, messaging
- Example: `aws-rds`, `azure-cosmosdb`, `k8s-namespace`

**Service Providers (Product Teams)**
- Business domain resources
- Application-specific workflows
- ML pipelines, analytics, ecommerce
- Example: `ml-model-registry`, `analytics-dashboard`

## Kubernetes Deployment

**Quick Install:**
```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set postgresql.auth.password=strongPassword123
```

**Production (External DB):**
```bash
helm install innominatus ./charts/innominatus \
  --set postgresql.enabled=false \
  --set externalDatabase.enabled=true \
  --set externalDatabase.host=postgres.example.com \
  --set replicaCount=3 \
  --set ingress.enabled=true
```

See: [docs/platform-team-guide/kubernetes-deployment.md](docs/platform-team-guide/kubernetes-deployment.md)

## Demo Environment

**Commands:**
```bash
./innominatus-ctl demo-time    # Install full demo (Gitea, ArgoCD, Vault, Minio, etc.)
./innominatus-ctl demo-status  # Check health
./innominatus-ctl demo-nuke    # Remove demo
```

**Included Services:**
- Gitea (http://gitea.localtest.me) - admin/admin
- ArgoCD (http://argocd.localtest.me) - admin/argocd123
- Vault (http://vault.localtest.me) - root
- Minio (http://minio.localtest.me) - minioadmin/minioadmin
- Grafana (http://grafana.localtest.me) - admin/admin

**Prerequisites:** Docker Desktop with Kubernetes enabled

⚠️ **Demo only** - not for production. Integrate via API instead.

## Important Endpoints

```
http://localhost:8081/              # Web UI
http://localhost:8081/health        # Health check
http://localhost:8081/ready         # Readiness probe
http://localhost:8081/metrics       # Prometheus metrics
http://localhost:8081/swagger-user  # API docs (user)
http://localhost:8081/swagger-admin # API docs (admin)
```

## Release Process

```bash
# Create and push tag to trigger release
git tag v0.1.0
git push origin v0.1.0

# GitHub Actions automatically:
# - Builds binaries (Linux, macOS, Windows - AMD64 & ARM64)
# - Creates Docker images (ghcr.io/philipsahli/innominatus:v0.1.0)
# - Publishes GitHub release with artifacts
```

## Development Principles

### SOLID
- **SRP**: One responsibility per component (handlers, database, workflow)
- **OCP**: Extensible via config (golden paths, resource definitions)
- **LSP**: Interchangeable interfaces (PostgreSQL/SQLite)
- **ISP**: Small, focused interfaces (WorkflowExecutor, AuthProvider)
- **DIP**: Depend on abstractions, not implementations

### KISS (Keep It Simple)
- YAML config over complex DSLs
- RESTful API with standard HTTP
- Shell commands for workflow steps
- Clear error messages

### YAGNI (You Aren't Gonna Need It)
- Build features when needed, not speculatively
- Defer abstractions until 3+ use cases
- Remove unused code aggressively

### Minimal Documentation
- **Code is documentation**: Self-documenting names, type-safe interfaces
- **Document when**: Public APIs, complex algorithms, architecture
- **Don't document**: Obvious code, implementation details, temporary hacks
- **Hierarchy**: Code → Inline comments → READMEs → Guides

### Verification-First
1. Write verification test first
2. Implement feature
3. Run verification
4. Iterate until pass

## Code Standards

### Go
- **Structure**: `cmd/` (entry), `internal/` (private), `migrations/` (DB)
- **Naming**: Packages (lowercase), Interfaces (nouns), Functions (verbs)
- **Errors**: Always return, wrap with context, structured logging
- **Tests**: Table-driven, mock external deps

### TypeScript/React
- **Structure**: `web-ui/src/components/`, `web-ui/src/app/`
- **Naming**: Components (PascalCase), hooks (useCamelCase), types (PascalCase)
- **State**: React hooks (local), Context API (global)
- **Type Safety**: Strict mode, no `any` types

### Database
- **Migrations**: Never modify existing, always provide rollback
- **Tables**: plural, snake_case (`workflow_executions`)
- **Columns**: snake_case (`created_at`)
- **Queries**: GORM (simple), raw SQL (complex), parameterized (security)

---

*Updated: 2025-10-29* - Added Provider Architecture documentation with unified workflows concept
