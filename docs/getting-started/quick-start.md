# Quick Start

Get innominatus up and running in under 5 minutes.

## Prerequisites

- Go 1.21 or later
- PostgreSQL 15+ (optional, uses memory storage by default)
- Docker Desktop with Kubernetes (for demo environment)
- `kubectl` and `helm` (for Kubernetes deployments)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/innominatus/innominatus.git
cd innominatus
```

### 2. Build the Components

```bash
# Build the server
go build -o innominatus cmd/server/main.go

# Build the CLI
go build -o innominatus-ctl cmd/cli/main.go

# (Optional) Build the Web UI
cd web-ui && npm install && npm run build && cd ..
```

### 3. Start the Server

```bash
# Start with default settings (memory storage)
./innominatus

# Or with PostgreSQL
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=yourpassword
export DB_NAME=idp_orchestrator
./innominatus
```

The server starts on `http://localhost:8081`

**Available endpoints:**
- Web UI: `http://localhost:8081/`
- API: `http://localhost:8081/api/`
- Swagger: `http://localhost:8081/swagger`
- Health: `http://localhost:8081/health`
- Metrics: `http://localhost:8081/metrics`

## Your First Workflow

### Create a Score Specification

Create `my-app.yaml`:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-first-app

containers:
  main:
    image: nginx:latest
    ports:
      - name: http
        port: 80
        protocol: TCP

resources:
  route:
    type: route
    params:
      host: myapp.localtest.me
      port: 80

environment:
  type: kubernetes
  ttl: 1h
```

### Deploy Using Golden Path

```bash
# Deploy the application
./innominatus-ctl run deploy-app my-app.yaml
```

Output:
```
✓ Workflow 'deploy-app' started for application 'my-first-app'
📋 Executing 6 workflow steps...

[1/6] ✓ validate-spec (validation) - 0.5s
[2/6] ✓ provision-namespace (kubernetes) - 1.2s
[3/6] ✓ apply-resources (kubernetes) - 2.1s
[4/6] ✓ health-check (validation) - 3.0s
[5/6] ✓ register-app (monitoring) - 0.8s
[6/6] ✓ notify (monitoring) - 0.3s

✅ Workflow completed successfully in 7.9s
🔗 Application URL: http://myapp.localtest.me
```

### Check Application Status

```bash
# Get deployment status
./innominatus-ctl status my-first-app
```

### List Deployed Applications

```bash
# List all applications
./innominatus-ctl list
```

Output:
```
NAME            STATUS    WORKFLOWS    RESOURCES    ENVIRONMENT
my-first-app    deployed  1            2            kubernetes
```

## What Just Happened?

1. **Score Spec Parsed**: innominatus read your Score specification
2. **Golden Path Executed**: The `deploy-app` workflow orchestrated:
   - Validation of the Score spec
   - Kubernetes namespace creation
   - Resource provisioning (route/ingress)
   - Health checks
   - Application registration
3. **Application Deployed**: Your app is now running in Kubernetes

## Next Steps

### Try Different Golden Paths

```bash
# List available golden paths
./innominatus-ctl list-goldenpaths

# Create an ephemeral environment
./innominatus-ctl run ephemeral-env my-app.yaml --param ttl=2h

# Run database lifecycle workflow
./innominatus-ctl run db-lifecycle my-app.yaml
```

### Explore the Web UI

Open `http://localhost:8081` to:
- View deployed applications
- Monitor workflow executions
- Browse workflow history
- Check system health

### Use the API

```bash
# Deploy via API
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  --data-binary @my-app.yaml

# Get application status
curl http://localhost:8081/api/apps/my-first-app
```

### Demo Environment

For a complete platform experience:

```bash
# Install full demo environment (Gitea, ArgoCD, Vault, Grafana)
./innominatus-ctl demo-time

# Check demo services health
./innominatus-ctl demo-status

# Access demo services
# - Gitea: http://gitea.localtest.me (admin/admin)
# - ArgoCD: http://argocd.localtest.me (admin/argocd123)
# - Vault: http://vault.localtest.me (root token: root)
# - Grafana: http://grafana.localtest.me (admin/admin)
```

## Common Tasks

### Delete an Application

```bash
./innominatus-ctl delete my-first-app
```

### View Workflow History

```bash
# Get workflow execution history for an app
curl http://localhost:8081/api/workflows?app=my-first-app
```

### Validate a Score Spec

```bash
./innominatus-ctl validate my-app.yaml
```

## Troubleshooting

### Server Won't Start

Check if port 8081 is available:
```bash
lsof -i :8081
```

Start on different port:
```bash
PORT=8082 ./innominatus
```

### Workflow Fails

Check server logs:
```bash
# Server logs show detailed execution information
tail -f innominatus.log
```

View workflow execution details:
```bash
curl http://localhost:8081/api/workflows?app=my-first-app | jq
```

### Kubernetes Connection Issues

Ensure kubectl is configured:
```bash
kubectl config current-context
kubectl cluster-info
```

## What's Next?

- [Learn Core Concepts](concepts.md) - Understand workflows, golden paths, and resources
- [User Guide](../guides/workflows.md) - Deep dive into workflow creation
- [Examples](../examples/basic-workflow.md) - Real-world workflow examples
- [API Reference](../api/rest-api.md) - Integrate with your platform

## Getting Help

- Run `./innominatus-ctl --help` for CLI documentation
- Check the [examples](../examples/) directory for sample workflows
- Visit [GitHub Issues](https://github.com/innominatus/innominatus/issues) for support
