data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_tag_rule" "releases" {
  repo_id           = data.gitmoap_repository_by_path.api.repository_id
  tag_pattern       = "v*"
  allowed_to_create = "maintainer"
}
