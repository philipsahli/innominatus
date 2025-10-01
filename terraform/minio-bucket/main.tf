# Minio S3 Bucket Provisioning Module
# This module provisions S3 buckets on Minio using the aminueza/minio provider

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
  default     = "http://minio.minio-system.svc.cluster.local:9000"
}

variable "minio_user" {
  description = "Minio admin user"
  type        = string
  default     = "minioadmin"
}

variable "minio_password" {
  description = "Minio admin password"
  type        = string
  sensitive   = true
  default     = "minioadmin"
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

output "bucket_arn" {
  value       = "arn:aws:s3:::${minio_s3_bucket.bucket.bucket}"
  description = "ARN-style identifier for the bucket"
}
