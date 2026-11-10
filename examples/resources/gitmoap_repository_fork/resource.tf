data "gitmoap_repository_by_path" "fork" {
  full_path = "ada/api"
}

# A fork's settings: keep its main in step with the upstream's main.
resource "gitmoap_repository_fork" "api" {
  repo_id         = data.gitmoap_repository_by_path.fork.repository_id
  tracked_branch  = "main"
  upstream_branch = "main"
  auto_sync       = true
}
