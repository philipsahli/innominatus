# Orchestration Architecture

## Overview

innominatus implements **event-driven automatic resource provisioning** using a provider-based architecture. When developers deploy Score specifications requesting resources (e.g., postgres, s3, namespace), the orchestration engine automatically provisions them by matching resource types to providers via declared capabilities.

## Architecture Components

### 1. Orchestration Engine

**Location:** `internal/orchestration/engine.go`

The orchestration engine is the heart of automatic provisioning. It runs as a background goroutine polling for pending resources.

**Key Responsibilities:**
- Poll database every 5 seconds for resources with `state='requested'` and `workflow_execution_id IS NULL`
- Use resolver to match resource types to providers
- Load workflow YAML files from provider directories
- Execute workflows with resource configuration as inputs
- Update dependency graph with provider nodes
- Transition resource states through lifecycle

**Configuration:**
```go
engine := orchestration.NewEngine(
    db, registry, workflowRepo, resourceRepo,
    workflowExec, graphAdapter, providersDir,
)
engine.Start(ctx)  // Non-blocking, runs in background
```

**Polling Query:**
```sql
SELECT id, application_name, resource_name, resource_type, state,
       configuration, provider_id, workflow_execution_id,
       created_at, updated_at
FROM resource_instances
WHERE state IN ('requested', 'pending')
  AND workflow_execution_id IS NULL
ORDER BY created_at ASC
LIMIT 100
```

**Performance:**
- Poll interval: 5 seconds (configurable via `engine.pollInterval`)
- Batch size: 100 resources per poll
- Non-blocking: Runs in separate goroutine
- Graceful shutdown: Uses context cancellation

### 2. Provider Registry

**Location:** `internal/providers/registry.go`

The provider registry manages all available providers and validates capability declarations.

**Key Responsibilities:**
- Load providers from filesystem or Git repositories
- Validate provider manifests (schema, compatibility checks)
- Register providers at startup
- Provide lookup by provider name
- Support hot-reload for Git-sourced providers

**Provider Sources:**
```yaml
# admin-config.yaml
providers:
  # Filesystem (local development)
  - source: filesystem
    path: ./providers/database-team

  # Git repository (production)
  - source: git
    url: https://github.com/myorg/database-team-provider
    ref: v1.2.3  # Tag, branch, or commit SHA
```

**Validation:**
- Schema validation (required fields, valid types)
- Compatibility version checks (minCoreVersion, maxCoreVersion)
- Duplicate provider name detection
- Workflow file existence verification

### 3. Resolver

**Location:** `internal/orchestration/resolver.go`

The resolver performs capability-based matching of resource types to providers.

**Key Responsibilities:**
- Match resource types to providers via capability declarations
- Select appropriate provisioner workflow from matched provider
- Detect and report capability conflicts at startup
- Handle resource type aliases (e.g., 'postgres' and 'postgresql')

**Resolution Algorithm:**
```go
func (r *Resolver) ResolveProviderForResource(resourceType string) (*sdk.Provider, *sdk.WorkflowMetadata, error) {
    // 1. Find all providers that can handle this resource type
    allProviders := r.registry.ListProviders()
    var matchedProviders []*sdk.Provider

    for _, provider := range allProviders {
        if provider.CanProvisionResourceType(resourceType) {
            matchedProviders = append(matchedProviders, provider)
        }
    }

    // 2. Ensure exactly one provider matches
    if len(matchedProviders) == 0 {
        return nil, nil, fmt.Errorf("no provider found for resource type '%s'", resourceType)
    }
    if len(matchedProviders) > 1 {
        return nil, nil, fmt.Errorf("multiple providers claim resource type '%s'", resourceType)
    }

    // 3. Get provisioner workflow from matched provider
    provider := matchedProviders[0]
    workflow := provider.GetProvisionerWorkflow()

    return provider, workflow, nil
}
```

**Conflict Detection:**
At startup, the resolver validates that no two providers claim the same resource type:

```go
func (r *Resolver) ValidateProviders() error {
    resourceTypeMap := make(map[string]string)  // resource type → provider name

    for _, provider := range r.registry.ListProviders() {
        for _, resourceType := range provider.Capabilities.ResourceTypes {
            if existingProvider, exists := resourceTypeMap[resourceType]; exists {
                return fmt.Errorf(
                    "capability conflict: resource type '%s' claimed by both '%s' and '%s'",
                    resourceType, existingProvider, provider.Metadata.Name,
                )
            }
            resourceTypeMap[resourceType] = provider.Metadata.Name
        }
    }

    return nil
}
```

