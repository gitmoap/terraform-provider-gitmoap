data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_push_rules" "api" {
  repo_id                         = data.gitmoap_repository_by_path.api.repository_id
  max_file_size_mb                = 10
  require_closing_issue_reference = true
}
