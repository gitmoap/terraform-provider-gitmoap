data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_branch" "api" {
  branch_name = "main"
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
}
