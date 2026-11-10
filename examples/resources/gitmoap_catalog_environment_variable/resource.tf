data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_catalog_environment" "staging" {
  group_id = data.gitmoap_group_by_path.platform.group_id
  slug     = "staging"
  tier     = "staging"
}

resource "gitmoap_catalog_environment_variable" "api_url" {
  environment_id = gitmoap_group_catalog_environment.staging.environment_id
  name           = "API_URL"
  value          = "https://staging.api.example.com"
}
