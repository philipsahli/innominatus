# innominatus

**Score-based platform orchestration for enterprise Internal Developer Platforms**

innominatus is a workflow orchestration component that executes multi-step deployments and infrastructure provisioning from [Score specifications](https://score.dev). Built for platform teams, it integrates seamlessly with existing IDP ecosystems through a RESTful API, enabling centralized workflow execution with full observability and audit trails.

[![codecov](https://codecov.io/github/philipsahli/idp-o/graph/badge.svg?token=757WSWZMKD)](https://codecov.io/github/philipsahli/idp-o) [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Tests](https://github.com/philipsahli/idp-o/actions/workflows/test.yml/badge.svg)](https://github.com/philipsahli/idp-o/actions/workflows/test.yml) [![Security](https://github.com/philipsahli/idp-o/actions/workflows/security.yml/badge.svg)](https://github.com/philipsahli/idp-o/actions/workflows/security.yml)

---

## ⚠️ Status & Disclaimer

**This project is early-stage and experimental.** It is provided **"AS IS"** without warranty of any kind, express or implied. Use at your own risk in non-production environments. For production deployments, thorough testing and validation are required.

Licensed under [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) – see [LICENSE](./LICENSE) for full terms.

---

## Overview

innominatus addresses a common challenge in platform engineering: orchestrating complex, multi-step workflows that span infrastructure provisioning, application deployment, and service configuration. Rather than building workflow logic into every platform tool, innominatus provides a centralized orchestration layer that:

- **Executes workflows** defined in Score specifications
- **Tracks execution state** with database persistence for full audit trails
- **Integrates via REST API** with Backstage, CNOE, custom IDPs, and CI/CD systems
- **Supports multiple tools** including Kubernetes, Terraform, Ansible, ArgoCD, and Git operations

This is an **integration component**, not a standalone platform. It's designed to be embedded into your existing IDP architecture.

---

## Features

### Core Capabilities

- **Score Specification Orchestration**: Parse, validate, and execute workflows defined in Score specs
- **Multi-Step Workflows**: Chain together Kubernetes deployments, Terraform provisioning, Ansible configuration, and GitOps operations
- **Database Persistence**: PostgreSQL-backed execution tracking with full workflow history and audit logs
- **Real-Time Status**: Monitor workflow progress with streaming updates and detailed step-level reporting
- **Error Handling**: Automatic rollback support, retry logic, and comprehensive error context

### Supported Workflow Steps

| Step Type | Description | Example Use Case |
|-----------|-------------|------------------|
| **Kubernetes** | Namespace creation, manifest generation, `kubectl apply` | Deploy containerized applications |
| **Terraform** | Infrastructure provisioning via Terraform Enterprise | Provision cloud resources, databases, networks |
| **Ansible** | Configuration management with playbook execution | Configure VMs, install dependencies |
| **Git Operations** | Repository creation, manifest commits, PR automation | GitOps workflows, config management |
| **ArgoCD** | GitOps application onboarding and sync management | Continuous deployment to Kubernetes |
| **Custom Steps** | Extensible framework for additional integrations | Platform-specific automation |

### Enterprise-Ready

- **Authentication**: OIDC/SSO integration (Keycloak, Okta, etc.) with dual authentication (sessions + API keys)
- **Authorization**: Role-based access control (RBAC) for multi-tenant environments
- **Observability**: Prometheus metrics, structured logging, distributed tracing
- **Scalability**: Horizontal pod autoscaling, database connection pooling
- **Security**: Network policies, secret management, vulnerability scanning, SHA-256 API key hashing

---

## Installation

### Prerequisites

- **Go 1.21+** for building from source
- **PostgreSQL 13+** for workflow persistence
- **Kubernetes cluster** (for deployment and workflow execution)
- **kubectl** configured with cluster access

### Build from Source

```bash
# Clone the repository
git clone https://github.com/philipsahli/idp-o.git
cd idp-o

# Build the server
go build -o innominatus cmd/server/main.go

# Build the CLI (optional, for development)
go build -o innominatus-ctl cmd/cli/main.go

# Build the web UI (optional)
cd web-ui && npm install && npm run build
```

### Docker Image

```bash
# Pull latest version
docker pull ghcr.io/philipsahli/innominatus:latest

# Run container
docker run -p 8081:8081 \
  -e DB_HOST=postgres.example.com \
  -e DB_USER=orchestrator \
  -e DB_NAME=idp_orchestrator \
  ghcr.io/philipsahli/innominatus:latest
```

### Kubernetes Deployment

```yaml
# Example Kubernetes deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: innominatus
  namespace: platform
spec:
  replicas: 3
  selector:
    matchLabels:
      app: innominatus
  template:
    metadata:
      labels:
        app: innominatus
    spec:
      containers:
      - name: server
        image: ghcr.io/philipsahli/innominatus:latest
        ports:
        - containerPort: 8081
        env:
        - name: DB_HOST
          value: "postgres.platform.svc.cluster.local"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: orchestrator-db
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: orchestrator-db
              key: password
```

---

## Quickstart

### Local Development Setup

The fastest way to explore innominatus is using the built-in demo environment, which runs everything on Docker Desktop with Kubernetes enabled.

**Prerequisites**: Docker Desktop with Kubernetes enabled

```bash
# 1. Build the CLI
go build -o innominatus-ctl cmd/cli/main.go

# 2. Install demo environment (Gitea, ArgoCD, Vault, Grafana, Prometheus)
./innominatus-ctl demo-time

# 3. Check demo status
./innominatus-ctl demo-status

# 4. Access demo services
# - Gitea: http://gitea.localtest.me (admin/admin)
# - ArgoCD: http://argocd.localtest.me (admin/argocd123)
# - Vault: http://vault.localtest.me (root token: root)
# - Grafana: http://grafana.localtest.me (admin/admin)

# 5. Clean up when done
./innominatus-ctl demo-nuke
```

### Running a Simple Workflow

```bash
# 1. Start the server
export DB_USER=postgres
export DB_NAME=idp_orchestrator
./innominatus

# OR with OIDC authentication enabled:
OIDC_ENABLED=true ./innominatus

# 2. Access Web UI and login
# Browser: http://localhost:8081
# - Without OIDC: Use admin/admin123 (default credentials)
# - With OIDC: Click "Login with Keycloak" button

# 3. Deploy a Score specification (API)
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $API_KEY" \
  --data-binary @example-score-spec.yaml

# 4. Monitor execution
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8081/api/workflows
```

**Important**: The demo environment is for **local development only**. Do not use it for production workloads.

---

## Production Setup

### Database Configuration

innominatus requires PostgreSQL for workflow persistence and audit trails.

```bash
# Environment variables
export DB_HOST=postgres.production.internal
export DB_PORT=5432
export DB_USER=orchestrator_service
export DB_PASSWORD=$(cat /secrets/db-password)
export DB_NAME=idp_orchestrator
export DB_SSLMODE=require

# Database initialization (run once)
psql -h $DB_HOST -U postgres -c "CREATE DATABASE idp_orchestrator;"
psql -h $DB_HOST -U postgres -c "CREATE USER orchestrator_service WITH PASSWORD 'secure_password';"
psql -h $DB_HOST -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE idp_orchestrator TO orchestrator_service;"
```

The server automatically creates required tables on startup.

### Authentication & Authorization

#### OIDC (OpenID Connect) Authentication

innominatus supports enterprise SSO via OIDC providers like Keycloak:

```bash
# Environment variables
export OIDC_ENABLED=true
export OIDC_ISSUER="https://keycloak.company.com/realms/production"
export OIDC_CLIENT_ID="innominatus"
export OIDC_CLIENT_SECRET="your-client-secret"
export OIDC_REDIRECT_URL="https://innominatus.company.com/auth/oidc/callback"
```

**Features:**
- SSO login via "Login with Keycloak" button in Web UI
- Session-based authentication for browser access
- API key generation for CLI/API access (database-backed for OIDC users)
- Automatic user type detection (local vs OIDC users)

**See:** [OIDC Authentication Guide](./docs/OIDC_AUTHENTICATION.md) for complete setup instructions.

#### Dual Authentication Support

innominatus supports two authentication backends:

1. **Local Users** (`users.yaml`): Admin and service accounts with file-based API keys
2. **OIDC Users** (database): Enterprise users with database-backed API keys

Both types coexist seamlessly with automatic user type detection.

#### Role-Based Access Control (RBAC)

```yaml
# admin-config.yaml
rbac:
  roles:
    - name: "platform-admin"
      permissions:
        - "workflows:*"
        - "specs:*"
        - "admin:*"

    - name: "developer"
      permissions:
        - "workflows:read"
        - "specs:create"
        - "specs:read"

    - name: "viewer"
      permissions:
        - "workflows:read"
        - "specs:read"
```

### Secrets Management

Never hardcode secrets. Use Kubernetes secrets, Vault, or your enterprise secret manager:

```yaml
# Kubernetes Secret example
apiVersion: v1
kind: Secret
metadata:
  name: orchestrator-secrets
  namespace: platform
type: Opaque
stringData:
  db-password: "postgres_password_here"
  jwt-secret: "random_jwt_secret_here"
  argocd-token: "argocd_auth_token_here"
```

### Monitoring & Observability

#### Prometheus Metrics

innominatus exposes metrics at `/metrics`:

```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'innominatus'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['platform']
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: innominatus
```

**Key metrics**:
- `workflow_executions_total` - Total workflow executions by status
- `workflow_duration_seconds` - Workflow execution duration histogram
- `workflow_step_duration_seconds` - Individual step execution time
- `api_requests_total` - HTTP API request count
- `database_connections_active` - Active database connections

#### Health Checks

```bash
# Liveness probe (server is running)
curl http://localhost:8081/health

# Readiness probe (server is ready to accept traffic)
curl http://localhost:8081/ready
```

#### Structured Logging

```json
{
  "level": "info",
  "timestamp": "2025-10-02T12:00:00Z",
  "workflow_id": "wf-12345",
  "app_name": "my-application",
  "step": "kubernetes-deploy",
  "message": "Deployment successful"
}
```

### Scaling & High Availability

#### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: innominatus-hpa
  namespace: platform
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: innominatus
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

#### Database Connection Pooling

```yaml
# admin-config.yaml
database:
  maxOpenConnections: 25
  maxIdleConnections: 10
  connectionMaxLifetime: "15m"
  connectionMaxIdleTime: "5m"
```

### Security Hardening

#### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: innominatus-network-policy
  namespace: platform
spec:
  podSelector:
    matchLabels:
      app: innominatus
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8081
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: platform
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
```

#### Security Scanning

The project includes:
- **gosec**: Go security vulnerability scanning
- **golangci-lint**: Static code analysis with security rules
- **Dependabot**: Automated dependency updates
- **Pre-commit hooks**: Enforce security checks before commits

### Operational Procedures

#### Backup & Recovery

```bash
# Backup workflows database
pg_dump -h postgres.production.internal \
  -U orchestrator_service \
  -d idp_orchestrator \
  -F c \
  -f orchestrator-backup-$(date +%Y%m%d).dump

# Restore from backup
pg_restore -h postgres.production.internal \
  -U orchestrator_service \
  -d idp_orchestrator \
  -c orchestrator-backup-20251002.dump
```

#### Troubleshooting

```bash
# Check workflow execution logs
kubectl logs -n platform deployment/innominatus -f

# Query workflow status
psql -h postgres -U orchestrator_service -d idp_orchestrator \
  -c "SELECT id, app_name, status, started_at FROM workflow_executions ORDER BY started_at DESC LIMIT 10;"

# Debug failed workflows
./innominatus-ctl status <app-name>
```

---

## Usage

### API Endpoints

innominatus provides a RESTful API for platform integration:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/specs` | POST | Deploy Score specification with workflows |
| `/api/specs` | GET | List all deployed applications |
| `/api/specs/{app}` | GET | Get application details |
| `/api/specs/{app}` | DELETE | Remove application and cleanup resources |
| `/api/workflows` | GET | List workflow executions (filter by `?app=name`) |
| `/api/workflows/{id}` | GET | Get workflow execution details and logs |
| `/api/auth/config` | GET | Get OIDC configuration (public endpoint) |
| `/auth/oidc/login` | GET | Initiate OIDC login flow |
| `/auth/oidc/callback` | GET | OIDC callback handler |
| `/api/profile` | GET | Get user profile information |
| `/api/profile/api-keys` | GET | List user's API keys (masked) |
| `/api/profile/api-keys` | POST | Generate new API key |
| `/api/profile/api-keys/{name}` | DELETE | Revoke API key |
| `/health` | GET | Server health check (liveness probe) |
| `/ready` | GET | Server readiness check |
| `/metrics` | GET | Prometheus metrics |
| `/swagger` | GET | OpenAPI documentation |

### API Examples

#### Deploy Application

```bash
# Deploy Score specification
curl -X POST http://orchestrator.company.com/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  --data-binary @score-spec.yaml

# Response
{
  "app_name": "my-application",
  "workflow_id": "wf-abc123",
  "status": "running",
  "started_at": "2025-10-02T12:00:00Z"
}
```

#### Monitor Workflows

```bash
# Get all workflows for an application
curl -H "Authorization: Bearer $PLATFORM_TOKEN" \
  "http://orchestrator.company.com/api/workflows?app=my-application"

# Get specific workflow execution
curl -H "Authorization: Bearer $PLATFORM_TOKEN" \
  "http://orchestrator.company.com/api/workflows/wf-abc123"

# Response includes step-by-step execution details
{
  "id": "wf-abc123",
  "app_name": "my-application",
  "status": "completed",
  "steps": [
    {
      "name": "Create repository",
      "status": "completed",
      "started_at": "2025-10-02T12:00:00Z",
      "completed_at": "2025-10-02T12:00:15Z"
    },
    ...
  ]
}
```

#### List Applications

```bash
# Get all deployed applications
curl -H "Authorization: Bearer $PLATFORM_TOKEN" \
  "http://orchestrator.company.com/api/specs"

# Response
{
  "my-application": {
    "metadata": {
      "name": "my-application"
    },
    "workflows": {...},
    "containers": {...}
  }
}
```

### CLI Usage (Development)

The CLI is primarily for local development and testing:

```bash
# Validate Score specification
./innominatus-ctl validate score-spec.yaml

# List deployed applications
./innominatus-ctl list

# Get application status
./innominatus-ctl status my-application

# Show admin configuration
./innominatus-ctl admin show

# List available golden paths
./innominatus-ctl list-goldenpaths

# Run a golden path workflow
./innominatus-ctl run deploy-app score-spec.yaml

# Delete application
./innominatus-ctl delete my-application
```

---

## Architecture & Integrations

### Design Principles

innominatus follows an **API-first, integration-focused** architecture:

1. **RESTful API**: All functionality exposed via HTTP endpoints
2. **Database Persistence**: PostgreSQL for workflow state and audit trails
3. **Kubernetes-Native**: Designed to run in Kubernetes, operate on Kubernetes
4. **Modular Workflows**: Pluggable step execution framework
5. **Idempotent Operations**: Safe to retry workflow steps

### Integration Patterns

#### 1. Backstage Software Catalog

```typescript
// Backstage plugin integration example
import { orchestratorApiRef } from '@internal/plugin-orchestrator';

const OrchestratorDeployButton: React.FC = () => {
  const orchestratorApi = useApi(orchestratorApiRef);

  const handleDeploy = async (scoreSpec: string) => {
    const response = await orchestratorApi.deploySpec(scoreSpec);
    console.log('Workflow started:', response.workflow_id);
  };

  return <Button onClick={handleDeploy}>Deploy</Button>;
};
```

#### 2. CNOE Platform Integration

```yaml
# CNOE workflow example
apiVersion: idpbuilder.io/v1
kind: Workflow
metadata:
  name: deploy-application
spec:
  steps:
  - name: orchestrate-deployment
    type: http
    config:
      url: "http://innominatus.platform.svc.cluster.local:8081/api/specs"
      method: POST
      headers:
        Content-Type: application/yaml
      body: ${scoreSpecification}
```

#### 3. GitOps Integration

```bash
# Webhook trigger from Git repository
# Repository: platform-config
# Path: applications/my-app/score.yaml
# On commit → trigger orchestrator

curl -X POST http://orchestrator.company.com/api/specs \
  -H "Content-Type: application/yaml" \
  -H "X-GitHub-Event: push" \
  --data-binary @applications/my-app/score.yaml
```

#### 4. CI/CD Pipeline Integration

```yaml
# GitHub Actions example
name: Deploy Application
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Deploy via Orchestrator
      run: |
        curl -X POST ${{ secrets.ORCHESTRATOR_URL }}/api/specs \
          -H "Authorization: Bearer ${{ secrets.ORCHESTRATOR_TOKEN }}" \
          -H "Content-Type: application/yaml" \
          --data-binary @score.yaml
```

### Supported Workflow Steps

Each workflow step type has specific capabilities:

#### Kubernetes Steps

```yaml
- name: "Deploy to Kubernetes"
  type: "kubernetes"
  config:
    namespace: "my-app"
    manifests: "./k8s/*.yaml"
    waitForReady: true
```

#### Terraform Steps

```yaml
- name: "Provision Infrastructure"
  type: "terraform"
  config:
    operation: "apply"
    working_dir: "./terraform"
    variables:
      environment: "production"
      region: "us-east-1"
    outputs:
      - vpc_id
      - subnet_ids
```

#### Git Operations

```yaml
- name: "Create Repository"
  type: "gitea-repo"
  config:
    repoName: "my-application"
    description: "Application repository"
    private: false
```

#### ArgoCD Integration

```yaml
- name: "Onboard to ArgoCD"
  type: "argocd-app"
  config:
    appName: "my-application"
    repoURL: "https://github.com/org/my-app"
    targetPath: "manifests/"
    syncPolicy: "auto"
```

---

## Example Score Specification

Here's a complete example showing how to define workflows in a Score specification:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: my-microservice
  version: "1.0.0"

# Define the application's workflows
workflows:
  deploy:
    steps:
    # Step 1: Create Git repository for GitOps
    - name: "Create application repository"
      type: "gitea-repo"
      repoName: "my-microservice"
      description: "Microservice application"
      private: false

    # Step 2: Provision cloud infrastructure
    - name: "Provision AWS resources"
      type: "terraform"
      config:
        operation: "apply"
        working_dir: "./terraform/aws"
        variables:
          app_name: "my-microservice"
          environment: "production"
          instance_type: "t3.medium"
        outputs:
          - vpc_id
          - database_endpoint
          - redis_endpoint

    # Step 3: Generate and commit Kubernetes manifests
    - name: "Generate Kubernetes manifests"
      type: "git-commit-manifests"
      repoName: "my-microservice"
      manifestPath: "k8s/"

    # Step 4: Deploy to Kubernetes
    - name: "Deploy to Kubernetes cluster"
      type: "kubernetes"
      config:
        namespace: "production"
        manifests: "./k8s/*.yaml"
        waitForReady: true
        timeout: "10m"

    # Step 5: Onboard to ArgoCD for GitOps
    - name: "Onboard to ArgoCD"
      type: "argocd-app"
      config:
        appName: "my-microservice"
        repoURL: "http://gitea.company.com/platform/my-microservice"
        targetPath: "k8s/"
        syncPolicy: "auto"
        namespace: "production"

# Define the application containers
containers:
  web:
    image: nginx:1.25-alpine
    variables:
      APP_NAME: "my-microservice"
      ENVIRONMENT: "production"
      DATABASE_URL: "${resources.db.connection_string}"
      REDIS_URL: "${resources.cache.endpoint}"

# Define required resources
resources:
  db:
    type: postgres
    params:
      version: "15"
      storage: "100Gi"

  cache:
    type: redis
    params:
      version: "7"
      memory: "2Gi"

  ingress:
    type: route
    params:
      host: "my-microservice.company.com"
      port: 80
```

**Workflow Execution Flow**:

1. Platform submits Score spec to `/api/specs`
2. innominatus validates the specification
3. Executes each workflow step sequentially
4. Tracks progress in database
5. Returns workflow ID for monitoring
6. Platform polls `/api/workflows/{id}` for status

---

## Contributing

We welcome contributions from the community! innominatus is designed as a focused platform component, and we want to maintain that clarity of purpose.

### How to Contribute

1. **Check existing issues** or create a new one to discuss your idea
2. **Fork the repository** and create a feature branch
3. **Make your changes** following our coding standards
4. **Write tests** for new functionality
5. **Submit a pull request** with a clear description

For detailed guidelines, see [CONTRIBUTING.md](./CONTRIBUTING.md).

### Contribution Focus Areas

- **Workflow Step Types**: New integrations (e.g., Helm, Flux, Crossplane)
- **Platform Integrations**: Backstage plugins, CNOE components
- **Observability**: Enhanced metrics, tracing, logging
- **Security**: Auth improvements, RBAC enhancements
- **Documentation**: Tutorials, examples, API guides
- **Testing**: Unit tests, integration tests, E2E tests

### Code of Conduct

This project follows a Code of Conduct to ensure a welcoming environment for all contributors. See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

---

## Documentation

- **[Score Specification](https://score.dev)** - Official Score documentation and standards
- **[OIDC Authentication Guide](./docs/OIDC_AUTHENTICATION.md)** - Enterprise SSO and API key management
- **[Terraform Enterprise Workflow](./TFE-WORKFLOW-README.md)** - TFE integration guide
- **[Development Guide](./CLAUDE.md)** - Development setup and testing
- **[CONTRIBUTING.md](./CONTRIBUTING.md)** - Contribution guidelines
- **[CHANGELOG.md](./CHANGELOG.md)** - Release notes and version history

### Additional Resources

- **API Documentation**: Available at `http://localhost:8081/swagger` when server is running
- **Monitoring Guide**: See "Production Setup > Monitoring & Observability" section
- **Security Hardening**: See "Production Setup > Security Hardening" section

---

## License

Copyright © 2024-2025 innominatus contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

---

**Built for platform teams, by platform engineers.** 🚀
