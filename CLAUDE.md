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
cd web-ui && npm run build
```

### Running the Components

**Start the Server:**
```bash
./innominatus
# Server runs on http://localhost:8081 by default
# Web UI: http://localhost:8081/
# API Docs: http://localhost:8081/swagger
```

**Health & Monitoring Endpoints:**
```
http://localhost:8081/health   - Liveness probe (Kubernetes health checks)
http://localhost:8081/ready    - Readiness probe (service ready for traffic)
http://localhost:8081/metrics  - Prometheus metrics (performance monitoring)
```

See [docs/HEALTH_MONITORING.md](docs/HEALTH_MONITORING.md) for detailed monitoring documentation.

**Note:** When server source code is modified, you must rebuild and restart the server for changes to take effect:
```bash
# Stop the running server (Ctrl+C)
go build -o innominatus cmd/server/main.go
./innominatus
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

### Workflow Capabilities

innominatus supports multi-step workflows with:

- **Terraform:** Infrastructure provisioning (`terraform init`, `apply`, `output`)
- **Ansible:** Configuration management (`ansible-playbook`)
- **Kubernetes:** Application deployment (namespace creation, manifest generation, `kubectl apply`)

Each workflow step runs in app and environment-specific workspaces for multi-tenant isolation.

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

---

*Created: 2025-09-13*
