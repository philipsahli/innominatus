# Provider Workflows Guide

**Audience:** Product Teams
**Status:** ✅ Available
**Last Updated:** 2025-10-29

---

## Overview

Provider workflows enable product teams to offer infrastructure services (databases, storage, secrets) to application developers through automated provisioning. This guide covers how to create, test, and deploy provider workflows.

**What You'll Learn:**
- Provider structure and metadata
- Workflow development patterns
- Variable interpolation
- Testing with demo environment
- Real-world examples from demo providers

---

## Quick Start

### 1. Create Provider Structure

**Location:** `providers/{your-team}/`

**Structure:**
```
providers/your-team/
├── provider.yaml          # Provider metadata
└── workflows/             # Workflow definitions
    ├── create-resource.yaml
    └── delete-resource.yaml
```

### 2. Create Provider Metadata

**File:** `providers/your-team/provider.yaml`

```yaml
name: your-team
description: Your infrastructure services
version: 1.0.0
owner: your-team@company.com
workflows_dir: workflows
supported_resources:
  - your-resource-type
tags:
  - infrastructure
  - automation
```

### 3. Create Your First Workflow

**File:** `providers/your-team/workflows/create-resource.yaml`

```yaml
name: create-resource
description: Provision resource for application
steps:
  - name: create-namespace
    type: kubernetes
    config:
      manifest: |
        apiVersion: v1
        kind: Namespace
        metadata:
          name: ${APP_NAME}
          labels:
            managed-by: innominatus
            team: your-team

  - name: deploy-resource
    type: kubernetes
    config:
      namespace: ${APP_NAME}
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: ${APP_NAME}-config
        data:
          resource_url: "https://your-service.com/${APP_NAME}"
```

### 4. Test Your Workflow

**Using demo environment:**

```bash
# Install demo environment
./innominatus-ctl demo-time

# Test your workflow
./innominatus-ctl run your-team/create-resource examples/dev-team-app.yaml

# Check workflow status
./innominatus-ctl list-workflows
```

**Expected output:**
```
Workflow: your-team/create-resource
Status: Running
Steps:
  ✅ create-namespace (completed)
  🔄 deploy-resource (running)
```

### 5. Deploy Provider

**Option A - Local deployment:**
```bash
# Place provider in providers/ directory
cp -r providers/your-team /path/to/innominatus/providers/

# Restart server to load provider
pkill innominatus && ./innominatus
```

**Option B - Git-based deployment:**
```bash
# Register via admin API
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

Once merged and server restarted:

```bash
# Deploy test app
cat > test-payments-app.yaml <<EOF
apiVersion: score.dev/v1b1
metadata:
  name: test-checkout
  product: payments  # Triggers your workflow
containers:
  web:
    image: nginx:latest
EOF

# Deploy via API
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @test-payments-app.yaml

# Check workflows executed
curl http://localhost:8081/api/workflows?app=test-checkout
```

---

## Workflow Structure

### Complete YAML Schema

```yaml
apiVersion: workflow.dev/v1       # Required: API version
kind: ProductWorkflow             # Required: Must be "ProductWorkflow"

metadata:
  name: string                    # Required: Unique within product
  description: string             # Required: What this workflow does
  product: string                 # Required: Product name (matches directory)
  owner: string                   # Required: Team or contact
  phase: enum                     # Required: pre-deployment | deployment | post-deployment

spec:
  triggers: array                 # Required: When to run
    - product_deployment          # Run for all apps with this product
    - first_deployment            # Only first deployment of an app
    - manual                      # Only when explicitly triggered

  steps: array                    # Required: Workflow steps
    - name: string                # Required: Step name
      type: string                # Required: Step type (see allowed types)
      resource: string            # Optional: Resource name to track
      when: string                # Optional: Conditional execution
      config: object              # Optional: Step configuration
      env: object                 # Optional: Environment variables
      namespace: string           # Optional: Kubernetes namespace
