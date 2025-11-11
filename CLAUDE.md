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

**2. CLI Deploy Command (Recommended)**
```bash
# Deploy with real-time watch
./innominatus-ctl deploy score-spec.yaml -w

# Incremental deployment (add resources to existing app)
./innominatus-ctl deploy score-ecommerce-backend-v1.yaml -w  # Database only
./innominatus-ctl deploy score-ecommerce-backend-v2.yaml -w  # Database + S3
# System detects existing db, provisions only new S3 resource
```

**3. Golden Paths (Multi-Resource Orchestration)**
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
innominatus-ctl deploy score-spec.yaml -w        # Deploy Score spec with real-time watch
innominatus-ctl workflow logs <id> --step init   # Hierarchical subcommand
innominatus-ctl list-resources --type postgres   # Filtered listing
```

**Authentication:** Auto-authenticates for server commands, skips for local commands (validate, analyze, demo-*).

### Workflow Capabilities

Innominatus supports 16 step types for workflow execution:

**Infrastructure & Deployment:**
- **terraform** - Infrastructure provisioning (init, plan, apply, destroy, output)
- **terraform-generate** - Generate Terraform code from Score resources
- **kubernetes** - Kubernetes operations (apply, delete, create-namespace, get)
- **ansible** - Ansible playbook execution

**GitOps & Source Control:**
- **gitea-repo** - Gitea repository management (create, delete)
- **argocd-app** - ArgoCD application management (create, update, delete, sync)

**Security & Compliance:**
- **policy** - Custom shell script execution for validation/policies
- **security** - Security scanning (integration point for security tools)
- **vault-setup** - HashiCorp Vault configuration

**Resource Management:**
- **resource-provisioning** - Automatic resource provisioning from Score specs
- **database-migration** - Database schema migrations
- **tagging** - Resource tagging for governance

**Observability:**
- **monitoring** - Monitoring/observability setup (Prometheus, Grafana, etc.)
- **cost-analysis** - Cost estimation and tracking

**Utilities:**
- **validation** - Pre/post deployment validation checks

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
- Event-driven orchestration engine with automatic resource provisioning
- Workflow execution engine with 16 step executors (terraform, kubernetes, ansible, policy, gitea-repo, argocd-app, resource-provisioning, security, vault-setup, monitoring, validation, cost-analysis, tagging, database-migration, terraform-generate)
- Provider registry and resolver for automatic resource-to-provider matching with CRUD support
- API server and authentication (file-based, OIDC)
- Database persistence with graph storage

**2. Built-in Provider (Filesystem)**
- Standard workflows shipped with innominatus
- Located in `providers/builtin/`
- Example: postgres-cluster, redis-cache, deploy-database

**3. Extension Providers (Git Repositories)**
- Custom workflows from product/platform teams
- Loaded dynamically from Git with version pinning
- Configured in `admin-config.yaml`
- Supports hot-reload for rapid iteration

### Event-Driven Orchestration

**Automatic Resource Provisioning:**
When a developer deploys a Score spec requesting resources (e.g., postgres database), innominatus automatically:

1. **Creates Resource**: Resource stored with state='requested' and workflow_execution_id=NULL
2. **Background Polling**: Orchestration engine polls every 5 seconds for pending resources
3. **Provider Resolution**: Resolver matches resource type to provider via capabilities
4. **Workflow Execution**: Appropriate provisioner workflow executes automatically (state='provisioning')
5. **Graph Update**: Creates complete dependency graph: spec → resource → provider → workflow
6. **Workflow Monitoring**: Engine polls provisioning resources and checks workflow execution status
7. **State Transitions**: Resource state updates automatically: requested → provisioning → active (or failed)

**Polling Loop (every 5 seconds):**
1. **Poll Pending Resources**: Pick up new resources with state='requested'
2. **Poll Provisioning Resources**: Check completed workflows and update resource state to 'active' or 'failed'
3. **Recover Orphaned Resources**: Reset stuck resources back to 'requested' state

**Example Flow:**
```
Score spec (postgres) → Resource (state='requested') → Engine detects →
Resolver matches 'postgres' → database-team provider → provision-postgres workflow →
Execution (state='provisioning') → Workflow completes → Engine polls →
Resource becomes 'active' (or 'failed')
```

**Application UPDATE Flow:**
When redeploying an existing application with additional resources:
1. **Detect Existing**: Handler checks if application already exists
2. **UPSERT Spec**: Update application Score spec in database
3. **Compare Resources**: Identify NEW resources not already provisioned
4. **Create Only New**: Create resource instances only for new resources
5. **Skip Existing**: Existing resources remain unchanged (no duplicate provisioning)

This allows developers to add resources to running applications without redeploying everything.

### Workflow Types

**Provisioners (Single-Resource)**
- Create individual resources (database, namespace, bucket, repository)
- Simple, composable building blocks
- Automatically triggered by orchestration engine
- Declare capabilities for resource types they handle
- Example: `provision-postgres.yaml`, `provision-s3.yaml`, `provision-namespace.yaml`

**Golden Paths (Multi-Resource Orchestration)**
- Combine multiple provisioners into end-to-end flows
- Opinionated "happy path" for common scenarios
- Manually triggered via CLI or API
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

# Declare which resource types this provider can handle
capabilities:
  resourceTypes:
    - postgres      # Primary resource type
    - postgresql    # Alias
    - mysql
    - mongodb

compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0

workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    description: PostgreSQL cluster provisioner
    category: provisioner  # Auto-triggered by orchestration engine
    tags: [database, postgres]

  - name: onboard-team
    file: ./workflows/onboard-team.yaml
    description: Complete team onboarding
    category: goldenpath  # Manually triggered
    tags: [onboarding, team]
```

