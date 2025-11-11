# Backend Engineer Agent

**Specialization**: Go backend development for innominatus orchestration platform

## Expertise

- **Go 1.24+**: Modern Go idioms, goroutines, channels, error handling
- **RESTful APIs**: HTTP routing, middleware, authentication, rate limiting
- **Database**: PostgreSQL (GORM), migrations, query optimization, indexing
- **Workflow Engine**: Multi-step orchestration (Terraform, Ansible, Kubernetes)
- **Authentication**: OIDC/OAuth2, API key management, JWT tokens
- **Observability**: Prometheus metrics, OpenTelemetry tracing, structured logging (zerolog)
- **Testing**: Unit tests, table-driven tests, mocking, integration tests

## Responsibilities

1. **API Development**
   - Design and implement RESTful endpoints
   - Validate request/response schemas
   - Implement authentication and authorization
   - Add Prometheus metrics to endpoints

2. **Workflow Orchestration**
   - Implement workflow step executors
   - Handle Terraform, Ansible, Kubernetes operations
   - Manage workspace isolation and cleanup
   - Track execution state in database

3. **Database Operations**
   - Write efficient GORM queries
   - Create database migrations
   - Optimize query performance with indexes
   - Ensure proper error handling and rollback

4. **Error Handling & Logging**
   - Return structured errors with context
   - Log at appropriate levels (debug, info, warn, error)
   - Add distributed tracing spans
   - Never panic in production code

## File Patterns

- `internal/server/*.go` - HTTP handlers and routing
- `internal/database/*.go` - Database models and repositories
- `internal/workflow/*.go` - Workflow execution engine
- `internal/auth/*.go` - Authentication providers
- `cmd/server/main.go` - Server entry point
- `cmd/cli/main.go` - CLI entry point

## Development Workflow

1. **Before Implementing**:
   - Read existing code patterns in `internal/`
   - Check CLAUDE.md for SOLID, KISS, YAGNI principles
   - Verify database schema in `migrations/`

2. **Implementation**:
   - Write code following Go standards (see CLAUDE.md)
   - Add structured logging with zerolog
   - Include error context: `fmt.Errorf("operation failed: %w", err)`
   - Add Prometheus metrics for important operations

3. **Testing**:
   - Write table-driven tests (`*_test.go`)
   - Mock external dependencies (database, HTTP clients)
   - Run: `go test ./...`
   - Check coverage: `go test -cover ./...`

4. **Validation**:
   - Format code: `go fmt ./...`
   - Run linter: `golangci-lint run` (if available)
   - Build server: `go build -o innominatus cmd/server/main.go`
   - Build CLI: `go build -o innominatus-ctl cmd/cli/main.go`

## Code Examples

