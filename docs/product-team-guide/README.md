# Product Team Guide

**Welcome, Product Team!** 🛠️

This guide is for **internal product engineering teams** who build services consumed by application developers (e.g., payments, analytics, ML platforms).

---

## ⚠️ Current Status: Feature in Development

**Important:** Product workflow capabilities are **partially implemented** and not yet active in production deployments.

**What this means:**
- ✅ Core implementation exists (workflow resolver, executor, policy framework)
- ✅ Example workflows available (`workflows/products/ecommerce/`, `workflows/products/analytics/`)
- ✅ Can be tested via demo (`go run workflow-demo.go`)
- 🚧 **Not yet wired to the API server** - workflows won't execute in production
- 🚧 **No CLI tooling** - product teams can't self-service yet
- 🚧 **No API endpoints** - can't query/manage workflows programmatically

**See:** [Product Workflow Gaps Analysis](../PRODUCT_WORKFLOW_GAPS.md) for complete technical details.

---

## What is a Product Team?

### Personas in innominatus

| Persona | Responsibility | Example |
|---------|----------------|---------|
| **Platform Team** | Deploy and operate innominatus | You manage Kubernetes, databases, configure OIDC |
| **Product Team** (you!) | Build internal services for developers | E-commerce payments, analytics pipelines, ML platforms |
| **App Developers** | Use product team services | Deploy web apps that integrate payments, analytics |

### Product Team Use Cases

**E-commerce Team:**
- Provides payment gateway infrastructure
- When an app deploys, automatically:
  - Provision payment gateway credentials in Vault
  - Configure webhook endpoints
  - Set up monitoring for payment transactions

**Analytics Team:**
- Provides data pipeline infrastructure
- When an analytics-enabled app deploys, automatically:
  - Create Kafka topics for event streaming
  - Set up Spark cluster for data processing
  - Provision data lake storage

**ML Platform Team:**
- Provides model serving infrastructure
- When an ML app deploys, automatically:
  - Set up model registry access
  - Provision GPU resources
  - Configure model serving endpoints

---

## Core Capabilities (When Active)

### 1. Product Workflows

Define workflows that run automatically when applications deploy:

**File:** `workflows/products/ecommerce/payment-integration.yaml`
```yaml
apiVersion: workflow.dev/v1
kind: ProductWorkflow
metadata:
  name: payment-integration
  description: Payment service integration
  product: ecommerce
  owner: ecommerce-infrastructure-team
  phase: deployment
spec:
  triggers:
    - product_deployment
  steps:
    - name: setup-payment-vault
      type: vault-setup
      config:
        secrets: ["stripe-api-key", "paypal-client-secret"]

    - name: configure-payment-gateway
      type: kubernetes
      namespace: "${application.name}-payment"
      config:
        manifests: "./k8s/payment-gateway"

    - name: verify-payment-connectivity
      type: validation
      config:
        healthChecks: ["stripe-connectivity", "paypal-connectivity"]
        timeout: "30s"
```

**Triggers:** When an app developer deploys an app with:
```yaml
# app Score spec
metadata:
  name: checkout-service
  product: ecommerce  # <-- Triggers ecommerce product workflows
```

### 2. Custom Resources (🚧 Planned)

Define product-specific resource types:

```yaml
# In Score spec
resources:
  payment-gateway:
    type: payment-gateway  # Custom type from ecommerce team
    properties:
      provider: stripe
      webhook_url: https://...
```

**Requires:** Custom provisioner implementation (see [Provisioners Guide](provisioners.md))

### 3. Workflow Phases

Product workflows execute in 3 phases:

| Phase | Purpose | Example |
|-------|---------|---------|
| **pre-deployment** | Setup infrastructure | Provision databases, configure secrets |
| **deployment** | Deploy services | Configure gateways, set up integrations |
| **post-deployment** | Verify and monitor | Health checks, monitoring setup |

**Platform workflows** (security-scan, cost-monitoring) run across all phases
**Product workflows** run only for apps consuming your product
**Application workflows** (from Score spec) run for specific app

---

## Current Limitations

### ⚠️ Not Active in Production

**Current State:**
- Multi-tier workflow executor exists (`internal/workflow/executor.go:96-126`)
- But server uses single-tier executor (`internal/server/handlers.go:226`)
- Product workflows are **never loaded or executed**

**Workaround:**
- Can test workflows with: `go run workflow-demo.go`
- Can inspect workflow files manually
- Cannot trigger via API server

**Fix Required:** Platform team must update `handlers.go` to use `NewMultiTierWorkflowExecutor`

**See:** [Activation Guide](activation-guide.md) for platform team instructions

### 🚧 No Self-Service Tooling

