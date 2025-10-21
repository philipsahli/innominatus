# Product Workflows Activation Guide

**Audience:** Platform Teams
**Status:** ✅ Implementation Complete (US-005)
**Last Updated:** 2025-10-19

---

## Overview

This guide explains how to **activate product workflow capabilities** in your innominatus deployment. As of **US-005**, the multi-tier workflow executor is implemented and ready to use.

**What You Get:**
- Product teams can create workflows that run automatically
- Platform workflows run for all deployments (security, compliance)
- Application workflows continue to work as before
- Clear separation of concerns between teams

---

## Prerequisites

Before activation, ensure:

✅ innominatus version with US-005 (commit hash: TBD)
✅ `admin-config.yaml` file exists and is valid
✅ Product workflow files in `workflows/products/` directory
✅ Database connection working (required for workflow persistence)

---

## Activation Steps

### Step 1: Verify Code Version

Check that your innominatus build includes US-005:

```bash
# Check for multi-tier executor function
grep -n "NewServerWithDBAndAdminConfig" internal/server/handlers.go

# Expected: Line ~216
# func NewServerWithDBAndAdminConfig(db *database.Database, adminConfig interface{}) *Server
```

If not found, pull latest code or rebuild:

```bash
git pull origin main
go build -o innominatus cmd/server/main.go
```

### Step 2: Configure admin-config.yaml

Create or update `admin-config.yaml` with workflow policies:

```yaml
# admin-config.yaml
admin:
  defaultCostCenter: "engineering"
  defaultRuntime: "kubernetes"

workflowPolicies:
  # Root directory for workflows
  workflowsRoot: "./workflows"

  # Platform workflows that MUST run for all deployments
  requiredPlatformWorkflows:
    - security-scan
    - cost-monitoring

  # Product workflows that are ALLOWED to run
  # Add product teams here after review
  allowedProductWorkflows:
    - ecommerce/database-setup
    - ecommerce/payment-integration
    - analytics/data-pipeline

  # Override permissions
  workflowOverrides:
    platform: true   # Platform workflows can override product workflows
    product: true    # Product workflows can override application workflows

  # Execution constraints
  maxWorkflowDuration: "30m"
  maxConcurrentWorkflows: 10
  maxStepsPerWorkflow: 50

  # Security policies
  security:
    requireApproval:
      - production
    allowedExecutors:
      - platform-team
      - infrastructure-teams
    secretsAccess:
      vault: "read-only"
      kubernetes: "namespace-scoped"

  # Allowed step types (security control)
  allowedStepTypes:
    - terraform
    - kubernetes
    - ansible
    - database-migration
    - vault-setup
    - monitoring
    - validation
    - security
    - policy
    - tagging
    - cost-analysis
    - resource-provisioning
```

### Step 3: Create Workflow Directory Structure

```bash
# Create directory structure
mkdir -p workflows/platform
mkdir -p workflows/products/ecommerce
mkdir -p workflows/products/analytics

# Verify structure
tree workflows/
# workflows/
# ├── platform/
# │   ├── security-scan.yaml
# │   └── cost-monitoring.yaml
# └── products/
#     ├── ecommerce/
#     │   ├── database-setup.yaml
#     │   └── payment-integration.yaml
#     └── analytics/
#         └── data-pipeline.yaml
```

### Step 4: Add Example Platform Workflows

Create basic platform workflows:

**File:** `workflows/platform/security-scan.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: PlatformWorkflow
metadata:
  name: security-scan
  description: Security scanning for all deployments
  owner: platform-security-team
  phase: pre-deployment
spec:
  triggers:
    - all_deployments
  steps:
    - name: scan-containers
      type: security
      config:
        scanner: trivy
        severity: ["HIGH", "CRITICAL"]
        failOnVulnerabilities: true
```

**File:** `workflows/platform/cost-monitoring.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: PlatformWorkflow
metadata:
  name: cost-monitoring
  description: Cost tracking and tagging
  owner: platform-finops-team
  phase: deployment
spec:
  triggers:
    - all_deployments
  steps:
    - name: tag-resources
      type: tagging
      config:
        tags:
          managed_by: innominatus
          cost_center: "${application.metadata.costCenter}"
          team: "${application.metadata.team}"

    - name: estimate-cost
      type: cost-analysis
      config:
        estimateMonthly: true
        alertThreshold: 1000  # Alert if >$1000/month
```

### Step 5: Start Server

```bash
# Stop existing server if running
pkill innominatus

# Start with multi-tier executor
./innominatus

# Watch for activation message
# Expected output:
# Admin configuration loaded:
#   Workflows Root: ./workflows
#   Required Platform Workflows: [security-scan cost-monitoring]
#   Allowed Product Workflows: [ecommerce/database-setup ecommerce/payment-integration]
# ...
# Database connected successfully
# ✅ Multi-tier workflow executor enabled (platform + product + application workflows)
```

