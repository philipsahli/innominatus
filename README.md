# innominatus

A **Score-based platform orchestration component** designed for integration into enterprise Internal Developer Platform (IDP) ecosystems.

## Overview

innominatus provides centralized execution of multi-step workflows from [Score specifications](https://score.dev), enabling platform teams to automate complex deployment and infrastructure provisioning workflows. It's designed as an integration component for existing IDP platforms rather than a standalone solution.

## Architecture

### Integration Component Design
- **API-First**: RESTful interface for platform integration
- **Database Persistence**: Workflow execution tracking and history
- **Kubernetes-Native**: Built for container orchestration environments
- **Modular Workflows**: Pluggable step execution framework

### Enterprise Integration
innominatus is designed to integrate with popular IDP platforms:
- **Backstage**: Software catalog and developer portal integration
- **CNOE**: Cloud Native Operational Excellence platforms
- **Custom IDPs**: Any platform supporting REST API integration
- **CI/CD Systems**: Jenkins, GitLab, GitHub Actions, etc.

## Core Capabilities

### Score Specification Orchestration
- Parse and validate Score specifications with workflow definitions
- Execute multi-step workflows with proper error handling and rollback
- Database persistence for execution tracking and audit trails
- Real-time status monitoring and progress reporting

### Supported Workflow Steps
- **Kubernetes**: Namespace creation, manifest generation, `kubectl apply`
- **Terraform**: Infrastructure provisioning via TFE integration
- **Ansible**: Configuration management with playbook execution
- **Git Operations**: Repository creation, manifest commits, PR automation
- **ArgoCD**: GitOps application onboarding and sync management
- **Custom Steps**: Extensible framework for additional integrations

## Getting Started

### Prerequisites
- Go 1.21+
- Kubernetes cluster access
- PostgreSQL database (for persistence)

### Building the Components

**Build the Server:**
```bash
go build -o innominatus cmd/server/main.go
```

**Build the CLI:**
```bash
go build -o innominatus-ctl cmd/cli/main.go
```

### Running the Server

```bash
# Start with database persistence
export DB_USER=your_user
export DB_NAME=orchestrator_db
./innominatus

# Server runs on http://localhost:8081
# API Documentation: http://localhost:8081/swagger
```

### API Integration

innominatus provides REST endpoints for platform integration:

```bash
# Deploy Score specification with workflows
POST /api/specs
Content-Type: application/yaml
[Score specification with workflows]

# Monitor workflow execution
GET /api/workflows?app=my-app

# Get workflow execution details
GET /api/workflows/{id}
```

### Score Specification Example

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-application

workflows:
  deploy:
    steps:
    - name: "Create repository"
      type: "gitea-repo"
      repoName: "my-application"
      description: "Application repository"

    - name: "Generate Kubernetes manifests"
      type: "git-commit-manifests"
      repoName: "my-application"
      manifestPath: "."

    - name: "Onboard to ArgoCD"
      type: "argocd-app"
      appName: "my-application"
      repoURL: "http://gitea.example.com/platform/my-application"
      targetPath: "."
      syncPolicy: "auto"

containers:
  web:
    image: nginx:latest
    variables:
      APP_NAME: my-application

resources: {}
```

## Demo Environment

**Important**: The demo environment is provided for **demonstration and development purposes only**. Enterprise deployments should integrate with existing platform infrastructure.

### Demo Components
- Gitea (Git hosting)
- ArgoCD (GitOps)
- Vault (Secrets)
- Grafana/Prometheus (Monitoring)
- Kubernetes Dashboard

### Demo Commands
```bash
# Install demo environment (requires Docker Desktop + Kubernetes)
./innominatus-ctl demo-time

# Check demo status
./innominatus-ctl demo-status

# Remove demo environment
./innominatus-ctl demo-nuke
```

## Enterprise Usage

### Integration Patterns

**1. API Integration**
```bash
# Deploy via platform API
curl -X POST http://orchestrator.company.com/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $PLATFORM_TOKEN" \
  --data-binary @score-spec.yaml
```

**2. Backstage Plugin Integration**
```typescript
// Example Backstage plugin integration
const response = await fetch('/api/proxy/orchestrator/specs', {
  method: 'POST',
  headers: { 'Content-Type': 'application/yaml' },
  body: scoreSpecification
});
```

**3. GitOps Integration**
- Score specifications stored in Git repositories
- Platform triggers orchestrator via webhooks
- Workflow execution results update platform status

### Production Considerations
- **Authentication**: Integrate with enterprise identity providers
- **Authorization**: Role-based access control (RBAC)
- **Monitoring**: Prometheus metrics and distributed tracing
- **Scaling**: Horizontal pod autoscaling for workflow execution
- **Security**: Network policies and secret management integration

## Development

### CLI Usage (Development)
```bash
# Validate Score specification
./innominatus-ctl validate score-spec.yaml

# List deployed applications
./innominatus-ctl list

# Show admin configuration
./innominatus-ctl admin show
```

### Configuration
```yaml
# admin-config.yaml
admin:
  defaultCostCenter: "engineering"
  defaultRuntime: "kubernetes"

resourceDefinitions:
  postgres: "managed-postgres-cluster"
  redis: "redis-cluster"

policies:
  enforceBackups: true
  allowedEnvironments: ["dev", "staging", "prod"]
```

## Contributing

innominatus is designed as a focused platform component. Contributions should maintain:
- **Single Responsibility**: Score specification orchestration
- **Integration Focus**: API-first design for platform integration
- **Enterprise Ready**: Security, monitoring, and scalability considerations
- **Standard Compliance**: Follow Score specification standards

## Documentation

- [Score Specification](https://score.dev) - Official Score documentation
- [TFE Workflow Extension](./TFE-WORKFLOW-README.md) - Terraform Enterprise integration
- [Development Guide](./CLAUDE.md) - Development and testing instructions

## License

Open source - designed for integration into enterprise platforms and CNCF ecosystem components.