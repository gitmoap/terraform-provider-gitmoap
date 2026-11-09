resource "gitmoap_group_pipeline_variable" "region" {
  group_id = gitmoap_group.platform.group_id
  name     = "DEPLOY_REGION"
  value    = "eu-west-1"
}

# A secret value is write-only: the API never returns it.
resource "gitmoap_group_pipeline_variable" "registry_token" {
  group_id = gitmoap_group.platform.group_id
  name     = "REGISTRY_TOKEN"
  value    = var.registry_token
  secret   = true
}