**Success Indicators:**
- ✅ Admin configuration loaded without errors
- ✅ "Multi-tier workflow executor enabled" message appears
- ✅ Server starts without warnings

### Step 6: Verify Activation

Test with a sample deployment:

```bash
# Create test Score spec
cat > test-app.yaml <<EOF
apiVersion: score.dev/v1b1
metadata:
  name: activation-test
  product: ecommerce
  team: platform-team
  costCenter: engineering
containers:
  web:
    image: nginx:latest
resources:
  database:
    type: postgres
EOF

# Deploy
curl -X POST http://localhost:8081/api/specs \
  -H "Content-Type: application/yaml" \
  -H "Authorization: Bearer $API_TOKEN" \
  --data-binary @test-app.yaml
```

**Expected:** Multiple workflow executions:
1. `platform-security-scan` (platform workflow)
2. `platform-cost-monitoring` (platform workflow)
3. `product-ecommerce-database-setup` (product workflow)
4. `product-ecommerce-payment-integration` (product workflow)
5. `app-deployment-activation-test` (application workflow)

### Step 7: Check Workflow Executions

```bash
# Via API
curl http://localhost:8081/api/workflows?app=activation-test | jq

# Via Database
psql -d idp_orchestrator -c "
SELECT id, app_name, workflow_name, status
FROM workflow_executions
WHERE app_name = 'activation-test'
ORDER BY created_at;
"
```

**Expected Results:**

| ID | App Name | Workflow Name | Status |
|----|----------|---------------|--------|
| 1 | activation-test | platform-security-scan | completed |
| 2 | activation-test | platform-cost-monitoring | completed |
| 3 | activation-test | product-ecommerce-database-setup | completed |
| 4 | activation-test | product-ecommerce-payment-integration | completed |
| 5 | activation-test | app-deployment-activation-test | completed |

---

## Onboarding Product Teams

### Process

1. **Product team creates workflow files**
   - Location: `workflows/products/{product-name}/`
   - File format: `{workflow-name}.yaml`
   - Kind: `ProductWorkflow`

2. **Product team submits PR**
   - Add workflow files
   - Update `admin-config.yaml` → `allowedProductWorkflows`
   - Include: Owner, description, testing results

3. **Platform team reviews**
   - Check workflow structure
   - Verify allowed step types
   - Review security implications
   - Test in staging environment

4. **Merge and activate**
   - Merge PR
   - Restart innominatus (picks up new config)
   - Monitor first few executions

### Review Checklist

When reviewing product workflow PRs:

- [ ] Workflow file location: `workflows/products/{product}/`
- [ ] Valid YAML structure
- [ ] Kind: `ProductWorkflow`
- [ ] Owner specified
- [ ] Phase specified (pre-deployment, deployment, post-deployment)
- [ ] Only allowed step types used
- [ ] No hardcoded secrets (use Vault references)
- [ ] Reasonable timeout/duration
- [ ] Tested in staging
- [ ] Added to `allowedProductWorkflows` in admin-config.yaml

---

## Monitoring

### Key Metrics

Monitor these after activation:

```promql
# Total workflow executions by tier
sum by (tier) (innominatus_workflows_total)

# Product workflow success rate
sum by (product) (innominatus_product_workflows_success_total) /
sum by (product) (innominatus_product_workflows_total)

# Platform workflow duration (should be fast)
histogram_quantile(0.95, innominatus_platform_workflow_duration_seconds)
```

### Logs to Watch

```bash
# Multi-tier executor logs
tail -f innominatus.log | grep "Multi-tier"

# Product workflow loading
tail -f innominatus.log | grep "product-.*workflow"

# Policy validation (after US-006)
tail -f innominatus.log | grep "policy"
```

### Alerts to Set Up

```yaml
# Prometheus alerts
groups:
  - name: product_workflows
    rules:
      - alert: ProductWorkflowFailureRate
        expr: sum by (product) (rate(innominatus_product_workflows_failed_total[5m])) > 0.1
        annotations:
          summary: "Product {{ $labels.product }} workflow failure rate >10%"

      - alert: PlatformWorkflowBlocked
        expr: innominatus_platform_workflows_blocked_total > 0
        annotations:
          summary: "Platform workflow blocked - may impact all deployments"
```

---

## Troubleshooting

### Issue: "Single-tier workflow executor" Message

**Symptom:**
```
ℹ️  Single-tier workflow executor (use admin-config.yaml for product workflows)
```

**Cause:** admin-config.yaml failed to load

**Fix:**
1. Check file exists: `ls -la admin-config.yaml`
2. Check YAML syntax: `yamllint admin-config.yaml`
3. Check server logs for error: `grep "admin config" innominatus.log`

### Issue: Product Workflows Not Executing

