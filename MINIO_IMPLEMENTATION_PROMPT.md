# Implementation Task: Add Minio S3 Object Storage to Demo Environment

## Objective

Enhance the innominatus demo environment (`demo-time` command) to include Minio as an S3-compatible object storage service, and implement full Terraform-based resource provisioning support for creating S3 buckets using the `aminueza/minio` Terraform provider.

## Background

Developers frequently need object storage for applications. The demo environment should provide a production-like S3 service that can be provisioned through Score specifications and managed via Terraform workflows.

## Requirements

### 1. Demo Environment Enhancement

**Add Minio to `demo-time` command:**
- Install Minio using Helm chart in the demo Kubernetes cluster
- Configure Minio with:
  - Namespace: `minio-system`
  - Service exposed via Ingress: `http://minio.localtest.me`
  - Console UI via Ingress: `http://minio-console.localtest.me`
  - Default credentials: `minioadmin` / `minioadmin` (demo only)
  - Persistent storage with PVC
- Add Minio to health checks in `demo-status` command
- Include Minio cleanup in `demo-nuke` command (namespace deletion, PVC removal)
- Display Minio credentials and URLs in post-installation quick start guide

### 2. Terraform Step Type Implementation

**Implement the `terraform` step type in `internal/workflow/executor.go`:**

Currently, the executor only has 9 simulated step types. The `terraform` step type is referenced in workflows but **not implemented**. You must:

- Add `terraform` step executor to `registerDefaultStepExecutors()` function
- Support Terraform operations: `init`, `plan`, `apply`, `destroy`, `output`
- Execute Terraform commands in isolated workspaces per application/environment
- Capture Terraform outputs and make them available to subsequent workflow steps
- Handle Terraform state management (local state files in workspace directories)
- Provide clear error messages for Terraform failures

**Terraform Step Configuration:**
```yaml
- name: provision-s3-bucket
  type: terraform
  config:
    operation: apply  # init, plan, apply, destroy, output
    working_dir: ./terraform/minio-bucket
    variables:
      bucket_name: ${app.name}-data
      minio_endpoint: http://minio.minio-system.svc.cluster.local:9000
      minio_user: minioadmin
      minio_password: minioadmin
    outputs:
      - minio_url  # Captured from terraform output
```

### 3. Resource Type: minio-s3-bucket

**Add new Score resource type support:**

When a Score spec includes:
```yaml
resources:
  data-store:
    type: minio-s3-bucket
    properties:
      bucket_name: ${metadata.name}-storage
```

The system should:
- Recognize `minio-s3-bucket` as a valid resource type
- Generate appropriate Terraform configuration using `aminueza/minio` provider
- Provision the bucket via Terraform workflow step
- Capture bucket URL from Terraform output: `minio_url`
- Make bucket URL available for container environment variables

### 4. Terraform Provider Configuration

**Create Terraform module for Minio bucket provisioning:**

Location: `terraform/minio-bucket/main.tf`

```hcl
terraform {
  required_providers {
    minio = {
      source  = "aminueza/minio"
      version = "~> 2.0"
    }
  }
}

provider "minio" {
  minio_server   = var.minio_endpoint
  minio_user     = var.minio_user
  minio_password = var.minio_password
  minio_ssl      = false
}

variable "bucket_name" {
  description = "Name of the S3 bucket to create"
  type        = string
}

variable "minio_endpoint" {
  description = "Minio server endpoint"
  type        = string
}

variable "minio_user" {
  description = "Minio admin user"
  type        = string
}

variable "minio_password" {
  description = "Minio admin password"
  type        = string
  sensitive   = true
}

resource "minio_s3_bucket" "bucket" {
  bucket = var.bucket_name
  acl    = "private"
}

output "minio_url" {
  value       = "s3://${minio_s3_bucket.bucket.bucket}"
  description = "S3 URL for the created bucket"
}

output "bucket_name" {
  value       = minio_s3_bucket.bucket.bucket
  description = "Name of the created bucket"
}

output "endpoint" {
  value       = var.minio_endpoint
  description = "Minio endpoint URL"
}
```

### 5. Golden Path Integration

**Update or create golden path workflow to demonstrate S3 provisioning:**

Example workflow step in `workflows/deploy-app.yaml`:
```yaml
- name: provision-object-storage
  type: terraform
  config:
    operation: apply
    working_dir: ./terraform/minio-bucket
    variables:
      bucket_name: ${app.name}-storage
      minio_endpoint: http://minio.minio-system.svc.cluster.local:9000
      minio_user: minioadmin
      minio_password: minioadmin
    outputs:
      - minio_url
      - bucket_name
      - endpoint
```

