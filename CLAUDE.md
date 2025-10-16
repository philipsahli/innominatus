# Claude AI Assistant

This document provides information about working with Claude, your AI assistant in Warp terminal.

## Overview

Claude is an AI assistant powered by the "claude 4 sonnet" model, designed to help with software development tasks, terminal operations, and coding assistance.

## Capabilities

### Terminal & System Operations
- Execute shell commands and scripts
- Navigate file systems and directories
- Manage processes and system operations
- Work with environment variables and configurations

### Code Development
- Create, read, and edit code files
- Search through codebases semantically
- Perform text-based searches with grep
- Apply code changes using diff-based editing
- Support for multiple programming languages

### Version Control
- Git operations (commits, branches, diffs, etc.)
- Repository management
- Change tracking and history analysis
- Integration with repository hosting CLIs (like `gh` for GitHub)

### Project Management
- Todo list management for complex tasks
- Task breakdown and execution tracking
- File and directory organization

## Best Practices

### Security
- Never exposes secrets in plain text
- Uses environment variables for sensitive data
- Follows secure coding practices

### Code Quality
- Adheres to existing codebase patterns and idioms
- Maintains consistency with project standards
- Suggests appropriate testing and validation

### Workflow
- Creates structured plans for complex tasks
- Provides clear, actionable instructions
- Focuses on exactly what was requested

## Usage Examples

Ask Claude to help with tasks like:
- "Fix the bug in main.go"
- "Create a new React component"
- "Show me recent git changes"
- "Run the test suite"
- "Search for authentication code"

## Getting Started

Simply describe what you need help with, and Claude will:
1. Understand your request
2. Create a plan if needed
3. Execute the necessary commands and operations
4. Provide clear feedback on results

## innominatus

This project implements a **Score-based platform orchestration component** designed for integration into enterprise Internal Developer Platform (IDP) ecosystems. It provides centralized execution of multi-step workflows from Score specifications with database persistence and RESTful API integration.

### Building the Components

The project has three main components that must be built:

**Build the Server:**
```bash
go build -o innominatus cmd/server/main.go
```

**Build the CLI:**
```bash
go build -o innominatus-ctl cmd/cli/main.go
```

**Build the Web UI:**
```bash
# Using the build script (recommended)
./scripts/build-web-ui.sh

# Or manually from web-ui directory
cd web-ui && npm run build
```

The build script automatically:
- Checks for and installs dependencies if needed
- Builds the Next.js application for production
- Outputs static files to `web-ui/out/`
- These files are served by the Go server at http://localhost:8081

### Running the Components

**Start the Server:**
```bash
# Standard mode (file-based authentication)
./innominatus

# With OIDC authentication enabled (requires demo Keycloak running)
OIDC_ENABLED=true ./innominatus

# Server runs on http://localhost:8081 by default
# Web UI: http://localhost:8081/
# API Docs (User): http://localhost:8081/swagger-user
# API Docs (Admin): http://localhost:8081/swagger-admin
# API Docs (Legacy): http://localhost:8081/swagger
```

**OIDC Environment Variables (optional):**
```bash
export OIDC_ENABLED=true
export OIDC_ISSUER="http://keycloak.localtest.me/realms/demo-realm"  # Default for demo
export OIDC_CLIENT_ID="innominatus-web"                               # Default for demo
export OIDC_CLIENT_SECRET="innominatus-client-secret"                 # Default for demo
export OIDC_REDIRECT_URL="http://localhost:8081/auth/oidc/callback"  # Default for demo

# Start server with OIDC
./innominatus
```

**Health & Monitoring Endpoints:**
```
http://localhost:8081/health   - Liveness probe (Kubernetes health checks)
http://localhost:8081/ready    - Readiness probe (service ready for traffic)
http://localhost:8081/metrics  - Prometheus metrics (performance monitoring)
```

See [docs/HEALTH_MONITORING.md](docs/HEALTH_MONITORING.md) for detailed monitoring documentation.

**Note:** When source code is modified, you must rebuild and restart:

