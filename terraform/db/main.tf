# Simple demo Terraform configuration
terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "local_file" "db_config" {
  content  = "Database provisioned successfully!"
  filename = "${path.module}/db_provisioned.txt"
}

output "database_status" {
  value = "Database infrastructure provisioned"
}

output "connection_string" {
  value = "postgresql://dbuser:password@localhost:5432/myapp_db"
}