# Product Team Workflow Feature - Gap Analysis

**Status:** In Development
**Version:** 1.0.0
**Last Updated:** 2025-10-19
**Audience:** Platform Teams, Contributors

---

## Executive Summary

Product team workflow capabilities are **partially implemented** in the codebase but **NOT activated in the production server**. This document provides a comprehensive analysis of what exists, what's missing, and what's needed to make this feature production-ready.

### Critical Finding

⚠️ **The multi-tier workflow executor exists but is not wired to the API server**. Product workflows will never execute in production deployments until handlers.go is updated to use `NewMultiTierWorkflowExecutor`.

---

## Feature Overview

### What Are Product Workflows?

Product workflows allow internal product teams (e.g., e-commerce, analytics, payments teams) to extend innominatus with product-specific deployment logic that automatically runs when applications consuming their product are deployed.

**Example Use Cases:**
- E-commerce team: Automatically configure payment gateways when apps deploy
- Analytics team: Set up data pipelines when analytics-enabled apps deploy
- ML team: Provision model serving infrastructure

**Target Persona:** Product engineering teams building internal services consumed by application developers

---

## Implementation Status Matrix

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| **Data Structures** | ✅ Complete | `internal/workflow/resolver.go` | ProductWorkflow, WorkflowPhase, WorkflowTier defined |
| **Workflow Resolver** | ✅ Complete | `internal/workflow/resolver.go` | Full resolution logic implemented |
| **Multi-Tier Executor** | ⚠️ Implemented but Unused | `internal/workflow/executor.go:96-126` | Exists but not called by server |
| **Admin Config** | ✅ Complete | `internal/admin/config.go:62-79` | WorkflowPolicies struct complete |
| **Policy Validation** | ⚠️ Implemented but Unused | `internal/workflow/resolver.go:369` | Not called in production flow |
| **Example Workflows** | ✅ Complete | `workflows/products/` | 3 working examples |
| **SDK for Provisioners** | ✅ Interface Defined | `pkg/sdk/provisioner.go` | Interface exists, no registry |
| **API Endpoints** | ❌ Missing | N/A | No discovery/management APIs |
| **CLI Commands** | ❌ Missing | N/A | No product team tooling |
| **Documentation** | ❌ Missing | N/A | No user-facing docs |
| **Server Integration** | ❌ Missing | `internal/server/handlers.go:226` | Uses single-tier executor |

---

## Detailed Gap Analysis

### GAP 1: Multi-Tier Executor Not Used in Server ⚠️ **CRITICAL**

**Severity:** Critical
**Impact:** Product workflows never execute
**Effort:** 1 hour

#### Current State

**File:** `internal/server/handlers.go:226`

```go
// Current implementation (single-tier only)
workflowExecutor := workflow.NewWorkflowExecutorWithResourceManager(workflowRepo, resourceManager)
```

This creates a basic executor that:
- Executes workflows from Score spec only
- Ignores platform workflows in `workflows/platform/`
- Ignores product workflows in `workflows/products/`
- Does not load admin config workflow policies

#### Required Implementation

```go
// Required implementation (multi-tier)
// Load admin configuration
adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
if err != nil {
    // Handle error or fall back to single-tier
    logger.WarnWithFields("Failed to load admin config, using single-tier executor", map[string]interface{}{
        "error": err.Error(),
    })
    workflowExecutor = workflow.NewWorkflowExecutorWithResourceManager(workflowRepo, resourceManager)
} else {
    // Create resolver with policies from admin config
    policies := workflow.WorkflowPolicies{
        RequiredPlatformWorkflows: adminConfig.WorkflowPolicies.RequiredPlatformWorkflows,
        AllowedProductWorkflows:   adminConfig.WorkflowPolicies.AllowedProductWorkflows,
        WorkflowOverrides: struct {
            Platform bool `yaml:"platform"`
            Product  bool `yaml:"product"`
        }{
            Platform: adminConfig.WorkflowPolicies.WorkflowOverrides.Platform,
            Product:  adminConfig.WorkflowPolicies.WorkflowOverrides.Product,
        },
        MaxWorkflowDuration: adminConfig.WorkflowPolicies.MaxWorkflowDuration,
    }

    resolver := workflow.NewWorkflowResolver(adminConfig.WorkflowPolicies.WorkflowsRoot, policies)
    workflowExecutor = workflow.NewMultiTierWorkflowExecutorWithResourceManager(workflowRepo, resolver, resourceManager)
}
```