```

### Metadata Fields

#### name (required)
Unique identifier within your product.

**Good:**
```yaml
name: payment-gateway-setup
name: database-migration
name: monitoring-config
```

**Bad:**
```yaml
name: setup          # Too generic
name: step1          # Not descriptive
name: workflow       # Meaningless
```

#### description (required)
Clear explanation of what the workflow does.

**Good:**
```yaml
description: Configure Stripe payment gateway and webhook endpoints
description: Run database migrations for analytics schema
description: Set up Prometheus monitoring for ML model serving
```

**Bad:**
```yaml
description: Setup
description: Do stuff
description: Workflow for product
```

#### product (required)
Must match the directory name in `workflows/products/{product}/`.

**Example:**
- Directory: `workflows/products/payments/`
- YAML: `product: payments` ✅

#### owner (required)
Team or individual responsible for this workflow.

**Examples:**
```yaml
owner: payments-infrastructure-team
owner: analytics-platform-team
owner: ml-ops-team
owner: data-engineering@company.com
```

#### phase (required)
When the workflow executes relative to application deployment.

| Phase | Purpose | Example Use Cases |
|-------|---------|-------------------|
| **pre-deployment** | Setup before app deploys | Provision databases, create namespaces, configure secrets |
| **deployment** | Run during deployment | Configure services, setup integrations, deploy sidecars |
| **post-deployment** | Run after app is deployed | Health checks, smoke tests, monitoring setup |

**Example:**
```yaml
# Database provisioning happens first
phase: pre-deployment

# Gateway configuration during deployment
phase: deployment

# Monitoring setup after app is running
phase: post-deployment
```

---

## Workflow Triggers

### product_deployment
Runs for every deployment of an app with matching product metadata.

**Use when:** Workflow should run every time (most common)

**Example:**
```yaml
spec:
  triggers:
    - product_deployment
```

**Triggered by:**
```yaml
# Score spec
metadata:
  product: payments
```

### first_deployment
Runs only the first time an application is deployed.

**Use when:** One-time setup (database schema, initial configuration)

**Example:**
```yaml
spec:
  triggers:
    - first_deployment
```

### manual
Only runs when explicitly triggered (not automatic).

**Use when:** Maintenance workflows, manual operations

**Example:**
```yaml
spec:
  triggers:
    - manual
```

**Note:** Manual triggering API not yet implemented (see US-007)

---

## Workflow Steps

### Allowed Step Types

Per `admin-config.yaml`, only these step types are allowed:

```yaml
allowedStepTypes:
  - terraform          # Infrastructure provisioning
  - kubernetes         # K8s resource deployment
  - ansible            # Configuration management
  - database-migration # Database schema changes
  - vault-setup        # Secret management
  - monitoring         # Observability setup
  - validation         # Health checks, testing
  - security           # Security scanning
  - policy             # Policy enforcement
  - tagging            # Resource tagging
  - cost-analysis      # Cost estimation
  - resource-provisioning  # Generic provisioning
```

**Using an unlisted step type will fail policy validation (when US-006 is active).**

### Step Examples

#### 1. Vault Setup (Secrets)

```yaml
- name: create-secrets
  type: vault-setup
  config:
    secrets:
      - stripe-api-key
      - paypal-client-secret
      - webhook-signing-key
    path: "secret/apps/${application.name}/payments"
    policies:
      - payment-service-read
      - payment-audit-write
```

#### 2. Kubernetes Deployment

```yaml
- name: deploy-sidecar
  type: kubernetes
  namespace: "${application.name}"
  config:
    manifests: "./k8s/payment-sidecar"
    variables:
      APP_NAME: "${application.name}"
      ENVIRONMENT: "${workflow.ENVIRONMENT}"
      GATEWAY_URL: "${resources.payment-gateway.url}"
```

#### 3. Database Migration

```yaml
- name: run-migrations
  type: database-migration
  config:
    connectionString: "${resources.database.connection_string}"
    migrationsPath: "./migrations/payments"
    direction: up
    version: latest
```

#### 4. Validation

```yaml
- name: health-check
  type: validation
  config:
    healthChecks:
      - stripe-connectivity
      - paypal-connectivity
      - database-connectivity
    timeout: "30s"
    retries: 3
```

#### 5. Monitoring Setup

```yaml
- name: configure-monitoring
  type: monitoring
  config:
    service: "payment-gateway"
    alerts:
      - connection_count
      - error_rate
      - transaction_latency
    dashboards:
      - payment-overview
      - transaction-details
