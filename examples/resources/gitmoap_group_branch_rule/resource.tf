data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Protect main in every repository of the group.
resource "gitmoap_group_branch_rule" "main" {
  group_id         = data.gitmoap_group_by_path.platform.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
  allow_force_push = false
}