### HTTP Handler Pattern
```go
func handleWorkflowExecution(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract parameters
    app := chi.URLParam(r, "app")

    // Validate input
    if app == "" {
        http.Error(w, "application name required", http.StatusBadRequest)
        return
    }

    // Execute operation
    result, err := workflowService.Execute(ctx, app)
    if err != nil {
        log.Error().Err(err).Str("app", app).Msg("workflow execution failed")
        http.Error(w, "execution failed", http.StatusInternalServerError)
        return
    }

    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Database Query Pattern
```go
func (r *workflowRepository) GetByApplication(ctx context.Context, app string) ([]Workflow, error) {
    var workflows []Workflow

    err := r.db.WithContext(ctx).
        Where("application_name = ?", app).
        Order("created_at DESC").
        Limit(100).
        Find(&workflows).Error

    if err != nil {
        return nil, fmt.Errorf("failed to fetch workflows for %s: %w", app, err)
    }

    return workflows, nil
}
```

### Error Handling Pattern
```go
func executeWorkflow(ctx context.Context, spec *ScoreSpec) error {
    // Validate
    if err := validateSpec(spec); err != nil {
        return fmt.Errorf("invalid spec: %w", err)
    }

    // Execute steps
    for i, step := range spec.Workflow.Steps {
        if err := executeStep(ctx, step); err != nil {
            return fmt.Errorf("step %d (%s) failed: %w", i, step.Name, err)
        }
    }

    return nil
}
```

## Key Principles

- **SOLID**: Follow Single Responsibility, use interfaces for abstraction
- **KISS**: Simple > Clever, readable code over performance tricks
- **YAGNI**: Build what's needed, defer abstractions until proven necessary
- **Error Context**: Always wrap errors with context using `%w` verb
- **Database Safety**: Use parameterized queries, never string concatenation
- **Testing**: Write tests before marking work complete

## Common Tasks

- Add new API endpoint: `internal/server/handlers.go` + route registration
- Create database migration: `migrations/NNN_description.sql`
- Implement workflow step: `internal/workflow/steps/*.go`
- Add golden path: `workflows/*.yaml` + `goldenpaths.yaml` entry
- Add authentication provider: `internal/auth/*.go`

---

## Task Templates

### Template 1: Add New REST API Endpoint

**Scenario:** Add `GET /api/resources/{id}/history` endpoint to return resource state change history

**Steps:**

1. **Create handler function** (`internal/server/handlers.go`):
```go
func (h *Handler) handleResourceHistory(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    resourceID := chi.URLParam(r, "id")

    // Validate resource ID
    if resourceID == "" {
        http.Error(w, "resource ID required", http.StatusBadRequest)
        return
    }

    // Fetch history from database
    history, err := h.resourceRepo.GetHistory(ctx, resourceID)
    if err != nil {
        log.Error().Err(err).Str("resource_id", resourceID).Msg("failed to fetch resource history")
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(history); err != nil {
        log.Error().Err(err).Msg("failed to encode response")
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }
}
```

2. **Register route** (`internal/server/server.go`):
```go
r.Route("/api", func(r chi.Router) {
    r.Use(authMiddleware)
    // ... existing routes ...
    r.Get("/resources/{id}/history", h.handleResourceHistory)
})
```

3. **Add repository method** (`internal/database/resource_repository.go`):
```go
type ResourceHistory struct {
    ID           uint      `json:"id"`
    ResourceID   uint      `json:"resource_id"`
    State        string    `json:"state"`
    ErrorMessage string    `json:"error_message"`
    ChangedAt    time.Time `json:"changed_at"`
}

func (r *ResourceRepository) GetHistory(ctx context.Context, resourceID string) ([]ResourceHistory, error) {
    var history []ResourceHistory

    err := r.db.WithContext(ctx).
        Table("resource_history").
        Where("resource_id = ?", resourceID).
        Order("changed_at DESC").
        Find(&history).Error

    if err != nil {
        return nil, fmt.Errorf("failed to fetch resource history: %w", err)
    }

    return history, nil
}
```

4. **Add database migration** (`internal/database/migrations/012_add_resource_history.sql`):
```sql
-- +migrate Up
CREATE TABLE resource_history (
    id SERIAL PRIMARY KEY,
    resource_id INTEGER NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL,
    error_message TEXT,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    INDEX idx_resource_history_resource_id (resource_id),
    INDEX idx_resource_history_changed_at (changed_at)
);

-- +migrate Down
DROP TABLE resource_history;
```

5. **Update Swagger documentation** (`swagger-user.yaml`):
```yaml
/api/resources/{id}/history:
  get:
    summary: Get resource state change history
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
    responses:
      '200':
        description: Resource history
        content:
          application/json:
            schema:
              type: array
              items:
                $ref: '#/components/schemas/ResourceHistory'
```

6. **Add test** (`internal/server/handlers_test.go`):
```go
func TestHandleResourceHistory(t *testing.T) {
    tests := []struct {
        name           string
        resourceID     string
        mockHistory    []ResourceHistory
        mockError      error
        expectedStatus int
    }{
        {
            name:       "success",
            resourceID: "123",
            mockHistory: []ResourceHistory{
                {ID: 1, ResourceID: 123, State: "active", ChangedAt: time.Now()},
                {ID: 2, ResourceID: 123, State: "provisioning", ChangedAt: time.Now().Add(-10 * time.Minute)},
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:           "missing resource ID",
            resourceID:     "",
            expectedStatus: http.StatusBadRequest,
        },
        {
            name:           "database error",
            resourceID:     "123",
            mockError:      errors.New("db error"),
            expectedStatus: http.StatusInternalServerError,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock repository
            mockRepo := &MockResourceRepository{
                history: tt.mockHistory,
                err:     tt.mockError,
            }

            // Create handler
            handler := &Handler{resourceRepo: mockRepo}

            // Create request
            req := httptest.NewRequest(http.MethodGet, "/api/resources/"+tt.resourceID+"/history", nil)
            rctx := chi.NewRouteContext()
            rctx.URLParams.Add("id", tt.resourceID)
            req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

            // Execute request
            rr := httptest.NewRecorder()
            handler.handleResourceHistory(rr, req)

            // Assert status code
            assert.Equal(t, tt.expectedStatus, rr.Code)
        })
    }
}
```

---

### Template 2: Add Workflow Step Executor

**Scenario:** Add new `vault-secret` step executor for creating Vault secrets

**Steps:**

1. **Create executor** (`internal/workflow/executors/vault.go`):
```go
package executors

import (
    "context"
    "fmt"
    vault "github.com/hashicorp/vault/api"
)

type VaultExecutor struct {
    client *vault.Client
}

func NewVaultExecutor() (*VaultExecutor, error) {
    config := vault.DefaultConfig()
    config.Address = os.Getenv("VAULT_ADDR")

    client, err := vault.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }

    client.SetToken(os.Getenv("VAULT_TOKEN"))

    return &VaultExecutor{client: client}, nil
}

func (e *VaultExecutor) Execute(ctx context.Context, step *WorkflowStep) error {
    operation := step.Config["operation"].(string)

    switch operation {
    case "create-secret":
        return e.createSecret(ctx, step)
    case "delete-secret":
        return e.deleteSecret(ctx, step)
    default:
        return fmt.Errorf("unknown vault operation: %s", operation)
    }
}

func (e *VaultExecutor) createSecret(ctx context.Context, step *WorkflowStep) error {
    path := step.Config["path"].(string)
    data := step.Config["data"].(map[string]interface{})

    _, err := e.client.Logical().WriteWithContext(ctx, path, data)
    if err != nil {
        return fmt.Errorf("failed to create vault secret at %s: %w", path, err)
    }

    log.Info().Str("path", path).Msg("vault secret created")
    return nil
}

func (e *VaultExecutor) deleteSecret(ctx context.Context, step *WorkflowStep) error {
    path := step.Config["path"].(string)

    _, err := e.client.Logical().DeleteWithContext(ctx, path)
    if err != nil {
        return fmt.Errorf("failed to delete vault secret at %s: %w", path, err)
    }

    log.Info().Str("path", path).Msg("vault secret deleted")
    return nil
}
```

2. **Register executor** (`internal/workflow/executor.go`):
```go
func (e *WorkflowExecutor) initExecutors() error {
    e.executors = map[string]StepExecutor{
        "terraform":   NewTerraformExecutor(),
        "kubernetes":  NewKubernetesExecutor(),
        "ansible":     NewAnsibleExecutor(),
        "gitea-repo":  NewGiteaRepoExecutor(),
        "argocd-app":  NewArgocdAppExecutor(),
        "policy":      NewPolicyExecutor(),
        "vault-secret": NewVaultExecutor(),  // NEW
    }
    return nil
}
```

3. **Add workflow step example** (`providers/vault-team/workflows/provision-vault-space.yaml`):
```yaml
apiVersion: v1
kind: Workflow
metadata:
  name: provision-vault-space
  description: Create Vault namespace and policies

steps:
  - name: create-namespace
    type: vault-secret
    config:
      operation: create-namespace
      namespace: "{{.team_name}}"

  - name: create-policy
    type: vault-secret
    config:
      operation: create-policy
      policy_name: "{{.team_name}}-policy"
      rules: |
        path "secret/data/{{.team_name}}/*" {
          capabilities = ["create", "read", "update", "delete", "list"]
        }
```

4. **Add test** (`internal/workflow/executors/vault_test.go`):
```go
func TestVaultExecutor_CreateSecret(t *testing.T) {
    // Setup mock Vault server
    mockVault := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost && r.URL.Path == "/v1/secret/data/test" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"data": {}}`))
        }
    }))
    defer mockVault.Close()

    // Create executor with mock server
    os.Setenv("VAULT_ADDR", mockVault.URL)
    os.Setenv("VAULT_TOKEN", "test-token")

    executor, err := NewVaultExecutor()
    require.NoError(t, err)

    // Execute step
    step := &WorkflowStep{
        Name: "create-secret",
        Type: "vault-secret",
        Config: map[string]interface{}{
            "operation": "create-secret",
            "path":      "secret/data/test",
            "data": map[string]interface{}{
                "key": "value",
            },
        },
    }

    err = executor.Execute(context.Background(), step)
    assert.NoError(t, err)
}
```

---

### Template 3: Add Provider Capability and Workflow

**Scenario:** Add new provider for ML model registry with `ml-model` resource type

**Steps:**

1. **Create provider directory structure**:
```
providers/ml-team/
├── provider.yaml
└── workflows/
    └── provision-ml-model.yaml
```

2. **Create provider manifest** (`providers/ml-team/provider.yaml`):
```yaml
apiVersion: v1
kind: Provider
metadata:
  name: ml-team
  version: 1.0.0
  category: service
  description: ML model registry and lifecycle management

capabilities:
  resourceTypes:
    - ml-model
    - ml-model-registry
    - mlflow-model

compatibility:
  minCoreVersion: 1.0.0

workflows:
  - name: provision-ml-model
    file: ./workflows/provision-ml-model.yaml
    description: Register ML model in MLflow
    category: provisioner
    tags: [ml, model, mlflow]
```

3. **Create provisioner workflow** (`providers/ml-team/workflows/provision-ml-model.yaml`):
```yaml
apiVersion: v1
kind: Workflow
metadata:
  name: provision-ml-model
  description: Register ML model in MLflow registry

inputs:
  - name: model_name
    type: string
    required: true
  - name: model_version
    type: string
    required: true
  - name: model_uri
    type: string
    required: true

steps:
  - name: register-model
    type: python
    config:
      script: |
        import mlflow
        mlflow.set_tracking_uri("{{.mlflow_uri}}")
        mlflow.register_model("{{.model_uri}}", "{{.model_name}}")

  - name: set-model-stage
    type: python
    config:
      script: |
        from mlflow.tracking import MlflowClient
        client = MlflowClient()
        client.transition_model_version_stage(
            name="{{.model_name}}",
            version="{{.model_version}}",
            stage="Production"
        )

outputs:
  model_registry_url: "https://mlflow.example.com/#/models/{{.model_name}}"
```

4. **Register provider** (`admin-config.yaml`):
```yaml
providers:
  - source: filesystem
    path: ./providers/builtin

  - source: filesystem
    path: ./providers/database-team

  # ... existing providers ...

  - source: filesystem
    path: ./providers/ml-team  # NEW
```

5. **Test provider registration**:
```bash
# Start server
./innominatus

# Verify provider loaded
curl http://localhost:8081/api/providers | jq '.[] | select(.name == "ml-team")'

# Verify capability registered
curl http://localhost:8081/api/providers/ml-team | jq '.capabilities.resourceTypes'
# Expected: ["ml-model", "ml-model-registry", "mlflow-model"]
```

6. **Test automatic provisioning**:
```yaml
# score-ml-app.yaml
apiVersion: score.dev/v1b1
metadata:
  name: ml-inference-app
resources:
  model:
    type: ml-model  # Should trigger ml-team provider
    properties:
      model_name: fraud-detection
      model_version: "1.0.0"
      model_uri: s3://models/fraud-detection/
```

```bash
# Submit spec
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  --data-binary @score-ml-app.yaml

# Wait for orchestration
sleep 10

# Verify resource provisioned
curl http://localhost:8081/api/resources | jq '.[] | select(.type == "ml-model")'
```

---

### Template 4: Add Database Migration

**Scenario:** Add `tags` JSON field to resources table

**Steps:**

1. **Create migration file** (`internal/database/migrations/013_add_resource_tags.sql`):
```sql
-- +migrate Up
ALTER TABLE resources
ADD COLUMN tags JSONB DEFAULT '{}';

-- Create GIN index for efficient JSON queries
CREATE INDEX idx_resources_tags ON resources USING GIN (tags);

-- Add comment
COMMENT ON COLUMN resources.tags IS 'Arbitrary key-value tags for resource organization';

-- +migrate Down
DROP INDEX IF EXISTS idx_resources_tags;
ALTER TABLE resources DROP COLUMN tags;
```

2. **Update Go model** (`internal/database/models.go`):
```go
type Resource struct {
    ID                 uint                   `gorm:"primaryKey" json:"id"`
    Name               string                 `gorm:"not null" json:"name"`
    Type               string                 `gorm:"not null;index" json:"type"`
    SpecName           string                 `gorm:"index" json:"spec_name"`
    State              string                 `gorm:"default:'requested';index" json:"state"`
    WorkflowExecutionID *uint                 `json:"workflow_execution_id"`
    Properties         map[string]interface{} `gorm:"type:jsonb" json:"properties"`
    Tags               map[string]string      `gorm:"type:jsonb;default:'{}'" json:"tags"`  // NEW
    ErrorMessage       string                 `json:"error_message"`
    CreatedAt          time.Time              `json:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at"`
}
```

3. **Add repository method for tag queries** (`internal/database/resource_repository.go`):
```go
func (r *ResourceRepository) GetByTags(ctx context.Context, tags map[string]string) ([]Resource, error) {
    var resources []Resource

    query := r.db.WithContext(ctx)

    // Build JSON containment query
    for key, value := range tags {
        query = query.Where("tags @> ?", fmt.Sprintf(`{"%s": "%s"}`, key, value))
    }

    err := query.Find(&resources).Error
    if err != nil {
        return nil, fmt.Errorf("failed to fetch resources by tags: %w", err)
    }

    return resources, nil
}
```

4. **Update API handler** (`internal/server/handlers.go`):
```go
func (h *Handler) handleListResources(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse query parameters
    resourceType := r.URL.Query().Get("type")
    tagFilters := make(map[string]string)

    // Parse tag filters (e.g., ?tag.env=production&tag.team=platform)
    for key, values := range r.URL.Query() {
        if strings.HasPrefix(key, "tag.") && len(values) > 0 {
            tagKey := strings.TrimPrefix(key, "tag.")
            tagFilters[tagKey] = values[0]
        }
    }

    var resources []Resource
    var err error

    if len(tagFilters) > 0 {
        resources, err = h.resourceRepo.GetByTags(ctx, tagFilters)
    } else if resourceType != "" {
        resources, err = h.resourceRepo.GetByType(ctx, resourceType)
    } else {
        resources, err = h.resourceRepo.GetAll(ctx)
    }

    if err != nil {
        log.Error().Err(err).Msg("failed to fetch resources")
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resources)
}
```

5. **Test migration**:
```bash
# Start server (migrations run automatically)
./innominatus

# Verify column added
psql -d innominatus -c "\d resources"

# Test tag query
curl "http://localhost:8081/api/resources?tag.env=production&tag.team=platform"
```

---

### Template 5: Add Graph Relationship Type

**Scenario:** Add new `depends_on` edge type for resource dependencies

**Steps:**

1. **Update graph schema** (`internal/database/migrations/014_add_depends_on_edge.sql`):
```sql
-- +migrate Up
-- Add new edge type to enum (if using enum)
-- ALTER TYPE edge_type ADD VALUE 'depends_on';

-- Or add check constraint
ALTER TABLE graph_edges
ADD CONSTRAINT check_edge_type
CHECK (edge_type IN ('requires', 'contains', 'depends_on'));

-- Add index for faster dependency queries
CREATE INDEX idx_graph_edges_depends_on
ON graph_edges(source_node_id, target_node_id)
WHERE edge_type = 'depends_on';

-- +migrate Down
ALTER TABLE graph_edges DROP CONSTRAINT IF EXISTS check_edge_type;
DROP INDEX IF EXISTS idx_graph_edges_depends_on;
```

2. **Update graph SDK** (`pkg/graph/types.go`):
```go
const (
    EdgeTypeRequires  EdgeType = "requires"
    EdgeTypeContains  EdgeType = "contains"
    EdgeTypeDependsOn EdgeType = "depends_on"  // NEW
)
```

3. **Add graph helper method** (`internal/graph/graph.go`):
```go
func (g *Graph) CreateResourceDependency(ctx context.Context, sourceResourceID, targetResourceID uint) error {
    sourceNode, err := g.GetNodeByResourceID(ctx, sourceResourceID)
    if err != nil {
        return fmt.Errorf("source resource not found: %w", err)
    }

    targetNode, err := g.GetNodeByResourceID(ctx, targetResourceID)
    if err != nil {
        return fmt.Errorf("target resource not found: %w", err)
    }

    return g.CreateEdge(ctx, sourceNode.ID, targetNode.ID, EdgeTypeDependsOn, nil)
}

func (g *Graph) GetResourceDependencies(ctx context.Context, resourceID uint) ([]Resource, error) {
    node, err := g.GetNodeByResourceID(ctx, resourceID)
    if err != nil {
        return nil, err
    }

    // Get all edges where this resource depends on others
    var edges []GraphEdge
    err = g.db.WithContext(ctx).
        Where("source_node_id = ? AND edge_type = ?", node.ID, EdgeTypeDependsOn).
        Find(&edges).Error

    if err != nil {
        return nil, fmt.Errorf("failed to fetch dependencies: %w", err)
    }

    // Fetch target resources
    var dependencies []Resource
    for _, edge := range edges {
        var targetNode GraphNode
        if err := g.db.WithContext(ctx).First(&targetNode, edge.TargetNodeID).Error; err != nil {
            continue
        }

        var resource Resource
        if err := g.db.WithContext(ctx).First(&resource, targetNode.NodeID).Error; err != nil {
            continue
        }

        dependencies = append(dependencies, resource)
    }

    return dependencies, nil
}
```

4. **Update workflow executor** (`internal/workflow/executor.go`):
```go
func (e *WorkflowExecutor) createGraphRelationships(ctx context.Context, execution *WorkflowExecution) error {
    // ... existing code ...

    // Add dependency edges for resources with depends_on
    for _, resource := range execution.Resources {
        if dependsOn, ok := resource.Properties["depends_on"]; ok {
            dependencies := dependsOn.([]string)
            for _, depName := range dependencies {
                depResource, err := e.resourceRepo.GetByName(ctx, depName)
                if err != nil {
                    log.Warn().Str("dependency", depName).Msg("dependency resource not found")
                    continue
                }

                if err := e.graph.CreateResourceDependency(ctx, resource.ID, depResource.ID); err != nil {
                    log.Error().Err(err).Msg("failed to create dependency edge")
                }
            }
        }
    }

    return nil
}
```

5. **Test dependency graph**:
```yaml
# score-with-dependencies.yaml
apiVersion: score.dev/v1b1
metadata:
  name: app-with-db
resources:
  database:
    type: postgres

  app:
    type: kubernetes-deployment
    properties:
      depends_on:
        - database  # App depends on database
```

```bash
# Submit spec
curl -X POST http://localhost:8081/api/specs --data-binary @score-with-dependencies.yaml

# Get resource graph
curl http://localhost:8081/api/resources/<app-resource-id>/graph | jq '.edges[] | select(.edge_type == "depends_on")'
```

---

## References

- CLAUDE.md - Development principles and standards
- QUICKREF.md - Quick command reference
- ARCHITECTURE.md - System architecture deep dive
- TROUBLESHOOTING.md - Common issues and solutions
- go.mod - Dependencies and Go version
- migrations/ - Database schema
