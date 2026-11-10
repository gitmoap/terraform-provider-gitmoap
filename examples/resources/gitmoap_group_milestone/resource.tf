data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

resource "gitmoap_group_milestone" "launch" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  title       = "Public launch"
  description = "Everything that has to land before launch"
  due_on      = "2027-03-31"
}
