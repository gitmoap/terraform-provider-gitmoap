data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Forks stay inside the organization, in every repository beneath the group.
resource "gitmoap_group_fork_settings" "platform" {
  group_id            = data.gitmoap_group_by_path.platform.group_id
  forking             = "organization"
  forking_enforced    = true
  fork_merge_requests = "members"
}