*Server changes:*
```bash
# Stop the running server (Ctrl+C)
go build -o innominatus cmd/server/main.go
./innominatus
```

*Web UI changes:*
```bash
# Rebuild Web UI (server will pick up changes automatically)
./scripts/build-web-ui.sh
# No server restart needed - just refresh browser at http://localhost:8081
```

**Use the CLI:**
```bash
# List deployed applications
./innominatus-ctl list

# Get application status
./innominatus-ctl status app-name

# Validate a Score spec
./innominatus-ctl validate score-spec.yaml

# List environments
./innominatus-ctl environments

# Delete an application
./innominatus-ctl delete app-name

# Show admin configuration
./innominatus-ctl admin show

# List available golden paths
./innominatus-ctl list-goldenpaths

# Run a golden path workflow
./innominatus-ctl run deploy-app score-spec.yaml
./innominatus-ctl run ephemeral-env
./innominatus-ctl run db-lifecycle score-spec.yaml

# Demo Environment Commands
./innominatus-ctl demo-time    # Install/reconcile full demo environment
./innominatus-ctl demo-status  # Check demo environment health and status
./innominatus-ctl demo-nuke    # Uninstall and clean demo environment
```

### Deployment Options

innominatus provides two deployment approaches, each serving different use cases:

#### Option 1: Direct API Deployment (POST /api/specs)

**Use Case:** Simple deployments with workflows embedded directly in Score specifications

**Characteristics:**
- Direct HTTP API integration for platform tools
- Executes workflows defined within the Score spec itself
- Best for simple, self-contained deployments
- Ideal for platform integrations (Backstage, Port, custom IDPs)
- No CLI command required (use HTTP client or curl)

**Example:**
```bash
# Deploy via API with embedded workflow in Score spec
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @score-spec-with-workflow.yaml
```

#### Option 2: Golden Path Deployment (run deploy-app)

**Use Case:** Production deployments using standardized platform patterns

**Characteristics:**
- Uses pre-defined golden path workflows from `goldenpaths.yaml`
- Provides comprehensive, standardized deployment orchestration
- Recommended approach for production environments
- More control over deployment steps and dependencies
- Runs locally via CLI without server authentication

**Example:**
```bash
# Deploy via golden path (recommended)
./innominatus-ctl run deploy-app score-spec.yaml
```

**When to Use Each:**

- **Use POST /api/specs when:**
  - Building platform integrations that need direct API access
  - Deploying simple applications with basic workflows
  - Embedding deployment logic directly in Score specs
  - Integrating with external tools via HTTP API

- **Use run deploy-app when:**
  - Following standardized deployment patterns
  - Deploying to production environments
  - Requiring complex multi-step orchestration
  - Leveraging pre-defined golden path workflows
  - Running deployments from local development machines

### Kubernetes Deployment

**For platform teams deploying innominatus to production Kubernetes clusters**

innominatus can be deployed to Kubernetes using Helm for production, staging, or development environments.

#### Quick Install

```bash
# Install with bundled PostgreSQL (default)
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set postgresql.auth.password=strongPassword123
```

#### Production Install (External Database)

```bash
# Production deployment with external database
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set postgresql.enabled=false \
  --set externalDatabase.enabled=true \
  --set externalDatabase.host=postgres.example.com \
  --set externalDatabase.password=secretPassword \
  --set replicaCount=3 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=innominatus.example.com
```

#### Access innominatus in Kubernetes

```bash
# Port-forward for local access
kubectl port-forward -n innominatus-system svc/innominatus 8081:8081

# Or access via ingress (if enabled)
kubectl get ingress -n innominatus-system
```

#### Kubernetes Mode Features

When running in Kubernetes, innominatus automatically detects K8s mode and:
- Uses K8s service DNS names for component communication
- Leverages in-cluster database connection
- demo-time deploys components with K8s-native configurations
- RBAC permissions allow cluster-wide resource management

#### demo-time in Kubernetes

