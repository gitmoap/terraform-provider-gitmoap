data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_push_rules" "platform" {
  group_id                    = data.gitmoap_group_by_path.platform.group_id
  max_file_size_mb            = 50
  require_issue_reference     = true
  prohibited_file_names_regex = "(^|/)\\.env$"
}
