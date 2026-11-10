data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_branch" "release" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  branch_name = "release/1.x"
  from_branch = "main"
}
