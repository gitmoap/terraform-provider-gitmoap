data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_mr_settings" "platform" {
  group_id                     = data.gitmoap_group_by_path.platform.group_id
  delete_source_branch_default = true
  squash_option                = "allow"
}
