resource "gitmoap_repository" "api" {
  group_id        = gitmoap_group.platform.group_id
  repository_name = "api"
  description     = "The public API"
  visibility      = "private"
}
