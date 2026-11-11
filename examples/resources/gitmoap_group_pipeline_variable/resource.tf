variable "registry_token" {
  type      = string
  sensitive = true
  ephemeral = true
}

data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_pipeline_variable" "region" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  name     = "DEPLOY_REGION"
  value    = "eu-west-1"
}

# A secret value is never returned by the API, and value_wo keeps it out of Terraform's state too.
resource "gitmoap_group_pipeline_variable" "registry_token" {
  group_id         = data.gitmoap_group_by_path.platform.group_id
  name             = "REGISTRY_TOKEN"
  value_wo         = var.registry_token
  value_wo_version = 1
  secret           = true
}
