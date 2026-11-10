data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# A repository's settings only apply over the group's with override_group_settings.
resource "gitmoap_repository_mr_settings" "api" {
  repo_id                 = data.gitmoap_repository_by_path.api.repository_id
  override_group_settings = true
  squash_option           = "require"
}
