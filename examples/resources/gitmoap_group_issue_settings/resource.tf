data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_issue_settings" "platform" {
  group_id                = data.gitmoap_group_by_path.platform.group_id
  default_issue_type      = "task"
  close_on_commit_keyword = true
  todo_policy             = "index"

  # Work items follow their merge requests: in review when one opens, done when it merges, and back
  # where they were when it's closed without merging.
  on_mr_opened = { to = "in_review" }
  on_mr_merged = { to = "done" }
  on_mr_closed = { value = "restore" }
}
