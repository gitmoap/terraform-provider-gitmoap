data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_issue_blueprints" "api" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
}