**Key Fields:**
- `capabilities.resourceTypes`: Array of resource types this provider can provision (simple format for CREATE operations only)
- `capabilities.resourceTypeCapabilities`: Operation-specific workflow mapping (advanced format for full CRUD support)
- `workflows[].category`: Either `provisioner` (auto-triggered) or `goldenpath` (manual)
- `workflows[].operation`: CRUD operation type - `create`, `read`, `update`, or `delete`

**3. Register Provider (admin-config.yaml):**
```yaml
providers:
  - source: git
    url: https://github.com/my-org/my-provider
    ref: v1.0.0  # Tag, branch, or commit
```

**4. Use Workflows:**
```bash
# Automatic provisioning (via Score spec)
cat > score.yaml <<EOF
apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  main:
    image: myapp:latest
resources:
  db:
    type: postgres  # Automatically triggers database-team provider
    properties:
      version: "15"
      size: "medium"
EOF

# Submit spec - postgres resource will be auto-provisioned
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @score.yaml

# Manual golden path execution
./innominatus-ctl list-goldenpaths           # List available
./innominatus-ctl run onboard-team inputs.yaml
```

### Provider Categories

**Infrastructure Providers (Platform Teams)**
- AWS, Azure, GCP resources
- Kubernetes primitives
- Storage, databases, messaging
- Declare capabilities for automatic provisioning
- Example providers:
  - `database-team`: [postgres, postgresql]
  - `storage-team`: [s3, s3-bucket, object-storage, minio-bucket]
  - `container-team`: [namespace, kubernetes-namespace, gitea-repo, argocd-app]
  - `vault-team`: [vault-space, vault-namespace, secrets]
  - `identity-team`: [gitea-org, keycloak-group, iam-group]
  - `observability-team`: [prometheus, loki, tempo, grafana-dashboard]

**Service Providers (Product Teams)**
- Business domain resources
- Application-specific workflows
- ML pipelines, analytics, ecommerce
- Example: `ml-model-registry`, `analytics-dashboard`

### CRUD Workflow Operations

Providers can define separate workflows for CREATE, UPDATE, and DELETE operations, enabling full resource lifecycle management.

**Operation-Based Capabilities Format:**

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: database-team
  version: 1.0.0

capabilities:
  # Advanced format: Operation-specific workflow mapping
  resourceTypeCapabilities:
    - type: postgres
      operations:
        create:
          workflow: provision-postgres
        update:
          workflow: update-postgres
        delete:
          workflow: delete-postgres

    - type: postgresql
      aliasFor: postgres  # Alias resolves to primary type

  # Legacy format: Simple list (CREATE only, backward compatible)
  resourceTypes: [postgres, postgresql]

workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    category: provisioner
    operation: create  # Explicit operation
    tags: [database, postgres]

  - name: update-postgres
    file: ./workflows/update-postgres.yaml
    category: provisioner
    operation: update  # UPDATE operation
    tags: [database, postgres, scaling]

  - name: delete-postgres
    file: ./workflows/delete-postgres.yaml
    category: provisioner
    operation: delete  # DELETE operation
    tags: [database, postgres, cleanup]
```

**Using CRUD Operations:**

```go
// CREATE resource (default operation)
resource, err := resourceManager.CreateResourceInstance(
  "my-app",
  "my-postgres",
  "postgres",
  map[string]interface{}{
    "version": "15",
    "replicas": 2,
  },
)

// UPDATE resource (scale up replicas)
resource.DesiredOperation = stringPtr("update")
resource.Configuration = map[string]interface{}{
  "replicas": 5,  // Scale from 2 to 5
}
// Orchestration engine will use update-postgres workflow

// DELETE resource
resource.DesiredOperation = stringPtr("delete")
// Orchestration engine will use delete-postgres workflow
```

**Workflow Override (explicit selection):**

```go
// Use specific workflow instead of auto-resolution
resource.WorkflowOverride = stringPtr("scale-postgres")
resource.WorkflowTags = []string{"scaling"}  // For disambiguation
```

**Tag-Based Disambiguation:**

When multiple workflows handle the same operation, use tags to select:

```yaml
capabilities:
  resourceTypeCapabilities:
    - type: postgres
      operations:
        update:
          workflows:
            - name: update-postgres-config
              tags: [config]
            - name: scale-postgres
              tags: [scaling]
            - name: upgrade-postgres
              tags: [version, upgrade]
          default: update-postgres-config  # Fallback
```

**Orchestration Engine Behavior:**

1. **Polling**: Engine polls for resources with `state='requested'` and `workflow_execution_id=NULL`
2. **Operation Detection**: Reads `desired_operation` field (defaults to `create`)
3. **Workflow Resolution**:
   - Check `workflow_override` for explicit selection
   - Otherwise, resolve via `GetWorkflowForOperation(resourceType, operation, tags)`
4. **Execution**: Execute selected workflow
5. **State Transition**: Update resource state based on operation:
   - CREATE: `requested` → `provisioning` → `active`
   - UPDATE: `active` → `updating` → `active`
   - DELETE: `active` → `terminating` → `terminated`

**Example Workflows:**

See reference implementations:
- **database-team**: [provision-postgres.yaml](providers/database-team/workflows/provision-postgres.yaml), [update-postgres.yaml](providers/database-team/workflows/update-postgres.yaml), [delete-postgres.yaml](providers/database-team/workflows/delete-postgres.yaml)
- **container-team**: [provision-namespace.yaml](providers/container-team/workflows/provision-namespace.yaml), [delete-namespace.yaml](providers/container-team/workflows/delete-namespace.yaml)

### Provider Validation

**Conflict Detection:**
Innominatus validates at startup that no two providers claim the same resource type:

```bash
# If two providers both declare capability for 'postgres':
ERROR: Capability conflict detected: resource type 'postgres' claimed by multiple providers:
  - database-team
  - backup-team
```

**Resolution Logic:**
- Each resource type can only be handled by ONE provider
- Aliases allowed (e.g., 'postgres' and 'postgresql' both → database-team)
- First provisioner workflow in provider manifest is used
- Unknown resource types error gracefully

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

## Common Pitfalls & Solutions

### Provider Capability Conflicts

**Problem:** Two providers claim the same resource type
```
ERROR: Capability conflict: resource type 'postgres' claimed by multiple providers:
  - database-team
  - backup-team
```

**Solution:** Use distinct resource types or aliases
```yaml
# database-team (primary handler)
resourceTypes: [postgres, postgresql, postgres-db]