**Symptom:** Only platform and application workflows execute

**Debug Steps:**

1. **Check product metadata in Score spec:**
   ```yaml
   metadata:
     product: ecommerce  # Must match product directory name
   ```

2. **Check workflow files exist:**
   ```bash
   ls -la workflows/products/ecommerce/
   ```

3. **Check allowed list:**
   ```bash
   grep "allowedProductWorkflows" admin-config.yaml
   # Must include: ecommerce/workflow-name
   ```

4. **Check server logs:**
   ```bash
   tail -f innominatus.log | grep "product-ecommerce"
   ```

### Issue: All Workflows Failing

**Symptom:** All multi-tier workflows fail immediately

**Possible Causes:**

1. **Database connection lost:**
   ```bash
   psql -d idp_orchestrator -c "SELECT 1;"
   ```

2. **Workflow files have syntax errors:**
   ```bash
   for f in workflows/**/*.yaml; do
       echo "Validating $f"
       yaml-validator $f
   done
   ```

3. **Step type not allowed:**
   - Check `allowedStepTypes` in admin-config.yaml
   - Verify workflows only use allowed types

---

## Rollback Procedures

### Quick Rollback (No Code Change)

**Disable multi-tier executor:**

```bash
# Rename admin config
mv admin-config.yaml admin-config.yaml.disabled

# Restart server
pkill innominatus
./innominatus

# Verify fallback
# Expected: "Single-tier workflow executor" message
```

**Result:** Only application workflows execute

### Full Rollback (Code Change)

If issues persist:

```bash
# Revert main.go change
git diff cmd/server/main.go

# Line 116 should be:
srv = server.NewServerWithDB(db)

# Rebuild
go build -o innominatus cmd/server/main.go

# Restart
./innominatus
```

---

## Security Considerations

### Current State (Before US-006)

⚠️ **Policy enforcement NOT active**

**Risk:** Any workflow in `workflows/products/` will execute

**Mitigation:**
1. Restrict write access to `workflows/` directory
2. Require PR reviews for all workflow changes
3. Monitor workflow executions closely
4. Implement US-006 immediately (see below)

### After US-006 Implementation

✅ **Policy enforcement active**

**Benefit:** Only workflows in `allowedProductWorkflows` execute

**Timeline:** US-006 estimated 2 hours (next priority)

---

## Performance Tuning

### Workflow Concurrency

Adjust based on infrastructure capacity:

```yaml
# admin-config.yaml
workflowPolicies:
  maxConcurrentWorkflows: 10  # Increase for high-volume deployments
```

**Guidance:**
- Start with 10
- Monitor CPU/memory during peak deployments
- Increase to 20-30 for production clusters with >100 deployments/day

### Workflow Timeout

```yaml
workflowPolicies:
  maxWorkflowDuration: "30m"  # Per workflow
```

**Guidance:**
- Platform workflows: <5 minutes
- Product workflows: <15 minutes
- Application workflows: <30 minutes

---

## Next Steps

After successful activation:

1. **✅ US-005 Complete:** Multi-tier executor active
2. **➡️ US-006 (Next):** Enable policy enforcement (2 hours)
3. **➡️ US-007 (Week 2):** Add API endpoints for workflow discovery
4. **➡️ US-008 (Week 2):** Add CLI commands for product teams

**See:** [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md) for full roadmap

---

## Getting Help

**Platform Team Resources:**
- **Gap Analysis:** [PRODUCT_WORKFLOW_GAPS.md](../PRODUCT_WORKFLOW_GAPS.md)
- **Implementation Summary:** [US-005_IMPLEMENTATION_SUMMARY.md](../US-005_IMPLEMENTATION_SUMMARY.md)
- **Backlog:** US-005, US-006 in [BACKLOG.md](../../BACKLOG.md)

**Product Team Resources:**
- **User Guide:** [README.md](README.md)
- **Workflow Development:** [product-workflows.md](product-workflows.md) (coming soon)

**Support:**
- GitHub Issues: Tag with `product-workflows`
- Internal Slack: #platform-team
- Email: platform-team@yourcompany.com

---

## Success Metrics

Track these to measure activation success:

**Week 1:**
- [ ] Multi-tier executor activated in production
- [ ] 3+ platform workflows running
- [ ] 2+ product teams onboarded
- [ ] Zero unauthorized workflow executions

**Week 2:**
- [ ] US-006 policy enforcement active
- [ ] 10+ product workflows deployed
- [ ] <5% workflow failure rate
- [ ] Product teams self-servicing (after US-007/US-008)

**Month 1:**
- [ ] 5+ product teams active
- [ ] 50+ product workflows
- [ ] Platform workflows covering 100% of deployments
- [ ] Product team satisfaction >90%

---

**Questions?** Contact platform team or open a GitHub issue.

**Ready to activate?** Follow Step 1 above! 🚀
