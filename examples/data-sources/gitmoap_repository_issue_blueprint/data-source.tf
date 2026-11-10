data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_issue_blueprint" "api" {
  blueprint_id = "incident"
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
}