# backup-team (different capability)
resourceTypes: [postgres-backup, db-backup]
```

**Prevention:** Run provider validation at startup (already implemented in loader.go:98)

### Workflow Step Executor Selection

**Problem:** Wrong executor chosen for workflow step type

**Common causes:**
- Typo in step type (`terraforms` instead of `terraform`)
- Executor not registered in executor registry
- Missing executor binary (terraform, kubectl, ansible)

**Solution:**
```go
// Check registered executors: internal/workflow/executor.go:1300-1685
// Executors are registered as functions in the stepExecutors map

// Modern executors (in stepExecutors registry):
e.stepExecutors["terraform"] = func(ctx context.Context, step types.Step, appName string, execID int64) error { /*...*/ }
e.stepExecutors["kubernetes"] = func(ctx context.Context, step types.Step, appName string, execID int64) error { /*...*/ }
e.stepExecutors["policy"] = func(ctx context.Context, step types.Step, appName string, execID int64) error { /*...*/ }
e.stepExecutors["resource-provisioning"] = func(ctx context.Context, step types.Step, appName string, execID int64) error { /*...*/ }
e.stepExecutors["terraform-generate"] = func(ctx context.Context, step types.Step, appName string, execID int64) error { /*...*/ }
// ... and more (security, cost-analysis, tagging, database-migration, vault-setup, monitoring, validation)

// Legacy executors (fallback via runStepWithSpinner in workflow.go):
// - ansible, gitea-repo, argocd-app
// These will be migrated to stepExecutors in future releases
```

**Available Step Types:**
- `terraform` - Terraform operations (init, plan, apply, destroy)
- `terraform-generate` - Generate Terraform code from templates
- `kubernetes` - Kubernetes operations (apply, delete, create-namespace, get)
- `ansible` - Ansible playbook execution
- `policy` - Custom shell script execution with validation
- `gitea-repo` - Gitea repository management
- `argocd-app` - ArgoCD application management
- `resource-provisioning` - Automatic resource provisioning
- `security` - Security scanning
- `cost-analysis` - Cost estimation
- `tagging` - Resource tagging
- `database-migration` - Database migrations
- `vault-setup` - Vault configuration
- `monitoring` - Monitoring setup
- `validation` - Validation checks

**Validation:** Use `innominatus-ctl validate workflow.yaml` to catch typos

### Resource State Machine Edge Cases

**Problem:** Resource stuck in 'provisioning' state

**Causes:**
- Workflow execution failed but resource state not updated
- Orchestration engine crashed mid-execution
- workflow_execution_id set but execution doesn't exist

**Solution:**
```bash
# Check workflow execution status
innominatus-ctl workflow detail <execution-id>

# If execution failed, resource should be 'failed'
# Manual fix (admin only):
psql -c "UPDATE resources SET state='failed', error_message='Workflow execution failed' WHERE id=<resource-id>"
```

**Prevention:** Transaction boundaries in engine.go ensure atomic updates

### OIDC Auto-Detection Issues

**Problem:** Server fails to start with OIDC discovery error
```
FATAL: OIDC issuer validation failed: failed to fetch discovery document
```

**Common causes:**
- OIDC_ISSUER URL incorrect (missing /realms/production for Keycloak)
- Network unreachable (firewall, DNS)
- OIDC provider not running
- TLS certificate issues

**Solution:**
```bash
# 1. Verify discovery endpoint manually
curl https://keycloak.example.com/realms/production/.well-known/openid-configuration

# 2. Check network connectivity
ping keycloak.example.com

# 3. Disable OIDC for troubleshooting
export OIDC_ENABLED=false
export AUTH_TYPE=file
./innominatus

# 4. For self-signed certs (development only)
export OIDC_SKIP_TLS_VERIFY=true  # If implemented
```

### Graph Relationship Integrity

**Problem:** Graph edges created without nodes existing

**Cause:** Creating edges before nodes, or nodes not committed to DB

**Solution:**
```go
// Always create nodes first, then edges
tx := db.Begin()

// 1. Create nodes
specNode := graph.CreateNode(tx, "spec", specName, metadata)
resourceNode := graph.CreateNode(tx, "resource", resourceID, metadata)

