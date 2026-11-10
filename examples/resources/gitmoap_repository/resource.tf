data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_repository" "api" {
  group_id        = data.gitmoap_group_by_path.platform.group_id
  repository_name = "api"
  description     = "The public API"
  visibility      = "private"
}