```

#### 6. Terraform Provisioning

```yaml
- name: provision-load-balancer
  type: terraform
  config:
    operation: apply
    working_dir: ./terraform/payment-lb
    variables:
      app_name: "${application.name}"
      environment: "${workflow.ENVIRONMENT}"
    outputs:
      - lb_dns_name
      - lb_arn
```

---

## Variable Interpolation

### Available Variables

Use `${variable.path}` syntax to access:

#### Application Variables

```yaml
${application.name}                 # App name from Score spec
${application.metadata.team}        # Team from metadata
${application.metadata.costCenter}  # Cost center from metadata
${application.metadata.product}     # Product name
```

#### Resource Variables

```yaml
${resources.database.connection_string}  # Database connection
${resources.database.host}               # Database host
${resources.database.port}               # Database port
${resources.vault-space.path}            # Vault path
${resources.route.host}                  # Route hostname
```

#### Workflow Variables

```yaml
${workflow.ENVIRONMENT}      # Environment name
${workflow.REGION}           # Region
${workflow.COST_CENTER}      # Cost center
```

#### Terraform Outputs (from previous steps)

```yaml
# Step 1: Provision with Terraform
- name: provision-db
  type: terraform
  config:
    outputs:
      - db_host
      - db_port

# Step 2: Use Terraform outputs
- name: configure-app
  type: kubernetes
  env:
    DATABASE_HOST: "${terraform.db_host}"
    DATABASE_PORT: "${terraform.db_port}"
```

### Example: Complete Variable Usage

```yaml
steps:
  - name: provision-infrastructure
    type: terraform
    config:
      working_dir: ./terraform/payments
      variables:
        app_name: "${application.name}"
        team: "${application.metadata.team}"
        cost_center: "${application.metadata.costCenter}"
      outputs:
        - gateway_url
        - api_key_id

  - name: configure-application
    type: kubernetes
    namespace: "${application.name}"
    env:
      PAYMENT_GATEWAY_URL: "${terraform.gateway_url}"
      API_KEY_ID: "${terraform.api_key_id}"
      DATABASE_URL: "${resources.database.connection_string}"
      VAULT_PATH: "${resources.vault-space.path}"
```

---

## Conditional Execution

### Basic Conditionals

**Not yet fully implemented (see GAP in PRODUCT_WORKFLOW_GAPS.md)**

**Planned syntax:**

```yaml
- name: production-only-step
  type: monitoring
  when: "${workflow.ENVIRONMENT} == 'production'"
  config:
    alerts: high-priority

- name: development-skip
  type: security
  unless: "${workflow.ENVIRONMENT} == 'development'"
  config:
    scanner: deep-scan
```

**Current workaround:** Create separate workflows per environment

---

## Real-World Examples

### Example 1: Database Team - PostgreSQL Provisioning

**Provider:** `database-team`
**File:** `providers/database-team/workflows/create-postgres.yaml`

This workflow uses the CloudNativePG operator to provision PostgreSQL databases:

```yaml
name: create-postgres
description: Create PostgreSQL database via CloudNativePG operator
steps:
  - name: create-postgres-cluster
    type: kubernetes
    config:
      manifest: |
        apiVersion: postgresql.cnpg.io/v1
        kind: Cluster
        metadata:
          name: ${APP_NAME}-db
          namespace: ${NAMESPACE}
        spec:
          instances: 1
          storage:
            size: 1Gi
          postgresql:
            parameters:
              max_connections: "100"
              shared_buffers: "256MB"

  - name: wait-for-database-ready
    type: shell
    config:
      command: |
        kubectl wait --for=condition=Ready \
          cluster/${APP_NAME}-db \
          -n ${NAMESPACE} \
          --timeout=300s
        echo "Database cluster is ready"

  - name: get-database-credentials
    type: shell
    config:
      command: |
        # Get credentials from operator-created secret
        kubectl get secret ${APP_NAME}-db-app \
          -n ${NAMESPACE} \
          -o jsonpath='{.data.username}' | base64 -d > /tmp/db-user
        kubectl get secret ${APP_NAME}-db-app \
          -n ${NAMESPACE} \
          -o jsonpath='{.data.password}' | base64 -d > /tmp/db-pass

        echo "Database credentials retrieved"

  - name: store-credentials-in-vault
    type: shell
    config:
      command: |
        # Store credentials in Vault for application access
        vault kv put secret/${APP_NAME}/database \
          username=$(cat /tmp/db-user) \
          password=$(cat /tmp/db-pass) \
          host=${APP_NAME}-db-rw.${NAMESPACE}.svc.cluster.local \
          port=5432 \
          database=app

  - name: create-vso-externalsecret
    type: kubernetes
    config:
      manifest: |
        apiVersion: secrets.hashicorp.com/v1beta1
        kind: VaultStaticSecret
        metadata:
          name: ${APP_NAME}-db-credentials
          namespace: ${NAMESPACE}
        spec:
          vaultAuthRef: default
          mount: secret
          path: ${APP_NAME}/database
          refreshAfter: 1h
          destination:
            name: ${APP_NAME}-db-credentials
            create: true
