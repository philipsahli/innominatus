# Minio S3 Bucket Terraform Module

This Terraform module provisions S3 buckets on Minio using the `aminueza/minio` provider.

## Usage

### Via Workflow (Recommended)

Use this module in a workflow YAML file:

```yaml
steps:
  - name: provision-object-storage
    type: terraform
    config:
      operation: apply
      working_dir: ./terraform/minio-bucket
      variables:
        bucket_name: my-app-storage
        minio_endpoint: http://minio.minio-system.svc.cluster.local:9000
        minio_user: minioadmin
        minio_password: minioadmin
      outputs:
        - minio_url
        - bucket_name
        - endpoint
```

### Direct Terraform

```bash
cd terraform/minio-bucket

terraform init

terraform apply \
  -var="bucket_name=my-app-storage" \
  -var="minio_endpoint=http://minio.minio-system.svc.cluster.local:9000" \
  -var="minio_user=minioadmin" \
  -var="minio_password=minioadmin"
```

## Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|----------|
| bucket_name | Name of the S3 bucket to create | string | n/a | yes |
| minio_endpoint | Minio server endpoint | string | `http://minio.minio-system.svc.cluster.local:9000` | no |
| minio_user | Minio admin user | string | `minioadmin` | no |
| minio_password | Minio admin password | string | `minioadmin` | no |

## Outputs

| Name | Description |
|------|-------------|
| minio_url | S3 URL for the created bucket (e.g., `s3://my-app-storage`) |
| bucket_name | Name of the created bucket |
| endpoint | Minio endpoint URL |
| bucket_arn | ARN-style identifier for the bucket |

## Requirements

- Terraform >= 1.0
- Minio server accessible at the specified endpoint
- Valid Minio credentials

## Provider

This module uses the `aminueza/minio` Terraform provider version ~> 2.0.

## Notes

- Buckets are created with `private` ACL by default
- SSL is disabled for demo environment compatibility
- For production use, enable SSL and use proper secret management for credentials
