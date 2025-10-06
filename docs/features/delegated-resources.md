# Delegated Resources

> **Track externally-managed infrastructure within your unified resource inventory**

## Overview

Delegated Resources represent infrastructure managed by external systems (GitOps workflows, Terraform Enterprise, manual processes) while maintaining visibility and lifecycle tracking within the innominatus orchestrator.

**Key Benefits:**
- **Unified Inventory**: Developers see all resources in one place, regardless of management method
- **Clear Ownership**: Explicit tracking of which system manages each resource
- **External Integration**: Seamless GitOps and platform integration workflows
- **State Transparency**: Track external provisioning progress and health

---

## Resource Types

innominatus supports three resource types:

| Type | Managed By | Reconciliation | Use Case |
|------|------------|----------------|----------|
| **native** | innominatus Orchestrator | Active | Resources provisioned directly via Terraform/Ansible/K8s steps |
| **delegated** | External System (GitOps, Terraform Enterprise) | Passive | Resources requiring GitOps PR workflows or external approval |
| **external** | External (read-only) | None | Imported references to pre-existing infrastructure |

### Type Comparison

**Native Resources:**
```yaml
resources:
  database:
    type: postgres
    # innominatus provisions directly via Terraform
```

**Delegated Resources:**
```yaml
resources:
  vpc:
    type: delegated
    provider: gitops
    reference: https://github.com/platform/vpc-configs/pull/123
    # Managed by external GitOps workflow
```

**External Resources:**
```yaml
resources:
  shared-vpc:
    type: external
    reference: vpc-prod-us-east-1
    # Read-only reference, managed elsewhere
```

---

## Lifecycle & States

### Resource Lifecycle States

All resources (native, delegated, external) share core lifecycle states:

```
requested → provisioning → active → terminating → terminated
              ↓                ↓
            failed         degraded
```

### External States (Delegated Resources Only)

Delegated resources have an additional `external_state` field tracking the external system's progress:

| External State | Description | Typical Duration |
|----------------|-------------|------------------|
| **WaitingExternal** | Waiting for external system to start | Immediate |
| **BuildingExternal** | External system is provisioning | 5-30 minutes |
| **Healthy** | External resource is provisioned and healthy | Steady state |
| **Error** | External provisioning failed | Requires intervention |
| **Unknown** | External state cannot be determined | Investigation needed |

**State Transition Diagram:**
```
WaitingExternal → BuildingExternal → Healthy
                        ↓
                      Error
```

---

## Configuration Examples

### Score Specification with Delegated Resource

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: payment-api

resources:
  # Native resource - managed by innominatus
  database:
    type: postgres
    properties:
      version: "15"
      storage: "100Gi"

  # Delegated resource - managed by GitOps
  vpc:
    type: delegated
    provider: gitops
    metadata:
      repo: platform/network-configs
      branch: add-payment-vpc
      pr: "https://github.com/platform/network-configs/pull/456"

containers:
  api:
    image: payment-api:v1.2.0
    env:
      DATABASE_URL: ${resources.database.connection_string}
      VPC_ID: ${resources.vpc.id}
```

### Workflow Example: GitOps Provisioning

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: gitops-vpc-provision
  description: Request VPC provisioning via GitOps workflow

variables:
  VPC_NAME: payment-vpc
  REGION: us-east-1

spec:
  steps:
    - name: create-vpc-pr
      type: gitops-provision
      config:
        repo: platform/network-configs
        base_branch: main
        pr_title: "Add VPC for ${workflow.VPC_NAME}"
        pr_body: |
          Requested by: ${user}
          Application: ${app}
          Region: ${workflow.REGION}

        template_dir: ./templates/vpc
        template_vars:
          vpc_name: ${workflow.VPC_NAME}
          region: ${workflow.REGION}
          cidr_block: "10.100.0.0/16"

        ci_check: terraform/plan
        wait_for_merge: true
        timeout_minutes: 60

    - name: verify-vpc
      type: validation
      when: on_success
      env:
        VPC_ID: ${resources.vpc.id}
        REGION: ${workflow.REGION}
```

---

## API Reference

### List Resources with Filtering

**Endpoint:** `GET /api/resources`

