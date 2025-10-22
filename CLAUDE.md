# innominatus - Score-based Platform Orchestration

Score-based platform orchestration component for enterprise Internal Developer Platform (IDP) ecosystems. Provides centralized execution of multi-step workflows with database persistence and RESTful API integration.

## Quick Start

### Fast Start with SQLite (2 minutes, zero dependencies)

```bash
make build                                 # Build server
./scripts/add-sqlite-dependency.sh         # Add SQLite driver
DB_DRIVER=sqlite ./innominatus             # Start server (http://localhost:8081)
```

**Perfect for:** New developers, quick prototyping, demos
**See:** [SQLite Development Guide](docs/quick-start/sqlite-development.md)

### Full Start with PostgreSQL (production-like)

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
./innominatus                                        # Start server only (PostgreSQL)
./innominatus-ctl list                               # List applications
./innominatus-ctl run deploy-app score-spec.yaml    # Deploy via golden path
```

**Development:**
```bash
make dev              # Start both server + web UI
# Or separately:
go run cmd/server/main.go                            # Dev server
cd web-ui && npm run dev                             # Dev web UI (http://localhost:3000)

# With SQLite (faster):
DB_DRIVER=sqlite go run cmd/server/main.go           # SQLite dev server
```

**Testing:**
```bash
make test             # Run all local tests (PostgreSQL)
make test-sqlite      # Run tests with SQLite (faster, no Docker)
make test-both        # Run tests with both databases
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

### Database Options

Innominatus supports two database drivers for different use cases:

**SQLite (Development):**
```bash
DB_DRIVER=sqlite ./innominatus                 # File-based (default: ./data/innominatus.db)
DB_DRIVER=sqlite DB_PATH=:memory: ./innominatus # In-memory (fastest)
```

**Benefits:**
- Zero setup (no Docker, no PostgreSQL)
- Fast startup (~1s vs ~5s)
- Perfect for onboarding, prototyping, demos
- File-based or in-memory options

**PostgreSQL (Production):**
```bash
./innominatus  # Uses default PostgreSQL config
# Or with custom config:
DB_DRIVER=postgres DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_NAME=idp_orchestrator ./innominatus
```

**Benefits:**
- Production-ready
- Full concurrency support
- Advanced features (JSONB, full-text search)
- Required for production deployments

**Configuration Files:**
- SQLite: Copy [.env.sqlite.example](.env.sqlite.example) to `.env`
- PostgreSQL: Copy [.env.postgres.example](.env.postgres.example) to `.env`

**See:** [Database Options Documentation](docs/quick-start/README.md)

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

*Updated: 2025-10-19* - Compacted for readability and essential information only
