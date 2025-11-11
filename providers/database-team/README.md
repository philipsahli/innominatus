# Database Team Provider

PostgreSQL database provisioning using the [Zalando PostgreSQL Operator](https://github.com/zalando/postgres-operator).

## Overview

The database-team provider automatically provisions PostgreSQL databases when applications request them via Score specifications. It uses the Zalando PostgreSQL Operator to create highly available PostgreSQL clusters on Kubernetes.

## Capabilities

This provider handles the following resource types:
- `postgres`
- `postgresql`

When a Score spec includes a resource with `type: postgres`, the orchestration engine automatically:
1. Detects the resource request (polls every 5 seconds)
2. Resolves `postgres` → database-team provider
3. Executes the `provision-postgres` workflow
4. Creates a PostgreSQL cluster using Zalando operator
5. Waits for the cluster to be ready
6. Extracts credentials from Kubernetes secrets
7. Makes connection details available to the application

## Prerequisites

### 1. Zalando PostgreSQL Operator

The provider requires the Zalando PostgreSQL Operator to be installed in your Kubernetes cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/zalando/postgres-operator/master/manifests/postgresql-operator.yaml
```

Verify installation:

```bash
kubectl get pods -n postgres-operator
# Should show: postgres-operator-xxxxx   1/1     Running
```

### 2. RBAC Permissions

The innominatus service account needs permissions to:
- Create PostgreSQL custom resources
- Read/watch PostgreSQL status
- Read secrets in target namespaces

Example RBAC configuration:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: innominatus-database-provisioner
rules:
  - apiGroups: ["acid.zalan.do"]
    resources: ["postgresqls"]
    verbs: ["create", "get", "list", "watch", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: innominatus-database-provisioner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: innominatus-database-provisioner
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
    image: myapp:latest
    env:
      DATABASE_URL: ${resources.db.connection_string}
resources:
  db:
    type: postgres
```

### With Custom Configuration

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  main:
    image: myapp:latest
    env:
      DB_HOST: ${resources.db.host}
      DB_PORT: ${resources.db.port}
      DB_NAME: ${resources.db.database_name}
      DB_USER: ${resources.db.username}
      DB_PASSWORD: ${resources.db.password}
resources:
  db:
    type: postgres
    properties:
      version: "15"      # PostgreSQL version
      size: "medium"     # Database size (small/medium/large)
      replicas: 3        # Number of replicas
```

## Resource Sizes

The provider supports three predefined sizes:

### Small (Development)
- Storage: 5Gi
- CPU: 100m-500m
- Memory: 256Mi-512Mi
- Default replicas: 2

### Medium (Staging)
- Storage: 20Gi
- CPU: 500m-2000m
- Memory: 1Gi-2Gi
- Default replicas: 2

### Large (Production)
- Storage: 100Gi
- CPU: 2000m-4000m
- Memory: 4Gi-8Gi
- Default replicas: 3 (recommended)

## Workflow Parameters

The `provision-postgres` workflow accepts the following parameters:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| db_name | string | ✓ | - | Database cluster name |
| namespace | string | ✓ | - | Kubernetes namespace |
| team_id | string | ✓ | - | Team identifier |
| size | string | ✗ | small | Database size (small/medium/large) |
| replicas | number | ✗ | 2 | Number of replicas |
| version | string | ✗ | 15 | PostgreSQL version |

## Workflow Steps

### 1. create-postgres-cluster
Creates a PostgreSQL custom resource using Zalando operator specification. Configures:
- Number of instances (replicas)
- PostgreSQL version
- Volume size (based on size parameter)
- Resource requests/limits (CPU, memory)
- Database users (owner and app user)
- Databases

### 2. wait-for-database
Polls the PostgreSQL cluster status until it reaches `Running` state:
- Checks every 10 seconds
- Maximum 60 attempts (10 minutes timeout)
- Exits successfully when cluster is running

### 3. get-credentials
Extracts database credentials from Kubernetes secret:
- Waits for credentials secret to be created
- Retrieves username, password, host, port, database name
- Outputs connection information as JSON

## Outputs

The workflow provides the following outputs:

| Output | Example | Description |
|--------|---------|-------------|
| database_name | myapp-db | Database name |
| cluster_name | team-myapp-db | Full cluster name |
| namespace | production | Kubernetes namespace |
| host | team-myapp-db.production.svc.cluster.local | Service endpoint |
| port | 5432 | PostgreSQL port |
| credentials_secret | team-myapp-db.myapp-db-app.credentials | K8s secret name |
| username | myapp_app | Database username |
| password | xxx | Database password |
| connection_string | postgresql://user:pass@host:5432/db | Full connection URI |

## Testing

### Unit Tests

```bash
cd providers/database-team
go test -v
```

Tests include:
- Provider YAML structure validation
- Provider loading with SDK
- Workflow file existence
- Capability declarations

### Integration Tests

```bash
go test innominatus/internal/orchestration -v -run TestDatabaseTeamProviderIntegration
```

Tests include:
- Provider capability checks (postgres, postgresql)
- Provisioner workflow selection
- Resolver matching (postgres → database-team)

### Manual Testing

1. Create a Score spec with postgres resource
2. Submit to innominatus
3. Verify automatic provisioning
4. Check PostgreSQL cluster status
5. Verify credentials

```bash
# Submit Score spec
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @examples/score-postgres-app.yaml

# Check resource status
curl http://localhost:8081/api/resources | jq '.[] | select(.resource_type=="postgres")'

# Verify PostgreSQL cluster
kubectl get postgresql -n <namespace>

# Check credentials secret
kubectl get secret <team>-<db-name>.<db-name>-app.credentials -n <namespace>
```

## Troubleshooting

### Resource Stuck in 'requested' State

**Symptoms:** Resource created but workflow never executes

**Possible Causes:**
1. Orchestration engine not running
2. Provider not registered
3. Capability conflict

**Resolution:**
```bash
# Check provider registration
curl http://localhost:8081/api/admin/providers | jq '.[] | select(.name=="database-team")'

# Check orchestration engine logs
kubectl logs -n innominatus-system deployment/innominatus | grep orchestration

# Verify provider capabilities
go test innominatus/internal/orchestration -v -run TestAllProviderCapabilitiesValid
```

### PostgreSQL Cluster Not Ready

**Symptoms:** Workflow times out waiting for cluster

**Possible Causes:**
1. Zalando operator not running
2. Insufficient cluster resources
3. Storage class not available

**Resolution:**
```bash
# Check operator status
kubectl get pods -n postgres-operator

# Check PostgreSQL cluster events
kubectl describe postgresql <cluster-name> -n <namespace>

# Check operator logs
kubectl logs -n postgres-operator deployment/postgres-operator
```

### Credentials Secret Not Found

**Symptoms:** get-credentials step fails

**Possible Causes:**
1. Cluster not fully ready
2. Secret naming mismatch
3. RBAC permissions missing

**Resolution:**
```bash
# List secrets in namespace
kubectl get secrets -n <namespace> | grep <cluster-name>

# Verify secret format
kubectl get secret <cluster-name>.<db-name>-app.credentials -n <namespace> -o yaml

# Check innominatus service account permissions
kubectl auth can-i get secrets --as=system:serviceaccount:innominatus-system:innominatus -n <namespace>
```

## Configuration

### Provider Registration

Add to `admin-config.yaml`:

```yaml
providers:
  - name: database-team
    type: filesystem
    path: ./providers/database-team
    enabled: true
```

Or from Git repository:

```yaml
providers:
  - name: database-team
    type: git
    repository: https://github.com/myorg/database-team-provider
    ref: v1.0.0
    enabled: true
```

### Environment Variables

No provider-specific environment variables required. Uses standard Kubernetes configuration.

## Architecture

### Event-Driven Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. Developer submits Score spec                        │
│    resources.db.type = "postgres"                      │
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
│ 4. Resolver matches capability                         │
│    "postgres" → database-team provider                 │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 5. Workflow execution                                   │
│    - Create PostgreSQL CR (Zalando operator)           │
│    - Wait for Running status (max 10 min)              │
│    - Extract credentials from K8s secret               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│ 6. Resource becomes active                             │
│    state = "active", connection info available         │
└─────────────────────────────────────────────────────────┘
```

## Examples

See `examples/` directory for complete Score specifications:
- **score-postgres-app.yaml**: Multiple examples from basic to production

## Contributing

To modify or extend this provider:

1. Edit `provider.yaml` to update capabilities or workflows
2. Modify `workflows/provision-postgres.yaml` for workflow changes
3. Run tests: `go test -v`
4. Update this README with any new features or parameters

## License

Same as innominatus project (Apache License 2.0)

## Support

For issues or questions:
- Create an issue in the innominatus repository
- Check the [Orchestration Architecture](../../docs/ORCHESTRATION_ARCHITECTURE.md) documentation
- Consult [Zalando PostgreSQL Operator docs](https://postgres-operator.readthedocs.io/)
