data "gitmoap_group_by_path" "acme" {
  full_path = "acme"
}

resource "gitmoap_group_iteration_cadence" "sprints" {
  group_id              = data.gitmoap_group_by_path.acme.group_id
  title                 = "Sprint"
  first_start_date      = "2027-01-04"
  start_day             = "monday"
  duration_weeks        = 2
  iterations_in_advance = 3
}

resource "gitmoap_group_milestone" "launch" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  title       = "Public launch"
  description = "Everything that has to land before launch"
  due_on      = "2027-03-31"
}

resource "gitmoap_group_iteration" "hardening" {
  group_id    = data.gitmoap_group_by_path.acme.group_id
  title       = "Hardening before launch"
  description = "No new features: fix, test and tidy up."
  start_date  = "2027-03-22"
  due_date    = "2027-03-26"
}
