data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# Nobody approves their own change, and a new push asks for a fresh review. Enforced, so the
# groups and repositories beneath can't relax it.
resource "gitmoap_group_approval_policy" "platform" {
  group_id                = data.gitmoap_group_by_path.platform.group_id
  author_can_approve      = false
  committers_can_approve  = true
  last_pusher_can_approve = false
  reset_on_push           = true
  enforcement             = "enforced"
}
