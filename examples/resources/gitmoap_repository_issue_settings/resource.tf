data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# A repository's settings only take effect over its group's with override_group_settings.
resource "gitmoap_repository_issue_settings" "api" {
  repo_id                 = data.gitmoap_repository_by_path.api.repository_id
  override_group_settings = true
  default_issue_type      = "bug"
  public_issue_filing     = true
}
