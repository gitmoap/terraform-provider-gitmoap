data "gitmoap_repository_by_path" "api" {
  full_path = "acme/platform/api"
}

data "gitmoap_repository_tag" "api" {
  repo_id  = data.gitmoap_repository_by_path.api.repository_id
  tag_name = "v1.0.0"
}