#### Verification Steps

1. Make the code change above
2. Run server: `./innominatus`
3. Deploy an ecommerce app with Score spec containing product metadata
4. Check logs for "Executing workflow phase: pre-deployment" with product workflows
5. Verify database `workflow_executions` table shows multi-tier workflow names

#### Dependencies

- `adminConfig` must be passed from `main.go` to handlers (requires refactoring)
- OR: Load admin config directly in handlers (simpler, but duplicates loading)

---

### GAP 2: No API Endpoints for Product Workflow Discovery

**Severity:** High
**Impact:** Product teams cannot discover or manage their workflows
**Effort:** 4 hours

#### Missing Endpoints

1. **GET /api/workflows/products**
   - List all product workflows discovered in `workflows/products/`
   - Group by product name
   - Show metadata (name, description, phase, triggers)

2. **GET /api/workflows/products/{product-name}**
   - List workflows for specific product
   - Include file paths and YAML content

3. **GET /api/workflows/products/{product-name}/{workflow-name}**
   - Get details of specific workflow
   - Show resolved steps
   - Indicate if allowed by policy

4. **POST /api/workflows/products/validate**
   - Validate a product workflow YAML
   - Check against allowed step types
   - Verify metadata structure

#### Example Response

```json
{
  "product": "ecommerce",
  "workflows": [
    {
      "name": "payment-integration",
      "file": "workflows/products/ecommerce/payment-integration.yaml",
      "metadata": {
        "description": "Payment service integration and configuration",
        "owner": "ecommerce-infrastructure-team",
        "phase": "deployment"
      },
      "triggers": ["product_deployment"],
      "step_count": 3,
      "allowed": true
    }
  ]
}
```

---

### GAP 3: Policy Enforcement Not Active

**Severity:** High
**Impact:** Security risk - any workflow can run regardless of policy
**Effort:** 2 hours

#### Current State

**File:** `internal/workflow/resolver.go:369`

```go
func (r *WorkflowResolver) ValidateWorkflowPolicies(resolved map[WorkflowPhase][]ResolvedWorkflow) error {
    // Implementation exists but is only called in workflow-demo.go, not in server
}
```

**admin-config.yaml** defines policies:
```yaml
workflowPolicies:
  allowedProductWorkflows:
    - ecommerce/database-setup
    - ecommerce/payment-integration
    - analytics/data-pipeline
```

But these policies are **never enforced** in production.

#### Required Implementation

In `internal/server/handlers.go` (or wherever workflows are executed):

```go
// After resolving workflows
resolvedWorkflows, err := workflowExecutor.ResolveWorkflows(app)
if err != nil {
    return err
}

// VALIDATE POLICIES (currently missing)
if err := resolver.ValidateWorkflowPolicies(resolvedWorkflows); err != nil {
    return fmt.Errorf("workflow policy violation: %w", err)
}

// Then execute
if err := workflowExecutor.ExecuteMultiTierWorkflows(ctx, app); err != nil {
    return err
}
```

#### Security Implications

Without policy validation:
- Unauthorized product workflows can be added to `workflows/products/` directory
- Malicious workflow files could execute arbitrary steps
- No audit trail of which workflows were approved

---

### GAP 4: No CLI Commands for Product Teams

**Severity:** Medium
**Impact:** Poor developer experience for product teams
**Effort:** 8 hours

#### Missing Commands

1. **`innominatus-ctl list-products`**
   ```bash
   $ innominatus-ctl list-products

   Product Name    Workflows    Allowed
   ------------    ---------    -------
   ecommerce       3            Yes
   analytics       1            Yes
   ml-platform     2            No (not in allowedProductWorkflows)
   ```