### 4. Provider SDK

**Location:** `pkg/sdk/provider.go`

The Provider SDK defines the type-safe structures for provider manifests.

**Key Structures:**

```go
type Provider struct {
    APIVersion string           `yaml:"apiVersion"`
    Kind       string           `yaml:"kind"`
    Metadata   ProviderMetadata `yaml:"metadata"`
    Capabilities ProviderCapabilities `yaml:"capabilities"`
    Compatibility Compatibility `yaml:"compatibility,omitempty"`
    Workflows  []WorkflowMetadata `yaml:"workflows"`
}

type ProviderCapabilities struct {
    ResourceTypes []string `yaml:"resourceTypes,omitempty"`
}

type WorkflowMetadata struct {
    Name        string   `yaml:"name"`
    File        string   `yaml:"file"`
    Description string   `yaml:"description,omitempty"`
    Category    string   `yaml:"category,omitempty"`  // "provisioner" or "goldenpath"
    Tags        []string `yaml:"tags,omitempty"`
}
```

**Methods:**

```go
// Check if provider can handle a resource type
func (p *Provider) CanProvisionResourceType(resourceType string) bool {
    for _, rt := range p.Capabilities.ResourceTypes {
        if rt == resourceType {
            return true
        }
    }
    return false
}

// Get the first provisioner workflow (auto-triggered)
func (p *Provider) GetProvisionerWorkflow() *WorkflowMetadata {
    for i := range p.Workflows {
        if p.Workflows[i].Category == "provisioner" || p.Workflows[i].Category == "" {
            return &p.Workflows[i]
        }
    }
    return nil
}
```

### 5. Graph Storage

**Location:** `internal/graph/graph.go` + `innominatus-graph` SDK

The graph tracks complete dependency relationships between specs, resources, providers, and workflows.

**Node Types:**
```go
const (
    NodeTypeSpec     NodeType = "spec"      // Score specification
    NodeTypeResource NodeType = "resource"  // Resource instance
    NodeTypeProvider NodeType = "provider"  // Provider that handles resource
    NodeTypeWorkflow NodeType = "workflow"  // Workflow execution
)
```

**Edge Types:**
```go
const (
    EdgeTypeRequires  = "requires"  // Resource requires provider, workflow depends on resource
    EdgeTypeContains  = "contains"  // Spec contains resources
)
```

**Graph Update Flow:**
When a resource is provisioned, the engine creates:

```go
// 1. Provider node
providerNode := &graphSDK.Node{
    ID:   fmt.Sprintf("provider:%s", provider.Metadata.Name),
    Type: "provider",
    Metadata: map[string]interface{}{
        "name":    provider.Metadata.Name,
        "version": provider.Metadata.Version,
    },
}

// 2. Resource → Provider edge
resourceToProviderEdge := &graphSDK.Edge{
    FromNodeID: resourceNodeID,
    ToNodeID:   providerNodeID,
    Type:       "requires",
}

// 3. Provider → Workflow edge
providerToWorkflowEdge := &graphSDK.Edge{
    FromNodeID: providerNodeID,
    ToNodeID:   workflowNodeID,
    Type:       "executes",
}
```

**Query Examples:**
```go
// Find all resources using a provider
resources := graph.GetResourcesByProvider("database-team")

// Get complete dependency chain
chain := graph.GetDependencyChain("spec:my-app")
// Returns: spec:my-app → resource:my-db → provider:database-team → workflow:provision-postgres
```

## Resource Lifecycle

### State Machine

```
requested → provisioning → active
    ↓
  failed (with error details)
```

**State Transitions:**

1. **requested**: Resource created from Score spec, waiting for provisioning
   - Created by: API handlers when processing Score specs
   - Triggers: Orchestration engine polling

2. **provisioning**: Workflow execution in progress
   - Set by: Orchestration engine when workflow starts
   - Duration: Depends on workflow complexity (typically 1-10 minutes)

3. **active**: Resource successfully provisioned and ready
   - Set by: Orchestration engine when workflow completes successfully
   - Final state for successful provisioning

4. **failed**: Provisioning failed with error
   - Set by: Orchestration engine on workflow failure or resolution errors
   - Includes error message and stack trace

### Resource Instance Schema

```sql
CREATE TABLE resource_instances (
    id UUID PRIMARY KEY,
    application_name VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255) NOT NULL,  -- Used for provider resolution
    state VARCHAR(50) NOT NULL,           -- Lifecycle state
    configuration JSONB,                  -- Resource properties from Score spec
    provider_id VARCHAR(255),             -- Matched provider name
    workflow_execution_id UUID,           -- FK to workflow_executions
    error_message TEXT,                   -- Error details if failed
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(application_name, resource_name)
);
```

