# Web application terraform configuration
terraform {
  required_providers {
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "local_file" "web_config" {
  content  = "Web application infrastructure provisioned successfully!"
  filename = "${path.module}/web_provisioned.txt"
}

output "web_status" {
  value = "Web infrastructure provisioned"
}

output "load_balancer_ip" {
  value = "192.168.1.100"
}