// 2. Commit nodes
tx.Commit()

// 3. Create edges (after nodes exist)
tx = db.Begin()
graph.CreateEdge(tx, specNode.ID, resourceNode.ID, "contains", nil)
tx.Commit()
```

**Validation:** Foreign key constraints prevent orphaned edges (migrations/001_create_graph_tables.sql)

### Database Migration Ordering

**Problem:** Migration fails because dependent migration hasn't run
```
ERROR: column "workflow_execution_id" does not exist
```

**Cause:** Migrations run out of order, or migration file number skipped

**Solution:**
```bash
# Check migration status
psql -c "SELECT version, dirty FROM schema_migrations ORDER BY version"

# Migrations must be sequential: 001, 002, 003, ...
# NOT: 001, 002, 005 (skipping 003, 004)

# Fix: Rename migrations to be sequential
mv 010_add_field.sql 003_add_field.sql
```

**Prevention:** Use sequential numbering, commit migrations before code using them

### Terraform State Management

**Problem:** Terraform state conflicts between workflow executions

**Cause:** Multiple workflows using same Terraform working directory

**Solution:**
```go
// Each workflow execution gets unique working directory
workingDir := filepath.Join("/tmp/innominatus/workflows", executionID, "terraform")
os.MkdirAll(workingDir, 0755)

// State file isolated per execution
// /tmp/innominatus/workflows/123/terraform/terraform.tfstate
// /tmp/innominatus/workflows/124/terraform/terraform.tfstate
```

**Already implemented in:** internal/workflow/executors/terraform.go

### Kubernetes Context Switching

**Problem:** Wrong Kubernetes cluster targeted by workflow

**Cause:** Multiple kubeconfig contexts, workflow uses default instead of specified

**Solution:**
```yaml
# Specify context in workflow step
- name: create-namespace
  type: kubernetes
  config:
    operation: create-namespace
    namespace: my-app
    context: production-cluster  # Explicit context
```

**Implementation:**
```go
// internal/workflow/executors/kubernetes.go
config := clientcmd.BuildConfigFromKubeconfigGetter("", func() (*clientcmdapi.Config, error) {
    return clientcmd.LoadFromFile(kubeconfigPath)
})
if context := step.Config["context"]; context != "" {
    config.CurrentContext = context.(string)
}
```

### API Authentication Token Expiry

**Problem:** CLI commands fail with "token expired" after period of inactivity

**Cause:** OIDC access token has short lifespan (default: 5 minutes in Keycloak)

**Solution:**
```bash
# 1. Re-authenticate
innominatus-ctl login  # If implemented

# 2. Use API key instead (doesn't expire)
# Web UI → Profile → Generate API Key
export INNOMINATUS_API_KEY=inn_xyz...
innominatus-ctl list

