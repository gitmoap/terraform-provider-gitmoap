# A repository's settings only apply over the group's with override_group_settings.
resource "gitmoap_repository_mr_settings" "api" {
  repo_id                 = gitmoap_repository.api.repository_id
  override_group_settings = true
  squash_option           = "require"
}