```

**Key Features:**
- Operator-based database provisioning
- Automatic credential management
- Vault integration for secret storage
- VSO for Kubernetes secret synchronization

**Try it:**
```bash
./innominatus-ctl demo-time
./innominatus-ctl run database-team/create-postgres examples/dev-team-app.yaml
```

---

### Example 2: Storage Team - MinIO Bucket Provisioning

**Provider:** `storage-team`
**File:** `providers/storage-team/workflows/create-bucket.yaml`

This workflow provisions S3-compatible object storage using MinIO:

```yaml
name: create-bucket
description: Create MinIO bucket for application storage
steps:
  - name: create-minio-bucket
    type: shell
    config:
      command: |
        # Configure MinIO client
        mc alias set minio http://minio.demo.svc.cluster.local:9000 \
          minioadmin minioadmin

        # Create bucket
        mc mb minio/${APP_NAME}-bucket --ignore-existing

        # Enable versioning
        mc version enable minio/${APP_NAME}-bucket

        echo "Bucket created: ${APP_NAME}-bucket"

  - name: generate-access-credentials
    type: shell
    config:
      command: |
        # Generate service account for application
        mc admin user svcacct add minio minioadmin \
          --access-key ${APP_NAME}-access \
          --secret-key $(openssl rand -base64 32) \
          --policy readwrite

        # Store for next step
        mc admin user svcacct info minio ${APP_NAME}-access --json > /tmp/minio-creds.json

  - name: set-bucket-policy
    type: shell
    config:
      command: |
        # Create policy for application access
        cat > /tmp/bucket-policy.json <<EOF
        {
          "Version": "2012-10-17",
          "Statement": [{
            "Effect": "Allow",
            "Principal": {"AWS": ["arn:aws:iam:::user/${APP_NAME}-access"]},
            "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
            "Resource": ["arn:aws:s3:::${APP_NAME}-bucket/*"]
          }]
        }
        EOF

        mc anonymous set-json /tmp/bucket-policy.json minio/${APP_NAME}-bucket

  - name: store-credentials-in-vault
    type: shell
    config:
      command: |
        ACCESS_KEY=$(jq -r '.accessKey' /tmp/minio-creds.json)
        SECRET_KEY=$(jq -r '.secretKey' /tmp/minio-creds.json)

        vault kv put secret/${APP_NAME}/storage \
          access_key=$ACCESS_KEY \
          secret_key=$SECRET_KEY \
          bucket=${APP_NAME}-bucket \
          endpoint=http://minio.demo.svc.cluster.local:9000
