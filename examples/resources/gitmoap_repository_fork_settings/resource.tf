data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

# This repository can't be forked, whatever its group allows.
resource "gitmoap_repository_fork_settings" "api" {
  repo_id                 = data.gitmoap_repository_by_path.api.repository_id
  override_group_settings = true
  forking                 = "disabled"
}