### 6. Documentation Updates

**Update documentation to reflect new capabilities:**

- `CLAUDE.md`: Add Minio to demo environment component list with credentials
- `docs/DEMO_ENVIRONMENT.md`: Document Minio installation, configuration, and usage
- `docs/TERRAFORM_INTEGRATION.md`: Document Terraform step type and provider usage
- `docs/RESOURCE_TYPES.md`: Document `minio-s3-bucket` resource type and properties

### 7. Testing Requirements

**Add tests for new functionality:**

- `internal/demo/minio_test.go`: Test Minio installation and health checks
- `internal/workflow/terraform_test.go`: Test Terraform step executor
- Integration test: Deploy Score spec with `minio-s3-bucket` resource
- Verify Terraform outputs are captured and available to subsequent steps

## Implementation Steps

1. **Add Minio Helm chart installation to demo environment**
   - Update `internal/demo/install.go` to include Minio
   - Configure Ingress rules for Minio service and console
   - Add health check for Minio in `demo-status`

2. **Implement Terraform step type executor**
   - Add to `internal/workflow/executor.go`
   - Support init, plan, apply, destroy, output operations
   - Implement workspace isolation and state management
   - Capture outputs and make available to workflow context

3. **Create Terraform module for Minio bucket provisioning**
   - Create `terraform/minio-bucket/` directory
   - Write `main.tf` with `aminueza/minio` provider configuration
   - Define variables and outputs

4. **Add minio-s3-bucket resource type support**
   - Update Score resource type validation to recognize `minio-s3-bucket`
   - Generate Terraform configuration from Score resource definition
   - Map resource properties to Terraform variables

5. **Update golden path workflows**
   - Add object storage provisioning to `deploy-app` workflow
   - Create example Score spec with `minio-s3-bucket` resource

6. **Update documentation**
   - Document Minio in demo environment guides
   - Document Terraform step type usage
   - Add examples and tutorials

7. **Add tests and verify integration**
   - Write unit tests for Terraform executor
   - Create integration test for end-to-end provisioning
   - Test demo environment installation and cleanup

## Success Criteria

- [ ] `./innominatus-ctl demo-time` successfully installs Minio alongside other services
- [ ] Minio accessible at `http://minio.localtest.me` and console at `http://minio-console.localtest.me`
- [ ] `./innominatus-ctl demo-status` shows Minio health status
- [ ] `./innominatus-ctl demo-nuke` cleanly removes Minio resources
- [ ] Terraform step type executor implemented and functional
- [ ] Score spec with `minio-s3-bucket` resource successfully provisions bucket
- [ ] Terraform outputs captured and available in workflow context
- [ ] Bucket URL accessible via `${resources.data-store.minio_url}` syntax
- [ ] Documentation updated with Minio and Terraform usage
- [ ] Tests pass with 90%+ coverage for new code

## Example Usage

**Score Spec with Minio S3 Bucket:**
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app

containers:
  main:
    image: myapp:latest
    variables:
      S3_BUCKET: ${resources.data-store.bucket_name}
      S3_ENDPOINT: ${resources.data-store.endpoint}
      S3_URL: ${resources.data-store.minio_url}

resources:
  data-store:
    type: minio-s3-bucket
    properties:
      bucket_name: ${metadata.name}-storage
```

**Deploy Command:**
```bash
./innominatus-ctl run deploy-app score-spec.yaml
```

**Expected Behavior:**
1. Terraform provisions Minio bucket named `my-app-storage`
2. Terraform outputs captured: `minio_url`, `bucket_name`, `endpoint`
3. Container environment variables populated with bucket details
4. Application deployed with access to S3 storage

## Notes

- Minio credentials are hardcoded for demo purposes only
- Production deployments should use proper secret management
- Terraform state files stored locally in workspace directories
- This implementation addresses Gap 1.1 from `GAP_ANALYSIS-GOLDEN_PATHS-2025-10-01.md`
- The `terraform` step type is one of 7 critical missing step types identified in the gap analysis

## Related Files

- `internal/workflow/executor.go` - Add Terraform step executor here (line 643-750)
- `internal/demo/install.go` - Add Minio installation
- `internal/demo/status.go` - Add Minio health checks
- `workflows/deploy-app.yaml` - Add Terraform provisioning step
- `terraform/minio-bucket/main.tf` - Create Terraform module
- `GAP_ANALYSIS-GOLDEN_PATHS-2025-10-01.md` - Reference document for missing implementations
