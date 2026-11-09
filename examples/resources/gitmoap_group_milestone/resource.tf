resource "gitmoap_group_milestone" "launch" {
  group_id    = gitmoap_group.platform.group_id
  title       = "Public launch"
  description = "Everything that has to land before launch"
  due_on      = "2027-03-31"
}
