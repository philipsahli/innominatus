# Product Team Guide

**Welcome, Product Team!** 🛠️

This guide is for **internal product engineering teams** who build services consumed by application developers (e.g., payments, analytics, ML platforms).

---

## ✅ Current Status: Available

**Important:** Product team provider capabilities are **implemented and available** for production use.

**What's Available:**
- ✅ Multi-team provider architecture (4 demo providers included)
- ✅ Git-based provider loading from remote repositories
- ✅ API integration with server endpoints
- ✅ PostgreSQL Operator auto-installation with demo environment
- ✅ Vault Secrets Operator (VSO) for secret synchronization
- ✅ Complete demo environment via `innominatus-ctl demo-time`
- ✅ Golden path workflows for developer onboarding

**Demo Providers Included:**
- `container-team` - Container registry and image management
- `database-team` - PostgreSQL databases via operator
- `storage-team` - MinIO object storage buckets
- `vault-team` - Secret management via Vault + VSO

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

## Core Capabilities

### 1. Git-Based Provider Loading

Providers can be loaded from Git repositories and registered via admin configuration:

**Provider Structure:**
```
providers/database-team/
├── provider.yaml          # Provider metadata
└── workflows/
    ├── create-postgres.yaml
    └── delete-postgres.yaml
```

**Admin Registration:**
```yaml
providers:
  - name: database-team
    git_url: https://github.com/yourorg/providers
    path: providers/database-team
    ref: main
```

**Available via API:** `GET /api/providers` and `POST /api/admin/providers`

### 2. Golden Path Workflows

Complete multi-step workflows that orchestrate infrastructure provisioning:

**Example:** `workflows/onboard-dev-team.yaml` demonstrates a full developer onboarding flow:
- Creates Gitea repository
- Provisions PostgreSQL database via operator
- Creates MinIO storage bucket
- Configures Vault secrets with VSO synchronization
- Deploys application to Kubernetes

**Run with CLI:**
```bash
./innominatus-ctl run onboard-dev-team score-spec.yaml
```

### 3. Provider Workflows

Each provider team defines workflows for their domain:

**Database Team Example:** `providers/database-team/workflows/create-postgres.yaml`
- Creates PostgreSQL CRD via operator
- Waits for database to be ready
- Stores credentials in Vault
- Configures VSO for secret sync

**Storage Team Example:** `providers/storage-team/workflows/create-bucket.yaml`
- Creates MinIO bucket
- Generates access credentials
- Stores credentials in Vault
- Sets up bucket policies

**Vault Team Example:** `providers/vault-team/workflows/create-secrets.yaml`
- Creates Vault secret path
- Generates VSO ExternalSecret CRD
- Syncs secrets to Kubernetes

### 4. Integrated Demo Environment

Complete multi-team infrastructure available via:

```bash
./innominatus-ctl demo-time    # Install all demo components
./innominatus-ctl demo-status  # Check health
```