**Missing CLI Commands:**
```bash
# Desired (not yet available)
innominatus-ctl list-products
innominatus-ctl list-product-workflows ecommerce
innominatus-ctl validate-product-workflow my-workflow.yaml
innominatus-ctl test-product-workflow my-workflow.yaml test-app.yaml --dry-run
```

**Workaround:**
- Manually read YAML files in `workflows/products/`
- Validate YAML syntax with generic tools
- Test via `workflow-demo.go` (requires Go development environment)

### 🚧 No API Endpoints

**Missing:**
- `GET /api/workflows/products` - List products with workflows
- `GET /api/workflows/products/{product}` - List workflows for product
- `POST /api/workflows/products/validate` - Validate workflow file

**Workaround:**
- Direct file system access to `workflows/products/` directory

### 🚧 Policy Enforcement Not Active

**admin-config.yaml** defines policies:
```yaml
workflowPolicies:
  allowedProductWorkflows:
    - ecommerce/database-setup
    - ecommerce/payment-integration
```

**BUT:** These policies are not enforced in production server

**Risk:** Any workflow file in `workflows/products/` directory could execute
**Mitigation:** Platform team must manually review workflow files before adding

---

## Getting Started (Current State)

### Prerequisites

- Git repository access to innominatus codebase
- Permission to create files in `workflows/products/{your-product}/`
- Approval from platform team for new product workflows
- Go development environment (for testing via `workflow-demo.go`)

### Step 1: Create Your Product Directory

```bash
mkdir -p workflows/products/your-product
```

### Step 2: Create Your First Workflow

**File:** `workflows/products/your-product/setup-infrastructure.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: ProductWorkflow
metadata:
  name: setup-infrastructure
  description: Setup infrastructure for your-product
  product: your-product
  owner: your-product-infrastructure-team
  phase: pre-deployment

spec:
  triggers:
    - product_deployment

  steps:
    - name: provision-resources
      type: terraform
      config:
        working_dir: ./terraform/your-product
        operation: apply

    - name: configure-monitoring
      type: monitoring
      config:
        service: your-product
        alerts: ["error_rate", "latency"]
```

### Step 3: Register in Admin Config

Add your workflow to allowed list:

**File:** `admin-config.yaml`

```yaml
workflowPolicies:
  allowedProductWorkflows:
    - ecommerce/database-setup
    - ecommerce/payment-integration
    - analytics/data-pipeline
    - your-product/setup-infrastructure  # Add this
```

### Step 4: Test with Demo

```bash
# Modify workflow-demo.go to test your product
# Update app configuration:
app := &workflow.ApplicationInstance{
    Name: "test-app",
    Configuration: map[string]interface{}{
        "metadata": map[string]interface{}{
            "product": "your-product",  # Your product name
        },
    },
}

# Run demo
go run workflow-demo.go
```

**Expected Output:**
```
🔄 Resolving Multi-Tier Workflows...
📊 Workflow Resolution Results:
  pre-deployment Phase (1 workflows):
    🔧 product-your-product-setup-infrastructure (2 steps)
       Step 1: provision-resources (terraform)
       Step 2: configure-monitoring (monitoring)
```

### Step 5: Submit for Review

1. Create Git branch: `git checkout -b add-your-product-workflows`
2. Add workflow files and admin config changes
3. Create pull request for platform team review
4. Platform team approves and merges

**Note:** Currently, workflows won't execute in production until multi-tier executor is activated.

---

## Workflow Development Best Practices

### 1. Use Descriptive Names

**Good:**
```yaml
name: setup-payment-vault-credentials
```

**Bad:**
```yaml
name: setup
```

### 2. Specify Correct Phase

- **pre-deployment**: Infrastructure setup, database provisioning
- **deployment**: Service configuration, integration setup
- **post-deployment**: Verification, monitoring, health checks

**Wrong phase = workflow runs at wrong time**

### 3. Set Appropriate Triggers

```yaml
triggers:
  - product_deployment  # Run for all apps using this product
  # OR
  - first_deployment    # Run only on first deployment
  # OR
  - manual             # Only run when explicitly invoked
```

### 4. Add Owner and Description

```yaml
metadata:
  owner: your-product-infrastructure-team  # Who maintains this
  description: Clear description of what this workflow does
```

**Why:** Platform team needs to know who to contact for issues

### 5. Validate Step Types Against Policy

**Check admin-config.yaml:**
```yaml
workflowPolicies:
  allowedStepTypes:
    - terraform
    - kubernetes
    - ansible
    - database-migration
    - vault-setup
    - monitoring
    - validation
```

**Only use allowed step types** - others will fail policy validation (when enforcement is active)

---

## Examples

### Example 1: E-commerce Payment Integration

**See:** [workflows/products/ecommerce/payment-integration.yaml](../../workflows/products/ecommerce/payment-integration.yaml)

