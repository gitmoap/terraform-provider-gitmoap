data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_approval_policy" "api" {
  repo_id                 = data.gitmoap_repository_by_path.api.repository_id
  author_can_approve      = false
  committers_can_approve  = false
  last_pusher_can_approve = false
  reset_on_push           = true
}