**Includes:**
- **Gitea** - Git repository (http://gitea.localtest.me)
- **ArgoCD** - GitOps deployment (http://argocd.localtest.me)
- **Vault** - Secret management (http://vault.localtest.me)
- **MinIO** - Object storage (http://minio.localtest.me)
- **PostgreSQL Operator** - Database provisioning
- **Vault Secrets Operator** - Secret synchronization

---

## Completed Features ✅

The following features have been implemented and are available:

### Multi-Team Provider Architecture
- 4 demo providers included (container, database, storage, vault teams)
- Provider directory structure with workflows subdirectory
- Provider metadata via `provider.yaml` files

### Git-Based Provider Loading
- Load providers from remote Git repositories
- Register providers via admin configuration
- API endpoints for provider management

### Workflow Execution
- Golden path workflows via CLI (`innominatus-ctl run`)
- Multi-step workflow orchestration
- Support for Terraform, Kubernetes, Ansible, and shell commands

### Demo Infrastructure
- Complete demo environment via `demo-time` command
- PostgreSQL Operator auto-installation
- Vault Secrets Operator for secret synchronization
- Integrated services (Gitea, ArgoCD, Vault, MinIO)

### API Integration
- Provider management endpoints
- Workflow execution APIs
- Admin configuration endpoints

---

## Future Enhancements 🚧

### Custom Resource Types
Define product-specific resource types in Score specs:
```yaml
resources:
  payment-gateway:
    type: payment-gateway  # Custom type from product team
    properties:
      provider: stripe
```

**Status:** SDK interface exists, provisioner registry not yet implemented

### Enhanced CLI Tooling
Additional commands for workflow management:
```bash
innominatus-ctl list-providers
innominatus-ctl validate-provider ./my-provider
innominatus-ctl test-workflow my-workflow.yaml --dry-run
```

**Status:** Basic workflow commands available, enhanced tooling planned

### Multi-Tenant Support
Isolated provider spaces per team with RBAC:
- Team-scoped provider registration
- Workflow execution permissions
- Resource quota enforcement

**Status:** Single-tenant currently, multi-tenancy planned for future release

### Workflow Policy Enforcement
Runtime policy validation:
- Allowed step types per team
- Resource limits per workflow
- Approval gates for sensitive operations

**Status:** Policy framework exists, enforcement planned

---

## Getting Started

### Prerequisites

- **Access to innominatus instance** - Server running with admin permissions
- **Docker Desktop with Kubernetes** - For demo environment (optional but recommended)
- **Git repository** - For storing provider configurations (optional)

### Option 1: Use Demo Environment (Recommended)

Start with the complete demo environment to see providers in action:

```bash
# Install demo environment
./innominatus-ctl demo-time

# Check status
./innominatus-ctl demo-status

# Explore demo providers
ls providers/
# Output: container-team/  database-team/  storage-team/  vault-team/
```

**Demo Components:**
- Gitea repository at http://gitea.localtest.me (admin/admin)
- Vault at http://vault.localtest.me (root token)
- MinIO at http://minio.localtest.me (minioadmin/minioadmin)
- PostgreSQL Operator (auto-installed)
- Vault Secrets Operator (auto-installed)

### Option 2: Create Your Own Provider

#### Step 1: Create Provider Structure

```bash
mkdir -p providers/your-team
cd providers/your-team
```

#### Step 2: Create Provider Metadata

**File:** `providers/your-team/provider.yaml`

```yaml
name: your-team
description: Your team's infrastructure services
version: 1.0.0
owner: your-team@company.com
workflows_dir: workflows
```

#### Step 3: Create Your First Workflow

**File:** `providers/your-team/workflows/provision-service.yaml`

```yaml
name: provision-service
description: Provision service infrastructure
steps:
  - name: create-namespace
    type: kubernetes
    config:
      manifest: |
        apiVersion: v1
        kind: Namespace
        metadata:
          name: ${APP_NAME}

  - name: deploy-service
    type: kubernetes
    config:
      namespace: ${APP_NAME}
      manifest_path: ./manifests/service.yaml
```

#### Step 4: Register Provider

**Via Admin API:**
```bash
curl -X POST http://localhost:8081/api/admin/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "your-team",
    "git_url": "https://github.com/yourorg/providers",
    "path": "providers/your-team",
    "ref": "main"
  }'
```

**Or via local directory:**
Place your provider in `providers/your-team/` and restart the server.

#### Step 5: Test Your Provider

```bash
# List available providers
./innominatus-ctl list-goldenpaths

# Run workflow
./innominatus-ctl run your-team/provision-service examples/app-spec.yaml
```

### Option 3: Study Demo Providers

Explore the included demo providers for complete examples:

**Database Team:** `providers/database-team/`
- Creates PostgreSQL databases via operator
- Stores credentials in Vault
- Configures VSO for secret sync

**Storage Team:** `providers/storage-team/`
- Creates MinIO buckets
- Manages access policies
- Integrates with Vault

**Vault Team:** `providers/vault-team/`
- Manages secret paths
- Creates VSO ExternalSecrets
- Syncs to Kubernetes

**Container Team:** `providers/container-team/`
- Registry management
- Image policies
- Harbor integration

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

## Real-World Examples

### Example 1: Database Team - PostgreSQL Provisioning

**Location:** `providers/database-team/workflows/create-postgres.yaml`

**What it does:**
1. Creates PostgreSQL custom resource via operator
2. Waits for database pod to be ready
3. Retrieves connection credentials from Kubernetes secret
4. Stores credentials in Vault
5. Creates VSO ExternalSecret for automatic sync

**Key Features:**
- Operator-based database management
- Automatic credential rotation
- Kubernetes-native secret sync

**Try it:**
```bash
./innominatus-ctl demo-time  # Install postgres-operator
./innominatus-ctl run database-team/create-postgres examples/dev-team-app.yaml
```

### Example 2: Storage Team - MinIO Bucket Creation

**Location:** `providers/storage-team/workflows/create-bucket.yaml`

**What it does:**
1. Creates MinIO bucket with versioning enabled
2. Generates access key and secret key
3. Stores credentials in Vault
4. Configures bucket policies for application access

**Key Features:**
- S3-compatible object storage
- Automatic policy management
- Secure credential storage

**Try it:**
```bash
./innominatus-ctl run storage-team/create-bucket examples/dev-team-app.yaml
```

### Example 3: Complete Developer Onboarding

**Location:** `workflows/onboard-dev-team.yaml`

**What it does:**
1. Creates Gitea repository for application code
2. Provisions PostgreSQL database
3. Creates MinIO storage bucket
4. Sets up Vault secret paths
5. Configures VSO for secret synchronization
6. Deploys application to Kubernetes with ArgoCD

**Key Features:**
- End-to-end automation
- Multi-provider orchestration
- GitOps integration

**Try it:**
```bash
./innominatus-ctl demo-time  # Required
./innominatus-ctl run onboard-dev-team examples/dev-team-app.yaml
```

---

## Development Progress

### ✅ Phase 1: Core Infrastructure (Completed)
- ✅ Multi-team provider architecture
- ✅ Git-based provider loading
- ✅ Workflow execution engine
- ✅ 4 demo providers (database, storage, vault, container)
- ✅ PostgreSQL Operator integration
- ✅ Vault Secrets Operator integration
- ✅ Demo environment automation

### ✅ Phase 2: Developer Experience (Completed)
- ✅ CLI commands for workflow execution
- ✅ API endpoints for provider management
- ✅ Golden path workflows
- ✅ Complete onboarding automation
- ✅ Demo environment tooling

### 🚧 Phase 3: Enterprise Features (In Progress)
- [ ] Custom provisioner SDK
- [ ] Product-specific resource types in Score specs
- [ ] Workflow templates and generators
- [ ] Multi-tenant RBAC
- [ ] Runtime policy enforcement
- [ ] Approval gates and governance
- [ ] Advanced monitoring and observability

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
1. **Review demo providers:** Study working examples in `providers/`
2. **Read detailed guides:** [Product Workflows Guide](product-workflows.md)
3. **Contact platform team:** Your platform team manages the innominatus instance
4. **Join community:** GitHub issues and discussions

**For Platform Teams:**
1. **Deployment guide:** See [Kubernetes Deployment](../platform-team-guide/kubernetes-deployment.md)
2. **Activation guide:** [Product Team Activation](activation-guide.md)
3. **OIDC setup:** Configure authentication for your organization
4. **Monitor system:** Use `/health`, `/ready`, and `/metrics` endpoints

---

## Frequently Asked Questions

### Q: Can I use this feature today?

**A:** Yes! Provider capabilities are fully functional. Start with `./innominatus-ctl demo-time` to explore the demo environment, or create your own provider following the Getting Started guide.

### Q: How do I create a new provider?

**A:** Create a provider directory with `provider.yaml` and `workflows/` subdirectory. Register it via admin API or place it in the `providers/` directory. See "Getting Started → Option 2" above.

### Q: Where can I see working examples?

**A:** Check the 4 demo providers:
- `providers/database-team/` - PostgreSQL via operator
- `providers/storage-team/` - MinIO object storage
- `providers/vault-team/` - Secret management
- `providers/container-team/` - Container registry

### Q: Can I load providers from Git?

**A:** Yes! Providers can be loaded from Git repositories. Register them via the admin API with `git_url`, `path`, and `ref` parameters.

### Q: Can I create custom resource types?

**A:** Not yet. The SDK interface exists but the provisioner registry is not fully implemented. This is planned for Phase 3 (Enterprise Features).

### Q: How does this differ from Score spec deployments?

**A:** Score specs describe applications. Providers offer services (databases, storage, secrets) that applications consume. Providers run workflows to provision these services automatically.

### Q: Is this multi-tenant?

**A:** Currently single-tenant. All providers share the same namespace. Multi-tenant RBAC with team isolation is planned for Phase 3.

---

## Next Steps

### For Product Teams

1. **Try the demo:** Run `./innominatus-ctl demo-time` to explore the full demo environment
2. **Study examples:** Review the 4 demo providers in `providers/`
3. **Identify your services:** What infrastructure do you provide to app developers?
4. **Create your provider:** Follow "Getting Started → Option 2" to build your first provider
5. **Test workflows:** Use `./innominatus-ctl run` to test your workflows

### For Platform Teams

1. **Deploy innominatus:** Set up the server with Kubernetes or standalone
2. **Configure OIDC:** Set up authentication for your organization
3. **Register providers:** Add product team providers via admin API or Git
4. **Set up demo:** Install demo environment for product teams to explore
5. **Enable self-service:** Configure admin settings for provider registration

---

**Questions?** Contact your platform team or open an issue on GitHub.

**Want to contribute?** Provider examples and workflow patterns are welcome! Submit PRs with new provider implementations or workflow improvements.