2. **`innominatus-ctl list-product-workflows [product-name]`**
   ```bash
   $ innominatus-ctl list-product-workflows ecommerce

   Workflow                 Phase            Steps    Owner
   --------                 -----            -----    -----
   database-setup           pre-deployment   3        ecommerce-infrastructure-team
   payment-integration      deployment       3        ecommerce-infrastructure-team
   ```

3. **`innominatus-ctl validate-product-workflow <file>`**
   ```bash
   $ innominatus-ctl validate-product-workflow workflows/products/ecommerce/new-workflow.yaml

   ✅ Workflow structure valid
   ✅ All step types allowed
   ✅ Metadata complete
   ⚠️  Warning: Workflow not in admin-config.yaml allowedProductWorkflows
   ```

4. **`innominatus-ctl test-product-workflow <workflow-file> <score-spec>`**
   ```bash
   $ innominatus-ctl test-product-workflow workflows/products/ecommerce/payment-integration.yaml test-app.yaml

   🔄 Resolving workflows for test-app...
   ✅ Product workflow triggered: payment-integration
   📋 Steps to execute:
       1. setup-payment-vault (vault-setup)
       2. configure-payment-gateway (kubernetes)
       3. verify-payment-connectivity (validation)

   Run with --execute to actually execute steps
   ```

#### Implementation Location

**File:** `cmd/cli/main.go`

Add new command group:

```go
var productCmd = &cobra.Command{
    Use:   "product",
    Short: "Manage product workflows",
}

var listProductsCmd = &cobra.Command{
    Use:   "list-products",
    Short: "List all products with workflows",
    Run:   runListProducts,
}

var validateWorkflowCmd = &cobra.Command{
    Use:   "validate-product-workflow <file>",
    Short: "Validate a product workflow YAML file",
    Run:   runValidateWorkflow,
}
```

---

### GAP 5: Product Provisioner Registration Missing

**Severity:** Medium
**Impact:** Product teams cannot add custom resource types
**Effort:** 16 hours

#### Current State

**File:** `pkg/sdk/provisioner.go`

The SDK defines the `Provisioner` interface:

```go
type Provisioner interface {
    Name() string
    Type() string
    Version() string
    Provision(ctx context.Context, resource *Resource, config Config) error
    Deprovision(ctx context.Context, resource *Resource) error
    GetStatus(ctx context.Context, resource *Resource) (*ResourceStatus, error)
    GetHints(ctx context.Context, resource *Resource) ([]Hint, error)
}
```

**BUT**: No mechanism exists to:
1. Register product-specific provisioners
2. Discover available provisioners
3. Route resource types to product provisioners
4. Validate provisioner implementations

#### Example Need

E-commerce team wants to define a custom resource:

```yaml
# Score spec
resources:
  payment-gateway:
    type: payment-gateway  # Custom type
    properties:
      provider: stripe
      webhook_url: https://...
```

**Required:**
- E-commerce team creates `PaymentGatewayProvisioner` implementing SDK interface
- Provisioner is registered in `admin-config.yaml`:
  ```yaml
  productProvisioners:
    - product: ecommerce
      type: payment-gateway
      provisioner: github.com/company/ecommerce-provisioners/payment-gateway
      version: 1.0.0
  ```
- innominatus loads and routes `payment-gateway` type to this provisioner

#### Missing Implementation

1. **Provisioner Registry**
   ```go
   // internal/platform/product_registry.go (needs to be created)
   type ProductProvisionerRegistry struct {
       provisioners map[string]sdk.Provisioner
   }

   func (r *ProductProvisionerRegistry) Register(productName, resourceType string, prov sdk.Provisioner) error
   func (r *ProductProvisionerRegistry) Get(resourceType string) (sdk.Provisioner, error)
   func (r *ProductProvisionerRegistry) List(productName string) []sdk.Provisioner
   ```

