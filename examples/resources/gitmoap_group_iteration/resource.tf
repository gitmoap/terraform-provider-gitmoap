data "gitmoap_group_by_path" "platform" {
  full_path = "acme/platform"
}

# An iteration that follows no cadence, such as a hardening week.
resource "gitmoap_group_iteration" "hardening" {
  group_id    = data.gitmoap_group_by_path.platform.group_id
  title       = "Hardening week"
  description = "No new features: fix, test and tidy up."
  start_date  = "2027-06-28"
  due_date    = "2027-07-02"
}
