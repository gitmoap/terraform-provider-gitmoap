data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

resource "gitmoap_repository_label" "breaking" {
  repo_id     = data.gitmoap_repository_by_path.api.repository_id
  key         = "breaking-change"
  color       = "#b60205"
  description = "Changes the API in a way clients notice"
}