```

**Key Features:**
- S3-compatible object storage
- Automatic credential generation
- Bucket policy management
- Vault integration

**Try it:**
```bash
./innominatus-ctl run storage-team/create-bucket examples/dev-team-app.yaml
```

---

### Example 3: Complete Developer Onboarding Golden Path

**File:** `workflows/onboard-dev-team.yaml`

This comprehensive workflow orchestrates multiple providers for complete developer onboarding:

```yaml
name: onboard-dev-team
description: Complete developer team onboarding with infrastructure provisioning
steps:
  - name: create-gitea-repository
    type: shell
    config:
      command: |
        # Create Git repository for application code
        curl -X POST http://gitea.demo.svc.cluster.local:3000/api/v1/user/repos \
          -H "Authorization: token ${GITEA_TOKEN}" \
          -H "Content-Type: application/json" \
          -d '{"name": "${APP_NAME}", "private": false}'

  - name: provision-postgres-database
    type: workflow
    config:
      provider: database-team
      workflow: create-postgres
      variables:
        APP_NAME: ${APP_NAME}
        NAMESPACE: ${NAMESPACE}

  - name: provision-minio-bucket
    type: workflow
    config:
      provider: storage-team
      workflow: create-bucket
      variables:
        APP_NAME: ${APP_NAME}

  - name: setup-vault-secrets
    type: workflow
    config:
      provider: vault-team
      workflow: create-secrets
      variables:
        APP_NAME: ${APP_NAME}
        NAMESPACE: ${NAMESPACE}

  - name: deploy-application
    type: kubernetes
    config:
      namespace: ${NAMESPACE}
      manifest: |
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: ${APP_NAME}
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: ${APP_NAME}
          template:
            metadata:
              labels:
                app: ${APP_NAME}
            spec:
              containers:
              - name: app
                image: nginx:latest
                envFrom:
                - secretRef:
                    name: ${APP_NAME}-db-credentials
                - secretRef:
                    name: ${APP_NAME}-storage-credentials

  - name: create-argocd-application
    type: kubernetes
    config:
      manifest: |
        apiVersion: argoproj.io/v1alpha1
        kind: Application
        metadata:
          name: ${APP_NAME}
          namespace: argocd
        spec:
          project: default
          source:
            repoURL: http://gitea.demo.svc.cluster.local:3000/admin/${APP_NAME}
            path: k8s
            targetRevision: HEAD
          destination:
            server: https://kubernetes.default.svc
            namespace: ${NAMESPACE}
          syncPolicy:
            automated:
              prune: true
              selfHeal: true
```

**What This Provides:**
- Git repository for code
- PostgreSQL database via operator
- S3-compatible object storage
- Secret management via Vault + VSO
- Kubernetes deployment
- GitOps with ArgoCD

**Try it:**
```bash
./innominatus-ctl demo-time
./innominatus-ctl run onboard-dev-team examples/dev-team-app.yaml
```

---

## Best Practices

### 1. Keep Workflows Focused

**Good - Single Responsibility:**
```yaml
# workflows/products/payments/gateway-setup.yaml
name: gateway-setup
description: Configure payment gateway

# workflows/products/payments/fraud-detection.yaml
name: fraud-detection
description: Setup fraud detection rules
```

**Bad - Too Much in One:**
```yaml
# workflows/products/payments/everything.yaml
name: everything
description: Setup gateway, fraud detection, reporting, and compliance
# 50 steps...
```

### 2. Use Descriptive Step Names

**Good:**
```yaml
- name: provision-postgres-database
- name: run-schema-migrations
- name: verify-database-connectivity
```

**Bad:**
```yaml
- name: step1
- name: db
- name: check
```

### 3. Set Appropriate Timeouts

```yaml
- name: quick-validation
  type: validation
  config:
    timeout: "30s"  # Fast operations

- name: database-backup
  type: ansible
  config:
    timeout: "10m"  # Longer operations
```

### 4. Handle Errors Gracefully

```yaml
- name: optional-monitoring
  type: monitoring
  config:
    continueOnFailure: true  # Don't fail deployment if monitoring setup fails
    alerts: ["error_rate"]

- name: critical-security-scan
  type: security
  config:
    failOnVulnerabilities: true  # Fail deployment on security issues
```

### 5. Document Dependencies

```yaml
metadata:
  description: |
    Configures payment gateway integration.

    Dependencies:
    - Vault must be accessible
    - Kubernetes namespace must exist
    - Payment provider API keys in Vault

    Outputs:
    - Payment gateway URL
    - Webhook endpoint
```

---

## Testing Workflows

### Current Testing Method (workflow-demo.go)

**Step 1:** Create test application instance

```go
// workflow-demo.go
app := &workflow.ApplicationInstance{
    ID:   1,
    Name: "test-payments-app",
    Configuration: map[string]interface{}{
        "metadata": map[string]interface{}{
            "product":    "payments",
            "team":       "checkout-team",
            "costCenter": "engineering",
        },
    },
    Resources: []workflow.ResourceRef{
        {ResourceName: "database", ResourceType: "postgres"},
    },
}
```

**Step 2:** Run demo

```bash
go run workflow-demo.go
```

**Step 3:** Check output

```
🔄 Resolving Multi-Tier Workflows...
📊 Workflow Resolution Results:
  deployment Phase (1 workflows):
    🔧 product-payments-gateway-setup (3 steps)
       Step 1: create-vault-secrets (vault-setup)
       Step 2: configure-gateway (kubernetes)
       Step 3: verify-connectivity (validation)
