terraform {
  required_providers {
    rt = {
      source  = "registry-tools/rt"
      version = "1.0.0"
    }
  }
}

variable "github_token" {
  type = string
  sensitive = true
}

provider "rt" {
}

# Your registry namespace. This becomes part of the module path
resource "rt_namespace" "this" {
  name        = "platform"
  description = "Modules provided by the platform team"
}

# A token that can be used to provision modules
# from the namespace
resource "rt_terraform_token" "provisioner" {
  namespace_id = rt_namespace.this.id
  role         = "provisioner"
  expires_in   = "never"
}

# A connection to a VCS provider that can be used to automate
# module publishing
resource "rt_vcs_connector" "github" {
  github = {
    token = var.github_token
  }
}

resource "rt_tag_publisher" "tag_publishers" {
  vcs_connector_id = rt_vcs_connector.github.id
  namespace_id     = rt_namespace.this.id
  repo_identifier  = "registry-tools/terraform-aws-example-module"
}