```bash
# Get pod name
POD=$(kubectl get pod -n innominatus-system -l app.kubernetes.io/name=innominatus -o jsonpath='{.items[0].metadata.name}')

# Run demo environment installation
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-time

# Check demo status
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-status

# Cleanup demo
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-nuke
```

#### Complete Documentation

📖 **[Kubernetes Deployment Guide](docs/platform-team-guide/kubernetes-deployment.md)** - Comprehensive guide for platform engineers covering:
- Prerequisites and installation options
- Production configuration examples
- Database setup (bundled vs. external)
- OIDC/SSO authentication setup
- Monitoring, security, and troubleshooting
- Upgrade procedures and best practices

📖 **[Helm Chart README](charts/innominatus/README.md)** - Quick reference for chart configuration

### Workflow Capabilities

innominatus supports multi-step workflows with:

- **Terraform:** Infrastructure provisioning (`terraform init`, `plan`, `apply`, `destroy`, `output`)
  - Executes in isolated workspaces per application/environment
  - Supports variable injection and output capture
  - Example: Minio S3 bucket provisioning using `aminueza/minio` provider
- **Ansible:** Configuration management (`ansible-playbook`)
- **Kubernetes:** Application deployment (namespace creation, manifest generation, `kubectl apply`)

Each workflow step runs in app and environment-specific workspaces for multi-tenant isolation.

**Terraform Step Example:**
```yaml
- name: provision-object-storage
  type: terraform
  config:
    operation: apply
    working_dir: ./terraform/minio-bucket
    variables:
      bucket_name: my-app-storage
      minio_endpoint: http://minio.minio-system.svc.cluster.local:9000
    outputs:
      - minio_url
      - bucket_name
```

### Golden Paths

innominatus supports multiple golden paths - pre-defined workflows that solve common developer and platform needs. Golden paths are configured in `goldenpaths.yaml`:

```yaml
goldenpaths:
  deploy-app: ./workflows/deploy-app.yaml
  ephemeral-env: ./workflows/ephemeral-env.yaml
  db-lifecycle: ./workflows/db-lifecycle.yaml
  observability-setup: ./workflows/observability-setup.yaml
```

**Available Golden Paths:**
- `deploy-app`: Deploy application with full infrastructure provisioning
- `ephemeral-env`: Create temporary environments for testing
- `db-lifecycle`: Manage database operations (backup, migration, health check)
- `observability-setup`: Setup monitoring and observability stack

**Golden Path Commands:**
- List available paths: `./innominatus-ctl list-goldenpaths`
- Run a path: `./innominatus-ctl run <path-name> [score-spec.yaml] [--param key=value]`
- Workflows are executed locally without requiring server authentication

**Golden Path Metadata:**
Golden paths support rich metadata including descriptions, tags, categories, and configurable parameters:
- **Description**: Human-readable explanation of what the path does
- **Category**: Grouping (deployment, cleanup, environment, database, observability)
- **Tags**: Searchable keywords for filtering
- **Estimated Duration**: Expected completion time
- **Parameters**: Required and optional parameters with defaults

Example with parameters:
```bash
# Override optional parameters
./innominatus-ctl run ephemeral-env score-spec.yaml --param ttl=4h --param environment_type=staging

# List shows all metadata and parameters
./innominatus-ctl list-goldenpaths
```

See [docs/GOLDEN_PATHS_METADATA.md](docs/GOLDEN_PATHS_METADATA.md) for detailed documentation.

### Demo Environment

**Important**: The demo environment is provided for **demonstration and development purposes only**. Enterprise deployments should integrate with existing platform infrastructure using the RESTful API.

innominatus includes a complete demo environment feature that sets up a development platform on Docker Desktop Kubernetes for testing and demonstration. The demo environment includes:

**Included Components:**
- **Gitea**: Git repository hosting (http://gitea.localtest.me) - `admin/admin`
- **ArgoCD**: GitOps continuous deployment (http://argocd.localtest.me) - `admin/argocd123`
- **Vault**: Secret management (http://vault.localtest.me) - Root token: `root`
- **Minio**: S3-compatible object storage (http://minio.localtest.me, Console: http://minio-console.localtest.me) - `minioadmin/minioadmin`
- **Grafana**: Monitoring dashboards (http://grafana.localtest.me) - `admin/admin`
- **Prometheus**: Metrics collection (http://prometheus.localtest.me)
- **Kubernetes Dashboard**: Cluster management UI (http://k8s.localtest.me) - Token: `kubectl -n kubernetes-dashboard create token admin-user`
- **Demo App**: Sample application (http://demo.localtest.me)
- **NGINX Ingress**: Ingress controller for local routing

**Prerequisites:**
- Docker Desktop with Kubernetes enabled
- `kubectl` context set to `docker-desktop`
- `helm` installed
- Internet connection for downloading Helm charts

**Demo Commands:**

```bash
# Install/reconcile the complete demo environment
./innominatus-ctl demo-time

# Check health status of all services
./innominatus-ctl demo-status

# Completely remove the demo environment
./innominatus-ctl demo-nuke
```

**Features:**
- **Immutable Installation**: `demo-time` ensures consistent state by reconciling all components
- **Health Monitoring**: Real-time health checks for all services
- **GitOps Ready**: Automatically seeds platform-config repository in Gitea
- **Local Domains**: Uses `*.localtest.me` for local ingress (resolves to 127.0.0.1)
- **ArgoCD Integration**: Creates Application manifests for GitOps workflows
- **Kubernetes Dashboard**: Full cluster management UI with admin ServiceAccount
- **Grafana Dashboards**: Pre-loaded with Cluster Health Dashboard for monitoring
- **Credential Display**: Shows all service credentials and quick start guide
- **Database Cleanup**: `demo-nuke` automatically cleans database tables using default connection settings

**Architecture:**
- All services run in dedicated Kubernetes namespaces
- Git repository seeding creates ArgoCD Application manifests
- Demo app deployed via Score specification
- Complete cleanup with namespace deletion, PVC removal, and database table truncation (localhost:5432/idp_orchestrator)

The demo environment is perfect for:
- Testing Score specifications and workflow development
- Learning orchestration capabilities
- Demonstrating platform integration capabilities
- Local development and experimentation
- **Not for enterprise production use** - use API integration instead

### Admin Configuration

innominatus supports admin-level configuration through `admin-config.yaml`:

```yaml
admin:
  defaultCostCenter: "engineering"
  defaultRuntime: "kubernetes"
  splunkIndex: "orchestrator-logs"

resourceDefinitions:
  postgres: "managed-postgres-cluster"
  redis: "redis-cluster"
  volume: "persistent-volume-claim"
  route: "ingress-route"

policies:
  enforceBackups: true
  allowedEnvironments:
    - "development"
    - "staging"
    - "production"
```

**Features:**
- Loads at server startup and displays configuration
- CLI command `./innominatus-ctl admin show` displays current settings
- Supports admin defaults, resource definitions, and policies
- Uses `gopkg.in/yaml.v3` for configuration parsing

### OIDC Authentication

innominatus supports enterprise SSO authentication via OpenID Connect (OIDC) with providers like Keycloak.

**Starting Server with OIDC:**
```bash
# Demo environment (with Keycloak from demo-time)
OIDC_ENABLED=true ./innominatus

# Production with custom Keycloak
export OIDC_ENABLED=true
export OIDC_ISSUER="https://keycloak.company.com/realms/production"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="your-client-secret"
export OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback"
./innominatus
```

**Authentication Features:**
- **Web UI Login**: "Login with Keycloak" button appears on login page
- **Session Management**: HttpOnly cookies + localStorage tokens
- **API Key Generation**: OIDC users can generate API keys for CLI/API access
- **Dual User Sources**:
  - Local users (users.yaml): File-based API keys
  - OIDC users (database): Database-backed API keys with SHA-256 hashing
- **Automatic Detection**: System automatically determines user type

**User Workflow:**
1. Login via OIDC SSO (Web UI)
2. Navigate to Profile page
3. Generate API key (provide name and expiry days)
4. Use API key for CLI/API access:
   ```bash
   export IDP_API_KEY="your-generated-key"
   curl -H "Authorization: Bearer $IDP_API_KEY" \
     http://localhost:8081/api/specs
   ```

**Database Requirements:**
OIDC users require PostgreSQL for API key storage:
- Table: `user_api_keys` (created automatically via migration)
- API keys stored as SHA-256 hashes (never plaintext)
- Supports key lifecycle management (creation, listing, revocation)

**See:** [OIDC Authentication Guide](docs/OIDC_AUTHENTICATION.md) for complete setup instructions.

### Enterprise Integration

**API-First Design**: innominatus is designed as an integration component for existing IDP platforms:

**Platform Integration Examples:**
```bash
# Backstage Software Catalog Integration
curl -X POST http://orchestrator.company.com/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $BACKSTAGE_TOKEN" \
  --data-binary @score-spec.yaml

# Monitor workflow execution from platform
curl -H "Authorization: Bearer $PLATFORM_TOKEN" \
  http://orchestrator.company.com/api/workflows?app=my-app
```

**Integration Patterns:**
- **Backstage Plugin**: Score catalog integration with workflow execution
- **CNOE Platform**: Orchestration component within larger IDP ecosystem
- **GitOps Triggers**: Webhook-driven workflow execution from Git repositories
- **CI/CD Integration**: Platform-triggered deployments via REST API
- **Custom IDPs**: Any platform supporting REST API integration

**Production Considerations:**
- Database persistence for workflow tracking and audit trails
- Authentication integration with enterprise identity providers
- Role-based access control (RBAC) for multi-tenant environments
- Prometheus metrics and distributed tracing integration
- Horizontal pod autoscaling for workflow execution workloads

### Development Mode

For development, you can also run directly:

```bash
# Run server in development
go run cmd/server/main.go

# Run CLI commands in development
go run cmd/cli/main.go run deploy-app score-spec.yaml

# Run workflow from Score spec
go run . score-spec-with-workflow.yaml

# Run web-ui in development mode (from web-ui directory)
cd web-ui && npm run dev
```

### Release Process

innominatus uses **GoReleaser** for automated multi-platform releases and Docker container builds.

#### Creating a Release

1. **Ensure all changes are committed:**
```bash
git add .
git commit -m "feat: your changes"
git push origin main
```

2. **Create and push a version tag:**
```bash
# Create a semantic version tag
git tag v0.1.0

# Push the tag to trigger the release workflow
git push origin v0.1.0
```

3. **GitHub Actions automatically:**
   - Runs tests to ensure code quality
   - Builds binaries for all platforms (Linux, macOS, Windows - AMD64 & ARM64)
   - Creates multi-arch Docker images (AMD64 & ARM64)
   - Pushes images to GitHub Container Registry (ghcr.io)
   - Creates GitHub Release with:
     - Binary archives for all platforms
     - Docker pull instructions
     - Auto-generated changelog
     - SHA256 checksums

#### Release Artifacts

**Binary Downloads:**
- `innominatus_v0.1.0_linux_amd64.tar.gz` - Linux x86_64
- `innominatus_v0.1.0_linux_arm64.tar.gz` - Linux ARM64
- `innominatus_v0.1.0_darwin_amd64.tar.gz` - macOS Intel
- `innominatus_v0.1.0_darwin_arm64.tar.gz` - macOS Apple Silicon
- `innominatus_v0.1.0_windows_amd64.zip` - Windows 64-bit

**Docker Images:**
```bash
# Pull latest version
docker pull ghcr.io/philipsahli/innominatus:latest

# Pull specific version
docker pull ghcr.io/philipsahli/innominatus:v0.1.0

# Run container
docker run -p 8081:8081 ghcr.io/philipsahli/innominatus:v0.1.0
```

#### Testing Releases Locally

```bash
# Validate GoReleaser configuration
goreleaser check

# Build snapshot (for testing, no tag required)
goreleaser build --snapshot --clean --single-target

# Test binaries
./dist/innominatus_darwin_arm64_v8.0/innominatus --help
./dist/innominatus-ctl_darwin_arm64_v8.0/innominatus-ctl --help
```

#### Docker Image Contents

The Docker image is a multi-stage build that includes:
- **innominatus** server binary
- **innominatus-ctl** CLI binary (for debugging)
- **Next.js web-ui** (standalone build)
- Configuration files (admin-config.yaml, goldenpaths.yaml)
- Workflow templates (workflows/)
- Documentation (docs/)

**Image Size:** ~50MB (using distroless base)

#### Versioning Strategy

innominatus follows [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., v1.2.3)
- **v0.x.x** - Pre-release versions (current)
- **v1.0.0** - First stable release
- **v1.1.0** - Minor version with new features
- **v1.1.1** - Patch version with bug fixes

---

## Development Principles

### SOLID Principles

innominatus follows SOLID design principles for maintainable, scalable code:

**Single Responsibility Principle (SRP)**
- Each component has one reason to change
- `internal/server/handlers.go` - HTTP routing only
- `internal/database/` - Database operations only
- `internal/workflow/` - Workflow execution only

**Open/Closed Principle (OCP)**
- Workflow steps are extensible via configuration
- Golden paths can be added without modifying core engine
- Resource definitions configurable via `admin-config.yaml`

**Liskov Substitution Principle (LSP)**
- Workflow step interfaces are polymorphic
- Database interfaces (PostgreSQL, SQLite) are interchangeable
- Storage backends can be swapped without breaking contracts

**Interface Segregation Principle (ISP)**
- Small, focused interfaces (WorkflowExecutor, DatabaseRepository, AuthProvider)
- Clients depend only on methods they use

**Dependency Inversion Principle (DIP)**
- High-level modules depend on abstractions, not implementations
- Database layer abstracted via interfaces
- Workflow engine doesn't depend on concrete step implementations

### KISS Philosophy (Keep It Simple, Stupid)

**Simplicity Over Complexity**
- Configuration via YAML files, not complex DSLs
- RESTful API with standard HTTP methods
- Database schema normalized, no over-engineering
- Workflow steps are shell commands with clear inputs/outputs

**Simple > Clever**
- Readable code over performance tricks
- Clear error messages over generic failures
- Straightforward authentication (API keys, OIDC) over custom auth schemes

**Avoid Premature Optimization**
- Build features when needed, not "just in case"
- Measure before optimizing (use Prometheus metrics)
- Simple PostgreSQL queries first, optimize only when proven necessary

### YAGNI (You Aren't Gonna Need It)

**Build What You Need**
- No speculative features or "future-proofing"
- Implement based on actual requirements, not hypothetical use cases
- Remove unused code aggressively

**Defer Decisions**
- Don't build abstraction layers until you have 3+ use cases
- Start with simple implementation, refactor when patterns emerge
- Configuration files over hard-coded values, but avoid config bloat

**Examples in innominatus:**
- Golden paths added when users requested them, not upfront
- OIDC authentication added when enterprise users needed it
- Prometheus metrics added when observability became required

### Minimal Documentation Philosophy

**Code is Documentation**
- Self-documenting code with clear names (`handleGraphHistory`, `executeWorkflow`, `validateScoreSpec`)
- Type-safe interfaces over comments (Go types, TypeScript interfaces)
- Examples in code over lengthy prose

**When to Document:**
- **Public APIs**: RESTful endpoints, CLI commands (required)
- **Complex Algorithms**: Critical path calculation, workflow engine logic
- **Configuration**: YAML structure, environment variables
- **Architecture**: High-level system design (Mermaid diagrams)

**When NOT to Document:**
- **Obvious Code**: `getUserByID()` doesn't need "Gets a user by their ID" comment
- **Implementation Details**: How loops work, basic language features
- **Temporary Workarounds**: Fix the code instead of documenting hacks

**Documentation Hierarchy:**
1. **Code First**: Clear function/variable names, type signatures
2. **Inline Comments**: Only when "why" is not obvious from code
3. **README Files**: Quick starts, examples, architecture diagrams
4. **Comprehensive Guides**: User guide, platform team guide (minimal, example-driven)

**innominatus Documentation Strategy:**
- `CLAUDE.md`: Development context for AI assistants
- `README.md`: Quick overview with setup commands
- `docs/`: Minimal guides (getting-started, troubleshooting)
- Code comments: Only for non-obvious business logic

### Verification-First Development Protocol

**Test Before Code**
1. Write verification script first (what does success look like?)
2. Implement feature
3. Run verification
4. Iterate until verified

**Verification Types:**
- **Unit Tests**: Go test suite (`go test ./...`)
- **Integration Tests**: Database, API, workflow execution
- **UI Tests**: Puppeteer tests for web-ui
- **Manual Verification**: CLI commands, curl tests

**Example Workflow:**
```bash
# 1. Write verification script
cat > verification/test-golden-path.mjs <<EOF
// Verify golden path execution
EOF

# 2. Implement feature
# ... code changes ...

# 3. Run verification
node verification/test-golden-path.mjs

# 4. Iterate until pass
```

**CI/CD Verification:**
- GitHub Actions runs tests on every PR
- Security scanning (CodeQL)
- Build verification (multi-platform)
- Docker image build and push

**Verification Artifacts:**
- `verification/` directory contains test scripts
- `.github/workflows/` contains CI automation
- `tests/` directory for integration tests

---

## Code Quality Standards

### Go Backend Standards

**File Organization:**
- `cmd/` - Entry points (server, cli)
- `internal/` - Private application code
- `pkg/` - Public libraries (if any)
- `migrations/` - Database schema migrations

**Naming Conventions:**
- Packages: lowercase, single word (`database`, `server`, `workflow`)
- Interfaces: noun or adjective (`Repository`, `Executor`, `Authenticator`)
- Functions: verb or verb phrase (`executeWorkflow`, `validateSpec`)

**Error Handling:**
- Always return errors, never panic in production code
- Wrap errors with context: `fmt.Errorf("failed to execute workflow %s: %w", name, err)`
- Use structured logging (`zerolog`) for error context

**Testing:**
- Unit tests alongside code (`handlers_test.go`)
- Table-driven tests for multiple scenarios
- Mock external dependencies (database, HTTP clients)

### TypeScript/React Frontend Standards

**Component Organization:**
- `web-ui/src/components/` - Reusable components
- `web-ui/src/app/` - Next.js pages and routing
- One component per file, named after component

**Naming Conventions:**
- Components: PascalCase (`GraphVisualization`, `PerformanceMetrics`)
- Hooks: camelCase with `use` prefix (`useAuth`, `useFetchGraph`)
- Types/Interfaces: PascalCase (`WorkflowExecution`, `GraphNode`)

**State Management:**
- React hooks for local state (`useState`, `useEffect`)
- Context API for global state (auth, theme)
- Avoid prop drilling beyond 2-3 levels

**Type Safety:**
- Define interfaces for all API responses
- Use TypeScript strict mode
- No `any` types except for truly dynamic data

### Database Standards

**Migration Rules:**
- Never modify existing migrations
- New changes = new migration file
- Rollback migrations must be provided
- Test migrations against production-like data

**Query Patterns:**
- Use GORM for simple queries
- Raw SQL for complex joins/aggregations
- Always use parameterized queries (prevent SQL injection)
- Index columns used in WHERE/JOIN clauses

**Schema Conventions:**
- Table names: plural, lowercase, snake_case (`workflow_executions`, `user_api_keys`)
- Column names: snake_case (`created_at`, `workflow_name`)
- Primary keys: `id SERIAL PRIMARY KEY`
- Timestamps: `created_at`, `updated_at` (auto-managed)

---

*Created: 2025-09-13*
*Updated: 2025-10-16* - Added SOLID, KISS, YAGNI, Minimal Documentation, Verification-First, and Code Quality Standards