2. **Dynamic Loading**
   - Load provisioners from Go plugins (`.so` files)
   - Or: Compile-time registration via imports
   - Or: gRPC-based provisioners (Hashicorp plugin pattern)

3. **Admin Config Schema**
   ```yaml
   # admin-config.yaml
   productProvisioners:
     - product: ecommerce
       name: payment-gateway-provisioner
       type: payment-gateway
       version: 1.0.0
       plugin: /path/to/provisioner.so  # Go plugin
       # OR
       module: github.com/company/ecommerce-provisioners  # Go module
   ```

---

### GAP 6: No Documentation

**Severity:** High
**Impact:** Feature is unusable without documentation
**Effort:** 12 hours

#### Missing Documentation

| Document | Purpose | Audience |
|----------|---------|----------|
| `docs/product-team-guide/README.md` | Introduction & quick start | Product teams |
| `docs/product-team-guide/product-workflows.md` | How to create product workflows | Product teams |
| `docs/product-team-guide/custom-resources.md` | Defining custom resource types | Product teams |
| `docs/product-team-guide/provisioners.md` | Building provisioners | Product teams |
| `docs/product-team-guide/governance.md` | Approval workflows & policies | Product teams |
| `docs/product-team-guide/activation-guide.md` | How to enable multi-tier workflows | Platform teams |
| `docs/product-team-guide/examples.md` | Real-world examples | Product teams |

#### Documentation Strategy

Each guide should include:
- ✅ **Available Now**: What currently works (even if only in demo)
- 🚧 **Coming Soon**: What exists but isn't active
- ❌ **Not Yet Implemented**: What's missing (reference this gap analysis)

---

## Dependency Graph

```
GAP 1 (Critical: Wire Multi-Tier Executor)
  ↓ Required before
GAP 3 (Policy Enforcement)
  ↓ Required before
GAP 2 (API Endpoints) + GAP 4 (CLI Commands)
  ↓ Can be parallel with
GAP 5 (Provisioner Registry)
  ↓ After all above
GAP 6 (Documentation)
```

**Rationale:**
- Must fix GAP 1 first - nothing works without it
- Policy enforcement (GAP 3) must be active before exposing APIs
- APIs and CLI can be built in parallel once core works
- Provisioner registry is advanced feature, can be later
- Documentation last, once features are stable

---

## Implementation Roadmap

### Phase 1: Minimum Viable Product (MVP) - 1 week

**Goal:** Product workflows execute when apps are deployed

- [ ] **GAP 1**: Wire multi-tier executor in handlers.go (1 hour)
  - Update `internal/server/handlers.go:226`
  - Pass admin config from main.go
  - Test with existing product workflows
- [ ] **GAP 3**: Activate policy validation (2 hours)
  - Call `ValidateWorkflowPolicies` before execution
  - Return 403 if policy violated
  - Add unit tests
- [ ] **Testing** (8 hours)
  - Integration tests for multi-tier execution
  - Policy enforcement tests
  - End-to-end test with ecommerce example
- [ ] **Docs** (8 hours)
  - Create `docs/product-team-guide/README.md` with limitations
  - Document activation steps for platform teams
  - Update `docs/index.md` with product team persona

**Deliverable:** Product workflows work end-to-end via API

---

### Phase 2: Developer Experience - 2 weeks

**Goal:** Product teams can self-service workflow development

- [ ] **GAP 4**: CLI commands (8 hours)
  - `innominatus-ctl list-products`
  - `innominatus-ctl validate-product-workflow`
  - `innominatus-ctl test-product-workflow` (dry-run)
- [ ] **GAP 2**: API endpoints (8 hours)
  - GET `/api/workflows/products`
  - POST `/api/workflows/products/validate`
  - OpenAPI spec updates
- [ ] **Docs** (16 hours)
  - Complete `product-workflows.md` guide
  - Complete `governance.md` with approval process
  - Example workflows for 3 product types
  - Troubleshooting guide

**Deliverable:** Product teams can develop and test workflows locally

---

### Phase 3: Advanced Features - 3 weeks

**Goal:** Custom resource types and provisioners

