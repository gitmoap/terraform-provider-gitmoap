data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_catalog_environment" "staging" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "staging"
  tier     = "staging"
}

data "gitmoap_catalog_environment_deployments" "example" {
  environment_id = gitmoap_group_catalog_environment.staging.environment_id
}
