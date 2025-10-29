# Provider Architecture - Unified Workflows

**Status**: Implemented (v1.0)
**Last Updated**: 2025-10-29

## Overview

Innominatus uses a **three-layer provider architecture** to enable extensibility while maintaining simplicity. All provisioning logic is defined as **YAML-based workflows**, eliminating the need for Go-based provisioners.

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                     Extension Providers                      │
│          (Product/Platform Teams - Git Repos)                │
│                                                               │
│  • Custom workflows for business domains                     │
│  • Loaded dynamically from Git with version pinning          │
│  • Hot-reload support for rapid iteration                    │
│  • Examples: ml-pipeline, ecommerce-app, analytics           │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Built-in Provider                        │
│                (Filesystem - providers/builtin/)             │
│                                                               │
│  • Standard workflows shipped with innominatus               │
│  • Infrastructure primitives (databases, caches, repos)      │
│  • Examples: postgres-cluster, redis-cache, gitea-repo       │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Core (Go Engine)                        │
│                                                               │
│  • Workflow execution engine                                 │
│  • Step executors: terraform, kubernetes, ansible, policy    │
│  • API server, authentication, database persistence          │
└─────────────────────────────────────────────────────────────┘
```

## Key Concepts

### 1. Providers Bundle Workflows

A **provider** is a collection of workflows defined in a `provider.yaml` manifest:

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: my-provider
  version: 1.0.0
  category: infrastructure  # or "service"

workflows:
  - name: postgres-cluster
    file: ./workflows/postgres.yaml
    category: provisioner

  - name: onboard-team
    file: ./workflows/onboard-team.yaml
    category: goldenpath
```

### 2. Two Workflow Types

**Provisioners** (category: `provisioner`)
- Single-resource workflows
- Create one type of resource (database, namespace, bucket, repo)
- Composable building blocks
- Example: `postgres-cluster.yaml`, `gitea-repo.yaml`

**Golden Paths** (category: `goldenpath`)
- Multi-resource orchestration workflows
- Combine multiple provisioners into end-to-end flows
- Opinionated "happy path" for common scenarios
- Example: `onboard-dev-team.yaml` (namespace + repo + ArgoCD app)

### 3. Workflow Steps

Workflows execute a series of steps using built-in step executors:

```yaml
apiVersion: innominatus.io/v1alpha1
kind: Workflow
metadata:
  name: postgres-cluster
steps:
  - name: provision-db
    type: terraform
    config:
      operation: apply
      working_dir: ./terraform/postgres

  - name: create-namespace
    type: kubernetes
    config:
      operation: apply
      manifest: |
        apiVersion: v1
        kind: Namespace
        metadata:
          name: {{ .namespace }}
```

**Available Step Executors:**
- `terraform` - Infrastructure provisioning (init, plan, apply, destroy)
- `kubernetes` - K8s resource management (apply, delete, status)
- `ansible` - Configuration management
- `policy` - Policy validation and enforcement
- `gitea-repo` - Git repository creation
- `argocd-app` - ArgoCD application creation

## Provider Categories

### Infrastructure Providers (Platform Teams)

Created by platform/infrastructure teams to provide cloud and infrastructure primitives:

**Examples:**
- AWS resources (RDS, S3, EKS)
- Azure resources (CosmosDB, Storage, AKS)
- GCP resources (CloudSQL, GCS, GKE)
- Kubernetes primitives (namespaces, RBAC, operators)
- Storage systems (Minio, Ceph)
- Databases (PostgreSQL, MySQL, Redis, MongoDB)

**Characteristics:**
- Generic, reusable infrastructure
- Cloud/platform-specific
- Technology-focused

### Service Providers (Product Teams)

Created by product teams to provide business domain resources:

**Examples:**
- ML model registries and pipelines
- Analytics dashboards
- E-commerce catalogs
- Payment processing workflows
- Content management systems

**Characteristics:**
- Business domain-specific
- Application-focused
- Compose infrastructure providers

## Usage Examples

### 1. List Available Workflows

```bash
# List all golden paths
./innominatus-ctl list-goldenpaths

# List all resources (provisioners)
./innominatus-ctl list-resources
```

### 2. Execute Golden Path

```bash
./innominatus-ctl run onboard-dev-team inputs.yaml
```

**inputs.yaml:**
```yaml
team_name: backend-team
namespace: backend-prod
git_repo: https://github.com/myorg/backend
```

### 3. Create Extension Provider

**Directory Structure:**
```
my-provider/
├── provider.yaml
└── workflows/
    ├── ml-model.yaml
    └── training-pipeline.yaml
```

**provider.yaml:**
```yaml
apiVersion: v1
kind: Provider
metadata:
  name: ml-platform
  version: 1.0.0
  category: service
  description: ML platform workflows

compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0

workflows:
  - name: ml-model-registry
    file: ./workflows/ml-model.yaml
    category: provisioner
    description: Create ML model registry
    tags: [ml, models, registry]

  - name: training-pipeline
    file: ./workflows/training-pipeline.yaml
    category: goldenpath
    description: End-to-end ML training pipeline
    tags: [ml, training, pipeline]
```

**Register in admin-config.yaml:**
```yaml
providers:
  - source: git
    url: https://github.com/myorg/ml-platform-provider
    ref: v1.0.0  # Tag, branch, or commit SHA
```