- [ ] **GAP 5**: Provisioner registry (16 hours)
  - Design plugin architecture (Go plugins vs gRPC)
  - Implement registry and loader
  - Add admin config schema for provisioners
- [ ] **GAP 5**: Example provisioners (16 hours)
  - E-commerce payment gateway provisioner
  - Analytics data lake provisioner
  - Documentation with SDK examples
- [ ] **Docs** (16 hours)
  - Complete `custom-resources.md`
  - Complete `provisioners.md` with SDK reference
  - Migration guide from platform extensions

**Deliverable:** Product teams can register custom provisioners

---

## Testing Strategy

### Unit Tests

**Required for:**
- Workflow resolver (`internal/workflow/resolver_test.go`)
- Policy validation
- Provisioner registry (when implemented)

### Integration Tests

**Required for:**
- Multi-tier workflow execution end-to-end
- Policy enforcement blocking unauthorized workflows
- Product workflow triggering based on app metadata

### E2E Tests

**Required for:**
- Deploy ecommerce app → product workflows execute
- Validate policy violation → deployment blocked
- Custom provisioner → resource created

---

## Migration Strategy

### For Existing Deployments

**Before Migration:**
- All workflows execute as single-tier (from Score spec only)
- No product workflows

**After Migration:**
- Backward compatible: Apps without product metadata continue as-is
- Apps with `metadata.product` trigger product workflows
- Platform workflows (security-scan, cost-monitoring) run for all apps

**No Breaking Changes:**
- Existing Score specs work unchanged
- No database migrations needed
- API endpoints remain compatible

---

## Open Questions

### 1. Platform vs Product Provisioners

**Question:** Should product provisioners be in separate repositories like platform extensions?

**Options:**
- **A**: Monorepo - product provisioners in `internal/resources/products/`
- **B**: Separate repos - each product team owns their provisioner repo
- **C**: Hybrid - core products in monorepo, custom in separate repos

**Recommendation:** Option C (hybrid) for flexibility

### 2. Workflow Approval Process

**Question:** How do product teams get workflows approved by platform team?

**Options:**
- **A**: Git-based - PR to add workflow to `workflows/products/`, platform team reviews
- **B**: UI-based - Product team uploads via Web UI, platform admin approves
- **C**: Automated - Workflow validator checks policy, auto-approves if passes

**Recommendation:** Start with A (Git PR), add B (UI) in Phase 2

### 3. Multi-Tenancy

**Question:** Can multiple organizations use same innominatus instance with isolated product workflows?

**Current State:** Not supported - all workflows in same directory tree

**Future Enhancement:** Namespace workflows by organization

---

## Success Metrics

**Phase 1 (MVP) Success:**
- [ ] 3 product teams onboarded (ecommerce, analytics, ML)
- [ ] 10+ product workflows deployed
- [ ] Zero policy violations in production
- [ ] <5 minute time-to-first-workflow for new product team

**Phase 2 (DX) Success:**
- [ ] Product teams can develop workflows without platform team help
- [ ] <1 hour from workflow creation to testing
- [ ] 90% of product teams use CLI tooling

**Phase 3 (Advanced) Success:**
- [ ] 5+ custom provisioners registered
- [ ] Product teams ship custom resources without core changes

---

## References

- [Workflow Resolver Implementation](../internal/workflow/resolver.go)
- [Multi-Tier Executor](../internal/workflow/executor.go)
- [Admin Config Schema](../internal/admin/config.go)
- [SDK Provisioner Interface](../pkg/sdk/provisioner.go)
- [Example Product Workflows](../workflows/products/)
- [Demo Implementation](../workflow-demo.go)
- [Platform Extension Guide](PLATFORM_EXTENSION_GUIDE.md) - Similar pattern for infrastructure platforms

---

**Next Steps:**
1. Review this gap analysis with core team
2. Prioritize gaps (recommend starting with GAP 1 + GAP 3)
3. Create implementation issues/tickets
4. Begin Phase 1 development

**Questions? Contact:** Platform Team (#platform-team on Slack)