**Query Parameters:**
- `app` - Filter by application name
- `type` - Filter by resource type (`native`, `delegated`, `external`)
- `provider` - Filter by provider (`gitops`, `terraform-enterprise`, etc.)

**Examples:**

```bash
# Get all delegated resources
curl http://localhost:8081/api/resources?type=delegated

# Get GitOps resources for payment-api
curl http://localhost:8081/api/resources?app=payment-api&provider=gitops

# Get all native resources
curl http://localhost:8081/api/resources?type=native
```

**Response Format:**
```json
{
  "application": "payment-api",
  "type": "delegated",
  "provider": "gitops",
  "resources": [
    {
      "id": 42,
      "application_name": "payment-api",
      "resource_name": "vpc",
      "resource_type": "vpc",
      "state": "active",
      "health_status": "healthy",
      "type": "delegated",
      "provider": "gitops",
      "reference_url": "https://github.com/platform/network-configs/pull/456",
      "external_state": "Healthy",
      "last_sync": "2025-10-06T14:30:00Z",
      "created_at": "2025-10-06T12:00:00Z",
      "updated_at": "2025-10-06T14:30:00Z"
    }
  ]
}
```

### Update External Resource State

**Endpoint:** `POST /api/resources/{id}/external-state`

**Request Body:**
```json
{
  "external_state": "Healthy",
  "reference_url": "https://github.com/platform/network-configs/pull/456"
}
```

**Valid External States:**
- `WaitingExternal`
- `BuildingExternal`
- `Healthy`
- `Error`
- `Unknown`

---

## Monitoring & Metrics

### Prometheus Metrics

innominatus exposes delegated resource metrics at `http://localhost:8081/metrics`:

```promql
# Count resources by type
innominatus_resources_total{type="native"}
innominatus_resources_total{type="delegated"}
innominatus_resources_total{type="external"}

# External resource health
innominatus_resources_external_healthy_total
innominatus_resources_external_failed_total

# GitOps operation performance
innominatus_gitops_wait_duration_seconds
```

### Grafana Dashboard Examples

**Resource Type Distribution:**
```promql
sum by (type) (innominatus_resources_total)
```

**External Resource Health:**
```promql
rate(innominatus_resources_external_healthy_total[5m])
rate(innominatus_resources_external_failed_total[5m])
```

**GitOps Wait Time (p95):**
```promql
histogram_quantile(0.95, innominatus_gitops_wait_duration_seconds)
```

---

## Developer Experience

### Unified Resource View

Developers interact with all resources uniformly, regardless of management type:

```bash
# List all resources for my application
./innominatus-ctl list-resources --app my-app

# Output shows both native and delegated resources
RESOURCE NAME    TYPE        MANAGED BY           STATE     HEALTH
database         postgres    Orchestrator         active    healthy
cache            redis       Orchestrator         active    healthy
vpc              vpc         GitOps (PR #456)     active    healthy
cdn              cloudfront  Terraform Enterprise active    healthy
```

### Automatic Tracking

When a workflow creates a delegated resource:
1. innominatus creates resource record with `type=delegated`
2. Sets `external_state=WaitingExternal`
3. Stores reference URL (PR link, build URL, etc.)
4. Polls/waits for external system completion
5. Updates `external_state=Healthy` when ready

**Developer sees:**
- Resource appears in inventory immediately (with "Waiting" status)
- Progress updates as external system provisions
- Final state when provisioning completes
- Reference link to external PR/build for troubleshooting

---

## Use Cases

### 1. VPC Provisioning via GitOps

**Scenario:** Network team manages VPCs through GitOps with manual approval

**Implementation:**
```yaml
steps:
  - name: request-vpc
    type: gitops-provision
    resource: vpc
    config:
      repo: platform/network-configs
      pr_title: "Add VPC for ${app}"
      wait_for_merge: true
```

**Developer sees:**
- Resource shows `type: delegated`, `provider: gitops`
- Reference URL links to GitHub PR
- State transitions: WaitingExternal → BuildingExternal → Healthy
- Can proceed with deployment once VPC is ready

### 2. Terraform Enterprise Integration

**Scenario:** Enterprise Terraform workspaces for compliance-sensitive resources