## Migration from Old Architecture

### Deprecated: Go Provisioner Interface

The Go-based `Provisioner` interface is **deprecated** and will be removed in v2.0:

```go
// DEPRECATED: Use YAML workflows instead
type Provisioner interface {
    Name() string
    Type() string
    Provision(ctx context.Context, resource *Resource) error
    Deprovision(ctx context.Context, resource *Resource) error
}
```

### Backward Compatibility

The provider loader automatically migrates old provider.yaml formats:

**Old Format (deprecated):**
```yaml
provisioners:
  - name: postgres-cluster
    type: database
    version: 1.2.0

goldenpaths:
  - name: deploy-database
    description: Deploy managed database
```

**New Format (recommended):**
```yaml
workflows:
  - name: postgres-cluster
    file: ./workflows/postgres-cluster.yaml
    category: provisioner
    version: 1.2.0

  - name: deploy-database
    file: ./workflows/deploy-database.yaml
    category: goldenpath
```

The loader's `migrateProvider()` function automatically converts old formats to the new unified structure.

## Benefits

### 1. Simplicity
- **Single abstraction**: Workflows replace multiple concepts (Provisioners, GoldenPaths, Steps)
- **YAML-only**: No Go code required for provisioning logic
- **Clear separation**: Core (engine) vs. Built-in (standard) vs. Extension (custom)

### 2. Extensibility
- **Git-based**: Providers loaded from Git repositories
- **Version pinning**: Tag/branch/commit references
- **Hot-reload**: Dynamic loading without server restart
- **Team ownership**: Product teams own their service providers

### 3. Composability
- **Building blocks**: Provisioners are reusable components
- **Orchestration**: Golden paths combine provisioners
- **Cross-provider**: Golden paths can reference workflows from different providers

### 4. Maintainability
- **Backward compatible**: Old provider.yaml formats automatically migrated
- **Progressive enhancement**: Teams can migrate at their own pace
- **Clear deprecation path**: Go Provisioner interface deprecated, will be removed in v2.0

## API Changes

### Provider Struct (pkg/sdk/provider.go)

```go
type Provider struct {
    APIVersion    string
    Kind          string
    Metadata      ProviderMetadata
    Compatibility ProviderCompatibility

    // NEW: Unified workflows
    Workflows []WorkflowMetadata `yaml:"workflows,omitempty"`

    // DEPRECATED: For backward compat only
    Provisioners []ProvisionerMetadata `yaml:"provisioners,omitempty"`
    GoldenPaths  []GoldenPathMetadata  `yaml:"goldenpaths,omitempty"`
}

type WorkflowMetadata struct {
    Name        string   `yaml:"name"`
    File        string   `yaml:"file"`
    Version     string   `yaml:"version,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Category    string   `yaml:"category,omitempty"`  // "provisioner" or "goldenpath"
    Tags        []string `yaml:"tags,omitempty"`
}
```

### Provider Loader (internal/providers/loader.go)

```go
// Automatically migrates old format to new
func (l *Loader) migrateProvider(provider *sdk.Provider) {
    if len(provider.Workflows) > 0 {
        return // Already using new format
    }

    // Migrate goldenpaths to workflows with category="goldenpath"
    for _, gp := range provider.GoldenPaths {
        workflow := gp
        if workflow.Category == "" {
            workflow.Category = "goldenpath"
        }
        provider.Workflows = append(provider.Workflows, workflow)
    }
}
```

## UI Changes

The Providers UI page now displays:

1. **Statistics Cards** (4 cards):
   - Total Providers
   - Total Workflows (combined count)
   - Provisioners (single-resource)
   - Golden Paths (multi-resource orchestration)

2. **Provider Table**:
   - Unified "Workflows" column showing total with breakdown
   - Badge indicators: (P) for provisioners, (GP) for golden paths

3. **Information Card**:
   - Explains workflow types
   - Documents provider sources (Built-in vs Extension)
   - Clarifies provider categories (Infrastructure vs Service)

## Future Considerations

### v2.0 Breaking Changes
- Remove deprecated `Provisioner` Go interface
- Remove deprecated `Provisioners[]` and `GoldenPaths[]` fields from Provider struct
- Require all providers to use `workflows[]` field

### Enhancements
- **Workflow Registry**: Central registry of available workflows across all providers
- **Dependency Management**: Explicit dependencies between workflows
- **Validation**: JSON Schema validation for workflow definitions
- **Testing Framework**: Built-in testing for workflow execution
- **Marketplace**: Public registry of community providers

## References

- **Provider Examples**: `providers/builtin/provider.yaml`
- **Workflow Examples**: `providers/builtin/workflows/*.yaml`
- **API Documentation**: `pkg/sdk/provider.go`, `pkg/sdk/provisioner.go`
- **Loader Implementation**: `internal/providers/loader.go`
- **UI Implementation**: `web-ui/src/app/providers/page.tsx`
- **Main Documentation**: `CLAUDE.md` (Provider Architecture section)

---

**Related Documents:**
- [EXTENSIBILITY_ARCHITECTURE.md](./EXTENSIBILITY_ARCHITECTURE.md) - Earlier extensibility design
- [DEMO.md](../DEMO.md) - Multi-team usage scenarios
- [CLAUDE.md](../CLAUDE.md) - Main project documentation
