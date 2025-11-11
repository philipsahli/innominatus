# Container Team Provider

Complete GitOps deployment flow: creates namespace, Git repository, generates Kubernetes manifests, and provisions ArgoCD application.

## Overview

The container-team provider automatically provisions complete GitOps deployments when applications request container resources via Score specifications. It creates the entire infrastructure needed for GitOps-based continuous deployment.

## Capabilities

This provider handles the following resource types:

### Complete GitOps Flow
- `container` - Complete flow: namespace + Git repo + manifests + ArgoCD app
- `application` - Alias for container

### Individual Resources
- `namespace` / `kubernetes-namespace` - Individual namespace provisioning
- `gitea-repo` / `git-repository` - Individual Git repository provisioning
- `argocd-app` / `argocd-application` - Individual ArgoCD application provisioning

## Event-Driven Orchestration

When a Score spec includes a resource with `type: container` or `type: application`, the orchestration engine automatically:

1. **Detects the resource request** (polls every 5 seconds)
2. **Resolves** `container` → container-team provider
3. **Executes** the `provision-container` workflow with 5 steps:
   - Creates Kubernetes namespace
   - Creates Git repository in Gitea
   - Generates Kubernetes manifests (Deployment + Service)
   - Commits manifests to Git repo
   - Creates ArgoCD application pointing to the repo
4. **Waits** for ArgoCD to sync and application to become healthy
5. **Updates** resource state to `active`

## Prerequisites

### 1. Gitea (Git Server)

The provider requires Gitea for Git repository hosting:

```bash
# Install Gitea (example with Helm)
helm repo add gitea-charts https://dl.gitea.io/charts/
helm install gitea gitea-charts/gitea \
  --namespace git-system \
  --create-namespace \
  --set service.http.type=LoadBalancer
```

**Create organization:**
```bash
# Via Gitea UI or API
curl -X POST "http://gitea.localtest.me/api/v1/orgs" \
  -H "Authorization: token $GITEA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username": "platform"}'
```

**Set environment variable:**
```bash
export GITEA_TOKEN="your-gitea-api-token"
```

### 2. ArgoCD (GitOps Controller)

The provider requires ArgoCD for GitOps deployments:

```bash
# Install ArgoCD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

**Get admin password:**
```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

### 3. RBAC Permissions

The innominatus service account needs permissions to:
- Create Kubernetes namespaces
- Create ArgoCD applications
- Access Gitea API (via token)

