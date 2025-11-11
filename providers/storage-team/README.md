# Storage Team Provider

This provider offers S3-compatible object storage provisioners and golden paths for the platform.

## Provisioners

### s3-bucket
Standard S3-compatible bucket (Minio-backed) with encryption and versioning support.

### s3-bucket-with-lifecycle
S3 bucket with automatic lifecycle policies:
- Transition to archive after 90 days
- Delete after 365 days
- Supports custom retention rules

### object-storage-access
IAM access keys for bucket access with least-privilege permissions.

## Golden Paths

### create-storage-bucket
Creates a new S3 bucket with:
- Encryption at rest
- Versioning enabled
- Access logging
- CORS configuration
- Bucket policies

### setup-backup-storage
Automated backup storage setup:
- Dedicated backup bucket
- 30-day retention policy
- Cross-region replication (optional)
- Automated snapshots

### migrate-storage
Data migration between buckets:
- Parallel transfers for performance
- Integrity verification
- Progress tracking
- Rollback capability

## Usage Example

```yaml
# score-spec.yaml
resources:
  app-storage:
    type: s3
    properties:
      bucket_name: my-app-data
      versioning: true
      lifecycle_days: 90
```

## Complete Examples

See these Score specifications that use S3 storage:

- **`examples/score-ecommerce-backend-v2.yaml`** - Adding S3 to existing app
- **`examples/score-order-service-v2.yaml`** - Order service with S3 for receipt PDFs
- **`examples/score-spec-with-product-metadata.yaml`** - Full app with multiple resources including storage

### Incremental Deployment Pattern

```bash
# Step 1: Deploy v1 with database only
./innominatus-ctl deploy examples/score-ecommerce-backend-v1.yaml -w

# Step 2: Add S3 storage later (v2)
./innominatus-ctl deploy examples/score-ecommerce-backend-v2.yaml -w

# Output:
# ℹ️  Detected existing: db (postgres) - Skipping
# 🆕 Detected new: storage (s3) - Provisioning
```

## Git Repository Configuration

To use this provider from Git (recommended for production):

```yaml
# admin-config.yaml
providers:
  - name: storage-team
    type: git
    category: infrastructure
    repository: https://gitea.localtest.me/platform-team/storage-provider
    ref: v1.2.0  # Pin to stable version
    enabled: true
```

For development, you can track the latest changes:

```yaml
providers:
  - name: storage-team-dev
    type: git
    repository: https://gitea.localtest.me/platform-team/storage-provider
    ref: main  # Track main branch
    enabled: false  # Enable for testing only
```

## Maintenance

- **Repository**: https://gitea.localtest.me/platform-team/storage-provider
- **Owner**: Platform Storage Team
- **Support**: storage-team@example.com
- **Versioning**: Semantic versioning (v1.2.3)
