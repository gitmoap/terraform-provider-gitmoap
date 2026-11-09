resource "gitmoap_group_mr_settings" "platform" {
  group_id                     = gitmoap_group.platform.group_id
  delete_source_branch_default = true
  squash_option                = "allow"
}
