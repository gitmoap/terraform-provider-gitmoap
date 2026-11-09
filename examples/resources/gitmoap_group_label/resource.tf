resource "gitmoap_group_label" "bug" {
  group_id    = gitmoap_group.platform.group_id
  key         = "bug"
  color       = "#d73a4a"
  description = "Something isn't working"
}
