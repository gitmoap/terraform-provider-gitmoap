data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_issue_settings" "platform" {
  group_id                = data.gitmoap_group_by_path.platform.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
  todo_policy             = "index"
}