## Event-Driven Provisioning Flow

### Complete Flow Example

**Step 1: Developer Submits Score Spec**

```yaml
# score.yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
containers:
  main:
    image: myapp:latest
resources:
  db:
    type: postgres  # ← This triggers automatic provisioning
    properties:
      version: "15"
      size: "medium"
      replicas: 2
```

```bash
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @score.yaml
```

**Step 2: API Handler Creates Resource**

```go
// internal/server/handlers.go
resource := &database.ResourceInstance{
    ApplicationName: "my-app",
    ResourceName:    "db",
    ResourceType:    "postgres",  // ← Used for matching
    State:          "requested",  // ← Initial state
    Configuration: map[string]interface{}{
        "version":  "15",
        "size":     "medium",
        "replicas": 2,
    },
}
db.Create(&resource)
```

**Step 3: Orchestration Engine Detects Resource**

After 0-5 seconds (poll interval), the engine finds the pending resource:

```go
// internal/orchestration/engine.go (poll method)
rows, err := e.db.Query(ctx, `
    SELECT id, application_name, resource_name, resource_type, state, configuration
    FROM resource_instances
    WHERE state IN ('requested', 'pending')
      AND workflow_execution_id IS NULL
    LIMIT 100
`)

// Found resource: id=uuid, resource_type=postgres, state=requested
```

**Step 4: Resolver Matches Provider**

```go
provider, workflow, err := e.resolver.ResolveProviderForResource("postgres")
// Result:
//   provider.Metadata.Name = "database-team"
//   workflow.Name = "provision-postgres"
//   workflow.File = "./workflows/provision-postgres.yaml"
```

**Step 5: Engine Loads Workflow**

```go
workflowPath := filepath.Join(
    e.providersDir,
    "database-team",
    "workflows",
    "provision-postgres.yaml",
)
workflowDef, err := e.loadWorkflowFromProvider(provider, workflow)
// Parses YAML workflow definition
```

**Step 6: Workflow Execution**

```go
// Build inputs from resource configuration
inputs := e.buildWorkflowInputs(resource, workflowDef)
// inputs = {
//   "app_name": "my-app",
//   "resource_name": "db",
//   "resource_type": "postgres",
//   "version": "15",
//   "size": "medium",
//   "replicas": "2",
// }

// Execute workflow
execution, err := e.workflowExec.ExecuteWorkflow(ctx, workflowDef, inputs)

// Update resource with workflow execution ID
resource.WorkflowExecutionID = &execution.ID
resource.State = "provisioning"
e.resourceRepo.Update(resource)
```

**Step 7: Graph Update**

```go
// Create provider node
providerNode := &graphSDK.Node{
    ID:   "provider:database-team",
    Type: "provider",
}
e.graphAdapter.AddNode(ctx, providerNode)

// Create edges
e.graphAdapter.AddEdge(ctx, &graphSDK.Edge{
    FromNodeID: "resource:my-app:db",
    ToNodeID:   "provider:database-team",
    Type:       "requires",
})

e.graphAdapter.AddEdge(ctx, &graphSDK.Edge{
    FromNodeID: "provider:database-team",
    ToNodeID:   fmt.Sprintf("workflow:%s", execution.ID),
    Type:       "executes",
})
```

**Step 8: Workflow Completes**

The workflow executor runs the workflow steps (create-postgres-cluster, wait-for-database, get-credentials) and updates the resource state when complete:

```go
resource.State = "active"  // or "failed" if error
e.resourceRepo.Update(resource)
```

**Final State:**

```
Graph: spec:my-app → resource:my-app:db → provider:database-team → workflow:uuid
Resource: state=active, provider_id=database-team, workflow_execution_id=uuid
PostgreSQL: Cluster running with credentials stored in K8s secret
```

## Provider Implementation Guide

### Creating a New Provider

**1. Directory Structure:**

```
my-provider/
├── provider.yaml                      # Provider manifest
└── workflows/
    ├── provision-resource.yaml        # Provisioner workflow
    └── manage-lifecycle.yaml          # Optional: lifecycle management
```

**2. Provider Manifest:**

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: my-provider
  version: 1.0.0
  category: infrastructure  # or "service"
  description: Provision custom resources

# Declare which resource types this provider handles
capabilities:
  resourceTypes:
    - my-resource         # Primary type
    - my-resource-alias   # Alias

compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0