**Implementation:**
```yaml
steps:
  - name: provision-database
    type: terraform-enterprise
    resource: database
    config:
      workspace: prod-databases
      variables:
        db_name: ${app}-db
        backup_retention: 30
```

**Platform team sees:**
- Centralized tracking of all TFE-managed databases
- Clear ownership and approval workflows
- Integration with existing TFE governance

### 3. Manual Resource References

**Scenario:** Import existing shared infrastructure

**Implementation:**
```yaml
resources:
  shared-s3:
    type: external
    reference: arn:aws:s3:::company-shared-assets
    provider: manual
```

**Developer sees:**
- Read-only reference in resource inventory
- Clear indication resource is managed elsewhere
- Can use resource outputs in workflows

---

## Best Practices

### 1. Choose the Right Resource Type

**Use Native when:**
- Resource is application-specific
- No external approval needed
- Standard terraform/ansible provisioning works

**Use Delegated when:**
- Resource requires GitOps PR workflow
- External approval or review needed
- Provisioned by Terraform Enterprise or similar
- Cross-team coordination required

**Use External when:**
- Resource exists outside orchestrator
- Read-only reference needed
- Managed by separate platform team

### 2. Provide Clear Reference URLs

Always include actionable reference URLs:

```yaml
# Good - Developer can investigate
reference_url: https://github.com/platform/infra/pull/456

# Bad - Not actionable
reference_url: "GitOps workflow in progress"
```

### 3. Set Appropriate Timeouts

GitOps workflows can take time:

```yaml
config:
  wait_for_merge: true
  timeout_minutes: 60  # Allow time for review + CI
```

### 4. Monitor External State Transitions

Alert on resources stuck in `WaitingExternal` or `BuildingExternal` for too long:

```promql
# Alert if external provisioning takes > 30 minutes
innominatus_gitops_wait_duration_seconds > 1800
```

### 5. Document Ownership

Use provider metadata to clarify ownership:

```yaml
provider: gitops
metadata:
  team: platform-networking
  slack: #platform-networking
  docs: https://wiki.company.com/platform/networking
```

---

## Migration Guide

### Existing Resources

All existing resources automatically become `type=native` due to database default value. No action needed.

### Adding Delegated Resources

1. Update Score spec with resource type:
```yaml
resources:
  my-resource:
    type: delegated
    provider: gitops
```

2. Create workflow step:
```yaml
steps:
  - name: provision-resource
    type: gitops-provision
    resource: my-resource
```

3. Monitor via API:
```bash
curl http://localhost:8081/api/resources?type=delegated
```

---

## Troubleshooting

### Resource Stuck in WaitingExternal

**Symptoms:**
- External state shows `WaitingExternal` for extended period
- No reference URL or URL is invalid

**Solutions:**
1. Check workflow logs for PR creation errors
2. Verify external system credentials
3. Check reference URL manually
4. Update external state manually if PR was created externally:
```bash
curl -X POST http://localhost:8081/api/resources/42/external-state \
  -d '{"external_state":"BuildingExternal","reference_url":"https://..."}'
```

### External State Not Updating

**Symptoms:**
- Resource stays in `BuildingExternal` despite external completion

**Solutions:**
1. Check webhook configuration if using event-driven updates
2. Verify polling interval in workflow configuration
3. Manually update state:
```bash
curl -X POST http://localhost:8081/api/resources/42/external-state \
  -d '{"external_state":"Healthy"}'
```

### Missing Reference URL

**Symptoms:**
- `reference_url` is null or empty

**Solutions:**
1. Update workflow step to capture PR URL
2. Add explicit reference URL in resource definition
3. Populate via API:
```bash
curl -X POST http://localhost:8081/api/resources/42/external-state \
  -d '{"reference_url":"https://github.com/org/repo/pull/123"}'
```

---

## See Also

- [Workflow Developer Guide](../guides/workflow-developer-guide.md) - Workflow step types
- [API Reference](../api/resources.md) - Complete REST API documentation
- [Golden Paths](../guides/golden-paths.md) - Pre-defined workflow templates
- [Monitoring Guide](../platform-team-guide/monitoring.md) - Metrics and dashboards