# 3. Increase token lifespan in Keycloak (admin)
# Keycloak → Realm → Tokens → Access Token Lifespan: 1 hour
```

### WebSocket Connection Handling

**Problem:** Workflow log streaming disconnects unexpectedly

**Cause:** Load balancer timeout, network interruption, server restart

**Solution:**
```javascript
// web-ui/src/lib/api.ts - Implement reconnection logic
const connectWebSocket = (workflowId) => {
  let ws = new WebSocket(`ws://localhost:8081/ws/workflows/${workflowId}/logs`);

  ws.onclose = () => {
    // Reconnect after 5 seconds
    setTimeout(() => connectWebSocket(workflowId), 5000);
  };

  return ws;
};
```

**Prevention:** Server-side ping/pong heartbeat (already implemented)

### Provider Hot-Reload Race Conditions

**Problem:** Provider workflows executed during reload, causing inconsistent state

**Cause:** Orchestration engine picks up resource while provider registry is reloading

**Solution:**
```go
// internal/providers/loader.go
func (l *Loader) Reload() error {
    l.mu.Lock()
    defer l.mu.Unlock()

    // Stop orchestration engine
    l.engine.Pause()
    defer l.engine.Resume()

    // Reload providers
    newProviders, err := l.loadProviders()
    if err != nil {
        return err
    }

    // Validate no conflicts
    if err := l.validateCapabilities(newProviders); err != nil {
        return err  // Don't update if validation fails
    }

    // Atomic swap
    l.providers = newProviders
    return nil
}
```

**Best practice:** Reload providers during maintenance window

---

**See also:**
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Detailed troubleshooting guide
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - System architecture deep dive

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

## Architecture Components

### Orchestration Engine (internal/orchestration/engine.go)
- **Background polling**: Queries for resources with state='requested' and workflow_execution_id=NULL
- **Poll interval**: 5 seconds (configurable)
- **Batch size**: 100 resources per poll
- **Workflow loading**: Loads YAML workflows from provider directories
- **Graph updates**: Creates complete dependency graph with provider nodes
- **Error handling**: Updates resource state to 'failed' on errors

### Resolver (internal/orchestration/resolver.go)
- **Provider matching**: Maps resource types to providers via capabilities
- **Conflict detection**: Validates no duplicate capability claims at startup
- **Workflow selection**: Returns first provisioner workflow from matched provider
- **Error cases**: Unknown resource types, multiple providers claiming same type

### Provider Registry (internal/providers/registry.go)
- **Provider loading**: From filesystem or Git repositories
- **Validation**: Schema validation and compatibility checks
- **Hot-reload**: Supports dynamic provider updates (Git sources)
- **SDK**: Type-safe Go SDK for provider definitions (pkg/sdk/provider.go)

### Resource Lifecycle States
```
requested → provisioning → active
    ↓
  failed (with error details)
```

### Graph Structure (innominatus-graph SDK)
**Node Types:**
- `spec`: Score specification
- `resource`: Individual resource instance
- `provider`: Provider that handles resource
- `workflow`: Workflow execution

**Edge Types:**
- `requires`: Resource requires provider, workflow depends on resource
- `contains`: Spec contains resources

## AI Assistant Integration (MCP)

innominatus provides an MCP (Model Context Protocol) server that enables Claude AI to interact with the platform conversationally.

### MCP Server

**Location:** `mcp-server-innominatus/`

**What it provides:**
- 10 tools for Claude AI to query and control innominatus
- Real-time access to providers, workflows, resources, and specs
- Conversational platform management

**Tools available:**
1. `list_golden_paths` - List all golden path workflows
2. `list_providers` - List platform providers and capabilities
3. `get_provider_details` - Get detailed provider information
4. `execute_workflow` - Execute a workflow
5. `get_workflow_status` - Check workflow execution status
6. `list_workflow_executions` - List recent executions
7. `list_resources` - List provisioned resources
8. `get_resource_details` - Get resource details
9. `list_specs` - List deployed Score specifications
10. `submit_spec` - Deploy a new Score specification

### Installation

**1. Build MCP server:**
```bash
cd mcp-server-innominatus
npm install
npm run build
```

**2. Generate API token:**
- Open Web UI → Profile → Generate API Key
- Copy token (starts with `inn_...`)

**3. Configure Claude Desktop:**

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "innominatus": {
      "command": "node",
      "args": ["/path/to/innominatus/mcp-server-innominatus/build/index.js"],
      "env": {
        "INNOMINATUS_API_BASE": "http://localhost:8081",
        "INNOMINATUS_API_TOKEN": "your-api-token-here"
      }
    }
  }
}
```

**4. Restart Claude Desktop**

### Usage Examples

**In Claude AI Chat:**

```
User: "What golden paths are available?"
Claude: [Uses list_golden_paths tool]
        Lists: onboard-dev-team, provision-postgres, etc.

User: "Execute the onboard-dev-team workflow for platform-team"
Claude: [Uses execute_workflow tool]
        Returns: Execution ID 123, status: running

User: "Show all postgres resources"
Claude: [Uses list_resources tool with type filter]
        Lists: ecommerce-db (active), analytics-db (provisioning)
```

**See:** [mcp-server-innominatus/README.md](mcp-server-innominatus/README.md) for full documentation

---

*Updated: 2025-10-30* - Added event-driven orchestration architecture with automatic resource provisioning and MCP integration