workflows:
  - name: provision-my-resource
    file: ./workflows/provision-resource.yaml
    description: Provision my-resource instance
    category: provisioner  # ← Auto-triggered by orchestration engine
    tags: [infrastructure, custom]
```

**3. Provisioner Workflow:**

```yaml
apiVersion: innominatus.io/v1alpha1
kind: Workflow
metadata:
  name: provision-my-resource
  description: Provision my-resource instance

# Parameters from resource configuration
parameters:
  - name: resource_name
    type: string
    required: true

  - name: size
    type: string
    required: false
    default: "small"

steps:
  - name: create-resource
    type: kubernetes
    config:
      operation: apply
      manifest: |
        apiVersion: custom.io/v1
        kind: MyResource
        metadata:
          name: {{ .parameters.resource_name }}
        spec:
          size: {{ .parameters.size }}

  - name: wait-for-ready
    type: policy
    config:
      command: |
        #!/bin/bash
        kubectl wait --for=condition=Ready myresource/{{ .parameters.resource_name }} --timeout=300s

outputs:
  resource_name: "{{ .parameters.resource_name }}"
  endpoint: "{{ .steps.create-resource.output.endpoint }}"
```

**4. Register Provider:**

```yaml
# admin-config.yaml
providers:
  - source: git
    url: https://github.com/myorg/my-provider
    ref: v1.0.0
```

**5. Use Provider:**

```yaml
# score.yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
resources:
  custom:
    type: my-resource  # ← Automatically triggers my-provider
    properties:
      size: "large"
```

### Provider Best Practices

**1. Capability Naming:**
- Use descriptive, lowercase names
- Provide common aliases (e.g., 'postgres' + 'postgresql')
- Avoid overlapping with other providers
- Document supported resource types clearly

**2. Workflow Design:**
- Make idempotent (can run multiple times safely)
- Include wait/polling steps for async operations
- Output all connection details (host, port, credentials)
- Handle errors gracefully with clear messages
- Set reasonable timeouts (don't run forever)

**3. Error Handling:**
```yaml
steps:
  - name: create-resource
    type: kubernetes
    config:
      operation: apply
      manifest: |
        ...
    on_error:
      action: rollback  # or "continue" or "fail"
      rollback_steps:
        - name: cleanup
          type: kubernetes
          config:
            operation: delete
            ...
```

**4. Testing:**
```bash
# Unit tests
cd providers/my-provider
go test -v

# Integration tests
go test innominatus/internal/orchestration -v -run TestMyProviderIntegration

# Manual testing
./innominatus-ctl validate providers/my-provider/provider.yaml
```

## Monitoring and Observability

### Metrics

The orchestration engine exposes Prometheus metrics:

```
# Pending resources waiting for provisioning
innominatus_orchestration_pending_resources{resource_type="postgres"} 5

# Provisioning duration histogram
innominatus_orchestration_provisioning_duration_seconds{provider="database-team",resource_type="postgres"} 120.5

# Provisioning failures
innominatus_orchestration_provisioning_failures_total{provider="database-team",resource_type="postgres",error="timeout"} 2
```

### Logging

Structured logging with context:

```json
{
  "level": "info",
  "msg": "Resource provisioning started",
  "resource_id": "uuid",
  "resource_type": "postgres",
  "provider": "database-team",
  "workflow": "provision-postgres"
}
```

### Health Checks

```bash
# Orchestration engine health
curl http://localhost:8081/health
# Returns: {"status":"ok","orchestration_engine":"running"}

# Provider validation
curl http://localhost:8081/api/admin/providers
# Returns list of registered providers with capability conflicts highlighted
```

## Troubleshooting

### Resource Stuck in 'requested' State

**Symptoms:**
- Resource created but never transitions to 'provisioning'
- No workflow execution created

**Causes:**
1. No provider registered for resource type
2. Provider capability conflict (multiple providers claim same type)
3. Orchestration engine not running
4. Database connectivity issues

**Resolution:**
```bash
# Check provider registration
curl http://localhost:8081/api/admin/providers

# Check engine logs
kubectl logs -n innominatus-system deployment/innominatus | grep orchestration

# Validate provider capabilities
go test innominatus/internal/orchestration -v -run TestAllProviderCapabilitiesValid
```

### Capability Conflict

**Error Message:**
```
ERROR: Capability conflict detected: resource type 'postgres' claimed by multiple providers:
  - database-team
  - backup-team
