# Protect main in every repository of the group.
resource "gitmoap_group_branch_rule" "main" {
  group_id         = gitmoap_group.platform.group_id
  branch_pattern   = "main"
  allowed_to_push  = "maintainer"
  allowed_to_merge = "developer"
  allow_force_push = false
}