✅ All workflow policies validated successfully!
```

### Future Testing (US-008 - CLI Commands)

**Coming soon:**

```bash
# Validate workflow YAML
innominatus-ctl validate-product-workflow workflows/products/payments/gateway-setup.yaml

# Test workflow execution (dry-run)
innominatus-ctl test-product-workflow \
  workflows/products/payments/gateway-setup.yaml \
  test-app.yaml \
  --dry-run

# Output:
# ✅ Workflow structure valid
# ✅ All step types allowed
# 📋 Steps to execute:
#     1. create-vault-secrets (vault-setup)
#     2. configure-gateway (kubernetes)
#     3. verify-connectivity (validation)
```

---

## Troubleshooting

### Workflow Not Executing

**Check 1: Product metadata present?**

```yaml
# Score spec must include:
metadata:
  product: payments  # Must match your product directory
```

**Check 2: Workflow in allowed list?**

```bash
grep "allowedProductWorkflows" admin-config.yaml
# Must include: payments/gateway-setup
```

**Check 3: Multi-tier executor enabled?**

```bash
# Check server logs
tail -f innominatus.log | grep "Multi-tier"
# Expected: "✅ Multi-tier workflow executor enabled"
```

### Workflow Fails Immediately

**Check step types:**

```yaml
# All step types must be in admin-config.yaml allowedStepTypes
allowedStepTypes:
  - terraform
  - kubernetes
  # ... etc
```

**Check variable syntax:**

```yaml
# Correct:
env:
  VAR: "${application.name}"

# Wrong:
env:
  VAR: "$application.name"     # Missing braces
  VAR: "{application.name}"    # Missing $
```

### Variables Not Substituted

**Current limitation:** Variable substitution in some contexts not fully implemented (GAP in PRODUCT_WORKFLOW_GAPS.md)

**Workaround:** Use environment variables passed to steps

---

## Limitations & Roadmap

### Current Limitations (2025-10-19)

1. **❌ No CLI validation** (US-008 in progress)
   - Can't validate YAML before deployment
   - Workaround: Use `workflow-demo.go`

2. **❌ No API endpoints** (US-007 in progress)
   - Can't query workflows programmatically
   - Workaround: Read YAML files directly

3. **⚠️ Policy enforcement not active** (US-006 next priority)
   - `allowedProductWorkflows` not enforced yet
   - Risk: Unauthorized workflows can execute
   - Mitigation: Manual PR review

4. **⚠️ Conditional execution limited**
   - `when`/`unless` syntax not fully implemented
   - Workaround: Create separate workflows

### Roadmap

**Week 1 (Current):**
- ✅ US-005: Multi-tier executor active
- 🔄 US-006: Policy enforcement (2 hours)

**Week 2-3:**
- US-007: API endpoints (8 hours)
- US-008: CLI commands (8 hours)

**Month 2:**
- FEAT-001: Custom provisioners (16 hours)
- Enhanced conditionals
- Parallel execution improvements

---

## Getting Help

### For Product Teams

**Documentation:**
- Overview: [README.md](README.md)
- Activation: [activation-guide.md](activation-guide.md) (for platform team)
- Examples: See `workflows/products/ecommerce/` and `workflows/products/analytics/`

**Support:**
- Platform team: #platform-team on Slack
- GitHub issues: Tag with `product-workflows`
- Email: platform-team@yourcompany.com

### For Platform Teams

**Documentation:**
- Gap analysis: [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md)
- Implementation: [US-005_IMPLEMENTATION_SUMMARY.md](../US-005_IMPLEMENTATION_SUMMARY.md)
- Backlog: US-005, US-006, US-007, US-008 in [BACKLOG.md](../../BACKLOG.md)

---

## Next Steps

1. **Review examples:** Study existing workflows in `workflows/products/`
2. **Create workflow:** Follow Quick Start section above
3. **Test locally:** Use `workflow-demo.go`
4. **Submit for approval:** PR with workflow files + admin-config.yaml update
5. **Monitor execution:** Check logs and database after deployment

**Questions?** Contact your platform team or see [README.md](README.md).