**What it does:**
1. Sets up payment gateway secrets in Vault
2. Deploys payment gateway Kubernetes resources
3. Verifies connectivity to payment providers

**When it runs:** When any app with `metadata.product: ecommerce` deploys

### Example 2: Analytics Data Pipeline

**See:** [workflows/products/analytics/data-pipeline.yaml](../../workflows/products/analytics/data-pipeline.yaml)

**What it does:**
1. Provisions data lake storage
2. Sets up Kafka streaming infrastructure
3. Configures Spark cluster for data processing
4. Creates data catalog schema

**When it runs:** When any app with `metadata.product: analytics` deploys

### Example 3: E-commerce Database Setup

**See:** [workflows/products/ecommerce/database-setup.yaml](../../workflows/products/ecommerce/database-setup.yaml)

**What it does:**
1. Provisions production database via Terraform
2. Runs database migrations
3. Sets up database monitoring alerts

**When it runs:** Pre-deployment phase for e-commerce apps

---

## Roadmap

### Phase 1: Activation (Target: 1 week)
- [ ] Multi-tier executor activated in production server
- [ ] Policy enforcement enabled
- [ ] Basic documentation for product teams
- [ ] 3 pilot product teams onboarded

### Phase 2: Developer Experience (Target: 2 weeks)
- [ ] CLI commands for workflow management
- [ ] API endpoints for workflow discovery
- [ ] Web UI for workflow inspection
- [ ] Self-service onboarding

### Phase 3: Advanced Features (Target: 4 weeks)
- [ ] Custom provisioner registration
- [ ] Product-specific resource types
- [ ] Workflow templates and generators
- [ ] Multi-tenant support

**Track Progress:** See [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md) for detailed status

---

## Documentation

| Guide | Description | Status |
|-------|-------------|--------|
| **[Product Workflows](product-workflows.md)** | Create and manage workflows | 🚧 Draft |
| **[Custom Resources](custom-resources.md)** | Define resource types | 🚧 Draft |
| **[Provisioners](provisioners.md)** | Build custom provisioners | 🚧 Planned |
| **[Governance](governance.md)** | Approval workflows and policies | 🚧 Draft |
| **[Activation Guide](activation-guide.md)** | For platform teams to enable feature | 🚧 Draft |
| **[Examples](examples.md)** | Real-world product workflows | 🚧 Draft |

---

## Getting Help

**For Product Teams:**
1. **Check examples:** Review existing workflows in `workflows/products/`
2. **Read gap analysis:** [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md)
3. **Contact platform team:** Your platform team manages innominatus

**For Platform Teams:**
1. **Activation steps:** See [Activation Guide](activation-guide.md)
2. **Technical details:** See [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md)
3. **Implementation:** Review GAP 1-6 in gap analysis

---

## Frequently Asked Questions

### Q: Can I use this feature today?

**A:** You can create workflow files and test with `workflow-demo.go`, but workflows won't execute in production deployments until the multi-tier executor is activated by your platform team.

### Q: When will this be production-ready?

**A:** The core implementation exists. Platform teams can activate it with a small code change (GAP 1). See [Activation Guide](activation-guide.md).

### Q: How do I get my workflows approved?

**A:** Currently, submit a pull request adding your workflows to `workflows/products/` and updating `admin-config.yaml`. Platform team reviews and approves. (Self-service tooling coming in Phase 2)

### Q: Can I create custom resource types?

**A:** Not yet. SDK interface exists (`pkg/sdk/provisioner.go`) but provisioner registry is not implemented (GAP 5). Planned for Phase 3.

### Q: How does this differ from platform extensions?

**A:** Platform extensions add infrastructure platforms (AWS, Azure). Product workflows add product-specific logic that runs during app deployments. See [Platform Extension Guide](../PLATFORM_EXTENSION_GUIDE.md).

### Q: Is this multi-tenant?

**A:** Not yet. All products share the same `workflows/products/` directory. Multi-tenancy is a future enhancement.

---

## Next Steps

### For Product Teams

1. **Review examples:** Study existing workflows in `workflows/products/ecommerce/` and `workflows/products/analytics/`
2. **Identify use cases:** What should happen when apps use your product?
3. **Draft workflows:** Create YAML files following examples
4. **Contact platform team:** Request review and activation timeline
5. **Test with demo:** Validate workflow structure with `workflow-demo.go`

### For Platform Teams

1. **Review gap analysis:** Read [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md)
2. **Plan activation:** Estimate effort for GAP 1 + GAP 3 (critical gaps)
3. **Pilot with 1-2 teams:** Choose product teams for early testing
4. **Follow activation guide:** Implement multi-tier executor (see [Activation Guide](activation-guide.md))
5. **Enable production:** Roll out to all product teams

---

**Questions?** Contact your platform team or open an issue on GitHub.

**Contributing?** See [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md) for implementation opportunities.
