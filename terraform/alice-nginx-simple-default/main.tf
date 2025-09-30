# Infrastructure terraform configuration
terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "local_file" "infra_config" {
  content  = "Infrastructure provisioned successfully!"
  filename = "${path.module}/infra_provisioned.txt"
}

output "infra_status" {
  value = "Infrastructure ready for Kubernetes deployment"
}

output "cluster_endpoint" {
  value = "https://k8s-cluster.example.com"
}