Example RBAC configuration:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: innominatus-container-provisioner
rules:
  # Namespace creation
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list", "update", "patch"]

  # ArgoCD application management
  - apiGroups: ["argoproj.io"]
    resources: ["applications"]
    verbs: ["create", "get", "list", "watch", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: innominatus-container-provisioner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: innominatus-container-provisioner
subjects:
  - kind: ServiceAccount
    name: innominatus
    namespace: innominatus-system
```

## Usage

### Basic Example

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  main:
    image: nginx:latest
resources:
  app:
    type: container
```

### With Database (Multi-Provider Example)

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  main:
    image: myapp:latest
    env:
      DATABASE_URL: ${resources.db.connection_string}
resources:
  db:
    type: postgres  # Triggers database-team provider
    properties:
      version: "15"
      size: "medium"

  app:
    type: container  # Triggers container-team provider
    properties:
      namespace: production
      team_id: platform
      container_port: 8080
      service_port: 80
      gitea_org: platform
```

### With Custom Configuration

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: production-app
containers:
  api:
    image: myapi:v2.0.0
resources:
  deployment:
    type: application
    properties:
      namespace: production
      team_id: backend-team
      container_image: myapi:v2.0.0
      container_port: 3000
      service_port: 80
      service_type: LoadBalancer
      cpu_request: "500m"
      memory_request: "512Mi"
      cpu_limit: "2000m"
      memory_limit: "2Gi"
      gitea_url: "https://git.company.com"
      gitea_org: backend-team
      argocd_namespace: argocd
      argocd_project: production
      sync_policy: automated
```

## Workflow Parameters

The `provision-container` workflow accepts the following parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| app_name | string | ✓ | - | Application name |
| namespace_name | string | ✗ | app_name | Kubernetes namespace |
| team_id | string | ✓ | - | Team identifier |
| container_image | string | ✓ | - | Container image (e.g., nginx:latest) |
| container_env | object | ✗ | {} | Environment variables |
| container_port | number | ✗ | 8080 | Container port to expose |
| service_type | string | ✗ | ClusterIP | Service type (ClusterIP, NodePort, LoadBalancer) |
| service_port | number | ✗ | 80 | Service port |
| cpu_request | string | ✗ | 100m | CPU request |
| memory_request | string | ✗ | 128Mi | Memory request |
| cpu_limit | string | ✗ | 500m | CPU limit |
| memory_limit | string | ✗ | 512Mi | Memory limit |
| gitea_url | string | ✗ | http://gitea.localtest.me | Gitea base URL |
| gitea_org | string | ✗ | platform | Gitea organization |
| gitea_token | string | ✗ | $GITEA_TOKEN | Gitea API token |
| argocd_namespace | string | ✗ | argocd | ArgoCD installation namespace |
| argocd_project | string | ✗ | default | ArgoCD project name |
| sync_policy | string | ✗ | automated | ArgoCD sync policy |

## Workflow Steps

### 1. create-namespace
Creates a Kubernetes namespace with labels for app, team, and management tracking.

### 2. create-git-repo
Creates a Git repository in Gitea organization using the Gitea API. Idempotent - checks if repository exists before creating.

**Output:**
- repo_url: Repository web URL
- clone_url: Git clone URL
- repo_name: Repository name
- org_name: Organization name

### 3. generate-manifests
Generates Kubernetes manifests and commits them to the Git repository:
- **Deployment manifest** - with container image, ports, resources from Score spec
- **Service manifest** - with service type and port configuration

Uses bash script to:
- Clone the repository (with token authentication)
- Create manifests/ directory
- Generate Deployment and Service YAML files
- Commit and push to main branch

**Output:**
- manifest_path: Path in repo (manifests)
- commit_sha: Git commit SHA
- deployment_file: Deployment manifest filename
- service_file: Service manifest filename

### 4. create-argocd-app
Creates an ArgoCD Application CR that points to the Git repository's manifests directory.

Configuration:
- Source: Git repository (from step 2)
- Path: manifests (from step 3)
- Destination: Target namespace (from step 1)
- Sync policy: Automated with self-heal and prune

### 5. wait-for-sync
Waits for the ArgoCD application to sync and become healthy:
- Polls every 10 seconds
- Maximum 60 attempts (10 minutes timeout)
- Checks sync status and health status
- Succeeds when both are "Synced" and "Healthy"

## Outputs

The workflow provides the following outputs:

| Output | Example | Description |
|--------|---------|-------------|
| namespace | production | Kubernetes namespace |
| app_name | my-app | Application name |
| repo_url | http://gitea.localtest.me/platform/my-app | Repository web URL |
| clone_url | http://gitea.localtest.me/platform/my-app.git | Git clone URL |
| manifest_path | manifests | Path to manifests in repo |
| commit_sha | abc123... | Git commit SHA |
| argocd_app | my-app | ArgoCD application name |
| deployment_status | deployed | Deployment status |

## Testing

### Unit Tests

```bash
cd providers/container-team
go test -v
```

Tests include:
- Provider YAML structure validation
- Provider loading with SDK
- Workflow file existence
- Capability declarations (container, application, namespace, etc.)
- Workflow step validation

### Integration Tests

```bash
go test innominatus/internal/orchestration -v -run TestContainerTeamProviderIntegration
```

Tests include:
- Provider capability checks for all resource types
- Provisioner workflow selection
- Resolver matching:
  - `container` → container-team provider
  - `application` → container-team provider (alias)

### Manual Testing

1. **Set up environment:**
```bash
# Set environment variables
export DB_NAME=idp_orchestrator2
export GITEA_TOKEN="your-gitea-token"

# Start innominatus
./innominatus
```

2. **Submit Score spec:**
```bash
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @providers/container-team/examples/score-container-with-database.yaml
```

3. **Verify automatic provisioning:**
```bash
# Check resource status
curl http://localhost:8081/api/resources | jq '.[] | select(.resource_type=="container")'

# Verify namespace created
kubectl get namespace production

# Verify Git repository created
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/repos/platform/my-app

# Verify manifests in repo
git clone http://gitea.localtest.me/platform/my-app.git
cd my-app
cat manifests/deployment.yaml
cat manifests/service.yaml

# Verify ArgoCD app created
kubectl get application my-app -n argocd

# Check ArgoCD sync status
argocd app get my-app
```

## Architecture

### Event-Driven GitOps Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Developer submits Score spec                        │
│    resources.app.type = "container"                    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 2. API creates resource instance                       │
│    state = "requested", workflow_execution_id = NULL   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Orchestration engine detects (every 5s)             │
│    Polls for: state='requested' AND workflow_id=NULL   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Resolver matches                                     │
│    "container" → container-team provider               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Execute provision-container workflow                │
│    Step 1: Create namespace                            │
│    Step 2: Create Git repo in Gitea                    │
│    Step 3: Generate & commit K8s manifests             │
│    Step 4: Create ArgoCD application                   │
│    Step 5: Wait for sync & healthy                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Resource becomes active                             │
│    state = "active", application deployed via GitOps   │
└─────────────────────────────────────────────────────────┘
```

### Multi-Provider Orchestration

When Score spec requests both `postgres` (database-team) and `container` (container-team):

```
Score Spec Submitted
       │
       ├─→ Resource: db (type: postgres)
       │   ├─→ Orchestration engine detects
       │   ├─→ Resolver: postgres → database-team provider
       │   └─→ Execute: provision-postgres workflow
       │       └─→ PostgreSQL cluster created
       │
       └─→ Resource: app (type: container)
           ├─→ Orchestration engine detects
           ├─→ Resolver: container → container-team provider
           └─→ Execute: provision-container workflow
               └─→ Namespace + Repo + Manifests + ArgoCD created

Both workflows run independently in parallel.
Database credentials available for manifest generation.
```

## Troubleshooting

### Resource Stuck in 'requested' State

**Symptoms:** Resource created but workflow never executes

**Possible Causes:**
1. Orchestration engine not running
2. Provider not registered
3. Capability conflict with another provider

**Resolution:**
```bash
# Check provider registration
curl http://localhost:8081/api/admin/providers | jq '.[] | select(.name=="container-team")'

# Check orchestration engine logs
kubectl logs -n innominatus-system deployment/innominatus | grep orchestration

# Verify provider capabilities
go test innominatus/internal/orchestration -v -run TestAllProviderCapabilitiesValid
```

### Git Repository Creation Failed

**Symptoms:** Step 2 (create-git-repo) fails

**Possible Causes:**
1. Gitea not accessible
2. Invalid API token
3. Organization doesn't exist
4. Repository name conflict

**Resolution:**
```bash
# Test Gitea API access
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/user

# Verify organization exists
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/orgs/platform

# Check if repository already exists
curl -H "Authorization: token $GITEA_TOKEN" \
  http://gitea.localtest.me/api/v1/repos/platform/my-app
```

### Manifest Generation Failed

**Symptoms:** Step 3 (generate-manifests) fails

**Possible Causes:**
1. Git clone failed (authentication)
2. Git push failed (permissions)
3. jq not installed

**Resolution:**
```bash
# Test git clone with token
git clone http://$GITEA_TOKEN:x-oauth-basic@gitea.localtest.me/platform/my-app.git

# Verify jq is installed
which jq || apt-get install -y jq

# Check git configuration in pod
kubectl exec -it <innominatus-pod> -- git config --global --list
```

### ArgoCD Application Not Syncing

**Symptoms:** Step 5 (wait-for-sync) times out

**Possible Causes:**
1. ArgoCD can't access Git repository
2. Invalid manifest YAML
3. ArgoCD project doesn't exist
4. Resource limits preventing deployment

**Resolution:**
```bash
# Check ArgoCD application status
kubectl get application my-app -n argocd -o yaml

# View ArgoCD application details
argocd app get my-app

# Check ArgoCD controller logs
kubectl logs -n argocd deployment/argocd-application-controller

# Manually sync
argocd app sync my-app
```

## Configuration

### Provider Registration

Add to `admin-config.yaml`:

```yaml
providers:
  - name: container-team
    type: filesystem
    path: ./providers/container-team
    enabled: true
```

Or from Git repository:

```yaml
providers:
  - name: container-team
    type: git
    repository: https://github.com/myorg/container-team-provider
    ref: v1.0.0
    enabled: true
```

### Environment Variables

Required environment variables:

```bash
# Gitea Configuration
export GITEA_TOKEN="your-gitea-api-token"

# Optional overrides (defaults shown)
export GITEA_URL="http://gitea.localtest.me"
export GITEA_ORG="platform"
export ARGOCD_NAMESPACE="argocd"
```

## Examples

See `examples/` directory for complete Score specifications:
- **score-container-with-database.yaml**: Multiple examples including:
  - Production app with PostgreSQL database
  - Development environment with minimal configuration
  - Multi-container microservices application
  - Application with custom environment variables

## Individual Provisioners

The container-team provider also includes individual provisioners that can be used independently:

### provision-namespace
Creates Kubernetes namespace with resource quotas and network policies.

**Trigger:** `type: namespace` or `type: kubernetes-namespace`

### provision-gitea-repo
Creates Git repository in Gitea organization.

**Trigger:** `type: gitea-repo` or `type: git-repository`

### provision-argocd-app
Creates ArgoCD application for existing Git repository.

**Trigger:** `type: argocd-app` or `type: argocd-application`

## Contributing

To modify or extend this provider:

1. Edit `provider.yaml` to update capabilities or workflows
2. Modify `workflows/provision-container.yaml` for workflow changes
3. Update individual workflows in `workflows/` directory
4. Run tests: `go test -v`
5. Update this README with any new features or parameters

## Future Enhancements

- **ConfigMap/Secret generation** for environment variables from Score spec
- **Ingress manifest generation** for external access configuration
- **Multi-manifest support** for complex applications
- **Helm chart generation** as alternative to raw manifests
- **Kustomize support** for environment-specific overlays
- **Resource dependency management** (wait for database before deployment)

## License

Same as innominatus project (Apache License 2.0)

## Support

For issues or questions:
- Create an issue in the innominatus repository
- Check the [Orchestration Architecture](../../docs/ORCHESTRATION_ARCHITECTURE.md) documentation
- Consult [ArgoCD docs](https://argo-cd.readthedocs.io/)
- Consult [Gitea API docs](https://docs.gitea.io/en-us/api-usage/)