```

**Resolution:**
1. Remove duplicate capability declaration from one provider
2. Rename resource type in one provider (e.g., 'postgres-backup')
3. Restart innominatus to reload providers

### Workflow Loading Failures

**Symptoms:**
- Resource transitions to 'failed' with error "workflow file not found"

**Causes:**
1. Workflow file path incorrect in provider manifest
2. Provider directory not accessible
3. Git repository clone failure

**Resolution:**
```bash
# Check provider directory
ls -la providers/database-team/workflows/

# Validate workflow file exists
cat providers/database-team/workflows/provision-postgres.yaml

# Check Git clone status (for Git-sourced providers)
ls -la /tmp/innominatus-providers/
```

## Architecture Diagrams

### Provider Resolution Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Developer Submits Score Spec                             │
│    resources.db.type = "postgres"                            │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. API Handler Creates Resource                             │
│    state = "requested", workflow_execution_id = NULL        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Orchestration Engine Polls (every 5s)                    │
│    SELECT * FROM resource_instances WHERE state='requested' │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Resolver Matches Resource Type → Provider                │
│    "postgres" → database-team provider                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. Get Provisioner Workflow                                 │
│    database-team.workflows[0] (category=provisioner)        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. Load Workflow YAML                                        │
│    providers/database-team/workflows/provision-postgres.yaml│
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. Execute Workflow                                          │
│    - Create PostgreSQL CR (Zalando operator)                │
│    - Wait for cluster Running status                        │
│    - Extract credentials from K8s secret                    │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. Update Graph & Resource State                            │
│    spec → resource → provider → workflow                    │
│    state = "active"                                          │
└─────────────────────────────────────────────────────────────┘
```

### Provider Registry

```
┌────────────────────────────────────────────────────────────────┐
│                    Provider Registry                           │
│                                                                │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐│
│  │ database-team    │  │ storage-team     │  │ container-   ││
│  │ ────────────────│  │ ────────────────│  │ team         ││
│  │ postgres        │  │ s3              │  │ namespace    ││
│  │ postgresql      │  │ s3-bucket       │  │ gitea-repo   ││
│  │                 │  │ object-storage  │  │ argocd-app   ││
│  └──────────────────┘  └──────────────────┘  └──────────────┘│
│                                                                │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐│
│  │ vault-team       │  │ identity-team    │  │ observability││
│  │ ────────────────│  │ ────────────────│  │ -team        ││
│  │ vault-space     │  │ gitea-org       │  │ prometheus   ││
│  │ vault-namespace │  │ keycloak-group  │  │ loki         ││
│  │ secrets         │  │ iam-group       │  │ tempo        ││
│  └──────────────────┘  └──────────────────┘  └──────────────┘│
└────────────────────────────────────────────────────────────────┘
                              │
                              │ Resolver queries
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Resource Type Mapping                                          │
│ ─────────────────────────────────────────────────────────────│
│ "postgres"          → database-team                            │
│ "s3"                → storage-team                             │
│ "namespace"         → container-team                           │
│ "vault-space"       → vault-team                              │
│ "prometheus"        → observability-team                       │
└────────────────────────────────────────────────────────────────┘
```

## Performance Considerations

### Polling Interval Tuning

```go
// Default: 5 seconds
engine.pollInterval = 5 * time.Second

// High-throughput environments (more resources, faster detection)
engine.pollInterval = 1 * time.Second

// Low-throughput environments (fewer resources, less DB load)
engine.pollInterval = 10 * time.Second
```

### Batch Size

```sql
-- Default: 100 resources per poll
LIMIT 100

-- Adjust based on:
-- - Average workflow execution time
-- - Number of concurrent workers
-- - Database performance
```

### Concurrent Execution

The engine processes resources sequentially by default. For parallel execution:

```go
// Future enhancement: Worker pool
for i := 0; i < numWorkers; i++ {
    go engine.worker(ctx, resourcesChan)
}
```

## Future Enhancements

### 1. Parallel Resource Provisioning
- Worker pool for concurrent workflow execution
- Dependency-aware scheduling (respect resource dependencies)

### 2. Retry Logic
- Automatic retry for transient failures
- Exponential backoff
- Max retry count configuration

### 3. Resource Lifecycle Management
- Deprovisioner workflows (cleanup on deletion)
- Update workflows (modify existing resources)
- Drift detection and reconciliation

### 4. Advanced Routing
- Multi-provider support (primary + fallback)
- Load balancing across provider instances
- Geography-aware provider selection

### 5. Observability Enhancements
- OpenTelemetry tracing for end-to-end visibility
- Detailed workflow step metrics
- Resource provisioning SLOs/SLIs

---

**Architecture Version:** 1.0.0
**Last Updated:** 2025-10-30
**Author:** innominatus contributors
