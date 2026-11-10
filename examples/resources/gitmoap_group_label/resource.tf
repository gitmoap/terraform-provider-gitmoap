data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_label" "bug" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  key         = "bug"
  color       = "#d73a4a"
  description = "Something isn't working"
}
