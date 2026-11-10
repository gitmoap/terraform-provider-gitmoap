data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_milestone" "v2" {
  repo_id = data.gitmoap_repository_by_path.api.repository_id
  title   = "v2.0"
}

data "gitmoap_repository_milestone" "api" {
  milestone_id = gitmoap_repository_milestone.v2.milestone_id
  repo_id      = data.gitmoap_repository_by_path.api.repository_id
}
